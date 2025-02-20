// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// NCP VM Spec Handler
//
// by ETRI, 2020.09.

package resources

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	// "github.com/davecgh/go-spew/spew"

	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/server"
	cblog "github.com/cloud-barista/cb-log"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NcpVMSpecHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	VMClient       *server.APIClient
}

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("NCP VMSpecHandler")
}

func (vmSpecHandler *NcpVMSpecHandler) ListVMSpec() ([]*irs.VMSpecInfo, error) {
	cblogger.Info("NCP Classic Cloud Driver: called ListVMSpec()!")
	InitLog()
	callLogInfo := GetCallLogScheme(vmSpecHandler.RegionInfo.Zone, call.VMIMAGE, "ListVMSpec()", "ListVMSpec()")

	ncpRegion := vmSpecHandler.RegionInfo.Region
	cblogger.Infof("Region : [%s]", ncpRegion)

	vmHandler := NcpVMHandler{
		RegionInfo: vmSpecHandler.RegionInfo,
		VMClient:   vmSpecHandler.VMClient,
	}
	regionNo, err := vmHandler.getRegionNo(vmSpecHandler.RegionInfo.Region)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the NCP Region No of the Region Code: [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	zoneNo, err := vmHandler.getZoneNo(vmSpecHandler.RegionInfo.Region, vmSpecHandler.RegionInfo.Zone)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NCP Zone No of the Zone Code : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	imageHandler := NcpImageHandler{
		CredentialInfo: vmSpecHandler.CredentialInfo,
		RegionInfo:     vmSpecHandler.RegionInfo, //CAUTION!!
		VMClient:       vmSpecHandler.VMClient,
	}
	imageListResult, err := imageHandler.ListImage()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the NCP Image list!! : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	// cblogger.Info("Image list Count : ", len(imageListResult))
	// spew.Dump(imageListResult)

	// Note : var vmProductList []*server.Product  //NCP Product(Spec) Info.
	var vmSpecInfoMap = make(map[string]*irs.VMSpecInfo) // Map to track unique VMSpec Info.
	var vmSpecInfoList []*irs.VMSpecInfo                 // List to return unique VMSpec Info.

	for _, image := range imageListResult {
		cblogger.Infof("# Lookup by NCP Image ID(ImageProductCode) : [%s]", image.IId.SystemId)

		vmSpecReq := server.GetServerProductListRequest{
			RegionNo:               regionNo,
			ZoneNo:                 zoneNo,
			ServerImageProductCode: ncloud.String(image.IId.SystemId), // ***** Caution : ImageProductCode is mandatory. *****
			// GenerationCode: 		ncloud.String("G2"),  				// # Caution!! : Generations are divided only in the Korean Region.
		}
		result, err := vmSpecHandler.VMClient.V2Api.GetServerProductList(&vmSpecReq)
		if err != nil {
			cblogger.Error(*result.ReturnMessage)
			cblogger.Error(fmt.Sprintf("Failed to Get VMSpec list from NCP : [%v]", err))
			return nil, err
		} else {
			cblogger.Infof("Lookup by NCP Image ID(ImageProductCode) : [%s]", image.IId.SystemId)
			cblogger.Infof("Number of VMSpec info looked up : [%d]", len(result.ProductList))
		}

		for _, product := range result.ProductList {
			vmSpecInfo := mappingVMSpecInfo(ncpRegion, *product)
			if existingSpec, exists := vmSpecInfoMap[vmSpecInfo.Name]; exists {
				// If the VMSpec already exists, add the image ID to the corresponding images list in KeyValueList
				for i, kv := range existingSpec.KeyValueList {
					if kv.Key == "CorrespondingImageIds" {
						existingSpec.KeyValueList[i].Value += "," + image.IId.SystemId
						break
					}
				}
			} else {
				// If the VMSpec is new, add it to the map and initialize the corresponding images list in KeyValueList
				vmSpecInfo.KeyValueList = append(vmSpecInfo.KeyValueList, irs.KeyValue{
					Key:   "CorrespondingImageIds",
					Value: image.IId.SystemId,
				})
				vmSpecInfoMap[vmSpecInfo.Name] = &vmSpecInfo
			}
		}
	}

	// Convert the map to a list
	for _, specInfo := range vmSpecInfoMap {
		vmSpecInfoList = append(vmSpecInfoList, specInfo)
	}
	// cblogger.Infof("# Total count of the VMSpec Info : [%d]", len(vmSpecInfoList))
	return vmSpecInfoList, err
}

func (vmSpecHandler *NcpVMSpecHandler) GetVMSpec(specName string) (irs.VMSpecInfo, error) {
	cblogger.Info("NCP Classic Cloud Driver: called GetVMSpec()!")
	InitLog()
	callLogInfo := GetCallLogScheme(vmSpecHandler.RegionInfo.Zone, call.VMSPEC, specName, "GetVMSpec()")

	if strings.EqualFold(specName, "") {
		newErr := fmt.Errorf("Invalid specName!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMSpecInfo{}, newErr
	}

	// Note!!) Use ListVMSpec() to include 'CorrespondingImageIds' parameter.
	specListResult, err := vmSpecHandler.ListVMSpec()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VMSpec info list!! : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMSpecInfo{}, newErr
	}

	for _, spec := range specListResult {
		if strings.EqualFold(spec.Name, specName) {
			return *spec, nil
		}
	}
	return irs.VMSpecInfo{}, errors.New("Failed to find the VMSpec info : '" + specName)
}

func (vmSpecHandler *NcpVMSpecHandler) ListOrgVMSpec() (string, error) {
	cblogger.Info("NCP Classic Cloud Driver: called ListOrgVMSpec()!")
	InitLog()
	callLogInfo := GetCallLogScheme(vmSpecHandler.RegionInfo.Zone, call.VMIMAGE, "ListOrgVMSpec()", "ListOrgVMSpec()")

	ncpRegion := vmSpecHandler.RegionInfo.Region
	cblogger.Infof("Region : [%s]", ncpRegion)

	vmHandler := NcpVMHandler{
		RegionInfo: vmSpecHandler.RegionInfo,
		VMClient:   vmSpecHandler.VMClient,
	}
	regionNo, err := vmHandler.getRegionNo(vmSpecHandler.RegionInfo.Region)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the NCP Region No of the Region Code: [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}
	zoneNo, err := vmHandler.getZoneNo(vmSpecHandler.RegionInfo.Region, vmSpecHandler.RegionInfo.Zone)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NCP Zone No of the Zone Code : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}

	imageHandler := NcpImageHandler{
		CredentialInfo: vmSpecHandler.CredentialInfo,
		RegionInfo:     vmSpecHandler.RegionInfo, //CAUTION!!
		VMClient:       vmSpecHandler.VMClient,
	}
	imageListResult, err := imageHandler.ListImage()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the NCP Image list!! : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	} else {
		cblogger.Info("Image list Count : ", len(imageListResult))
		// spew.Dump(imageListResult)
	}

	var vmSpecInfoMap = make(map[string]*irs.VMSpecInfo) // Map to track unique VMSpec Info.
	for _, image := range imageListResult {
		cblogger.Infof("# Lookup by NCP Image ID(ImageProductCode) : [%s]", image.IId.SystemId)

		vmSpecReq := server.GetServerProductListRequest{
			RegionNo:               regionNo,
			ZoneNo:                 zoneNo,
			ServerImageProductCode: ncloud.String(image.IId.SystemId), // ***** Caution : ImageProductCode is mandatory. *****
			// GenerationCode: 		ncloud.String("G2"),  				// # Caution!! : Generations are divided only in the Korean Region.
		}
		result, err := vmSpecHandler.VMClient.V2Api.GetServerProductList(&vmSpecReq)
		if err != nil {
			cblogger.Error(*result.ReturnMessage)
			cblogger.Error(fmt.Sprintf("Failed to Get VMSpec list from NCP : [%v]", err))
			return "", err
		} else {
			cblogger.Infof("Lookup by NCP Image ID(ImageProductCode) : [%s]", image.IId.SystemId)
			cblogger.Infof("Number of VMSpec info looked up : [%d]", len(result.ProductList))
		}

		for _, product := range result.ProductList {
			vmSpecInfo := mappingVMSpecInfo(ncpRegion, *product)
			if existingSpec, exists := vmSpecInfoMap[vmSpecInfo.Name]; exists {
				// If the VMSpec already exists, add the image ID to the corresponding images list in KeyValueList
				for i, kv := range existingSpec.KeyValueList {
					if kv.Key == "CorrespondingImageIds" {
						existingSpec.KeyValueList[i].Value += "," + image.IId.SystemId
						break
					}
				}
			} else {
				// If the VMSpec is new, add it to the map and initialize the corresponding images list in KeyValueList
				vmSpecInfo.KeyValueList = append(vmSpecInfo.KeyValueList, irs.KeyValue{
					Key:   "CorrespondingImageIds",
					Value: image.IId.SystemId,
				})
				vmSpecInfoMap[vmSpecInfo.Name] = &vmSpecInfo
			}
		}
	}

	// Convert the map to a list
	var vmSpecInfoList []*irs.VMSpecInfo // List to return unique VMSpec Info.
	for _, specInfo := range vmSpecInfoMap {
		vmSpecInfoList = append(vmSpecInfoList, specInfo)
	}
	// cblogger.Infof("# VMSpec Count : [%d]", len(vmSpecInfoList))

	jsonString, jsonErr := ConvertJsonString(vmSpecInfoList)
	if jsonErr != nil {
		cblogger.Error(jsonErr)
		return "", jsonErr
	}
	return jsonString, jsonErr
}

