package resources

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
)

type OracleDiskHandler struct {
	Region             idrv.RegionInfo
	CompartmentID      string
	BlockstorageClient core.BlockstorageClient
	ComputeClient      core.ComputeClient
	Ctx                context.Context
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// diskTypeToVpus converts a CB-Spider DiskType string to OCI VpusPerGB.
// OCI performance tiers:
//
//	0  → Lower Cost
//	10 → Balanced   (default)
//	20 → Higher Performance
//	30 → Ultra High Performance
func diskTypeToVpus(diskType string) *int64 {
	switch strings.ToLower(strings.ReplaceAll(diskType, " ", "")) {
	case "lowercost":
		v := int64(0)
		return &v
	case "higherperformance":
		v := int64(20)
		return &v
	case "ultrahighperformance":
		v := int64(30)
		return &v
	default: // "balanced" or empty/unrecognised
		v := int64(10)
		return &v
	}
}

// vpusToDiskType converts OCI VpusPerGB to a human-readable DiskType string.
func vpusToDiskType(vpus *int64) string {
	if vpus == nil {
		return "Balanced"
	}
	switch *vpus {
	case 0:
		return "Lower Cost"
	case 10:
		return "Balanced"
	case 20:
		return "Higher Performance"
	case 30:
		return "Ultra High Performance"
	default:
		return fmt.Sprintf("VpusPerGB-%d", *vpus)
	}
}

// volumeStateToDiskStatus maps OCI Volume lifecycle states to CB-Spider DiskStatus.
func volumeStateToDiskStatus(state core.VolumeLifecycleStateEnum) irs.DiskStatus {
	switch state {
	case core.VolumeLifecycleStateProvisioning, core.VolumeLifecycleStateRestoring:
		return irs.DiskCreating
	case core.VolumeLifecycleStateAvailable:
		return irs.DiskAvailable
	case core.VolumeLifecycleStateTerminating:
		return irs.DiskDeleting
	default:
		return irs.DiskError
	}
}

// volumeToIID builds a CB-Spider IID from an OCI Volume.
func (h *OracleDiskHandler) volumeToIID(v core.Volume) irs.IID {
	return irs.IID{NameId: stringValue(v.DisplayName), SystemId: stringValue(v.Id)}
}

// activeAttachmentForVolume returns the current ATTACHED attachment for a volume, if any.
func (h *OracleDiskHandler) activeAttachmentForVolume(volumeID string) (core.VolumeAttachment, bool, error) {
	resp, err := h.ComputeClient.ListVolumeAttachments(h.Ctx, core.ListVolumeAttachmentsRequest{
		CompartmentId: common.String(h.CompartmentID),
		VolumeId:      common.String(volumeID),
	})
	if err != nil {
		return nil, false, fmt.Errorf("failed to list volume attachments: %w", err)
	}
	for _, att := range resp.Items {
		if att.GetLifecycleState() == core.VolumeAttachmentLifecycleStateAttached {
			return att, true, nil
		}
	}
	return nil, false, nil
}

// volumeToDiskInfo converts an OCI Volume to a CB-Spider DiskInfo.
func (h *OracleDiskHandler) volumeToDiskInfo(v core.Volume) (irs.DiskInfo, error) {
	info := irs.DiskInfo{
		IId:      h.volumeToIID(v),
		Zone:     stringValue(v.AvailabilityDomain),
		DiskType: vpusToDiskType(v.VpusPerGB),
		Status:   volumeStateToDiskStatus(v.LifecycleState),
	}
	if v.TimeCreated != nil {
		info.CreatedTime = v.TimeCreated.Time
	}
	if v.SizeInGBs != nil {
		info.DiskSize = strconv.FormatInt(*v.SizeInGBs, 10)
	}

	// Determine if the volume is currently attached
	att, attached, err := h.activeAttachmentForVolume(stringValue(v.Id))
	if err != nil {
		return irs.DiskInfo{}, err
	}
	if attached {
		info.Status = irs.DiskAttached
		info.OwnerVM = irs.IID{SystemId: stringValue(att.GetInstanceId())}
	}

	// FreeformTags → TagList
	for k, val := range v.FreeformTags {
		info.TagList = append(info.TagList, irs.KeyValue{Key: k, Value: val})
	}
	return info, nil
}

// listAllVolumes fetches every Volume in the compartment (handles pagination).
func (h *OracleDiskHandler) listAllVolumes() ([]core.Volume, error) {
	var all []core.Volume
	req := core.ListVolumesRequest{CompartmentId: common.String(h.CompartmentID)}
	for {
		resp, err := h.BlockstorageClient.ListVolumes(h.Ctx, req)
		if err != nil {
			return nil, fmt.Errorf("failed to list volumes: %w", err)
		}
		all = append(all, resp.Items...)
		if resp.OpcNextPage == nil {
			break
		}
		req.Page = resp.OpcNextPage
	}
	return all, nil
}

// findVolumeByIID resolves an IID to an OCI Volume.
// SystemId is preferred; NameId is used when SystemId is empty.
func (h *OracleDiskHandler) findVolumeByIID(diskIID irs.IID) (core.Volume, error) {
	if diskIID.SystemId != "" {
		resp, err := h.BlockstorageClient.GetVolume(h.Ctx, core.GetVolumeRequest{
			VolumeId: common.String(diskIID.SystemId),
		})
		if err != nil {
			return core.Volume{}, fmt.Errorf("failed to get volume (%s): %w", diskIID.SystemId, err)
		}
		return resp.Volume, nil
	}
	// Fall back to searching by DisplayName
	volumes, err := h.listAllVolumes()
	if err != nil {
		return core.Volume{}, err
	}
	for _, v := range volumes {
		if stringValue(v.DisplayName) == diskIID.NameId {
			return v, nil
		}
	}
	return core.Volume{}, fmt.Errorf("disk not found: %s", diskIID.NameId)
}

// waitForVolumeState polls until the volume reaches one of the target states.
func (h *OracleDiskHandler) waitForVolumeState(volumeID string, targets []core.VolumeLifecycleStateEnum) (core.Volume, error) {
	const maxWait = 10 * time.Minute
	const poll = 5 * time.Second
	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		resp, err := h.BlockstorageClient.GetVolume(h.Ctx, core.GetVolumeRequest{
			VolumeId: common.String(volumeID),
		})
		if err != nil {
			return core.Volume{}, fmt.Errorf("failed to poll volume state: %w", err)
		}
		for _, t := range targets {
			if resp.Volume.LifecycleState == t {
				return resp.Volume, nil
			}
		}
		time.Sleep(poll)
	}
	return core.Volume{}, fmt.Errorf("timeout waiting for volume %s to reach target state", volumeID)
}

