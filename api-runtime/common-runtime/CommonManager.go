// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.06.

package commonruntime

import (
	"fmt"
	"strings"

	"github.com/cloud-barista/cb-spider/api-runtime/common-runtime/sp-lock"
	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	iidm "github.com/cloud-barista/cb-spider/cloud-control-manager/iid-manager"
	"github.com/cloud-barista/cb-store/config"
	"github.com/sirupsen/logrus"
	"encoding/json"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
)

// define string of resource types
const (
	rsImage string = "image"
	rsVPC   string = "vpc"
	rsSubnet string = "subnet"	
	rsSG  string = "sg"
	rsKey string = "keypair"
	rsVM  string = "vm"
	rsNLB  string = "nlb"
	rsDisk  string = "disk"
	rsMyImage  string = "myimage"
	rsCluster  string = "cluster"
	rsNodeGroup  string = "nodegroup"
)

func RsTypeString(rsType string) string {
	switch rsType {
	case rsImage:
		return "VM Image"
	case rsVPC:
		return "VPC"
	case rsSubnet:
		return "Subnet"
	case rsSG:
		return "Security Group"
	case rsKey:
		return "VM KeyPair"
	case rsVM:
		return "VM"
	case rsNLB:
		return "nlb"
	case rsDisk:
		return "disk"
	case rsMyImage:
		return "MyImage"
	case rsCluster:
		return "Cluster"
	case rsNodeGroup:
		return "NodeGroup"
        default:
                return rsType + " is not supported Resource!!"

	}
}

// definition of SPLock for each Resource Ops
var imgSPLock = splock.New()
var vpcSPLock = splock.New()
var sgSPLock = splock.New()
var keySPLock = splock.New()
var vmSPLock = splock.New()
var nlbSPLock = splock.New()
var diskSPLock = splock.New()
var myImageSPLock = splock.New()
var clusterSPLock = splock.New()

// definition of IIDManager RWLock
var iidRWLock = new(iidm.IIDRWLOCK)

var cblog *logrus.Logger
var callogger *logrus.Logger

func init() {
	cblog = config.Cblogger
	// logger for HisCall
        callogger = call.GetLogger("HISCALL")
}

type AllResourceList struct {
	AllList struct {
		MappedList     []*cres.IID `json:"MappedList"`
		OnlySpiderList []*cres.IID `json:"OnlySpiderList"`
		OnlyCSPList    []*cres.IID `json:"OnlyCSPList"`
	}
}

func GetAllSPLockInfo() []string {
	var results []string

	results = append(results, vpcSPLock.GetSPLockMapStatus("VPC SPLock"))
	results = append(results, sgSPLock.GetSPLockMapStatus("SG SPLock"))
	results = append(results, keySPLock.GetSPLockMapStatus("Key SPLock"))
	results = append(results, vmSPLock.GetSPLockMapStatus("VM SPLock"))

	return results
}

func getMSShortID(inID string) string {
	// /subscriptions/a20fed83~/Microsoft.Network/~/sg01-c5n27e2ba5ofr0fnbck0
        // ==> sg01-c5n27e2ba5ofr0fnbck0
	var shortID string
        if strings.Contains(inID, "/Microsoft.") {
                strList := strings.Split(inID, "/")
                shortID = strList[len(strList)-1]
        } else {
                return inID
        }
	return shortID
}

func checkNotFoundError(err error) bool {
	msg := err.Error()
	msg = strings.ReplaceAll(msg, " ", "")
	msg = strings.ToLower(msg)

	return strings.Contains(msg, "notfound") || strings.Contains(msg, "notexist") || strings.Contains(msg, "failedtofind") || strings.Contains(msg, "failedtogetthevm")
}



func getUserIIDList(iidInfoList []*iidm.IIDInfo) []*cres.IID {
	iidList := []*cres.IID{}
	for _, iidInfo := range iidInfoList {
		userIId := getUserIID(iidInfo.IId)
		iidList = append(iidList, &userIId)
	}
	return iidList
}

// Get driverSystemId from SpiderIID
func getDriverSystemId(spiderIId cres.IID) string {
	// if AWS NLB's SystmeId, 
	//     ex) arn:aws:elasticloadbalancing:us-east-2:635484366616:loadbalancer/net/spider-nl-cangp8aba5o2pi8oa7o0/1dee7370037afd6d
	strArray := strings.Split(spiderIId.SystemId, ":")
	systemId := strings.ReplaceAll(spiderIId.SystemId, strArray[0]+":", "")
	return systemId
}

