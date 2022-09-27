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
	"strings"

	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	iidm "github.com/cloud-barista/cb-spider/cloud-control-manager/iid-manager"
)


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

        rsType := rsSG

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
        iidInfoList, err := getAllSGIIDInfoList(connectionName)
        if err != nil {
//vpcSPLock.RUnlock()
//sgSPLock.RUnlock()
                cblog.Error(err)
                return cres.IID{}, err
        }
        var isExist bool=false
        var nameId string
        for _, OneIIdInfo := range iidInfoList {
                if getMSShortID(getDriverSystemId(OneIIdInfo.IId)) == cspID {
                        nameId = OneIIdInfo.IId.NameId
                        isExist = true
                        break
                }
        }
        if isExist == true {
//vpcSPLock.RUnlock()
//sgSPLock.RUnlock()
                err :=  fmt.Errorf(rsType + "-" + cspID + " already exists with " + nameId + "!")
                cblog.Error(err)
                return cres.IID{}, err
        }

        // (2) get resource info(CSP-ID)
        // check existence and get info of this resouce in the CSP
        // Do not user NameId, because Azure driver use it like SystemId
        getInfo, err := handler.GetSecurity( cres.IID{getMSShortID(cspID), cspID} )
        if err != nil {
//vpcSPLock.RUnlock()
//sgSPLock.RUnlock()
                cblog.Error(err)
                return cres.IID{}, err
        }

        // (3) get VPC IID:list
        vpcIIDInfoList, err := iidRWLock.ListIID(iidm.IIDSGROUP, connectionName, rsVPC)
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
                return cres.IID{"", vpcCSPID}, nil
        }

        // (4) check existence in the MetaDB
        for _, one := range vpcIIDInfoList {
                if getMSShortID(getDriverSystemId(one.IId)) == vpcCSPID {
                        return cres.IID{one.IId.NameId, vpcCSPID}, nil
                }
        }

        return cres.IID{"", vpcCSPID}, nil
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

	emptyPermissionList := []string{
        }

        err = ValidateStruct(userIID, emptyPermissionList)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        rsType := rsSG

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
        bool_ret, err := iidRWLock.IsExistIID(iidm.IIDSGROUP, connectionName, rsVPC, cres.IID{vpcUserID, ""})
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        if bool_ret == false {
		err := fmt.Errorf("The %s '%s' does not exist!", RsTypeString(rsVPC), vpcUserID)
		cblog.Error(err)
                return nil, err
        }

        // (1) check existence(UserID)
        iidInfoList, err := getAllSGIIDInfoList(connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        var isExist bool=false
        for _, OneIIdInfo := range iidInfoList {
                if OneIIdInfo.IId.NameId == userIID.NameId {
                        isExist = true
			break
                }
        }

        if isExist == true {
                err :=  fmt.Errorf(rsType + "-" + userIID.NameId + " already exists!")
                cblog.Error(err)
                return nil, err
        }


        // (2) get resource info(CSP-ID)
        // check existence and get info of this resouce in the CSP
	// Do not user NameId, because Azure driver use it like SystemId
        getInfo, err := handler.GetSecurity( cres.IID{getMSShortID(userIID.SystemId), userIID.SystemId} )
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
        spiderIId := cres.IID{userIID.NameId, systemId + ":" + getInfo.IId.SystemId}


        // (4) insert spiderIID
        // insert SecurityGroup SpiderIID to metadb
	_, err = iidRWLock.CreateIID(iidm.SGGROUP, connectionName, vpcUserID, spiderIId)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // set up SecurityGroup User IID for return info
        getInfo.IId = userIID

        // set up VPC UserIID for return info
        iidInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsVPC, cres.IID{vpcUserID, ""})
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        getInfo.VpcIID = getUserIID(iidInfo.IId)
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
func CreateSecurity(connectionName string, rsType string, reqInfo cres.SecurityReqInfo) (*cres.SecurityInfo, error) {
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
	vpcIIDInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsVPC, reqInfo.VpcIID)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	reqInfo.VpcIID.SystemId = getDriverSystemId(vpcIIDInfo.IId)
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
	iidInfoList, err := getAllSGIIDInfoList(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	var isExist bool=false
	for _, OneIIdInfo := range iidInfoList {
		if OneIIdInfo.IId.NameId == reqInfo.IId.NameId {
			isExist = true
		}
	}

	if isExist == true {
		err :=  fmt.Errorf(rsType + "-" + reqInfo.IId.NameId + " already exists!")
		cblog.Error(err)
		return nil, err
	}

	// (2) generate SP-XID and create reqIID, driverIID
	//     ex) SP-XID {"vm-01-9m4e2mr0ui3e8a215n4g"}
	//
	//     create reqIID: {reqNameID, reqSystemID}   # reqSystemID=SP-XID
	//         ex) reqIID {"seoul-service", "vm-01-9m4e2mr0ui3e8a215n4g"} 
	//
	//     create driverIID: {driverNameID, driverSystemID}   # driverNameID=SP-XID, driverSystemID=csp's ID
	//         ex) driverIID {"vm-01-9m4e2mr0ui3e8a215n4g", "i-0bc7123b7e5cbf79d"}
	spUUID, err := iidm.New(connectionName, rsType, reqInfo.IId.NameId)
	if err != nil {
                cblog.Error(err)
                return nil, err
        }

	// reqIID
	reqIId := cres.IID{reqInfo.IId.NameId, spUUID}
	// driverIID
	driverIId := cres.IID{spUUID, ""}
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
	spiderIId := cres.IID{reqIId.NameId, spUUID + ":" + info.IId.SystemId}

	// (5) insert spiderIID
	iidInfo, err := iidRWLock.CreateIID(iidm.SGGROUP, connectionName, reqInfo.VpcIID.NameId, spiderIId)  // reqIId.NameId => rsType
	if err != nil {
		cblog.Error(err)
		// rollback
		_, err2 := handler.DeleteSecurity(info.IId)
		if err2 != nil {
			cblog.Error(err2)
			return nil, fmt.Errorf(err.Error() + ", " + err2.Error())
		}
		cblog.Error(err)
		return nil, err
	}

	// (6) create userIID: {reqNameID, driverSystemID}
	//     ex) userIID {"seoul-service", "i-0bc7123b7e5cbf79d"}
	info.IId = getUserIID(iidInfo.IId)

	// set VPC SystemId
	info.VpcIID.SystemId = getDriverSystemId(vpcIIDInfo.IId)

	return &info, nil
}

