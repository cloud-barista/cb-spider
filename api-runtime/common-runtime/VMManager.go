// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.09.

package commonruntime

import (
	"fmt"
	"os"
	"strings"
	"strconv"
	"sync"
	"errors"

	cim "github.com/cloud-barista/cb-spider/cloud-info-manager"
	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	ccon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
	iidm "github.com/cloud-barista/cb-spider/cloud-control-manager/iid-manager"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/vm-ssh"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
)


//================ VM Handler

type VMUsingResources struct {
        Resources struct {
                VPC	*cres.IID  `json:"VPC"`
                SGList	[]*cres.IID `json:"SGList"`
                VMKey	*cres.IID  `json:"VMKey"`
        }
}

func GetVMUsingRS(connectionName string, cspID string) (VMUsingResources, error) {
        cblog.Info("call GetVMUsingRS()")

        // check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
                cblog.Error(err)
                return VMUsingResources{}, err
        }

        cspID, err = EmptyCheckAndTrim("cspID", cspID)
        if err != nil {
                cblog.Error(err)
                return VMUsingResources{}, err
        }

        rsType := rsVM

        cldConn, err := ccm.GetCloudConnection(connectionName)
        if err != nil {
                cblog.Error(err)
                return VMUsingResources{}, err
        }

        handler, err := cldConn.CreateVMHandler()
        if err != nil {
                cblog.Error(err)
                return VMUsingResources{}, err
        }

// Except Management API
//vmSPLock.RLock()
//defer vmSPLock.RUnlock()
//vpcSPLock.RLock()
//defer vpcSPLock.RUnlock()
//sgSPLock.RLock()
//defer sgSPLock.RUnlock()
//keySPLock.RLock()
//defer keySPLock.RUnlock()

        // (1) check existence(cspID)
        iidInfoList, err := iidRWLock.ListIID(iidm.IIDSGROUP, connectionName, rsVM)
        if err != nil {
                cblog.Error(err)
                return VMUsingResources{}, err
        }
        var isExist bool=false
        var nameId string
        for _, OneIIdInfo := range iidInfoList {
                if getDriverSystemId(OneIIdInfo.IId) == cspID {
                        nameId = OneIIdInfo.IId.NameId
                        isExist = true
                        break
                }
        }
        if isExist == true {
                err :=  fmt.Errorf(rsType + "-" + cspID + " already exists with " + nameId + "!")
                cblog.Error(err)
                return VMUsingResources{}, err
        }

        // (2) get resource info(CSP-ID)
        // check existence and get info of this resouce in the CSP
        // Do not user NameId, because Azure driver use it like SystemId
        getInfo, err := handler.GetVM( cres.IID{getMSShortID(cspID), cspID} )
        if err != nil {
                cblog.Error(err)
                return VMUsingResources{}, err
        }


	////////////////////////////////////////////
	// (3) Get using IIDs of (a) VPC, (b) SG, (c) Key
	////////////////////////////////////////////

        //// ---(a) Get Using a VPC IID

        // get VPC IID:list
        vpcIIDInfoList, err := iidRWLock.ListIID(iidm.IIDSGROUP, connectionName, rsVPC)
        if err != nil {
                cblog.Error(err)
                return VMUsingResources{}, err
        }

        // ex) spiderIID {"vpc-01", "vpc-01-9m4e2mr0ui3e8a215n4g:i-0bc7123b7e5cbf79d"}
        // Do not use NameId, because Azure driver use it like SystemId
        vpcCSPID := getMSShortID(getInfo.VpcIID.SystemId)

	vpcIID := cres.IID{"", vpcCSPID}

        // check existence in the MetaDB
        for _, one := range vpcIIDInfoList {
                if getMSShortID(getDriverSystemId(one.IId)) == vpcCSPID {
                        vpcIID = cres.IID{one.IId.NameId, vpcCSPID}
                }
        }

        //// ---(b) Get Using SG IID List

        // get SG IID:list
        sgIIDInfoList, err := getAllSGIIDInfoList(connectionName)
        if err != nil {
                cblog.Error(err)
                return VMUsingResources{}, err
        }

        // ex) spiderIID {"vpc-01", "vpc-01-9m4e2mr0ui3e8a215n4g:i-0bc7123b7e5cbf79d"}
        // Do not use NameId, because Azure driver use it like SystemId
        var sgCSPIDList []*string
	for _, one := range getInfo.SecurityGroupIIds {
		sgCSPID := getMSShortID(one.SystemId)
		sgCSPIDList = append(sgCSPIDList, &sgCSPID)
	}

        var sgIIDList []*cres.IID

        // check existence in the MetaDB
	for _, cspID := range sgCSPIDList {
		has := false
		for _, one := range sgIIDInfoList {
			if getMSShortID(getDriverSystemId(one.IId)) == *cspID {
				sgIID := cres.IID{one.IId.NameId, *cspID}
				sgIIDList = append(sgIIDList, &sgIID) // mapped SG
				has = true;
				break;
			}
		}
		if !has {
			sgIIDList = append(sgIIDList, &cres.IID{"", *cspID}) // unmapped SG
		}
	}


        //// ---(c) Get Using Key IID List

        // get Key IID:list
        keyIIDInfoList, err := iidRWLock.ListIID(iidm.IIDSGROUP, connectionName, rsKey)
        if err != nil {
                cblog.Error(err)
                return VMUsingResources{}, err
        }

        // ex) spiderIID {"vpc-01", "vpc-01-9m4e2mr0ui3e8a215n4g:i-0bc7123b7e5cbf79d"}
        // Do not use NameId, because Azure driver use it like SystemId
        keyCSPID := getMSShortID(getInfo.KeyPairIId.SystemId)

        keyIID := cres.IID{"", keyCSPID}

        // check existence in the MetaDB
        for _, one := range keyIIDInfoList {
                if getMSShortID(getDriverSystemId(one.IId)) == keyCSPID {
                        keyIID = cres.IID{one.IId.NameId, keyCSPID}
                }
        }


	var vmUsingRS VMUsingResources
	vmUsingRS.Resources.VPC = &vpcIID
	vmUsingRS.Resources.SGList = sgIIDList
	vmUsingRS.Resources.VMKey = &keyIID

        return vmUsingRS, nil
}


