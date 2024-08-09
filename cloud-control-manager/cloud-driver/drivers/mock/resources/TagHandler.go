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
	"fmt"
	"strings"

	cblog "github.com/cloud-barista/cb-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type MockTagHandler struct {
	MockName string
}

func (tagHandler *MockTagHandler) AddTag(resType irs.RSType, resIID irs.IID, tag irs.KeyValue) (irs.KeyValue, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called AddTag()!")

	mockName := tagHandler.MockName
	resType = irs.RSType(strings.ToLower(string(resType)))

	switch resType {
	case irs.VPC:
		return addTagToVPC(mockName, resIID, tag)
	case irs.SUBNET:
		return addTagToSubnet(mockName, resIID, tag)
	case irs.SG:
		return addTagToSG(mockName, resIID, tag)
	case irs.KEY:
		return addTagToKeyPair(mockName, resIID, tag)
	case irs.VM:
		return addTagToVM(mockName, resIID, tag)
	case irs.NLB:
		return addTagToNLB(mockName, resIID, tag)
	case irs.DISK:
		return addTagToDisk(mockName, resIID, tag)
	case irs.MYIMAGE:
		return addTagToMyImage(mockName, resIID, tag)
	case irs.CLUSTER:
		return addTagToCluster(mockName, resIID, tag)
	default:
		return irs.KeyValue{}, fmt.Errorf("unsupported resource type %s", resType)
	}
}

func addTagToVPC(mockName string, resIID irs.IID, tag irs.KeyValue) (irs.KeyValue, error) {
	vpcMapLock.Lock()
	defer vpcMapLock.Unlock()
	infoList, ok := vpcInfoMap[mockName]
	if !ok {
		return irs.KeyValue{}, fmt.Errorf("resource not found for %s", resIID.NameId)
	}
	for i, info := range infoList {
		if info.IId.NameId == resIID.NameId {
			vpcInfoMap[mockName][i].TagList = append(info.TagList, tag)
			return tag, nil
		}
	}
	return irs.KeyValue{}, fmt.Errorf("resource not found for %s", resIID.NameId)
}

func addTagToSubnet(mockName string, resIID irs.IID, tag irs.KeyValue) (irs.KeyValue, error) {
	vpcMapLock.Lock()
	defer vpcMapLock.Unlock()

	vpcInfoList, ok := vpcInfoMap[mockName]
	if !ok {
		return irs.KeyValue{}, fmt.Errorf("VPC not found for %s", resIID.NameId)
	}

	for vpcIdx, vpcInfo := range vpcInfoList {
		for subnetIdx, subnetInfo := range vpcInfo.SubnetInfoList {
			if subnetInfo.IId.NameId == resIID.NameId {
				vpcInfoMap[mockName][vpcIdx].SubnetInfoList[subnetIdx].TagList = append(subnetInfo.TagList, tag)
				return tag, nil
			}
		}
	}
	return irs.KeyValue{}, fmt.Errorf("Subnet not found for %s", resIID.NameId)
}

func addTagToSG(mockName string, resIID irs.IID, tag irs.KeyValue) (irs.KeyValue, error) {
	sgMapLock.Lock()
	defer sgMapLock.Unlock()
	infoList, ok := securityInfoMap[mockName]
	if !ok {
		return irs.KeyValue{}, fmt.Errorf("resource not found for %s", resIID.NameId)
	}
	for i, info := range infoList {
		if info.IId.NameId == resIID.NameId {
			securityInfoMap[mockName][i].TagList = append(info.TagList, tag)
			return tag, nil
		}
	}
	return irs.KeyValue{}, fmt.Errorf("resource not found for %s", resIID.NameId)
}

func addTagToKeyPair(mockName string, resIID irs.IID, tag irs.KeyValue) (irs.KeyValue, error) {
	keyMapLock.Lock()
	defer keyMapLock.Unlock()
	infoList, ok := keyPairInfoMap[mockName]
	if !ok {
		return irs.KeyValue{}, fmt.Errorf("resource not found for %s", resIID.NameId)
	}
	for i, info := range infoList {
		if info.IId.NameId == resIID.NameId {
			keyPairInfoMap[mockName][i].TagList = append(info.TagList, tag)
			return tag, nil
		}
	}
	return irs.KeyValue{}, fmt.Errorf("resource not found for %s", resIID.NameId)
}

func addTagToVM(mockName string, resIID irs.IID, tag irs.KeyValue) (irs.KeyValue, error) {
	vmMapLock.Lock()
	defer vmMapLock.Unlock()
	infoList, ok := vmInfoMap[mockName]
	if !ok {
		return irs.KeyValue{}, fmt.Errorf("resource not found for %s", resIID.NameId)
	}
	for i, info := range infoList {
		if info.IId.NameId == resIID.NameId {
			vmInfoMap[mockName][i].TagList = append(info.TagList, tag)
			return tag, nil
		}
	}
	return irs.KeyValue{}, fmt.Errorf("resource not found for %s", resIID.NameId)
}

