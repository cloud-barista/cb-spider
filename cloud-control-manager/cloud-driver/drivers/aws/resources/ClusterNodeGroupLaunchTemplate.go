// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// EKS managed NodeGroup Launch Template handling.
//
// Pods on EKS nodes (e.g. aws-ebs-csi-driver controller) get AWS credentials from the
// node role through IMDSv2, which needs HttpPutResponseHopLimit >= 2 (#1487).
// Modifying only the instances that exist right after NodeGroup creation is not enough:
// nodes launched later by the Auto Scaling group (scale out/in, replacement) come up with
// the default hop limit 1, and the EBS CSI controller then crash-loops with
// "no EC2 IMDS role found". So spider creates a Launch Template carrying the metadata
// options and lets the NodeGroup's Auto Scaling group use it for every node.

package resources

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/eks"
)

const nodeGroupLaunchTemplatePrefix = "cb-spider-eks-"

// nodeGroupLaunchTemplateName returns the name of the Launch Template spider creates for a NodeGroup.
func nodeGroupLaunchTemplateName(clusterName, nodeGroupName string) string {
	return nodeGroupLaunchTemplatePrefix + clusterName + "-" + nodeGroupName
}

// rootDeviceNameForAmiType returns the block device that holds the node's OS/container data for an EKS AMI type.
func rootDeviceNameForAmiType(amiType string) string {
	switch {
	case strings.HasPrefix(amiType, "BOTTLEROCKET"):
		return "/dev/xvdb" // Bottlerocket: /dev/xvda is the OS volume, /dev/xvdb holds container data
	case strings.HasPrefix(amiType, "WINDOWS"):
		return "/dev/sda1"
	default:
		return "/dev/xvda" // AL2 / AL2023
	}
}

// createNodeGroupLaunchTemplate creates the Launch Template used by a spider-managed EKS NodeGroup.
// RemoteAccess and DiskSize cannot be combined with a Launch Template, so the key pair, security groups
// and root disk size move into the template. Security groups set in a Launch Template replace the
// EKS default, so the cluster security group is included explicitly.
func (ClusterHandler *AwsClusterHandler) createNodeGroupLaunchTemplate(clusterName, nodeGroupName, amiType, keyName string,
	nodeSecurityGroupList []*string, rootDiskSize int64) (*string, error) {

	clusterDesc, err := ClusterHandler.Client.DescribeCluster(&eks.DescribeClusterInput{Name: aws.String(clusterName)})
	if err != nil {
		return nil, fmt.Errorf("failed to describe cluster %s: %w", clusterName, err)
	}

	securityGroupIds := []*string{}
	if vpcConfig := clusterDesc.Cluster.ResourcesVpcConfig; vpcConfig != nil && aws.StringValue(vpcConfig.ClusterSecurityGroupId) != "" {
		securityGroupIds = append(securityGroupIds, vpcConfig.ClusterSecurityGroupId)
	}
	securityGroupIds = append(securityGroupIds, nodeSecurityGroupList...)

	templateData := &ec2.RequestLaunchTemplateData{
		MetadataOptions: &ec2.LaunchTemplateInstanceMetadataOptionsRequest{
			HttpEndpoint:            aws.String(ec2.LaunchTemplateInstanceMetadataEndpointStateEnabled),
			HttpTokens:              aws.String(ec2.LaunchTemplateHttpTokensStateRequired),
			HttpPutResponseHopLimit: aws.Int64(int64(httpPutResponseHopLimitIs2)),
		},
		SecurityGroupIds: securityGroupIds,
	}
	if keyName != "" {
		templateData.KeyName = aws.String(keyName)
	}
	if rootDiskSize > 0 {
		templateData.BlockDeviceMappings = []*ec2.LaunchTemplateBlockDeviceMappingRequest{{
			DeviceName: aws.String(rootDeviceNameForAmiType(amiType)),
			Ebs: &ec2.LaunchTemplateEbsBlockDeviceRequest{
				VolumeSize:          aws.Int64(rootDiskSize),
				VolumeType:          aws.String(ec2.VolumeTypeGp3),
				DeleteOnTermination: aws.Bool(true),
			},
		}}
	}

	templateName := nodeGroupLaunchTemplateName(clusterName, nodeGroupName)
	input := &ec2.CreateLaunchTemplateInput{
		LaunchTemplateName: aws.String(templateName),
		LaunchTemplateData: templateData,
		TagSpecifications: []*ec2.TagSpecification{{
			ResourceType: aws.String(ec2.ResourceTypeLaunchTemplate),
			Tags: []*ec2.Tag{
				{Key: aws.String("CreatedBy"), Value: aws.String("Cloud-Barista-Spider")},
				{Key: aws.String("eks-cluster"), Value: aws.String(clusterName)},
				{Key: aws.String("eks-nodegroup"), Value: aws.String(nodeGroupName)},
			},
		}},
	}

	result, err := ClusterHandler.EC2Client.CreateLaunchTemplate(input)
	if err != nil {
		// A template left over from a failed attempt with the same NodeGroup name: replace it.
		if aerr, ok := err.(awserr.Error); ok && strings.Contains(aerr.Code(), "AlreadyExists") {
			cblogger.Infof("Launch Template %s already exists. Recreating it.", templateName)
			if delErr := ClusterHandler.deleteNodeGroupLaunchTemplate(clusterName, nodeGroupName); delErr != nil {
				return nil, delErr
			}
			result, err = ClusterHandler.EC2Client.CreateLaunchTemplate(input)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to create Launch Template %s: %w", templateName, err)
		}
	}

	cblogger.Infof("Created Launch Template %s (%s) for NodeGroup %s", templateName, aws.StringValue(result.LaunchTemplate.LaunchTemplateId), nodeGroupName)
	return result.LaunchTemplate.LaunchTemplateId, nil
}