// Get driverIID from SpiderIID
func getDriverIID(spiderIId cres.IID) cres.IID {
	// if AWS NLB's SystmeId, 
	//     ex) arn:aws:elasticloadbalancing:us-east-2:635484366616:loadbalancer/net/spider-nl-cangp8aba5o2pi8oa7o0/1dee7370037afd6d
	strArray := strings.Split(spiderIId.SystemId, ":")
	systemId := strings.ReplaceAll(spiderIId.SystemId, strArray[0]+":", "")
	driverIId := cres.IID{strArray[0], systemId}
	return driverIId
}

// Get userIID from SpiderIID
func getUserIID(spiderIId cres.IID) cres.IID {
	// if AWS NLB's SystmeId, 
	//     ex) arn:aws:elasticloadbalancing:us-east-2:635484366616:loadbalancer/net/spider-nl-cangp8aba5o2pi8oa7o0/1dee7370037afd6d
	strArray := strings.Split(spiderIId.SystemId, ":")
	userIId := cres.IID{spiderIId.NameId, strings.ReplaceAll(spiderIId.SystemId, strArray[0]+":", "")}
	return userIId
}

func  findUserIID(iidInfoList []*iidm.IIDInfo, systemId string) cres.IID {
        for _, iidInfo := range iidInfoList {
                if getDriverSystemId(iidInfo.IId) == systemId {
                        return getUserIID(iidInfo.IId)
                }
        }
        return cres.IID{}
}


