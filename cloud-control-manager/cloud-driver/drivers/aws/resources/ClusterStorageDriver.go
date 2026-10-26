// AWS Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// CSI storage driver management of a Kubernetes Cluster.
//
// EKS manages the CSI drivers as EKS add-ons:
//   - Block: aws-ebs-csi-driver (ebs.csi.aws.com)
//   - File:  aws-efs-csi-driver (efs.csi.aws.com)
//
// Each controller gets its IAM role through an EKS Pod Identity association that is
// created together with the add-on (see ClusterEBSCSIPodIdentity.go).
// Disabling deletes the add-on only. Volumes, file systems and IAM roles are kept.
//
// by CB-Spider Team, 2026.10.

package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/iam"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	eksv2 "github.com/aws/aws-sdk-go-v2/service/eks"
	eksv2types "github.com/aws/aws-sdk-go-v2/service/eks/types"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	podIdentityAgentAddonName = "eks-pod-identity-agent" // Eks pod identity agent, don't change

	efsCSIDriverAddonName          = "aws-efs-csi-driver" // EFS CSI Driver, don't change
	efsCSIControllerServiceAccount = "efs-csi-controller-sa"
	// The role name is fixed per account, like the EBS CSI driver role.
	efsCSIDriverRoleName  = "cloud-barista-eks-efs-csi-driver-role"
	efsCSIDriverPolicyArn = "arn:aws:iam::aws:policy/service-role/AmazonEFSCSIDriverPolicy"
)

// eksCSIDriverSpec describes a CSI driver that EKS manages as an add-on.
type eksCSIDriverSpec struct {
	kind            irs.StorageDriverKind
	addonName       string // EKS add-on name
	csiDriverName   string // name of the Kubernetes CSIDriver object
	namespace       string // namespace of the controller service account
	serviceAccount  string // controller service account that gets the IAM role through Pod Identity
	roleName        string
	policyArn       string
	roleDescription string

	// requiresFileSystem is true when dynamic provisioning needs an existing file system.
	requiresFileSystem bool
	// nodeRolePolicy is an inline policy the node IAM roles need for the driver's node plugin.
	nodeRolePolicy string
	// installWithoutPodIdentity installs the add-on even when the IAM role cannot be prepared,
	// so the controller falls back to the node role (the node role has the EBS policy only).
	installWithoutPodIdentity bool
}

var ebsCSIDriverSpec = eksCSIDriverSpec{
	kind:                      irs.StorageDriverBlock,
	addonName:                 ebsCSIDriverAddonName,
	csiDriverName:             "ebs.csi.aws.com",
	namespace:                 ebsCSIDriverNamespace,
	serviceAccount:            ebsCSIControllerServiceAccount,
	roleName:                  ebsCSIDriverRoleName,
	policyArn:                 ebsCSIDriverPolicyArn,
	roleDescription:           "IAM role for the EBS CSI driver of Cloud-Barista EKS clusters (EKS Pod Identity)",
	installWithoutPodIdentity: true,
}

// EFS dynamic provisioning creates access points on an existing EFS file system,
// which needs mount targets in the subnets of the cluster nodes.
var efsCSIDriverSpec = eksCSIDriverSpec{
	kind:               irs.StorageDriverFile,
	addonName:          efsCSIDriverAddonName,
	csiDriverName:      "efs.csi.aws.com",
	namespace:          "kube-system",
	serviceAccount:     efsCSIControllerServiceAccount,
	roleName:           efsCSIDriverRoleName,
	policyArn:          efsCSIDriverPolicyArn,
	roleDescription:    "IAM role for the EFS CSI driver of Cloud-Barista EKS clusters (EKS Pod Identity)",
	requiresFileSystem: true,
	nodeRolePolicy:     efsNodeRolePolicy,
}

// efsNodeRolePolicyName is the inline policy added to the node roles when the EFS driver is enabled.
// The EFS mount helper on the node resolves fs-xxx.efs.<region>.amazonaws.com; when the VPC has
// no DNS hostnames it looks the mount target IP up with the node role instead, which needs these
// actions.
const efsNodeRolePolicyName = "cloud-barista-efs-csi-mount-target-lookup"