// UserIID{UserID, CSP-ID} => SpiderIID{UserID, SP-XID:CSP-ID}
// (1) check existence(UserID)
// (2) get resource info(CSP-ID)
// (3) create spiderIID: {UserID, SP-XID:CSP-ID}
// (4) insert spiderIID
func RegisterVM(connectionName string, userIID cres.IID) (*cres.VMInfo, error) {
        cblog.Info("call RegisterVM()")

	// check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
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

        rsType := rsVM

        cldConn, err := ccm.GetCloudConnection(connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        handler, err := cldConn.CreateVMHandler()
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        vmSPLock.Lock(connectionName, userIID.NameId)
        defer vmSPLock.Unlock(connectionName, userIID.NameId)

        // (1) check existence(UserID)
	bool_ret, err := iidRWLock.IsExistIID(iidm.IIDSGROUP, connectionName, rsType, userIID)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        if bool_ret == true {
		err := fmt.Errorf(rsType + "-" + userIID.NameId + " already exists!")
		cblog.Error(err)
                return nil, err
        }

        // (2) get resource info(CSP-ID)
        // check existence and get info of this resouce in the CSP
	// Do not user NameId, because Azure driver use it like SystemId
        getInfo, err := handler.GetVM( cres.IID{userIID.SystemId, userIID.SystemId} )
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

	// check and set ID
	err = getSetNameId(connectionName, &getInfo)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

	        // check Winddows GuestOS
        isWindowsOS := false
        isWindowsOS, err = checkImageWindowsOS(cldConn, getInfo.ImageType, getInfo.ImageIId)
        if err != nil {
                if strings.Contains(err.Error(), "yet!") {
                        cblog.Info(err)
                } else {
                        cblog.Error(err)
                        return nil, err
                }
        }

	if isWindowsOS {
                getInfo.VMUserId = "Administrator"
		getInfo.SSHAccessPoint = "RDP: " + getInfo.PublicIP
        } else {
                getInfo.VMUserId = "cb-user"
                // current: Assume 22 port, except Cloud-Twin, by powerkim, 2021.03.24.
                if getInfo.SSHAccessPoint == "" {
                        getInfo.SSHAccessPoint = getInfo.PublicIP + ":22"
                }
        }


        // (3) create spiderIID: {UserID, SP-XID:CSP-ID}
        //     ex) spiderIID {"vpc-01", "vpc-01-9m4e2mr0ui3e8a215n4g:i-0bc7123b7e5cbf79d"}
	// Do not user NameId, because Azure driver use it like SystemId
	systemId := getMSShortID(getInfo.IId.SystemId)
        spiderIId := cres.IID{userIID.NameId, systemId + ":" + getInfo.IId.SystemId}

        // (4) insert spiderIID
        // insert VM SpiderIID to metadb
        _, err = iidRWLock.CreateIID(iidm.IIDSGROUP, connectionName, rsType, spiderIId)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // set up VM User IID for return info
        getInfo.IId = userIID


        return &getInfo, nil
}

// (1) check exist(NameID)
// (2) generate SP-XID and create reqIID, driverIID
// (3) clone the reqInfo with DriverIID
// (4) create Resource
// (5) create spiderIID: {reqNameID, "driverNameID:driverSystemID"}
// (6) insert spiderIID
// (7) create userIID
func StartVM(connectionName string, rsType string, reqInfo cres.VMReqInfo) (*cres.VMInfo, error) {
	cblog.Info("call StartVM()")

	// check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
		cblog.Error(err)
                return nil, err
        }

	emptyPermissionList := []string{
                "resources.IID:SystemId",
                "resources.VMReqInfo:RootDiskType", // because can be set without disk type
                "resources.VMReqInfo:RootDiskSize", // because can be set without disk size
               // "resources.VMReqInfo:KeyPairName",  // because can be set without KeyPair for Windows
	//	"resources.IID:NameId",
                "resources.VMReqInfo:VMUserId",     // because can be set without VM User
                "resources.VMReqInfo:VMUserPasswd", // because can be set without VM PW
        }

        err = ValidateStruct(reqInfo, emptyPermissionList)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

	err = checkImageType(&reqInfo)
	if err != nil {
                cblog.Error(err)
                return nil, err
        }

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateVMHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	vmSPLock.Lock(connectionName, reqInfo.IId.NameId)
	defer vmSPLock.Unlock(connectionName, reqInfo.IId.NameId)

	// (1) check exist(NameID)
	dockerTest := os.Getenv("DOCKER_POC_TEST") // For docker poc tests, this is currently the best method.
	if dockerTest == "" || dockerTest == "OFF" {
		bool_ret, err := iidRWLock.IsExistIID(iidm.IIDSGROUP, connectionName, rsType, reqInfo.IId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}

		if bool_ret == true {
			err := fmt.Errorf(rsType + "-" + reqInfo.IId.NameId + " already exists!")
			cblog.Error(err)
			return nil, err
		}
	}

	providerName, err := ccm.GetProviderNameByConnectionName(connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        regionName, zoneName, err := ccm.GetRegionNameByConnectionName(connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

	// Translate user's root disk setting info into driver's root disk setting info.
	err = translateRootDiskSetupInfo(providerName, &reqInfo) 
        if err != nil {
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

	// (3) clone the reqInfo with DriverIID
	var reqInfoForDriver cres.VMReqInfo
	if dockerTest == "ON" {
		reqInfoForDriver = reqInfo
	}else {
		reqInfoForDriver, err = cloneReqInfoWithDriverIID(connectionName, reqInfo)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}

	// check Winddows GuestOS
	isWindowsOS := false
	isWindowsOS, err = checkImageWindowsOS(cldConn, reqInfoForDriver.ImageType, reqInfoForDriver.ImageIID)
	if err != nil {
		if strings.Contains(err.Error(), "yet!") {
			cblog.Info(err)
		} else {
			cblog.Error(err)
			return nil, err
		}
	}

	if isWindowsOS {
		adminID := "Administrator"
		if reqInfoForDriver.VMUserId == "" {
			reqInfo.VMUserId = adminID
			reqInfoForDriver.VMUserId = adminID
		}
		if reqInfoForDriver.VMUserId != adminID {
			cblog.Error(err)
			return nil, fmt.Errorf(reqInfoForDriver.VMUserId + ": cannot be used for Windows GuestOS UserID!")
		}
	}

	callInfo := call.CLOUDLOGSCHEMA {
                CloudOS: call.CLOUD_OS(providerName),
                RegionZone: regionName + "/" + zoneName,
                ResourceType: call.VM,
                ResourceName: reqInfo.IId.NameId,
		CloudOSAPI: "CB-Spider:StartVM()",
                ElapsedTime: "",
                ErrorMSG: "",
        }
        start := call.Start()

	// (4) create Resource
	info, err := handler.StartVM(reqInfoForDriver)
	if err != nil {
		cblog.Error(err)
		callInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callInfo))
		return nil, err
	}

	// Check Sync Called and Make sure cb-user prepared -----------------
	// --- <step-1> Get PublicIP of new VM
	var checkError struct {
		Flag bool
		MSG string
	}

	waiter := NewWaiter(5, 240) // (sleep, timeout)
	var publicIP string
	for {
		vmInfo, err := handler.GetVM(info.IId)
		if err != nil {
			cblog.Error(err)
			if checkNotFoundError(err) { // VM is not created yet.
				continue
			}
			callInfo.ErrorMSG = err.Error()
			callogger.Info(call.String(callInfo))

			//handler.TerminateVM(info.IId)

			return nil, err
		}
		if vmInfo.PublicIP != "" {
			publicIP = vmInfo.PublicIP
			break
		}

		if !waiter.Wait() {
			//handler.TerminateVM(info.IId)
			checkError.Flag = true
			checkError.MSG = fmt.Sprintf("[%s] Failed to Start VM %s when getting PublicIP. (Timeout=%v)", connectionName, reqIId.NameId, waiter.Timeout)
			break
                }
	}

	if !checkError.Flag && !isWindowsOS && providerName != "MOCK" {
		// --- <step-2> Check SSHD Daemon of new VM
		waiter2 := NewWaiter(2, 120) // (sleep, timeout) 

		for {
			if checkSSH(publicIP+":22") {
				break
			}

			if !waiter2.Wait() {
				//handler.TerminateVM(info.IId)
				checkError.Flag = true
				checkError.MSG = fmt.Sprintf("[%s] Failed to Start VM %s when checking SSHD Daemon. (Timeout=%v)", connectionName, reqIId.NameId, waiter2.Timeout)
				break
			}
		}
	}

	callInfo.ElapsedTime = call.Elapsed(start)
	callogger.Info(call.String(callInfo))

	// End : Check Sync Called and Make sure cb-user prepared -----------------



	// (5) create spiderIID: {reqNameID, "driverNameID:driverSystemID"}
	//     ex) spiderIID {"seoul-service", "vm-01-9m4e2mr0ui3e8a215n4g:i-0bc7123b7e5cbf79d"}
	spiderIId := cres.IID{reqIId.NameId, spUUID + ":" + info.IId.SystemId}



	// (6) insert spiderIID
	iidInfo, err := iidRWLock.CreateIID(iidm.IIDSGROUP, connectionName, rsType, spiderIId)
	if err != nil {
		cblog.Error(err)
		// rollback
		_, err2 := handler.TerminateVM(info.IId) // @todo check validation
		if err2 != nil {
			cblog.Error(err2)
			return nil, fmt.Errorf(err.Error() + ", " + err2.Error())
		}
		cblog.Error(err)
		return nil, err
	}

	
/*
	// set sg NameId from VPCNameId-SecurityGroupNameId
	// IID.NameID format => {VPC NameID} + SG_DELIMITER + {SG NameID}
	for i, sgIID := range info.SecurityGroupIIds {
		vpc_sg_nameid := strings.Split(sgIID.NameId, SG_DELIMITER)
		info.SecurityGroupIIds[i].NameId = vpc_sg_nameid[1]
	}
*/
	// (7) create userIID: {reqNameID, driverSystemID}
	//     ex) userIID {"seoul-service", "i-0bc7123b7e5cbf79d"}
	info.IId = getUserIID(iidInfo.IId)

	/////////////////////////////////
	// set NameId for info by reqInfo
	/////////////////////////////////
	setNameId(connectionName, &info, &reqInfo)

	if isWindowsOS {
		info.VMUserId = reqInfo.VMUserId
		info.VMUserPasswd = reqInfo.VMUserPasswd
		info.SSHAccessPoint = "RDP: " + info.PublicIP
	} else {
		info.VMUserId = "cb-user"
		// current: Assume 22 port, except Cloud-Twin, by powerkim, 2021.03.24.
		if info.SSHAccessPoint == "" {
			info.SSHAccessPoint = info.PublicIP + ":22"
		}
	}

	//if checkError.Flag {
	//	return &info, fmt.Errorf(checkError.MSG)
	//} else {
		return &info, nil
	//}
}

func checkImageType(reqInfo *cres.VMReqInfo) error {

	if reqInfo.ImageType == "" {
		reqInfo.ImageType = cres.PublicImage
	}
	if reqInfo.ImageType == cres.MyImage {
		// checking to change ther Root-Disk
		if reqInfo.RootDiskType != "" || reqInfo.RootDiskSize != "" {
			return errors.New("MyImage can not configure the Root-Disk!!")
		}
		// checking to add Data-Disks
		if reqInfo.DataDiskIIDs == nil && len(reqInfo.DataDiskIIDs) > 0 {
			return errors.New("MyImage can not have a Data-Disk!!")
		}
	}
	return nil
}

func checkImageWindowsOS(cldConn ccon.CloudConnection, imageType cres.ImageType, imageIID cres.IID) (bool, error) {

	if imageType == cres.PublicImage {
		handler, err := cldConn.CreateImageHandler()
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		return handler.CheckWindowsImage(imageIID)
	}
	if imageType == cres.MyImage {
		handler, err := cldConn.CreateMyImageHandler()
                if err != nil {
                        cblog.Error(err)
                        return false, err
                }
                return handler.CheckWindowsImage(imageIID)
	}
	return false, fmt.Errorf(string(imageType) + " is not supported ImageType!")
}


func cloneReqInfoWithDriverIID(ConnectionName string, reqInfo cres.VMReqInfo) (cres.VMReqInfo, error) {

	newReqInfo := cres.VMReqInfo {
		IId:       cres.IID{reqInfo.IId.NameId, reqInfo.IId.SystemId},

		ImageType:        cres.ImageType(reqInfo.ImageType),
		// set Image SystemId
		//ImageIID:         cres.IID{reqInfo.ImageIID.NameId, reqInfo.ImageIID.NameId},
		//VpcIID:           cres.IID{reqInfo.VpcIID.NameId, reqInfo.VpcIID.SystemId},
		//SubnetIID:        cres.IID{reqInfo.SubnetIID.NameId, reqInfo.SubnetIID.SystemId},
		//SecurityGroupIIDs: getSecurityGroupIIDs(),

		VMSpecName:       reqInfo.VMSpecName,
		//KeyPairIID:       cres.IID{reqInfo.KeyPairIID.NameId, reqInfo.KeyPairIID.SystemId},

		RootDiskType:	  reqInfo.RootDiskType, 
		RootDiskSize:	  reqInfo.RootDiskSize,

		// DataDiskIIDs

		VMUserId:         reqInfo.VMUserId,
		VMUserPasswd:	  reqInfo.VMUserPasswd,
	}

	// set Image SystemId
	if reqInfo.ImageType == cres.PublicImage {
		newReqInfo.ImageIID = cres.IID{reqInfo.ImageIID.NameId, reqInfo.ImageIID.NameId}
	}
	if reqInfo.ImageType == cres.MyImage {
		if reqInfo.ImageIID.NameId != "" {		
			// get MyImage's SystemId
			imageIIdInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, ConnectionName, rsMyImage, reqInfo.ImageIID)
			if err != nil {
				cblog.Error(err)
				return cres.VMReqInfo{}, err
			}
			newReqInfo.ImageIID = getDriverIID(imageIIdInfo.IId)
		}
	}

	// set VPC SystemId
	if reqInfo.VpcIID.NameId != "" {
		// get spiderIID
		IIdInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, ConnectionName, rsVPC, reqInfo.VpcIID)
		if err != nil {
			cblog.Error(err)
			return cres.VMReqInfo{}, err
		}
		// set driverIID
		newReqInfo.VpcIID = getDriverIID(IIdInfo.IId)
	}

	// set Subnet SystemId
	if reqInfo.SubnetIID.NameId != "" {		
		IIdInfo, err := iidRWLock.GetIID(iidm.SUBNETGROUP, ConnectionName, reqInfo.VpcIID.NameId, reqInfo.SubnetIID) // reqInfo.VpcIID.NameId => rsType
		if err != nil {
			cblog.Error(err)
			return cres.VMReqInfo{}, err
		}
		// set driverIID
		newReqInfo.SubnetIID = getDriverIID(IIdInfo.IId)
	}

	// set SecurityGroups SystemId
	for _, sgIID := range reqInfo.SecurityGroupIIDs {
		IIdInfo, err := iidRWLock.GetIID(iidm.SGGROUP, ConnectionName, reqInfo.VpcIID.NameId, sgIID)  // reqInfo.VpcIID.NameId => rsType
		if err != nil {
			cblog.Error(err)
			return cres.VMReqInfo{}, err
		}
		// set driverIID
		newReqInfo.SecurityGroupIIDs = append(newReqInfo.SecurityGroupIIDs, getDriverIID(IIdInfo.IId))
	}

        // set Data Disk SystemId
        for _, diskIID := range reqInfo.DataDiskIIDs {
                IIdInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, ConnectionName, rsDisk, diskIID) 
                if err != nil {
                        cblog.Error(err)
                        return cres.VMReqInfo{}, err
                }
                // set driverIID
                newReqInfo.DataDiskIIDs = append(newReqInfo.DataDiskIIDs, getDriverIID(IIdInfo.IId))
        }

	// set KeyPair SystemId
	if reqInfo.KeyPairIID.NameId != "" {
		IIdInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, ConnectionName, rsKey, reqInfo.KeyPairIID)
		if err != nil {
			cblog.Error(err)
			return cres.VMReqInfo{}, err
		}
		newReqInfo.KeyPairIID = getDriverIID(IIdInfo.IId)
	}

	return newReqInfo, nil
}