// Get All IID:list of SecurityGroup
// (1) Get VPC's Name List
// (2) Create All SG's IIDInfo List
func getAllSGIIDInfoList(connectionName string) ([]*iidm.IIDInfo, error) {

        // (1) Get VPC's Name List
        // format) /resource-info-spaces/{iidGroup}/{connectionName}/{resourceType}/{resourceName} [{resourceID}]
        vpcNameList, err := iidRWLock.ListResourceType(iidm.SGGROUP, connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
	vpcNameList = uniqueNameList(vpcNameList)
        // (2) Create All SG's IIDInfo List
        iidInfoList := []*iidm.IIDInfo{}
        for _, vpcName := range vpcNameList {
                iidInfoListForOneVPC, err := iidRWLock.ListIID(iidm.SGGROUP, connectionName, vpcName)
                if err != nil {
                        cblog.Error(err)
                        return nil, err
                }
                iidInfoList = append(iidInfoList, iidInfoListForOneVPC...)
        }
        return iidInfoList, nil
}

func uniqueNameList(vpcNameList []string) []string {
    keys := make(map[string]bool)
    list := []string{}	
    for _, entry := range vpcNameList {
        if _, value := keys[entry]; !value {
            keys[entry] = true
            list = append(list, entry)
        }
    }    
    return list
}

//======================== Common Handling

// UnregisterResource API does not delete the real resource.
// This API just unregister the resource from Spider.
// (1) check exist(NameID)
// (2) delete SpiderIID
func UnregisterResource(connectionName string, rsType string, nameId string) (bool, error) {
        cblog.Info("call UnregisterResource()")

	// check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
		cblog.Error(err)
                return false, err
        }

        nameId, err = EmptyCheckAndTrim("nameId", nameId)
        if err != nil {
		cblog.Error(err)
                return false, err
        }

	switch rsType {
        case rsVPC:
                vpcSPLock.Lock(connectionName, nameId)
                defer vpcSPLock.Unlock(connectionName, nameId)
        case rsSG:
                sgSPLock.Lock(connectionName, nameId)
                defer sgSPLock.Unlock(connectionName, nameId)
        case rsKey:
                keySPLock.Lock(connectionName, nameId)
                defer keySPLock.Unlock(connectionName, nameId)
        case rsVM:
                vmSPLock.Lock(connectionName, nameId)
                defer vmSPLock.Unlock(connectionName, nameId)
        case rsNLB:
                nlbSPLock.Lock(connectionName, nameId)
                defer nlbSPLock.Unlock(connectionName, nameId)
        case rsDisk:
                diskSPLock.Lock(connectionName, nameId)
                defer diskSPLock.Unlock(connectionName, nameId)
        case rsMyImage:
                myImageSPLock.Lock(connectionName, nameId)
                defer myImageSPLock.Unlock(connectionName, nameId)
        case rsCluster:
                clusterSPLock.Lock(connectionName, nameId)
                defer clusterSPLock.Unlock(connectionName, nameId)
        default:
                return false, fmt.Errorf(rsType + " is not supported Resource!!")
        }


        // (1) check existence(UserID)
	var isExist bool=false
	var vpcName string 
	switch rsType {
        case rsSG:
		iidInfoList, err := getAllSGIIDInfoList(connectionName)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		for _, OneIIdInfo := range iidInfoList {
			if OneIIdInfo.IId.NameId == nameId {
				vpcName = OneIIdInfo.ResourceType/*vpcName*/  // ---------- Don't forget
				isExist = true
				break
			}
		}
        case rsNLB:
                iidInfoList, err := getAllNLBIIDInfoList(connectionName)
                if err != nil {
                        cblog.Error(err)
                        return false, err
                }
                for _, OneIIdInfo := range iidInfoList {
                        if OneIIdInfo.IId.NameId == nameId {
                                vpcName = OneIIdInfo.ResourceType/*vpcName*/  // ---------- Don't forget
                                isExist = true
                                break
                        }
                }
        case rsCluster:
                iidInfoList, err := getAllClusterIIDInfoList(connectionName)
                if err != nil {
                        cblog.Error(err)
                        return false, err
                }
                for _, OneIIdInfo := range iidInfoList {
                        if OneIIdInfo.IId.NameId == nameId {
                                vpcName = OneIIdInfo.ResourceType/*vpcName*/  // ---------- Don't forget
                                isExist = true
                                break
                        }
                }

	default:
		// (1) check exist(NameID)
		var err error
		isExist, err = iidRWLock.IsExistIID(iidm.IIDSGROUP, connectionName, rsType, cres.IID{nameId, ""})
		if err != nil {
			cblog.Error(err)
			return false, err
		}
	} // end of switch

	if isExist == false {
		return false, fmt.Errorf("The %s '%s' does not exist!", RsTypeString(rsType), nameId)
	}

	// (2) delete the IID from Metadb
	switch rsType {
        case rsVPC:
		// if vpc, delete all subnet meta data
                // (a) for vPC
		_, err := iidRWLock.DeleteIID(iidm.IIDSGROUP, connectionName, rsType, cres.IID{nameId, ""})
                if err != nil {
                        cblog.Error(err)
			return false, err
                }

                // (b) for Subnet list
                // key-value structure: ~/{SUBNETGROUP}/{ConnectionName}/{VPC-NameId}/{Subnet-reqNameId} [subnet-driverNameId:subnet-driverSystemId]  # VPC NameId => rsType
                subnetIIdInfoList, err2 := iidRWLock.ListIID(iidm.SUBNETGROUP, connectionName, nameId/*vpcName*/)
                if err2 != nil {
                        cblog.Error(err)
			return false, err
                }
                for _, subnetIIdInfo := range subnetIIdInfoList {
                        // key-value structure: ~/{SUBNETGROUP}/{ConnectionName}/{VPC-NameId}/{Subnet-reqNameId} [subnet-driverNameId:subnet-driverSystemId]  # VPC NameId => rsType
                        _, err := iidRWLock.DeleteIID(iidm.SUBNETGROUP, connectionName, nameId/*vpcName*/, subnetIIdInfo.IId)
                        if err != nil {
                                cblog.Error(err)
				return false, err
                        }
                }

                // @todo Should we also delete the SG list of this VPC ?


        case rsSG:
		_, err := iidRWLock.DeleteIID(iidm.SGGROUP, connectionName, vpcName/*rsType*/, cres.IID{nameId, ""})
		if err != nil {
			cblog.Error(err)
			return false, err
		}

        case rsNLB:
                _, err := iidRWLock.DeleteIID(iidm.NLBGROUP, connectionName, vpcName/*rsType*/, cres.IID{nameId, ""})
                if err != nil {
                        cblog.Error(err)
                        return false, err
                }

        case rsCluster:
                _, err := iidRWLock.DeleteIID(iidm.CLUSTERGROUP, connectionName, vpcName/*rsType*/, cres.IID{nameId, ""})
                if err != nil {
                        cblog.Error(err)
                        return false, err
                }

	default: // other resources(key, vm, ...)
		_, err := iidRWLock.DeleteIID(iidm.IIDSGROUP, connectionName, rsType, cres.IID{nameId, ""})
		if err != nil {
			cblog.Error(err)
			return false, err
		}
	} // end of switch

	return true, nil
}


