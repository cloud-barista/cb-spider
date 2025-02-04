// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.

package commonruntime

import (
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

type SGIIDInfo VPCDependentIIDInfo

func (SGIIDInfo) TableName() string {
	return "sg_iid_infos"
}

//====================================================================

func init() {
	db, err := infostore.Open()
	if err != nil {
		cblog.Error(err)
		return
	}
	db.AutoMigrate(&SGIIDInfo{})
	infostore.Close(db)
}

//================ SecurityGroup Handler

func GetSGOwnerVPC(connectionName string, cspID string) (owerVPC cres.IID, err error) {
	cblog.Info("call GetSGOwnerVPC()")

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

	rsType := SG

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return cres.IID{}, err
	}

	handler, err := cldConn.CreateSecurityHandler()
	if err != nil {
		cblog.Error(err)
		return cres.IID{}, err
	}

	// Except Management API
	//sgSPLock.RLock()
	//vpcSPLock.RLock()

	// (1) check existence(cspID)
	var iidInfoList []*SGIIDInfo
	err = infostore.ListByCondition(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName)
	if err != nil {
		//vpcSPLock.RUnlock()
		//sgSPLock.RUnlock()
		cblog.Error(err)
		return cres.IID{}, err
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
		//sgSPLock.RUnlock()
		err := fmt.Errorf("%s", rsType+"-"+cspID+" already exists with "+nameId+"!")
		cblog.Error(err)
		return cres.IID{}, err
	}

	// (2) get resource info(CSP-ID)
	// check existence and get info of this resouce in the CSP
	// Do not user NameId, because Azure driver use it like SystemId
	getInfo, err := handler.GetSecurity(cres.IID{NameId: getMSShortID(cspID), SystemId: cspID})
	if err != nil {
		//vpcSPLock.RUnlock()
		//sgSPLock.RUnlock()
		cblog.Error(err)
		return cres.IID{}, err
	}

	// (3) get VPC IID:list
	var vpcIIDInfoList []*VPCIIDInfo
	err = infostore.ListByCondition(&vpcIIDInfoList, CONNECTION_NAME_COLUMN, connectionName)
	if err != nil {
		//vpcSPLock.RUnlock()
		//sgSPLock.RUnlock()
		cblog.Error(err)
		return cres.IID{}, err
	}
	//vpcSPLock.RUnlock()
	//sgSPLock.RUnlock()

	//--------
	//-------- ex) spiderIID {"vpc-01", "vpc-01-9m4e2mr0ui3e8a215n4g:i-0bc7123b7e5cbf79d"}
	//--------
	// Do not user NameId, because Azure driver use it like SystemId
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
func RegisterSecurity(connectionName string, vpcUserID string, userIID cres.IID) (*cres.SecurityInfo, error) {
	cblog.Info("call RegisterSecurity()")

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

	rsType := SG

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateSecurityHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	vpcSPLock.Lock(connectionName, vpcUserID)
	defer vpcSPLock.Unlock(connectionName, vpcUserID)
	sgSPLock.Lock(connectionName, userIID.NameId)
	defer sgSPLock.Unlock(connectionName, userIID.NameId)

	// (0) check VPC existence(VPC UserID)
	bool_ret, err := infostore.HasByConditions(&VPCIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, vpcUserID)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	if !bool_ret {
		err := fmt.Errorf("The %s '%s' does not exist!", RSTypeString(VPC), vpcUserID)
		cblog.Error(err)
		return nil, err
	}

	// (1) check existence(UserID)
	isExist, err := infostore.HasByConditions(&SGIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, userIID.NameId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	if isExist {
		err := fmt.Errorf("%s", rsType+"-"+userIID.NameId+" already exists!")
		cblog.Error(err)
		return nil, err
	}

	// (2) get resource info(CSP-ID)
	// check existence and get info of this resouce in the CSP
	// Do not user NameId, because Azure driver use it like SystemId
	getInfo, err := handler.GetSecurity(cres.IID{NameId: getMSShortID(userIID.SystemId), SystemId: userIID.SystemId})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	// Direction: to lower
	// IPProtocol: to upper
	// no CIDR: "0.0.0.0/0"
	transformArgs(getInfo.SecurityRules)

	// (3) create spiderIID: {UserID, SP-XID:CSP-ID}
	//     ex) spiderIID {"vpc-01", "vpc-01-9m4e2mr0ui3e8a215n4g:i-0bc7123b7e5cbf79d"}
	// Do not user NameId, because Azure driver use it like SystemId
	systemId := getMSShortID(getInfo.IId.SystemId)
	spiderIId := cres.IID{NameId: userIID.NameId, SystemId: systemId + ":" + getInfo.IId.SystemId}

	// (4) insert spiderIID
	// insert SecurityGroup SpiderIID to metadb
	err = infostore.Insert(&SGIIDInfo{ConnectionName: connectionName,
		NameId: spiderIId.NameId, SystemId: spiderIId.SystemId, OwnerVPCName: vpcUserID})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// set up SecurityGroup User IID for return info
	getInfo.IId = userIID

	// set up VPC UserIID for return info
	var iidInfo VPCIIDInfo
	err = infostore.GetByConditions(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, vpcUserID)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	getInfo.VpcIID = getUserIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})

	return &getInfo, nil
}

