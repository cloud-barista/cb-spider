// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2022.09.

package commonruntime

import (
	"errors"
	"fmt"
	"os"
	"strings"

	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	iidm "github.com/cloud-barista/cb-spider/cloud-control-manager/iid-manager"
	infostore "github.com/cloud-barista/cb-spider/info-store"
)

// ====================================================================
// type for GORM

type NLBIIDInfo VPCDependentIIDInfo

func (NLBIIDInfo) TableName() string {
	return "nlb_iid_infos"
}

//====================================================================

func init() {
	db, err := infostore.Open()
	if err != nil {
		cblog.Error(err)
		return
	}
	db.AutoMigrate(&NLBIIDInfo{})
	infostore.Close(db)
}

//================ NLB Handler

func GetNLBOwnerVPC(connectionName string, cspID string) (owerVPC cres.IID, err error) {
	cblog.Info("call GetNLBOwnerVPC()")

	// check empty and trim user inputs
	connectionName, err = EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return cres.IID{}, err
	}

	cspID, err = EmptyCheckAndTrim("cspID", cspID)
	if err != nil {
		cblog.Error(err)
		return cres.IID{}, err
	}

	rsType := NLB

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return cres.IID{}, err
	}

	handler, err := cldConn.CreateNLBHandler()
	if err != nil {
		cblog.Error(err)
		return cres.IID{}, err
	}

	// Except Management API
	//nlbSPLock.RLock()
	//vpcSPLock.RLock()

	// (1) check existence(cspID)
	var iidInfoList []*NLBIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			//vpcSPLock.RUnlock()
			//nlbSPLock.RUnlock()
			cblog.Error(err)
			return cres.IID{}, err
		}
	} else {
		err = infostore.ListByCondition(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName)
		if err != nil {
			//vpcSPLock.RUnlock()
			//nlbSPLock.RUnlock()
			cblog.Error(err)
			return cres.IID{}, err
		}
	}
	var isExist bool = false
	var nameId string
	for _, OneIIdInfo := range iidInfoList {
		if getMSShortID(getDriverSystemId(cres.IID{NameId: OneIIdInfo.NameId, SystemId: OneIIdInfo.SystemId})) == cspID {
			nameId = OneIIdInfo.NameId
			isExist = true
			break
		}
	}
	if isExist {
		//vpcSPLock.RUnlock()
		//nlbSPLock.RUnlock()
		err := fmt.Errorf("%s with SystemID '%s' already exists in connection '%s' (NameID: '%s')", RSTypeString(rsType), cspID, connectionName, nameId)
		cblog.Error(err)
		return cres.IID{}, err
	}

	// (2) get resource info(CSP-ID)
	// check existence and get info of this resouce in the CSP
	// Do not user NameId, because Azure driver use it like SystemId
	getInfo, err := handler.GetNLB(cres.IID{NameId: getMSShortID(cspID), SystemId: cspID})
	if err != nil {
		//vpcSPLock.RUnlock()
		//nlbSPLock.RUnlock()
		cblog.Error(err)
		return cres.IID{}, err
	}

	// (3) get VPC IID:list
	var vpcIIDInfoList []*VPCIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		err = getAuthIIDInfoList(connectionName, &vpcIIDInfoList)
		if err != nil {
			//vpcSPLock.RUnlock()
			//nlbSPLock.RUnlock()
			cblog.Error(err)
			return cres.IID{}, err
		}
	} else {
		err = infostore.ListByCondition(&vpcIIDInfoList, CONNECTION_NAME_COLUMN, connectionName)
		if err != nil {
			//vpcSPLock.RUnlock()
			//nlbSPLock.RUnlock()
			cblog.Error(err)
			return cres.IID{}, err
		}
	}
	//vpcSPLock.RUnlock()
	//nlbSPLock.RUnlock()

	//--------
	//-------- ex) spiderIID {"vpc-01", "vpc-01-9m4e2mr0ui3e8a215n4g:i-0bc7123b7e5cbf79d"}
	//--------
	// Do not use NameId, because Azure driver use it like SystemId
	vpcCSPID := getMSShortID(getInfo.VpcIID.SystemId)
	if vpcIIDInfoList == nil || len(vpcIIDInfoList) <= 0 {
		return cres.IID{NameId: "", SystemId: vpcCSPID}, nil
	}

	// (4) check existence in the MetaDB
	for _, one := range vpcIIDInfoList {
		if getMSShortID(getDriverSystemId(cres.IID{NameId: one.NameId, SystemId: one.SystemId})) == vpcCSPID {
			return cres.IID{NameId: one.NameId, SystemId: vpcCSPID}, nil
		}
	}

	return cres.IID{NameId: "", SystemId: vpcCSPID}, nil
}