func addTagToNLB(mockName string, resIID irs.IID, tag irs.KeyValue) (irs.KeyValue, error) {
	nlbMapLock.Lock()
	defer nlbMapLock.Unlock()
	infoList, ok := nlbInfoMap[mockName]
	if !ok {
		return irs.KeyValue{}, fmt.Errorf("resource not found for %s", resIID.NameId)
	}
	for i, info := range infoList {
		if info.IId.NameId == resIID.NameId {
			nlbInfoMap[mockName][i].TagList = append(info.TagList, tag)
			return tag, nil
		}
	}
	return irs.KeyValue{}, fmt.Errorf("resource not found for %s", resIID.NameId)
}

func addTagToDisk(mockName string, resIID irs.IID, tag irs.KeyValue) (irs.KeyValue, error) {
	diskMapLock.Lock()
	defer diskMapLock.Unlock()
	infoList, ok := diskInfoMap[mockName]
	if !ok {
		return irs.KeyValue{}, fmt.Errorf("resource not found for %s", resIID.NameId)
	}
	for i, info := range infoList {
		if info.IId.NameId == resIID.NameId {
			diskInfoMap[mockName][i].TagList = append(info.TagList, tag)
			return tag, nil
		}
	}
	return irs.KeyValue{}, fmt.Errorf("resource not found for %s", resIID.NameId)
}

func addTagToMyImage(mockName string, resIID irs.IID, tag irs.KeyValue) (irs.KeyValue, error) {
	myImageMapLock.Lock()
	defer myImageMapLock.Unlock()
	infoList, ok := myImageInfoMap[mockName]
	if !ok {
		return irs.KeyValue{}, fmt.Errorf("resource not found for %s", resIID.NameId)
	}
	for i, info := range infoList {
		if info.IId.NameId == resIID.NameId {
			myImageInfoMap[mockName][i].TagList = append(info.TagList, tag)
			return tag, nil
		}
	}
	return irs.KeyValue{}, fmt.Errorf("resource not found for %s", resIID.NameId)
}

func addTagToCluster(mockName string, resIID irs.IID, tag irs.KeyValue) (irs.KeyValue, error) {
	clusterMapLock.Lock()
	defer clusterMapLock.Unlock()
	infoList, ok := clusterInfoMap[mockName]
	if !ok {
		return irs.KeyValue{}, fmt.Errorf("resource not found for %s", resIID.NameId)
	}
	for i, info := range infoList {
		if info.IId.NameId == resIID.NameId {
			clusterInfoMap[mockName][i].TagList = append(info.TagList, tag)
			return tag, nil
		}
	}
	return irs.KeyValue{}, fmt.Errorf("resource not found for %s", resIID.NameId)
}

func (tagHandler *MockTagHandler) ListTag(resType irs.RSType, resIID irs.IID) ([]irs.KeyValue, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called ListTag()!")

	mockName := tagHandler.MockName
	resType = irs.RSType(strings.ToLower(string(resType)))

	switch resType {
	case irs.VPC:
		return listTagsFromVPC(mockName, resIID)
	case irs.SUBNET:
		return listTagsFromSubnet(mockName, resIID)
	case irs.SG:
		return listTagsFromSG(mockName, resIID)
	case irs.KEY:
		return listTagsFromKeyPair(mockName, resIID)
	case irs.VM:
		return listTagsFromVM(mockName, resIID)
	case irs.NLB:
		return listTagsFromNLB(mockName, resIID)
	case irs.DISK:
		return listTagsFromDisk(mockName, resIID)
	case irs.MYIMAGE:
		return listTagsFromMyImage(mockName, resIID)
	case irs.CLUSTER:
		return listTagsFromCluster(mockName, resIID)
	default:
		return nil, fmt.Errorf("unsupported resource type %s", resType)
	}
}

func listTagsFromVPC(mockName string, resIID irs.IID) ([]irs.KeyValue, error) {
	vpcMapLock.RLock()
	defer vpcMapLock.RUnlock()
	infoList, ok := vpcInfoMap[mockName]
	if !ok {
		return nil, fmt.Errorf("resource not found for %s", resIID.NameId)
	}
	for _, info := range infoList {
		if info.IId.NameId == resIID.NameId {
			return info.TagList, nil
		}
	}
	return nil, fmt.Errorf("resource not found for %s", resIID.NameId)
}

