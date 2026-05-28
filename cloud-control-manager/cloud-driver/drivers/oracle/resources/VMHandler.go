package resources

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
)

type OracleVMHandler struct {
	CredentialInfo       idrv.CredentialInfo
	Region               idrv.RegionInfo
	CompartmentID        string
	ComputeClient        core.ComputeClient
	VirtualNetworkClient core.VirtualNetworkClient
	BlockstorageClient   core.BlockstorageClient
	Ctx                  context.Context
}

func (handler *OracleVMHandler) StartVM(req irs.VMReqInfo) (irs.VMInfo, error) {
	if req.IId.NameId == "" || req.ImageIID.SystemId == "" || req.SubnetIID.SystemId == "" || req.VMSpecName == "" || req.KeyPairIID.NameId == "" {
		return irs.VMInfo{}, errors.New("invalid VM request")
	}
	imageID, err := handler.resolveImageID(req.ImageIID)
	if err != nil {
		return irs.VMInfo{}, err
	}
	shape, err := handler.getShapeForImage(req.VMSpecName, imageID)
	if err != nil {
		return irs.VMInfo{}, err
	}
	keyHandler := OracleKeyPairHandler{CredentialInfo: handler.CredentialInfo, Region: handler.Region}
	keyInfo, err := keyHandler.GetKey(req.KeyPairIID)
	if err != nil {
		return irs.VMInfo{}, err
	}
	nsgIDs := make([]string, 0, len(req.SecurityGroupIIDs))
	for _, sgIID := range req.SecurityGroupIIDs {
		if sgIID.SystemId != "" {
			nsgIDs = append(nsgIDs, sgIID.SystemId)
		}
	}
	userData, err := cloudInitUserData(keyInfo.PublicKey)
	if err != nil {
		return irs.VMInfo{}, err
	}
	metadata := map[string]string{"ssh_authorized_keys": keyInfo.PublicKey, "user_data": userData}
	vmTags := freeformTagsWith(req.TagList, map[string]string{oracleVMKeyPairNameTag: req.KeyPairIID.NameId, oracleVMUserIDTag: defaultVMUserID})
	details := core.LaunchInstanceDetails{CompartmentId: common.String(handler.CompartmentID), AvailabilityDomain: common.String(handler.Region.Zone), DisplayName: common.String(req.IId.NameId), Shape: common.String(req.VMSpecName), SourceDetails: core.InstanceSourceViaImageDetails{ImageId: common.String(imageID)}, CreateVnicDetails: &core.CreateVnicDetails{SubnetId: common.String(req.SubnetIID.SystemId), AssignPublicIp: common.Bool(true), DisplayName: common.String(req.IId.NameId), HostnameLabel: common.String(dnsLabel(req.IId.NameId)), NsgIds: nsgIDs, FreeformTags: vmTags}, Metadata: metadata, FreeformTags: vmTags}
	shapeConfig, err := handler.launchShapeConfig(shape)
	if err != nil {
		return irs.VMInfo{}, err
	}
	if shapeConfig != nil {
		details.ShapeConfig = shapeConfig
	}
	if req.RootDiskSize != "" && req.RootDiskSize != "default" {
		if size, err := strconv.ParseInt(req.RootDiskSize, 10, 64); err == nil {
			details.SourceDetails = core.InstanceSourceViaImageDetails{ImageId: common.String(imageID), BootVolumeSizeInGBs: common.Int64(size)}
		}
	}
	retryPolicy := common.DefaultRetryPolicyWithoutEventualConsistency()
	resp, err := handler.ComputeClient.LaunchInstance(handler.Ctx, core.LaunchInstanceRequest{LaunchInstanceDetails: details, OpcRetryToken: common.String(newOracleRetryToken(req.IId.NameId)), RequestMetadata: common.RequestMetadata{RetryPolicy: &retryPolicy}})
	if err != nil {
		return irs.VMInfo{}, statusErr("failed to launch Oracle instance", err)
	}
	instance, err := handler.waitForInstanceState(stringValue(resp.Instance.Id),
		[]core.InstanceLifecycleStateEnum{core.InstanceLifecycleStateRunning})
	if err != nil {
		return irs.VMInfo{}, err
	}
	return handler.vmInfo(instance)
}