// UserIID{UserID, CSP-ID} => SpiderIID{UserID, SP-XID:CSP-ID}
// (0) check VPC existence(VPC UserID)
// (1) check existence(UserID)
// (2) get resource info(CSP-ID)
// (3) create spiderIID: {UserID, SP-XID:CSP-ID}
// (4) insert spiderIID
func RegisterNLB(connectionName string, vpcUserID string, userIID cres.IID) (*cres.NLBInfo, error) {
	cblog.Info("call RegisterNLB()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	vpcUserID, err = EmptyCheckAndTrim("vpcUserID", vpcUserID)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	emptyPermissionList := []string{}

	err = ValidateStruct(userIID, emptyPermissionList)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	rsType := NLB

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateNLBHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	vpcSPLock.RLock(connectionName, vpcUserID)
	defer vpcSPLock.RUnlock(connectionName, vpcUserID)
	nlbSPLock.Lock(connectionName, userIID.NameId)
	defer nlbSPLock.Unlock(connectionName, userIID.NameId)

	// (0) check VPC existence(VPC UserID)
	var bool_ret bool
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		// check permission to vpcName
		var iidInfoList []*VPCIIDInfo
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		bool_ret, err = isNameIdExists(&iidInfoList, vpcUserID)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	} else {
		bool_ret, err = infostore.HasByConditions(&VPCIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, vpcUserID)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}
	if !bool_ret {
		err := fmt.Errorf("%s '%s' does not exist in connection '%s'", RSTypeString(VPC), vpcUserID, connectionName)
		cblog.Error(err)
		return nil, err
	}

	// (1) check existence(UserID)
	var isExist bool
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		isExist, err = infostore.HasByCondition(&NLBIIDInfo{}, NAME_ID_COLUMN, userIID.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	} else {
		isExist, err = infostore.HasByConditions(&NLBIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, userIID.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}

	if isExist {
		err := fmt.Errorf("%s '%s' already exists in connection '%s'", RSTypeString(rsType), userIID.NameId, connectionName)
		cblog.Error(err)
		return nil, err
	}

	// (2) get resource info(CSP-ID)
	// check existence and get info of this resouce in the CSP
	// Do not user NameId, because Azure driver use it like SystemId
	getInfo, err := handler.GetNLB(cres.IID{NameId: getMSShortID(userIID.SystemId), SystemId: userIID.SystemId})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// Protocol: to upper
	transformArgsToUpper(&getInfo)

	// (3) create spiderIID: {UserID, SP-XID:CSP-ID}
	//     ex) spiderIID {"vpc-01", "vpc-01-9m4e2mr0ui3e8a215n4g:i-0bc7123b7e5cbf79d"}
	// Do not user NameId, because Azure driver use it like SystemId
	systemId := getMSShortID(getInfo.IId.SystemId)
	// for AWS NLB Long SystemId
	systemId = getAWSNLBShortID(systemId)
	spiderIId := cres.IID{NameId: userIID.NameId, SystemId: systemId + ":" + getInfo.IId.SystemId}

	// (4) insert spiderIID
	// insert NLB SpiderIID to metadb
	err = infostore.Insert(&NLBIIDInfo{ConnectionName: connectionName, NameId: spiderIId.NameId, SystemId: spiderIId.SystemId,
		OwnerVPCName: vpcUserID})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// set up NLB User IID for return info
	getInfo.IId = userIID

	// set up VPC UserIID for return info
	var iidInfo VPCIIDInfo
	err = infostore.GetByConditions(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, vpcUserID)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	getInfo.VpcIID = getUserIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	return &getInfo, nil
}

// (1) check exist(NameID)
// (2) generate SP-XID and create reqIID, driverIID
// (3) create Resource
// (4) create spiderIID: {reqNameID, "driverNameID:driverSystemID"}
// (5) insert spiderIID
// (6) create userIID
func CreateNLB(connectionName string, rsType string, reqInfo cres.NLBInfo, IDTransformMode string) (*cres.NLBInfo, error) {
	cblog.Info("call CreateNLB()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// @todo
	/* Currently, Validator does not support the struct has a point of Array such as SecurityReqInfo
	   emptyPermissionList := []string{
	           "resources.IID:SystemId",
	           "resources.SecurityReqInfo:Direction", // because can be unused in some CSP
	           "resources.SecurityRuleInfo:CIDR",     // because can be set without soruce CIDR
	   }

	   err = ValidateStruct(reqInfo, emptyPermissionList)
	   if err != nil {
	           cblog.Error(err)
	           return nil, err
	   }
	*/

	vpcSPLock.RLock(connectionName, reqInfo.VpcIID.NameId)
	defer vpcSPLock.RUnlock(connectionName, reqInfo.VpcIID.NameId)

	//+++++++++++++++++++++++++++++++++++++++++++
	// set VPC's SystemId
	var vpcIIDInfo VPCIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*VPCIIDInfo
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, reqInfo.VpcIID.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		vpcIIDInfo = *castedIIDInfo.(*VPCIIDInfo)
	} else {
		err = infostore.GetByConditions(&vpcIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, reqInfo.VpcIID.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}
	reqInfo.VpcIID = getDriverIID(cres.IID{NameId: vpcIIDInfo.NameId, SystemId: vpcIIDInfo.SystemId})
	//+++++++++++++++++++++++++++++++++++++++++++

	// VMGroup.VMs is optional, skip if not provided
	vmList := reqInfo.VMGroup.VMs
	if vmList != nil && len(*vmList) > 0 {
		for idx, vmIID := range *vmList {
			var vmIIDInfo VMIIDInfo
			if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
				var iidInfoList []*VMIIDInfo
				err = getAuthIIDInfoList(connectionName, &iidInfoList)
				if err != nil {
					cblog.Error(err)
					return nil, err
				}
				castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, vmIID.NameId)
				if err != nil {
					cblog.Error(err)
					return nil, err
				}
				vmIIDInfo = *castedIIDInfo.(*VMIIDInfo)
			} else {
				err = infostore.GetByConditions(&vmIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, vmIID.NameId)
				if err != nil {
					cblog.Error(err)
					return nil, err
				}
			}
			(*vmList)[idx] = getDriverIID(cres.IID{NameId: vmIIDInfo.NameId, SystemId: vmIIDInfo.SystemId})
		}
	}
	//+++++++++++++++++++++++++++++++++++++++++++

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateNLBHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// Protocol: to upper
	transformArgsToUpper(&reqInfo)

	nlbSPLock.Lock(connectionName, reqInfo.IId.NameId)
	defer nlbSPLock.Unlock(connectionName, reqInfo.IId.NameId)

	// (1) check exist(NameID)
	// Use original VPC NameId (not driver IID) for duplicate check
	var iidInfoList []*NLBIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		err = infostore.ListByConditions(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName, OWNER_VPC_NAME_COLUMN, vpcIIDInfo.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	} else {
		err = infostore.ListByConditions(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName, OWNER_VPC_NAME_COLUMN, vpcIIDInfo.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}
	var isExist bool = false
	for _, OneIIdInfo := range iidInfoList {
		if OneIIdInfo.NameId == reqInfo.IId.NameId {
			isExist = true
		}
	}

	if isExist {
		err := fmt.Errorf("%s '%s' already exists in connection '%s'", RSTypeString(rsType), reqInfo.IId.NameId, connectionName)
		cblog.Error(err)
		return nil, err
	}

	spUUID := ""
	if GetID_MGMT(IDTransformMode) == "ON" { // Use IID Management
		// (2) generate SP-XID and create reqIID, driverIID
		//     ex) SP-XID {"vm-01-9m4e2mr0ui3e8a215n4g"}
		//
		//     create reqIID: {reqNameID, reqSystemID}   # reqSystemID=SP-XID
		//         ex) reqIID {"seoul-service", "vm-01-9m4e2mr0ui3e8a215n4g"}
		//
		//     create driverIID: {driverNameID, driverSystemID}   # driverNameID=SP-XID, driverSystemID=csp's ID
		//         ex) driverIID {"vm-01-9m4e2mr0ui3e8a215n4g", "i-0bc7123b7e5cbf79d"}
		spUUID, err = iidm.New(connectionName, rsType, reqInfo.IId.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	} else { // No Use IID Management
		spUUID = reqInfo.IId.NameId
	}

	// reqIID
	reqIId := cres.IID{NameId: reqInfo.IId.NameId, SystemId: spUUID}
	// driverIID
	driverIId := cres.IID{NameId: spUUID, SystemId: ""}
	reqInfo.IId = driverIId

	// get Provider Name
	providerName, err := ccm.GetProviderNameByConnectionName(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	// set default configuration of HealthChecker
	setDefaultHealthCheckerConfig(providerName, &reqInfo.HealthChecker)

	// (3) create Resource
	info, err := handler.CreateNLB(reqInfo)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	// Protocol: to upper
	transformArgsToUpper(&info)

	// set VPC NameId
	info.VpcIID.NameId = vpcIIDInfo.NameId

	// set VM's IID with NameId
	info.VMGroup.VMs = reqInfo.VMGroup.VMs

	// (4) create spiderIID: {reqNameID, "driverNameID:driverSystemID"}
	//     ex) spiderIID {"seoul-service", "vm-01-9m4e2mr0ui3e8a215n4g:i-0bc7123b7e5cbf79d"}
	spiderIId := cres.IID{NameId: reqIId.NameId, SystemId: spUUID + ":" + info.IId.SystemId}

	// (5) insert spiderIID
	iidInfo := NLBIIDInfo{ConnectionName: connectionName, NameId: spiderIId.NameId, SystemId: spiderIId.SystemId,
		OwnerVPCName: vpcIIDInfo.NameId}
	err = infostore.Insert(&iidInfo)
	if err != nil {
		cblog.Error(err)
		// rollback
		_, err2 := handler.DeleteNLB(info.IId)
		if err2 != nil {
			cblog.Error(err2)
			return nil, fmt.Errorf(err.Error() + ", " + err2.Error())
		}
		cblog.Error(err)
		return nil, err
	}

	// (6) create userIID: {reqNameID, driverSystemID}
	//     ex) userIID {"seoul-service", "i-0bc7123b7e5cbf79d"}
	info.IId = getUserIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})

	return &info, nil
}

func setDefaultHealthCheckerConfig(providerName string, reqInfo *cres.HealthCheckerInfo) {

	// * -1(int) => set up with spider's default value
	// * Spider's default values for Health Checking
	//	[TCP]  Interval:10 / Timeout:10 / Threshold:3
	//	[HTTP] Interval:10 / Timeout:6 (Azure:10) / Threshold:3
	// * AWS, Azure: disable Timeout Configuration

	// (1) TCP
	if reqInfo.Protocol == "TCP" {
		if reqInfo.Interval == -1 {
			reqInfo.Interval = 10
		}
		if reqInfo.Timeout == -1 {
			if providerName != "AWS" && providerName != "AZURE" {
				reqInfo.Timeout = 10
			}
		}
		if reqInfo.Threshold == -1 {
			reqInfo.Threshold = 3
		}
	}
	// (2) HTTP
	if reqInfo.Protocol == "HTTP" {
		if reqInfo.Interval == -1 {
			reqInfo.Interval = 10
		}
		if reqInfo.Timeout == -1 {
			if providerName != "AWS" && providerName != "AZURE" {
				reqInfo.Timeout = 6
			}
		}
		if reqInfo.Threshold == -1 {
			reqInfo.Threshold = 3
		}
	}
}

func transformArgsToUpper(nlbInfo *cres.NLBInfo) {
	nlbInfo.Type = strings.ToUpper(nlbInfo.Type)
	nlbInfo.Scope = strings.ToUpper(nlbInfo.Scope)

	// ListnerInfo
	nlbInfo.Listener.Protocol = strings.ToUpper(nlbInfo.Listener.Protocol)
	// VMGroupInfo
	nlbInfo.VMGroup.Protocol = strings.ToUpper(nlbInfo.VMGroup.Protocol)
	// HealthCheckerInfo
	nlbInfo.HealthChecker.Protocol = strings.ToUpper(nlbInfo.HealthChecker.Protocol)
}

// (1) get IID:list
// (2) get NLBInfo:list
// (3) set userIID, and ...
func ListNLB(connectionName string, rsType string) ([]*cres.NLBInfo, error) {
	cblog.Info("call ListNLB()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateNLBHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (1) get IID:list
	var iidInfoList []*NLBIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	} else {
		err = infostore.ListByCondition(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}

	var infoList []*cres.NLBInfo
	if iidInfoList == nil || len(iidInfoList) <= 0 {
		infoList = []*cres.NLBInfo{}
		return infoList, nil
	}

	// (2) Get NLBInfo-list with IID-list
	infoList2 := []*cres.NLBInfo{}
	for _, iidInfo := range iidInfoList {

		nlbSPLock.RLock(connectionName, iidInfo.NameId)

		// get resource(SystemId)
		info, err := handler.GetNLB(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}))
		if err != nil {
			nlbSPLock.RUnlock(connectionName, iidInfo.NameId)
			if checkNotFoundError(err) {
				cblog.Error(err)
				info = cres.NLBInfo{IId: cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}}
				infoList2 = append(infoList2, &info)
				continue
			}
			cblog.Error(err)
			return nil, err
		}
		nlbSPLock.RUnlock(connectionName, iidInfo.NameId)

		// Protocol: to upper
		transformArgsToUpper(&info)

		// (3) set ResourceInfo(IID.NameId)
		// set ResourceInfo
		info.IId = getUserIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})

		// set VPC UserIID
		var vpcIIDInfo VPCIIDInfo
		if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
			var iidInfoList []*VPCIIDInfo
			err = getAuthIIDInfoList(connectionName, &iidInfoList)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
			castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, iidInfo.OwnerVPCName)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
			vpcIIDInfo = *castedIIDInfo.(*VPCIIDInfo)
		} else {
			err = infostore.GetByConditions(&vpcIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, iidInfo.OwnerVPCName)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
		}
		info.VpcIID = getUserIID(cres.IID{NameId: vpcIIDInfo.NameId, SystemId: vpcIIDInfo.SystemId})

		// set VM's UserIID
		for idx, vmIID := range *info.VMGroup.VMs {
			var vmIIDInfo VMIIDInfo
			if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
				var iidInfoList []*VMIIDInfo
				err := getAuthIIDInfoList(connectionName, &iidInfoList)
				if err != nil {
					cblog.Error(err)
					return nil, err
				}
				castedIIDInfo, err := getAuthIIDInfoBySystemIdContain(&iidInfoList, vmIID.SystemId)
				if err != nil {
					cblog.Error(err)
					return nil, err
				}
				vmIIDInfo = *castedIIDInfo.(*VMIIDInfo)
			} else {
				err := infostore.GetByContain(&vmIIDInfo, CONNECTION_NAME_COLUMN, connectionName, SYSTEM_ID_COLUMN, vmIID.SystemId)
				if err != nil {
					cblog.Error(err)
					return nil, err
				}
			}
			(*info.VMGroup.VMs)[idx].NameId = vmIIDInfo.NameId
		}

		infoList2 = append(infoList2, &info)
	}

	return infoList2, nil
}