func transformArgs(ruleList *[]cres.SecurityRuleInfo) {
        for n, _ := range *ruleList {
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
	iidInfoList, err := getAllSGIIDInfoList(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	var infoList []*cres.SecurityInfo
	if iidInfoList == nil || len(iidInfoList) <= 0 {
		infoList = []*cres.SecurityInfo{}
		return infoList, nil
	}

	// (2) Get SecurityInfo-list with IID-list
	infoList2 := []*cres.SecurityInfo{}
	for _, iidInfo := range iidInfoList {

sgSPLock.RLock(connectionName, iidInfo.IId.NameId)

		// get resource(SystemId)
		info, err := handler.GetSecurity(getDriverIID(iidInfo.IId))
		if err != nil {
sgSPLock.RUnlock(connectionName, iidInfo.IId.NameId)
			if checkNotFoundError(err) {
				cblog.Info(err)
				continue
			}
			cblog.Error(err)
			return nil, err
		}
sgSPLock.RUnlock(connectionName, iidInfo.IId.NameId)
		// Direction: to lower
		// IPProtocol: to upper
		// no CIDR: "0.0.0.0/0"
		transformArgs(info.SecurityRules)

		// (3) set ResourceInfo(IID.NameId)
		// set ResourceInfo
		info.IId = getUserIID(iidInfo.IId)

		// set VPC SystemId
		vpcIIDInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsVPC, cres.IID{iidInfo.ResourceType/*vpcName*/, ""})
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		info.VpcIID = getUserIID(vpcIIDInfo.IId)

		infoList2 = append(infoList2, &info)
	}

	return infoList2, nil
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
	iidInfoList, err := getAllSGIIDInfoList(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	var iidInfo *iidm.IIDInfo
	var bool_ret = false
	for _, OneIIdInfo := range iidInfoList {
		if OneIIdInfo.IId.NameId == nameID {
			iidInfo = OneIIdInfo
			bool_ret = true
			break;
		}
	}
	if bool_ret == false {
		err := fmt.Errorf("The %s '%s' does not exist!", RsTypeString(rsType), nameID)
		cblog.Error(err)
                return nil, err
        }

	// (2) get resource(SystemId)
	info, err := handler.GetSecurity(getDriverIID(iidInfo.IId))
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
	info.IId = getUserIID(iidInfo.IId)

	// set VPC SystemId
	vpcIIDInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsVPC, cres.IID{iidInfo.ResourceType/*vpcName*/, ""})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	info.VpcIID = getUserIID(vpcIIDInfo.IId)

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
        iidInfoList, err := getAllSGIIDInfoList(connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        var iidInfo *iidm.IIDInfo
        var bool_ret = false
        for _, OneIIdInfo := range iidInfoList {
                if OneIIdInfo.IId.NameId == sgName {
                        iidInfo = OneIIdInfo
                        bool_ret = true
                        break;
                }
        }
        if bool_ret == false {
                err := fmt.Errorf("The %s '%s' does not exist!", RsTypeString(rsSG), sgName)
                cblog.Error(err)
                return nil, err
        }

        // (2) add Rules
        // driverIID for driver
        info, err := handler.AddRules(getDriverIID(iidInfo.IId), &reqInfoList)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        // Direction: to lower
        // IPProtocol: to upper
        // no CIDR: "0.0.0.0/0"
        transformArgs(info.SecurityRules)

        // (3) set ResourceInfo(userIID)
        info.IId = getUserIID(iidInfo.IId)

        // set VPC SystemId
        vpcIIDInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsVPC, cres.IID{iidInfo.ResourceType/*vpcName*/, ""})
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        info.VpcIID = getUserIID(vpcIIDInfo.IId)

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
        iidInfoList, err := getAllSGIIDInfoList(connectionName)
        if err != nil {
                cblog.Error(err)
                return false, err
        }
        var iidInfo *iidm.IIDInfo
        var bool_ret = false
        for _, OneIIdInfo := range iidInfoList {
                if OneIIdInfo.IId.NameId == sgName {
                        iidInfo = OneIIdInfo
                        bool_ret = true
                        break;
                }
        }
        if bool_ret == false {
                err := fmt.Errorf("The %s '%s' does not exist!", RsTypeString(rsSG), sgName)
                cblog.Error(err)
                return false, err
        }

        // (2) remove Rules
        // driverIID for driver
        result, err := handler.RemoveRules(getDriverIID(iidInfo.IId), &reqRuleInfoList)
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        return result, nil
}
