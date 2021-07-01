// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Mock Driver.
//
// by CB-Spider Team, 2020.10.

package resources

import (
	"fmt"

	cblog "github.com/cloud-barista/cb-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	_ "github.com/sirupsen/logrus"
)

var vpcInfoMap map[string][]*irs.VPCInfo

type MockVPCHandler struct {
	MockName string
}

func init() {
	// cblog is a global variable.
	vpcInfoMap = make(map[string][]*irs.VPCInfo)
}

// (1) create vpcInfo object
// (2) insert vpcInfo into global Map
func (vpcHandler *MockVPCHandler) CreateVPC(vpcReqInfo irs.VPCReqInfo) (irs.VPCInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called CreateVPC()!")

	mockName := vpcHandler.MockName
	vpcReqInfo.IId.SystemId = vpcReqInfo.IId.NameId

	// set SystemID of Subnet list
	for i, subnetInfo := range vpcReqInfo.SubnetInfoList {
		subnetInfo.IId.SystemId = subnetInfo.IId.NameId
		vpcReqInfo.SubnetInfoList[i] = subnetInfo
	}

	// (1) create vpcInfo object
	vpcInfo := irs.VPCInfo{
		vpcReqInfo.IId,
		vpcReqInfo.IPv4_CIDR,
		vpcReqInfo.SubnetInfoList,
		nil}

	// (2) insert VPCInfo into global Map
	infoList, _ := vpcInfoMap[mockName]
	infoList = append(infoList, &vpcInfo)
	vpcInfoMap[mockName] = infoList

	return vpcInfo, nil
}

func (vpcHandler *MockVPCHandler) ListVPC() ([]*irs.VPCInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called ListVPC()!")

	mockName := vpcHandler.MockName
	infoList, ok := vpcInfoMap[mockName]
	if !ok {
		return []*irs.VPCInfo{}, nil
	}
	// cloning list of VPC
	resultList := make([]*irs.VPCInfo, len(infoList))
	copy(resultList, infoList)
	return resultList, nil
}

func (vpcHandler *MockVPCHandler) GetVPC(iid irs.IID) (irs.VPCInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called GetVPC()!")

	infoList, err := vpcHandler.ListVPC()
	if err != nil {
		cblogger.Error(err)
		return irs.VPCInfo{}, err
	}

	for _, info := range infoList {
		if (*info).IId.NameId == iid.NameId {
			return *info, nil
		}
	}

	return irs.VPCInfo{}, fmt.Errorf("%s VPCGroup does not exist!!", iid.NameId)
}

func (vpcHandler *MockVPCHandler) DeleteVPC(iid irs.IID) (bool, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called DeleteVPC()!")

	infoList, err := vpcHandler.ListVPC()
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	mockName := vpcHandler.MockName
	for idx, info := range infoList {
		if info.IId.SystemId == iid.SystemId {
			infoList = append(infoList[:idx], infoList[idx+1:]...)
			vpcInfoMap[mockName] = infoList
			return true, nil
		}
	}
	return false, nil
}

func (vpcHandler *MockVPCHandler) AddSubnet(iid irs.IID, subnetInfo irs.SubnetInfo) (irs.VPCInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called AddSubnet()!")

	infoList, err := vpcHandler.ListVPC()
	if err != nil {
		cblogger.Error(err)
		return irs.VPCInfo{}, err
	}

	subnetInfo.IId.SystemId = subnetInfo.IId.NameId
	for _, info := range infoList {
		if (*info).IId.NameId == iid.NameId {
			info.SubnetInfoList = append(info.SubnetInfoList, subnetInfo)
			return *info, nil
		}
	}

	return irs.VPCInfo{}, fmt.Errorf("%s VPCGroup does not exist!!", iid.NameId)
}

func (vpcHandler *MockVPCHandler) RemoveSubnet(iid irs.IID, subnetIID irs.IID) (bool, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called RemoveSubnet()!")

	infoList, err := vpcHandler.ListVPC()
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	for _, info := range infoList {
		if (*info).IId.NameId == iid.NameId {
			for idx, subInfo := range info.SubnetInfoList {
				if subInfo.IId.SystemId == subnetIID.SystemId {
					info.SubnetInfoList = append(info.SubnetInfoList[:idx], info.SubnetInfoList[idx+1:]...)
					return true, nil
				}
			}
		}
	}

	return false, nil
}