// (1) get IID(NameId)
// (2) get resource(SystemId)
// (3) set ResourceInfo(IID.NameId)
func GetNLB(connectionName string, rsType string, nameID string) (*cres.NLBInfo, error) {
	cblog.Info("call GetNLB()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	nameID, err = EmptyCheckAndTrim("nameID", nameID)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateNLBHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	nlbSPLock.RLock(connectionName, nameID)
	defer nlbSPLock.RUnlock(connectionName, nameID)

	// (1) get IID(NameId)
	var iidInfoList []*NLBIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	} else {
		err = infostore.ListByCondition(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}

	var iidInfo *NLBIIDInfo
	var bool_ret = false
	for _, OneIIdInfo := range iidInfoList {
		if OneIIdInfo.NameId == nameID {
			iidInfo = OneIIdInfo
			bool_ret = true
			break
		}
	}
	if !bool_ret {
		err := fmt.Errorf("%s '%s' does not exist in connection '%s'", RSTypeString(rsType), nameID, connectionName)
		cblog.Error(err)
		return nil, err
	}

	// (2) get resource(SystemId)
	info, err := handler.GetNLB(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}))
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	// Protocol: to upper
	transformArgsToUpper(&info)

	// (3) set ResourceInfo(IID.NameId)
	// set ResourceInfo
	info.IId = getUserIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})

	// set VPC UserIID
	var vpcIIDInfo VPCIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*VPCIIDInfo
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, iidInfo.OwnerVPCName)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		vpcIIDInfo = *castedIIDInfo.(*VPCIIDInfo)
	} else {
		err = infostore.GetByConditions(&vpcIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, iidInfo.OwnerVPCName)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}
	info.VpcIID = getUserIID(cres.IID{NameId: vpcIIDInfo.NameId, SystemId: vpcIIDInfo.SystemId})

	// set VM's UserIID
	for idx, vmIID := range *info.VMGroup.VMs {
		var vmIIDInfo VMIIDInfo
		if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
			var iidInfoList []*VMIIDInfo
			err = getAuthIIDInfoList(connectionName, &iidInfoList)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
			castedIIDInfo, err := getAuthIIDInfoBySystemIdContain(&iidInfoList, vmIID.SystemId)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
			vmIIDInfo = *castedIIDInfo.(*VMIIDInfo)
		} else {
			err := infostore.GetByContain(&vmIIDInfo, CONNECTION_NAME_COLUMN, connectionName, SYSTEM_ID_COLUMN, vmIID.SystemId)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
		}
		(*info.VMGroup.VMs)[idx].NameId = vmIIDInfo.NameId
	}

	return &info, nil
}