// list all Resources for management
// (1) get IID:list
// (2) get CSP:list
// (3) filtering CSP-list by IID-list
// (4) make MappedList, OnlySpiderList, OnlyCSPList
func ListAllResource(connectionName string, rsType string) (AllResourceList, error) {
	cblog.Info("call ListAllResource()")

	// check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
                return AllResourceList{}, err
		cblog.Error(err)
        }

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		return AllResourceList{}, err
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
	case rsNLB:
		handler, err = cldConn.CreateNLBHandler()
	case rsDisk:
		handler, err = cldConn.CreateDiskHandler()
	case rsMyImage:
		handler, err = cldConn.CreateMyImageHandler()
	case rsCluster:
		handler, err = cldConn.CreateClusterHandler()		
	default:
		return AllResourceList{}, fmt.Errorf(rsType + " is not supported Resource!!")
	}
	if err != nil {
		return AllResourceList{}, err
	}

	var allResList AllResourceList

	// (1) get IID:list
	iidInfoList := []*iidm.IIDInfo{}
	switch rsType {
	case rsSG:
		iidInfoList, err = getAllSGIIDInfoList(connectionName)
		if err != nil {
			cblog.Error(err)
			return AllResourceList{}, err
		}
        case rsNLB:
                iidInfoList, err = getAllNLBIIDInfoList(connectionName)
                if err != nil {
                        cblog.Error(err)
                        return AllResourceList{}, err
                }
        case rsCluster:
                iidInfoList, err = getAllClusterIIDInfoList(connectionName)
                if err != nil {
                        cblog.Error(err)
                        return AllResourceList{}, err
                }

	default:
		iidInfoList, err = iidRWLock.ListIID(iidm.IIDSGROUP, connectionName, rsType)
		if err != nil {
			cblog.Error(err)
			return AllResourceList{}, err
		}
	}

	// if iidInfoList is empty, OnlySpiderList is empty.
	if iidInfoList == nil || len(iidInfoList) <= 0 {
		emptyIIDInfoList := []*cres.IID{}
		allResList.AllList.MappedList = emptyIIDInfoList
		allResList.AllList.OnlySpiderList = emptyIIDInfoList
	}

	// (2) get CSP:list
	iidCSPList := []*cres.IID{}
	switch rsType {
	case rsVPC:
		infoList, err := handler.(cres.VPCHandler).ListVPC()
		if err != nil {
			cblog.Error(err)
			return AllResourceList{}, err
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
			return AllResourceList{}, err
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
			return AllResourceList{}, err
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
			return AllResourceList{}, err
		}
		if infoList != nil {
			for _, info := range infoList {
				iidCSPList = append(iidCSPList, &info.IId)
			}
		}
        case rsNLB:
                infoList, err := handler.(cres.NLBHandler).ListNLB()
                if err != nil {
                        cblog.Error(err)
                        return AllResourceList{}, err
                }
                if infoList != nil {
                        for _, info := range infoList {
                                iidCSPList = append(iidCSPList, &info.IId)
                        }
                }
        case rsDisk:
                infoList, err := handler.(cres.DiskHandler).ListDisk()
                if err != nil {
                        cblog.Error(err)
                        return AllResourceList{}, err
                }
                if infoList != nil {
                        for _, info := range infoList {
                                iidCSPList = append(iidCSPList, &info.IId)
                        }
                }
        case rsMyImage:
                infoList, err := handler.(cres.MyImageHandler).ListMyImage()
                if err != nil {
                        cblog.Error(err)
                        return AllResourceList{}, err
                }
                if infoList != nil {
                        for _, info := range infoList {
                                iidCSPList = append(iidCSPList, &info.IId)
                        }
                }
        case rsCluster:
                infoList, err := handler.(cres.ClusterHandler).ListCluster()
                if err != nil {
                        cblog.Error(err)
                        return AllResourceList{}, err
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
			allResList.AllList.OnlyCSPList = emptyIIDInfoList

			return allResList, nil
		} else { // iidCSPList is empty and iidInfoList has value => only OnlySpiderList <--(2)
			emptyIIDInfoList := []*cres.IID{}
			allResList.AllList.MappedList = emptyIIDInfoList
			allResList.AllList.OnlyCSPList = emptyIIDInfoList
			allResList.AllList.OnlySpiderList = getUserIIDList(iidInfoList)

			return allResList, nil
		}
	}

	// iidInfoList is empty, iidCSPList has values => only OnlyCSPList <--------------------------(3)
	if iidInfoList == nil || len(iidInfoList) <= 0 {
		OnlyCSPList := []*cres.IID{}
		for _, iid := range iidCSPList {
			OnlyCSPList = append(OnlyCSPList, iid)
		}
		allResList.AllList.OnlyCSPList = OnlyCSPList

		return allResList, nil
	}

	////// iidInfoList has values, iidCSPList has values  <----------------------------------(4)
	// (3) filtering CSP-list by IID-list
	MappedList := []*cres.IID{}
	OnlySpiderList := []*cres.IID{}
	for _, iidInfo := range iidInfoList {
		exist := false
		for _, iid := range iidCSPList {
			userIId := getUserIID(iidInfo.IId)
			if userIId.SystemId == iid.SystemId {
				MappedList = append(MappedList, &userIId)
				exist = true
			}
		}
		if exist == false {
			userIId := getUserIID(iidInfo.IId)
			OnlySpiderList = append(OnlySpiderList, &userIId)
		}
	}

	allResList.AllList.MappedList = MappedList
	allResList.AllList.OnlySpiderList = OnlySpiderList

	OnlyCSPList := []*cres.IID{}
	for _, iid := range iidCSPList {
		if MappedList == nil || len(MappedList) <= 0 {
			//userIId := getUserIID(*iid)
			OnlyCSPList = append(OnlyCSPList, iid)
		} else {
			isMapped := false
			for _, mappedInfo := range MappedList {
				if iid.SystemId == mappedInfo.SystemId {
					isMapped = true
				}
			}
			if isMapped == false {
				// userIId := getUserIID(*iid)
				OnlyCSPList = append(OnlyCSPList, iid)
			}
		}
	}
	allResList.AllList.OnlyCSPList = OnlyCSPList

	return allResList, nil
}

