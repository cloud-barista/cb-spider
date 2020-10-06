// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Mock Driver.
//
// by CB-Spider Team, 2020.09.

package resources

import (
        cblog "github.com/cloud-barista/cb-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"fmt"
)

var vmSpecInfoMap map[string][]*irs.VMSpecInfo

type MockVMSpecHandler struct {
	MockName      string
}

//var vmSpecHandler irs.VMSpecHandler
var PrepareInfoList []irs.VMSpecInfo

func init() {
        // cblog is a global variable.
	vmSpecInfoMap = make(map[string][]*irs.VMSpecInfo)
	prepare("MockDriver-01")
}

func prepare(mockName string) {
        cblogger := cblog.GetLogger("CB-SPIDER")
        cblogger.Info("Mock Driver: called prepare()!")
/*
	PrepareInfoList := []irs.VMSpecInfo{
		{Region: "mock-region01", Name: "mock-vmspec-01", VCpu: {Count:"4", Clock:"2.7"}, Mem: "32768", Gpu: {Count:"2", Mfr:"NVIDIA", Model:"V100", Mem:"16384MB"}, KeyValueList:nil},
		//{"mock-region01", "mock-vmspec-02", {"4", "3.2"}, "32768", {"1", "NVIDIA", "V100", "16384MB"}},
		//{"mock-region02", "mock-vmspec-03", {"8", "2.7"}, "62464"},
		//{"mock-region01", "mock-vmspec-04", {"8", "2.7"}, "1024"},
	}
        for _, info := range PrepareInfoList {
		infoList, _ := vmSpecInfoMap[mockName]
		infoList = append(infoList, &info)
		vmSpecInfoMap[mockName]=infoList
	}
*/
}

func (vmSpecHandler *MockVMSpecHandler) ListVMSpec(Region string) ([]*irs.VMSpecInfo, error) {
        cblogger := cblog.GetLogger("CB-SPIDER")
        cblogger.Info("Mock Driver: called ListVMSpec()!")
	
	mockName := vmSpecHandler.MockName
	infoList, ok := vmSpecInfoMap[mockName]
	if !ok {
		return []*irs.VMSpecInfo{}, nil
	}
	var list []*irs.VMSpecInfo
	for _, info := range infoList {
		if info.Region == Region {
			list = append(list, info)
		}
	}
	// cloning list of VMSpec
	resultList := make([]*irs.VMSpecInfo, len(list))
	copy(resultList, list)
	return resultList, nil
}

func (vmSpecHandler *MockVMSpecHandler) GetVMSpec(Region string, Name string) (irs.VMSpecInfo, error) {
        cblogger := cblog.GetLogger("CB-SPIDER")
        cblogger.Info("Mock Driver: called GetVMSpec()!")

	infoList, err := vmSpecHandler.ListVMSpec(Region)
	if err != nil {
		cblogger.Error(err)
		return irs.VMSpecInfo{}, err
	}

	for _, info := range infoList {
		if((*info).Name == Name) {
			return *info, nil
		}
	}
	
	return irs.VMSpecInfo{}, fmt.Errorf("%s VMSpec does not exist!!")
}


func (vmSpecHandler *MockVMSpecHandler) ListOrgVMSpec(Region string) (string, error) {             // return string: json format
        cblogger := cblog.GetLogger("CB-SPIDER")
        cblogger.Info("Mock Driver: called ListOrgVMSpec()!")
	return "", nil	
}

func (vmSpecHandler *MockVMSpecHandler) GetOrgVMSpec(Region string, Name string) (string, error) { // return string: json format
        cblogger := cblog.GetLogger("CB-SPIDER")
        cblogger.Info("Mock Driver: called GetOrgVMSpec()!")
	return "", nil	
}