// (1) check exist(NameID) and VMs
// (2) add VMs
// (3) Get NLBInfo
// (4) Set ResoureInfo
func AddNLBVMs(connectionName string, nlbName string, vmNames []string) (*cres.NLBInfo, error) {
	cblog.Info("call AddNLBVMs()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	nlbName, err = EmptyCheckAndTrim("nlbName", nlbName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateNLBHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	nlbSPLock.Lock(connectionName, nlbName)
	defer nlbSPLock.Unlock(connectionName, nlbName)

	// (1) check exist(nlbName)
	var iidInfoList []*NLBIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	} else {
		err = infostore.ListByCondition(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}

	var iidInfo *NLBIIDInfo
	var bool_ret = false
	for _, OneIIdInfo := range iidInfoList {
		if OneIIdInfo.NameId == nlbName {
			iidInfo = OneIIdInfo
			bool_ret = true
			break
		}
	}
	if bool_ret == false {
		err := fmt.Errorf("%s '%s' does not exist in connection '%s'", RSTypeString(NLB), nlbName, connectionName)
		cblog.Error(err)
		return nil, err
	}

	// (2) add VMs
	// driverIID for driver
	var vmIIDs []cres.IID
	for _, one := range vmNames {
		// check vm existence
		bool_ret := false
		if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
			var iidInfoList []*VMIIDInfo
			err = getAuthIIDInfoList(connectionName, &iidInfoList)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
			bool_ret, err = isNameIdExists(&iidInfoList, one)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
		} else {
			bool_ret, err = infostore.HasByConditions(&VMIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, one)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
		}
		if bool_ret == false {
			err := fmt.Errorf("%s '%s' does not exist in connection '%s'", RSTypeString(VM), one, connectionName)
			cblog.Error(err)
			return nil, err
		}

		var vmIIDInfo VMIIDInfo
		if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
			var iidInfoList []*VMIIDInfo
			err = getAuthIIDInfoList(connectionName, &iidInfoList)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
			castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, one)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
			vmIIDInfo = *castedIIDInfo.(*VMIIDInfo)
		} else {
			err = infostore.GetByConditions(&vmIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, one)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
		}
		vmIID := getDriverIID(cres.IID{NameId: vmIIDInfo.NameId, SystemId: vmIIDInfo.SystemId})

		vmIIDs = append(vmIIDs, vmIID)
	}

	_, err = handler.AddVMs(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}), &vmIIDs)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (3) Get NLBInfo
	info, err := handler.GetNLB(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}))
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// Protocol: to upper
	transformArgsToUpper(&info)

	// (4) set ResourceInfo(userIID)
	info.IId = getUserIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})

	// set VPC UserIID
	var vpcIIDInfo VPCIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*VPCIIDInfo
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, iidInfo.OwnerVPCName)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		vpcIIDInfo = *castedIIDInfo.(*VPCIIDInfo)
	} else {
		err = infostore.GetByConditions(&vpcIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, iidInfo.OwnerVPCName)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}
	info.VpcIID = getUserIID(cres.IID{NameId: vpcIIDInfo.NameId, SystemId: vpcIIDInfo.SystemId})

	// set VM's UserIID
	for idx, vmIID := range *info.VMGroup.VMs {
		var vmIIDInfo VMIIDInfo
		if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
			var iidInfoList []*VMIIDInfo
			err := getAuthIIDInfoList(connectionName, &iidInfoList)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
			castedIIDInfo, err := getAuthIIDInfoBySystemIdContain(&iidInfoList, vmIID.SystemId)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
			vmIIDInfo = *castedIIDInfo.(*VMIIDInfo)
		} else {
			err := infostore.GetByContain(&vmIIDInfo, CONNECTION_NAME_COLUMN, connectionName, SYSTEM_ID_COLUMN, vmIID.SystemId)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
		}
		(*info.VMGroup.VMs)[idx].NameId = vmIIDInfo.NameId
	}

	return &info, nil
}

