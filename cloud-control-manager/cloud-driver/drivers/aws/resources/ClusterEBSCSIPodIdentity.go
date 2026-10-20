package resources

// EBS CSI driver credentials via EKS Pod Identity.
//
// The aws-ebs-csi-driver controller used to borrow the node IAM role through IMDS.
// That only works on nodes whose IMDS HttpPutResponseHopLimit is 2, which CB-Spider
// sets on the nodes that exist right after CreateNodeGroup only (ref: #1487).
// Nodes launched later by the Auto Scaling group (scale out, replacement) keep the
// default hop limit of 1, so the controller crash-loops with "no EC2 IMDS role found".
//
// Instead, the controller service account gets a dedicated IAM role through an
// EKS Pod Identity association, so it no longer depends on IMDS.
// (ref: https://docs.aws.amazon.com/eks/latest/userguide/ebs-csi.html)
//
// Pod Identity APIs and CreateAddon.PodIdentityAssociations do not exist in
// aws-sdk-go v1.39.4 (pinned for the rest of this driver), so only these EKS calls
// use aws-sdk-go-v2.

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	credentialsv2 "github.com/aws/aws-sdk-go-v2/credentials"
	eksv2 "github.com/aws/aws-sdk-go-v2/service/eks"
	eksv2types "github.com/aws/aws-sdk-go-v2/service/eks/types"
)

const (
	ebsCSIDriverAddonName          = "aws-ebs-csi-driver" // EBS CSI Driver, don't change
	ebsCSIDriverNamespace          = "kube-system"
	ebsCSIControllerServiceAccount = "ebs-csi-controller-sa"
	// The role name is fixed per account, like the cluster and node group roles.
	ebsCSIDriverRoleName  = "cloud-barista-eks-ebs-csi-driver-role"
	ebsCSIDriverPolicyArn = "arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy"
)

func (ClusterHandler *AwsClusterHandler) newEKSV2Client() *eksv2.Client {
	// CB-Spider fills an unset StsToken with the literal string "Not set" (see PublicIPHandler).
	stsToken := ClusterHandler.CredentialInfo.StsToken
	if stsToken == "Not set" {
		stsToken = ""
	}
	return eksv2.New(eksv2.Options{
		Region: ClusterHandler.Region.Region,
		Credentials: credentialsv2.NewStaticCredentialsProvider(
			ClusterHandler.CredentialInfo.ClientId, ClusterHandler.CredentialInfo.ClientSecret, stsToken),
	})
}

