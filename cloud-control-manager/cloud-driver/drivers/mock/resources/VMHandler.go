// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Mock Driver.
//
// by CB-Spider Team, 2020.10.

package resources

import (
	"fmt"
	"sync"
	"time"

	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

var vmInfoMap map[string][]*irs.VMInfo
var vmStatusInfoMap map[string][]*irs.VMStatusInfo

type MockVMHandler struct {
	Region   idrv.RegionInfo
	MockName string
}

func init() {
	vmInfoMap = make(map[string][]*irs.VMInfo)
	vmStatusInfoMap = make(map[string][]*irs.VMStatusInfo)
}

var vmMapLock = new(sync.RWMutex)

func (vmHandler *MockVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (irs.VMInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called StartVM()!")

	mockName := vmHandler.MockName
	vmReqInfo.IId.SystemId = vmReqInfo.IId.NameId

	validatedImageIID := irs.IID{}
	// public image validation
	if vmReqInfo.ImageType == irs.PublicImage {
		imageHandler := MockImageHandler{mockName}
		validatedImgInfo, err := imageHandler.GetImage(vmReqInfo.ImageIID)
		if err != nil {
			cblogger.Error(err)
			return irs.VMInfo{}, err
		}
		validatedImageIID = validatedImgInfo.IId
	}

	// MyImage validation
	if vmReqInfo.ImageType == irs.MyImage {
		myImageHandler := MockMyImageHandler{mockName}
		validatedMyImgInfo, err := myImageHandler.GetMyImage(vmReqInfo.ImageIID)
		if err != nil {
			cblogger.Error(err)
			return irs.VMInfo{}, err
		}
		validatedImageIID = validatedMyImgInfo.IId
	}

	// spec validation
	vmSpecHandler := MockVMSpecHandler{mockName}
	validatedSpecInfo, err := vmSpecHandler.GetVMSpec(vmReqInfo.VMSpecName)
	if err != nil {
		cblogger.Error(err)
		return irs.VMInfo{}, err
	}

	// vpc validation
	vpcHandler := MockVPCHandler{mockName}
	validatedVPCInfo, err := vpcHandler.GetVPC(vmReqInfo.VpcIID)
	if err != nil {
		cblogger.Error(err)
		return irs.VMInfo{}, err
	}

	// subnet validation
	var validatedSubnetInfo *irs.SubnetInfo = nil
	for _, info := range validatedVPCInfo.SubnetInfoList {
		if info.IId.NameId == vmReqInfo.SubnetIID.NameId {
			validatedSubnetInfo = &info
			break;
		}
	}
	if validatedSubnetInfo == nil {
		errMSG := vmReqInfo.SubnetIID.NameId + " subnet iid does not exist!!"
		cblogger.Error(errMSG)
		return irs.VMInfo{}, fmt.Errorf(errMSG)
	}

	// sg validation
	securityHandler := MockSecurityHandler{mockName}
	sgInfoList, err := securityHandler.ListSecurity()
	if err != nil {
		cblogger.Error(err)
		return irs.VMInfo{}, err
	}
	validatedSgIIDs := []irs.IID{}
	for _, info1 := range vmReqInfo.SecurityGroupIIDs {
		flg := false
		for _, info2 := range sgInfoList {
			if (*info2).IId.NameId == info1.NameId {
				validatedSgIIDs = append(validatedSgIIDs, info2.IId)
				flg = true
				break
			}
		}
		if !flg {
			errMSG := info1.NameId + " security group iid does not exist!!"
			cblogger.Error(errMSG)
			return irs.VMInfo{}, fmt.Errorf(errMSG)
		}
	}

        // data disk validation
        diskHandler := MockDiskHandler{mockName}
        diskInfoList, err := diskHandler.ListDisk()
        if err != nil {
                cblogger.Error(err)
                return irs.VMInfo{}, err
        }
        validatedDiskIIDs := []irs.IID{}
        for _, info1 := range vmReqInfo.DataDiskIIDs {
                flg := false
                for _, info2 := range diskInfoList {
                        if (*info2).IId.NameId == info1.NameId {
                                validatedDiskIIDs = append(validatedDiskIIDs, info2.IId)
                                flg = true
                        }
                }
                if !flg {
                        errMSG := info1.NameId + " Data Disk iid does not exist!!"
                        cblogger.Error(errMSG)
                        return irs.VMInfo{}, fmt.Errorf(errMSG)
                }
        }

	// keypair validation
	keyPairHandler := MockKeyPairHandler{mockName}
	validatedKeyPairInfo, err := keyPairHandler.GetKey(vmReqInfo.KeyPairIID)
	if err != nil {
		cblogger.Error(err)
		return irs.VMInfo{}, err
	}

	// vm creation
	vmInfo := irs.VMInfo{
		IId:       vmReqInfo.IId,
		StartTime: time.Now(),

		Region:            irs.RegionInfo{vmHandler.Region.Region, vmHandler.Region.Zone},
		ImageIId:          validatedImageIID,
		VMSpecName:        validatedSpecInfo.Name,
		VpcIID:            validatedVPCInfo.IId,
		SubnetIID:         validatedSubnetInfo.IId,
		SecurityGroupIIds: validatedSgIIDs,

		KeyPairIId: validatedKeyPairInfo.IId,

		VMUserId:     vmReqInfo.VMUserId,
		VMUserPasswd: vmReqInfo.VMUserPasswd,

		NetworkInterface: "mockni0",
		PublicIP:         "4.3.2.1",
		PublicDNS:        vmReqInfo.IId.NameId + ".spider.barista.com",
		PrivateIP:        "1.2.3.4",
		PrivateDNS:       vmReqInfo.IId.NameId + ".spider.barista.com",

		VMBootDisk:  "/dev/sda1",
		VMBlockDisk: "/dev/sda1",

		RootDiskType:  "SSD", 
		RootDiskSize:  "32",
		RootDeviceName:  "/dev/sda1",

		DataDiskIIDs:  validatedDiskIIDs,

		KeyValueList: nil,
	}

	// attach disks
	for _, diskIID := range validatedDiskIIDs {
		_, err := justAttachDisk(mockName, diskIID, vmReqInfo.IId)
		if err != nil {
			cblogger.Error(err)
			return irs.VMInfo{}, err
		}
	}

vmMapLock.Lock()
defer vmMapLock.Unlock()

	infoList, _ := vmInfoMap[mockName]
	infoList = append(infoList, &vmInfo)
	vmInfoMap[mockName] = infoList

	// vm status creation
	vmStatusInfo := irs.VMStatusInfo{vmReqInfo.IId, irs.Running}

	statusInfoList, _ := vmStatusInfoMap[mockName]
	statusInfoList = append(statusInfoList, &vmStatusInfo)
	vmStatusInfoMap[mockName] = statusInfoList

	return vmInfo, nil
}

func (vmHandler *MockVMHandler) SuspendVM(iid irs.IID) (irs.VMStatus, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called SuspendVM()!")

	mockName := vmHandler.MockName

vmMapLock.Lock()
defer vmMapLock.Unlock()

	statusInfoList, ok := vmStatusInfoMap[mockName]
	if !ok {
		errMSG := mockName + " vm status does not exist!!"
		cblogger.Error(errMSG)
		return "", fmt.Errorf(errMSG)
	}

	var validatedStatusInfo *irs.VMStatusInfo = nil
	for _, info := range statusInfoList {
		if (*info).IId.NameId == iid.NameId {
			validatedStatusInfo = info
		}
	}
	if validatedStatusInfo == nil {
		errMSG := iid.NameId + " status iid does not exist!!"
		cblogger.Error(errMSG)
		return "", fmt.Errorf(errMSG)
	}

	validatedStatusInfo.VmStatus = irs.Suspended
	return irs.Suspending, nil
}

func (vmHandler *MockVMHandler) ResumeVM(iid irs.IID) (irs.VMStatus, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called ResumeVM()!")

	mockName := vmHandler.MockName

vmMapLock.Lock()
defer vmMapLock.Unlock()

	statusInfoList, ok := vmStatusInfoMap[mockName]
	if !ok {

		errMSG := mockName + " vm status does not exist!!"
		cblogger.Error(errMSG)
		return "", fmt.Errorf(errMSG)
	}

	var validatedStatusInfo *irs.VMStatusInfo = nil
	for _, info := range statusInfoList {
		if (*info).IId.NameId == iid.NameId {
			validatedStatusInfo = info
		}
	}
	if validatedStatusInfo == nil {
		errMSG := iid.NameId + " vm status iid does not exist!!"
		cblogger.Error(errMSG)
		return "", fmt.Errorf(errMSG)
	}

	validatedStatusInfo.VmStatus = irs.Running
	return irs.Resuming, nil
}

func (vmHandler *MockVMHandler) RebootVM(iid irs.IID) (irs.VMStatus, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called RebootVM()!")

	mockName := vmHandler.MockName

vmMapLock.Lock()
defer vmMapLock.Unlock()

	statusInfoList, ok := vmStatusInfoMap[mockName]
	if !ok {
		errMSG := mockName + " vm status does not exist!!"
		cblogger.Error(errMSG)
		return "", fmt.Errorf(errMSG)
	}

	var validatedStatusInfo *irs.VMStatusInfo = nil
	for _, info := range statusInfoList {
		if (*info).IId.NameId == iid.NameId {
			validatedStatusInfo = info
		}
	}
	if validatedStatusInfo == nil {
		errMSG := iid.NameId + " vm status iid does not exist!!"
		cblogger.Error(errMSG)
		return "", fmt.Errorf(errMSG)
	}

	if validatedStatusInfo.VmStatus == irs.Suspended {
		errMSG := "reboot not supported in SUSPENDED status"
		cblogger.Error(errMSG)
		return "", fmt.Errorf(errMSG)
	}

	validatedStatusInfo.VmStatus = irs.Running
	return irs.Rebooting, nil
}

func (vmHandler *MockVMHandler) TerminateVM(iid irs.IID) (irs.VMStatus, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called TerminateVM()!")

        mockName := vmHandler.MockName

vmMapLock.Lock()
defer vmMapLock.Unlock()

        infoList, ok := vmInfoMap[mockName]
        if !ok {
		errMSG := iid.NameId + " vm iid does not exist!!"
                return "", fmt.Errorf(errMSG)
        }

        statusInfoList, ok := vmStatusInfoMap[mockName]
        if !ok {
		errMSG := iid.NameId + " vm iid does not exist!!"
                return "", fmt.Errorf(errMSG)
        }

	for idx, info := range infoList {
		if info.IId.SystemId == iid.SystemId {
			for _, diskIID := range info.DataDiskIIDs {
				justDetachDisk(mockName, diskIID, info.IId) 
			}
			infoList = append(infoList[:idx], infoList[idx+1:]...)
		}
	}
	vmInfoMap[mockName] = infoList

	for idx, info := range statusInfoList {
		if info.IId.SystemId == iid.SystemId {
			statusInfoList = append(statusInfoList[:idx], statusInfoList[idx+1:]...)
		}
	}
	vmStatusInfoMap[mockName] = statusInfoList

	return irs.Terminating, nil
}

func (vmHandler *MockVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called ListVMStatus()!")

	mockName := vmHandler.MockName

vmMapLock.RLock()
defer vmMapLock.RUnlock()

	infoList, ok := vmStatusInfoMap[mockName]
	if !ok {
		return []*irs.VMStatusInfo{}, nil
	}
	// cloning list of VM Status
	return CloneVMStatusInfoList(infoList), nil
}

func CloneVMStatusInfoList(srcInfoList []*irs.VMStatusInfo) []*irs.VMStatusInfo {
        clonedInfoList := []*irs.VMStatusInfo{}
        for _, srcInfo := range srcInfoList {
                clonedInfo := CloneVMStatusInfo(*srcInfo)
                clonedInfoList = append(clonedInfoList, &clonedInfo)
        }
        return clonedInfoList
}

func CloneVMStatusInfo(srcInfo irs.VMStatusInfo) irs.VMStatusInfo {
        /*
		type VMStatusInfo struct {
			IId      IID // {NameId, SystemId}
			VmStatus VMStatus
		}

	*/
	
	// clone VMStatusInfo
	clonedInfo := irs.VMStatusInfo {
		IId:            irs.IID{srcInfo.IId.NameId, srcInfo.IId.SystemId},
		VmStatus:	srcInfo.VmStatus,
	}

	return clonedInfo
}



func (vmHandler *MockVMHandler) GetVMStatus(iid irs.IID) (irs.VMStatus, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called GetVMStatus()!")

	mockName := vmHandler.MockName

vmMapLock.RLock()
defer vmMapLock.RUnlock()

        infoList, ok := vmStatusInfoMap[mockName]
        if !ok {
		errMSG := iid.NameId + " vm iid does not exist!!"
                return "", fmt.Errorf(errMSG)
        }


	for _, info := range infoList {
		if (*info).IId.NameId == iid.NameId {
			return (*info).VmStatus, nil
		}
	}

	errMSG := iid.NameId + " status iid does not exist!!"
	cblogger.Error(errMSG)
	return "", fmt.Errorf(errMSG)
}

func (vmHandler *MockVMHandler) ListVM() ([]*irs.VMInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called ListVM()!")

	mockName := vmHandler.MockName

vmMapLock.RLock()
defer vmMapLock.RUnlock()

	infoList, ok := vmInfoMap[mockName]
	if !ok {
		return []*irs.VMInfo{}, nil
	}

	// cloning list of VM
	return CloneVMInfoList(infoList), nil
}

func CloneVMInfoList(srcInfoList []*irs.VMInfo) []*irs.VMInfo {
        clonedInfoList := []*irs.VMInfo{}
        for _, srcInfo := range srcInfoList {
                clonedInfo := CloneVMInfo(*srcInfo)
                clonedInfoList = append(clonedInfoList, &clonedInfo)
        }
        return clonedInfoList
}

func CloneVMInfo(srcInfo irs.VMInfo) irs.VMInfo {
        /*
		type VMInfo struct {
			IId       IID       // {NameId, SystemId}
			StartTime time.Time // Timezone: based on cloud-barista server location.

			Region            RegionInfo //  ex) {us-east1, us-east1-c} or {ap-northeast-2}
			ImageType         ImageType // PublicImage | MyImage
			ImageIId          IID
			VMSpecName        string //  instance type or flavour, etc... ex) t2.micro or f1.micro
			VpcIID            IID
			SubnetIID         IID   // AWS, ex) subnet-8c4a53e4
			SecurityGroupIIds []IID // AWS, ex) sg-0b7452563e1121bb6

			KeyPairIId IID

			RootDiskType    string  // "SSD(gp2)", "Premium SSD", ...
			RootDiskSize    string  // "default", "50", "1000" (GB)
			RootDeviceName  string // "/dev/sda1", ...

			DataDiskIIDs []IID

			VMBootDisk      string // Deprecated soon
			VMBlockDisk     string // Deprecated soon

                        VMUserId     string // ex) user1
                        VMUserPasswd string

                        NetworkInterface string // ex) eth0
                        PublicIP         string
                        PublicDNS        string
                        PrivateIP        string
                        PrivateDNS       string

			SSHAccessPoint string // ex) 10.2.3.2:22, 123.456.789.123:4321

			KeyValueList []KeyValue
		}
        */

        // clone VMInfo
        clonedInfo := irs.VMInfo{
                IId:            irs.IID{srcInfo.IId.NameId, srcInfo.IId.SystemId},
                StartTime:      srcInfo.StartTime,
		Region:		srcInfo.Region,
                ImageType:      srcInfo.ImageType,
                ImageIId:       srcInfo.ImageIId,
                VMSpecName:     srcInfo.VMSpecName,
                VpcIID:         irs.IID{srcInfo.VpcIID.NameId, srcInfo.VpcIID.SystemId},
                SubnetIID:      irs.IID{srcInfo.SubnetIID.NameId, srcInfo.SubnetIID.SystemId},
		SecurityGroupIIds: cloneIIDArray(srcInfo.SecurityGroupIIds),
                KeyPairIId:     irs.IID{srcInfo.KeyPairIId.NameId, srcInfo.KeyPairIId.SystemId},

		RootDiskType:   srcInfo.RootDiskType,
		RootDiskSize:   srcInfo.RootDiskSize,
		RootDeviceName: srcInfo.RootDeviceName,

		DataDiskIIDs:   cloneIIDArray(srcInfo.DataDiskIIDs),

		VMBootDisk:     srcInfo.VMBootDisk,
		VMBlockDisk:    srcInfo.VMBlockDisk,

		VMUserId:       srcInfo.VMUserId,
		VMUserPasswd:   srcInfo.VMUserPasswd,
		NetworkInterface:   srcInfo.NetworkInterface,
		PublicIP:       srcInfo.PublicIP,
		PublicDNS:      srcInfo.PublicDNS,
		PrivateIP:      srcInfo.PrivateIP,
		PrivateDNS:     srcInfo.PrivateDNS,

		SSHAccessPoint: srcInfo.SSHAccessPoint,

                KeyValueList:   srcInfo.KeyValueList, // now, do not need cloning
        }

        return clonedInfo
}

func cloneIIDArray(srcIIDArray []irs.IID) []irs.IID {
	clonedIIDs := []irs.IID{}
	for _, iid := range srcIIDArray {
		clonedIIDs = append(clonedIIDs, irs.IID{iid.NameId, iid.SystemId})
	}
	return clonedIIDs
}

func (vmHandler *MockVMHandler) GetVM(iid irs.IID) (irs.VMInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called GetVM()!")

        mockName := vmHandler.MockName

vmMapLock.RLock()
defer vmMapLock.RUnlock()

        infoList, ok := vmInfoMap[mockName]
        if !ok {
		errMSG := iid.NameId + " vm iid does not exist!!"
		cblogger.Error(errMSG)
		return irs.VMInfo{}, fmt.Errorf(errMSG)
        }

	for _, info := range infoList {
		if (*info).IId.NameId == iid.NameId {
			return CloneVMInfo(*info), nil
		}
	}

	errMSG := iid.NameId + " vm iid does not exist!!"
	cblogger.Error(errMSG)
	return irs.VMInfo{}, fmt.Errorf(errMSG)
}



func diskAttach(mockName string, iid irs.IID, diskIID irs.IID) (bool, error) {
        cblogger := cblog.GetLogger("CB-SPIDER")
        cblogger.Info("Mock Driver: called diskAttach()!")


vmMapLock.RLock()
defer vmMapLock.RUnlock()

        infoList, ok := vmInfoMap[mockName]
        if !ok {
                errMSG := iid.NameId + " vm iid does not exist!!"
                cblogger.Error(errMSG)
                return false, fmt.Errorf(errMSG)
        }

        for _, info := range infoList {
                if (*info).IId.SystemId == iid.SystemId {
			info.DataDiskIIDs = append(info.DataDiskIIDs, diskIID)
                        return true, nil
                }
        }

        errMSG := iid.NameId + " vm iid does not exist!!"
        cblogger.Error(errMSG)
        return false, fmt.Errorf(errMSG)
}

func diskDetach(mockName string, iid irs.IID, diskIID irs.IID) (bool, error) {
        cblogger := cblog.GetLogger("CB-SPIDER")
        cblogger.Info("Mock Driver: called diskDetach()!")


vmMapLock.RLock()
defer vmMapLock.RUnlock()

        infoList, ok := vmInfoMap[mockName]
        if !ok {
                errMSG := iid.NameId + " vm iid does not exist!!"
                cblogger.Error(errMSG)
                return false, fmt.Errorf(errMSG)
        }

        for _, info := range infoList {
                if (*info).IId.NameId == iid.NameId {
			for idx, oneIID := range info.DataDiskIIDs { 
				if oneIID.SystemId == diskIID.SystemId {
					info.DataDiskIIDs = append(info.DataDiskIIDs[:idx], info.DataDiskIIDs[idx+1:]...)
					return true, nil
				}
			}
                }
        }

        errMSG := iid.NameId + " vm iid does not exist!!"
        cblogger.Error(errMSG)
        return false, fmt.Errorf(errMSG)
}