// (1) check exist(NameID)
// (2) remove VMs
func RemoveNLBVMs(connectionName string, nlbName string, vmNames []string) (bool, error) {
	cblog.Info("call RemoveNLBVMs()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	nlbName, err = EmptyCheckAndTrim("nlbName", nlbName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	handler, err := cldConn.CreateNLBHandler()
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	nlbSPLock.Lock(connectionName, nlbName)
	defer nlbSPLock.Unlock(connectionName, nlbName)

	// (1) check exist(nlbName)
	var iidInfoList []*NLBIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
	} else {
		err = infostore.ListByCondition(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
	}

	var iidInfo *NLBIIDInfo
	var bool_ret = false
	for _, OneIIdInfo := range iidInfoList {
		if OneIIdInfo.NameId == nlbName {
			iidInfo = OneIIdInfo
			bool_ret = true
			break
		}
	}
	if bool_ret == false {
		err := fmt.Errorf("%s '%s' does not exist in connection '%s'", RSTypeString(NLB), nlbName, connectionName)
		cblog.Error(err)
		return false, err
	}

	// (2) remove VMs
	// driverIID for driver
	var vmIIDs []cres.IID
	for _, one := range vmNames {
		var vmIIDInfo VMIIDInfo
		if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
			var iidInfoList []*VMIIDInfo
			err = getAuthIIDInfoList(connectionName, &iidInfoList)
			if err != nil {
				cblog.Error(err)
				return false, err
			}
			castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, one)
			if err != nil {
				cblog.Error(err)
				return false, err
			}
			vmIIDInfo = *castedIIDInfo.(*VMIIDInfo)
		} else {
			err = infostore.GetByConditions(&vmIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, one)
			if err != nil {
				cblog.Error(err)
				return false, err
			}
		}
		vmIID := getDriverIID(cres.IID{NameId: vmIIDInfo.NameId, SystemId: vmIIDInfo.SystemId})

		vmIIDs = append(vmIIDs, vmIID)
	}

	result, err := handler.RemoveVMs(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}), &vmIIDs)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	return result, nil
}