func checkSSH(serverPort string) bool {

        dummyKey  := []byte(`
-----BEGIN RSA PRIVATE KEY-----
MIIEoQIBAAKCAQEArVNOLwMIp5VmZ4VPZotcoCHdEzimKalAsz+ccLfvAA1Y2ELH
VwihRvkrqukUlkC7B3ASSCtgxIt5ZqfAKy9JvlT+Po/XHfaIpu9KM/XsZSdsF2jS
zv3TCSvod2f09Bx7ebowLVRzyJe4UG+0OuM10Sk9dXRXL+viizyyPp1Ie2+FN32i
KVTG9jVd21kWUYxT7eKuqH78Jt5Ezmsqs4ArND5qM3B2BWQ9GiyOcOl6NfyA4+RH
wv8eYRJkkjv5q7R675U+EWLe7ktpmboOgl/I5hV1Oj/SQ3F90RqUcLrRz9XTsRKl
nKY2KG/2Q3ZYabf9TpZ/DeHNLus5n4STzFmukQIBIwKCAQEAqF+Nx0TGlCq7P/3Y
GnjAYQr0BAslEoco6KQxkhHDmaaQ0hT8KKlMNlEjGw5Og1TS8UhMRhuCkwsleapF
pksxsZRksc2PJGvVNHNsp4EuyKnz+XvFeJ7NAZheKtoD5dKGk4GrJLhwebf04GyD
MeQIZMj539AaLo1funV58667cJaekV7/uvnX49MdAmZdrUteMMO42RzFOgA5JC8o
30DfxR+nABRAq+nopYBxqFAYSa+Eis0KSd2Gm5w2uuaGBqM1Nqw/EcS41aIFGAvL
gSsAP6ot2W9trWQWGkVvmprFQ64LQ5xwJHf74Ig+t2XjIQ6dkJH6DQjU1nUMMklq
60WagwKBgQDcuFx2GgxbED4Ruv7S/R5ysZuaVpw03S0rKcC3k8lE5xCmrM0E1Q6Z
U2h52ZO4WmXQuTCMh8PIsWKLg7BzacTWd91xGKWE3tD3wXK334fRwVa3ARKgaaH6
Rs1h+a0U8js5T//mf/NYYPKbltWrtXTcuwFt6XG2RWDzn1sPbf8h4wKBgQDJB5m7
ZWVY8+lE2h4QEvql6/YSRTYaYM788FvJDLfh1RS1u0NMu5mOo+0JAKj0JlLzBTsD
drktAHDsAtp0wqH8v2/mZnLYBmK35SwjQ4YNecvLQsIEtmD0USPWKrm1kGdwqohL
q90AJB5HSjBC5Q5vUZVij32WKuSbU+z/t3TH+wKBgBLrOyAQ3HzVgam/ki9XhkRY
XctmgmruYvUSNRcMqtoFLVAdcKikjDkHJjZUemBCQz3GuwS7LgnjUZbuB89g1luG
nfPASLOeEelZuWA3uy88dSWhAZi4mNrwIDuZDtXo4IFBXxPB0weTR/61KEHq+2Ng
fHcio1jEHkDEhCXk21qtAoGAROypvJfK+e06CPpTczm1Ba/8mIzCF6wptc7AYjA/
C5mDcYIIchRvKZdJ9HVBPcP/Lr/2+d+P8iwJdX1SNqkhmHwmXZ931QmA7pe3XIwt
9f3feOOwPCFF0BvRxcWBgBRAuOoC2B2q23oZAn/WCE6ImzHqEynh6lfZWdOhtsKO
cHMCgYBmdhIjJnWbqU5oHVQHN7sVCiRAScAUyTqlUCqB/qSpweZfR+aQ72thnx7C
0j+fdgy90is7ARo9Jam6jFtHwa9JXqH+g24Gdxk+smBeUgiZu63ZG/Z70L4okr4K
6BQlL1pZI4zGbG4H34TPraxvJVdVKVSLAXPur1pqgbJzD2nFUg==
-----END RSA PRIVATE KEY-----
`)

        sshInfo := sshrun.SSHInfo{
                UserName: "cb-user",
                PrivateKey: dummyKey,
                ServerPort: serverPort,
                Timeout: 3, // 3 sec
        }

	cmd := "whoami"

	// ssh: handshake failed: 
	// ssh: unable to authenticate, attempted methods [none publickey], no supported methods remain
	expectedErrMSG := "handshake failed"

        _, err := sshrun.SSHRun(sshInfo, cmd)
	if strings.Contains(err.Error(), expectedErrMSG) {
		// Note: Can't check cb-user without Private Key.
		return true
	}
	return false
}