// (1) get spiderIID
// (2) delete Resource(SystemId)
// (3) delete IID
func DeleteResource(connectionName string, rsType string, nameID string, force string) (bool, cres.VMStatus, error) {
	cblog.Info("call DeleteResource()")

	// check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
		cblog.Error(err)
                return false, "", err
        }

        nameID, err = EmptyCheckAndTrim("nameID", nameID)
        if err != nil {
		cblog.Error(err)
                return false, "", err
        }

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return false, "", err
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
	case rsNLB:
		handler, err = cldConn.CreateNLBHandler()
	case rsDisk:
		handler, err = cldConn.CreateDiskHandler()
	case rsMyImage:
		handler, err = cldConn.CreateMyImageHandler()
	case rsCluster:
		handler, err = cldConn.CreateClusterHandler()
	default:
		err := fmt.Errorf(rsType + " is not supported Resource!!")
		return false, "", err
	}
	if err != nil {
		cblog.Error(err)
		return false, "", err
	}

	switch rsType {
	case rsVPC:
		vpcSPLock.Lock(connectionName, nameID)
		defer vpcSPLock.Unlock(connectionName, nameID)
	case rsSG:
		sgSPLock.Lock(connectionName, nameID)
		defer sgSPLock.Unlock(connectionName, nameID)
	case rsKey:
		keySPLock.Lock(connectionName, nameID)
		defer keySPLock.Unlock(connectionName, nameID)
	case rsVM:
		vmSPLock.Lock(connectionName, nameID)
		defer vmSPLock.Unlock(connectionName, nameID)
	case rsNLB:
		nlbSPLock.Lock(connectionName, nameID)
		defer nlbSPLock.Unlock(connectionName, nameID)
	case rsDisk:
		diskSPLock.Lock(connectionName, nameID)
		defer diskSPLock.Unlock(connectionName, nameID)
	case rsMyImage:
		myImageSPLock.Lock(connectionName, nameID)
		defer myImageSPLock.Unlock(connectionName, nameID)
	case rsCluster:
		clusterSPLock.Lock(connectionName, nameID)
		defer clusterSPLock.Unlock(connectionName, nameID)

	default:
		err := fmt.Errorf(rsType + " is not supported Resource!!")
		return false, "", err
	}

	// (1) get spiderIID for creating driverIID
	var iidInfo *iidm.IIDInfo
	switch rsType {
	case rsSG:
		iidInfoList, err := getAllSGIIDInfoList(connectionName)
		if err != nil {
			cblog.Error(err)
			return false, "", err
		}
		var bool_ret = false
		for _, OneIIdInfo := range iidInfoList {
			if OneIIdInfo.IId.NameId == nameID {
				iidInfo = OneIIdInfo
				bool_ret = true
				break;
			}
		}
		if bool_ret == false {
			err := fmt.Errorf("[" + connectionName + ":" + RsTypeString(rsType) +  ":" + nameID + "] does not exist!")
			cblog.Error(err)
                return false, "", err
		}

        case rsNLB:
                iidInfoList, err := getAllNLBIIDInfoList(connectionName)
                if err != nil {
                        cblog.Error(err)
                        return false, "", err
                }
                var bool_ret = false
                for _, OneIIdInfo := range iidInfoList {
                        if OneIIdInfo.IId.NameId == nameID {
                                iidInfo = OneIIdInfo
                                bool_ret = true
                                break;
                        }
                }
                if bool_ret == false {
			err := fmt.Errorf("[" + connectionName + ":" + RsTypeString(rsType) +  ":" + nameID + "] does not exist!")
			cblog.Error(err)
                return false, "", err
                }

        case rsCluster:
                iidInfoList, err := getAllClusterIIDInfoList(connectionName)
                if err != nil {
                        cblog.Error(err)
                        return false, "", err
                }
                var bool_ret = false
                for _, OneIIdInfo := range iidInfoList {
                        if OneIIdInfo.IId.NameId == nameID {
                                iidInfo = OneIIdInfo
                                bool_ret = true
                                break;
                        }
                }
                if bool_ret == false {
			err := fmt.Errorf("[" + connectionName + ":" + RsTypeString(rsType) +  ":" + nameID + "] does not exist!")
			cblog.Error(err)
                return false, "", err
                }

	default:
		iidInfo, err = iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsType, cres.IID{nameID, ""})
		if err != nil {
			cblog.Error(err)
			return false, "", err
		}
	}

	// (2) delete Resource(SystemId)
	driverIId := getDriverIID(iidInfo.IId)
	result := false
	var vmStatus cres.VMStatus
	switch rsType {
	case rsVPC:
		result, err = handler.(cres.VPCHandler).DeleteVPC(driverIId)
		if err != nil {
			cblog.Error(err)
			if force != "true" {
				return false, "", err
			}
		}
	case rsSG:		
		result, err = handler.(cres.SecurityHandler).DeleteSecurity(driverIId)
		if err != nil {
			cblog.Error(err)
			if force != "true" {
				return false, "", err
			}
		}
	case rsKey:
		result, err = handler.(cres.KeyPairHandler).DeleteKey(driverIId)
		if err != nil {
			cblog.Error(err)
			if force != "true" {
				return false, "", err
			}
		}
	case rsVM:
		providerName, err := ccm.GetProviderNameByConnectionName(connectionName)
		if err != nil {
			cblog.Error(err)
			return false, "", err
		}

		regionName, zoneName, err := ccm.GetRegionNameByConnectionName(connectionName)
		if err != nil {
			cblog.Error(err)
			return false, "", err
		}

		callInfo := call.CLOUDLOGSCHEMA {
			CloudOS: call.CLOUD_OS(providerName),
			RegionZone: regionName + "/" + zoneName,
			ResourceType: call.VM,
			ResourceName: iidInfo.IId.NameId,
			CloudOSAPI: "CB-Spider:TerminateVM()",
			ElapsedTime: "",
			ErrorMSG: "",
		}
		start := call.Start()
		vmStatus, err = handler.(cres.VMHandler).TerminateVM(driverIId)
                if err != nil {
                        cblog.Error(err)
                        if force != "true" {
				callInfo.ErrorMSG = err.Error()
				callogger.Info(call.String(callInfo))
                                return false, vmStatus, err
                        }else {
				break
			}
                }

		if vmStatus == cres.Terminated {
			break
		}

		// Check Sync Called
		waiter := NewWaiter(5, 240) // (sleep, timeout)

		for {
			status, err := handler.(cres.VMHandler).GetVMStatus(driverIId)
			if status == cres.NotExist { // alibaba returns NotExist with err==nil
				err = fmt.Errorf("Not Found %s", driverIId.SystemId)
			}
			if err != nil {
				if checkNotFoundError(err) { // VM can be deleted after terminate.
					break
				}
				if status == cres.Failed { // tencent returns Failed with "Not Found Status error msg" in Korean
					break
				}
				cblog.Error(err)
				if force != "true" {
					callInfo.ErrorMSG = err.Error()
					callogger.Info(call.String(callInfo))
					return false, status, err
				}else {
					break
				}
			}
			if status == cres.Terminated {
				vmStatus = status
				break
			}

			if !waiter.Wait() {
				err := fmt.Errorf("[%s] Failed to terminate VM %s. (Timeout=%v)", connectionName, driverIId.NameId, waiter.Timeout)
				if force != "true" {
					callInfo.ErrorMSG = err.Error()
					callogger.Info(call.String(callInfo))
					return false, status, err
				}else {
					break
				}
			}
		}

		callInfo.ElapsedTime = call.Elapsed(start)
		callogger.Info(call.String(callInfo))
        case rsNLB:
                result, err = handler.(cres.NLBHandler).DeleteNLB(driverIId)
                if err != nil {
                        cblog.Error(err)
                        if force != "true" {
                                return false, "", err
                        }
                }
        case rsDisk:
                result, err = handler.(cres.DiskHandler).DeleteDisk(driverIId)
                if err != nil {
                        cblog.Error(err)
                        if force != "true" {
                                return false, "", err
                        }
                }
        case rsMyImage:
                result, err = handler.(cres.MyImageHandler).DeleteMyImage(driverIId)
                if err != nil {
                        cblog.Error(err)
                        if force != "true" {
                                return false, "", err
                        }
                }
        case rsCluster:
                result, err = handler.(cres.ClusterHandler).DeleteCluster(driverIId)
                if err != nil {
                        cblog.Error(err)
                        if force != "true" {
                                return false, "", err
                        }
                }

	default:
		err := fmt.Errorf(rsType + " is not supported Resource!!")
		return false, "", err
	}

	if force != "true" {
		if rsType != rsVM {
			if result == false {
				return result, "", nil
			}
		}
	}

	// (3) delete IID
        switch rsType {
        case rsVPC:
		// for vPC
		_, err = iidRWLock.DeleteIID(iidm.IIDSGROUP, connectionName, rsType, iidInfo.IId)
		if err != nil {
			cblog.Error(err)
			if force != "true" {
				return false, "", err
			}
		}
                // for Subnet list
                // key-value structure: ~/{SUBNETGROUP}/{ConnectionName}/{VPC-NameId}/{Subnet-reqNameId} [subnet-driverNameId:subnet-driverSystemId]  # VPC NameId => rsType
                subnetIIdInfoList, err2 := iidRWLock.ListIID(iidm.SUBNETGROUP, connectionName, iidInfo.IId.NameId/*vpcName*/)
                if err2 != nil {
                        cblog.Error(err)
                        if force != "true" {
                                return false, "", err
                        }
                }
                for _, subnetIIdInfo := range subnetIIdInfoList {
                        // key-value structure: ~/{SUBNETGROUP}/{ConnectionName}/{VPC-NameId}/{Subnet-reqNameId} [subnet-driverNameId:subnet-driverSystemId]  # VPC NameId => rsType
                        _, err := iidRWLock.DeleteIID(iidm.SUBNETGROUP, connectionName, iidInfo.IId.NameId/*vpcName*/, subnetIIdInfo.IId)
                        if err != nil {
                                cblog.Error(err)
                                if force != "true" {
                                        return false, "", err
                                }
                        }
                }
                // @todo Should we also delete the SG list of this VPC ? NO, We Can't delete the VPC had SGs

        case rsSG:
                _, err = iidRWLock.DeleteIID(iidm.SGGROUP, connectionName, iidInfo.ResourceType/*vpcName*/, cres.IID{nameID, ""})
                if err != nil {
                        cblog.Error(err)
                        if force != "true" {
                                return false, "", err
                        }
                }
        case rsVM:
                _, err = iidRWLock.DeleteIID(iidm.IIDSGROUP, connectionName, rsType, iidInfo.IId)
                if err != nil {
                        cblog.Error(err)
                        if force != "true" {
                                return false, "", err
                        }
                }
		return result, vmStatus, nil
        case rsNLB:
                _, err = iidRWLock.DeleteIID(iidm.NLBGROUP, connectionName, iidInfo.ResourceType/*vpcName*/, cres.IID{nameID, ""})
                if err != nil {
                        cblog.Error(err)
                        if force != "true" {
                                return false, "", err
                        }
                }
        case rsCluster:
                _, err = iidRWLock.DeleteIID(iidm.CLUSTERGROUP, connectionName, iidInfo.ResourceType/*vpcName*/, cres.IID{nameID, ""})
                if err != nil {
                        cblog.Error(err)
                        if force != "true" {
                                return false, "", err
                        }
                }

                // for NodeGroup list
                // key-value structure: ~/{NODEGROUP}/{ConnectionName}/{Cluster-NameId}/{NodeGroup-reqNameId} [nodegroup-driverNameId:nodegroup-driverSystemId]  # Cluster NameId => rsType
                ngIIdInfoList, err2 := iidRWLock.ListIID(iidm.NGGROUP, connectionName, iidInfo.IId.NameId/*clusterName*/)
                if err2 != nil {
                        cblog.Error(err)
                        if force != "true" {
                                return false, "", err
                        }
                }
                for _, ngIIdInfo := range ngIIdInfoList {                        
                        _, err := iidRWLock.DeleteIID(iidm.NGGROUP, connectionName, iidInfo.IId.NameId/*clusterName*/, ngIIdInfo.IId)
                        if err != nil {
                                cblog.Error(err)
                                if force != "true" {
                                        return false, "", err
                                }
                        }
                }


        default: // ex) KeyPair, Disk
		_, err = iidRWLock.DeleteIID(iidm.IIDSGROUP, connectionName, rsType, iidInfo.IId)
		if err != nil {
			cblog.Error(err)
			if force != "true" {
				return false, "", err
			}
		}
        }


	// except rsVM
	return result, "", nil
}

