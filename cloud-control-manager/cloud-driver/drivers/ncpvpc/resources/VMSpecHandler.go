// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// NCP VPC VMSpec Handler
//
// by ETRI, 2020.12.

package resources

import (
	"errors"
	"strconv"
	// "github.com/davecgh/go-spew/spew"

	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vserver"
	cblog "github.com/cloud-barista/cb-log"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NcpVpcVMSpecHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	VMClient       *vserver.APIClient
}

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("NCP VPC VMSpecHandler")
}

func (vmSpecHandler *NcpVpcVMSpecHandler) ListVMSpec() ([]*irs.VMSpecInfo, error) {
	cblogger.Info("NCP VPC Cloud driver: called ListVMSpec()!")
	InitLog()
	callLogInfo := GetCallLogScheme(vmSpecHandler.RegionInfo.Zone, call.VMSPEC, "ListVMSpec()", "ListVMSpec()")

	imageHandler := NcpVpcImageHandler{
		RegionInfo:     vmSpecHandler.RegionInfo,  //CAUTION!!
		VMClient:       vmSpecHandler.VMClient,
	}
	imgListResult, err := imageHandler.ListImage()
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get Image Info list :  : ", err)
		return nil, rtnErr
	} else {
		cblogger.Infof("Image list Count of the Region : [%d]", len(imgListResult))
	}

	var vmSpecInfoList []*irs.VMSpecInfo
	for _, image := range imgListResult {
		cblogger.Infof("\n### 기준 NCP VPC Image ID(ImageProductCode) : [%s]", image.IId.SystemId)
		vmSpecReq := vserver.GetServerProductListRequest{
			RegionCode:  			&vmSpecHandler.RegionInfo.Region,
			ServerImageProductCode: ncloud.String(image.IId.SystemId),  // ***** Caution : ImageProductCode is mandatory. *****
		}
		callLogStart := call.Start()
		result, err := vmSpecHandler.VMClient.V2Api.GetServerProductList(&vmSpecReq)
		if err != nil { 
			rtnErr := logAndReturnError(callLogInfo, "Failed to Get VMSpec list from NCP VPC Cloud : ", err)
			return nil, rtnErr
		}
		LoggingInfo(callLogInfo, callLogStart)

		// spew.Dump(result)
		if len(result.ProductList) < 1 {
			rtnErr := logAndReturnError(callLogInfo, "# VMSpec info corresponding to the Image ID does Not Exist!!", "")
			return nil, rtnErr
		} else {
			for _, NcpVMSpec := range result.ProductList {
				vmSpecInfo := vmSpecHandler.MappingVMSpecInfo(image.IId.SystemId, NcpVMSpec)
				vmSpecInfoList = append(vmSpecInfoList, &vmSpecInfo)
			}
		}
	}
	return vmSpecInfoList, nil
}

func (vmSpecHandler *NcpVpcVMSpecHandler) GetVMSpec(specName string) (irs.VMSpecInfo, error) {
	cblogger.Info("NCP VPC Cloud driver: called GetVMSpec()!")
	InitLog()
	callLogInfo := GetCallLogScheme(vmSpecHandler.RegionInfo.Zone, call.VMSPEC, specName, "GetVMSpec()")

	imageId, ncpVpcVMspec, err := vmSpecHandler.getNcpVpcVMSpec(specName, "GetVMSpec()")
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get VMSpec from NCP VPC Cloud : ", err)
		return irs.VMSpecInfo{}, rtnErr
	}
	specInfo := vmSpecHandler.MappingVMSpecInfo(imageId, ncpVpcVMspec)
	return specInfo, nil
}

