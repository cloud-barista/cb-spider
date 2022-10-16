// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2022.09.

package commonruntime

import (
	"fmt"
	"strings"
	"errors"

	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	iidm "github.com/cloud-barista/cb-spider/cloud-control-manager/iid-manager"
)


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

        rsType := rsNLB

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
        iidInfoList, err := getAllNLBIIDInfoList(connectionName)
        if err != nil {
//vpcSPLock.RUnlock()
//nlbSPLock.RUnlock()
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
//nlbSPLock.RUnlock()
                err :=  fmt.Errorf(rsType + "-" + cspID + " already exists with " + nameId + "!")
                cblog.Error(err)
                return cres.IID{}, err
        }

        // (2) get resource info(CSP-ID)
        // check existence and get info of this resouce in the CSP
        // Do not user NameId, because Azure driver use it like SystemId
        getInfo, err := handler.GetNLB( cres.IID{getMSShortID(cspID), cspID} )
        if err != nil {
//vpcSPLock.RUnlock()
//nlbSPLock.RUnlock()
                cblog.Error(err)
                return cres.IID{}, err
        }

        // (3) get VPC IID:list
        vpcIIDInfoList, err := iidRWLock.ListIID(iidm.IIDSGROUP, connectionName, rsVPC)
        if err != nil {
//vpcSPLock.RUnlock()
//nlbSPLock.RUnlock()
                cblog.Error(err)
                return cres.IID{}, err
        }
//vpcSPLock.RUnlock()
//nlbSPLock.RUnlock()

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

	emptyPermissionList := []string{
        }

        err = ValidateStruct(userIID, emptyPermissionList)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        rsType := rsNLB

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
        iidInfoList, err := getAllNLBIIDInfoList(connectionName)
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
        getInfo, err := handler.GetNLB( cres.IID{getMSShortID(userIID.SystemId), userIID.SystemId} )
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
        spiderIId := cres.IID{userIID.NameId, systemId + ":" + getInfo.IId.SystemId}


        // (4) insert spiderIID
        // insert NLB SpiderIID to metadb
	_, err = iidRWLock.CreateIID(iidm.NLBGROUP, connectionName, vpcUserID, spiderIId)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // set up NLB User IID for return info
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
func CreateNLB(connectionName string, rsType string, reqInfo cres.NLBInfo) (*cres.NLBInfo, error) { cblog.Info("call CreateNLB()")

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
	vpcIIDInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsVPC, reqInfo.VpcIID)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	reqInfo.VpcIID = getDriverIID(vpcIIDInfo.IId)
	//+++++++++++++++++++++++++++++++++++++++++++

	vmList := reqInfo.VMGroup.VMs
	for idx, vmIID := range *vmList { 
		vmIIDInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsVM, vmIID)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		(*vmList)[idx] = getDriverIID(vmIIDInfo.IId)
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
	iidInfoList, err := getAllNLBIIDInfoList(connectionName)
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
	info.VpcIID.NameId = vpcIIDInfo.IId.NameId

	// set VM's IID with NameId
	info.VMGroup.VMs = reqInfo.VMGroup.VMs

	// (4) create spiderIID: {reqNameID, "driverNameID:driverSystemID"}
	//     ex) spiderIID {"seoul-service", "vm-01-9m4e2mr0ui3e8a215n4g:i-0bc7123b7e5cbf79d"}
	spiderIId := cres.IID{reqIId.NameId, spUUID + ":" + info.IId.SystemId}

	// (5) insert spiderIID
	iidInfo, err := iidRWLock.CreateIID(iidm.NLBGROUP, connectionName, vpcIIDInfo.IId.NameId, spiderIId)  // reqIId.NameId => rsType
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
	info.IId = getUserIID(iidInfo.IId)

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