func newOracleRetryToken(name string) string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("cb-spider-oracle-%s", dnsLabel(name))
	}
	return fmt.Sprintf("cb-spider-oracle-%s-%s", dnsLabel(name), hex.EncodeToString(bytes))
}

func (handler *OracleVMHandler) resolveImageID(iid irs.IID) (string, error) {
	id := iid.SystemId
	if id == "" {
		id = iid.NameId
	}
	if isOracleOCID(id) {
		return id, nil
	}
	imageHandler := OracleImageHandler{Region: handler.Region, CompartmentID: handler.CompartmentID, Client: handler.ComputeClient, Ctx: handler.Ctx}
	image, err := imageHandler.GetImage(iid)
	if err != nil {
		return "", err
	}
	if image.IId.SystemId == "" || !isOracleOCID(image.IId.SystemId) {
		return "", fmt.Errorf("Oracle image OCID not found: %s", id)
	}
	return image.IId.SystemId, nil
}

func (handler *OracleVMHandler) getShapeForImage(shapeName string, imageID string) (core.Shape, error) {
	resp, err := handler.ComputeClient.ListShapes(handler.Ctx, core.ListShapesRequest{CompartmentId: common.String(handler.CompartmentID), AvailabilityDomain: common.String(handler.Region.Zone), ImageId: common.String(imageID), Shape: common.String(shapeName)})
	if err != nil {
		return core.Shape{}, statusErr("failed to validate Oracle image and VM spec compatibility", err)
	}
	if len(resp.Items) > 0 {
		return resp.Items[0], nil
	}
	validShapes, err := handler.ComputeClient.ListShapes(handler.Ctx, core.ListShapesRequest{CompartmentId: common.String(handler.CompartmentID), AvailabilityDomain: common.String(handler.Region.Zone), ImageId: common.String(imageID)})
	if err != nil {
		return core.Shape{}, statusErr("failed to list Oracle VM specs compatible with image", err)
	}
	shapeNames := make([]string, 0, len(validShapes.Items))
	for index, shape := range validShapes.Items {
		if index >= 10 {
			shapeNames = append(shapeNames, "...")
			break
		}
		shapeNames = append(shapeNames, stringValue(shape.Shape))
	}
	return core.Shape{}, fmt.Errorf("Oracle VM spec %s is not compatible with image %s. Compatible specs include: %s", shapeName, imageID, strings.Join(shapeNames, ", "))
}

func (handler *OracleVMHandler) launchShapeConfig(shape core.Shape) (*core.LaunchInstanceShapeConfigDetails, error) {
	if shape.IsFlexible == nil || !*shape.IsFlexible {
		return nil, nil
	}
	if shape.Ocpus == nil || shape.MemoryInGBs == nil {
		return nil, fmt.Errorf("Oracle flexible VM spec %s has no default OCPU or memory values from OCI", stringValue(shape.Shape))
	}
	return &core.LaunchInstanceShapeConfigDetails{Ocpus: shape.Ocpus, MemoryInGBs: shape.MemoryInGBs}, nil
}

func (handler *OracleVMHandler) SuspendVM(iid irs.IID) (irs.VMStatus, error) {
	return handler.instanceAction(iid, core.InstanceActionActionStop)
}

func (handler *OracleVMHandler) ResumeVM(iid irs.IID) (irs.VMStatus, error) {
	return handler.instanceAction(iid, core.InstanceActionActionStart)
}

func (handler *OracleVMHandler) RebootVM(iid irs.IID) (irs.VMStatus, error) {
	return handler.instanceAction(iid, core.InstanceActionActionSoftreset)
}

func (handler *OracleVMHandler) TerminateVM(iid irs.IID) (irs.VMStatus, error) {
	instance, err := handler.getInstance(iid)
	if err != nil {
		return irs.NotExist, err
	}
	_, err = handler.ComputeClient.TerminateInstance(handler.Ctx, core.TerminateInstanceRequest{InstanceId: instance.Id})
	if err != nil {
		return irs.Failed, statusErr("failed to terminate Oracle instance", err)
	}
	return irs.Terminating, nil
}