func listTagsFromSubnet(mockName string, resIID irs.IID) ([]irs.KeyValue, error) {
	vpcMapLock.RLock()
	defer vpcMapLock.RUnlock()
	vpcInfoList, ok := vpcInfoMap[mockName]
	if !ok {
		return nil, fmt.Errorf("VPC not found for %s", resIID.NameId)
	}
	for _, vpcInfo := range vpcInfoList {
		for _, subnetInfo := range vpcInfo.SubnetInfoList {
			if subnetInfo.IId.NameId == resIID.NameId {
				return subnetInfo.TagList, nil
			}
		}
	}
	return nil, fmt.Errorf("Subnet not found for %s", resIID.NameId)
}

func listTagsFromSG(mockName string, resIID irs.IID) ([]irs.KeyValue, error) {
	sgMapLock.RLock()
	defer sgMapLock.RUnlock()
	infoList, ok := securityInfoMap[mockName]
	if !ok {
		return nil, fmt.Errorf("resource not found for %s", resIID.NameId)
	}
	for _, info := range infoList {
		if info.IId.NameId == resIID.NameId {
			return info.TagList, nil
		}
	}
	return nil, fmt.Errorf("resource not found for %s", resIID.NameId)
}

func listTagsFromKeyPair(mockName string, resIID irs.IID) ([]irs.KeyValue, error) {
	keyMapLock.RLock()
	defer keyMapLock.RUnlock()
	infoList, ok := keyPairInfoMap[mockName]
	if !ok {
		return nil, fmt.Errorf("resource not found for %s", resIID.NameId)
	}
	for _, info := range infoList {
		if info.IId.NameId == resIID.NameId {
			return info.TagList, nil
		}
	}
	return nil, fmt.Errorf("resource not found for %s", resIID.NameId)
}

func listTagsFromVM(mockName string, resIID irs.IID) ([]irs.KeyValue, error) {
	vmMapLock.RLock()
	defer vmMapLock.RUnlock()
	infoList, ok := vmInfoMap[mockName]
	if !ok {
		return nil, fmt.Errorf("resource not found for %s", resIID.NameId)
	}
	for _, info := range infoList {
		if info.IId.NameId == resIID.NameId {
			return info.TagList, nil
		}
	}
	return nil, fmt.Errorf("resource not found for %s", resIID.NameId)
}

func listTagsFromNLB(mockName string, resIID irs.IID) ([]irs.KeyValue, error) {
	nlbMapLock.RLock()
	defer nlbMapLock.RUnlock()
	infoList, ok := nlbInfoMap[mockName]
	if !ok {
		return nil, fmt.Errorf("resource not found for %s", resIID.NameId)
	}
	for _, info := range infoList {
		if info.IId.NameId == resIID.NameId {
			return info.TagList, nil
		}
	}
	return nil, fmt.Errorf("resource not found for %s", resIID.NameId)
}

func listTagsFromDisk(mockName string, resIID irs.IID) ([]irs.KeyValue, error) {
	diskMapLock.RLock()
	defer diskMapLock.RUnlock()
	infoList, ok := diskInfoMap[mockName]
	if !ok {
		return nil, fmt.Errorf("resource not found for %s", resIID.NameId)
	}
	for _, info := range infoList {
		if info.IId.NameId == resIID.NameId {
			return info.TagList, nil
		}
	}
	return nil, fmt.Errorf("resource not found for %s", resIID.NameId)
}

func listTagsFromMyImage(mockName string, resIID irs.IID) ([]irs.KeyValue, error) {
	myImageMapLock.RLock()
	defer myImageMapLock.RUnlock()
	infoList, ok := myImageInfoMap[mockName]
	if !ok {
		return nil, fmt.Errorf("resource not found for %s", resIID.NameId)
	}
	for _, info := range infoList {
		if info.IId.NameId == resIID.NameId {
			return info.TagList, nil
		}
	}
	return nil, fmt.Errorf("resource not found for %s", resIID.NameId)
}

func listTagsFromCluster(mockName string, resIID irs.IID) ([]irs.KeyValue, error) {
	clusterMapLock.RLock()
	defer clusterMapLock.RUnlock()
	infoList, ok := clusterInfoMap[mockName]
	if !ok {
		return nil, fmt.Errorf("resource not found for %s", resIID.NameId)
	}
	for _, info := range infoList {
		if info.IId.NameId == resIID.NameId {
			return info.TagList, nil
		}
	}
	return nil, fmt.Errorf("resource not found for %s", resIID.NameId)
}

