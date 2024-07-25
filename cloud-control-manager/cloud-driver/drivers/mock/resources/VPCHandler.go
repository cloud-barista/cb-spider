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
	"sync"

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

var vpcMapLock = new(sync.RWMutex)

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
		IId:            vpcReqInfo.IId,
		IPv4_CIDR:      vpcReqInfo.IPv4_CIDR,
		SubnetInfoList: vpcReqInfo.SubnetInfoList,
		TagList:        vpcReqInfo.TagList,
		KeyValueList:   nil,
	}

	// (2) insert VPCInfo into global Map
	vpcMapLock.Lock()
	defer vpcMapLock.Unlock()
	infoList, _ := vpcInfoMap[mockName]
	infoList = append(infoList, &vpcInfo)
	vpcInfoMap[mockName] = infoList

	return CloneVPCInfo(vpcInfo), nil
}

func (vpcHandler *MockVPCHandler) ListVPC() ([]*irs.VPCInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called ListVPC()!")

	mockName := vpcHandler.MockName
	vpcMapLock.RLock()
	defer vpcMapLock.RUnlock()
	infoList, ok := vpcInfoMap[mockName]
	if !ok {
		return []*irs.VPCInfo{}, nil
	}

	// cloning list of VPC
	return CloneVPCInfoList(infoList), nil
}

func CloneVPCInfoList(srcInfoList []*irs.VPCInfo) []*irs.VPCInfo {
	clonedInfoList := []*irs.VPCInfo{}
	for _, srcInfo := range srcInfoList {
		clonedInfo := CloneVPCInfo(*srcInfo)
		clonedInfoList = append(clonedInfoList, &clonedInfo)
	}
	return clonedInfoList
}

func CloneVPCInfo(srcInfo irs.VPCInfo) irs.VPCInfo {
	/*
		type VPCInfo struct {
			IId            IID // {NameId, SystemId}
			IPv4_CIDR      string
			SubnetInfoList []SubnetInfo

			TagList      []KeyValue
			KeyValueList []KeyValue
		}
	*/

	// clone VPCInfo
	clonedInfo := irs.VPCInfo{
		IId:            irs.IID{srcInfo.IId.NameId, srcInfo.IId.SystemId},
		IPv4_CIDR:      srcInfo.IPv4_CIDR,
		SubnetInfoList: CloneSubnetInfoList(srcInfo.SubnetInfoList),
		TagList:        srcInfo.TagList, // clone TagList
		KeyValueList:   srcInfo.KeyValueList,
	}

	return clonedInfo
}

func CloneSubnetInfoList(srcInfoList []irs.SubnetInfo) []irs.SubnetInfo {
	clonedInfoList := []irs.SubnetInfo{}
	for _, srcInfo := range srcInfoList {
		clonedInfo := CloneSubnetInfo(srcInfo)
		clonedInfoList = append(clonedInfoList, clonedInfo)
	}
	return clonedInfoList
}

func CloneSubnetInfo(srcInfo irs.SubnetInfo) irs.SubnetInfo {
	/*
		type SubnetInfo struct {
			IId       IID    // {NameId, SystemId}
			Zone      string // Target Zone Name
			IPv4_CIDR string

			TagList      []KeyValue
			KeyValueList []KeyValue
		}
	*/

	// clone SubnetInfo
	clonedInfo := irs.SubnetInfo{
		IId:          irs.IID{srcInfo.IId.NameId, srcInfo.IId.SystemId},
		Zone:         srcInfo.Zone,
		IPv4_CIDR:    srcInfo.IPv4_CIDR,
		TagList:      srcInfo.TagList, // clone TagList
		KeyValueList: srcInfo.KeyValueList,
	}

	return clonedInfo
}

func (vpcHandler *MockVPCHandler) GetVPC(iid irs.IID) (irs.VPCInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called GetVPC()!")

	vpcMapLock.RLock()
	defer vpcMapLock.RUnlock()
	mockName := vpcHandler.MockName
	infoList, ok := vpcInfoMap[mockName]
	if !ok {
		return irs.VPCInfo{}, fmt.Errorf("%s VPC does not exist!!", iid.NameId)
	}

	for _, info := range infoList {
		if info.IId.NameId == iid.NameId {
			return CloneVPCInfo(*info), nil
		}
	}

	return irs.VPCInfo{}, fmt.Errorf("%s VPC does not exist!!", iid.NameId)
}

func (vpcHandler *MockVPCHandler) DeleteVPC(iid irs.IID) (bool, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called DeleteVPC()!")

	vpcMapLock.Lock()
	defer vpcMapLock.Unlock()

	mockName := vpcHandler.MockName
	infoList, ok := vpcInfoMap[mockName]
	if !ok {
		return false, fmt.Errorf("%s VPC does not exist!!", iid.NameId)
	}

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

	vpcMapLock.Lock()
	defer vpcMapLock.Unlock()

	mockName := vpcHandler.MockName
	infoList, ok := vpcInfoMap[mockName]
	if !ok {
		return irs.VPCInfo{}, fmt.Errorf("%s VPC does not exist!!", iid.NameId)
	}

	subnetInfo.IId.SystemId = subnetInfo.IId.NameId
	for _, info := range infoList {
		if info.IId.NameId == iid.NameId {
			info.SubnetInfoList = append(info.SubnetInfoList, subnetInfo)

			return CloneVPCInfo(*info), nil
		}
	}

	return irs.VPCInfo{}, fmt.Errorf("%s VPC does not exist!!", iid.NameId)
}

func (vpcHandler *MockVPCHandler) RemoveSubnet(iid irs.IID, subnetIID irs.IID) (bool, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called RemoveSubnet()!")

	vpcMapLock.Lock()
	defer vpcMapLock.Unlock()

	mockName := vpcHandler.MockName
	infoList, ok := vpcInfoMap[mockName]
	if !ok {
		return false, fmt.Errorf("%s VPC does not exist!!", iid.NameId)
	}

	for _, info := range infoList {
		if info.IId.NameId == iid.NameId {
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