// waitForAttachmentState polls until an attachment for the volume reaches the target state.
func (h *OracleDiskHandler) waitForAttachmentState(volumeID string, target core.VolumeAttachmentLifecycleStateEnum) (core.VolumeAttachment, error) {
	const maxWait = 10 * time.Minute
	const poll = 5 * time.Second
	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		resp, err := h.ComputeClient.ListVolumeAttachments(h.Ctx, core.ListVolumeAttachmentsRequest{
			CompartmentId: common.String(h.CompartmentID),
			VolumeId:      common.String(volumeID),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to poll attachment state: %w", err)
		}
		for _, att := range resp.Items {
			if att.GetLifecycleState() == target {
				return att, nil
			}
		}
		time.Sleep(poll)
	}
	return nil, fmt.Errorf("timeout waiting for volume %s attachment to reach %s", volumeID, target)
}

// ---------------------------------------------------------------------------
// DiskHandler interface implementation
// ---------------------------------------------------------------------------

// ListIID returns all volume IIDs in the compartment.
func (h *OracleDiskHandler) ListIID() ([]*irs.IID, error) {
	volumes, err := h.listAllVolumes()
	if err != nil {
		return nil, err
	}
	result := make([]*irs.IID, 0, len(volumes))
	for i := range volumes {
		iid := h.volumeToIID(volumes[i])
		result = append(result, &iid)
	}
	return result, nil
}