// (1) check exist(NameID)
// (2) generate SP-XID and create reqIID, driverIID
// (3) create Resource
// (4) create spiderIID: {reqNameID, "driverNameID:driverSystemID"}
// (5) insert spiderIID
// (6) create userIID
func CreateSecurity(connectionName string, rsType string, reqInfo cres.SecurityReqInfo, IDTransformMode string) (*cres.SecurityInfo, error) {
	cblog.Info("call CreateSecurity()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

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

	vpcSPLock.Lock(connectionName, reqInfo.VpcIID.NameId)
	defer vpcSPLock.Unlock(connectionName, reqInfo.VpcIID.NameId)

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

	reqInfo.VpcIID.SystemId = getDriverSystemId(cres.IID{NameId: vpcIIDInfo.NameId, SystemId: vpcIIDInfo.SystemId})
	//+++++++++++++++++++++++++++++++++++++++++++

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateSecurityHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// Direction: to lower
	// IPProtocol: to upper
	// no CIDR: "0.0.0.0/0"
	transformArgs(reqInfo.SecurityRules)

	sgSPLock.Lock(connectionName, reqInfo.IId.NameId)
	defer sgSPLock.Unlock(connectionName, reqInfo.IId.NameId)
	// (1) check exist(NameID)
	var iidInfoList []*SGIIDInfo
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
	var isExist bool = false
	for _, OneIIdInfo := range iidInfoList {
		if OneIIdInfo.NameId == reqInfo.IId.NameId {
			isExist = true
		}
	}

	if isExist {
		err := fmt.Errorf("%s", rsType+"-"+reqInfo.IId.NameId+" already exists!")
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

	// (3) create Resource
	info, err := handler.CreateSecurity(reqInfo)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	// Direction: to lower
	// IPProtocol: to upper
	// no CIDR: "0.0.0.0/0"
	transformArgs(info.SecurityRules)

	// set VPC NameId
	info.VpcIID.NameId = reqInfo.VpcIID.NameId

	// (4) create spiderIID: {reqNameID, "driverNameID:driverSystemID"}
	//     ex) spiderIID {"seoul-service", "vm-01-9m4e2mr0ui3e8a215n4g:i-0bc7123b7e5cbf79d"}
	spiderIId := cres.IID{NameId: reqIId.NameId, SystemId: spUUID + ":" + info.IId.SystemId}

	// (5) insert spiderIID
	err = infostore.Insert(&SGIIDInfo{ConnectionName: connectionName,
		NameId: spiderIId.NameId, SystemId: spiderIId.SystemId, OwnerVPCName: reqInfo.VpcIID.NameId})
	if err != nil {
		cblog.Error(err)
		// rollback
		_, err2 := handler.DeleteSecurity(info.IId)
		if err2 != nil {
			cblog.Error(err2)
			return nil, fmt.Errorf("%s", err.Error()+", "+err2.Error())
		}
		cblog.Error(err)
		return nil, err
	}

	// (6) create userIID: {reqNameID, driverSystemID}
	//     ex) userIID {"seoul-service", "i-0bc7123b7e5cbf79d"}
	// info.IId = getUserIID(iidInfo.IId)
	info.IId = cres.IID{NameId: reqIId.NameId, SystemId: info.IId.SystemId}

	// set VPC SystemId
	info.VpcIID.SystemId = getDriverSystemId(cres.IID{NameId: vpcIIDInfo.NameId, SystemId: vpcIIDInfo.SystemId})

	return &info, nil
}

func transformArgs(ruleList *[]cres.SecurityRuleInfo) {
	for n := range *ruleList {
		// Direction: to lower => inbound | outbound
		(*ruleList)[n].Direction = strings.ToLower((*ruleList)[n].Direction)
		// IPProtocol: to upper => ALL | TCP | UDP | ICMP
		(*ruleList)[n].IPProtocol = strings.ToUpper((*ruleList)[n].IPProtocol)
		// no CIDR, set default ("0.0.0.0/0")
		if (*ruleList)[n].CIDR == "" {
			(*ruleList)[n].CIDR = "0.0.0.0/0"
		}
	}
}

// (1) get IID:list
// (2) get SecurityInfo:list
// (3) set userIID, and ...
func ListSecurity(connectionName string, rsType string) ([]*cres.SecurityInfo, error) {
	cblog.Info("call ListSecurity()")

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

	handler, err := cldConn.CreateSecurityHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (1) get IID:list
	var iidInfoList []*SGIIDInfo
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

	infoList := []*cres.SecurityInfo{}
	if iidInfoList == nil || len(iidInfoList) <= 0 {
		return infoList, nil
	}

	// (2) Get SecurityInfo-list with IID-list
	infoList2 := []*cres.SecurityInfo{}
	for _, iidInfo := range iidInfoList {

		sgSPLock.RLock(connectionName, iidInfo.NameId)

		// get resource(SystemId)
		info, err := handler.GetSecurity(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}))
		if err != nil {
			sgSPLock.RUnlock(connectionName, iidInfo.NameId)
			if checkNotFoundError(err) {
				cblog.Info(err)
				continue
			}
			cblog.Error(err)
			return nil, err
		}
		sgSPLock.RUnlock(connectionName, iidInfo.NameId)
		// Direction: to lower
		// IPProtocol: to upper
		// no CIDR: "0.0.0.0/0"
		transformArgs(info.SecurityRules)

		// (3) set ResourceInfo(IID.NameId)
		// set ResourceInfo
		info.IId = getUserIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})

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

		infoList2 = append(infoList2, &info)
	}

	return infoList2, nil
}