func (vmSpecHandler *NcpVpcVMSpecHandler) ListOrgVMSpec() (string, error) {
	cblogger.Info("NCP VPC Cloud driver: called ListOrgVMSpec()!")
	InitLog()
	callLogInfo := GetCallLogScheme(vmSpecHandler.RegionInfo.Zone, call.VMSPEC, "ListOrgVMSpec()", "ListOrgVMSpec()")

	ncpVpcVMSpecList, err := vmSpecHandler.getNcpVpcVMSpecList("ListOrgVMSpec()")
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get VMSpec from NCP VPC Cloud : ", err)
		return "", rtnErr
	}
	jsonString, cvtErr := ConvertJsonString(ncpVpcVMSpecList)
	if cvtErr != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Convert JSON to String : ", cvtErr)
		return "", rtnErr
	}
	return jsonString, nil
}

func (vmSpecHandler *NcpVpcVMSpecHandler) GetOrgVMSpec(specName string) (string, error) {
	cblogger.Info("NCP VPC Cloud driver: called GetOrgVMSpec()!")
	InitLog()
	callLogInfo := GetCallLogScheme(vmSpecHandler.RegionInfo.Zone, call.VMSPEC, specName, "GetOrgVMSpec()")

	_, ncpVpcVMSpec, err := vmSpecHandler.getNcpVpcVMSpec(specName, "GetOrgVMSpec()")
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get VMSpec from NCP VPC Cloud : ", err)
		return "", rtnErr
	}
	jsonString, cvtErr := ConvertJsonString(ncpVpcVMSpec)
	if cvtErr != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Convert JSON to String : ", cvtErr)
		return "", rtnErr
	}
	return jsonString, nil
}

func (vmSpecHandler *NcpVpcVMSpecHandler) MappingVMSpecInfo(ImageId string, NcpVMSpec *vserver.Product) irs.VMSpecInfo {
	cblogger.Info("NCP VPC Cloud driver: called MappingVMSpecInfo()!")
	// spew.Dump(NcpVMSpec)

	vmSpecInfo := irs.VMSpecInfo{
		Region: vmSpecHandler.RegionInfo.Region,
		Name:   *NcpVMSpec.ProductCode,
		// int32 to string 변환 : String(), int64 to string 변환 : strconv.Itoa()
		VCpu: irs.VCpuInfo{Count: String(*NcpVMSpec.CpuCount), Clock: "N/A"},

		// vserver.Product에 GPU 정보는 없음.
		Gpu: []irs.GpuInfo{{Count: "N/A", Mfr: "N/A", Model: "N/A", Mem: "N/A"}},

		KeyValueList: []irs.KeyValue{
			// {Key: "ProductName", Value: *vmSpec.ProductName}, // This is same to 'ProductDescription'.
			{Key: "ProductType", Value: *NcpVMSpec.ProductType.CodeName},
			{Key: "InfraResourceType", Value: *NcpVMSpec.InfraResourceType.CodeName},
			// {Key: "PlatformType", Value: *NcpVMSpec.PlatformType.Code}, // ### This makes "invalid memory address or nil pointer dereference" error
			{Key: "BaseBlockStorageSize(GB)", Value: strconv.FormatFloat(float64(*NcpVMSpec.BaseBlockStorageSize)/(1024*1024*1024), 'f', 0, 64)},
			{Key: "DiskType", Value: *NcpVMSpec.DiskType.CodeName},
			{Key: "ProductDescription", Value: *NcpVMSpec.ProductDescription},
			{Key: "SupportingImageSystemId", Value: ImageId},
			{Key: "Region", Value: vmSpecHandler.RegionInfo.Region},
		},
	}
	// vmSpecInfo.Mem = strconv.FormatFloat(float64(*vmSpec.MemorySize)*1024, 'f', 0, 64) // GB->MB로 변환
	vmSpecInfo.Mem = strconv.FormatFloat(float64(*NcpVMSpec.MemorySize)/(1024*1024), 'f', 0, 64)
	return vmSpecInfo
}

