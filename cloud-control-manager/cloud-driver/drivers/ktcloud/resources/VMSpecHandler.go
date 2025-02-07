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
	"strings"
	"strconv"
	"regexp"
	"fmt"
	"time"
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

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("KT Cloud VMSpec Handler")
}

// The response of ListAvailableProductTypes(zoneId)
// 'TemplateId' in KT Cloud : supporting OS info ID
// 'ServiceOfferingId' in KT Cloud : CPU/Memory info ID

func (vmSpecHandler *KtCloudVMSpecHandler) GetVMSpec(specName string) (irs.VMSpecInfo, error) {
	cblogger.Info("KT Cloud cloud driver: called GetVMSpec()!")
	// Caution!! : KT Cloud doesn't support 'Region' officially, so we use 'Zone info.' which is from the connection info.

	// Caution!! : When searching for Image info/VMSpc info, KT Cloud inquires using Zoneid.
	result, err := vmSpecHandler.Client.ListAvailableProductTypes(vmSpecHandler.RegionInfo.Zone)
	if err != nil {
		cblogger.Error("Failed to Get List of Available Product Types: %s", err)
		return irs.VMSpecInfo{}, err
	}

	if len(result.Listavailableproducttypesresponse.ProductTypes) < 1 {
		return irs.VMSpecInfo{}, errors.New("Failed to Find Product types!!")
	}

	// specName ex) d3530ad2-462b-43ad-97d5-e1087b952b7d!87c0a6f6-c684-4fbe-a393-d8412bcf788d_disk100GB
	// Caution) If you use # instead of ! among the string split symbols below, the entire string is not delivered when calling through the CB-Spider API, but only before #.
	instanceSpecString := strings.Split(specName, "!")
	for i := range instanceSpecString {
		cblogger.Info("instanceSpecString : ", instanceSpecString[i])
	}

	ktVMSpecId := instanceSpecString[0]
	// cblogger.Info("vmSpecID : ", ktVMSpecId)

    // Ex) 87c0a6f6-c684-4fbe-a393-d8412bcf788d_disk100GB
	tempOfferingString := instanceSpecString[1]
	// cblogger.Info("tempOfferingString : ", tempOfferingString)

	diskOfferingString := strings.Split(tempOfferingString, "_")

	ktDiskOfferingId := diskOfferingString[0]
	// cblogger.Info("ktDiskOfferingId : ", ktDiskOfferingId)

	var resultVMSpecInfo irs.VMSpecInfo
	for _, productType := range result.Listavailableproducttypesresponse.ProductTypes {
		cblogger.Info("# Search criteria of Serviceofferingid : ", ktVMSpecId)		
		// if serverProductType.ServiceOfferingId == ktVMSpecId {
		if productType.ServiceOfferingId == ktVMSpecId {
			if productType.DiskOfferingId == ktDiskOfferingId {
				resultVMSpecInfo, err = vmSpecHandler.mappingVMSpecInfo(&productType)
				if err != nil {
					newErr := fmt.Errorf("Failed to Map the VMSpec info : [%v]", err)
					cblogger.Error(newErr.Error())
					return irs.VMSpecInfo{}, newErr
				}
				break
			}
		}
	}
	return resultVMSpecInfo, nil
}

func (vmSpecHandler *KtCloudVMSpecHandler) ListVMSpec() ([]*irs.VMSpecInfo, error) {
	cblogger.Info("KT Cloud cloud driver: called ListVMSpec()!")

	imageHandler := KtCloudImageHandler{
		Client:         vmSpecHandler.Client,
		RegionInfo:     vmSpecHandler.RegionInfo,  //CAUTION!! : Must input this!!
	}
	
	imageListResult, err := imageHandler.ListImage()
	if err != nil {
		cblogger.Infof("Failed to Get Image list!! : ", err)
		return nil, errors.New("Failed to Get Image list!!")
	}

	specListResult, err := vmSpecHandler.Client.ListAvailableProductTypes(vmSpecHandler.RegionInfo.Zone)
	if err != nil {
		cblogger.Error("Failed to Get List of Available Product Types: %s", err)
		return []*irs.VMSpecInfo{}, errors.New("Failed to Get Product Type list!!")
	} else {
		cblogger.Info("Succeeded in Getting Product Type list!!")
	}
	cblogger.Infof("### Spec list Count : [%d]", len(specListResult.Listavailableproducttypesresponse.ProductTypes))
	// spew.Dump(specListResult)
	// spew.Dump(result)

	if len(specListResult.Listavailableproducttypesresponse.ProductTypes) < 1 {
		return []*irs.VMSpecInfo{}, errors.New("Failed to Find Product types!!")
	}

	vmSpecMap := make(map[string]*irs.VMSpecInfo)
	for _, image := range imageListResult {
		cblogger.Infof("# Lookup by KT Cloud Image ID(Product type -> Template) : [%s]", image.IId.SystemId)
	
		for _, productType := range specListResult.Listavailableproducttypesresponse.ProductTypes {
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
				time.Sleep(50 * time.Millisecond)
				// To prvent error : "Unable to execute API command listAvailableProductTypes  due to ratelimit timeout"
			}
		}
	}
	
	var vmSpecInfoList []*irs.VMSpecInfo
	for _, specInfo := range vmSpecMap {
		vmSpecInfoList = append(vmSpecInfoList, specInfo)
	}	
	// cblogger.Info("# Supported VM Spec count : ", len(vmSpecInfoList))
	return vmSpecInfoList, nil
}