func (tagHandler *MockTagHandler) GetTag(resType irs.RSType, resIID irs.IID, key string) (irs.KeyValue, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called GetTag()!")

	mockName := tagHandler.MockName
	resType = irs.RSType(strings.ToLower(string(resType)))

	switch resType {
	case irs.VPC:
		return getTagFromVPC(mockName, resIID, key)
	case irs.SUBNET:
		return getTagFromSubnet(mockName, resIID, key)
	case irs.SG:
		return getTagFromSG(mockName, resIID, key)
	case irs.KEY:
		return getTagFromKeyPair(mockName, resIID, key)
	case irs.VM:
		return getTagFromVM(mockName, resIID, key)
	case irs.NLB:
		return getTagFromNLB(mockName, resIID, key)
	case irs.DISK:
		return getTagFromDisk(mockName, resIID, key)
	case irs.MYIMAGE:
		return getTagFromMyImage(mockName, resIID, key)
	case irs.CLUSTER:
		return getTagFromCluster(mockName, resIID, key)
	default:
		return irs.KeyValue{}, fmt.Errorf("unsupported resource type %s", resType)
	}
}

func getTagFromVPC(mockName string, resIID irs.IID, key string) (irs.KeyValue, error) {
	vpcMapLock.RLock()
	defer vpcMapLock.RUnlock()
	infoList, ok := vpcInfoMap[mockName]
	if !ok {
		return irs.KeyValue{}, fmt.Errorf("resource not found for %s", resIID.NameId)
	}
	for _, info := range infoList {
		if info.IId.NameId == resIID.NameId {
			for _, tag := range info.TagList {
				if tag.Key == key {
					return tag, nil
				}
			}
		}
	}
	return irs.KeyValue{}, fmt.Errorf("tag %s not found for %s %s", key, irs.VPC, resIID.NameId)
}

func getTagFromSubnet(mockName string, resIID irs.IID, key string) (irs.KeyValue, error) {
	vpcMapLock.RLock()
	defer vpcMapLock.RUnlock()
	vpcInfoList, ok := vpcInfoMap[mockName]
	if !ok {
		return irs.KeyValue{}, fmt.Errorf("VPC not found for %s", resIID.NameId)
	}
	for _, vpcInfo := range vpcInfoList {
		for _, subnetInfo := range vpcInfo.SubnetInfoList {
			if subnetInfo.IId.NameId == resIID.NameId {
				for _, tag := range subnetInfo.TagList {
					if tag.Key == key {
						return tag, nil
					}
				}
			}
		}
	}
	return irs.KeyValue{}, fmt.Errorf("tag %s not found for %s %s", key, irs.SUBNET, resIID.NameId)
}

func getTagFromSG(mockName string, resIID irs.IID, key string) (irs.KeyValue, error) {
	sgMapLock.RLock()
	defer sgMapLock.RUnlock()
	infoList, ok := securityInfoMap[mockName]
	if !ok {
		return irs.KeyValue{}, fmt.Errorf("resource not found for %s", resIID.NameId)
	}
	for _, info := range infoList {
		if info.IId.NameId == resIID.NameId {
			for _, tag := range info.TagList {
				if tag.Key == key {
					return tag, nil
				}
			}
		}
	}
	return irs.KeyValue{}, fmt.Errorf("tag %s not found for %s %s", key, irs.SG, resIID.NameId)
}

func getTagFromKeyPair(mockName string, resIID irs.IID, key string) (irs.KeyValue, error) {
	keyMapLock.RLock()
	defer keyMapLock.RUnlock()
	infoList, ok := keyPairInfoMap[mockName]
	if !ok {
		return irs.KeyValue{}, fmt.Errorf("resource not found for %s", resIID.NameId)
	}
	for _, info := range infoList {
		if info.IId.NameId == resIID.NameId {
			for _, tag := range info.TagList {
				if tag.Key == key {
					return tag, nil
				}
			}
		}
	}
	return irs.KeyValue{}, fmt.Errorf("tag %s not found for %s %s", key, irs.KEY, resIID.NameId)
}

func getTagFromVM(mockName string, resIID irs.IID, key string) (irs.KeyValue, error) {
	vmMapLock.RLock()
	defer vmMapLock.RUnlock()
	infoList, ok := vmInfoMap[mockName]
	if !ok {
		return irs.KeyValue{}, fmt.Errorf("resource not found for %s", resIID.NameId)
	}
	for _, info := range infoList {
		if info.IId.NameId == resIID.NameId {
			for _, tag := range info.TagList {
				if tag.Key == key {
					return tag, nil
				}
			}
		}
	}
	return irs.KeyValue{}, fmt.Errorf("tag %s not found for %s %s", key, irs.VM, resIID.NameId)
}

