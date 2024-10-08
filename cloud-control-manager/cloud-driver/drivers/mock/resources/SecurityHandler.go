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
)

var securityInfoMap map[string][]*irs.SecurityInfo

type MockSecurityHandler struct {
	MockName string
}

func init() {
	// cblog is a global variable.
	securityInfoMap = make(map[string][]*irs.SecurityInfo)
}

var sgMapLock = new(sync.RWMutex)

// (1) create securityInfo object
// (2) insert securityInfo into global Map
func (securityHandler *MockSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called CreateSecurity()!")

	mockName := securityHandler.MockName
	securityReqInfo.IId.SystemId = securityReqInfo.IId.NameId
	securityReqInfo.VpcIID.SystemId = securityReqInfo.VpcIID.NameId
	// (1) create securityInfo object
	securityInfo := irs.SecurityInfo{
		IId:           securityReqInfo.IId,
		VpcIID:        securityReqInfo.VpcIID,
		SecurityRules: securityReqInfo.SecurityRules,
		TagList:       securityReqInfo.TagList,
		KeyValueList:  nil,
	}

	// (2) insert SecurityInfo into global Map
	sgMapLock.Lock()
	defer sgMapLock.Unlock()
	infoList, _ := securityInfoMap[mockName]
	infoList = append(infoList, &securityInfo)
	securityInfoMap[mockName] = infoList

	return CloneSecurityInfo(securityInfo), nil
}

func CloneSecurityInfoList(srcInfoList []*irs.SecurityInfo) []*irs.SecurityInfo {
	clonedInfoList := []*irs.SecurityInfo{}
	for _, srcInfo := range srcInfoList {
		clonedInfo := CloneSecurityInfo(*srcInfo)
		clonedInfoList = append(clonedInfoList, &clonedInfo)
	}
	return clonedInfoList
}

func CloneSecurityInfo(srcInfo irs.SecurityInfo) irs.SecurityInfo {
	/*
		type SecurityInfo struct {
			IId IID // {NameId, SystemId}
			VpcIID        IID    // {NameId, SystemId}
			Direction     string // @todo userd??
			SecurityRules *[]SecurityRuleInfo
			KeyValueList []KeyValue
		}
	*/

	// clone SecurityInfo
	clonedInfo := irs.SecurityInfo{
		IId:           irs.IID{srcInfo.IId.NameId, srcInfo.IId.SystemId},
		VpcIID:        irs.IID{srcInfo.VpcIID.NameId, srcInfo.VpcIID.SystemId},
		SecurityRules: srcInfo.SecurityRules,
		TagList:       srcInfo.TagList, // clone TagList
		KeyValueList:  srcInfo.KeyValueList,
	}

	return clonedInfo
}

func (securityHandler *MockSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called ListSecurity()!")

	mockName := securityHandler.MockName
	sgMapLock.RLock()
	defer sgMapLock.RUnlock()
	infoList, ok := securityInfoMap[mockName]
	if !ok {
		return []*irs.SecurityInfo{}, nil
	}

	return CloneSecurityInfoList(infoList), nil
}

func (securityHandler *MockSecurityHandler) GetSecurity(iid irs.IID) (irs.SecurityInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called GetSecurity()!")

	sgMapLock.RLock()
	defer sgMapLock.RUnlock()

	mockName := securityHandler.MockName
	infoList, ok := securityInfoMap[mockName]
	if !ok {
		return irs.SecurityInfo{}, fmt.Errorf("%s SecurityGroup does not exist!!", iid.NameId)
	}

	// infoList is already cloned in ListSecurity()
	for _, info := range infoList {
		if info.IId.NameId == iid.NameId {
			return CloneSecurityInfo(*info), nil
		}
	}

	return irs.SecurityInfo{}, fmt.Errorf("%s SecurityGroup does not exist!!", iid.NameId)
}

func (securityHandler *MockSecurityHandler) DeleteSecurity(iid irs.IID) (bool, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called DeleteSecurity()!")

	sgMapLock.Lock()
	defer sgMapLock.Unlock()

	mockName := securityHandler.MockName
	infoList, ok := securityInfoMap[mockName]
	if !ok {
		return false, fmt.Errorf("%s SecurityGroup does not exist!!", iid.NameId)
	}

	for idx, info := range infoList {
		if info.IId.SystemId == iid.SystemId {
			infoList = append(infoList[:idx], infoList[idx+1:]...)
			securityInfoMap[mockName] = infoList
			return true, nil
		}
	}
	return false, nil
}