//Extract instance Specification information
func (vmSpecHandler *KtCloudVMSpecHandler) mappingVMSpecInfo(productType *ktsdk.ProductTypes) (irs.VMSpecInfo, error) {
	// cblogger.Infof("\n*** Mapping VMSpecInfo : SpecName: [%s]", productType.ServiceOfferingId)
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
	productVCpu = strings.TrimSuffix(productVCpu, "vcore") // Remove 'vcore' from the right side of the string
	productMem := strings.TrimRight(specSlice[2], "GB") // Remove G and B from the right side of the string
	//productMem := strings.TrimSuffix(specSlice[2], "GB")

	MemCountGb, err := strconv.Atoi(productMem) // Convert string to number
	if err != nil {
		cblogger.Error(err)
	}
	MemCountMbStr := strconv.Itoa(MemCountGb * 1024) // Convert number to string

	// ### Note!!) If the diskofferingid value exists, additional data disks are created.(=> So to get the 'Correct RootDiskSize')
	diskSize, err := vmSpecHandler.getRootDiskSize(&productType.DiskOfferingDesc, &productType.TemplateId, &productType.DiskOfferingId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Root Disk Size of the VMSpec: [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.VMSpecInfo{}, newErr
	}
	// In case: productType.DiskOfferingDesc : 100G (After Disk Add)
	// if len(productType.DiskOfferingId) > 1 {
	// 	ktVMSpecID = productType.ServiceOfferingId + "#" + productType.DiskOfferingId + "_disk" + productType.DiskOfferingDesc
	// }

	vmSpecInfo := irs.VMSpecInfo{
		Region: productType.ZoneDesc,
		Name:   ktVMSpecId,
		VCpu: 	irs.VCpuInfo{Count: productVCpu, Clock: "-1"},
		Mem: 	MemCountMbStr,
		Disk: 	diskSize,
		Gpu: 	[]irs.GpuInfo{{Count: "-1", Mfr: "NA", Model: "NA", Mem: "-1"}},

		// Since KT Cloud supports different specs for each zone, the zone information is also provided.
		KeyValueList: []irs.KeyValue{
			{Key: "KtServiceOffering", Value: productType.ServiceOfferingDesc},	
			{Key: "TotalDiskSize(includeDataDisk)", Value: productType.DiskOfferingDesc},
			{Key: "ProductState", Value: productType.ProductState},
			{Key: "Zone", Value: productType.ZoneDesc},
		},
	}
	return vmSpecInfo, nil
}

func (vmSpecHandler *KtCloudVMSpecHandler) ListOrgVMSpec() (string, error) {
	cblogger.Info("KT Cloud cloud driver: called ListOrgVMSpec()!")

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
	cblogger.Info("KT Cloud cloud driver: called GetOrgVMSpec()!")

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

// ### Note!!) If the diskofferingid value exists, additional data disks are created.(=> So to get the 'Correct RootDiskSize')
func (vmSpecHandler *KtCloudVMSpecHandler) getRootDiskSize(diskOfferingDesc *string, templateId *string, diskOfferingId *string) (string, error) {
	cblogger.Info("KT Cloud cloud driver: called getRootDiskSize()!")

	if strings.EqualFold(*diskOfferingDesc, "") {
		newErr := fmt.Errorf("Invalid diskOfferingDesc value!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	if strings.EqualFold(*templateId, "") {
		newErr := fmt.Errorf("Invalid templateId value!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	// diskOfferingDesc Ex : "100GB"
	re := regexp.MustCompile(`(\d+)GB`)
	matches := re.FindStringSubmatch(*diskOfferingDesc) // Find the match
	
	var offeringDiskSizeStr string
	if len(matches) > 1 {
		offeringDiskSizeStr = matches[1] // Extract only the numeric part. Ex : "100"
	}
	if strings.EqualFold(offeringDiskSizeStr, "") {
		return "-1", nil
	}

	if !strings.EqualFold(*diskOfferingId, "") {
		imageHandler := KtCloudImageHandler{
			RegionInfo: vmSpecHandler.RegionInfo,
			Client:     vmSpecHandler.Client,
		}
		imageInfo, err := imageHandler.GetImage(irs.IID{SystemId: *templateId})
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the VM Image Info : [%v]", err)
			return "", newErr
		}
	
		offeringDiskSizeInt, err := strconv.Atoi(offeringDiskSizeStr)
		if err != nil {
			newErr := fmt.Errorf("Failed to Convert the string to number : [%v]", err)
			return "", newErr
		}
		osDiskSizeInGBInt, err := strconv.Atoi(imageInfo.OSDiskSizeInGB)
		if err != nil {
			newErr := fmt.Errorf("Failed to Convert the string to number : [%v]", err)
			return "", newErr
		}
		// cblogger.Infof("offeringDiskSizeInt : [%d] / osDiskSizeInGBInt : [%d]", offeringDiskSizeInt, osDiskSizeInGBInt)

		rootDiskSizeInt := offeringDiskSizeInt - osDiskSizeInGBInt
		return strconv.Itoa(rootDiskSizeInt), nil // rootDiskSizeStr Ex : "50"
	} else {		
		return offeringDiskSizeStr, nil
	}
}