const efsNodeRolePolicy = `{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": ["elasticfilesystem:DescribeMountTargets", "ec2:DescribeAvailabilityZones"],
    "Resource": "*"
  }]
}`

// eksCSIDriverSpecs is in the order ListStorageDriver returns: Block, then File.
var eksCSIDriverSpecs = []eksCSIDriverSpec{ebsCSIDriverSpec, efsCSIDriverSpec}

func getEKSCSIDriverSpec(kind irs.StorageDriverKind) (eksCSIDriverSpec, error) {
	for _, spec := range eksCSIDriverSpecs {
		if strings.EqualFold(string(spec.kind), string(kind)) {
			return spec, nil
		}
	}
	return eksCSIDriverSpec{}, fmt.Errorf("unsupported storage driver kind %q: must be %q or %q",
		kind, irs.StorageDriverBlock, irs.StorageDriverFile)
}

// convertEKSAddonStatusToStorageDriverStatus maps the status of an installed EKS add-on.
// A missing add-on is Disabled and is handled by the caller.
func convertEKSAddonStatusToStorageDriverStatus(status eksv2types.AddonStatus) irs.StorageDriverStatus {
	switch status {
	case eksv2types.AddonStatusActive:
		return irs.StorageDriverEnabled
	case eksv2types.AddonStatusCreating, eksv2types.AddonStatusUpdating, eksv2types.AddonStatusDeleting:
		return irs.StorageDriverUpdating
	case eksv2types.AddonStatusCreateFailed, eksv2types.AddonStatusDegraded,
		eksv2types.AddonStatusUpdateFailed, eksv2types.AddonStatusDeleteFailed:
		return irs.StorageDriverFailed
	default:
		return irs.StorageDriverUnknown
	}
}

// convertEKSAddonToStorageDriverInfo converts an EKS add-on. A nil add-on means it is not installed.
func convertEKSAddonToStorageDriverInfo(spec eksCSIDriverSpec, addon *eksv2types.Addon) irs.StorageDriverInfo {
	info := irs.StorageDriverInfo{
		Kind:               spec.kind,
		CSPDriverName:      spec.addonName,
		CSIDriverName:      spec.csiDriverName,
		ManagementMode:     irs.StorageDriverAddon,
		Status:             irs.StorageDriverDisabled,
		RequiresFileSystem: spec.requiresFileSystem,
	}
	if addon == nil {
		return info
	}

	info.Status = convertEKSAddonStatusToStorageDriverStatus(addon.Status)
	info.Version = awsv2.ToString(addon.AddonVersion)

	kvList := []irs.KeyValue{{Key: "AddonStatus", Value: string(addon.Status)}}
	if addon.AddonArn != nil {
		kvList = append(kvList, irs.KeyValue{Key: "AddonArn", Value: *addon.AddonArn})
	}
	if addon.ServiceAccountRoleArn != nil {
		kvList = append(kvList, irs.KeyValue{Key: "ServiceAccountRoleArn", Value: *addon.ServiceAccountRoleArn})
	}
	if len(addon.PodIdentityAssociations) > 0 {
		kvList = append(kvList, irs.KeyValue{Key: "PodIdentityAssociations", Value: strings.Join(addon.PodIdentityAssociations, ",")})
	}
	if addon.Health != nil {
		for i, issue := range addon.Health.Issues {
			kvList = append(kvList, irs.KeyValue{
				Key:   fmt.Sprintf("HealthIssue%d", i+1),
				Value: fmt.Sprintf("%s: %s", issue.Code, awsv2.ToString(issue.Message)),
			})
		}
	}
	info.KeyValueList = kvList
	return info
}