func setVMGroupSystemId(connectionName string, vmList *[]cres.IID) error {
        // set VM's SystemId
        for idx, vmIID := range *vmList {
                vmIIDInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsVM, vmIID)
                if err != nil {
                        cblog.Error(err)
                        return err
                }
                (*vmList)[idx] = getUserIID(vmIIDInfo.IId)
        }
	return nil
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
	iidInfoList, err := getAllNLBIIDInfoList(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	var infoList []*cres.NLBInfo
	if iidInfoList == nil || len(iidInfoList) <= 0 {
		infoList = []*cres.NLBInfo{}
		return infoList, nil
	}

	// (2) Get NLBInfo-list with IID-list
	infoList2 := []*cres.NLBInfo{}
	for _, iidInfo := range iidInfoList {

nlbSPLock.RLock(connectionName, iidInfo.IId.NameId)

		// get resource(SystemId)
		info, err := handler.GetNLB(getDriverIID(iidInfo.IId))
		if err != nil {
nlbSPLock.RUnlock(connectionName, iidInfo.IId.NameId)
			if checkNotFoundError(err) {
				cblog.Info(err)
				continue
			}
			cblog.Error(err)
			return nil, err
		}
nlbSPLock.RUnlock(connectionName, iidInfo.IId.NameId)

		// Protocol: to upper
		transformArgsToUpper(&info)

		// (3) set ResourceInfo(IID.NameId)
		// set ResourceInfo
		info.IId = getUserIID(iidInfo.IId)

		// set VPC UserIID
		vpcIIDInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsVPC, cres.IID{iidInfo.ResourceType/*vpcName*/, ""})
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		info.VpcIID = getUserIID(vpcIIDInfo.IId)

		// set VM's UserIID
		for idx, vmIID := range *info.VMGroup.VMs {
			vmIIDInfo, err := iidRWLock.GetIIDbySystemID(iidm.IIDSGROUP, connectionName, rsVM, vmIID)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
			(*info.VMGroup.VMs)[idx].NameId = vmIIDInfo.IId.NameId 
		}

		infoList2 = append(infoList2, &info)
	}

	return infoList2, nil
}

