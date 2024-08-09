package resources

import (
	"fmt"
	"sync"

	cblog "github.com/cloud-barista/cb-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

var vpcInfoMap map[string][]*irs.VPCInfo

type MockVPCHandler struct {
	MockName string
}

func init() {
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
	for i := range vpcReqInfo.SubnetInfoList {
		vpcReqInfo.SubnetInfoList[i].IId.SystemId = vpcReqInfo.SubnetInfoList[i].IId.NameId
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
	vpcInfoMap[mockName] = append(vpcInfoMap[mockName], &vpcInfo)

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

	return CloneVPCInfoList(infoList), nil
}

func CloneVPCInfoList(srcInfoList []*irs.VPCInfo) []*irs.VPCInfo {
	clonedInfoList := make([]*irs.VPCInfo, len(srcInfoList))
	for i, srcInfo := range srcInfoList {
		clonedInfo := CloneVPCInfo(*srcInfo)
		clonedInfoList[i] = &clonedInfo
	}
	return clonedInfoList
}

func CloneVPCInfo(srcInfo irs.VPCInfo) irs.VPCInfo {
	clonedInfo := irs.VPCInfo{
		IId:            irs.IID{srcInfo.IId.NameId, srcInfo.IId.SystemId},
		IPv4_CIDR:      srcInfo.IPv4_CIDR,
		SubnetInfoList: CloneSubnetInfoList(srcInfo.SubnetInfoList),
		TagList:        srcInfo.TagList, // 필요시 깊은 복사 추가 가능
		KeyValueList:   srcInfo.KeyValueList,
	}

	return clonedInfo
}

func CloneSubnetInfoList(srcInfoList []irs.SubnetInfo) []irs.SubnetInfo {
	clonedInfoList := make([]irs.SubnetInfo, len(srcInfoList))
	for i, srcInfo := range srcInfoList {
		clonedInfoList[i] = CloneSubnetInfo(srcInfo)
	}
	return clonedInfoList
}

func CloneSubnetInfo(srcInfo irs.SubnetInfo) irs.SubnetInfo {
	clonedInfo := irs.SubnetInfo{
		IId:          irs.IID{srcInfo.IId.NameId, srcInfo.IId.SystemId},
		Zone:         srcInfo.Zone,
		IPv4_CIDR:    srcInfo.IPv4_CIDR,
		TagList:      srcInfo.TagList, // 필요시 깊은 복사 추가 가능
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
			vpcInfoMap[mockName] = append(infoList[:idx], infoList[idx+1:]...)
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
	for i, info := range infoList {
		if info.IId.NameId == iid.NameId {
			vpcInfoMap[mockName][i].SubnetInfoList = append(info.SubnetInfoList, subnetInfo)
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

	for i, info := range infoList {
		if info.IId.NameId == iid.NameId {
			for idx, subInfo := range info.SubnetInfoList {
				if subInfo.IId.SystemId == subnetIID.SystemId {
					vpcInfoMap[mockName][i].SubnetInfoList = append(info.SubnetInfoList[:idx], info.SubnetInfoList[idx+1:]...)
					return true, nil
				}
			}
		}
	}

	return false, nil
}