// (1) get IID of Security Group for typhical VPC:list
// (2) get SecurityInfo:list
// (3) set userIID, and ...
func ListVpcSecurity(connectionName, rsType, vpcName string) ([]*cres.SecurityInfo, error) {
	cblog.Info("call ListVpcSecurity()")

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

	handler, err := cldConn.CreateSecurityHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	//(1) get IId of SG for typhical vpc -> iidInfoList
	var iidInfoList []*SGIIDInfo
	err = infostore.ListByConditions(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName, OWNER_VPC_NAME_COLUMN, vpcName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	//(2) Get Security Group list with iidInfoList
	infoList := []*cres.SecurityInfo{}
	for _, iidInfo := range iidInfoList {
		sgSPLock.RLock(connectionName, iidInfo.NameId)
		defer sgSPLock.RUnlock(connectionName, iidInfo.NameId)

		info, err := handler.GetSecurity(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}))

		if err != nil {
			if checkNotFoundError(err) {
				cblog.Info(err)
				continue
			}
			cblog.Error(err)
			return nil, err
		}

		//Transform security rules
		transformArgs(info.SecurityRules)

		// Set resource info
		info.IId = getUserIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})

		// Set VPC SystemId
		var vpcIIDInfo VPCIIDInfo
		err = infostore.GetByConditions(&vpcIIDInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, iidInfo.OwnerVPCName)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		info.VpcIID = getUserIID(cres.IID{NameId: vpcIIDInfo.NameId, SystemId: vpcIIDInfo.SystemId})

		infoList = append(infoList, &info)
	}

	return infoList, nil
}

// (1) get IID(NameId)
// (2) get resource(SystemId)
// (3) set ResourceInfo(IID.NameId)
func GetSecurity(connectionName string, rsType string, nameID string) (*cres.SecurityInfo, error) {
	cblog.Info("call GetSecurity()")

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

	handler, err := cldConn.CreateSecurityHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	sgSPLock.RLock(connectionName, nameID)
	defer sgSPLock.RUnlock(connectionName, nameID)

	// (1) get IID(NameId)
	var iidInfo SGIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*SGIIDInfo
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, nameID)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		iidInfo = *castedIIDInfo.(*SGIIDInfo)
	} else {
		err = infostore.GetByConditions(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}

	// (2) get resource(SystemId)
	info, err := handler.GetSecurity(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}))
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	// Direction: to lower
	// IPProtocol: to upper
	// no CIDR: "0.0.0.0/0"
	transformArgs(info.SecurityRules)

	// (3) set ResourceInfo(IID.NameId)
	// set ResourceInfo
	info.IId = getUserIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})

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