// Get All IID:list of NLB
// (1) Get VPC's Name List
// (2) Create All NLB's IIDInfo List
func getAllNLBIIDInfoList(connectionName string) ([]*iidm.IIDInfo, error) {

        // (1) Get VPC's Name List
        // format) /resource-info-spaces/{iidGroup}/{connectionName}/{resourceType}/{resourceName} [{resourceID}]
        vpcNameList, err := iidRWLock.ListResourceType(iidm.NLBGROUP, connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
	vpcNameList = uniqueNameList(vpcNameList)
        // (2) Create All NLB's IIDInfo List
        iidInfoList := []*iidm.IIDInfo{}
        for _, vpcName := range vpcNameList {
                iidInfoListForOneVPC, err := iidRWLock.ListIID(iidm.NLBGROUP, connectionName, vpcName)
                if err != nil {
                        cblog.Error(err)
                        return nil, err
                }
                iidInfoList = append(iidInfoList, iidInfoListForOneVPC...)
        }
        return iidInfoList, nil
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
	iidInfoList, err := getAllNLBIIDInfoList(connectionName)
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
	info, err := handler.GetNLB(getDriverIID(iidInfo.IId))
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
        // Protocol: to upper
        transformArgsToUpper(&info)

	// (3) set ResourceInfo(IID.NameId)
	// set ResourceInfo
	info.IId = getUserIID(iidInfo.IId)

	// set VPC UserIID
	vpcIIDInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsVPC, cres.IID{iidInfo.ResourceType/*vpcName*/, ""})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	info.VpcIID = getUserIID(vpcIIDInfo.IId)

	// set VM's UserIID
	for idx, vmIID := range *info.VMGroup.VMs {
		vmIIDInfo, err := iidRWLock.GetIIDbySystemID(iidm.IIDSGROUP, connectionName, rsVM, vmIID)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		(*info.VMGroup.VMs)[idx].NameId = vmIIDInfo.IId.NameId
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
        iidInfoList, err := getAllNLBIIDInfoList(connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        var iidInfo *iidm.IIDInfo
        var bool_ret = false
        for _, OneIIdInfo := range iidInfoList {
                if OneIIdInfo.IId.NameId == nlbName {
                        iidInfo = OneIIdInfo
                        bool_ret = true
                        break;
                }
        }
        if bool_ret == false {
                err := fmt.Errorf("The %s '%s' does not exist!", RsTypeString(rsNLB), nlbName)
                cblog.Error(err)
                return nil, err
        }

        // (2) add VMs
        // driverIID for driver
	var vmIIDs []cres.IID
	for _, one := range vmNames {
		// check vm existence
		bool_ret, err := iidRWLock.IsExistIID(iidm.IIDSGROUP, connectionName, rsVM, cres.IID{one, ""})
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		if bool_ret == false {
			err := fmt.Errorf("The %s '%s' does not exist!", RsTypeString(rsVM), one)
			cblog.Error(err)
			return nil, err
		}

		vmIIDInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsVM, cres.IID{one, ""})
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		vmIID := getDriverIID(vmIIDInfo.IId)

		vmIIDs = append(vmIIDs, vmIID)
	}
        _, err = handler.AddVMs(getDriverIID(iidInfo.IId), &vmIIDs) 
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

	// (3) Get NLBInfo
	info, err := handler.GetNLB(getDriverIID(iidInfo.IId))
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // Protocol: to upper
        transformArgsToUpper(&info)

        // (4) set ResourceInfo(userIID)
        info.IId = getUserIID(iidInfo.IId)

        // set VPC UserIID
        vpcIIDInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsVPC, cres.IID{iidInfo.ResourceType/*vpcName*/, ""})
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        info.VpcIID = getUserIID(vpcIIDInfo.IId)

        // set VM's UserIID
        for idx, vmIID := range *info.VMGroup.VMs {
                vmIIDInfo, err := iidRWLock.GetIIDbySystemID(iidm.IIDSGROUP, connectionName, rsVM, vmIID)
                if err != nil {
                        cblog.Error(err)
                        return nil, err
                }
                (*info.VMGroup.VMs)[idx].NameId = vmIIDInfo.IId.NameId
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
        iidInfoList, err := getAllNLBIIDInfoList(connectionName)
        if err != nil {
                cblog.Error(err)
                return false, err
        }
        var iidInfo *iidm.IIDInfo
        var bool_ret = false
        for _, OneIIdInfo := range iidInfoList {
                if OneIIdInfo.IId.NameId == nlbName {
                        iidInfo = OneIIdInfo
                        bool_ret = true
                        break;
                }
        }
        if bool_ret == false {
                err := fmt.Errorf("The %s '%s' does not exist!", RsTypeString(rsNLB), nlbName)
                cblog.Error(err)
                return false, err
        }

        // (2) remove VMs
        // driverIID for driver
        var vmIIDs []cres.IID
        for _, one := range vmNames {

                vmIIDInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsVM, cres.IID{one, ""})
                if err != nil {
                        cblog.Error(err)
                        return false, err
                }
                vmIID := getDriverIID(vmIIDInfo.IId)

                vmIIDs = append(vmIIDs, vmIID)
        }

        result, err := handler.RemoveVMs(getDriverIID(iidInfo.IId), &vmIIDs)
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        return result, nil
}


