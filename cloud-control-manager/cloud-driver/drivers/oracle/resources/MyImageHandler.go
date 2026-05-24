package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
)

// myImageSourceVMTag is the FreeformTag key used to record the source VM OCID.
const myImageSourceVMTag = "cb-spider-source-vm"

type OracleMyImageHandler struct {
	Region        idrv.RegionInfo
	CompartmentID string
	Client        core.ComputeClient
	Ctx           context.Context
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func (h *OracleMyImageHandler) imageToMyImageInfo(img core.Image) irs.MyImageInfo {
	name := stringValue(img.DisplayName)
	if name == "" {
		name = stringValue(img.Id)
	}

	status := irs.MyImageUnavailable
	if img.LifecycleState == core.ImageLifecycleStateAvailable {
		status = irs.MyImageAvailable
	}

	info := irs.MyImageInfo{
		IId:    irs.IID{NameId: name, SystemId: stringValue(img.Id)},
		Status: status,
	}
	if img.TimeCreated != nil {
		info.CreatedTime = img.TimeCreated.Time
	}
	// Recover source VM OCID stored as a freeform tag
	if vmID, ok := img.FreeformTags[myImageSourceVMTag]; ok && vmID != "" {
		info.SourceVM = irs.IID{SystemId: vmID}
	}
	// FreeformTags → TagList (exclude internal cb-spider tags)
	for k, v := range img.FreeformTags {
		if k == myImageSourceVMTag {
			continue
		}
		info.TagList = append(info.TagList, irs.KeyValue{Key: k, Value: v})
	}
	return info
}

// isCustomImage returns true for images that were created from an instance
// (custom/snapshot images). OCI sets BaseImageId for custom images; platform
// images have no BaseImageId.
func isCustomImage(img core.Image) bool {
	return img.BaseImageId != nil && *img.BaseImageId != ""
}

// listCustomImages returns all custom images (non-platform) in the compartment.
func (h *OracleMyImageHandler) listCustomImages() ([]core.Image, error) {
	var all []core.Image
	req := core.ListImagesRequest{
		CompartmentId:  common.String(h.CompartmentID),
		LifecycleState: core.ImageLifecycleStateAvailable,
	}
	for {
		resp, err := h.Client.ListImages(h.Ctx, req)
		if err != nil {
			return nil, fmt.Errorf("failed to list custom images: %w", err)
		}
		for _, img := range resp.Items {
			if isCustomImage(img) {
				all = append(all, img)
			}
		}
		if resp.OpcNextPage == nil || *resp.OpcNextPage == "" {
			break
		}
		req.Page = resp.OpcNextPage
	}
	return all, nil
}

// findCustomImageByIID resolves an IID to an OCI Image among custom images.
func (h *OracleMyImageHandler) findCustomImageByIID(iid irs.IID) (core.Image, error) {
	if iid.SystemId != "" && isOracleOCID(iid.SystemId) {
		resp, err := h.Client.GetImage(h.Ctx, core.GetImageRequest{ImageId: common.String(iid.SystemId)})
		if err != nil {
			return core.Image{}, fmt.Errorf("failed to get custom image (%s): %w", iid.SystemId, err)
		}
		if !isCustomImage(resp.Image) {
			return core.Image{}, fmt.Errorf("image %s is not a custom image", iid.SystemId)
		}
		return resp.Image, nil
	}
	// Search by DisplayName
	name := iid.NameId
	if name == "" {
		name = iid.SystemId
	}
	images, err := h.listCustomImages()
	if err != nil {
		return core.Image{}, err
	}
	for _, img := range images {
		if stringValue(img.DisplayName) == name {
			return img, nil
		}
	}
	return core.Image{}, fmt.Errorf("custom image not found: %s", name)
}

// waitForImageState polls until the image reaches one of the target lifecycle states.
func (h *OracleMyImageHandler) waitForImageState(imageID string, targets []core.ImageLifecycleStateEnum) (core.Image, error) {
	const maxWait = 30 * time.Minute
	const poll = 15 * time.Second
	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		resp, err := h.Client.GetImage(h.Ctx, core.GetImageRequest{ImageId: common.String(imageID)})
		if err != nil {
			return core.Image{}, fmt.Errorf("failed to poll image state: %w", err)
		}
		for _, t := range targets {
			if resp.Image.LifecycleState == t {
				return resp.Image, nil
			}
		}
		time.Sleep(poll)
	}
	return core.Image{}, fmt.Errorf("timeout waiting for image %s to reach target state", imageID)
}