// deleteNodeGroupLaunchTemplate deletes the Launch Template spider created for a NodeGroup. A missing template is not an error.
func (ClusterHandler *AwsClusterHandler) deleteNodeGroupLaunchTemplate(clusterName, nodeGroupName string) error {
	templateName := nodeGroupLaunchTemplateName(clusterName, nodeGroupName)
	_, err := ClusterHandler.EC2Client.DeleteLaunchTemplate(&ec2.DeleteLaunchTemplateInput{LaunchTemplateName: aws.String(templateName)})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && strings.Contains(aerr.Code(), "NotFound") {
			return nil
		}
		return fmt.Errorf("failed to delete Launch Template %s: %w", templateName, err)
	}
	cblogger.Infof("Deleted Launch Template %s", templateName)
	return nil
}

// deleteNodeGroupLaunchTemplateAfterNodeGroupDeleted waits for the NodeGroup deletion to finish
// (a template cannot be deleted while its instances still use it) and then removes the template.
func (ClusterHandler *AwsClusterHandler) deleteNodeGroupLaunchTemplateAfterNodeGroupDeleted(clusterName, nodeGroupName string) {
	err := ClusterHandler.Client.WaitUntilNodegroupDeleted(&eks.DescribeNodegroupInput{
		ClusterName:   aws.String(clusterName),
		NodegroupName: aws.String(nodeGroupName),
	})
	if err != nil {
		cblogger.Errorf("failed to wait for NodeGroup %s deletion; Launch Template %s is left: %v", nodeGroupName, nodeGroupLaunchTemplateName(clusterName, nodeGroupName), err)
		return
	}
	if err := ClusterHandler.deleteNodeGroupLaunchTemplate(clusterName, nodeGroupName); err != nil {
		cblogger.Error(err)
	}
}

// nodeGroupLaunchTemplateInfo reads the key pair and root disk size of the Launch Template a NodeGroup uses.
func (ClusterHandler *AwsClusterHandler) nodeGroupLaunchTemplateInfo(spec *eks.LaunchTemplateSpecification) (keyName string, rootDiskSize int64) {
	if spec == nil || (aws.StringValue(spec.Id) == "" && aws.StringValue(spec.Name) == "") {
		return "", 0
	}
	input := &ec2.DescribeLaunchTemplateVersionsInput{}
	if aws.StringValue(spec.Id) != "" {
		input.LaunchTemplateId = spec.Id
	} else {
		input.LaunchTemplateName = spec.Name
	}
	if aws.StringValue(spec.Version) != "" {
		input.Versions = []*string{spec.Version}
	}
	out, err := ClusterHandler.EC2Client.DescribeLaunchTemplateVersions(input)
	if err != nil || len(out.LaunchTemplateVersions) == 0 || out.LaunchTemplateVersions[0].LaunchTemplateData == nil {
		if err != nil {
			cblogger.Error(err)
		}
		return "", 0
	}
	data := out.LaunchTemplateVersions[0].LaunchTemplateData
	keyName = aws.StringValue(data.KeyName)
	for _, bdm := range data.BlockDeviceMappings {
		if bdm.Ebs != nil && aws.Int64Value(bdm.Ebs.VolumeSize) > 0 {
			rootDiskSize = aws.Int64Value(bdm.Ebs.VolumeSize)
			break
		}
	}
	return keyName, rootDiskSize
}