//---------------------------------------------------//
// @todo  To support or not will be decided later.   //
//---------------------------------------------------//
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
        iidInfoList, err := getAllNLBIIDInfoList(connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        var iidInfo *iidm.IIDInfo
        var bool_ret = false
        for _, OneIIdInfo := range iidInfoList {
                if OneIIdInfo.IId.NameId == nlbName {
                        iidInfo = OneIIdInfo
                        bool_ret = true
                        break;
                }
        }
        if bool_ret == false {
                err := fmt.Errorf("The %s '%s' does not exist!", RsTypeString(rsNLB), nlbName)
                cblog.Error(err)
                return nil, err
        }

        // (2) change listener
        // driverIID for driver
        _, err = handler.ChangeListener(getDriverIID(iidInfo.IId), listener)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

	// (3) Get NLBInfo
        info, err := handler.GetNLB(getDriverIID(iidInfo.IId))
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // Protocol: to upper
        transformArgsToUpper(&info)

        // (4) set ResourceInfo(userIID)
        info.IId = getUserIID(iidInfo.IId)

        // set VPC UserIID
        vpcIIDInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsVPC, cres.IID{iidInfo.ResourceType/*vpcName*/, ""})
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        info.VpcIID = getUserIID(vpcIIDInfo.IId)

        // set VM's UserIID
        for idx, vmIID := range *info.VMGroup.VMs {
                vmIIDInfo, err := iidRWLock.GetIIDbySystemID(iidm.IIDSGROUP, connectionName, rsVM, vmIID)
                if err != nil {
                        cblog.Error(err)
                        return nil, err
                }
                (*info.VMGroup.VMs)[idx].NameId = vmIIDInfo.IId.NameId
        }

        return &info, nil
}

//---------------------------------------------------//
// @todo  To support or not will be decided later.   //
//---------------------------------------------------//
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
        iidInfoList, err := getAllNLBIIDInfoList(connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        var iidInfo *iidm.IIDInfo
        var bool_ret = false
        for _, OneIIdInfo := range iidInfoList {
                if OneIIdInfo.IId.NameId == nlbName {
                        iidInfo = OneIIdInfo
                        bool_ret = true
                        break;
                }
        }
        if bool_ret == false {
                err := fmt.Errorf("The %s '%s' does not exist!", RsTypeString(rsNLB), nlbName)
                cblog.Error(err)
                return nil, err
        }

        // (2) change VMGroup
        // driverIID for driver
        _, err = handler.ChangeVMGroupInfo(getDriverIID(iidInfo.IId), vmGroup)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // (3) Get NLBInfo
        info, err := handler.GetNLB(getDriverIID(iidInfo.IId))
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // Protocol: to upper
        transformArgsToUpper(&info)

        // (4) set ResourceInfo(userIID)
        info.IId = getUserIID(iidInfo.IId)

        // set VM's UserIID
        for idx, vmIID := range *info.VMGroup.VMs {
                vmIIDInfo, err := iidRWLock.GetIIDbySystemID(iidm.IIDSGROUP, connectionName, rsVM, vmIID)
                if err != nil {
                        cblog.Error(err)
                        return nil, err
                }
                (*info.VMGroup.VMs)[idx].NameId = vmIIDInfo.IId.NameId
        }

        // set VPC SystemId
        vpcIIDInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsVPC, cres.IID{iidInfo.ResourceType/*vpcName*/, ""})
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        info.VpcIID = getUserIID(vpcIIDInfo.IId)

        return &info, nil
}