func translateRootDiskSetupInfo(providerName string, reqInfo *cres.VMReqInfo) error {

	// get Provider's Meta Info
	cloudOSMetaInfo, err := cim.GetCloudOSMetaInfo(providerName)
	if err != nil {
		cblog.Error(err)
		return err
	}

	// for Root Disk Type
	switch strings.ToUpper(reqInfo.RootDiskType) {
	case "", "DEFAULT": // bypass
		reqInfo.RootDiskType = ""
	default: // TYPE1, TYPE2, TYPE3, ... or "pd-balanced", check validation, bypass
	        // TYPE2, ...
		if strings.Contains(strings.ToUpper(reqInfo.RootDiskType), "TYPE") {
			strType := strings.ToUpper(reqInfo.RootDiskType)
			typeNum, _ := strconv.Atoi(strings.Replace(strType, "TYPE", "", -1)) // "TYPE2" => "2" => 2
			typeMax := len(cloudOSMetaInfo.RootDiskType)
			if typeNum > typeMax {
				typeNum = typeMax
			}
			reqInfo.RootDiskType = cloudOSMetaInfo.RootDiskType[typeNum-1]
		} else if !validateRootDiskType(reqInfo.RootDiskType, cloudOSMetaInfo.RootDiskType) {
                        errMSG :=reqInfo.RootDiskType + " is not a valid Root Disk Type of " + providerName + "!"
                        cblog.Error(errMSG)
                        return fmt.Errorf(errMSG)
                }
	}


	// for Root Disk Size
	switch strings.ToUpper(reqInfo.RootDiskSize) {
        case "", "DEFAULT": // bypass
                reqInfo.RootDiskSize = ""
        default: // "100", bypass
		err := validateRootDiskSize(reqInfo.RootDiskSize)
		if err != nil {
                        errMSG :=reqInfo.RootDiskSize + " is not a valid Root Disk Size: " + err.Error() + "!"
                        cblog.Error(errMSG)
                        return fmt.Errorf(errMSG)
                }
        }
	return nil
}