func getTagFromNLB(mockName string, resIID irs.IID, key string) (irs.KeyValue, error) {
	nlbMapLock.RLock()
	defer nlbMapLock.RUnlock()
	infoList, ok := nlbInfoMap[mockName]
	if !ok {
		return irs.KeyValue{}, fmt.Errorf("resource not found for %s", resIID.NameId)
	}
	for _, info := range infoList {
		if info.IId.NameId == resIID.NameId {
			for _, tag := range info.TagList {
				if tag.Key == key {
					return tag, nil
				}
			}
		}
	}
	return irs.KeyValue{}, fmt.Errorf("tag %s not found for %s %s", key, irs.NLB, resIID.NameId)
}

func getTagFromDisk(mockName string, resIID irs.IID, key string) (irs.KeyValue, error) {
	diskMapLock.RLock()
	defer diskMapLock.RUnlock()
	infoList, ok := diskInfoMap[mockName]
	if !ok {
		return irs.KeyValue{}, fmt.Errorf("resource not found for %s", resIID.NameId)
	}
	for _, info := range infoList {
		if info.IId.NameId == resIID.NameId {
			for _, tag := range info.TagList {
				if tag.Key == key {
					return tag, nil
				}
			}
		}
	}
	return irs.KeyValue{}, fmt.Errorf("tag %s not found for %s %s", key, irs.DISK, resIID.NameId)
}

func getTagFromMyImage(mockName string, resIID irs.IID, key string) (irs.KeyValue, error) {
	myImageMapLock.RLock()
	defer myImageMapLock.RUnlock()
	infoList, ok := myImageInfoMap[mockName]
	if !ok {
		return irs.KeyValue{}, fmt.Errorf("resource not found for %s", resIID.NameId)
	}
	for _, info := range infoList {
		if info.IId.NameId == resIID.NameId {
			for _, tag := range info.TagList {
				if tag.Key == key {
					return tag, nil
				}
			}
		}
	}
	return irs.KeyValue{}, fmt.Errorf("tag %s not found for %s %s", key, irs.MYIMAGE, resIID.NameId)
}

func getTagFromCluster(mockName string, resIID irs.IID, key string) (irs.KeyValue, error) {
	clusterMapLock.RLock()
	defer clusterMapLock.RUnlock()
	infoList, ok := clusterInfoMap[mockName]
	if !ok {
		return irs.KeyValue{}, fmt.Errorf("resource not found for %s", resIID.NameId)
	}
	for _, info := range infoList {
		if info.IId.NameId == resIID.NameId {
			for _, tag := range info.TagList {
				if tag.Key == key {
					return tag, nil
				}
			}
		}
	}
	return irs.KeyValue{}, fmt.Errorf("tag %s not found for %s %s", key, irs.CLUSTER, resIID.NameId)
}

func (tagHandler *MockTagHandler) RemoveTag(resType irs.RSType, resIID irs.IID, key string) (bool, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called RemoveTag()!")

	mockName := tagHandler.MockName
	resType = irs.RSType(strings.ToLower(string(resType)))

	switch resType {
	case irs.VPC:
		return removeTagFromVPC(mockName, resIID, key)
	case irs.SUBNET:
		return removeTagFromSubnet(mockName, resIID, key)
	case irs.SG:
		return removeTagFromSG(mockName, resIID, key)
	case irs.KEY:
		return removeTagFromKeyPair(mockName, resIID, key)
	case irs.VM:
		return removeTagFromVM(mockName, resIID, key)
	case irs.NLB:
		return removeTagFromNLB(mockName, resIID, key)
	case irs.DISK:
		return removeTagFromDisk(mockName, resIID, key)
	case irs.MYIMAGE:
		return removeTagFromMyImage(mockName, resIID, key)
	case irs.CLUSTER:
		return removeTagFromCluster(mockName, resIID, key)
	default:
		return false, fmt.Errorf("unsupported resource type %s", resType)
	}
}

func removeTagFromVPC(mockName string, resIID irs.IID, key string) (bool, error) {
	vpcMapLock.Lock()
	defer vpcMapLock.Unlock()
	infoList, ok := vpcInfoMap[mockName]
	if !ok {
		return false, fmt.Errorf("resource not found for %s", resIID.NameId)
	}
	for i, info := range infoList {
		if info.IId.NameId == resIID.NameId {
			for idx, tag := range info.TagList {
				if tag.Key == key {
					vpcInfoMap[mockName][i].TagList = append(info.TagList[:idx], info.TagList[idx+1:]...)
					return true, nil
				}
			}
		}
	}
	return false, fmt.Errorf("tag %s not found for %s %s", key, irs.VPC, resIID.NameId)
}