// ---------------------------------------------------//
// @todo  To support or not will be decided later.   //
// ---------------------------------------------------//
// (1) check exist(NameID)
// (2) change listener
// (3) Get NLBInfo
// (4) Set ResoureInfo
func ChangeListener(connectionName string, nlbName string, listener cres.ListenerInfo) (*cres.NLBInfo, error) {
	cblog.Info("call ChangeListener()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	nlbName, err = EmptyCheckAndTrim("nlbName", nlbName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	emptyPermissionList := []string{
		"resources.IID:SystemId",
		"resources.ListenerInfo:IP",
		"resources.ListenerInfo:DNSName",
		"resources.ListenerInfo:CspID", // because can be unused in some CSP
	}
	err = ValidateStruct(listener, emptyPermissionList)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateNLBHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	nlbSPLock.Lock(connectionName, nlbName)
	defer nlbSPLock.Unlock(connectionName, nlbName)

	// (1) check exist(nlbName)
	var iidInfoList []*NLBIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	} else {
		err = infostore.ListByCondition(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}

	var iidInfo *NLBIIDInfo
	var bool_ret = false
	for _, OneIIdInfo := range iidInfoList {
		if OneIIdInfo.NameId == nlbName {
			iidInfo = OneIIdInfo
			bool_ret = true
			break
		}
	}
	if bool_ret == false {
		err := fmt.Errorf("%s '%s' does not exist in connection '%s'", RSTypeString(NLB), nlbName, connectionName)
		cblog.Error(err)
		return nil, err
	}

	// (2) change listener
	// driverIID for driver
	_, err = handler.ChangeListener(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}), listener)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (3) Get NLBInfo
	info, err := handler.GetNLB(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}))
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// Protocol: to upper
	transformArgsToUpper(&info)

	// (4) set ResourceInfo(userIID)
	info.IId = getUserIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})

	// set VPC UserIID
	var vpcIIDInfo VPCIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*VPCIIDInfo
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, iidInfo.OwnerVPCName)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		vpcIIDInfo = *castedIIDInfo.(*VPCIIDInfo)
	} else {
		err = infostore.GetByConditions(&vpcIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, iidInfo.OwnerVPCName)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}
	info.VpcIID = getUserIID(cres.IID{NameId: vpcIIDInfo.NameId, SystemId: vpcIIDInfo.SystemId})

	// set VM's UserIID
	for idx, vmIID := range *info.VMGroup.VMs {
		var vmIIDInfo VMIIDInfo
		if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
			var iidInfoList []*VMIIDInfo
			err := getAuthIIDInfoList(connectionName, &iidInfoList)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
			castedIIDInfo, err := getAuthIIDInfoBySystemIdContain(&iidInfoList, vmIID.SystemId)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
			vmIIDInfo = *castedIIDInfo.(*VMIIDInfo)
		} else {
			err := infostore.GetByContain(&vmIIDInfo, CONNECTION_NAME_COLUMN, connectionName, SYSTEM_ID_COLUMN, vmIID.SystemId)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
		}
		(*info.VMGroup.VMs)[idx].NameId = vmIIDInfo.NameId
	}

	return &info, nil
}

// ---------------------------------------------------//
// @todo  To support or not will be decided later.   //
// ---------------------------------------------------//
// (1) check exist(NameID)
// (2) change VMGroup
// (3) Get NLBInfo
// (4) Set ResoureInfo
func ChangeVMGroup(connectionName string, nlbName string, vmGroup cres.VMGroupInfo) (*cres.NLBInfo, error) {
	cblog.Info("call ChangeVMGroup()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	nlbName, err = EmptyCheckAndTrim("nlbName", nlbName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// @todo
	/* Currently, Validator does not support the struct has a point of Array such as SecurityReqInfo
	   emptyPermissionList := []string{
	           "resources.IID:SystemId",
	           "resources.ListenerInfo:CspID", // because can be unused in some CSP
	   }
	   err = ValidateStruct(listener, emptyPermissionList)
	   if err != nil {
	           cblog.Error(err)
	           return nil, err
	   }
	*/

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateNLBHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	nlbSPLock.Lock(connectionName, nlbName)
	defer nlbSPLock.Unlock(connectionName, nlbName)

	// (1) check exist(nlbName)
	var iidInfoList []*NLBIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	} else {
		err = infostore.ListByCondition(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}

	var iidInfo *NLBIIDInfo
	var bool_ret = false
	for _, OneIIdInfo := range iidInfoList {
		if OneIIdInfo.NameId == nlbName {
			iidInfo = OneIIdInfo
			bool_ret = true
			break
		}
	}
	if bool_ret == false {
		err := fmt.Errorf("%s '%s' does not exist in connection '%s'", RSTypeString(NLB), nlbName, connectionName)
		cblog.Error(err)
		return nil, err
	}

	// (2) change VMGroup
	// driverIID for driver
	_, err = handler.ChangeVMGroupInfo(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}), vmGroup)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (3) Get NLBInfo
	info, err := handler.GetNLB(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}))
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// Protocol: to upper
	transformArgsToUpper(&info)

	// (4) set ResourceInfo(userIID)
	info.IId = getUserIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})

	// set VM's UserIID
	for idx, vmIID := range *info.VMGroup.VMs {
		var vmIIDInfo VMIIDInfo
		if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
			var iidInfoList []*VMIIDInfo
			err := getAuthIIDInfoList(connectionName, &iidInfoList)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
			castedIIDInfo, err := getAuthIIDInfoBySystemIdContain(&iidInfoList, vmIID.SystemId)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
			vmIIDInfo = *castedIIDInfo.(*VMIIDInfo)
		} else {
			err := infostore.GetByContain(&vmIIDInfo, CONNECTION_NAME_COLUMN, connectionName, SYSTEM_ID_COLUMN, vmIID.SystemId)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
		}
		(*info.VMGroup.VMs)[idx].NameId = vmIIDInfo.NameId
	}

	// set VPC SystemId
	var vpcIIDInfo VPCIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*VPCIIDInfo
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, iidInfo.OwnerVPCName)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		vpcIIDInfo = *castedIIDInfo.(*VPCIIDInfo)
	} else {
		err = infostore.GetByConditions(&vpcIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, iidInfo.OwnerVPCName)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}
	info.VpcIID = getUserIID(cres.IID{NameId: vpcIIDInfo.NameId, SystemId: vpcIIDInfo.SystemId})

	return &info, nil
}

