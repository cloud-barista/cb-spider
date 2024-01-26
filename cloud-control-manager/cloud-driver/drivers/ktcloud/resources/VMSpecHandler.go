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

package resources

import (
	ktsdk "github.com/cloud-barista/ktcloud-sdk-go"

	"errors"
	"fmt"
	"strings"
	"strconv"
	// "github.com/davecgh/go-spew/spew"

	cblog "github.com/cloud-barista/cb-log"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
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

func (vmSpecHandler *KtCloudVMSpecHandler) GetVMSpec(VMSpecName string) (irs.VMSpecInfo, error) {
	cblogger.Info("KT Cloud cloud driver: called GetVMSpec()!")
	// Caution!! : KT Cloud doesn't support 'Region' officially, so we use 'Zone info.' which is from the connection info.

	var resultVMSpecInfo irs.VMSpecInfo
	regionInfo := vmSpecHandler.RegionInfo.Region
	zoneId := vmSpecHandler.RegionInfo.Zone

	cblogger.Infof("Region : [%s] / SpecName : [%s]", regionInfo, VMSpecName)
	cblogger.Info("RegionInfo.Zone : ", zoneId)

	//var ktZoneInfo ktsdk.Zone

	//To get the Zone Name of the zoneId
	// var ktZoneName string
	// response, err := vmSpecHandler.Client.ListZones(true, "", "", "")
	// if err != nil {
	// 	cblogger.Error("Error to get list of available Zones: %s", err)

	// 	return irs.VMSpecInfo{}, err
	// }

	// for _, Zone := range response.Listzonesresponse.Zone {
	// 	cblogger.Info("# Search criteria of Zoneid : ", zoneId)

	// 	if Zone.Id == zoneId {
	// 		// KT Cloud는 Zone별로 Spec이 다르므로 Zone 정보도 넘김.
	// 		ktZoneName = Zone.Name

	// 		break
	// 	}
	// }
	// cblogger.Info("Zone Name : ", ktZoneName)


	// Caution!! : KT Cloud는 Image info/VMSpc info 조회시 zoneid로 조회함.
	result, err := vmSpecHandler.Client.ListAvailableProductTypes(zoneId)
	if err != nil {
		cblogger.Error("Failed to Get List of Available Product Types: %s", err)
		return irs.VMSpecInfo{}, err
	}

	if len(result.Listavailableproducttypesresponse.ProductTypes) < 1 {
		return irs.VMSpecInfo{}, errors.New("Failed to Find Product types!!")
	}

	// Name ex) d3530ad2-462b-43ad-97d5-e1087b952b7d!87c0a6f6-c684-4fbe-a393-d8412bcf788d_disk100GB
	// Caution : 아래의 string split 기호 중 ! 대신 #을 사용하면 CB-Spider API를 통해 call할 시 전체의 string이 전달되지 않고 # 전까지만 전달됨. 
	instanceSpecString := strings.Split(VMSpecName, "!")
	for i := range instanceSpecString {
		cblogger.Info("instanceSpecString : ", instanceSpecString[i])
	}

	ktVMSpecId := instanceSpecString[0]
	cblogger.Info("vmSpecID : ", ktVMSpecId)

    // Ex) 87c0a6f6-c684-4fbe-a393-d8412bcf788d_disk100GB
	tempOfferingString := instanceSpecString[1]
	cblogger.Info("tempOfferingString : ", tempOfferingString)

	diskOfferingString := strings.Split(tempOfferingString, "_")

	ktDiskOfferingId := diskOfferingString[0]
	cblogger.Info("ktDiskOfferingId : ", ktDiskOfferingId)

	for _, productType := range result.Listavailableproducttypesresponse.ProductTypes {
		cblogger.Info("# Search criteria of Serviceofferingid : ", ktVMSpecId)		
		// if serverProductType.ServiceOfferingId == ktVMSpecId {
		if productType.ServiceOfferingId == ktVMSpecId {
			if productType.DiskOfferingId == ktDiskOfferingId {
				resultVMSpecInfo = mappingVMSpecInfo(zoneId, "", productType) //Spec 상세 정보 조회시 Image 정보는 불필요
				break
			}
		}
	}
	return resultVMSpecInfo, nil
}

func (vmSpecHandler *KtCloudVMSpecHandler) ListVMSpec() ([]*irs.VMSpecInfo, error) {
	cblogger.Info("KT Cloud cloud driver: called ListVMSpec()!")

	regionInfo := vmSpecHandler.RegionInfo.Region
	zoneId := vmSpecHandler.RegionInfo.Zone

	cblogger.Infof("Region : [%s] ", regionInfo)
	cblogger.Info("vmSpecHandler.RegionInfo.Zone : ", zoneId)
	
	// Image(KT Cloud Template) list 조회 (Name)
	imageHandler := KtCloudImageHandler{
		Client:         vmSpecHandler.Client,
		RegionInfo:     vmSpecHandler.RegionInfo,  //CAUTION!! : Must input this!!
	}
	cblogger.Info("imageHandler.RegionInfo.Zone : ", imageHandler.RegionInfo.Zone)  //Need to Check!!
	
	imageListResult, err := imageHandler.ListImage()
	if err != nil {
		cblogger.Infof("Failed to Get Image list!! : ", err)

		return nil, errors.New("Failed to Get Image list!!")
	} else {
		cblogger.Info("Succeeded in Getting Image list!!")
		cblogger.Info("Image list Count : ", len(imageListResult))
		// spew.Dump(imageListResult)
	}

	specListResult, err := vmSpecHandler.Client.ListAvailableProductTypes(zoneId)
	if err != nil {
		cblogger.Error("Failed to Get List of Available Product Types: %s", err)
		return []*irs.VMSpecInfo{}, errors.New("Failed to Get Product Type list!!")
	} else {
		cblogger.Info("Succeeded in Getting Product Type list!!")
	}
	cblogger.Info("Spec list Count : ", len(specListResult.Listavailableproducttypesresponse.ProductTypes))
	// spew.Dump(specListResult)
	// spew.Dump(result)

	if len(specListResult.Listavailableproducttypesresponse.ProductTypes) < 1 {
		return []*irs.VMSpecInfo{}, errors.New("Failed to Find Product types!!")
	}

	var vmSpecInfoList []*irs.VMSpecInfo //Cloud-Barista Spec Info.
	for _, image := range imageListResult {
		cblogger.Info("# 기준 KT Cloud Image ID(Product type -> Template) : ", image.IId.SystemId)

		for _, productType := range specListResult.Listavailableproducttypesresponse.ProductTypes {
			var serverProductType ktsdk.ProductTypes
			serverProductType = productType

			if image.IId.SystemId == productType.TemplateId {

			vmSpecInfo := mappingVMSpecInfo(zoneId, image.IId.SystemId, serverProductType)
			vmSpecInfoList = append(vmSpecInfoList, &vmSpecInfo)
			}
		}
	}	
	cblogger.Info("# Supported VM Spec count : ", len(vmSpecInfoList))
	return vmSpecInfoList, nil
}

//Extract instance Specification information
func mappingVMSpecInfo(ZoneId string, ImageId string, ktServerProductType ktsdk.ProductTypes) irs.VMSpecInfo {
	cblogger.Infof("\n*** Mapping VMSpecInfo : SpecName: [%s]", ktServerProductType.ServiceOfferingId)
	// spew.Dump(vmSpec)

	// Caution : 아래의 string split 기호 중 ! 대신 #을 사용하면 CB-Spider API를 통해 call할 시 전체의 string이 전달되지 않고 # 전까지만 전달됨. 
	ktVMSpecId := ktServerProductType.ServiceOfferingId + "!" + ktServerProductType.DiskOfferingId + "_disk" + ktServerProductType.DiskOfferingDesc
	ktVMSpecString := ktServerProductType.ServiceOfferingDesc
	// ex) ktServerProductType.Serviceofferingdesc => "XS71 12vCore 16GB" //Caution!!
	// vCpuCount := strtmp[5:7]
	// productMem := strtmp[13:15]

	// Split 함수로 문자열을 " "(공백) 기준으로 분리
	specSlice := strings.Split(ktVMSpecString, " ")
	for _, str := range specSlice {
		fmt.Println(str)
	}

	// KT Cloud에서 core 수를 '4vcore' or '16vCore'와 같은 형태로 제공함.(String 처리시 자리수, 대수문자 주의 필요)
    // vCpuCount := productVCpu[0:2] // 24vCore는 괜찮지만, 1vCore 같은 경우도 있어서 전체 자리수가 달라 적당하지 않음

	productVCpu := strings.Replace(specSlice[1], "C", "c", 1) //대문자 C가 있으면 소문자 c로 변경 ex) 1vCore -> 1vcore
	productVCpu = strings.TrimSuffix(productVCpu, "vcore") // 문자열 우측에서 'vcore'를 제거
	productMem := strings.TrimRight(specSlice[2], "GB") // 문자열 우측에서 G와 B를 제거
	//productMem := strings.TrimSuffix(specSlice[2], "GB")

	MemCountGb, err := strconv.Atoi(productMem) // 문자열을 숫자으로 변환
	if err != nil {
		cblogger.Error(err)
	}
	
	MemCountMb := MemCountGb*1024
	MemCountMbStr := strconv.Itoa(MemCountMb) // 숫자를 문자열로 변환

	// In case : ktServerProductType.DiskOfferingDesc : 100G (After Disk Add)
	// if len(ktServerProductType.DiskOfferingId) > 1 {
	// 	ktVMSpecID = ktServerProductType.ServiceOfferingId + "#" + ktServerProductType.DiskOfferingId + "_disk" + ktServerProductType.DiskOfferingDesc
	// }

	vmSpecInfo := irs.VMSpecInfo{
		Region: ktServerProductType.ZoneDesc,
		// Region: ktServerProductType.ZoneId,
		Name:   ktVMSpecId,
		VCpu: irs.VCpuInfo{Count: productVCpu, Clock: "N/A"},
		// VCpu: irs.VCpuInfo{Count: vCpuCount, Clock: "N/A"},
		Mem: MemCountMbStr,
		// Mem: productMem,

		// GPU 정보는 없음.
		Gpu: []irs.GpuInfo{{Count: "N/A", Mfr: "N/A", Model: "N/A", Mem: "N/A"}},

		// KT Cloud는 Zone별로 지원 Spec이 다르므로 Zone 정보도 제시
		KeyValueList: []irs.KeyValue{
			{Key: "Zone", Value: ktServerProductType.ZoneDesc},
			{Key: "KtServiceOffering", Value: ktServerProductType.ServiceOfferingDesc},	
			{Key: "DiskSize", Value: ktServerProductType.DiskOfferingDesc},
			//{Key: "AdditionalDiskOfferingID", Value: ktServerProductType.DiskOfferingId},
			{Key: "SupportingImage(Template)ID", Value: ImageId},
			{Key: "ProductState", Value: ktServerProductType.ProductState},
		},
	}
	return vmSpecInfo
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
