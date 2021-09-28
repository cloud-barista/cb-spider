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
	return CloneVPCInfoList(infoList), nil
}

func CloneVPCInfoList(srcInfoList []*irs.VPCInfo) ([]*irs.VPCInfo) {
    clonedInfoList := []*irs.VPCInfo{}
    for _, srcInfo := range srcInfoList {
        clonedInfo := CloneVPCInfo(*srcInfo)
        clonedInfoList = append(clonedInfoList, &clonedInfo)
    }
    return clonedInfoList
}

func CloneVPCInfo(srcInfo irs.VPCInfo) (irs.VPCInfo) {
    /*
	type VPCInfo struct {
		IId   IID       // {NameId, SystemId}
		IPv4_CIDR string
		SubnetInfoList []SubnetInfo

		KeyValueList []KeyValue
	}
    */

    // clone VPCInfo
    clonedInfo := irs.VPCInfo {
        IId: irs.IID{srcInfo.IId.NameId, srcInfo.IId.SystemId},
        IPv4_CIDR: srcInfo.IPv4_CIDR,
        SubnetInfoList: CloneSubnetInfoList(srcInfo.SubnetInfoList),

        // Need not clone
        KeyValueList: srcInfo.KeyValueList,
    }

    return clonedInfo
}

func CloneSubnetInfoList(srcInfoList []irs.SubnetInfo) ([]irs.SubnetInfo) {
    clonedInfoList := []irs.SubnetInfo{}
    for _, srcInfo := range srcInfoList {
        clonedInfo := CloneSubnetInfo(srcInfo)
        clonedInfoList = append(clonedInfoList, clonedInfo)
    }
    return clonedInfoList
}

func CloneSubnetInfo(srcInfo irs.SubnetInfo) (irs.SubnetInfo) {
    /*
	type SubnetInfo struct {
		IId   IID       // {NameId, SystemId}
		IPv4_CIDR string

		KeyValueList []KeyValue
	}
    */

    // clone SubnetInfo
    clonedInfo := irs.SubnetInfo {
        IId: irs.IID{srcInfo.IId.NameId, srcInfo.IId.SystemId},
        IPv4_CIDR: srcInfo.IPv4_CIDR,

        // Need not clone
        KeyValueList: srcInfo.KeyValueList,
    }

    return clonedInfo
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
			return CloneVPCInfo(*info), nil
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

	// infoList: cloned list
	infoList, err := vpcHandler.ListVPC()
	if err != nil {
		cblogger.Error(err)
		return irs.VPCInfo{}, err
	}

	subnetInfo.IId.SystemId = subnetInfo.IId.NameId
	for _, info := range infoList {
		if (*info).IId.NameId == iid.NameId {
			info.SubnetInfoList = append(info.SubnetInfoList, subnetInfo)

			// don't forget, info is cloned object.
			// delete VPCInfo from global Map
			vpcHandler.DeleteVPC(info.IId)

			// insert VPCInfo into global Map
			mockName := vpcHandler.MockName
			infoList, _ := vpcInfoMap[mockName]
			infoList = append(infoList, info)
			vpcInfoMap[mockName] = infoList

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

					// don't forget, info is cloned object.
					// delete VPCInfo from global Map
					vpcHandler.DeleteVPC(info.IId)

					// insert VPCInfo into global Map
					mockName := vpcHandler.MockName
					infoList, _ := vpcInfoMap[mockName]
					infoList = append(infoList, info)
					vpcInfoMap[mockName] = infoList

					return true, nil
				}
			}
		}
	}

	return false, nil
}
