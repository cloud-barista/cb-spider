// AWS Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// CSI storage driver management of a Kubernetes Cluster.
//
// EKS manages the Block CSI driver as the aws-ebs-csi-driver add-on (ebs.csi.aws.com).
// The controller gets its IAM role through an EKS Pod Identity association that is
// created together with the add-on (see ClusterEBSCSIPodIdentity.go).
// Disabling deletes the add-on only. Volumes and the IAM role are kept.
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

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	eksv2 "github.com/aws/aws-sdk-go-v2/service/eks"
	eksv2types "github.com/aws/aws-sdk-go-v2/service/eks/types"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	podIdentityAgentAddonName = "eks-pod-identity-agent" // Eks pod identity agent, don't change
	ebsCSIDriverName          = "ebs.csi.aws.com"        // name of the Kubernetes CSIDriver object
)

func checkEKSStorageDriverKind(kind irs.StorageDriverKind) error {
	if !strings.EqualFold(string(kind), string(irs.StorageDriverBlock)) {
		return fmt.Errorf("storage driver kind(%s) is not supported; only Block is supported", kind)
	}
	return nil
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
func convertEKSAddonToStorageDriverInfo(addon *eksv2types.Addon) irs.StorageDriverInfo {
	info := irs.StorageDriverInfo{
		Kind:           irs.StorageDriverBlock,
		CSPDriverName:  ebsCSIDriverAddonName,
		CSIDriverName:  ebsCSIDriverName,
		ManagementMode: irs.StorageDriverAddon,
		Status:         irs.StorageDriverDisabled,
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
		AddonName:   awsv2.String(ebsCSIDriverAddonName),
	})
	if err != nil {
		var notFound *eksv2types.ResourceNotFoundException
		if errors.As(err, &notFound) {
			LoggingInfo(callLogInfo, start)
			return nil, nil
		}
		LoggingError(callLogInfo, err)
		return nil, fmt.Errorf("failed to describe %s add-on: %w", ebsCSIDriverAddonName, err)
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

	addon, err := ClusterHandler.describeEKSAddon(ctx, client, clusterID, ebsCSIDriverAddonName)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}
	return []irs.StorageDriverInfo{convertEKSAddonToStorageDriverInfo(addon)}, nil
}