func validateRootDiskType(diskType string, diskTypeList []string) bool {
	for _, v := range diskTypeList {
		if diskType == v {
			return true
		}
	}
	return false
}

func validateRootDiskSize(strSize string) error {
	_, err := strconv.Atoi(strSize)
	return err
}

func setNameId(ConnectionName string, vmInfo *cres.VMInfo, reqInfo *cres.VMReqInfo) error {
	
        // set Image Type & NameId (CSP dosen't return ImageType)
        if reqInfo.ImageType == cres.PublicImage {
		vmInfo.ImageType = cres.PublicImage
        	vmInfo.ImageIId.NameId = reqInfo.ImageIID.NameId		        		
        }
        if reqInfo.ImageType == cres.MyImage {                
		vmInfo.ImageType = cres.MyImage
        	if vmInfo.ImageIId.SystemId != "" {
	                // get MyImage's NameId
	                imageIIdInfo, err := iidRWLock.GetIIDbySystemID(iidm.IIDSGROUP, ConnectionName, rsMyImage, vmInfo.ImageIId)
	                if err != nil {
	                        cblog.Error(err)
	                        return err
	                }
	                vmInfo.ImageIId.NameId = imageIIdInfo.IId.NameId
	        }
        }

	// set VPC NameId
	if reqInfo.VpcIID.NameId != "" {
		vmInfo.VpcIID.NameId = reqInfo.VpcIID.NameId
	}

	// set Subnet NameId
	if reqInfo.SubnetIID.NameId != "" {
		vmInfo.SubnetIID.NameId = reqInfo.SubnetIID.NameId
	}

	// set SecurityGroups NameId
	for i, sgIID := range vmInfo.SecurityGroupIIds {
		IIdInfo, err := iidRWLock.GetIIDbySystemID(iidm.SGGROUP, ConnectionName, reqInfo.VpcIID.NameId, sgIID)
		if err != nil {
			cblog.Error(err)
			return err
		}
		vmInfo.SecurityGroupIIds[i].NameId = IIdInfo.IId.NameId
	}

	// When PublicImage Type, Set Disks NameId
	if reqInfo.ImageType == cres.PublicImage {
	        // set Data Disk NameId
	        for i, diskIID := range vmInfo.DataDiskIIDs {
	                IIdInfo, err := iidRWLock.GetIIDbySystemID(iidm.IIDSGROUP, ConnectionName, rsDisk, diskIID)
	                if err != nil {
	                        cblog.Error(err)
	                        return err
	                }
	                vmInfo.DataDiskIIDs[i].NameId = IIdInfo.IId.NameId
	        }
	}

	// When MyImage Type, Register auto-generated Disks into Spider-Server
	if reqInfo.ImageType == cres.MyImage {
	        for i, diskIID := range vmInfo.DataDiskIIDs {
	        	diskIID.NameId = reqInfo.IId.NameId + "-disk-" + strconv.Itoa(i)
	        	diskInfo, err := RegisterDisk(ConnectionName, diskIID)
	                if err != nil {
	                        cblog.Error(err)
	                        return err
	                }	                
	                vmInfo.DataDiskIIDs[i].NameId = diskInfo.IId.NameId
	        }
	}

	if reqInfo.KeyPairIID.NameId != "" {
		// set KeyPair SystemId
		vmInfo.KeyPairIId.NameId = reqInfo.KeyPairIID.NameId
	}

	return nil
}