// CreateDisk creates a new OCI block volume.
func (h *OracleDiskHandler) CreateDisk(diskReqInfo irs.DiskInfo) (irs.DiskInfo, error) {
	if diskReqInfo.IId.NameId == "" {
		return irs.DiskInfo{}, errors.New("disk NameId is required")
	}

	var sizeInGBs *int64
	if diskReqInfo.DiskSize != "" && diskReqInfo.DiskSize != "default" {
		s, err := strconv.ParseInt(diskReqInfo.DiskSize, 10, 64)
		if err != nil {
			return irs.DiskInfo{}, fmt.Errorf("invalid disk size value: %s", diskReqInfo.DiskSize)
		}
		if s < 50 {
			return irs.DiskInfo{}, fmt.Errorf("disk size must be at least 50 GB, got %d", s)
		}
		sizeInGBs = common.Int64(s)
	}

	resp, err := h.BlockstorageClient.CreateVolume(h.Ctx, core.CreateVolumeRequest{
		CreateVolumeDetails: core.CreateVolumeDetails{
			CompartmentId:      common.String(h.CompartmentID),
			AvailabilityDomain: common.String(h.Region.Zone),
			DisplayName:        common.String(diskReqInfo.IId.NameId),
			SizeInGBs:          sizeInGBs,
			VpusPerGB:          diskTypeToVpus(diskReqInfo.DiskType),
		},
	})
	if err != nil {
		return irs.DiskInfo{}, fmt.Errorf("failed to create volume: %w", err)
	}

	volume, err := h.waitForVolumeState(stringValue(resp.Volume.Id), []core.VolumeLifecycleStateEnum{
		core.VolumeLifecycleStateAvailable,
		core.VolumeLifecycleStateFaulty,
	})
	if err != nil {
		return irs.DiskInfo{}, err
	}
	if volume.LifecycleState == core.VolumeLifecycleStateFaulty {
		return irs.DiskInfo{}, fmt.Errorf("volume creation failed: volume is in FAULTY state")
	}
	return h.volumeToDiskInfo(volume)
}

// ListDisk returns all block volumes in the compartment.
func (h *OracleDiskHandler) ListDisk() ([]*irs.DiskInfo, error) {
	volumes, err := h.listAllVolumes()
	if err != nil {
		return nil, err
	}
	result := make([]*irs.DiskInfo, 0, len(volumes))
	for i := range volumes {
		info, err := h.volumeToDiskInfo(volumes[i])
		if err != nil {
			return nil, err
		}
		result = append(result, &info)
	}
	return result, nil
}

// GetDisk returns info for a specific block volume identified by IID.
func (h *OracleDiskHandler) GetDisk(diskIID irs.IID) (irs.DiskInfo, error) {
	volume, err := h.findVolumeByIID(diskIID)
	if err != nil {
		return irs.DiskInfo{}, err
	}
	return h.volumeToDiskInfo(volume)
}

