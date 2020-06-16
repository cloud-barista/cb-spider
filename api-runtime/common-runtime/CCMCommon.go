// Cloud Control Manager's Rest Runtime of CB-Spider.ll
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.06.

package commonruntime

import (
	"fmt"
	"sync"

	"github.com/cloud-barista/cb-store/config"
	"github.com/sirupsen/logrus"
	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	iidm "github.com/cloud-barista/cb-spider/cloud-control-manager/iid-manager"

//	"strings"
//	"strconv"

//	"time"
)


// define string of resource types
const (
        rsImage string = "image"
        rsVPC string = "vpc"  
	// rsSubnet = SUBNET:{VPC NameID} => cook in code
        rsSG string = "sg"
        rsKey string = "keypair"
        rsVM string = "vm"
)

const rsSubnetPrefix string = "subnet:"
const sgDELIMITER string = "-delimiter-"

// definition of RWLock for each Resource Ops
var imgRWLock = new(sync.RWMutex)
var vpcRWLock = new(sync.RWMutex)
var sgRWLock = new(sync.RWMutex)
var keyRWLock = new(sync.RWMutex)
var vmRWLock = new(sync.RWMutex)


// definition of IIDManager RWLock
var iidRWLock = new(iidm.IIDRWLOCK)

var cblog *logrus.Logger

func init() {
        cblog = config.Cblogger
}

type AllResourceList struct {
	AllList struct {
		MappedList []*cres.IID `json:"MappedList"`
		UserOnlyList []*cres.IID `json:"UserOnlyList"`
		CspOnlyList []*cres.IID `json:"CspOnlyList"`
	}
}


// list all Resources for management
// (1) get IID:list
// (2) get CSP:list
// (3) filtering CSP-list by IID-list
// (4) make MappedList, UserOnlyList, CspOnlyList
func ListAllResource(connectionName string, rsType string) (AllResourceList, error) {
        cblog.Info("call ListAllResource()")


        cldConn, err := ccm.GetCloudConnection(connectionName)
        if err != nil {
                return  AllResourceList{}, err
        }

	var handler interface{}

	switch rsType {
	case rsVPC:
		handler, err = cldConn.CreateVPCHandler()
	case rsSG:
		handler, err = cldConn.CreateSecurityHandler()
	case rsKey:
		handler, err = cldConn.CreateKeyPairHandler()
	case rsVM:
		handler, err = cldConn.CreateVMHandler()
	default:
		return AllResourceList{}, fmt.Errorf(rsType + " is not supported Resource!!")	
	}
        if err != nil {
                return  AllResourceList{}, err
        }

	var allResList AllResourceList


        switch rsType {
        case rsVPC:
vpcRWLock.RLock()
defer vpcRWLock.RUnlock()
        case rsSG:
sgRWLock.RLock()
defer sgRWLock.RUnlock()
        case rsKey:
keyRWLock.RLock()
defer keyRWLock.RUnlock()
        case rsVM:
vmRWLock.RLock()
defer vmRWLock.RUnlock()
        default:
                return AllResourceList{}, fmt.Errorf(rsType + " is not supported Resource!!")
        }

// (1) get IID:list
        iidInfoList, err := iidRWLock.ListIID(connectionName, rsType)
        if err != nil {
                cblog.Error(err)
                return  AllResourceList{}, err
        }

	// if iidInfoList is empty, UserOnlyList is empty.
        if iidInfoList == nil || len(iidInfoList) <= 0 {
                emptyIIDInfoList := []*cres.IID{}
                allResList.AllList.MappedList = emptyIIDInfoList
                allResList.AllList.UserOnlyList = emptyIIDInfoList
        }

// (2) get CSP:list
	iidCSPList := []*cres.IID{}
        switch rsType {
        case rsVPC:
		infoList, err := handler.(cres.VPCHandler).ListVPC()
		if err != nil {
			cblog.Error(err)
			return  AllResourceList{}, err
		}
		if infoList != nil {
			for _, info := range infoList {
				iidCSPList = append(iidCSPList, &info.IId)
			}
		}
        case rsSG:
		infoList, err := handler.(cres.SecurityHandler).ListSecurity()
		if err != nil {
			cblog.Error(err)
			return  AllResourceList{}, err
		}
		if infoList != nil {
			for _, info := range infoList {
				iidCSPList = append(iidCSPList, &info.IId)
			}
		}
        case rsKey:
		infoList, err := handler.(cres.KeyPairHandler).ListKey()
		if err != nil {
			cblog.Error(err)
			return  AllResourceList{}, err
		}
		if infoList != nil {
			for _, info := range infoList {
				iidCSPList = append(iidCSPList, &info.IId)
			}
		}
        case rsVM:
		infoList, err := handler.(cres.VMHandler).ListVM()
		if err != nil {
			cblog.Error(err)
			return  AllResourceList{}, err
		}
		if infoList != nil {
			for _, info := range infoList {
				iidCSPList = append(iidCSPList, &info.IId)
			}
		}
        default:
                return AllResourceList{}, fmt.Errorf(rsType + " is not supported Resource!!")
        }

        if iidCSPList == nil || len(iidCSPList) <= 0 { 
		// if iidCSPList is empty, iidInfoList is empty => all list is empty <-------------- (1)
		if iidInfoList == nil || len(iidInfoList) <= 0 {
			emptyIIDInfoList := []*cres.IID{}
			allResList.AllList.CspOnlyList = emptyIIDInfoList

			return allResList, nil
		} else { // iidCSPList is empty and iidInfoList has value => only UserOnlyList <--(2)
			emptyIIDInfoList := []*cres.IID{}
			allResList.AllList.MappedList = emptyIIDInfoList
			allResList.AllList.CspOnlyList = emptyIIDInfoList
			allResList.AllList.UserOnlyList = getIIDList(iidInfoList)

			return allResList, nil
		}
        }

	// iidInfoList is empty, iidCSPList has values => only CspOnlyList <--------------------------(3)
	if iidInfoList == nil || len(iidInfoList) <= 0 {
		CspOnlyList := []*cres.IID{}
                for _, iid := range iidCSPList {
			CspOnlyList = append(CspOnlyList, iid)
		}
		allResList.AllList.CspOnlyList = CspOnlyList

		return allResList, nil
	}

	////// iidInfoList has values, iidCSPList has values  <----------------------------------(4)
// (3) filtering CSP-list by IID-list
        MappedList := []*cres.IID{}
        UserOnlyList := []*cres.IID{}
        for _, iidInfo := range iidInfoList {
                exist := false
                for _, iid := range iidCSPList {
                        if iidInfo.IId.SystemId == iid.SystemId {
				MappedList = append(MappedList, &iidInfo.IId)
                                exist = true
                        }
                }
                if exist == false {
			UserOnlyList = append(UserOnlyList, &iidInfo.IId)
                }
        }

	allResList.AllList.MappedList = MappedList
	allResList.AllList.UserOnlyList = UserOnlyList

        CspOnlyList := []*cres.IID{}
	for _, iid := range iidCSPList {
		if MappedList == nil || len(MappedList) <= 0 {
			CspOnlyList = append(CspOnlyList, iid)
		} else {
			for _, mappedInfo := range MappedList {
				if iid.SystemId != mappedInfo.SystemId {
					CspOnlyList = append(CspOnlyList, iid)
				}
			}
		}
	}
	allResList.AllList.CspOnlyList = CspOnlyList


	return allResList, nil
}