func (handler *OracleVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
	instances, err := handler.listInstances()
	if err != nil {
		return nil, err
	}
	statuses := make([]*irs.VMStatusInfo, 0, len(instances))
	for _, instance := range instances {
		statuses = append(statuses, &irs.VMStatusInfo{IId: irs.IID{NameId: stringValue(instance.DisplayName), SystemId: stringValue(instance.Id)}, VmStatus: toCBVMStatus(instance.LifecycleState)})
	}
	return statuses, nil
}

func (handler *OracleVMHandler) GetVMStatus(iid irs.IID) (irs.VMStatus, error) {
	instance, err := handler.getInstance(iid)
	if err != nil {
		return irs.NotExist, err
	}
	return toCBVMStatus(instance.LifecycleState), nil
}

func (handler *OracleVMHandler) ListVM() ([]*irs.VMInfo, error) {
	instances, err := handler.listInstances()
	if err != nil {
		return nil, err
	}
	infos := make([]*irs.VMInfo, 0, len(instances))
	for _, instance := range instances {
		info, err := handler.vmInfo(instance)
		if err == nil {
			infos = append(infos, &info)
		}
	}
	return infos, nil
}

func (handler *OracleVMHandler) GetVM(iid irs.IID) (irs.VMInfo, error) {
	instance, err := handler.getInstance(iid)
	if err != nil {
		return irs.VMInfo{}, err
	}
	return handler.vmInfo(instance)
}

func (handler *OracleVMHandler) ListIID() ([]*irs.IID, error) {
	infos, err := handler.ListVM()
	if err != nil {
		return nil, err
	}
	iids := make([]*irs.IID, 0, len(infos))
	for _, info := range infos {
		iids = append(iids, &info.IId)
	}
	return iids, nil
}

func (handler *OracleVMHandler) instanceAction(iid irs.IID, action core.InstanceActionActionEnum) (irs.VMStatus, error) {
	instance, err := handler.getInstance(iid)
	if err != nil {
		return irs.NotExist, err
	}
	resp, err := handler.ComputeClient.InstanceAction(handler.Ctx, core.InstanceActionRequest{InstanceId: instance.Id, Action: action})
	if err != nil {
		return irs.Failed, statusErr("failed to run Oracle instance action", err)
	}
	return toCBVMStatus(resp.Instance.LifecycleState), nil
}

func (handler *OracleVMHandler) getInstance(iid irs.IID) (core.Instance, error) {
	id, displayName := idFilter(iid)
	if id != nil {
		resp, err := handler.ComputeClient.GetInstance(handler.Ctx, core.GetInstanceRequest{InstanceId: id})
		return resp.Instance, err
	}
	resp, err := handler.ComputeClient.ListInstances(handler.Ctx, core.ListInstancesRequest{CompartmentId: common.String(handler.CompartmentID), DisplayName: displayName})
	if err != nil {
		return core.Instance{}, err
	}
	if len(resp.Items) == 0 {
		return core.Instance{}, fmt.Errorf("Oracle instance not found: %s", iid.NameId)
	}
	return resp.Items[0], nil
}

func (handler *OracleVMHandler) listInstances() ([]core.Instance, error) {
	resp, err := handler.ComputeClient.ListInstances(handler.Ctx, core.ListInstancesRequest{CompartmentId: common.String(handler.CompartmentID)})
	if err != nil {
		return nil, statusErr("failed to list Oracle instances", err)
	}
	return resp.Items, nil
}

// dataDisksForInstance returns IIDs of all data volumes currently attached to an instance.
func (handler *OracleVMHandler) dataDisksForInstance(instanceID string) []irs.IID {
	resp, err := handler.ComputeClient.ListVolumeAttachments(handler.Ctx, core.ListVolumeAttachmentsRequest{
		CompartmentId: common.String(handler.CompartmentID),
		InstanceId:    common.String(instanceID),
	})
	if err != nil {
		return nil
	}
	var iids []irs.IID
	for _, att := range resp.Items {
		if att.GetLifecycleState() != core.VolumeAttachmentLifecycleStateAttached {
			continue
		}
		volumeID := stringValue(att.GetVolumeId())
		iid := irs.IID{SystemId: volumeID}
		volResp, err := handler.BlockstorageClient.GetVolume(handler.Ctx, core.GetVolumeRequest{
			VolumeId: common.String(volumeID),
		})
		if err == nil {
			iid.NameId = stringValue(volResp.Volume.DisplayName)
		}
		iids = append(iids, iid)
	}
	return iids
}