func (vmSpecHandler *NcpVpcVMSpecHandler) getNcpVpcVMSpecList(callLogfunc string) ([]*vserver.Product, error) {
	cblogger.Info("NCP VPC Cloud driver: called getNcpVpcVMSpecList()!")
	InitLog()
	callLogInfo := GetCallLogScheme(vmSpecHandler.RegionInfo.Zone, call.VMSPEC, callLogfunc, callLogfunc)

	imgHandler := NcpVpcImageHandler{
		RegionInfo:     vmSpecHandler.RegionInfo,
		VMClient: 		vmSpecHandler.VMClient,
	}
	imgListResult, err := imgHandler.ListImage()
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get Image Info list :  : ", err)
		return nil, rtnErr
	} else {
		cblogger.Infof("Image list Count of the Region : [%d]", len(imgListResult))
	}

	for _, image := range imgListResult {
		cblogger.Infof("\n### 기준 NCP VPC Image ID(ImageProductCode) : [%s]", image.IId.SystemId)
		specReq := vserver.GetServerProductListRequest{
			RegionCode:  			&vmSpecHandler.RegionInfo.Region,
			ServerImageProductCode: ncloud.String(image.IId.SystemId), 	// *** Caution : ImageProductCode is mandatory. ***
		}		
		callLogStart := call.Start()
		result, err := vmSpecHandler.VMClient.V2Api.GetServerProductList(&specReq)
		if err != nil {
			rtnErr := logAndReturnError(callLogInfo, "Failed to Get VMSpec list from NCP VPC : ", err)
			return nil, rtnErr
		}
		LoggingInfo(callLogInfo, callLogStart)

		// spew.Dump(result)
		if len(result.ProductList) < 1 {
			rtnErr := logAndReturnError(callLogInfo, "# VMSpec info corresponding to the Image ID does Not Exist!!", "")
			return nil, rtnErr
		} else {
			return result.ProductList, nil
		}
	}
	return nil, errors.New("Failed to Get NCP VPC VMSpec List!!")
}

func (vmSpecHandler *NcpVpcVMSpecHandler) getNcpVpcVMSpec(specName string, callLogfunc string) (string, *vserver.Product, error) {
	cblogger.Info("NCP VPC Cloud driver: called GetNcpVpcVMSpec()!")
	InitLog()
	callLogInfo := GetCallLogScheme(vmSpecHandler.RegionInfo.Zone, call.VMSPEC, specName, callLogfunc)

	imgHandler := NcpVpcImageHandler{
		RegionInfo:     vmSpecHandler.RegionInfo,
		VMClient: 		vmSpecHandler.VMClient,
	}
	imgListResult, err := imgHandler.ListImage()
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get Image Info list : ", err)
		return "", nil, rtnErr
	} else {
		cblogger.Infof("Image list count of the region : [%d]", len(imgListResult))
	}

	for _, image := range imgListResult {
		cblogger.Infof("\n### 기준 NCP VPC Image ID(ImageProductCode) : [%s]", image.IId.SystemId)
		specReq := vserver.GetServerProductListRequest{
			RegionCode:  			&vmSpecHandler.RegionInfo.Region,
			ProductCode: 			&specName,
			ServerImageProductCode: ncloud.String(image.IId.SystemId), 	// *** Caution : ImageProductCode is mandatory. ***
		}
		
		callLogStart := call.Start()
		result, err := vmSpecHandler.VMClient.V2Api.GetServerProductList(&specReq)
		if err != nil {
			if err != nil {
				rtnErr := logAndReturnError(callLogInfo, "Failed to Get VMSpec list from NCP VPC : ", err)
				return "", nil, rtnErr
			}
		}
		LoggingInfo(callLogInfo, callLogStart)

		// spew.Dump(result)
		if len(result.ProductList) < 1 {
			rtnErr := logAndReturnError(callLogInfo, "# VMSpec info corresponding to the VMSpec Name and Image ID does Not Exist!!", "")
			return "", nil, rtnErr
		} else {
			return image.IId.SystemId, result.ProductList[0], nil
		}
	}
	return "", nil, errors.New("Failed to Get the NCP VPC VMSpec!!")
}