func removeTagFromSubnet(mockName string, resIID irs.IID, key string) (bool, error) {
	vpcMapLock.Lock()
	defer vpcMapLock.Unlock()
	vpcInfoList, ok := vpcInfoMap[mockName]
	if !ok {
		return false, fmt.Errorf("VPC not found for %s", resIID.NameId)
	}
	for vpcIdx, vpcInfo := range vpcInfoList {
		for subnetIdx, subnetInfo := range vpcInfo.SubnetInfoList {
			if subnetInfo.IId.NameId == resIID.NameId {
				for idx, tag := range subnetInfo.TagList {
					if tag.Key == key {
						vpcInfoMap[mockName][vpcIdx].SubnetInfoList[subnetIdx].TagList = append(subnetInfo.TagList[:idx], subnetInfo.TagList[idx+1:]...)
						return true, nil
					}
				}
			}
		}
	}
	return false, fmt.Errorf("tag %s not found for %s %s", key, irs.SUBNET, resIID.NameId)
}

func removeTagFromSG(mockName string, resIID irs.IID, key string) (bool, error) {
	sgMapLock.Lock()
	defer sgMapLock.Unlock()
	infoList, ok := securityInfoMap[mockName]
	if !ok {
		return false, fmt.Errorf("resource not found for %s", resIID.NameId)
	}
	for i, info := range infoList {
		if info.IId.NameId == resIID.NameId {
			for idx, tag := range info.TagList {
				if tag.Key == key {
					securityInfoMap[mockName][i].TagList = append(info.TagList[:idx], info.TagList[idx+1:]...)
					return true, nil
				}
			}
		}
	}
	return false, fmt.Errorf("tag %s not found for %s %s", key, irs.SG, resIID.NameId)
}

func removeTagFromKeyPair(mockName string, resIID irs.IID, key string) (bool, error) {
	keyMapLock.Lock()
	defer keyMapLock.Unlock()
	infoList, ok := keyPairInfoMap[mockName]
	if !ok {
		return false, fmt.Errorf("resource not found for %s", resIID.NameId)
	}
	for i, info := range infoList {
		if info.IId.NameId == resIID.NameId {
			for idx, tag := range info.TagList {
				if tag.Key == key {
					keyPairInfoMap[mockName][i].TagList = append(info.TagList[:idx], info.TagList[idx+1:]...)
					return true, nil
				}
			}
		}
	}
	return false, fmt.Errorf("tag %s not found for %s %s", key, irs.KEY, resIID.NameId)
}

func removeTagFromVM(mockName string, resIID irs.IID, key string) (bool, error) {
	vmMapLock.Lock()
	defer vmMapLock.Unlock()
	infoList, ok := vmInfoMap[mockName]
	if !ok {
		return false, fmt.Errorf("resource not found for %s", resIID.NameId)
	}
	for i, info := range infoList {
		if info.IId.NameId == resIID.NameId {
			for idx, tag := range info.TagList {
				if tag.Key == key {
					vmInfoMap[mockName][i].TagList = append(info.TagList[:idx], info.TagList[idx+1:]...)
					return true, nil
				}
			}
		}
	}
	return false, fmt.Errorf("tag %s not found for %s %s", key, irs.VM, resIID.NameId)
}

func removeTagFromNLB(mockName string, resIID irs.IID, key string) (bool, error) {
	nlbMapLock.Lock()
	defer nlbMapLock.Unlock()
	infoList, ok := nlbInfoMap[mockName]
	if !ok {
		return false, fmt.Errorf("resource not found for %s", resIID.NameId)
	}
	for i, info := range infoList {
		if info.IId.NameId == resIID.NameId {
			for idx, tag := range info.TagList {
				if tag.Key == key {
					nlbInfoMap[mockName][i].TagList = append(info.TagList[:idx], info.TagList[idx+1:]...)
					return true, nil
				}
			}
		}
	}
	return false, fmt.Errorf("tag %s not found for %s %s", key, irs.NLB, resIID.NameId)
}

func removeTagFromDisk(mockName string, resIID irs.IID, key string) (bool, error) {
	diskMapLock.Lock()
	defer diskMapLock.Unlock()
	infoList, ok := diskInfoMap[mockName]
	if !ok {
		return false, fmt.Errorf("resource not found for %s", resIID.NameId)
	}
	for i, info := range infoList {
		if info.IId.NameId == resIID.NameId {
			for idx, tag := range info.TagList {
				if tag.Key == key {
					diskInfoMap[mockName][i].TagList = append(info.TagList[:idx], info.TagList[idx+1:]...)
					return true, nil
				}
			}
		}
	}
	return false, fmt.Errorf("tag %s not found for %s %s", key, irs.DISK, resIID.NameId)
}

