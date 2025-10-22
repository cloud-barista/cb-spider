// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// KT Cloud VM Spec Handler
//
// by ETRI, 2021.05.
// Updated by ETRI, 2023.11.
// Updated by ETRI, 2025.02.

package resources

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	// "time"
	// "sync"
	// "github.com/davecgh/go-spew/spew"

	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	ktsdk "github.com/cloud-barista/ktcloud-sdk-go"
)

type KtCloudVMSpecHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	Client         *ktsdk.KtCloudClient
}

var globalImageMap = make(map[string]*irs.ImageInfo)

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("KT Cloud VMSpec Handler")
}

// The response of ListAvailableProductTypes(zoneId)
// 'TemplateId' in KT Cloud : supporting OS info ID
// 'ServiceOfferingId' in KT Cloud : CPU/Memory info ID

func (vmSpecHandler *KtCloudVMSpecHandler) GetVMSpec(specName string) (irs.VMSpecInfo, error) {
	cblogger.Info("KT Classic driver: called GetVMSpec()!")
	// Caution!! : KT Cloud doesn't support 'Region' officially, so we use 'Zone info.' which is from the connection info.

	if strings.EqualFold(specName, "") {
		newErr := fmt.Errorf("Invalid specName!!")
		cblogger.Error(newErr.Error())
		return irs.VMSpecInfo{}, newErr
	}

	// Note!!) Use ListVMSpec() to include 'CorrespondingImageIds' parameter.
	specListResult, err := vmSpecHandler.ListVMSpec()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VMSpec info list!! : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.VMSpecInfo{}, newErr
	}

	for _, spec := range specListResult {
		if strings.EqualFold(spec.Name, specName) {
			return *spec, nil
		}
	}
	return irs.VMSpecInfo{}, errors.New("Failed to find the VMSpec info : '" + specName)
}

func (vmSpecHandler *KtCloudVMSpecHandler) ListVMSpec() ([]*irs.VMSpecInfo, error) {
	cblogger.Info("KT Classic driver: called ListVMSpec()!")

	imageHandler := KtCloudImageHandler{
		Client:     vmSpecHandler.Client,
		RegionInfo: vmSpecHandler.RegionInfo, //CAUTION!! : Must input this!!
	}

	imageInfoList, err := imageHandler.ListImage()
	if err != nil {
		cblogger.Infof("Failed to Get Image list!! : ", err)
		return nil, errors.New("Failed to Get Image list!!")
	}

	// Populate the Global Map : globalImageMap
	// cblogger.Infof("\n\n### Image list count in globalImageMap :  [%d]\n", len(globalImageMap))
	if len(globalImageMap) < 1 {
		for _, imageInfo := range imageInfoList {
			globalImageMap[imageInfo.Name] = imageInfo
		}
	}
	// cblogger.Infof("\n\n### Image list count in globalImageMap :  [%d]\n", len(globalImageMap))

	productList, err := vmSpecHandler.Client.ListAvailableProductTypes(vmSpecHandler.RegionInfo.Zone)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Product Type list from KT Cloud : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	// cblogger.Infof("### Spec list Count : [%d]", len(productList.Listavailableproducttypesresponse.ProductTypes))
	// spew.Dump(productList)

	if len(productList.Listavailableproducttypesresponse.ProductTypes) < 1 {
		return nil, errors.New("Failed to Find Any Product types!!")
	}

	vmSpecMap := make(map[string]*irs.VMSpecInfo)
	for _, image := range imageInfoList {
		cblogger.Infof("# Lookup by KT Cloud Image ID(Product type -> Template) : [%s]", image.IId.SystemId)

		for _, productType := range productList.Listavailableproducttypesresponse.ProductTypes {
			if strings.EqualFold(image.IId.SystemId, productType.TemplateId) {
				vmSpecInfo, err := vmSpecHandler.mappingVMSpecInfo(&productType)
				if err != nil {
					newErr := fmt.Errorf("Failed to Map the VMSpec info : [%v]", err)
					cblogger.Error(newErr.Error())
					return nil, newErr
				}

				if existingSpec, exists := vmSpecMap[vmSpecInfo.Name]; exists {
					// If the VMSpec already exists, add the image ID to the corresponding images list in KeyValueList
					found := false
					for i, kv := range existingSpec.KeyValueList {
						if kv.Key == "CorrespondingImageIds" {
							imageIds := strings.Split(kv.Value, ",")
							for _, id := range imageIds {
								if id == image.IId.SystemId {
									found = true
									break
								}
							}
							if !found {
								existingSpec.KeyValueList[i].Value += "," + image.IId.SystemId
							}
							break
						}
					}
					// if !found {
					// 	existingSpec.KeyValueList = append(existingSpec.KeyValueList, irs.KeyValue{
					// 		Key:   "CorrespondingImageIds",
					// 		Value: image.IId.SystemId,
					// 	})
					// }
				} else {
					// If the VMSpec is new, add it to the map and initialize the corresponding images list in KeyValueList
					vmSpecInfo.KeyValueList = append(vmSpecInfo.KeyValueList, irs.KeyValue{
						Key:   "CorrespondingImageIds",
						Value: image.IId.SystemId,
					})
					vmSpecMap[vmSpecInfo.Name] = &vmSpecInfo
				}
				// time.Sleep(30 * time.Millisecond)
				// To prvent error : "Unable to execute API command listAvailableProductTypes  due to ratelimit timeout"
			}
		}
	}

	// Reinitialize the Global Map to Clear all data : globalImageMap
	globalImageMap = make(map[string]*irs.ImageInfo)
	// cblogger.Infof("\n\n### Image list count in globalImageMap :  [%d]\n", len(globalImageMap))

	var vmSpecInfoList []*irs.VMSpecInfo
	for _, specInfo := range vmSpecMap {
		vmSpecInfoList = append(vmSpecInfoList, specInfo)
	}
	// cblogger.Info("# Supported VM Spec count : ", len(vmSpecInfoList))
	return vmSpecInfoList, nil
}