// ---------------------------------------------------//
// @todo  To support or not will be decided later.   //
// ---------------------------------------------------//
// (1) check exist(NameID)
// (2) change HealthCheckerInfo
// (3) Get NLBInfo
// (4) Set ResoureInfo
func ChangeHealthChecker(connectionName string, nlbName string, healthChecker cres.HealthCheckerInfo) (*cres.NLBInfo, error) {
	cblog.Info("call ChangeHealthChecker()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	nlbName, err = EmptyCheckAndTrim("nlbName", nlbName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	emptyPermissionList := []string{
		"resources.IID:SystemId",
		"resources.HealthCheckerInfo:CspID", // because can be unused in some CSP
	}
	err = ValidateStruct(healthChecker, emptyPermissionList)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateNLBHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	nlbSPLock.Lock(connectionName, nlbName)
	defer nlbSPLock.Unlock(connectionName, nlbName)

	// (1) check exist(nlbName)
	var iidInfoList []*NLBIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	} else {
		err = infostore.ListByCondition(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}

	var iidInfo *NLBIIDInfo
	var bool_ret = false
	for _, OneIIdInfo := range iidInfoList {
		if OneIIdInfo.NameId == nlbName {
			iidInfo = OneIIdInfo
			bool_ret = true
			break
		}
	}
	if bool_ret == false {
		err := fmt.Errorf("%s '%s' does not exist in connection '%s'", RSTypeString(NLB), nlbName, connectionName)
		cblog.Error(err)
		return nil, err
	}

	// (2) change VMGroup
	// driverIID for driver
	_, err = handler.ChangeHealthCheckerInfo(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}), healthChecker)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (3) Get NLBInfo
	info, err := handler.GetNLB(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}))
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// Protocol: to upper
	transformArgsToUpper(&info)

	// (4) set ResourceInfo(userIID)
	info.IId = getUserIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})

	// set VPC UserIID
	var vpcIIDInfo VPCIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*VPCIIDInfo
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, iidInfo.OwnerVPCName)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		vpcIIDInfo = *castedIIDInfo.(*VPCIIDInfo)
	} else {
		err = infostore.GetByConditions(&vpcIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, iidInfo.OwnerVPCName)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}
	info.VpcIID = getUserIID(cres.IID{NameId: vpcIIDInfo.NameId, SystemId: vpcIIDInfo.SystemId})

	// set VM's UserIID
	for idx, vmIID := range *info.VMGroup.VMs {
		var vmIIDInfo VMIIDInfo
		if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
			var iidInfoList []*VMIIDInfo
			err := getAuthIIDInfoList(connectionName, &iidInfoList)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
			castedIIDInfo, err := getAuthIIDInfoBySystemIdContain(&iidInfoList, vmIID.SystemId)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
			vmIIDInfo = *castedIIDInfo.(*VMIIDInfo)
		} else {
			err := infostore.GetByContain(&vmIIDInfo, CONNECTION_NAME_COLUMN, connectionName, SYSTEM_ID_COLUMN, vmIID.SystemId)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
		}
		(*info.VMGroup.VMs)[idx].NameId = vmIIDInfo.NameId
	}

	return &info, nil
}

// (1) check exist(NameID)
// (2) Get HealthInfo
// (3) Get NLBInfo
// (4) Set ResoureInfo
func GetVMGroupHealthInfo(connectionName string, nlbName string) (*cres.HealthInfo, error) {
	cblog.Info("call GetVMGroupHealthInfo()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	nlbName, err = EmptyCheckAndTrim("nlbName", nlbName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// @todo
	/* Currently, Validator does not support the struct has a point of Array such as SecurityReqInfo
	   emptyPermissionList := []string{
	           "resources.IID:SystemId",
	           "resources.ListenerInfo:CspID", // because can be unused in some CSP
	   }
	   err = ValidateStruct(healthChecker, emptyPermissionList)
	   if err != nil {
	           cblog.Error(err)
	           return nil, err
	   }
	*/

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateNLBHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	nlbSPLock.Lock(connectionName, nlbName)
	defer nlbSPLock.Unlock(connectionName, nlbName)

	// (1) check exist(nlbName)
	var iidInfoList []*NLBIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	} else {
		err = infostore.ListByCondition(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}

	var iidInfo *NLBIIDInfo
	var bool_ret = false
	for _, OneIIdInfo := range iidInfoList {
		if OneIIdInfo.NameId == nlbName {
			iidInfo = OneIIdInfo
			bool_ret = true
			break
		}
	}
	if bool_ret == false {
		err := fmt.Errorf("%s '%s' does not exist in connection '%s'", RSTypeString(NLB), nlbName, connectionName)
		cblog.Error(err)
		return nil, err
	}

	// (2) change VMGroup
	// driverIID for driver
	healthInfo, err := handler.GetVMGroupHealthInfo(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}))
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (3) set VM User IID with driver SystemId
	err = setVMUserIIDwithSystemId(connectionName, nlbName, &healthInfo)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (4)
	return &healthInfo, nil
}