type ResultVMInfo struct {
	vmInfo 	cres.VMInfo
	err	error
}

// (1) get IID:list
// (2) get VMInfo:list
func ListVM(connectionName string, rsType string) ([]*cres.VMInfo, error) {
	cblog.Info("call ListVM()")

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

	handler, err := cldConn.CreateVMHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (1) get IID:list
	iidInfoList, err := iidRWLock.ListIID(iidm.IIDSGROUP, connectionName, rsType)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	var infoList []*cres.VMInfo
	if iidInfoList == nil || len(iidInfoList) <= 0 {
		infoList = []*cres.VMInfo{}
		return infoList, nil
	}

	// (2) get VMInfo:list
	wg := new(sync.WaitGroup)
	infoList2 := []*cres.VMInfo{}
	var retChanInfos []chan ResultVMInfo
	for i:=0 ; i<len(iidInfoList); i++ {
		retChanInfos = append(retChanInfos, make(chan ResultVMInfo))
	}

	for idx, iidInfo := range iidInfoList {

		wg.Add(1)

		go getVMInfo(connectionName, handler, iidInfo.IId, retChanInfos[idx])

		wg.Done()

	}
	wg.Wait()

	var errList []string
	for idx, retChanInfo := range retChanInfos {
		chanInfo := <-retChanInfo

		if chanInfo.err  != nil {
			if checkNotFoundError(chanInfo.err) {
				cblog.Info(chanInfo.err) } else {
				errList = append(errList, connectionName + ":VM:" + iidInfoList[idx].IId.NameId + " # " + chanInfo.err.Error())
			}
		} else {
			infoList2 = append(infoList2, &chanInfo.vmInfo)
		}

		close(retChanInfo)
	}

	if len(errList) > 0 {
		cblog.Error(strings.Join(errList, "\n"))
		return nil, errors.New(strings.Join(errList, "\n"))
	}

	return infoList2, nil
}