func (securityHandler *MockSecurityHandler) AddRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (irs.SecurityInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called AddRules()!")

	sgMapLock.Lock()
	defer sgMapLock.Unlock()

	mockName := securityHandler.MockName
	infoList, ok := securityInfoMap[mockName]
	if !ok {
		return irs.SecurityInfo{}, fmt.Errorf("%s SecurityGroup does not exist!!", sgIID.NameId)
	}

	// check if all input rules exist
	for _, info := range infoList {
		if info.IId.NameId == sgIID.NameId {
			for _, reqRuleInfo := range *securityRules {
				for _, ruleInfo := range *info.SecurityRules {
					if isEqualRule(&ruleInfo, &reqRuleInfo) {
						errMSG := fmt.Sprintf("%s SecurityGroup already has this rule: %v!!", sgIID.NameId, reqRuleInfo)
						errMSG += fmt.Sprintf(" #### %s SecurityGroup has %v!!", sgIID.NameId, *info.SecurityRules)
						return irs.SecurityInfo{}, fmt.Errorf(errMSG)
					}
				}
			}
		}
	}

	// Add all rules
	for _, info := range infoList {
		if info.IId.NameId == sgIID.NameId {
			*info.SecurityRules = append(*info.SecurityRules, *securityRules...)
			return CloneSecurityInfo(*info), nil
		}
	}

	return irs.SecurityInfo{}, fmt.Errorf("%s SecurityGroup does not exist!!", sgIID.NameId)
}

func (securityHandler *MockSecurityHandler) RemoveRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (bool, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called RemoveRules()!")

	sgMapLock.Lock()
	defer sgMapLock.Unlock()

	mockName := securityHandler.MockName
	infoList, ok := securityInfoMap[mockName]
	if !ok {
		return false, fmt.Errorf("%s SecurityGroup does not exist!!", sgIID.NameId)
	}

	// check if all input rules do not exist
	for _, info := range infoList {
		if info.IId.NameId == sgIID.NameId {
			for _, reqRuleInfo := range *securityRules {
				existFlag := false
				for _, ruleInfo := range *info.SecurityRules {
					if isEqualRule(&ruleInfo, &reqRuleInfo) {
						existFlag = true
					}
				}
				if !existFlag {
					errMSG := fmt.Sprintf("%s SecurityGroup does not have this rule: %v!!", sgIID.NameId, reqRuleInfo)
					errMSG += fmt.Sprintf(" #### %s SecurityGroup has %v!!", sgIID.NameId, *info.SecurityRules)
					return false, fmt.Errorf(errMSG)
				}
			}
		}
	}

	for _, info := range infoList {
		if info.IId.NameId == sgIID.NameId {
			for idx := len(*info.SecurityRules) - 1; idx >= 0; idx-- {
				ruleInfo := (*info.SecurityRules)[idx]
				for _, reqRuleInfo := range *securityRules {
					if isEqualRule(&ruleInfo, &reqRuleInfo) {
						*info.SecurityRules = removeRule(info.SecurityRules, idx)
					}
				}
			}
		}
	}

	return true, nil
}

func isEqualRule(a *irs.SecurityRuleInfo, b *irs.SecurityRuleInfo) bool {
	/*------------------------------
	type SecurityRuleInfo struct {
		Direction  string
		IPProtocol string
		FromPort   string
		ToPort     string
		CIDR       string
	}
	-------------------------------*/

	if a.Direction != b.Direction {
		return false
	}
	if a.IPProtocol != b.IPProtocol {
		return false
	}
	if a.FromPort != b.FromPort {
		return false
	}
	if a.ToPort != b.ToPort {
		return false
	}
	if a.CIDR != b.CIDR {
		return false
	}

	return true
}

func removeRule(list *[]irs.SecurityRuleInfo, idx int) []irs.SecurityRuleInfo {
	return append((*list)[:idx], (*list)[idx+1:]...)
}

func (securityHandler *MockSecurityHandler) ListIID() ([]*irs.IID, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called ListIID()!")

	mockName := securityHandler.MockName
	sgMapLock.RLock()
	defer sgMapLock.RUnlock()
	infoList, ok := securityInfoMap[mockName]
	if !ok {
		return []*irs.IID{}, nil
	}

	iidList := make([]*irs.IID, len(infoList))
	for i, info := range infoList {
		iidList[i] = &info.IId
	}

	return iidList, nil
}