// getOrCreateEBSCSIDriverRole returns the ARN of the IAM role that EKS Pod Identity
// assigns to the EBS CSI controller, creating the role if it does not exist.
func (ClusterHandler *AwsClusterHandler) getOrCreateEBSCSIDriverRole() (string, error) {
	input := &iam.GetRoleInput{
		RoleName: aws.String(ebsCSIDriverRoleName),
	}

	result, err := ClusterHandler.Iam.GetRole(input)
	if err == nil {
		return aws.StringValue(result.Role.Arn), nil
	}
	if aerr, ok := err.(awserr.Error); !ok || aerr.Code() != iam.ErrCodeNoSuchEntityException {
		return "", fmt.Errorf("failed to get role %s: %w", ebsCSIDriverRoleName, err)
	}

	cblogger.Infof("Creating EBS CSI driver role: %s", ebsCSIDriverRoleName)

	// EKS Pod Identity requires both sts:AssumeRole and sts:TagSession.
	trustPolicy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {
					"Service": "pods.eks.amazonaws.com"
				},
				"Action": ["sts:AssumeRole", "sts:TagSession"]
			}
		]
	}`

	_, err = ClusterHandler.Iam.CreateRole(&iam.CreateRoleInput{
		RoleName:                 aws.String(ebsCSIDriverRoleName),
		AssumeRolePolicyDocument: aws.String(trustPolicy),
		Description:              aws.String("IAM role for the EBS CSI driver of Cloud-Barista EKS clusters (EKS Pod Identity)"),
		Tags: []*iam.Tag{
			{
				Key:   aws.String("ManagedBy"),
				Value: aws.String("Cloud-Barista-Spider"),
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create role %s: %w", ebsCSIDriverRoleName, err)
	}

	_, err = ClusterHandler.Iam.AttachRolePolicy(&iam.AttachRolePolicyInput{
		RoleName:  aws.String(ebsCSIDriverRoleName),
		PolicyArn: aws.String(ebsCSIDriverPolicyArn),
	})
	if err != nil {
		return "", fmt.Errorf("failed to attach policy %s: %w", ebsCSIDriverPolicyArn, err)
	}

	cblogger.Info("Waiting for IAM changes to propagate...")
	time.Sleep(10 * time.Second)

	result, err = ClusterHandler.Iam.GetRole(input)
	if err != nil {
		return "", fmt.Errorf("failed to get newly created role %s: %w", ebsCSIDriverRoleName, err)
	}
	return aws.StringValue(result.Role.Arn), nil
}

func ebsCSIPodIdentityAssociations(roleArn string) []eksv2types.AddonPodIdentityAssociations {
	return []eksv2types.AddonPodIdentityAssociations{
		{
			RoleArn:        awsv2.String(roleArn),
			ServiceAccount: awsv2.String(ebsCSIControllerServiceAccount),
		},
	}
}

// installEBSCSIDriver installs the aws-ebs-csi-driver add-on together with the
// Pod Identity association of its controller service account, so the association
// exists before the controller Pods are created.
func (ClusterHandler *AwsClusterHandler) installEBSCSIDriver(clusterID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	input := &eksv2.CreateAddonInput{
		ClusterName:      awsv2.String(clusterID),
		AddonName:        awsv2.String(ebsCSIDriverAddonName),
		ResolveConflicts: eksv2types.ResolveConflictsOverwrite,
	}

	roleArn, err := ClusterHandler.getOrCreateEBSCSIDriverRole()
	if err != nil {
		// Keep the previous behavior (node role through IMDS) rather than failing the cluster.
		cblogger.Errorf("Failed to get or create EBS CSI driver role, installing without Pod Identity: %v", err)
	} else {
		input.PodIdentityAssociations = ebsCSIPodIdentityAssociations(roleArn)
	}

	_, err = ClusterHandler.newEKSV2Client().CreateAddon(ctx, input)
	if err != nil {
		cblogger.Errorf("Failed to install EBS CSI Driver: %v", err)
		return err
	}
	return nil
}

// ensureEBSCSIDriverPodIdentity adds the Pod Identity association to an
// aws-ebs-csi-driver add-on that was installed without it (clusters created by
// an earlier CB-Spider version). It does nothing if the add-on is not installed.
//
// Pod Identity injects credentials when a Pod is created, so controller Pods that
// already run keep using the node role until they are recreated.
func (ClusterHandler *AwsClusterHandler) ensureEBSCSIDriverPodIdentity(clusterID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	client := ClusterHandler.newEKSV2Client()

	_, err := client.DescribeAddon(ctx, &eksv2.DescribeAddonInput{
		ClusterName: awsv2.String(clusterID),
		AddonName:   awsv2.String(ebsCSIDriverAddonName),
	})
	if err != nil {
		var notFound *eksv2types.ResourceNotFoundException
		if errors.As(err, &notFound) {
			return nil
		}
		return fmt.Errorf("failed to describe %s add-on: %w", ebsCSIDriverAddonName, err)
	}

	assocs, err := client.ListPodIdentityAssociations(ctx, &eksv2.ListPodIdentityAssociationsInput{
		ClusterName:    awsv2.String(clusterID),
		Namespace:      awsv2.String(ebsCSIDriverNamespace),
		ServiceAccount: awsv2.String(ebsCSIControllerServiceAccount),
	})
	if err != nil {
		return fmt.Errorf("failed to list pod identity associations: %w", err)
	}
	if len(assocs.Associations) > 0 {
		return nil
	}

	roleArn, err := ClusterHandler.getOrCreateEBSCSIDriverRole()
	if err != nil {
		return err
	}

	_, err = client.UpdateAddon(ctx, &eksv2.UpdateAddonInput{
		ClusterName:             awsv2.String(clusterID),
		AddonName:               awsv2.String(ebsCSIDriverAddonName),
		ResolveConflicts:        eksv2types.ResolveConflictsPreserve,
		PodIdentityAssociations: ebsCSIPodIdentityAssociations(roleArn),
	})
	if err != nil {
		return fmt.Errorf("failed to add pod identity association to %s: %w", ebsCSIDriverAddonName, err)
	}

	cblogger.Infof("Added EKS Pod Identity association for %s/%s in cluster %s",
		ebsCSIDriverNamespace, ebsCSIControllerServiceAccount, clusterID)
	return nil
}