func getVMInfo(connectionName string, handler cres.VMHandler, iid cres.IID, retInfo chan ResultVMInfo) { 

vmSPLock.RLock(connectionName, iid.NameId)
	// get resource(SystemId)
	info, err := handler.GetVM(getDriverIID(iid))
	if err != nil {
vmSPLock.RUnlock(connectionName, iid.NameId)
		cblog.Error(err)
		retInfo <- ResultVMInfo{cres.VMInfo{}, err}
		return 
	}

	// set ResourceInfo(IID.NameId)
	info.IId = getUserIID(iid)

	err = getSetNameId(connectionName, &info)
	if err != nil {
vmSPLock.RUnlock(connectionName, iid.NameId)
		cblog.Error(err)
		retInfo <- ResultVMInfo{cres.VMInfo{}, err}
		return 
	}
vmSPLock.RUnlock(connectionName, iid.NameId)


        cldConn, err := ccm.GetCloudConnection(connectionName)
        if err != nil {
                cblog.Error(err)
                return 
        }

        // check Winddows GuestOS
        isWindowsOS := false
        isWindowsOS, err = checkImageWindowsOS(cldConn, info.ImageType, info.ImageIId)
        if err != nil {
                if strings.Contains(err.Error(), "yet!") {
                        cblog.Info(err)
                } else {
                        cblog.Error(err)
                        return 
                }
        }

        if isWindowsOS {
                info.VMUserId = "Administrator"
		info.SSHAccessPoint = "RDP: " + info.PublicIP
        } else {
                info.VMUserId = "cb-user"
                // current: Assume 22 port, except Cloud-Twin, by powerkim, 2021.03.24.
                if info.SSHAccessPoint == "" {
                        info.SSHAccessPoint = info.PublicIP + ":22"
                }
        }

	retInfo <- ResultVMInfo{info, nil}
}


func getSetNameId(ConnectionName string, vmInfo *cres.VMInfo) error {

        // set Image Type and NameId (CSP dosen't return ImageType)
	// find Image.SystemId in MyImage to get ImageType
	// default imagetype is Public
	vmInfo.ImageType = cres.PublicImage
	if vmInfo.ImageIId.SystemId != "" {
		// get MyImage's NameId
		imageIIdInfo, err := iidRWLock.GetIIDbySystemID(iidm.IIDSGROUP, ConnectionName, rsMyImage, vmInfo.ImageIId)
		if err != nil {
			cblog.Error(err)
			return err
		}
		if imageIIdInfo != nil && imageIIdInfo.IId.NameId != "" {
			vmInfo.ImageType = cres.MyImage
			vmInfo.ImageIId.NameId = imageIIdInfo.IId.NameId
		}
	}
        if vmInfo.ImageType == cres.PublicImage {
        	vmInfo.ImageIId.NameId = vmInfo.ImageIId.SystemId		        		
        }

	if vmInfo.VpcIID.SystemId != "" {
		// set VPC NameId
		IIdInfo, err := iidRWLock.GetIIDbySystemID(iidm.IIDSGROUP, ConnectionName, rsVPC, vmInfo.VpcIID)
		if err != nil {
			cblog.Error(err)
			return err
		}
		vmInfo.VpcIID.NameId = IIdInfo.IId.NameId
	}

	if vmInfo.SubnetIID.SystemId != "" {
		// set Subnet NameId
		IIdInfo, err := iidRWLock.GetIIDbySystemID(iidm.SUBNETGROUP, ConnectionName, vmInfo.VpcIID.NameId, vmInfo.SubnetIID)  // reqInfo.VpcIID.NameId => rsType
		if err != nil {
			cblog.Error(err)
			return err
		}
		vmInfo.SubnetIID.NameId = IIdInfo.IId.NameId
	}

	// set SecurityGroups NameId
	for i, sgIID := range vmInfo.SecurityGroupIIds {
		IIdInfo, err := iidRWLock.GetIIDbySystemID(iidm.SGGROUP, ConnectionName, vmInfo.VpcIID.NameId, sgIID)  // reqInfo.VpcIID.NameId => rsType
		if err != nil {
			cblog.Error(err)
			return err
		}
		vmInfo.SecurityGroupIIds[i].NameId = IIdInfo.IId.NameId
	}

	if vmInfo.KeyPairIId.SystemId != "" {
		// set KeyPair NameId
		IIdInfo, err := iidRWLock.GetIIDbySystemID(iidm.IIDSGROUP, ConnectionName, rsKey, vmInfo.KeyPairIId)
		if err != nil {
			cblog.Error(err)
			return err
		}
		vmInfo.KeyPairIId.NameId = IIdInfo.IId.NameId
	}


	// set Data Disk NameId	
	for i, diskIID := range vmInfo.DataDiskIIDs {
                IIdInfo, err := iidRWLock.GetIIDbySystemID(iidm.IIDSGROUP, ConnectionName, rsDisk, diskIID)
                if err != nil {
                        cblog.Error(err)
					return err
				}
                vmInfo.DataDiskIIDs[i].NameId = IIdInfo.IId.NameId
	}

	return nil
}