// getEKSClusterStatus returns the status of the cluster, or an error if it does not exist.
// DescribeAddon also returns ResourceNotFoundException for a missing cluster, so the
// cluster is checked first to avoid reporting its drivers as Disabled.
func (ClusterHandler *AwsClusterHandler) getEKSClusterStatus(clusterID string) (string, error) {
	callLogInfo := GetCallLogScheme(ClusterHandler.Region, call.CLUSTER, clusterID, "DescribeCluster()")
	start := call.Start()
	result, err := ClusterHandler.Client.DescribeCluster(&eks.DescribeClusterInput{
		Name: aws.String(clusterID),
	})
	if err != nil {
		LoggingError(callLogInfo, err)
		return "", fmt.Errorf("failed to describe cluster %s: %w", clusterID, err)
	}
	LoggingInfo(callLogInfo, start)
	if result.Cluster == nil {
		return "", fmt.Errorf("cluster %s not found", clusterID)
	}
	return aws.StringValue(result.Cluster.Status), nil
}

func (ClusterHandler *AwsClusterHandler) checkEKSClusterActive(clusterID string) error {
	status, err := ClusterHandler.getEKSClusterStatus(clusterID)
	if err != nil {
		return err
	}
	if status != eks.ClusterStatusActive {
		return fmt.Errorf("cluster %s is %s; storage drivers can be changed only when the cluster is %s",
			clusterID, status, eks.ClusterStatusActive)
	}
	return nil
}

// describeEKSAddon returns the add-on, or nil if it is not installed.
func (ClusterHandler *AwsClusterHandler) describeEKSAddon(ctx context.Context, client *eksv2.Client, clusterID string, addonName string) (*eksv2types.Addon, error) {
	callLogInfo := GetCallLogScheme(ClusterHandler.Region, call.CLUSTER, clusterID, "DescribeAddon()")
	start := call.Start()
	result, err := client.DescribeAddon(ctx, &eksv2.DescribeAddonInput{
		ClusterName: awsv2.String(clusterID),
		AddonName:   awsv2.String(addonName),
	})
	if err != nil {
		var notFound *eksv2types.ResourceNotFoundException
		if errors.As(err, &notFound) {
			LoggingInfo(callLogInfo, start)
			return nil, nil
		}
		LoggingError(callLogInfo, err)
		return nil, fmt.Errorf("failed to describe %s add-on: %w", addonName, err)
	}
	LoggingInfo(callLogInfo, start)
	return result.Addon, nil
}

// ensurePodIdentityAgent installs the eks-pod-identity-agent add-on if it is not installed.
// The CSI driver controllers get their credentials through it.
func (ClusterHandler *AwsClusterHandler) ensurePodIdentityAgent(ctx context.Context, client *eksv2.Client, clusterID string) error {
	addon, err := ClusterHandler.describeEKSAddon(ctx, client, clusterID, podIdentityAgentAddonName)
	if err != nil {
		return err
	}
	if addon != nil {
		return nil
	}
	if err := ClusterHandler.installPodIdentityAgent(clusterID); err != nil {
		return fmt.Errorf("failed to install %s add-on: %w", podIdentityAgentAddonName, err)
	}
	return nil
}

// ensureNodeRolePolicy puts the driver's inline policy on the IAM roles of every node group of the cluster.
func (ClusterHandler *AwsClusterHandler) ensureNodeRolePolicy(ctx context.Context, client *eksv2.Client, clusterID string, spec eksCSIDriverSpec) error {
	if spec.nodeRolePolicy == "" {
		return nil
	}
	ngList, err := client.ListNodegroups(ctx, &eksv2.ListNodegroupsInput{ClusterName: awsv2.String(clusterID)})
	if err != nil {
		return fmt.Errorf("failed to list node groups of cluster %s: %w", clusterID, err)
	}
	roleNames := map[string]bool{}
	for _, ng := range ngList.Nodegroups {
		out, err := client.DescribeNodegroup(ctx, &eksv2.DescribeNodegroupInput{ClusterName: awsv2.String(clusterID), NodegroupName: awsv2.String(ng)})
		if err != nil {
			return fmt.Errorf("failed to describe node group %s: %w", ng, err)
		}
		if arn := awsv2.ToString(out.Nodegroup.NodeRole); arn != "" {
			roleNames[arn[strings.LastIndex(arn, "/")+1:]] = true
		}
	}
	for roleName := range roleNames {
		callLogInfo := GetCallLogScheme(ClusterHandler.Region, call.CLUSTER, clusterID, "PutRolePolicy()")
		start := call.Start()
		_, err := ClusterHandler.Iam.PutRolePolicy(&iam.PutRolePolicyInput{
			RoleName:       aws.String(roleName),
			PolicyName:     aws.String(efsNodeRolePolicyName),
			PolicyDocument: aws.String(spec.nodeRolePolicy),
		})
		if err != nil {
			LoggingError(callLogInfo, err)
			return fmt.Errorf("failed to put %s policy on node role %s: %w", efsNodeRolePolicyName, roleName, err)
		}
		LoggingInfo(callLogInfo, start)
	}
	return nil
}