// Extract instance Specification information
func (vmSpecHandler *KtCloudVMSpecHandler) mappingVMSpecInfo(productType *ktsdk.ProductTypes) (irs.VMSpecInfo, error) {
	cblogger.Info("KT Classic driver: called mappingVMSpecInfo()!")
	// spew.Dump(vmSpec)

	// Caution: If you use # instead of ! as the string split symbol below, the entire string will not be delivered through the CB-Spider API, only up to the #.
	ktVMSpecId := productType.ServiceOfferingId + "!" + productType.DiskOfferingId + "_disk" + productType.DiskOfferingDesc
	ktVMSpecString := productType.ServiceOfferingDesc
	// ex) productType.Serviceofferingdesc => "XS71 12vCore 16GB" //Caution!!
	// vCpuCount := strtmp[5:7]
	// productMem := strtmp[13:15]

	// Split the string based on " " (space) using the Split function
	specSlice := strings.Split(ktVMSpecString, " ")
	// for _, str := range specSlice {
	// 	cblogger.Infof("Splitted string : [%s]", str)
	// }

	// KT Cloud provides the number of cores in the form of '4vcore' or '16vCore'. (Be careful with string processing, number of digits, and case sensitivity)
	// vCpuCount := productVCpu[0:2] // 24vCore is fine, but 1vCore has a different total number of digits, so it's not appropriate

	productVCpu := strings.Replace(specSlice[1], "C", "c", 1) // If there is an uppercase C, change it to lowercase c ex) 1vCore -> 1vcore
	productVCpu = strings.TrimSuffix(productVCpu, "vcore")    // Remove 'vcore' from the right side of the string
	productMem := strings.TrimRight(specSlice[2], "GB")       // Remove G and B from the right side of the string
	//productMem := strings.TrimSuffix(specSlice[2], "GB")

	MemCountGb, err := strconv.Atoi(productMem) // Convert string to number
	if err != nil {
		cblogger.Error(err)
	}
	MemCountMbStr := strconv.Itoa(MemCountGb * 1024) // Convert number to string

	// In case: productType.DiskOfferingDesc : 100G (After Disk Add)
	// if len(productType.DiskOfferingId) > 1 {
	// 	ktVMSpecID = productType.ServiceOfferingId + "#" + productType.DiskOfferingId + "_disk" + productType.DiskOfferingDesc
	// }

	vmSpecInfo := irs.VMSpecInfo{
		Region:     productType.ZoneDesc,
		Name:       ktVMSpecId,
		VCpu:       irs.VCpuInfo{Count: productVCpu, ClockGHz: "-1"},
		MemSizeMiB: MemCountMbStr,
		DiskSizeGB: strings.TrimSuffix(productType.DiskOfferingDesc, "GB"),
		// No GPU, No Info Gpu:    []irs.GpuInfo{{Count: "-1", Mfr: "NA", Model: "NA", Mem: "-1"}},

		// Since KT Cloud supports different specs for each zone, the zone information is also provided.
		KeyValueList: irs.StructToKeyValueList(productType),
	}
	return vmSpecInfo, nil
}

func (vmSpecHandler *KtCloudVMSpecHandler) ListOrgVMSpec() (string, error) {
	cblogger.Info("KT Classic driver: called ListOrgVMSpec()!")

	regionInfo := vmSpecHandler.RegionInfo.Region
	cblogger.Infof("Region : [%s]", regionInfo)

	var vmSpecInfoList []*irs.VMSpecInfo
	vmSpecInfoList, err := vmSpecHandler.ListVMSpec()
	if err != nil {
		cblogger.Error(err)
		return "Error : ", err
	}
	jsonString, errJson := ConvertJsonString(vmSpecInfoList)
	if errJson != nil {
		cblogger.Error(errJson)
	}
	return jsonString, errJson
}

func (vmSpecHandler *KtCloudVMSpecHandler) GetOrgVMSpec(Name string) (string, error) {
	cblogger.Info("KT Classic driver: called GetOrgVMSpec()!")

	regionInfo := vmSpecHandler.RegionInfo.Region
	cblogger.Infof("Region : [%s] / SpecName : [%s]", regionInfo, Name)

	specInfo, err := vmSpecHandler.GetVMSpec(Name)
	if err != nil {
		cblogger.Error(err)
		return "Error : ", err
	}
	jsonString, errJson := ConvertJsonString(specInfo)
	if errJson != nil {
		cblogger.Error(errJson)
	}
	return jsonString, errJson
}