// ---------------------------------------------------------------------------
// MyImageHandler interface implementation
// ---------------------------------------------------------------------------

// SnapshotVM creates a custom image from a running or stopped VM instance.
func (h *OracleMyImageHandler) SnapshotVM(snapshotReqInfo irs.MyImageInfo) (irs.MyImageInfo, error) {
	if snapshotReqInfo.IId.NameId == "" {
		return irs.MyImageInfo{}, fmt.Errorf("MyImage NameId is required")
	}
	if snapshotReqInfo.SourceVM.SystemId == "" {
		return irs.MyImageInfo{}, fmt.Errorf("SourceVM SystemId is required")
	}

	tags := map[string]string{
		myImageSourceVMTag: snapshotReqInfo.SourceVM.SystemId,
	}
	for _, kv := range snapshotReqInfo.TagList {
		tags[kv.Key] = kv.Value
	}

	resp, err := h.Client.CreateImage(h.Ctx, core.CreateImageRequest{
		CreateImageDetails: core.CreateImageDetails{
			CompartmentId: common.String(h.CompartmentID),
			DisplayName:   common.String(snapshotReqInfo.IId.NameId),
			InstanceId:    common.String(snapshotReqInfo.SourceVM.SystemId),
			FreeformTags:  tags,
		},
	})
	if err != nil {
		return irs.MyImageInfo{}, fmt.Errorf("failed to create custom image: %w", err)
	}

	image, err := h.waitForImageState(stringValue(resp.Image.Id), []core.ImageLifecycleStateEnum{
		core.ImageLifecycleStateAvailable,
		core.ImageLifecycleStateDisabled,
	})
	if err != nil {
		return irs.MyImageInfo{}, err
	}
	if image.LifecycleState != core.ImageLifecycleStateAvailable {
		return irs.MyImageInfo{}, fmt.Errorf("image creation failed: image is in %s state", image.LifecycleState)
	}
	return h.imageToMyImageInfo(image), nil
}

// ListIID returns IIDs of all custom images in the compartment.
func (h *OracleMyImageHandler) ListIID() ([]*irs.IID, error) {
	images, err := h.listCustomImages()
	if err != nil {
		return nil, err
	}
	result := make([]*irs.IID, 0, len(images))
	for i := range images {
		iid := irs.IID{
			NameId:   stringValue(images[i].DisplayName),
			SystemId: stringValue(images[i].Id),
		}
		result = append(result, &iid)
	}
	return result, nil
}

// ListMyImage returns info for all custom images in the compartment.
func (h *OracleMyImageHandler) ListMyImage() ([]*irs.MyImageInfo, error) {
	images, err := h.listCustomImages()
	if err != nil {
		return nil, err
	}
	result := make([]*irs.MyImageInfo, 0, len(images))
	for i := range images {
		info := h.imageToMyImageInfo(images[i])
		result = append(result, &info)
	}
	return result, nil
}

// GetMyImage returns info for a specific custom image identified by IID.
func (h *OracleMyImageHandler) GetMyImage(myImageIID irs.IID) (irs.MyImageInfo, error) {
	image, err := h.findCustomImageByIID(myImageIID)
	if err != nil {
		return irs.MyImageInfo{}, err
	}
	return h.imageToMyImageInfo(image), nil
}

// CheckWindowsImage returns true if the custom image is Windows-based.
func (h *OracleMyImageHandler) CheckWindowsImage(myImageIID irs.IID) (bool, error) {
	image, err := h.findCustomImageByIID(myImageIID)
	if err != nil {
		return false, err
	}
	return strings.Contains(strings.ToLower(stringValue(image.OperatingSystem)), "windows"), nil
}

// DeleteMyImage deletes a custom image by IID.
func (h *OracleMyImageHandler) DeleteMyImage(myImageIID irs.IID) (bool, error) {
	image, err := h.findCustomImageByIID(myImageIID)
	if err != nil {
		return false, err
	}
	_, err = h.Client.DeleteImage(h.Ctx, core.DeleteImageRequest{
		ImageId: image.Id,
	})
	if err != nil {
		return false, fmt.Errorf("failed to delete custom image: %w", err)
	}
	return true, nil
}