func (ClusterHandler *AwsClusterHandler) EnableStorageDriver(clusterIID irs.IID, kind irs.StorageDriverKind, version string) (irs.StorageDriverInfo, error) {
	cblogger.Info("AWS Cloud Driver: called EnableStorageDriver()")
	clusterID := clusterIID.SystemId

	if err := checkEKSStorageDriverKind(kind); err != nil {
		return irs.StorageDriverInfo{}, err
	}
	if err := ClusterHandler.checkEKSClusterActive(clusterID); err != nil {
		cblogger.Error(err)
		return irs.StorageDriverInfo{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	client := ClusterHandler.newEKSV2Client()

	addon, err := ClusterHandler.describeEKSAddon(ctx, client, clusterID, ebsCSIDriverAddonName)
	if err != nil {
		cblogger.Error(err)
		return irs.StorageDriverInfo{}, err
	}

	if addon != nil {
		return ClusterHandler.reconcileInstalledEBSCSIDriverAddon(ctx, client, clusterID, addon, version)
	}

	// Install the pod identity agent first: the controller gets its credentials through it.
	if err := ClusterHandler.ensurePodIdentityAgent(ctx, client, clusterID); err != nil {
		cblogger.Error(err)
		return irs.StorageDriverInfo{}, err
	}

	addon, err = ClusterHandler.installEBSCSIDriverAddon(ctx, client, clusterID, version)
	if err != nil {
		cblogger.Error(err)
		return irs.StorageDriverInfo{}, err
	}
	info := convertEKSAddonToStorageDriverInfo(addon)
	if addon == nil {
		// CreateAddon succeeded but returned no add-on: the add-on is being created.
		info.Status = irs.StorageDriverUpdating
		info.Version = version
	}
	return info, nil
}

// installEBSCSIDriverAddon installs the EBS CSI driver add-on at the given version (EKS default if empty)
// together with the Pod Identity association of its controller service account, like installEBSCSIDriver.
func (ClusterHandler *AwsClusterHandler) installEBSCSIDriverAddon(ctx context.Context, client *eksv2.Client,
	clusterID string, version string) (*eksv2types.Addon, error) {

	input := &eksv2.CreateAddonInput{
		ClusterName:      awsv2.String(clusterID),
		AddonName:        awsv2.String(ebsCSIDriverAddonName),
		ResolveConflicts: eksv2types.ResolveConflictsOverwrite,
	}
	if version != "" {
		input.AddonVersion = awsv2.String(version)
	}

	roleArn, err := ClusterHandler.getOrCreateEBSCSIDriverRole()
	if err != nil {
		// Same as cluster creation: fall back to the node role (through IMDS) rather than failing.
		cblogger.Errorf("Failed to get or create EBS CSI driver role, installing without Pod Identity: %v", err)
	} else {
		input.PodIdentityAssociations = ebsCSIPodIdentityAssociations(roleArn)
	}

	callLogInfo := GetCallLogScheme(ClusterHandler.Region, call.CLUSTER, clusterID, "CreateAddon()")
	start := call.Start()
	output, err := client.CreateAddon(ctx, input)
	if err != nil {
		LoggingError(callLogInfo, err)
		return nil, fmt.Errorf("failed to create %s add-on: %w", ebsCSIDriverAddonName, err)
	}
	LoggingInfo(callLogInfo, start)
	return output.Addon, nil
}

// reconcileInstalledEBSCSIDriverAddon handles EnableStorageDriver for an add-on that already exists.
// It does nothing if the add-on is already at the requested version, and updates the version otherwise.
func (ClusterHandler *AwsClusterHandler) reconcileInstalledEBSCSIDriverAddon(ctx context.Context, client *eksv2.Client,
	clusterID string, addon *eksv2types.Addon, version string) (irs.StorageDriverInfo, error) {

	info := convertEKSAddonToStorageDriverInfo(addon)

	switch addon.Status {
	case eksv2types.AddonStatusDeleting:
		return info, fmt.Errorf("%s add-on is being deleted; enable it again after the deletion completes", ebsCSIDriverAddonName)
	case eksv2types.AddonStatusCreateFailed, eksv2types.AddonStatusDeleteFailed:
		return info, fmt.Errorf("%s add-on is in %s state; disable it and enable it again", ebsCSIDriverAddonName, addon.Status)
	case eksv2types.AddonStatusCreating, eksv2types.AddonStatusUpdating:
		// Another change is in progress. Report it as is.
		return info, nil
	}

	// An add-on installed by an earlier CB-Spider version or by the user may have no Pod Identity association.
	if err := ClusterHandler.ensureEBSCSIDriverPodIdentity(clusterID); err != nil {
		cblogger.Warnf("Failed to ensure %s pod identity: %v", ebsCSIDriverAddonName, err)
	}

	if version == "" || version == awsv2.ToString(addon.AddonVersion) {
		return info, nil
	}

	callLogInfo := GetCallLogScheme(ClusterHandler.Region, call.CLUSTER, clusterID, "UpdateAddon()")
	start := call.Start()
	_, err := client.UpdateAddon(ctx, &eksv2.UpdateAddonInput{
		ClusterName:      awsv2.String(clusterID),
		AddonName:        awsv2.String(ebsCSIDriverAddonName),
		AddonVersion:     awsv2.String(version),
		ResolveConflicts: eksv2types.ResolveConflictsOverwrite,
	})
	if err != nil {
		LoggingError(callLogInfo, err)
		err = fmt.Errorf("failed to update %s add-on to version %s: %w", ebsCSIDriverAddonName, version, err)
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

	if err := checkEKSStorageDriverKind(kind); err != nil {
		return false, err
	}
	if err := ClusterHandler.checkEKSClusterActive(clusterID); err != nil {
		cblogger.Error(err)
		return false, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	client := ClusterHandler.newEKSV2Client()

	addon, err := ClusterHandler.describeEKSAddon(ctx, client, clusterID, ebsCSIDriverAddonName)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}
	if addon == nil || addon.Status == eksv2types.AddonStatusDeleting {
		return true, nil
	}

	// Preserve=false removes the add-on's Kubernetes resources (controller, node plugin, CSIDriver).
	// EBS volumes and the IAM role are not deleted.
	callLogInfo := GetCallLogScheme(ClusterHandler.Region, call.CLUSTER, clusterID, "DeleteAddon()")
	start := call.Start()
	_, err = client.DeleteAddon(ctx, &eksv2.DeleteAddonInput{
		ClusterName: awsv2.String(clusterID),
		AddonName:   awsv2.String(ebsCSIDriverAddonName),
		Preserve:    false,
	})
	if err != nil {
		var notFound *eksv2types.ResourceNotFoundException
		if errors.As(err, &notFound) {
			LoggingInfo(callLogInfo, start)
			return true, nil
		}
		LoggingError(callLogInfo, err)
		err = fmt.Errorf("failed to delete %s add-on: %w", ebsCSIDriverAddonName, err)
		cblogger.Error(err)
		return false, err
	}
	LoggingInfo(callLogInfo, start)
	return true, nil
}