func setVMUserIIDwithSystemId(connectionName string, nlbName string, healthInfo *cres.HealthInfo) error {
	var errList []string

	// If AllVMs is nil or empty, return early (no VMs to process)
	if healthInfo.AllVMs == nil || len(*healthInfo.AllVMs) == 0 {
		return nil
	}

	vmIIDList := healthInfo.AllVMs
	for idx, vm := range *vmIIDList {
		foundFlag := false
		var vmIIDInfo VMIIDInfo
		if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
			var iidInfoList []*VMIIDInfo
			err := getAuthIIDInfoList(connectionName, &iidInfoList)
			if err != nil {
				cblog.Error(err)
				return err
			}
			castedIIDInfo, err := getAuthIIDInfoBySystemIdContain(&iidInfoList, vm.SystemId)
			if err != nil {
				cblog.Error(err)
				return err
			}
			vmIIDInfo = *castedIIDInfo.(*VMIIDInfo)
		} else {
			err := infostore.GetByContain(&vmIIDInfo, CONNECTION_NAME_COLUMN, connectionName, SYSTEM_ID_COLUMN, vm.SystemId)
			if err != nil {
				cblog.Error(err)
				return err
			}
		}
		if vm.SystemId == getDriverSystemId(cres.IID{NameId: vmIIDInfo.NameId, SystemId: vmIIDInfo.SystemId}) {
			foundFlag = true
			(*vmIIDList)[idx].NameId = vmIIDInfo.NameId
		}
		if !foundFlag {
			errList = append(errList, connectionName+":CSP-VM:"+vm.SystemId+" is not owned by CB-Spider!")
		}
	}

	vmIIDList = healthInfo.HealthyVMs
	for idx, vm := range *vmIIDList {
		foundFlag := false
		var vmIIDInfo VMIIDInfo
		if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
			var iidInfoList []*VMIIDInfo
			err := getAuthIIDInfoList(connectionName, &iidInfoList)
			if err != nil {
				cblog.Error(err)
				return err
			}
			castedIIDInfo, err := getAuthIIDInfoBySystemIdContain(&iidInfoList, vm.SystemId)
			if err != nil {
				cblog.Error(err)
				return err
			}
			vmIIDInfo = *castedIIDInfo.(*VMIIDInfo)
		} else {
			err := infostore.GetByContain(&vmIIDInfo, CONNECTION_NAME_COLUMN, connectionName, SYSTEM_ID_COLUMN, vm.SystemId)
			if err != nil {
				cblog.Error(err)
				return err
			}
		}
		if vm.SystemId == getDriverSystemId(cres.IID{NameId: vmIIDInfo.NameId, SystemId: vmIIDInfo.SystemId}) {
			foundFlag = true
			(*vmIIDList)[idx].NameId = vmIIDInfo.NameId
		}
		if !foundFlag {
			errList = append(errList, connectionName+":CSP-VM:"+vm.SystemId+" is not owned by CB-Spider!")
		}
	}

	vmIIDList = healthInfo.UnHealthyVMs
	for idx, vm := range *vmIIDList {
		foundFlag := false
		var vmIIDInfo VMIIDInfo
		if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
			var iidInfoList []*VMIIDInfo
			err := getAuthIIDInfoList(connectionName, &iidInfoList)
			if err != nil {
				cblog.Error(err)
				return err
			}
			castedIIDInfo, err := getAuthIIDInfoBySystemIdContain(&iidInfoList, vm.SystemId)
			if err != nil {
				cblog.Error(err)
				return err
			}
			vmIIDInfo = *castedIIDInfo.(*VMIIDInfo)
		} else {
			err := infostore.GetByContain(&vmIIDInfo, CONNECTION_NAME_COLUMN, connectionName, SYSTEM_ID_COLUMN, vm.SystemId)
			if err != nil {
				cblog.Error(err)
				return err
			}
		}
		if vm.SystemId == getDriverSystemId(cres.IID{NameId: vmIIDInfo.NameId, SystemId: vmIIDInfo.SystemId}) {
			foundFlag = true
			(*vmIIDList)[idx].NameId = vmIIDInfo.NameId
		}
		if !foundFlag {
			errList = append(errList, connectionName+":CSP-VM:"+vm.SystemId+" is not owned by CB-Spider!")
		}
	}

	// check error existence
	if len(errList) > 0 {
		cblog.Error(strings.Join(errList, "\n"))
		return errors.New(strings.Join(errList, "\n"))
	}

	return nil
}

// (1) get spiderIID
// (2) delete Resource(SystemId)
// (3) delete IID
func DeleteNLB(connectionName string, rsType string, nameID string, force string) (bool, error) {
	cblog.Info("call DeleteNLB()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	nameID, err = EmptyCheckAndTrim("nameID", nameID)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	handler, err := cldConn.CreateNLBHandler()
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	nlbSPLock.Lock(connectionName, nameID)
	defer nlbSPLock.Unlock(connectionName, nameID)

	// (1) get spiderIID for creating driverIID
	var iidInfoList []*NLBIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
	} else {
		err = infostore.ListByCondition(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
	}

	var iidInfo *NLBIIDInfo
	var bool_ret = false
	for _, OneIIdInfo := range iidInfoList {
		if OneIIdInfo.NameId == nameID {
			iidInfo = OneIIdInfo
			bool_ret = true
			break
		}
	}
	if !bool_ret {
		err := fmt.Errorf("%s '%s' does not exist in connection '%s'", RSTypeString(rsType), nameID, connectionName)
		cblog.Error(err)
		return false, err
	}

	// (2) delete Resource(SystemId)
	driverIId := getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})
	result := false

	result, err = handler.(cres.NLBHandler).DeleteNLB(driverIId)
	if err != nil {
		cblog.Error(err)
		if checkNotFoundError(err) {
			// if not found in CSP, continue
			force = "true"
		} else if force != "true" {
			return false, err
		}
	}

	if force != "true" {
		if result == false {
			return result, nil
		}
	}

	// (3) delete IID
	_, err = infostore.DeleteByConditions(&NLBIIDInfo{}, CONNECTION_NAME_COLUMN, iidInfo.ConnectionName, NAME_ID_COLUMN, nameID)
	if err != nil {
		cblog.Error(err)
		if force != "true" {
			return false, err
		}
	}

	return result, nil
}

func CountAllNLBs() (int64, error) {
	var info NLBIIDInfo
	count, err := infostore.CountAllNameIDs(&info)
	if err != nil {
		cblog.Error(err)
		return count, err
	}

	return count, nil
}

func CountNLBsByConnection(connectionName string) (int64, error) {
	var info NLBIIDInfo
	count, err := infostore.CountNameIDsByConnection(&info, connectionName)
	if err != nil {
		cblog.Error(err)
		return count, err
	}

	return count, nil
}