//---------------------------------------------------//
// @todo  To support or not will be decided later.   //
//---------------------------------------------------//
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
        iidInfoList, err := getAllNLBIIDInfoList(connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        var iidInfo *iidm.IIDInfo
        var bool_ret = false
        for _, OneIIdInfo := range iidInfoList {
                if OneIIdInfo.IId.NameId == nlbName {
                        iidInfo = OneIIdInfo
                        bool_ret = true
                        break;
                }
        }
        if bool_ret == false {
                err := fmt.Errorf("The %s '%s' does not exist!", RsTypeString(rsNLB), nlbName)
                cblog.Error(err)
                return nil, err
        }

        // (2) change VMGroup
        // driverIID for driver
        _, err = handler.ChangeHealthCheckerInfo(getDriverIID(iidInfo.IId), healthChecker)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // (3) Get NLBInfo
        info, err := handler.GetNLB(getDriverIID(iidInfo.IId))
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // Protocol: to upper
        transformArgsToUpper(&info)

        // (4) set ResourceInfo(userIID)
        info.IId = getUserIID(iidInfo.IId)

        // set VPC UserIID
        vpcIIDInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsVPC, cres.IID{iidInfo.ResourceType/*vpcName*/, ""})
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        info.VpcIID = getUserIID(vpcIIDInfo.IId)

        // set VM's UserIID
        for idx, vmIID := range *info.VMGroup.VMs {
                vmIIDInfo, err := iidRWLock.GetIIDbySystemID(iidm.IIDSGROUP, connectionName, rsVM, vmIID)
                if err != nil {
                        cblog.Error(err)
                        return nil, err
                }
                (*info.VMGroup.VMs)[idx].NameId = vmIIDInfo.IId.NameId
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
        iidInfoList, err := getAllNLBIIDInfoList(connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        var iidInfo *iidm.IIDInfo
        var bool_ret = false
        for _, OneIIdInfo := range iidInfoList {
                if OneIIdInfo.IId.NameId == nlbName {
                        iidInfo = OneIIdInfo
                        bool_ret = true
                        break;
                }
        }
        if bool_ret == false {
                err := fmt.Errorf("The %s '%s' does not exist!", RsTypeString(rsNLB), nlbName)
                cblog.Error(err)
                return nil, err
        }

        // (2) change VMGroup
        // driverIID for driver
        healthInfo, err := handler.GetVMGroupHealthInfo(getDriverIID(iidInfo.IId))
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
	vmIIDList := healthInfo.AllVMs	
	for idx, vm := range *vmIIDList {
		foundFlag := false
		vmIIDInfo, err := iidRWLock.GetIIDbySystemID(iidm.IIDSGROUP, connectionName, rsVM, vm)
		if err != nil {
			cblog.Error(err)
			return err
		}
		if vm.SystemId == getDriverSystemId(vmIIDInfo.IId) {
			foundFlag = true
			(*vmIIDList)[idx].NameId = vmIIDInfo.IId.NameId
		}
		if !foundFlag {
			errList = append(errList, connectionName + ":CSP-VM:" + vm.SystemId + " is not owned by CB-Spider!")
		}
	}

        vmIIDList = healthInfo.HealthyVMs
        for idx, vm := range *vmIIDList {
		foundFlag := false
                vmIIDInfo, err := iidRWLock.GetIIDbySystemID(iidm.IIDSGROUP, connectionName, rsVM, vm)
                if err != nil {
                        cblog.Error(err)
                        return err
                }
                if vm.SystemId == getDriverSystemId(vmIIDInfo.IId) {
                        foundFlag = true
                        (*vmIIDList)[idx].NameId = vmIIDInfo.IId.NameId
                }
		if !foundFlag {
			errList = append(errList, connectionName + ":CSP-VM:" + vm.SystemId + " is not owned by CB-Spider!")
		}
        }

        vmIIDList = healthInfo.UnHealthyVMs
        for idx, vm := range *vmIIDList {
		foundFlag := false
                vmIIDInfo, err := iidRWLock.GetIIDbySystemID(iidm.IIDSGROUP, connectionName, rsVM, vm)
                if err != nil {
                        cblog.Error(err)
                        return err
                }
                if vm.SystemId == getDriverSystemId(vmIIDInfo.IId) {
                        foundFlag = true
                        (*vmIIDList)[idx].NameId = vmIIDInfo.IId.NameId
                }
		if !foundFlag {
			errList = append(errList, connectionName + ":CSP-VM:" + vm.SystemId + " is not owned by CB-Spider!")
		}
        }

	// check error existence
	if len(errList) > 0 {
                cblog.Error(strings.Join(errList, "\n"))
                return errors.New(strings.Join(errList, "\n"))
	}

	return nil
}
