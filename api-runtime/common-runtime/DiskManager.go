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

	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	iidm "github.com/cloud-barista/cb-spider/cloud-control-manager/iid-manager"
)


//================ Disk Handler

// UserIID{UserID, CSP-ID} => SpiderIID{UserID, SP-XID:CSP-ID}
// (1) check existence(UserID)
// (2) get resource info(CSP-ID)
// (3) create spiderIID: {UserID, SP-XID:CSP-ID}
// (4) insert spiderIID
func RegisterDisk(connectionName string, userIID cres.IID) (*cres.DiskInfo, error) {
        cblog.Info("call RegisterDisk()")

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

        rsType := rsDisk

        cldConn, err := ccm.GetCloudConnection(connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        handler, err := cldConn.CreateDiskHandler()
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        diskSPLock.Lock(connectionName, userIID.NameId)
        defer diskSPLock.Unlock(connectionName, userIID.NameId)

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
        getInfo, err := handler.GetDisk( cres.IID{getMSShortID(userIID.SystemId), userIID.SystemId} )
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // (3) create spiderIID: {UserID, SP-XID:CSP-ID}
        //     ex) spiderIID {"vpc-01", "vpc-01-9m4e2mr0ui3e8a215n4g:i-0bc7123b7e5cbf79d"}
        // Do not user NameId, because Azure driver use it like SystemId
        systemId := getMSShortID(getInfo.IId.SystemId)
        spiderIId := cres.IID{userIID.NameId, systemId + ":" + getInfo.IId.SystemId}


        // (4) insert spiderIID
        // insert Disk SpiderIID to metadb
        _, err = iidRWLock.CreateIID(iidm.IIDSGROUP, connectionName, rsType, spiderIId)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // set up Disk User IID for return info
        getInfo.IId = userIID

        return &getInfo, nil
}

// (1) check exist(NameID)
// (2) generate SP-XID and create reqIID, driverIID
// (3) create Resource
// (4) create spiderIID: {reqNameID, "driverNameID:driverSystemID"}
// (5) insert spiderIID
// (6) create userIID
func CreateDisk(connectionName string, rsType string, reqInfo cres.DiskInfo) (*cres.DiskInfo, error) {
        cblog.Info("call CreateDisk()")

        // check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
/*
        emptyPermissionList := []string{
                "resources.IID:SystemId",
                "resources.DiskInfo:Status",
        }

        err = ValidateStruct(reqInfo, emptyPermissionList)
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

        handler, err := cldConn.CreateDiskHandler()
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

diskSPLock.Lock(connectionName, reqInfo.IId.NameId)
defer diskSPLock.Unlock(connectionName, reqInfo.IId.NameId)

        // (1) check exist(NameID)
        bool_ret, err := iidRWLock.IsExistIID(iidm.IIDSGROUP, connectionName, rsType, reqInfo.IId)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        if bool_ret == true {
                err := fmt.Errorf(reqInfo.IId.NameId + " already exists!")
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
	if strings.ToLower(reqInfo.DiskType) == "default" {
		reqInfo.DiskType = ""
	}
        // (3) create Resource
        info, err := handler.CreateDisk(reqInfo)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // (4) create spiderIID: {reqNameID, "driverNameID:driverSystemID"}
        //     ex) spiderIID {"seoul-service", "vm-01-9m4e2mr0ui3e8a215n4g:i-0bc7123b7e5cbf79d"}
        spiderIId := cres.IID{reqIId.NameId, info.IId.NameId + ":" + info.IId.SystemId}

        // (5) insert spiderIID
        iidInfo, err := iidRWLock.CreateIID(iidm.IIDSGROUP, connectionName, rsType, spiderIId)
        if err != nil {
                cblog.Error(err)
                // rollback
                _, err2 := handler.DeleteDisk(info.IId)
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

// (1) get IID:list
// (2) get DiskInfo:list
// (3) set userIID, and ...
func ListDisk(connectionName string, rsType string) ([]*cres.DiskInfo, error) {
        cblog.Info("call ListDisk()")

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

        handler, err := cldConn.CreateDiskHandler()
        if err != nil {
                cblog.Error(err)
                return nil, err
        }


        // (1) get IID:list
        // (1) get IID:list
        iidInfoList, err := iidRWLock.ListIID(iidm.IIDSGROUP, connectionName, rsType)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        var infoList []*cres.DiskInfo
        if iidInfoList == nil || len(iidInfoList) <= 0 {
                infoList = []*cres.DiskInfo{}
                return infoList, nil
        }

        // (2) Get DiskInfo-list with IID-list
        infoList2 := []*cres.DiskInfo{}
        for _, iidInfo := range iidInfoList {

diskSPLock.RLock(connectionName, iidInfo.IId.NameId)

                // get resource(SystemId)
                info, err := handler.GetDisk(getDriverIID(iidInfo.IId))
                if err != nil {
diskSPLock.RUnlock(connectionName, iidInfo.IId.NameId)
                        if checkNotFoundError(err) {
                                cblog.Info(err)
                                continue
                        }
                        cblog.Error(err)
                        return nil, err
                }
diskSPLock.RUnlock(connectionName, iidInfo.IId.NameId)

                info.IId = getUserIID(iidInfo.IId)

		if info.Status == cres.DiskAttached {
			// get Source VM's IID with VM's SystemId
			vmIIdInfo, err := iidRWLock.GetIIDbySystemID(iidm.IIDSGROUP, connectionName, rsVM, info.OwnerVM)
			if err != nil {
				cblog.Error(err)
				return nil, err
			}
			info.OwnerVM.NameId = vmIIdInfo.IId.NameId

		}

		infoList2 = append(infoList2, &info)
	}

	return infoList2, nil
}

	// (1) get IID(NameId)
	// (2) get resource(SystemId)
	// (3) set ResourceInfo(IID.NameId)
func GetDisk(connectionName string, rsType string, nameID string) (*cres.DiskInfo, error) {
        cblog.Info("call GetDisk()")

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

        handler, err := cldConn.CreateDiskHandler()
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

diskSPLock.RLock(connectionName, nameID)
defer diskSPLock.RUnlock(connectionName, nameID)

        // (1) get IID(NameId)
        iidInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsType, cres.IID{nameID, ""})
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // (2) get resource(SystemId)
        info, err := handler.GetDisk(getDriverIID(iidInfo.IId))
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // (3) set ResourceInfo(IID.NameId)
        // set ResourceInfo
        info.IId = getUserIID(iidInfo.IId)

	if info.Status == cres.DiskAttached {
		// get Source VM's IID with VM's SystemId
		vmIIdInfo, err := iidRWLock.GetIIDbySystemID(iidm.IIDSGROUP, connectionName, rsVM, info.OwnerVM)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		info.OwnerVM.NameId = vmIIdInfo.IId.NameId
	}

        return &info, nil
}

func ChangeDiskSize(connectionName string, diskName string, size string) (bool, error) {
        cblog.Info("call ChangeDiskSize()")

        // check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
                cblog.Error(err)
                return false, err
        }
        diskName, err = EmptyCheckAndTrim("diskName", diskName)
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        size, err = EmptyCheckAndTrim("size", size)
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        cldConn, err := ccm.GetCloudConnection(connectionName)
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        handler, err := cldConn.CreateDiskHandler()
        if err != nil {
                cblog.Error(err)
                return false, err
        }

diskSPLock.Lock(connectionName, diskName)
defer diskSPLock.Unlock(connectionName, diskName)

        // (1) check exist(diskName)
        diskIIDInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsDisk, cres.IID{diskName, ""})
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        // (2) change disk size
        info, err := handler.ChangeDiskSize(getDriverIID(diskIIDInfo.IId), size)
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        return info, nil
}

// (1) check exist(NameID) and VMs
// (2) attach disk to VM
// (3) Set ResoureInfo
func AttachDisk(connectionName string, diskName string, ownerVMName string) (*cres.DiskInfo, error) {
        cblog.Info("call AttachDisk()")

        // check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        diskName, err = EmptyCheckAndTrim("diskName", diskName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        ownerVMName, err = EmptyCheckAndTrim("ownerVMName", ownerVMName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        cldConn, err := ccm.GetCloudConnection(connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        handler, err := cldConn.CreateDiskHandler()
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

diskSPLock.Lock(connectionName, diskName)
defer diskSPLock.Unlock(connectionName, diskName)

        // (1) check exist(diskName)
        diskIIDInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsDisk, cres.IID{diskName, ""})
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // (1) check exist(ownerVMName)
        vmIIDInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsVM, cres.IID{ownerVMName, ""})
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // (2) attach disk to VM
        info, err := handler.AttachDisk(getDriverIID(diskIIDInfo.IId), getDriverIID(vmIIDInfo.IId))
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // (3) set ResourceInfo(userIID)
        info.IId = getUserIID(diskIIDInfo.IId)

        // set OwnerVM's UserIID
        info.OwnerVM = getUserIID(vmIIDInfo.IId)

        return &info, nil
}

// (1) check exist(NameID)
// (2) detach disk from VM
func DetachDisk(connectionName string, diskName string, ownerVMName string) (bool, error) {
        cblog.Info("call DetachDisk()")

        // check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        diskName, err = EmptyCheckAndTrim("diskName", diskName)
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        cldConn, err := ccm.GetCloudConnection(connectionName)
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        handler, err := cldConn.CreateDiskHandler()
        if err != nil {
                cblog.Error(err)
                return false, err
        }

diskSPLock.Lock(connectionName, diskName)
defer diskSPLock.Unlock(connectionName, diskName)

        // (1) check exist(diskName)
        diskIIDInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsDisk, cres.IID{diskName, ""})
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        // (1) check exist(ownerVMName)
        vmIIDInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsVM, cres.IID{ownerVMName, ""})
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        // (2) detach disk from VM
        info, err := handler.DetachDisk(getDriverIID(diskIIDInfo.IId), getDriverIID(vmIIDInfo.IId))
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        return info, nil
}