func removeTagFromMyImage(mockName string, resIID irs.IID, key string) (bool, error) {
	myImageMapLock.Lock()
	defer myImageMapLock.Unlock()
	infoList, ok := myImageInfoMap[mockName]
	if !ok {
		return false, fmt.Errorf("resource not found for %s", resIID.NameId)
	}
	for i, info := range infoList {
		if info.IId.NameId == resIID.NameId {
			for idx, tag := range info.TagList {
				if tag.Key == key {
					myImageInfoMap[mockName][i].TagList = append(info.TagList[:idx], info.TagList[idx+1:]...)
					return true, nil
				}
			}
		}
	}
	return false, fmt.Errorf("tag %s not found for %s %s", key, irs.MYIMAGE, resIID.NameId)
}

func removeTagFromCluster(mockName string, resIID irs.IID, key string) (bool, error) {
	clusterMapLock.Lock()
	defer clusterMapLock.Unlock()
	infoList, ok := clusterInfoMap[mockName]
	if !ok {
		return false, fmt.Errorf("resource not found for %s", resIID.NameId)
	}
	for i, info := range infoList {
		if info.IId.NameId == resIID.NameId {
			for idx, tag := range info.TagList {
				if tag.Key == key {
					clusterInfoMap[mockName][i].TagList = append(info.TagList[:idx], info.TagList[idx+1:]...)
					return true, nil
				}
			}
		}
	}
	return false, fmt.Errorf("tag %s not found for %s %s", key, irs.CLUSTER, resIID.NameId)
}

func (tagHandler *MockTagHandler) FindTag(resType irs.RSType, keyword string) ([]*irs.TagInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called FindTag()!")

	mockName := tagHandler.MockName
	resType = irs.RSType(strings.ToLower(string(resType)))

	switch resType {
	case irs.VPC:
		return findTagInVPC(mockName, keyword)
	case irs.SUBNET:
		return findTagInSubnet(mockName, keyword)
	case irs.SG:
		return findTagInSG(mockName, keyword)
	case irs.KEY:
		return findTagInKeyPair(mockName, keyword)
	case irs.VM:
		return findTagInVM(mockName, keyword)
	case irs.NLB:
		return findTagInNLB(mockName, keyword)
	case irs.DISK:
		return findTagInDisk(mockName, keyword)
	case irs.MYIMAGE:
		return findTagInMyImage(mockName, keyword)
	case irs.CLUSTER:
		return findTagInCluster(mockName, keyword)
	default:
		return nil, fmt.Errorf("unsupported resource type %s", resType)
	}
}

func findTagInVPC(mockName string, keyword string) ([]*irs.TagInfo, error) {
	vpcMapLock.RLock()
	defer vpcMapLock.RUnlock()
	infoList, ok := vpcInfoMap[mockName]
	if !ok {
		return nil, fmt.Errorf("no tags found for resType %s", irs.VPC)
	}
	var result []*irs.TagInfo
	for _, info := range infoList {
		for _, tag := range info.TagList {
			if keyword == "" || keyword == "*" || tag.Key == keyword || tag.Value == keyword {
				result = append(result, &irs.TagInfo{
					ResType: irs.VPC,
					ResIId:  info.IId,
					TagList: []irs.KeyValue{tag},
				})
			}
		}
	}
	return result, nil
}

func findTagInSubnet(mockName string, keyword string) ([]*irs.TagInfo, error) {
	vpcMapLock.RLock()
	defer vpcMapLock.RUnlock()
	vpcInfoList, ok := vpcInfoMap[mockName]
	if !ok {
		return nil, fmt.Errorf("no tags found for resType %s", irs.SUBNET)
	}
	var result []*irs.TagInfo
	for _, vpcInfo := range vpcInfoList {
		for _, subnetInfo := range vpcInfo.SubnetInfoList {
			for _, tag := range subnetInfo.TagList {
				if keyword == "" || keyword == "*" || tag.Key == keyword || tag.Value == keyword {
					result = append(result, &irs.TagInfo{
						ResType: irs.SUBNET,
						ResIId:  subnetInfo.IId,
						TagList: []irs.KeyValue{tag},
					})
				}
			}
		}
	}
	return result, nil
}

func findTagInSG(mockName string, keyword string) ([]*irs.TagInfo, error) {
	sgMapLock.RLock()
	defer sgMapLock.RUnlock()
	infoList, ok := securityInfoMap[mockName]
	if !ok {
		return nil, fmt.Errorf("no tags found for resType %s", irs.SG)
	}
	var result []*irs.TagInfo
	for _, info := range infoList {
		for _, tag := range info.TagList {
			if keyword == "" || keyword == "*" || tag.Key == keyword || tag.Value == keyword {
				result = append(result, &irs.TagInfo{
					ResType: irs.SG,
					ResIId:  info.IId,
					TagList: []irs.KeyValue{tag},
				})
			}
		}
	}
	return result, nil
}