// delete CSP's Resource(SystemId)
func DeleteCSPResource(connectionName string, rsType string, systemID string) (bool, cres.VMStatus, error) {
	cblog.Info("call DeleteCSPResource()")

	// check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
                return false, "", err
		cblog.Error(err)
        }

        systemID, err = EmptyCheckAndTrim("systemID", systemID)
        if err != nil {
                return false, "", err
		cblog.Error(err)
        }

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return false, "", err
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
	case rsNLB:
		handler, err = cldConn.CreateNLBHandler()
	case rsDisk:
		handler, err = cldConn.CreateDiskHandler()
	case rsMyImage:
		handler, err = cldConn.CreateMyImageHandler()
	case rsCluster:
		handler, err = cldConn.CreateClusterHandler()
	default:
		return false, "", fmt.Errorf(rsType + " is not supported Resource!!")
	}
	if err != nil {
		cblog.Error(err)
		return false, "", err
	}

	iid := cres.IID{getMSShortID(systemID), getMSShortID(systemID)}

	// delete CSP's Resource(SystemId)	
	result := false
	var vmStatus cres.VMStatus
	switch rsType {
	case rsVPC:
		result, err = handler.(cres.VPCHandler).DeleteVPC(iid)
		if err != nil {
			cblog.Error(err)
			return false, "", err
		}
	case rsSG:
		result, err = handler.(cres.SecurityHandler).DeleteSecurity(iid)
		if err != nil {
			cblog.Error(err)
			return false, "", err
		}
	case rsKey:
		result, err = handler.(cres.KeyPairHandler).DeleteKey(iid)
		if err != nil {
			cblog.Error(err)
			return false, "", err
		}
	case rsVM:
		vmStatus, err = handler.(cres.VMHandler).TerminateVM(iid)
		if err != nil {
			cblog.Error(err)
			return false, vmStatus, err
		}
	case rsNLB:
		result, err = handler.(cres.NLBHandler).DeleteNLB(iid)
		if err != nil {
			cblog.Error(err)
			return false, "", err
		}
        case rsDisk:
                result, err = handler.(cres.DiskHandler).DeleteDisk(iid)
                if err != nil {
                        cblog.Error(err)
                        return false, "", err
                }
        case rsMyImage:
                result, err = handler.(cres.MyImageHandler).DeleteMyImage(iid)
                if err != nil {
                        cblog.Error(err)
                        return false, "", err
                }
        case rsCluster:
                result, err = handler.(cres.ClusterHandler).DeleteCluster(iid)
                if err != nil {
                        cblog.Error(err)
                        return false, "", err
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
	} else {
		return result, "", nil
	}
}