// imageTypeForID returns MyImage if the given image OCID is a custom (user-created)
// image, otherwise PublicImage. Custom images have BaseImageId set by OCI.
func (handler *OracleVMHandler) imageTypeForID(imageID string) irs.ImageType {
	if imageID == "" {
		return irs.PublicImage
	}
	resp, err := handler.ComputeClient.GetImage(handler.Ctx, core.GetImageRequest{
		ImageId: common.String(imageID),
	})
	if err != nil {
		return irs.PublicImage
	}
	if resp.Image.BaseImageId != nil && *resp.Image.BaseImageId != "" {
		return irs.MyImage
	}
	return irs.PublicImage
}

func (handler *OracleVMHandler) vmInfo(instance core.Instance) (irs.VMInfo, error) {
	vnic, subnetID, vcnID, err := handler.primaryVnic(stringValue(instance.Id))
	if err != nil {
		return irs.VMInfo{}, err
	}
	imageID := stringValue(instance.ImageId)
	if source, ok := instance.SourceDetails.(core.InstanceSourceViaImageDetails); ok {
		imageID = stringValue(source.ImageId)
	}
	platform := irs.LINUX_UNIX
	keyPairName := tagValue(instance.FreeformTags, oracleVMKeyPairNameTag)
	vmUserID := tagValue(instance.FreeformTags, oracleVMUserIDTag)
	if vmUserID == "" {
		vmUserID = defaultVMUserID
	}
	dataDisks := handler.dataDisksForInstance(stringValue(instance.Id))
	imageType := handler.imageTypeForID(imageID)
	return irs.VMInfo{IId: irs.IID{NameId: stringValue(instance.DisplayName), SystemId: stringValue(instance.Id)}, StartTime: timeValue(instance.TimeCreated), Region: irs.RegionInfo{Region: handler.Region.Region, Zone: stringValue(instance.AvailabilityDomain)}, ImageType: imageType, ImageIId: irs.IID{SystemId: imageID, NameId: imageID}, VMSpecName: stringValue(instance.Shape), VpcIID: irs.IID{SystemId: vcnID}, SubnetIID: irs.IID{SystemId: subnetID}, SecurityGroupIIds: nsgIIDs(vnic.NsgIds), KeyPairIId: irs.IID{NameId: keyPairName, SystemId: keyPairName}, RootDiskType: "default", RootDiskSize: "default", RootDeviceName: "boot", DataDiskIIDs: dataDisks, VMUserId: vmUserID, NetworkInterface: stringValue(vnic.Id), PublicIP: stringValue(vnic.PublicIp), PrivateIP: stringValue(vnic.PrivateIp), Platform: platform, AccessPoint: accessPoint(stringValue(vnic.PublicIp)), TagList: tagList(instance.FreeformTags)}, nil
}

func cloudInitUserData(publicKey string) (string, error) {
	rootPath := os.Getenv("CBSPIDER_ROOT")
	fileData, err := os.ReadFile(rootPath + oracleCloudInitPath)
	if err != nil {
		return "", fmt.Errorf("failed to read Oracle cloud-init template: %w", err)
	}
	userData := strings.ReplaceAll(string(fileData), cloudInitPublicKeyVar, strings.TrimSpace(publicKey))
	return base64.StdEncoding.EncodeToString([]byte(userData)), nil
}