func findTagInKeyPair(mockName string, keyword string) ([]*irs.TagInfo, error) {
	keyMapLock.RLock()
	defer keyMapLock.RUnlock()
	infoList, ok := keyPairInfoMap[mockName]
	if !ok {
		return nil, fmt.Errorf("no tags found for resType %s", irs.KEY)
	}
	var result []*irs.TagInfo
	for _, info := range infoList {
		for _, tag := range info.TagList {
			if keyword == "" || keyword == "*" || tag.Key == keyword || tag.Value == keyword {
				result = append(result, &irs.TagInfo{
					ResType: irs.KEY,
					ResIId:  info.IId,
					TagList: []irs.KeyValue{tag},
				})
			}
		}
	}
	return result, nil
}

func findTagInVM(mockName string, keyword string) ([]*irs.TagInfo, error) {
	vmMapLock.RLock()
	defer vmMapLock.RUnlock()
	infoList, ok := vmInfoMap[mockName]
	if !ok {
		return nil, fmt.Errorf("no tags found for resType %s", irs.VM)
	}
	var result []*irs.TagInfo
	for _, info := range infoList {
		for _, tag := range info.TagList {
			if keyword == "" || keyword == "*" || tag.Key == keyword || tag.Value == keyword {
				result = append(result, &irs.TagInfo{
					ResType: irs.VM,
					ResIId:  info.IId,
					TagList: []irs.KeyValue{tag},
				})
			}
		}
	}
	return result, nil
}

func findTagInNLB(mockName string, keyword string) ([]*irs.TagInfo, error) {
	nlbMapLock.RLock()
	defer nlbMapLock.RUnlock()
	infoList, ok := nlbInfoMap[mockName]
	if !ok {
		return nil, fmt.Errorf("no tags found for resType %s", irs.NLB)
	}
	var result []*irs.TagInfo
	for _, info := range infoList {
		for _, tag := range info.TagList {
			if keyword == "" || keyword == "*" || tag.Key == keyword || tag.Value == keyword {
				result = append(result, &irs.TagInfo{
					ResType: irs.NLB,
					ResIId:  info.IId,
					TagList: []irs.KeyValue{tag},
				})
			}
		}
	}
	return result, nil
}

func findTagInDisk(mockName string, keyword string) ([]*irs.TagInfo, error) {
	diskMapLock.RLock()
	defer diskMapLock.RUnlock()
	infoList, ok := diskInfoMap[mockName]
	if !ok {
		return nil, fmt.Errorf("no tags found for resType %s", irs.DISK)
	}
	var result []*irs.TagInfo
	for _, info := range infoList {
		for _, tag := range info.TagList {
			if keyword == "" || keyword == "*" || tag.Key == keyword || tag.Value == keyword {
				result = append(result, &irs.TagInfo{
					ResType: irs.DISK,
					ResIId:  info.IId,
					TagList: []irs.KeyValue{tag},
				})
			}
		}
	}
	return result, nil
}

func findTagInMyImage(mockName string, keyword string) ([]*irs.TagInfo, error) {
	myImageMapLock.RLock()
	defer myImageMapLock.RUnlock()
	infoList, ok := myImageInfoMap[mockName]
	if !ok {
		return nil, fmt.Errorf("no tags found for resType %s", irs.MYIMAGE)
	}
	var result []*irs.TagInfo
	for _, info := range infoList {
		for _, tag := range info.TagList {
			if keyword == "" || keyword == "*" || tag.Key == keyword || tag.Value == keyword {
				result = append(result, &irs.TagInfo{
					ResType: irs.MYIMAGE,
					ResIId:  info.IId,
					TagList: []irs.KeyValue{tag},
				})
			}
		}
	}
	return result, nil
}

func findTagInCluster(mockName string, keyword string) ([]*irs.TagInfo, error) {
	clusterMapLock.RLock()
	defer clusterMapLock.RUnlock()
	infoList, ok := clusterInfoMap[mockName]
	if !ok {
		return nil, fmt.Errorf("no tags found for resType %s", irs.CLUSTER)
	}
	var result []*irs.TagInfo
	for _, info := range infoList {
		for _, tag := range info.TagList {
			if keyword == "" || keyword == "*" || tag.Key == keyword || tag.Value == keyword {
				result = append(result, &irs.TagInfo{
					ResType: irs.CLUSTER,
					ResIId:  info.IId,
					TagList: []irs.KeyValue{tag},
				})
			}
		}
	}
	return result, nil
}