// (1) get IID(NameId)
// (2) get resource(SystemId)
// (3) set ResourceInfo(IID.NameId)
func GetVM(connectionName string, rsType string, nameID string) (*cres.VMInfo, error) {
	cblog.Info("call GetVM()")

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

	handler, err := cldConn.CreateVMHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	vmSPLock.RLock(connectionName, nameID)
	defer vmSPLock.RUnlock(connectionName, nameID)

	// (1) get IID(NameId)
	iidInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsType, cres.IID{nameID, ""})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (2) get resource(SystemId)
	info, err := handler.GetVM(getDriverIID(iidInfo.IId))
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (3) set ResourceInfo(IID.NameId)
	// set ResourceInfo
	info.IId = getUserIID(iidInfo.IId)

	err = getSetNameId(connectionName, &info)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
/*
	// set sg NameId from VPCNameId-SecurityGroupNameId
	// IID.NameID format => {VPC NameID} + SG_DELIMITER + {SG NameID}
	for i, sgIID := range info.SecurityGroupIIds {
		vpc_sg_nameid := strings.Split(sgIID.NameId, SG_DELIMITER)
		info.SecurityGroupIIds[i].NameId = vpc_sg_nameid[1]
	}
*/

        // check Winddows GuestOS
        isWindowsOS := false
        isWindowsOS, err = checkImageWindowsOS(cldConn, info.ImageType, info.ImageIId)
        if err != nil {
                if strings.Contains(err.Error(), "yet!") {
                        cblog.Info(err)
                } else {
                        cblog.Error(err)
                        return nil, err
                }
        }

        if isWindowsOS {
                info.VMUserId = "Administrator"
		info.SSHAccessPoint = "RDP: " + info.PublicIP
        } else {
                info.VMUserId = "cb-user"
                // current: Assume 22 port, except Cloud-Twin, by powerkim, 2021.03.24.
                if info.SSHAccessPoint == "" {
                        info.SSHAccessPoint = info.PublicIP + ":22"
                }
        }

	return &info, nil
}

func GetCSPVM(connectionName string, rsType string, cspID string) (*cres.VMInfo, error) {
        cblog.Info("call GetVM()")

        // check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        cspID, err = EmptyCheckAndTrim("cspID", cspID)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        cldConn, err := ccm.GetCloudConnection(connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        handler, err := cldConn.CreateVMHandler()
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        info, err := handler.GetVM(cres.IID{"", cspID})
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        return &info, nil
}

// (1) get IID:list
// (2) get VMStatusInfo:list
func ListVMStatus(connectionName string, rsType string) ([]*cres.VMStatusInfo, error) {
	cblog.Info("call ListVMStatus()")

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

	handler, err := cldConn.CreateVMHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (1) get IID:list
	iidInfoList, err := iidRWLock.ListIID(iidm.IIDSGROUP, connectionName, rsType)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	var infoList []*cres.VMStatusInfo
	if iidInfoList == nil || len(iidInfoList) <= 0 {
		infoList = []*cres.VMStatusInfo{}
		return infoList, nil
	}

	// (2) get VMStatusInfo List with iidInoList
	infoList2 := []*cres.VMStatusInfo{}
	for _, iidInfo := range iidInfoList {

/* temporarily unlock
vmSPLock.RLock(connectionName, iidInfo.IId.NameId)
*/

		// 2. get CSP:VMStatus(SystemId)
		statusInfo, err := handler.GetVMStatus(getDriverIID(iidInfo.IId)) // type of info => string
		if err != nil {
//vmSPLock.RUnlock(connectionName, iidInfo.IId.NameId)
			if checkNotFoundError(err) {
				cblog.Info(err)
				continue
			}
			cblog.Error(err)
			return nil, err
		}
//vmSPLock.RUnlock(connectionName, iidInfo.IId.NameId)

		infoList2 = append(infoList2, &cres.VMStatusInfo{getUserIID(iidInfo.IId), statusInfo})
	}

	return infoList2, nil
}

// (1) get IID(NameId)
// (2) get CSP:VMStatus(SystemId)
func GetVMStatus(connectionName string, rsType string, nameID string) (cres.VMStatus, error) {
	cblog.Info("call GetVMStatus()")

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

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	handler, err := cldConn.CreateVMHandler()
	if err != nil {
		cblog.Error(err)
		return "", err
	}
/* temporarily unlocked
	vmSPLock.RLock(connectionName, nameID)
	defer vmSPLock.RUnlock(connectionName, nameID)
*/

	// (1) get IID(NameId)
	iidInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsType, cres.IID{nameID, ""})
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	// (2) get CSP:VMStatus(SystemId)
	info, err := handler.GetVMStatus(getDriverIID(iidInfo.IId)) // type of info => string
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	return info, nil
}

// (1) get IID(NameId)
// (2) control CSP:VM(SystemId)
func ControlVM(connectionName string, rsType string, nameID string, action string) (cres.VMStatus, error) {
	cblog.Info("call ControlVM()")

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	handler, err := cldConn.CreateVMHandler()
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	vmSPLock.RLock(connectionName, nameID)
	defer vmSPLock.RUnlock(connectionName, nameID)

	// (1) get IID(NameId)
	iidInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsType, cres.IID{nameID, ""})
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	// (2) control CSP:VM(SystemId)
	vmIID := getDriverIID(iidInfo.IId)

	var info cres.VMStatus

	switch strings.ToLower(action) {
	case "suspend":
		info, err = handler.SuspendVM(vmIID)
	case "resume":
		info, err = handler.ResumeVM(vmIID)
	case "reboot":
		info, err = handler.RebootVM(vmIID)
	default:
		return "", fmt.Errorf(action + " is not a valid action!!")

	}
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	return info, nil
}