func (ClusterHandler *AwsClusterHandler) ListStorageDriver(clusterIID irs.IID) ([]irs.StorageDriverInfo, error) {
	cblogger.Info("AWS Cloud Driver: called ListStorageDriver()")
	clusterID := clusterIID.SystemId

	if _, err := ClusterHandler.getEKSClusterStatus(clusterID); err != nil {
		cblogger.Error(err)
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	client := ClusterHandler.newEKSV2Client()

	driverList := make([]irs.StorageDriverInfo, 0, len(eksCSIDriverSpecs))
	for _, spec := range eksCSIDriverSpecs {
		addon, err := ClusterHandler.describeEKSAddon(ctx, client, clusterID, spec.addonName)
		if err != nil {
			cblogger.Error(err)
			return nil, err
		}
		driverList = append(driverList, convertEKSAddonToStorageDriverInfo(spec, addon))
	}
	return driverList, nil
}

func (ClusterHandler *AwsClusterHandler) EnableStorageDriver(clusterIID irs.IID, kind irs.StorageDriverKind, version string) (irs.StorageDriverInfo, error) {
	cblogger.Info("AWS Cloud Driver: called EnableStorageDriver()")
	clusterID := clusterIID.SystemId

	spec, err := getEKSCSIDriverSpec(kind)
	if err != nil {
		return irs.StorageDriverInfo{}, err
	}
	if err := ClusterHandler.checkEKSClusterActive(clusterID); err != nil {
		cblogger.Error(err)
		return irs.StorageDriverInfo{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	client := ClusterHandler.newEKSV2Client()

	addon, err := ClusterHandler.describeEKSAddon(ctx, client, clusterID, spec.addonName)
	if err != nil {
		cblogger.Error(err)
		return irs.StorageDriverInfo{}, err
	}

	if err := ClusterHandler.ensureNodeRolePolicy(ctx, client, clusterID, spec); err != nil {
		cblogger.Error(err)
		return irs.StorageDriverInfo{}, err
	}

	if addon != nil {
		return ClusterHandler.reconcileInstalledCSIDriverAddon(ctx, client, clusterID, spec, addon, version)
	}

	// Install the pod identity agent first: the controller gets its credentials through it.
	if err := ClusterHandler.ensurePodIdentityAgent(ctx, client, clusterID); err != nil {
		cblogger.Error(err)
		return irs.StorageDriverInfo{}, err
	}

	addon, err = ClusterHandler.installCSIDriverAddon(clusterID, spec, version)
	if err != nil {
		cblogger.Error(err)
		return irs.StorageDriverInfo{}, err
	}
	info := convertEKSAddonToStorageDriverInfo(spec, addon)
	if addon == nil {
		// CreateAddon succeeded but returned no add-on: the add-on is being created.
		info.Status = irs.StorageDriverUpdating
		info.Version = version
	}
	return info, nil
}

// reconcileInstalledCSIDriverAddon handles EnableStorageDriver for an add-on that already exists.
// It does nothing if the add-on is already at the requested version, and updates the version otherwise.
func (ClusterHandler *AwsClusterHandler) reconcileInstalledCSIDriverAddon(ctx context.Context, client *eksv2.Client,
	clusterID string, spec eksCSIDriverSpec, addon *eksv2types.Addon, version string) (irs.StorageDriverInfo, error) {

	info := convertEKSAddonToStorageDriverInfo(spec, addon)

	switch addon.Status {
	case eksv2types.AddonStatusDeleting:
		return info, fmt.Errorf("%s add-on is being deleted; enable it again after the deletion completes", spec.addonName)
	case eksv2types.AddonStatusCreateFailed, eksv2types.AddonStatusDeleteFailed:
		return info, fmt.Errorf("%s add-on is in %s state; disable it and enable it again", spec.addonName, addon.Status)
	case eksv2types.AddonStatusCreating, eksv2types.AddonStatusUpdating:
		// Another change is in progress. Report it as is.
		return info, nil
	}

	// An add-on installed by an earlier CB-Spider version or by the user may have no Pod Identity association.
	if err := ClusterHandler.ensureCSIDriverPodIdentity(clusterID, spec); err != nil {
		cblogger.Warnf("Failed to ensure %s pod identity: %v", spec.addonName, err)
	}

	if version == "" || version == awsv2.ToString(addon.AddonVersion) {
		return info, nil
	}

	callLogInfo := GetCallLogScheme(ClusterHandler.Region, call.CLUSTER, clusterID, "UpdateAddon()")
	start := call.Start()
	_, err := client.UpdateAddon(ctx, &eksv2.UpdateAddonInput{
		ClusterName:      awsv2.String(clusterID),
		AddonName:        awsv2.String(spec.addonName),
		AddonVersion:     awsv2.String(version),
		ResolveConflicts: eksv2types.ResolveConflictsOverwrite,
	})
	if err != nil {
		LoggingError(callLogInfo, err)
		err = fmt.Errorf("failed to update %s add-on to version %s: %w", spec.addonName, version, err)
		cblogger.Error(err)
		return info, err
	}
	LoggingInfo(callLogInfo, start)

	info.Status = irs.StorageDriverUpdating
	info.Version = version
	return info, nil
}

func (ClusterHandler *AwsClusterHandler) DisableStorageDriver(clusterIID irs.IID, kind irs.StorageDriverKind) (bool, error) {
	cblogger.Info("AWS Cloud Driver: called DisableStorageDriver()")
	clusterID := clusterIID.SystemId

	spec, err := getEKSCSIDriverSpec(kind)
	if err != nil {
		return false, err
	}
	if err := ClusterHandler.checkEKSClusterActive(clusterID); err != nil {
		cblogger.Error(err)
		return false, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	client := ClusterHandler.newEKSV2Client()

	addon, err := ClusterHandler.describeEKSAddon(ctx, client, clusterID, spec.addonName)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}
	if addon == nil || addon.Status == eksv2types.AddonStatusDeleting {
		return true, nil
	}

	// Preserve=false removes the add-on's Kubernetes resources (controller, node plugin, CSIDriver).
	// EBS volumes, EFS file systems and the IAM role are not deleted.
	callLogInfo := GetCallLogScheme(ClusterHandler.Region, call.CLUSTER, clusterID, "DeleteAddon()")
	start := call.Start()
	_, err = client.DeleteAddon(ctx, &eksv2.DeleteAddonInput{
		ClusterName: awsv2.String(clusterID),
		AddonName:   awsv2.String(spec.addonName),
		Preserve:    false,
	})
	if err != nil {
		var notFound *eksv2types.ResourceNotFoundException
		if errors.As(err, &notFound) {
			LoggingInfo(callLogInfo, start)
			return true, nil
		}
		LoggingError(callLogInfo, err)
		err = fmt.Errorf("failed to delete %s add-on: %w", spec.addonName, err)
		cblogger.Error(err)
		return false, err
	}
	LoggingInfo(callLogInfo, start)
	return true, nil
}