func (handler *OracleVMHandler) primaryVnic(instanceID string) (core.Vnic, string, string, error) {
	attachments, err := handler.waitPrimaryVnicAttachment(instanceID)
	if err != nil {
		return core.Vnic{}, "", "", err
	}
	attachment := attachments.Items[0]
	subnetID := stringValue(attachment.SubnetId)
	resp, err := handler.VirtualNetworkClient.GetVnic(handler.Ctx, core.GetVnicRequest{VnicId: attachment.VnicId})
	if err != nil {
		vcnID, subnetErr := handler.subnetVcnID(subnetID)
		if subnetErr != nil {
			return core.Vnic{}, "", "", subnetErr
		}
		return core.Vnic{}, subnetID, vcnID, nil
	}
	if stringValue(resp.Vnic.SubnetId) != "" {
		subnetID = stringValue(resp.Vnic.SubnetId)
	}
	vcnID, err := handler.subnetVcnID(subnetID)
	if err != nil {
		return core.Vnic{}, "", "", err
	}
	return resp.Vnic, subnetID, vcnID, nil
}

// waitForInstanceState polls until the instance reaches one of the target lifecycle states.
// Each API call uses an independent short-lived context so that the connection context
// timeout (cspTimeout=10min) does not abort long-running MyImage provisioning.
func (handler *OracleVMHandler) waitForInstanceState(instanceID string, targets []core.InstanceLifecycleStateEnum) (core.Instance, error) {
	const maxWait = 30 * time.Minute
	const poll = 10 * time.Second
	const apiTimeout = 30 * time.Second
	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
		getResp, err := handler.ComputeClient.GetInstance(ctx, core.GetInstanceRequest{
			InstanceId: common.String(instanceID),
		})
		cancel()
		if err != nil {
			return core.Instance{}, fmt.Errorf("failed to poll instance state: %w", err)
		}
		for _, t := range targets {
			if getResp.Instance.LifecycleState == t {
				return getResp.Instance, nil
			}
		}
		time.Sleep(poll)
	}
	return core.Instance{}, fmt.Errorf("timeout waiting for instance %s to reach target state", instanceID)
}

func (handler *OracleVMHandler) waitPrimaryVnicAttachment(instanceID string) (core.ListVnicAttachmentsResponse, error) {
	const apiTimeout = 30 * time.Second
	deadline := time.Now().Add(5 * time.Minute)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
		attachments, err := handler.ComputeClient.ListVnicAttachments(ctx, core.ListVnicAttachmentsRequest{CompartmentId: common.String(handler.CompartmentID), InstanceId: common.String(instanceID)})
		cancel()
		if err != nil {
			return core.ListVnicAttachmentsResponse{}, statusErr("failed to list Oracle VNIC attachments", err)
		}
		if len(attachments.Items) > 0 && attachments.Items[0].VnicId != nil {
			return attachments, nil
		}
		if time.Now().After(deadline) {
			return core.ListVnicAttachmentsResponse{}, fmt.Errorf("Oracle primary VNIC attachment not found: %s", instanceID)
		}
		time.Sleep(resourcePollInterval)
	}
}

func (handler *OracleVMHandler) subnetVcnID(subnetID string) (string, error) {
	if subnetID == "" {
		return "", nil
	}
	resp, err := handler.VirtualNetworkClient.GetSubnet(handler.Ctx, core.GetSubnetRequest{SubnetId: common.String(subnetID)})
	if err != nil {
		return "", statusErr("failed to get Oracle subnet for VM", err)
	}
	return stringValue(resp.Subnet.VcnId), nil
}

func nsgIIDs(nsgIDs []string) []irs.IID {
	iids := make([]irs.IID, 0, len(nsgIDs))
	for _, nsgID := range nsgIDs {
		iids = append(iids, irs.IID{SystemId: nsgID})
	}
	return iids
}

func accessPoint(publicIP string) string {
	if publicIP == "" {
		return ""
	}
	return publicIP + ":22"
}

func toCBVMStatus(state core.InstanceLifecycleStateEnum) irs.VMStatus {
	switch state {
	case core.InstanceLifecycleStateProvisioning, core.InstanceLifecycleStateStarting:
		return irs.Creating
	case core.InstanceLifecycleStateRunning:
		return irs.Running
	case core.InstanceLifecycleStateStopping:
		return irs.Suspending
	case core.InstanceLifecycleStateStopped:
		return irs.Suspended
	case core.InstanceLifecycleStateTerminating:
		return irs.Terminating
	case core.InstanceLifecycleStateTerminated:
		return irs.Terminated
	default:
		return irs.Failed
	}
}
