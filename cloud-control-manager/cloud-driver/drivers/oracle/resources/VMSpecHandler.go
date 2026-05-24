package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
)

type OracleVMSpecHandler struct {
	Region        idrv.RegionInfo
	CompartmentID string
	Client        core.ComputeClient
	Ctx           context.Context
}

func (handler *OracleVMSpecHandler) ListVMSpec() ([]*irs.VMSpecInfo, error) {
	shapes, err := handler.listShapes("")
	if err != nil {
		return nil, err
	}
	infos := make([]*irs.VMSpecInfo, 0, len(shapes))
	for _, shape := range shapes {
		info := handler.vmSpecInfo(shape)
		infos = append(infos, &info)
	}
	return infos, nil
}

func (handler *OracleVMSpecHandler) ListVMSpecByImage(imageIID irs.IID) ([]*irs.VMSpecInfo, error) {
	imageID, err := handler.resolveImageID(imageIID)
	if err != nil {
		return nil, err
	}
	shapes, err := handler.listShapesByImage("", imageID)
	if err != nil {
		return nil, err
	}
	infos := make([]*irs.VMSpecInfo, 0, len(shapes))
	for _, shape := range shapes {
		info := handler.vmSpecInfo(shape)
		infos = append(infos, &info)
	}
	return infos, nil
}

func (handler *OracleVMSpecHandler) GetVMSpec(name string) (irs.VMSpecInfo, error) {
	shapes, err := handler.listShapes(name)
	if err != nil {
		return irs.VMSpecInfo{}, err
	}
	if len(shapes) == 0 {
		return irs.VMSpecInfo{}, fmt.Errorf("Oracle VM spec not found: %s", name)
	}
	return handler.vmSpecInfo(shapes[0]), nil
}

func (handler *OracleVMSpecHandler) GetVMSpecByImage(name string, imageIID irs.IID) (irs.VMSpecInfo, error) {
	imageID, err := handler.resolveImageID(imageIID)
	if err != nil {
		return irs.VMSpecInfo{}, err
	}
	shapes, err := handler.listShapesByImage(name, imageID)
	if err != nil {
		return irs.VMSpecInfo{}, err
	}
	if len(shapes) == 0 {
		return irs.VMSpecInfo{}, fmt.Errorf("Oracle VM spec not found for image: %s", name)
	}
	return handler.vmSpecInfo(shapes[0]), nil
}

func (handler *OracleVMSpecHandler) ListOrgVMSpec() (string, error) {
	shapes, err := handler.listShapes("")
	if err != nil {
		return "", err
	}
	bytes, err := json.Marshal(shapes)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (handler *OracleVMSpecHandler) GetOrgVMSpec(name string) (string, error) {
	shapes, err := handler.listShapes(name)
	if err != nil {
		return "", err
	}
	if len(shapes) == 0 {
		return "", fmt.Errorf("Oracle VM spec not found: %s", name)
	}
	bytes, err := json.Marshal(shapes[0])
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (handler *OracleVMSpecHandler) listShapes(name string) ([]core.Shape, error) {
	req := core.ListShapesRequest{CompartmentId: common.String(handler.CompartmentID), AvailabilityDomain: common.String(handler.Region.Zone)}
	if name != "" {
		req.Shape = common.String(name)
	}
	return handler.listShapesWithRequest(req)
}

func (handler *OracleVMSpecHandler) listShapesByImage(name string, imageID string) ([]core.Shape, error) {
	req := core.ListShapesRequest{CompartmentId: common.String(handler.CompartmentID), AvailabilityDomain: common.String(handler.Region.Zone), ImageId: common.String(imageID)}
	if name != "" {
		req.Shape = common.String(name)
	}
	return handler.listShapesWithRequest(req)
}

func (handler *OracleVMSpecHandler) listShapesWithRequest(req core.ListShapesRequest) ([]core.Shape, error) {
	shapes := make([]core.Shape, 0)
	page := ""
	for {
		if page != "" {
			req.Page = common.String(page)
		}
		resp, err := handler.Client.ListShapes(handler.Ctx, req)
		if err != nil {
			return nil, statusErr("failed to list Oracle VM specs", err)
		}
		shapes = append(shapes, resp.Items...)
		if resp.OpcNextPage == nil || *resp.OpcNextPage == "" {
			break
		}
		page = *resp.OpcNextPage
	}
	return shapes, nil
}

func (handler *OracleVMSpecHandler) resolveImageID(iid irs.IID) (string, error) {
	id := iid.SystemId
	if id == "" {
		id = iid.NameId
	}
	if isOracleOCID(id) {
		return id, nil
	}
	imageHandler := OracleImageHandler{Region: handler.Region, CompartmentID: handler.CompartmentID, Client: handler.Client, Ctx: handler.Ctx}
	image, err := imageHandler.GetImage(iid)
	if err != nil {
		return "", err
	}
	if image.IId.SystemId == "" || !isOracleOCID(image.IId.SystemId) {
		return "", fmt.Errorf("Oracle image OCID not found: %s", id)
	}
	return image.IId.SystemId, nil
}

func (handler *OracleVMSpecHandler) vmSpecInfo(shape core.Shape) irs.VMSpecInfo {
	name := stringValue(shape.Shape)
	memMiB := "-1"
	if shape.MemoryInGBs != nil {
		memMiB = strconv.FormatInt(int64(*shape.MemoryInGBs*1024), 10)
	}
	vcpuCount := "-1"
	if shape.Ocpus != nil {
		vcpuCount = strconv.FormatFloat(float64(*shape.Ocpus), 'f', -1, 32)
	}
	gpus := make([]irs.GpuInfo, 0)
	if shape.Gpus != nil && *shape.Gpus > 0 {
		gpus = append(gpus, irs.GpuInfo{Count: strconv.Itoa(*shape.Gpus), Mfr: "NA", Model: stringValue(shape.GpuDescription), MemSizeGB: "-1", TotalMemSizeGB: "-1"})
	}
	diskSize := "-1"
	if shape.LocalDisksTotalSizeInGBs != nil {
		diskSize = strconv.FormatFloat(float64(*shape.LocalDisksTotalSizeInGBs), 'f', -1, 32)
	}
	return irs.VMSpecInfo{
		Region:     handler.Region.Region,
		Name:       name,
		VCpu:       irs.VCpuInfo{Count: vcpuCount, ClockGHz: "-1"},
		MemSizeMiB: memMiB,
		DiskSizeGB: diskSize,
		Gpu:        gpus,
		KeyValueList: []irs.KeyValue{
			{Key: "ProcessorDescription", Value: stringValue(shape.ProcessorDescription)},
			{Key: "IsFlexible", Value: boolString(shape.IsFlexible)},
			{Key: "IsSubcore", Value: boolString(shape.IsSubcore)},
			{Key: "NetworkingBandwidthInGbps", Value: float32String(shape.NetworkingBandwidthInGbps)},
			{Key: "MaxVnicAttachments", Value: intString(shape.MaxVnicAttachments)},
		},
	}
}

func float32String(value *float32) string {
	if value == nil {
		return ""
	}
	return strconv.FormatFloat(float64(*value), 'f', -1, 32)
}

func intString(value *int) string {
	if value == nil {
		return ""
	}
	return strconv.Itoa(*value)
}

func boolString(value *bool) string {
	if value == nil {
		return ""
	}
	return strconv.FormatBool(*value)
}