func getIIDList(iidInfoList []*iidm.IIDInfo) []*cres.IID {
        iidList := []*cres.IID{}
	for _, iidInfo := range iidInfoList {
		iidList = append(iidList, &iidInfo.IId)
	}
	return iidList
}


// (1) get IID(NameId)
// (2) delete Resource(SystemId)
// (3) delete IID
func DeleteResource(connectionName string, rsType string, nameID string, force string) (bool, cres.VMStatus, error) {
        cblog.Info("call DeleteResource()")

        cldConn, err := ccm.GetCloudConnection(connectionName)
        if err != nil {
                cblog.Error(err)
                return  false, "", err
        }

        var handler interface{}

        switch rsType {
        case rsVPC:
                handler, err = cldConn.CreateVPCHandler()
        case rsSG:
                handler, err = cldConn.CreateSecurityHandler()
        case rsKey:
                handler, err = cldConn.CreateKeyPairHandler()
        case rsVM:
                handler, err = cldConn.CreateVMHandler()
        default:
                return false, "", fmt.Errorf(rsType + " is not supported Resource!!")
        }
        if err != nil {
                cblog.Error(err)
                return  false, "", err
        }

        switch rsType {
        case rsVPC:
vpcRWLock.Lock()
defer vpcRWLock.Unlock()
        case rsSG:
sgRWLock.Lock()
defer sgRWLock.Unlock()
        case rsKey:
keyRWLock.Lock()
defer keyRWLock.Unlock()
        case rsVM:
vmRWLock.Lock()
defer vmRWLock.Unlock()
        default:
                return false, "", fmt.Errorf(rsType + " is not supported Resource!!")
        }


// (1) get IID(NameId) for getting SystemId
	var iidInfo *iidm.IIDInfo
	if rsType == rsSG {
		// SG NameID format => {VPC NameID} + sgDELIMITER + {SG NameID}
		iidInfo, err = iidRWLock.FindIID(connectionName, rsType, nameID)
		if err != nil {
			cblog.Error(err)
			return false, "", err
		}
	} else {
		iidInfo, err = iidRWLock.GetIID(connectionName, rsType, cres.IID{nameID, ""})
		if err != nil {
			cblog.Error(err)
			return false, "", err
		}
	}


// (2) delete Resource(SystemId)
	result := false
	var vmStatus cres.VMStatus
        switch rsType {
        case rsVPC:
		result, err = handler.(cres.VPCHandler).DeleteVPC(iidInfo.IId)
                if err != nil {
                        cblog.Error(err)
			if force != "true" {
				return  false, "", err
			}
                }
        case rsSG:
		result, err = handler.(cres.SecurityHandler).DeleteSecurity(iidInfo.IId)
                if err != nil {
                        cblog.Error(err)
			if force != "true" {
				return  false, "", err
			}
                }
        case rsKey:
		result, err = handler.(cres.KeyPairHandler).DeleteKey(iidInfo.IId)
                if err != nil {
                        cblog.Error(err)
			if force != "true" {
				return  false, "", err
			}
                }
        case rsVM:
		vmStatus, err = handler.(cres.VMHandler).TerminateVM(iidInfo.IId)
                if err != nil {
                        cblog.Error(err)
			if force != "true" {
				return  false, vmStatus, err
			}
                }
        default:
                return false, "", fmt.Errorf(rsType + " is not supported Resource!!")
        }

	if force != "true" {
		if rsType != rsVM {
			if result == false {
				return result, "", nil
			}
		}
	}


// (3) delete IID
        _, err = iidRWLock.DeleteIID(connectionName, rsType, iidInfo.IId)
        if err != nil {
                cblog.Error(err)
		if force != "true" {
			return false, "", err
		}
        }

        // if VPC
	if rsType == rsVPC {
		// for Subnet list
		// key-value structure: /{ConnectionName}/rsSubnetPrefix+{VPC-NameId}/{Subnet-IId}
		subnetInfoList, err2 := iidRWLock.ListIID(connectionName, rsSubnetPrefix + iidInfo.IId.NameId)
		if err2 != nil {
			cblog.Error(err)
			if force != "true" {
				return false, "", err
			}
		}
		for _, subnetInfo := range subnetInfoList {
			// key-value structure: /{ConnectionName}/rsSubnetPrefix+{VPC-NameId}/{Subnet-IId}
			_, err := iidRWLock.DeleteIID(connectionName, rsSubnetPrefix + iidInfo.IId.NameId, subnetInfo.IId)
			if err != nil {
				cblog.Error(err)
				if force != "true" {
					return false, "", err
				}
			}
		}
	}

	if rsType == rsVM {
		return result, vmStatus, nil
	}else {
		return result, "", nil
	}
}