// ChangeDiskSize resizes a block volume (OCI only allows increasing size).
func (h *OracleDiskHandler) ChangeDiskSize(diskIID irs.IID, size string) (bool, error) {
	newSize, err := strconv.ParseInt(size, 10, 64)
	if err != nil {
		return false, fmt.Errorf("invalid size value: %s", size)
	}
	if newSize < 50 {
		return false, fmt.Errorf("disk size must be at least 50 GB, got %d", newSize)
	}

	volume, err := h.findVolumeByIID(diskIID)
	if err != nil {
		return false, err
	}
	if volume.SizeInGBs != nil && *volume.SizeInGBs >= newSize {
		return false, fmt.Errorf("new size (%d GB) must be greater than current size (%d GB)", newSize, *volume.SizeInGBs)
	}

	_, err = h.BlockstorageClient.UpdateVolume(h.Ctx, core.UpdateVolumeRequest{
		VolumeId: volume.Id,
		UpdateVolumeDetails: core.UpdateVolumeDetails{
			SizeInGBs: common.Int64(newSize),
		},
	})
	if err != nil {
		return false, fmt.Errorf("failed to resize volume: %w", err)
	}

	_, err = h.waitForVolumeState(stringValue(volume.Id), []core.VolumeLifecycleStateEnum{
		core.VolumeLifecycleStateAvailable,
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

// DeleteDisk deletes a block volume by IID.
func (h *OracleDiskHandler) DeleteDisk(diskIID irs.IID) (bool, error) {
	volume, err := h.findVolumeByIID(diskIID)
	if err != nil {
		return false, err
	}

	_, err = h.BlockstorageClient.DeleteVolume(h.Ctx, core.DeleteVolumeRequest{
		VolumeId: volume.Id,
	})
	if err != nil {
		return false, fmt.Errorf("failed to delete volume: %w", err)
	}

	_, err = h.waitForVolumeState(stringValue(volume.Id), []core.VolumeLifecycleStateEnum{
		core.VolumeLifecycleStateTerminated,
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

// AttachDisk attaches a block volume to a VM using paravirtualized attachment.
func (h *OracleDiskHandler) AttachDisk(diskIID irs.IID, ownerVM irs.IID) (irs.DiskInfo, error) {
	volume, err := h.findVolumeByIID(diskIID)
	if err != nil {
		return irs.DiskInfo{}, err
	}
	if ownerVM.SystemId == "" {
		return irs.DiskInfo{}, errors.New("ownerVM SystemId is required for AttachDisk")
	}

	_, err = h.ComputeClient.AttachVolume(h.Ctx, core.AttachVolumeRequest{
		AttachVolumeDetails: core.AttachParavirtualizedVolumeDetails{
			InstanceId: common.String(ownerVM.SystemId),
			VolumeId:   volume.Id,
		},
	})
	if err != nil {
		return irs.DiskInfo{}, fmt.Errorf("failed to attach volume: %w", err)
	}

	_, err = h.waitForAttachmentState(stringValue(volume.Id), core.VolumeAttachmentLifecycleStateAttached)
	if err != nil {
		return irs.DiskInfo{}, err
	}
	return h.GetDisk(diskIID)
}

// DetachDisk detaches a block volume from its owning VM.
func (h *OracleDiskHandler) DetachDisk(diskIID irs.IID, ownerVM irs.IID) (bool, error) {
	volume, err := h.findVolumeByIID(diskIID)
	if err != nil {
		return false, err
	}

	resp, err := h.ComputeClient.ListVolumeAttachments(h.Ctx, core.ListVolumeAttachmentsRequest{
		CompartmentId: common.String(h.CompartmentID),
		VolumeId:      volume.Id,
	})
	if err != nil {
		return false, fmt.Errorf("failed to list volume attachments: %w", err)
	}

	var attachmentID *string
	for _, att := range resp.Items {
		state := att.GetLifecycleState()
		if state == core.VolumeAttachmentLifecycleStateAttached ||
			state == core.VolumeAttachmentLifecycleStateAttaching {
			id := att.GetId()
			attachmentID = id
			break
		}
	}
	if attachmentID == nil {
		return false, fmt.Errorf("no active attachment found for volume: %s", stringValue(volume.DisplayName))
	}

	_, err = h.ComputeClient.DetachVolume(h.Ctx, core.DetachVolumeRequest{
		VolumeAttachmentId: attachmentID,
	})
	if err != nil {
		return false, fmt.Errorf("failed to detach volume: %w", err)
	}

	// Poll until detached
	const maxWait = 10 * time.Minute
	const poll = 5 * time.Second
	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		attResp, err := h.ComputeClient.GetVolumeAttachment(h.Ctx, core.GetVolumeAttachmentRequest{
			VolumeAttachmentId: attachmentID,
		})
		if err != nil {
			// Attachment resource may already be gone
			break
		}
		if attResp.VolumeAttachment.GetLifecycleState() == core.VolumeAttachmentLifecycleStateDetached {
			break
		}
		time.Sleep(poll)
	}
	return true, nil
}
