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

	"encoding/json"

	cblogger "github.com/cloud-barista/cb-log"
	splock "github.com/cloud-barista/cb-spider/api-runtime/common-runtime/sp-lock"
	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	iidm "github.com/cloud-barista/cb-spider/cloud-control-manager/iid-manager"
	infostore "github.com/cloud-barista/cb-spider/info-store"

	"github.com/sirupsen/logrus"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
)

// define string of resource types
const (
	rsImage     string = "image"
	rsVPC       string = "vpc"
	rsSubnet    string = "subnet"
	rsSG        string = "sg"
	rsKey       string = "keypair"
	rsVM        string = "vm"
	rsNLB       string = "nlb"
	rsDisk      string = "disk"
	rsMyImage   string = "myimage"
	rsCluster   string = "cluster"
	rsNodeGroup string = "nodegroup"
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
var vpcSPLock = splock.New()
var sgSPLock = splock.New()
var keySPLock = splock.New()
var vmSPLock = splock.New()
var nlbSPLock = splock.New()
var diskSPLock = splock.New()
var myImageSPLock = splock.New()
var clusterSPLock = splock.New()

// ====================================================================
// Common column name and struct for GORM
const CONNECTION_NAME_COLUMN = "connection_name"
const NAME_ID_COLUMN = "name_id"
const SYSTEM_ID_COLUMN = "system_id"
const OWNER_VPC_NAME_COLUMN = "owner_vpc_name"
const OWNER_CLUSTER_NAME_COLUMN = "owner_cluster_name"

type FirstIIDInfo struct {
	ConnectionName string `gorm:"primaryKey"` // ex) "aws-seoul-config"
	NameId         string `gorm:"primaryKey"` // ex) "my_resource"
	SystemId       string // ID in CSP, ex) "i7baab81a4ez"
}

type VPCDependentIIDInfo struct {
	ConnectionName string `gorm:"primaryKey"` // ex) "aws-seoul-config"
	NameId         string `gorm:"primaryKey"` // ex) "my_resource"
	SystemId       string // ID in CSP, ex) "i7baab81a4ez"
	OwnerVPCName   string `gorm:"primaryKey"` // ex) "my_vpc" for NLB
}

type ClusterDependentIIDInfo struct {
	ConnectionName   string `gorm:"primaryKey"` // ex) "aws-seoul-config"
	NameId           string `gorm:"primaryKey"` // ex) "my_resource"
	SystemId         string // ID in CSP, ex) "i7baab81a4ez"
	OwnerClusterName string `gorm:"primaryKey"` // ex) "my_cluster"' for NodeGroup
}

// ====================================================================

var cblog *logrus.Logger
var callogger *logrus.Logger

func init() {
	cblog = cblogger.GetLogger("CLOUD-BARISTA")
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

	return strings.Contains(msg, "does not exist") || strings.Contains(msg, "notfound") ||
		strings.Contains(msg, "notexist") || strings.Contains(msg, "failedtofind") || strings.Contains(msg, "failedtogetthevm")
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
	driverIId := cres.IID{NameId: strArray[0], SystemId: systemId}
	return driverIId
}

// make a DriverIID from NameId and SystemId
func makeDriverIID(NameId string, SystemId string) cres.IID {
	// if AWS NLB's SystmeId,
	//     ex) arn:aws:elasticloadbalancing:us-east-2:635484366616:loadbalancer/net/spider-nl-cangp8aba5o2pi8oa7o0/1dee7370037afd6d
	strArray := strings.Split(SystemId, ":")
	systemId := strings.ReplaceAll(SystemId, strArray[0]+":", "")
	driverIId := cres.IID{NameId: strArray[0], SystemId: systemId}
	return driverIId
}

// Get userIID from SpiderIID
func getUserIID(spiderIId cres.IID) cres.IID {
	// if AWS NLB's SystmeId,
	//     ex) arn:aws:elasticloadbalancing:us-east-2:635484366616:loadbalancer/net/spider-nl-cangp8aba5o2pi8oa7o0/1dee7370037afd6d
	strArray := strings.Split(spiderIId.SystemId, ":")
	userIId := cres.IID{NameId: spiderIId.NameId, SystemId: strings.ReplaceAll(spiderIId.SystemId, strArray[0]+":", "")}
	return userIId
}

// make a UserIID from NameId and SystemId
func makeUserIID(NameId string, SystemId string) cres.IID {
	// if AWS NLB's SystmeId,
	//     ex) arn:aws:elasticloadbalancing:us-east-2:635484366616:loadbalancer/net/spider-nl-cangp8aba5o2pi8oa7o0/1dee7370037afd6d
	strArray := strings.Split(SystemId, ":")
	userIId := cres.IID{NameId: NameId, SystemId: strings.ReplaceAll(SystemId, strArray[0]+":", "")}
	return userIId
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
	case rsVPC, rsSubnet:
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

	// check existence(UserID) and unregister it from metadb
	switch rsType {
	case rsVPC:
		var iidInfoList []*VPCIIDInfo
		err := infostore.ListByConditions(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameId)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		if len(iidInfoList) <= 0 {
			return false, fmt.Errorf("The %s '%s' does not exist!", RsTypeString(rsType), nameId)
		}

		_, err = infostore.DeleteByConditions(&VPCIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameId)
		if err != nil {
			cblog.Error(err)
			return false, err
		}

		// unregister Subnets of this VPC
		_, err = infostore.DeleteByConditions(&SubnetIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, OWNER_VPC_NAME_COLUMN, nameId)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		return true, nil

	case rsKey:
		var iidInfoList []*KeyIIDInfo
		err := infostore.ListByConditions(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameId)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		if len(iidInfoList) <= 0 {
			return false, fmt.Errorf("The %s '%s' does not exist!", RsTypeString(rsType), nameId)
		}

		_, err = infostore.DeleteByConditions(&KeyIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameId)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		return true, nil

	case rsVM:
		var iidInfoList []*VMIIDInfo
		err := infostore.ListByConditions(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameId)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		if len(iidInfoList) <= 0 {
			return false, fmt.Errorf("The %s '%s' does not exist!", RsTypeString(rsType), nameId)
		}

		_, err = infostore.DeleteByConditions(&VMIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameId)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		return true, nil

	case rsDisk:
		var iidInfoList []*DiskIIDInfo
		err := infostore.ListByConditions(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameId)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		if len(iidInfoList) <= 0 {
			return false, fmt.Errorf("The %s '%s' does not exist!", RsTypeString(rsType), nameId)
		}

		_, err = infostore.DeleteByConditions(&DiskIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameId)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		return true, nil

	case rsMyImage:
		var iidInfoList []*MyImageIIDInfo
		err := infostore.ListByConditions(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameId)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		if len(iidInfoList) <= 0 {
			return false, fmt.Errorf("The %s '%s' does not exist!", RsTypeString(rsType), nameId)
		}

		_, err = infostore.DeleteByConditions(&KeyIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameId)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		return true, nil

	//// following resources are dependent on the VPC.
	case rsSG:
		var iidInfoList []*SGIIDInfo
		err := infostore.ListByConditions(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameId)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		if len(iidInfoList) <= 0 {
			return false, fmt.Errorf("The %s '%s' does not exist!", RsTypeString(rsType), nameId)
		}
		for _, OneIIdInfo := range iidInfoList {
			if OneIIdInfo.NameId == nameId {
				_, err2 := infostore.DeleteBy3Conditions(OneIIdInfo, CONNECTION_NAME_COLUMN, connectionName,
					NAME_ID_COLUMN, nameId, OWNER_VPC_NAME_COLUMN, OneIIdInfo.OwnerVPCName)
				if err2 != nil {
					cblog.Error(err2)
					return false, err2
				}
				return true, nil
			}
		}

	case rsNLB:
		var iidInfoList []*NLBIIDInfo
		err := infostore.ListByConditions(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameId)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		if len(iidInfoList) <= 0 {
			return false, fmt.Errorf("The %s '%s' does not exist!", RsTypeString(rsType), nameId)
		}
		for _, OneIIdInfo := range iidInfoList {
			if OneIIdInfo.NameId == nameId {
				_, err2 := infostore.DeleteBy3Conditions(OneIIdInfo, CONNECTION_NAME_COLUMN, connectionName,
					NAME_ID_COLUMN, nameId, OWNER_VPC_NAME_COLUMN, OneIIdInfo.OwnerVPCName)
				if err2 != nil {
					cblog.Error(err2)
					return false, err2
				}
				return true, nil
			}
		}

	case rsCluster:
		var iidInfoList []*ClusterIIDInfo
		err := infostore.ListByConditions(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameId)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		if len(iidInfoList) <= 0 {
			return false, fmt.Errorf("The %s '%s' does not exist!", RsTypeString(rsType), nameId)
		}
		for _, OneIIdInfo := range iidInfoList {
			if OneIIdInfo.NameId == nameId {
				_, err2 := infostore.DeleteBy3Conditions(OneIIdInfo, CONNECTION_NAME_COLUMN, connectionName,
					NAME_ID_COLUMN, nameId, OWNER_VPC_NAME_COLUMN, OneIIdInfo.OwnerVPCName)
				if err2 != nil {
					cblog.Error(err2)
					return false, err2
				}
				return true, nil
			}
		}

	default:
		return false, fmt.Errorf(rsType + " is not supported Resource!!")
	}

	return false, fmt.Errorf("The %s '%s' does not exist!", RsTypeString(rsType), nameId)
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
		cblog.Error(err)
		return AllResourceList{}, err
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

	// (1) get IID:list from metadb
	iidList := []*cres.IID{}
	switch rsType {
	case rsVPC:
		var iidInfoList []*VPCIIDInfo
		err := infostore.ListByCondition(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName)
		if err != nil {
			cblog.Error(err)
			return AllResourceList{}, err
		}
		for _, info := range iidInfoList {
			iid := makeUserIID(info.NameId, info.SystemId)
			iidList = append(iidList, &iid)
		}
	case rsKey:
		var iidInfoList []*KeyIIDInfo
		err := infostore.ListByCondition(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName)
		if err != nil {
			cblog.Error(err)
			return AllResourceList{}, err
		}
		for _, info := range iidInfoList {
			iid := makeUserIID(info.NameId, info.SystemId)
			iidList = append(iidList, &iid)
		}
	case rsVM:
		var iidInfoList []*VMIIDInfo
		err := infostore.ListByCondition(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName)
		if err != nil {
			cblog.Error(err)
			return AllResourceList{}, err
		}
		for _, info := range iidInfoList {
			iid := makeUserIID(info.NameId, info.SystemId)
			iidList = append(iidList, &iid)
		}
	case rsDisk:
		var iidInfoList []*DiskIIDInfo
		err := infostore.ListByCondition(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName)
		if err != nil {
			cblog.Error(err)
			return AllResourceList{}, err
		}
		for _, info := range iidInfoList {
			iid := makeUserIID(info.NameId, info.SystemId)
			iidList = append(iidList, &iid)
		}
	case rsMyImage:
		var iidInfoList []*MyImageIIDInfo
		err := infostore.ListByCondition(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName)
		if err != nil {
			cblog.Error(err)
			return AllResourceList{}, err
		}
		for _, info := range iidInfoList {
			iid := makeUserIID(info.NameId, info.SystemId)
			iidList = append(iidList, &iid)
		}
	case rsSG:
		var iidInfoList []*SGIIDInfo
		err = infostore.ListByCondition(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName)
		if err != nil {
			cblog.Error(err)
			return AllResourceList{}, err
		}
		for _, info := range iidInfoList {
			iid := makeUserIID(info.NameId, info.SystemId)
			iidList = append(iidList, &iid)
		}
	case rsNLB:
		var iidInfoList []*NLBIIDInfo
		err = infostore.ListByCondition(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName)
		if err != nil {
			cblog.Error(err)
			return AllResourceList{}, err
		}
		for _, info := range iidInfoList {
			iid := makeUserIID(info.NameId, info.SystemId)
			iidList = append(iidList, &iid)
		}
	case rsCluster:
		var iidInfoList []*ClusterIIDInfo
		err = infostore.ListByCondition(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName)
		if err != nil {
			cblog.Error(err)
			return AllResourceList{}, err
		}
		for _, info := range iidInfoList {
			iid := makeUserIID(info.NameId, info.SystemId)
			iidList = append(iidList, &iid)
		}

	default:
		return AllResourceList{}, fmt.Errorf(rsType + " is not supported Resource!!")
	}

	// if iidInfoList is empty, OnlySpiderList is empty.
	if iidList == nil || len(iidList) <= 0 {
		emptyIIDInfoList := []*cres.IID{}
		allResList.AllList.MappedList = emptyIIDInfoList
		allResList.AllList.OnlySpiderList = emptyIIDInfoList
	}

	// (2) get IID:list from CSP
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
		if iidList == nil || len(iidList) <= 0 {
			emptyIIDInfoList := []*cres.IID{}
			allResList.AllList.OnlyCSPList = emptyIIDInfoList

			return allResList, nil
		} else { // iidCSPList is empty and iidInfoList has value => only OnlySpiderList <--(2)
			emptyIIDInfoList := []*cres.IID{}
			allResList.AllList.MappedList = emptyIIDInfoList
			allResList.AllList.OnlyCSPList = emptyIIDInfoList
			allResList.AllList.OnlySpiderList = iidList

			return allResList, nil
		}
	}

	// iidInfoList is empty, iidCSPList has values => only OnlyCSPList <--------------------------(3)
	if iidList == nil || len(iidList) <= 0 {
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
	for _, iidInfo := range iidList {
		exist := false
		for _, iid := range iidCSPList {
			userIId := makeUserIID(iidInfo.NameId, iidInfo.SystemId)
			if userIId.SystemId == iid.SystemId {
				MappedList = append(MappedList, &userIId)
				exist = true
			}
		}
		if !exist {
			userIId := makeUserIID(iidInfo.NameId, iidInfo.SystemId)
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

// delete CSP's Resource(SystemId)
func DeleteCSPResource(connectionName string, rsType string, systemID string) (bool, cres.VMStatus, error) {
	cblog.Info("call DeleteCSPResource()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return false, "", err
	}

	systemID, err = EmptyCheckAndTrim("systemID", systemID)
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
		cblog.Error(err)
		return nil, err
	}

	systemID, err = EmptyCheckAndTrim("systemID", systemID)
	if err != nil {
		cblog.Error(err)
		return nil, err
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

// ================ get CSP Name
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

	switch rsType {
	case rsVPC:
		// (1) get IID(NameId)
		var iid VPCIIDInfo
		err = infostore.GetByConditions(&iid, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
		// (2) get DriverNameId and return it
		return makeDriverIID(iid.NameId, iid.SystemId).NameId, nil
	case rsSG:
		// (1) get IID(NameId)
		var iid SGIIDInfo
		err = infostore.GetByConditions(&iid, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
		// (2) get DriverNameId and return it
		return makeDriverIID(iid.NameId, iid.SystemId).NameId, nil
	case rsKey:
		// (1) get IID(NameId)
		var iid KeyIIDInfo
		err = infostore.GetByConditions(&iid, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
		// (2) get DriverNameId and return it
		return makeDriverIID(iid.NameId, iid.SystemId).NameId, nil
	case rsVM:
		// (1) get IID(NameId)
		var iid VMIIDInfo
		err = infostore.GetByConditions(&iid, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
		// (2) get DriverNameId and return it
		return makeDriverIID(iid.NameId, iid.SystemId).NameId, nil
	case rsNLB:
		// (1) get IID(NameId)
		var iid NLBIIDInfo
		err = infostore.GetByConditions(&iid, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
		// (2) get DriverNameId and return it
		return makeDriverIID(iid.NameId, iid.SystemId).NameId, nil
	case rsDisk:
		// (1) get IID(NameId)
		var iid DiskIIDInfo
		err = infostore.GetByConditions(&iid, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
		// (2) get DriverNameId and return it
		return makeDriverIID(iid.NameId, iid.SystemId).NameId, nil
	case rsMyImage:
		// (1) get IID(NameId)
		var iid MyImageIIDInfo
		err = infostore.GetByConditions(&iid, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
		// (2) get DriverNameId and return it
		return makeDriverIID(iid.NameId, iid.SystemId).NameId, nil
	case rsCluster:
		// (1) get IID(NameId)
		var iid ClusterIIDInfo
		err = infostore.GetByConditions(&iid, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
		// (2) get DriverNameId and return it
		return makeDriverIID(iid.NameId, iid.SystemId).NameId, nil
	default:
		return "", fmt.Errorf(rsType + " is not supported Resource!!")
	}
}