// (1) delete Resource(SystemId)
func DeleteCSPResource(connectionName string, rsType string, systemID string) (bool, cres.VMStatus, error) {
        cblog.Info("call DeleteResource()")

        cldConn, err := ccm.GetCloudConnection(connectionName)
        if err != nil {
                cblog.Error(err)
                return  false, "", err
        }

        var handler interface{}

        switch rsType {
        case rsVPC:
                handler, err = cldConn.CreateVPCHandler()
        case rsSG:
                handler, err = cldConn.CreateSecurityHandler()
        case rsKey:
                handler, err = cldConn.CreateKeyPairHandler()
        case rsVM:
                handler, err = cldConn.CreateVMHandler()
        default:
                return false, "", fmt.Errorf(rsType + " is not supported Resource!!")
        }
        if err != nil {
                cblog.Error(err)
                return  false, "", err
        }


	iid := cres.IID{"", systemID}

// (1) delete Resource(SystemId)
        result := false
        var vmStatus cres.VMStatus
        switch rsType {
        case rsVPC:
                result, err = handler.(cres.VPCHandler).DeleteVPC(iid)
                if err != nil {
                        cblog.Error(err)
			return  false, "", err
                }
        case rsSG:
                result, err = handler.(cres.SecurityHandler).DeleteSecurity(iid)
                if err != nil {
                        cblog.Error(err)
			return  false, "", err
                }
        case rsKey:
                result, err = handler.(cres.KeyPairHandler).DeleteKey(iid)
                if err != nil {
                        cblog.Error(err)
			return  false, "", err
                }
        case rsVM:
                vmStatus, err = handler.(cres.VMHandler).TerminateVM(iid)
                if err != nil {
                        cblog.Error(err)
			return  false, vmStatus, err
                }
        default:
                return false, "", fmt.Errorf(rsType + " is not supported Resource!!")
        }

	if rsType != rsVM {
		if result == false {
			return result, "", nil
		}
	}


        if rsType == rsVM {
                return result, vmStatus, nil
        }else {
                return result, "", nil
        }
}