// Get Json string of CSP's Resource(SystemId) Info
func GetCSPResourceInfo(connectionName string, rsType string, systemID string) ([]byte, error) {
	cblog.Info("call GetCSPResourceInfo()")

	// check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
                return nil, err
		cblog.Error(err)
        }

        systemID, err = EmptyCheckAndTrim("systemID", systemID)
        if err != nil {
                return nil, err
		cblog.Error(err)
        }

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
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
	case rsNLB:
		handler, err = cldConn.CreateNLBHandler()
	case rsDisk:
		handler, err = cldConn.CreateDiskHandler()
	case rsMyImage:
		handler, err = cldConn.CreateMyImageHandler()
	case rsCluster:
		handler, err = cldConn.CreateClusterHandler()
	default:
		return nil, fmt.Errorf(rsType + " is not supported Resource!!")
	}
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	iid := cres.IID{getMSShortID(systemID), getMSShortID(systemID)}

	// Get CSP's Resource(SystemId)	
	jsonResult := []byte{}
	switch rsType {
	case rsVPC:
		result, err := handler.(cres.VPCHandler).GetVPC(iid)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		jsonResult, _ = json.Marshal(result)		
	case rsSG:
		result, err := handler.(cres.SecurityHandler).GetSecurity(iid)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		jsonResult, _ = json.Marshal(result)
	case rsKey:
		result, err := handler.(cres.KeyPairHandler).GetKey(iid)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		jsonResult, _ = json.Marshal(result)
	case rsVM:
		result, err := handler.(cres.VMHandler).GetVM(iid)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		jsonResult, _ = json.Marshal(result)
	case rsNLB:
		result, err := handler.(cres.NLBHandler).GetNLB(iid)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		jsonResult, _ = json.Marshal(result)
        case rsDisk:
                result, err := handler.(cres.DiskHandler).GetDisk(iid)
                if err != nil {
                        cblog.Error(err)
                        return nil, err
                }
                jsonResult, _ = json.Marshal(result)
        case rsMyImage:
                result, err := handler.(cres.MyImageHandler).GetMyImage(iid)
                if err != nil {
                        cblog.Error(err)
                        return nil, err
                }
                jsonResult, _ = json.Marshal(result)
        case rsCluster:
                result, err := handler.(cres.ClusterHandler).GetCluster(iid)
                if err != nil {
                        cblog.Error(err)
                        return nil, err
                }
                jsonResult, _ = json.Marshal(result)

	default:
		return nil, fmt.Errorf(rsType + " is not supported Resource!!")
	}

	//return string(jsonResult), nil
	return jsonResult, nil
}

//================ get CSP Name
func GetCSPResourceName(connectionName string, rsType string, nameID string) (string, error) {
        cblog.Info("call GetCSPResourceName()")

        // check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
                cblog.Error(err)
                return "", err
        }

        nameID, err = EmptyCheckAndTrim("nameID", nameID)
        if err != nil {
                cblog.Error(err)
                return "", err
        }

        // (1) get IID(NameId)
        iidInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsType, cres.IID{nameID, ""})
        if err != nil {
                cblog.Error(err)
                return "", err
        }

        // (2) get DriverNameId and return it
        return getDriverIID(iidInfo.IId).NameId, nil
}