func (vmSpecHandler *NcpVMSpecHandler) GetOrgVMSpec(specName string) (string, error) {
	cblogger.Info("NCP Classic Cloud Driver: called GetOrgVMSpec()!")
	InitLog()
	callLogInfo := GetCallLogScheme(vmSpecHandler.RegionInfo.Zone, call.VMSPEC, specName, "GetOrgVMSpec()")

	if strings.EqualFold(specName, "") {
		newErr := fmt.Errorf("Invalid specName!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}

	ncpRegion := vmSpecHandler.RegionInfo.Region
	cblogger.Infof("Region : [%s] / SpecName : [%s]", ncpRegion, specName)

	vmHandler := NcpVMHandler{
		RegionInfo: vmSpecHandler.RegionInfo,
		VMClient:   vmSpecHandler.VMClient,
	}
	regionNo, err := vmHandler.getRegionNo(vmSpecHandler.RegionInfo.Region)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the NCP Region No of the Region Code: [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}
	zoneNo, err := vmHandler.getZoneNo(vmSpecHandler.RegionInfo.Region, vmSpecHandler.RegionInfo.Zone)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NCP Zone No of the Zone Code : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}

	imgHandler := NcpImageHandler{
		CredentialInfo: vmSpecHandler.CredentialInfo,
		RegionInfo:     vmSpecHandler.RegionInfo,
		VMClient:       vmSpecHandler.VMClient,
	}
	cblogger.Infof("imgHandler.RegionInfo.Zone : [%s]", imgHandler.RegionInfo.Zone) //Need to Check the value!!

	imgListResult, err := imgHandler.ListImage()
	if err != nil {
		cblogger.Infof("Failed to Find Image list!! : ", err)
		return "", errors.New("Failed to Find Image list!!")
	} else {
		cblogger.Info("Succeeded in Getting Image list!!")
		// cblogger.Info(imgListResult)
		cblogger.Infof("Image list Count : [%d]", len(imgListResult))
		// spew.Dump(imgListResult)
	}

	for _, image := range imgListResult {
		cblogger.Infof("# Lookup by NCP Image ID(ImageProductCode) : [%s]", image.IId.SystemId)

		specReq := server.GetServerProductListRequest{
			RegionNo:               regionNo,
			ZoneNo:                 zoneNo,
			ProductCode:            &specName,
			ServerImageProductCode: ncloud.String(image.IId.SystemId), // ***** Caution : ImageProductCode is mandatory. *****
		}
		callLogStart := call.Start()
		result, err := vmSpecHandler.VMClient.V2Api.GetServerProductList(&specReq)
		if err != nil {
			cblogger.Error(*result.ReturnMessage)
			newErr := fmt.Errorf("Failed to Find VMSpec list from NCP : [%v]", err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return "", newErr
		}
		LoggingInfo(callLogInfo, callLogStart)
		// spew.Dump(result)

		if len(result.ProductList) > 0 {
			jsonString, jsonErr := ConvertJsonString(*result.ProductList[0])
			if jsonErr != nil {
				cblogger.Error(jsonErr)
				return "", jsonErr
			}
			return jsonString, jsonErr
		}
	}

	return "", nil
}

func mappingVMSpecInfo(region string, vmSpec server.Product) irs.VMSpecInfo {
	//	server ProductList type : []*Product
	cblogger.Infof("*** Mapping VMSpecInfo : Region: [%s] / SpecName: [%s]", region, *vmSpec.ProductCode)
	// spew.Dump(vmSpec)

	// Since there is no region information in vmSpec, use the region information provided to the function
	// NOTE: Caution: vmSpec.ProductCode -> specName
	vmSpecInfo := irs.VMSpecInfo{
		Region: region,
		// Name:   *vmSpec.ProductName,
		Name: *vmSpec.ProductCode,
		// int32 to string : String(), int64 to string  : strconv.Itoa()
		VCpu: irs.VCpuInfo{Count: String(*vmSpec.CpuCount), Clock: "-1"},

		// 'server.Product' does not contain GPU information.
		// No GPU, No Info Gpu: []irs.GpuInfo{{Count: "-1", Mfr: "NA", Model: "NA", Mem: "-1"}},
		Mem:  strconv.FormatFloat(float64(*vmSpec.MemorySize)/(1024*1024), 'f', 0, 64),
		Disk: strconv.FormatFloat(float64(*vmSpec.BaseBlockStorageSize)/(1024*1024*1024), 'f', 0, 64),

		KeyValueList: irs.StructToKeyValueList(vmSpec),
	}
	// Mem : strconv.FormatFloat(float64(*vmSpec.MemorySize)*1024, 'f', 0, 64) // GB -> MB
	return vmSpecInfo
}
