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
	"os"
	"strings"
	"sync"
	"time"

	"encoding/json"

	cblogger "github.com/cloud-barista/cb-log"
	splock "github.com/cloud-barista/cb-spider/api-runtime/common-runtime/sp-lock"
	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	iidm "github.com/cloud-barista/cb-spider/cloud-control-manager/iid-manager"
	infostore "github.com/cloud-barista/cb-spider/info-store"

	"github.com/sirupsen/logrus"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
)

// define string of resource types
// redefined for backward compatibility
const (
	IMAGE     string = string(cres.IMAGE)
	VPC       string = string(cres.VPC)
	SUBNET    string = string(cres.SUBNET)
	SG        string = string(cres.SG)
	KEY       string = string(cres.KEY)
	VM        string = string(cres.VM)
	NLB       string = string(cres.NLB)
	DISK      string = string(cres.DISK)
	MYIMAGE   string = string(cres.MYIMAGE)
	CLUSTER   string = string(cres.CLUSTER)
	NODEGROUP string = string(cres.NODEGROUP)
)

func RSTypeString(rsType string) string {
	return cres.RSTypeString(cres.RSType(rsType))
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

type ZoneLevelIIDInfo struct {
	ConnectionName string `gorm:"primaryKey"` // ex) "aws-seoul-config"
	ZoneId         string // ex) "ap-northeast-2a"
	NameId         string `gorm:"primaryKey"` // ex) "my_resource"
	SystemId       string // ID in CSP, ex) "i7baab81a4ez"
}

type VPCDependentIIDInfo struct {
	ConnectionName string `gorm:"primaryKey"` // ex) "aws-seoul-config"
	NameId         string `gorm:"primaryKey"` // ex) "my_resource"
	SystemId       string // ID in CSP, ex) "i7baab81a4ez"
	OwnerVPCName   string `gorm:"primaryKey"` // ex) "my_vpc" for NLB
}

type ZoneLevelVPCDependentIIDInfo struct {
	ConnectionName string `gorm:"primaryKey"` // ex) "aws-seoul-config"
	ZoneId         string // ex) "ap-northeast-2a"
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
	setLogLevel()
}

type AllResourceList struct {
	AllList struct {
		MappedList     []*cres.IID `json:"MappedList"`
		OnlySpiderList []*cres.IID `json:"OnlySpiderList"`
		OnlyCSPList    []*cres.IID `json:"OnlyCSPList"`
	}
}

func setLogLevel() {
	logLevel := strings.ToLower(os.Getenv("SPIDER_LOG_LEVEL"))
	if logLevel != "" {
		cblogger.SetLevel(logLevel)
	}

	callLogLevel := strings.ToLower(os.Getenv("SPIDER_HISCALL_LOG_LEVEL"))
	if callLogLevel != "" {
		call.SetLevel(callLogLevel)
	}
}

func GetID_MGMT(thisMode string) string {

	switch strings.ToUpper(thisMode) {
	case "ON":
		return "ON"
	case "OFF":
		return "OFF"
	}

	// default: ON
	if os.Getenv("ID_TRANSFORM_MODE") != "OFF" {
		return "ON"
	}
	return "OFF"
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
		strings.Contains(msg, "notexist") || strings.Contains(msg, "failedtofind") || strings.Contains(msg, "failedtogetthevm") || strings.Contains(msg, "noresult")
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
	case VPC, SUBNET:
		vpcSPLock.Lock(connectionName, nameId)
		defer vpcSPLock.Unlock(connectionName, nameId)
	case SG:
		sgSPLock.Lock(connectionName, nameId)
		defer sgSPLock.Unlock(connectionName, nameId)
	case KEY:
		keySPLock.Lock(connectionName, nameId)
		defer keySPLock.Unlock(connectionName, nameId)
	case VM:
		vmSPLock.Lock(connectionName, nameId)
		defer vmSPLock.Unlock(connectionName, nameId)
	case NLB:
		nlbSPLock.Lock(connectionName, nameId)
		defer nlbSPLock.Unlock(connectionName, nameId)
	case DISK:
		diskSPLock.Lock(connectionName, nameId)
		defer diskSPLock.Unlock(connectionName, nameId)
	case MYIMAGE:
		myImageSPLock.Lock(connectionName, nameId)
		defer myImageSPLock.Unlock(connectionName, nameId)
	case CLUSTER:
		clusterSPLock.Lock(connectionName, nameId)
		defer clusterSPLock.Unlock(connectionName, nameId)
	default:
		return false, fmt.Errorf(rsType + " is not supported Resource!!")
	}

	// check existence(UserID) and unregister it from metadb
	switch rsType {
	case VPC:
		var iidInfoList []*VPCIIDInfo
		err := infostore.ListByConditions(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameId)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		if len(iidInfoList) <= 0 {
			return false, fmt.Errorf("The %s '%s' does not exist!", RSTypeString(rsType), nameId)
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

	case KEY:
		var iidInfoList []*KeyIIDInfo
		err := infostore.ListByConditions(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameId)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		if len(iidInfoList) <= 0 {
			return false, fmt.Errorf("The %s '%s' does not exist!", RSTypeString(rsType), nameId)
		}

		_, err = infostore.DeleteByConditions(&KeyIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameId)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		return true, nil

	case VM:
		var iidInfoList []*VMIIDInfo
		err := infostore.ListByConditions(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameId)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		if len(iidInfoList) <= 0 {
			return false, fmt.Errorf("The %s '%s' does not exist!", RSTypeString(rsType), nameId)
		}

		_, err = infostore.DeleteByConditions(&VMIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameId)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		return true, nil

	case DISK:
		var iidInfoList []*DiskIIDInfo
		err := infostore.ListByConditions(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameId)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		if len(iidInfoList) <= 0 {
			return false, fmt.Errorf("The %s '%s' does not exist!", RSTypeString(rsType), nameId)
		}

		_, err = infostore.DeleteByConditions(&DiskIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameId)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		return true, nil

	case MYIMAGE:
		var iidInfoList []*MyImageIIDInfo
		err := infostore.ListByConditions(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameId)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		if len(iidInfoList) <= 0 {
			return false, fmt.Errorf("The %s '%s' does not exist!", RSTypeString(rsType), nameId)
		}

		_, err = infostore.DeleteByConditions(&MyImageIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameId)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		return true, nil

	//// following resources are dependent on the VPC.
	case SG:
		var iidInfoList []*SGIIDInfo
		err := infostore.ListByConditions(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameId)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		if len(iidInfoList) <= 0 {
			return false, fmt.Errorf("The %s '%s' does not exist!", RSTypeString(rsType), nameId)
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

	case NLB:
		var iidInfoList []*NLBIIDInfo
		err := infostore.ListByConditions(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameId)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		if len(iidInfoList) <= 0 {
			return false, fmt.Errorf("The %s '%s' does not exist!", RSTypeString(rsType), nameId)
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

	case CLUSTER:
		var iidInfoList []*ClusterIIDInfo
		err := infostore.ListByConditions(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameId)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		if len(iidInfoList) <= 0 {
			return false, fmt.Errorf("The %s '%s' does not exist!", RSTypeString(rsType), nameId)
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

	return false, fmt.Errorf("The %s '%s' does not exist!", RSTypeString(rsType), nameId)
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
	case VPC:
		handler, err = cldConn.CreateVPCHandler()
	case SG:
		handler, err = cldConn.CreateSecurityHandler()
	case KEY:
		handler, err = cldConn.CreateKeyPairHandler()
	case VM:
		handler, err = cldConn.CreateVMHandler()
	case NLB:
		handler, err = cldConn.CreateNLBHandler()
	case DISK:
		handler, err = cldConn.CreateDiskHandler()
	case MYIMAGE:
		handler, err = cldConn.CreateMyImageHandler()
	case CLUSTER:
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
	case VPC:
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
	case KEY:
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
	case VM:
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
	case DISK:
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
	case MYIMAGE:
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
	case SG:
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
	case NLB:
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
	case CLUSTER:
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
	case VPC:
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
	case SG:
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
	case KEY:
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
	case VM:
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
	case NLB:
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
	case DISK:
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
	case MYIMAGE:
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
	case CLUSTER:
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
	for _, userIId := range iidList { // iidList has already userIId
		exist := false
		for _, iid := range iidCSPList {
			if userIId.SystemId == iid.SystemId {
				MappedList = append(MappedList, userIId)
				exist = true
			}
		}
		if !exist {
			OnlySpiderList = append(OnlySpiderList, userIId)
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

	var cldConn icon.CloudConnection
	zoneId := ""
	switch rsType {
	case DISK: // Zone-Level Control Resource(ex. Disk)
		// (1) get IID(SystemId)
		var iidInfo DiskIIDInfo
		err = infostore.GetByConditionAndContain(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, SYSTEM_ID_COLUMN, systemID)
		if err != nil {
			if strings.Contains(err.Error(), "not exist") {
				// if not exist, find Owner ZoneId
				zoneId, err = findDiskOwnerZoneId(connectionName, systemID)
				if err != nil {
					cblog.Error(err)
					return false, "", err
				}

			} else {
				cblog.Error(err)
				return false, "", err
			}
		} else {
			zoneId = iidInfo.ZoneId
		}

		cldConn, err = ccm.GetZoneLevelCloudConnection(connectionName, zoneId)

	default:
		cldConn, err = ccm.GetCloudConnection(connectionName)
	}
	if err != nil {
		cblog.Error(err)
		return false, "", err
	}

	var handler interface{}

	switch rsType {
	case VPC:
		handler, err = cldConn.CreateVPCHandler()
	case SG:
		handler, err = cldConn.CreateSecurityHandler()
	case KEY:
		handler, err = cldConn.CreateKeyPairHandler()
	case VM:
		handler, err = cldConn.CreateVMHandler()
	case NLB:
		handler, err = cldConn.CreateNLBHandler()
	case DISK:
		handler, err = cldConn.CreateDiskHandler()
	case MYIMAGE:
		handler, err = cldConn.CreateMyImageHandler()
	case CLUSTER:
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
	case VPC:
		result, err = handler.(cres.VPCHandler).DeleteVPC(iid)
		if err != nil {
			cblog.Error(err)
			return false, "", err
		}
	case SG:
		result, err = handler.(cres.SecurityHandler).DeleteSecurity(iid)
		if err != nil {
			cblog.Error(err)
			return false, "", err
		}
	case KEY:
		result, err = handler.(cres.KeyPairHandler).DeleteKey(iid)
		if err != nil {
			cblog.Error(err)
			return false, "", err
		}
	case VM:
		vmStatus, err = handler.(cres.VMHandler).TerminateVM(iid)
		if err != nil {
			cblog.Error(err)
			return false, vmStatus, err
		}
	case NLB:
		result, err = handler.(cres.NLBHandler).DeleteNLB(iid)
		if err != nil {
			cblog.Error(err)
			return false, "", err
		}
	case DISK:
		result, err = handler.(cres.DiskHandler).DeleteDisk(iid)
		if err != nil {
			cblog.Error(err)
			return false, "", err
		}
	case MYIMAGE:
		result, err = handler.(cres.MyImageHandler).DeleteMyImage(iid)
		if err != nil {
			cblog.Error(err)
			return false, "", err
		}
	case CLUSTER:
		result, err = handler.(cres.ClusterHandler).DeleteCluster(iid)
		if err != nil {
			cblog.Error(err)
			return false, "", err
		}

	default:
		return false, "", fmt.Errorf(rsType + " is not supported Resource!!")
	}

	if rsType != VM {
		if !result {
			return result, "", nil
		}
	}

	if rsType == VM {
		return result, vmStatus, nil
	} else {
		return result, "", nil
	}
}

func findDiskOwnerZoneId(connectionName string, systemID string) (string, error) {
	regionName, _, err := ccm.GetRegionNameByConnectionName(connectionName)
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	// Get current Region Info with ZoneList
	regionZoneInfo, err := GetRegionZone(connectionName, regionName)
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	// find Owner ZoneId in all Zones
	for _, zoneInfo := range regionZoneInfo.ZoneList {
		cldConn, err := ccm.GetZoneLevelCloudConnection(connectionName, zoneInfo.Name)
		if err != nil {
			cblog.Error(err)
			return "", err
		}

		handler, err := cldConn.CreateDiskHandler()
		if err != nil {
			cblog.Error(err)
			return "", err
		}

		// (2) get resource(SystemId)
		_, err = handler.GetDisk(getDriverIID(cres.IID{NameId: systemID, SystemId: systemID}))
		if err != nil {
			cblog.Info(err)
			continue // for loop
		}
		return zoneInfo.Name, nil
	}
	return "", fmt.Errorf("The '%s' does not exist in %s(%s)", systemID, connectionName, regionName)
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

	var cldConn icon.CloudConnection
	zoneId := ""
	switch rsType {
	case DISK: // Zone-Level Control Resource(ex. Disk)
		// (1) get IID(SystemId)
		var iidInfo DiskIIDInfo
		err = infostore.GetByConditionAndContain(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, SYSTEM_ID_COLUMN, systemID)
		if err != nil {
			if strings.Contains(err.Error(), "not exist") {
				// if not exist, find Owner ZoneId
				zoneId, err = findDiskOwnerZoneId(connectionName, systemID)
				if err != nil {
					cblog.Error(err)
					return nil, err
				}

			} else {
				cblog.Error(err)
				return nil, err
			}
		} else {
			zoneId = iidInfo.ZoneId
		}

		cldConn, err = ccm.GetZoneLevelCloudConnection(connectionName, zoneId)

	default:
		cldConn, err = ccm.GetCloudConnection(connectionName)
	}
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	var handler interface{}

	switch rsType {
	case VPC:
		handler, err = cldConn.CreateVPCHandler()
	case SG:
		handler, err = cldConn.CreateSecurityHandler()
	case KEY:
		handler, err = cldConn.CreateKeyPairHandler()
	case VM:
		handler, err = cldConn.CreateVMHandler()
	case NLB:
		handler, err = cldConn.CreateNLBHandler()
	case DISK:
		handler, err = cldConn.CreateDiskHandler()
	case MYIMAGE:
		handler, err = cldConn.CreateMyImageHandler()
	case CLUSTER:
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
	case VPC:
		result, err := handler.(cres.VPCHandler).GetVPC(iid)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		jsonResult, _ = json.Marshal(result)
	case SG:
		result, err := handler.(cres.SecurityHandler).GetSecurity(iid)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		jsonResult, _ = json.Marshal(result)
	case KEY:
		result, err := handler.(cres.KeyPairHandler).GetKey(iid)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		jsonResult, _ = json.Marshal(result)
	case VM:
		result, err := handler.(cres.VMHandler).GetVM(iid)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		jsonResult, _ = json.Marshal(result)
	case NLB:
		result, err := handler.(cres.NLBHandler).GetNLB(iid)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		jsonResult, _ = json.Marshal(result)
	case DISK:
		result, err := handler.(cres.DiskHandler).GetDisk(iid)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		jsonResult, _ = json.Marshal(result)
	case MYIMAGE:
		result, err := handler.(cres.MyImageHandler).GetMyImage(iid)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		jsonResult, _ = json.Marshal(result)
	case CLUSTER:
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
	case VPC:
		// (1) get IID(NameId)
		var iid VPCIIDInfo
		err = infostore.GetByConditions(&iid, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
		if err != nil {
			cblog.Error(err)
			return "", err
		}
		// (2) get DriverNameId and return it
		return makeDriverIID(iid.NameId, iid.SystemId).NameId, nil
	case SG:
		// (1) get IID(NameId)
		var iid SGIIDInfo
		err = infostore.GetByConditions(&iid, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
		if err != nil {
			cblog.Error(err)
			return "", err
		}
		// (2) get DriverNameId and return it
		return makeDriverIID(iid.NameId, iid.SystemId).NameId, nil
	case KEY:
		// (1) get IID(NameId)
		var iid KeyIIDInfo
		err = infostore.GetByConditions(&iid, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
		if err != nil {
			cblog.Error(err)
			return "", err
		}
		// (2) get DriverNameId and return it
		return makeDriverIID(iid.NameId, iid.SystemId).NameId, nil
	case VM:
		// (1) get IID(NameId)
		var iid VMIIDInfo
		err = infostore.GetByConditions(&iid, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
		if err != nil {
			cblog.Error(err)
			return "", err
		}
		// (2) get DriverNameId and return it
		return makeDriverIID(iid.NameId, iid.SystemId).NameId, nil
	case NLB:
		// (1) get IID(NameId)
		var iid NLBIIDInfo
		err = infostore.GetByConditions(&iid, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
		if err != nil {
			cblog.Error(err)
			return "", err
		}
		// (2) get DriverNameId and return it
		return makeDriverIID(iid.NameId, iid.SystemId).NameId, nil
	case DISK:
		// (1) get IID(NameId)
		var iid DiskIIDInfo
		err = infostore.GetByConditions(&iid, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
		if err != nil {
			cblog.Error(err)
			return "", err
		}
		// (2) get DriverNameId and return it
		return makeDriverIID(iid.NameId, iid.SystemId).NameId, nil
	case MYIMAGE:
		// (1) get IID(NameId)
		var iid MyImageIIDInfo
		err = infostore.GetByConditions(&iid, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
		if err != nil {
			cblog.Error(err)
			return "", err
		}
		// (2) get DriverNameId and return it
		return makeDriverIID(iid.NameId, iid.SystemId).NameId, nil
	case CLUSTER:
		// (1) get IID(NameId)
		var iid ClusterIIDInfo
		err = infostore.GetByConditions(&iid, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
		if err != nil {
			cblog.Error(err)
			return "", err
		}
		// (2) get DriverNameId and return it
		return makeDriverIID(iid.NameId, iid.SystemId).NameId, nil
	default:
		return "", fmt.Errorf(rsType + " is not supported Resource!!")
	}
}

// ListResourceName lists resource names by connectionName and rsType
func ListResourceName(connectionName, rsType string) ([]string, error) {
	var info interface{}

	// Determine the type of info based on rsType
	switch rsType {
	case VPC:
		v := VPCIIDInfo{}
		info = &v
	case SG:
		v := SGIIDInfo{}
		info = &v
	case KEY:
		v := KeyIIDInfo{}
		info = &v
	case VM:
		v := VMIIDInfo{}
		info = &v
	case NLB:
		v := NLBIIDInfo{}
		info = &v
	case DISK:
		v := DiskIIDInfo{}
		info = &v
	case MYIMAGE:
		v := MyImageIIDInfo{}
		info = &v
	case CLUSTER:
		v := ClusterIIDInfo{}
		info = &v
	default:
		return nil, fmt.Errorf("%s is not a supported Resource!!", rsType)
	}

	// List Name IDs by connectionName
	nameIds, err := infostore.ListNameIDByConnection(info, connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	return nameIds, nil
}

type DestroyedInfo struct {
	IsAllDestroyed bool                       `json:"IsAllDestroyed"` // true: all destroyed, false: some remained
	DestroyedList  []*DeletedResourceInfoList `json:"DeletedAllListByResourceType"`
}

type DeletedResourceInfoList struct {
	ResourceType          string               `json:"ResourceType"`
	IsAllDeleted          bool                 `json:"IsAllDeleted"`
	DeletedIIDList        []*cres.IID          `json:"DeletedIIDList"`
	RemainedErrorInfoList []*RemainedErrorInfo `json:"RemainedErrorInfoList"`
}

type RemainedErrorInfo struct {
	Name     string `json:"Name"`
	ErrorMsg string `json:"ErrorMsg"`
}

// Destroy all Resources in a Connection
func Destroy(connectionName string) (DestroyedInfo, error) {
	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Println(err)
		return DestroyedInfo{}, err
	}

	var destroyedInfo DestroyedInfo
	destroyedInfo.IsAllDestroyed = true

	// Define resource type groups
	resourceTypeGroups := [][]string{
		{CLUSTER, MYIMAGE, NLB},
		{VM},
		{DISK},
		{KEY, SG},
		{VPC},
	}

	for _, resourceTypes := range resourceTypeGroups {
		var wg sync.WaitGroup
		var mu sync.Mutex
		var groupErr error

		for _, resourceType := range resourceTypes {
			wg.Add(1)
			go func(resourceType string) {
				defer wg.Done()

				var finalDeletedResourceInfoList DeletedResourceInfoList
				finalDeletedResourceInfoList.ResourceType = resourceType

				for retry := 0; retry < 10; retry++ {
					deletedResourceInfoList, err := deleteAllResourcesInResType(connectionName, resourceType)
					mu.Lock()
					if err != nil {
						cblog.Println(err)
						groupErr = err
						mu.Unlock()
						return
					}
					if deletedResourceInfoList == nil {
						mu.Unlock()
						return
					}

					// Append the deleted resource info list
					finalDeletedResourceInfoList.DeletedIIDList = append(finalDeletedResourceInfoList.DeletedIIDList, deletedResourceInfoList.DeletedIIDList...)
					finalDeletedResourceInfoList.IsAllDeleted = deletedResourceInfoList.IsAllDeleted
					if !deletedResourceInfoList.IsAllDeleted {
						finalDeletedResourceInfoList.RemainedErrorInfoList = deletedResourceInfoList.RemainedErrorInfoList
					}

					if deletedResourceInfoList.IsAllDeleted {
						destroyedInfo.DestroyedList = append(destroyedInfo.DestroyedList, &finalDeletedResourceInfoList)
						mu.Unlock()
						return
					}
					mu.Unlock()
					time.Sleep(3 * time.Second)
				}

				mu.Lock()
				destroyedInfo.IsAllDestroyed = false
				destroyedInfo.DestroyedList = append(destroyedInfo.DestroyedList, &finalDeletedResourceInfoList)
				mu.Unlock()
			}(resourceType)
		}

		wg.Wait()

		if groupErr != nil {
			return DestroyedInfo{}, groupErr
		}
	}

	return destroyedInfo, nil
}

// deletes all resources of a specific resource type in a connection
func deleteAllResourcesInResType(connectionName string, rsType string) (*DeletedResourceInfoList, error) {

	nameList, err := ListResourceName(connectionName, rsType)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	if len(nameList) <= 0 {
		return nil, nil
	}

	deletedResourceInfoList := &DeletedResourceInfoList{
		ResourceType: rsType,
		IsAllDeleted: true,
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, nameId := range nameList {
		wg.Add(1)
		go func(nameId string) {
			defer wg.Done()
			var err error

			switch rsType {
			case VPC:
				_, err = DeleteVPC(connectionName, VPC, nameId, "false")
			case SG:
				_, err = DeleteSecurity(connectionName, SG, nameId, "false")
			case KEY:
				_, err = DeleteKey(connectionName, KEY, nameId, "false")
			case VM:
				_, _, err = DeleteVM(connectionName, VM, nameId, "false")
			case NLB:
				_, err = DeleteNLB(connectionName, NLB, nameId, "false")
			case DISK:
				_, err = DeleteDisk(connectionName, DISK, nameId, "false")
			case MYIMAGE:
				_, err = DeleteMyImage(connectionName, MYIMAGE, nameId, "false")
			case CLUSTER:
				_, err = DeleteCluster(connectionName, CLUSTER, nameId, "false")
			default:
				err = fmt.Errorf("%s is not supported Resource!!", rsType)
			}

			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				deletedResourceInfoList.IsAllDeleted = false
				deletedResourceInfoList.RemainedErrorInfoList = append(deletedResourceInfoList.RemainedErrorInfoList, &RemainedErrorInfo{
					Name:     nameId,
					ErrorMsg: err.Error(),
				})
			} else {
				deletedResourceInfoList.DeletedIIDList = append(deletedResourceInfoList.DeletedIIDList, &cres.IID{NameId: nameId})
			}
		}(nameId)
	}

	wg.Wait()

	return deletedResourceInfoList, nil
}