// (1) check exist(NameID)
// (2) add Rules
func AddRules(connectionName string, sgName string, reqInfoList []cres.SecurityRuleInfo) (*cres.SecurityInfo, error) {
	cblog.Info("call AddRules()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	sgName, err = EmptyCheckAndTrim("sgName", sgName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateSecurityHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// Direction: to lower
	// IPProtocol: to upper
	// no CIDR: "0.0.0.0/0"
	transformArgs(&reqInfoList)

	sgSPLock.Lock(connectionName, sgName)
	defer sgSPLock.Unlock(connectionName, sgName)

	// (1) check exist(sgName)
	bool_ret := false
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*SGIIDInfo
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		bool_ret, err = isNameIdExists(&iidInfoList, sgName)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	} else {
		bool_ret, err = infostore.HasByConditions(&SGIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, sgName)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}
	if !bool_ret {
		err := fmt.Errorf("The %s '%s' does not exist!", RSTypeString(SG), sgName)
		cblog.Error(err)
		return nil, err
	}

	var iidInfo SGIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*SGIIDInfo
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, sgName)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		iidInfo = *castedIIDInfo.(*SGIIDInfo)
	} else {
		err = infostore.GetByConditions(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, sgName)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}

	// (2) add Rules
	// driverIID for driver
	info, err := handler.AddRules(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}), &reqInfoList)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	// Direction: to lower
	// IPProtocol: to upper
	// no CIDR: "0.0.0.0/0"
	transformArgs(info.SecurityRules)

	// (3) set ResourceInfo(userIID)
	info.IId = getUserIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})

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

// (1) check exist(NameID)
// (2) remove Rules
func RemoveRules(connectionName string, sgName string, reqRuleInfoList []cres.SecurityRuleInfo) (bool, error) {
	cblog.Info("call RemoveRules()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	sgName, err = EmptyCheckAndTrim("sgName", sgName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	handler, err := cldConn.CreateSecurityHandler()
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	// Direction: to lower
	// IPProtocol: to upper
	// no CIDR: "0.0.0.0/0"
	transformArgs(&reqRuleInfoList)

	sgSPLock.Lock(connectionName, sgName)
	defer sgSPLock.Unlock(connectionName, sgName)

	// (1) check exist(sgName)
	bool_ret := false
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*SGIIDInfo
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		bool_ret, err = isNameIdExists(&iidInfoList, sgName)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
	} else {
		bool_ret, err = infostore.HasByConditions(&SGIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, sgName)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
	}
	if !bool_ret {
		err := fmt.Errorf("The %s '%s' does not exist!", RSTypeString(SG), sgName)
		cblog.Error(err)
		return false, err
	}

	var iidInfo SGIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*SGIIDInfo
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, sgName)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		iidInfo = *castedIIDInfo.(*SGIIDInfo)
	} else {
		err = infostore.GetByConditions(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, sgName)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
	}

	// (2) remove Rules
	// driverIID for driver
	result, err := handler.RemoveRules(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}), &reqRuleInfoList)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	return result, nil
}

// (1) get spiderIID
// (2) delete Resource(SystemId)
// (3) delete IID
func DeleteSecurity(connectionName string, rsType string, nameID string, force string) (bool, error) {
	cblog.Info("call DeleteSecurity()")

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

	handler, err := cldConn.CreateSecurityHandler()
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	sgSPLock.Lock(connectionName, nameID)
	defer sgSPLock.Unlock(connectionName, nameID)

	// (1) get spiderIID for creating driverIID
	var iidInfo SGIIDInfo
	if os.Getenv("PERMISSION_BASED_CONTROL_MODE") != "" {
		var iidInfoList []*SGIIDInfo
		err = getAuthIIDInfoList(connectionName, &iidInfoList)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		castedIIDInfo, err := getAuthIIDInfo(&iidInfoList, nameID)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		iidInfo = *castedIIDInfo.(*SGIIDInfo)
	} else {
		err = infostore.GetByConditions(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
	}

	// (2) delete Resource(SystemId)
	driverIId := getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})
	result, err := handler.(cres.SecurityHandler).DeleteSecurity(driverIId)
	if err != nil {
		cblog.Error(err)
		if force != "true" {
			return false, err
		}
	}

	if force != "true" {
		if !result {
			return result, nil
		}
	}

	// (3) delete IID
	_, err = infostore.DeleteBy3Conditions(&SGIIDInfo{}, CONNECTION_NAME_COLUMN, iidInfo.ConnectionName, NAME_ID_COLUMN, nameID,
		OWNER_VPC_NAME_COLUMN, iidInfo.OwnerVPCName)
	if err != nil {
		cblog.Error(err)
		if force != "true" {
			return false, err
		}
	}

	return result, nil
}

func CountAllSecurityGroups() (int64, error) {
	var info SGIIDInfo
	count, err := infostore.CountAllNameIDs(&info)
	if err != nil {
		cblog.Error(err)
		return count, err
	}

	return count, nil
}

func CountSecurityGroupsByConnection(connectionName string) (int64, error) {
	var info SGIIDInfo
	count, err := infostore.CountNameIDsByConnection(&info, connectionName)
	if err != nil {
		cblog.Error(err)
		return count, err
	}

	return count, nil
}
