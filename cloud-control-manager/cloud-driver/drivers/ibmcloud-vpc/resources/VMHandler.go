package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/IBM/go-sdk-core/v5/core"
	vpcv0230 "github.com/IBM/vpc-go-sdk/0.23.0/vpcv1"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	cdcom "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/common"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"io/ioutil"
	"math/rand"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type IbmVMHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	VpcService     *vpcv1.VpcV1
	VpcService0230 *vpcv0230.VpcV1
	Ctx            context.Context
}

func (vmHandler *IbmVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (irs.VMInfo, error) {
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, vmReqInfo.IId.NameId, "StartVM()")
	start := call.Start()

	// 1.Check VMReqInfo
	err := checkVMReqInfo(vmReqInfo)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}

	// 1-1 Exist Check
	exist, err := existInstance(vmReqInfo.IId, vmHandler.VpcService, vmHandler.Ctx)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	} else if exist {
		createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = The VM name %s already exists", vmReqInfo.IId.NameId))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	// 1-2. Setup Req Resource IID
	var image vpcv1.Image
	var myImage irs.MyImageInfo
	var isWindows bool
	if vmReqInfo.ImageType == irs.MyImage {
		myImageHandler := IbmMyImageHandler{
			CredentialInfo: vmHandler.CredentialInfo,
			Region:         vmHandler.Region,
			VpcService:     vmHandler.VpcService,
			Ctx:            vmHandler.Ctx,
		}
		var getMyImageErr error
		myImage, getMyImageErr = myImageHandler.GetMyImage(vmReqInfo.ImageIID)
		if getMyImageErr != nil {
			createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", getMyImageErr.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
		if myImage.Status != irs.MyImageAvailable {
			createErr := errors.New("Failed to Create VM. err = Source Image status is not Available")
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
		rawSnapshot, _, getRawSnapshotErr := myImageHandler.VpcService.GetSnapshotWithContext(myImageHandler.Ctx, &vpcv1.GetSnapshotOptions{ID: &myImage.IId.SystemId})
		if getRawSnapshotErr != nil {
			createErr := errors.New("Failed to Create VM. err = Cannot get Snapshot Detail of Source MyImage")
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}

		isWindows = strings.Contains(strings.ToLower(*rawSnapshot.OperatingSystem.Name), "windows")
	} else {
		var getImageErr error
		image, getImageErr = getRawImage(vmReqInfo.ImageIID, vmHandler.VpcService, vmHandler.Ctx)
		if getImageErr != nil {
			createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", getImageErr.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}

		isWindows = strings.Contains(strings.ToLower(*image.OperatingSystem.Name), "windows")
	}

	vpc, err := getRawVPC(vmReqInfo.VpcIID, vmHandler.VpcService, vmHandler.Ctx)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	vpcSubnet, err := getVPCRawSubnet(vpc, vmReqInfo.SubnetIID, vmHandler.VpcService, vmHandler.Ctx)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	key, err := getRawKey(vmReqInfo.KeyPairIID, vmHandler.VpcService, vmHandler.Ctx)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	spec, err := getRawSpec(vmReqInfo.VMSpecName, vmHandler.VpcService, vmHandler.Ctx)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	var securityGroups []vpcv1.SecurityGroup
	if vmReqInfo.SecurityGroupIIDs != nil {
		for _, SecurityGroupIID := range vmReqInfo.SecurityGroupIIDs {
			err := checkSecurityGroupIID(SecurityGroupIID)
			if err != nil {
				createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
				return irs.VMInfo{}, createErr
			}
			securityGroup, err := getRawSecurityGroup(SecurityGroupIID, vmHandler.VpcService, vmHandler.Ctx)
			if err != nil {
				createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
				return irs.VMInfo{}, createErr
			}
			securityGroups = append(securityGroups, securityGroup)
		}
	}

	// 1-3. cloud-init data set
	var userData string
	if isWindows {
		userId := vmReqInfo.VMUserId
		if userId == "" {
			userId = "Administrator"
		}

		pwValidErr := cdcom.ValidateWindowsPassword(vmReqInfo.VMUserPasswd)
		if pwValidErr != nil {
			return irs.VMInfo{}, pwValidErr
		}

		userData = fmt.Sprintf("#ps1_sysnative\nnet user \"%s\" \"%s\"", userId, vmReqInfo.VMUserPasswd)
	} else {
		rootPath := os.Getenv("CBSPIDER_ROOT")
		fileDataCloudInit, err := ioutil.ReadFile(rootPath + CBCloudInitFilePath)
		if err != nil {
			createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
		userData = string(fileDataCloudInit)
		userData = strings.ReplaceAll(userData, "{{username}}", CBDefaultVmUserName)
	}

	// 2.Create VM
	// TODO : UserData cloudInit
	createInstanceOptions := &vpcv0230.CreateInstanceOptions{}

	var existingDataVolumeAttachments []vpcv0230.VolumeAttachmentPrototypeInstanceContext
	for _, dataVolumeIID := range vmReqInfo.DataDiskIIDs {
		rawDisk, getRawDiskErr := getRawDisk(vmHandler.VpcService, vmHandler.Ctx, dataVolumeIID)
		if getRawDiskErr == nil {
			existingDataVolumeAttachments = append(existingDataVolumeAttachments, vpcv0230.VolumeAttachmentPrototypeInstanceContext{
				Volume: &vpcv0230.VolumeAttachmentVolumePrototypeInstanceContextVolumeIdentity{ID: rawDisk.ID},
			})
		}
	}

	if vmReqInfo.ImageType == irs.MyImage {
		snapshotList, _, listSnapshotErr := vmHandler.VpcService.ListSnapshotsWithContext(vmHandler.Ctx, &vpcv1.ListSnapshotsOptions{})
		if listSnapshotErr != nil {
			return irs.VMInfo{}, errors.New(fmt.Sprintf("Failed to Get MyImage. err = %s", listSnapshotErr.Error()))
		}

		var associatedSnapshots []vpcv1.Snapshot
		for _, snapshot := range snapshotList.Snapshots {
			if strings.Split(*snapshot.Name, DEV)[0] == myImage.IId.NameId {
				associatedSnapshots = append(associatedSnapshots, snapshot)
			}
		}

		sort.Slice(associatedSnapshots, func(i, j int) bool {
			return *associatedSnapshots[i].Name < *associatedSnapshots[j].Name
		})

		var dataVolumeAttachments []vpcv0230.VolumeAttachmentPrototypeInstanceContext
		var bootVolumeAttachment vpcv0230.VolumeAttachmentPrototypeInstanceBySourceSnapshotContext
		for _, snapshot := range associatedSnapshots {
			sourceVolume, _, getSourceVolumeErr := vmHandler.VpcService0230.GetVolumeWithContext(vmHandler.Ctx, &vpcv0230.GetVolumeOptions{ID: snapshot.SourceVolume.ID})
			if getSourceVolumeErr != nil {
				return irs.VMInfo{}, errors.New(fmt.Sprintf("Failed to Get Source Volume. err = %s", getSourceVolumeErr.Error()))
			}

			volumeName := fmt.Sprintf("%s%s%s", vmReqInfo.IId.NameId, DEV, strings.Split(*snapshot.Name, DEV)[1])
			if *snapshot.Bootable {
				bootVolumeAttachment = vpcv0230.VolumeAttachmentPrototypeInstanceBySourceSnapshotContext{
					Volume: &vpcv0230.VolumePrototypeInstanceBySourceSnapshotContext{
						Name:           core.StringPtr(volumeName),
						Profile:        &vpcv0230.VolumeProfileIdentityByName{Name: sourceVolume.Profile.Name},
						Capacity:       sourceVolume.Capacity,
						SourceSnapshot: &vpcv0230.SnapshotIdentityByID{ID: snapshot.ID},
					},
				}
			} else {
				model := vpcv0230.VolumeAttachmentPrototypeInstanceContext{
					Volume: &vpcv0230.VolumeAttachmentVolumePrototypeInstanceContextVolumePrototypeInstanceContextVolumePrototypeInstanceContextVolumeBySourceSnapshot{
						Name:           core.StringPtr(volumeName),
						Profile:        &vpcv0230.VolumeProfileIdentityByName{Name: sourceVolume.Profile.Name},
						Capacity:       sourceVolume.Capacity,
						SourceSnapshot: &vpcv0230.SnapshotIdentityByID{ID: snapshot.ID},
					},
				}

				dataVolumeAttachments = append(dataVolumeAttachments, model)
			}
		}

		dataVolumeAttachments = append(dataVolumeAttachments, existingDataVolumeAttachments...)

		createInstanceOptions.SetInstancePrototype(&vpcv0230.InstancePrototypeInstanceBySourceSnapshot{
			Name:                 &vmReqInfo.IId.NameId,
			BootVolumeAttachment: &bootVolumeAttachment,
			VolumeAttachments:    dataVolumeAttachments,
			Profile: &vpcv0230.InstanceProfileIdentity{
				Name: spec.Name,
			},
			Zone: &vpcv0230.ZoneIdentity{
				Name: &vmHandler.Region.Zone,
			},
			PrimaryNetworkInterface: &vpcv0230.NetworkInterfacePrototype{
				Subnet: &vpcv0230.SubnetIdentity{
					ID: vpcSubnet.ID,
				},
			},
			Keys: []vpcv0230.KeyIdentityIntf{
				&vpcv0230.KeyIdentity{
					ID: key.ID,
				},
			},
			VPC: &vpcv0230.VPCIdentity{
				ID: vpc.ID,
			},
			UserData: &userData,
		})
	} else {
		createInstanceOptions.SetInstancePrototype(&vpcv0230.InstancePrototype{
			Name: &vmReqInfo.IId.NameId,
			Image: &vpcv0230.ImageIdentity{
				ID: image.ID,
			},
			Profile: &vpcv0230.InstanceProfileIdentity{
				Name: spec.Name,
			},
			Zone: &vpcv0230.ZoneIdentity{
				Name: &vmHandler.Region.Zone,
			},
			PrimaryNetworkInterface: &vpcv0230.NetworkInterfacePrototype{
				Subnet: &vpcv0230.SubnetIdentity{
					ID: vpcSubnet.ID,
				},
			},
			Keys: []vpcv0230.KeyIdentityIntf{
				&vpcv0230.KeyIdentity{
					ID: key.ID,
				},
			},
			VPC: &vpcv0230.VPCIdentity{
				ID: vpc.ID,
			},
			UserData:          &userData,
			VolumeAttachments: existingDataVolumeAttachments,
		})
	}

	createInstance, _, err := vmHandler.VpcService0230.CreateInstanceWithContext(vmHandler.Ctx, createInstanceOptions)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}

	// 3.Attach SecurityGroup
	if securityGroups != nil && len(securityGroups) > 0 {
		for _, securityGroup := range securityGroups {
			options := &vpcv1.AddSecurityGroupNetworkInterfaceOptions{}
			options.SetSecurityGroupID(*securityGroup.ID)
			options.SetID(*createInstance.PrimaryNetworkInterface.ID)
			_, _, err = vmHandler.VpcService.AddSecurityGroupNetworkInterfaceWithContext(vmHandler.Ctx, options)
			if err != nil {
				//TODO DELETE
				deleteErr := deleteInstance(*createInstance.ID, vmHandler.VpcService, vmHandler.Ctx)
				if err != nil {
					if deleteErr != nil {
						newErrText := err.Error() + deleteErr.Error()
						err = errors.New(newErrText)
					}
					createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
					cblogger.Error(createErr.Error())
					LoggingError(hiscallInfo, createErr)
					return irs.VMInfo{}, createErr
				}
			}
		}
	}

	// 4.Attach FloatingIP

	// 4-1. Create FloatingIP
	rand.Seed(time.Now().UnixNano())
	floatingIPName := *createInstance.Zone.Name + "-floatingip-" + strconv.FormatInt(rand.Int63n(10000000), 10)
	floatingIPExist, err := vmHandler.checkFloatingIPName(floatingIPName)
	if err != nil || floatingIPExist {
		createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = Faild Generator FloatingIP Name"))
		deleteErr := deleteInstance(*createInstance.ID, vmHandler.VpcService, vmHandler.Ctx)
		if deleteErr != nil {
			createErr = errors.New(fmt.Sprintf("%s, %s ", createErr.Error(), deleteErr.Error()))
		}
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}
	createFloatingIPOptions := &vpcv1.CreateFloatingIPOptions{}
	createFloatingIPOptions.SetFloatingIPPrototype(&vpcv1.FloatingIPPrototype{
		Name: &floatingIPName,
		Zone: &vpcv1.ZoneIdentity{
			Name: createInstance.Zone.Name,
		},
	})

	floatingIP, _, err := vmHandler.VpcService.CreateFloatingIPWithContext(vmHandler.Ctx, createFloatingIPOptions)

	if err != nil {
		deleteErr := deleteInstance(*createInstance.ID, vmHandler.VpcService, vmHandler.Ctx)
		if err != nil {
			if deleteErr != nil {
				newErrText := err.Error() + deleteErr.Error()
				err = errors.New(newErrText)
			}
			createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
		createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VMInfo{}, createErr
	}

	//  4-2. Bind FloatingIP
	ipBindInfo := IBMIPBindReqInfo{
		vmID:               *createInstance.ID,
		floatingIPID:       *floatingIP.ID,
		NetworkInterfaceID: *createInstance.PrimaryNetworkInterface.ID,
	}

	_, err = floatingIPBind(ipBindInfo, vmHandler.VpcService, vmHandler.Ctx)

	if err != nil {
		deleteErr := deleteInstance(*createInstance.ID, vmHandler.VpcService, vmHandler.Ctx)
		if err != nil {
			if deleteErr != nil {
				newErrText := err.Error() + deleteErr.Error()
				err = errors.New(newErrText)
			}
			createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
	}
	createInstanceIId := irs.IID{
		NameId:   *createInstance.Name,
		SystemId: *createInstance.ID,
	}
	// TODO : runnigCheck
	curRetryCnt := 0
	maxRetryCnt := 120
	for {
		finalInstance, err := getRawInstance(createInstanceIId, vmHandler.VpcService, vmHandler.Ctx)
		if err != nil {
			removeFloatingIpsErr := removeFloatingIps(finalInstance, vmHandler.VpcService, vmHandler.Ctx)
			// 생성 완료후 running 기다리는중 에러.
			// 제거 로직을 위해 removeFloatingIp
			if removeFloatingIpsErr != nil {
				// 제거 로직을 위해 removeFloatingIp Error => instance에 대한 에러 + removeError + delete error
				newErrText := err.Error() + removeFloatingIpsErr.Error() + "and failed delete VM"
				err = errors.New(newErrText)
				createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
				return irs.VMInfo{}, createErr
			}
			// 제거 로직을 위해 deleteInstance
			deleteErr := deleteInstance(*createInstance.ID, vmHandler.VpcService, vmHandler.Ctx)
			if deleteErr != nil {
				newErrText := err.Error() + deleteErr.Error()
				err = errors.New(newErrText)
				createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
				return irs.VMInfo{}, createErr
			}
			err = errors.New("failed to create VM")
			createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
		if *finalInstance.Status == "running" {
			finalInstanceInfo, err := vmHandler.setVmInfo(finalInstance)
			if err != nil {
				removeFloatingIpsErr := removeFloatingIps(finalInstance, vmHandler.VpcService, vmHandler.Ctx)
				// 생성 완료후 running 기다리는중 에러.
				// 제거 로직을 위해 removeFloatingIp
				if removeFloatingIpsErr != nil {
					// 제거 로직을 위해 removeFloatingIp Error => instance에 대한 에러 + removeError + delete error
					newErrText := err.Error() + removeFloatingIpsErr.Error() + "and failed delete VM"
					err = errors.New(newErrText)
					createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
					cblogger.Error(createErr.Error())
					LoggingError(hiscallInfo, createErr)
					return irs.VMInfo{}, createErr
				}
				// 제거 로직을 위해 deleteInstance
				deleteErr := deleteInstance(*createInstance.ID, vmHandler.VpcService, vmHandler.Ctx)
				if deleteErr != nil {
					newErrText := err.Error() + deleteErr.Error()
					err = errors.New(newErrText)
					createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
					cblogger.Error(createErr.Error())
					LoggingError(hiscallInfo, createErr)
					return irs.VMInfo{}, createErr
				}
				err = errors.New("failed to create VM")
				createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
				return irs.VMInfo{}, createErr
			}
			LoggingInfo(hiscallInfo, start)
			if isWindows {
				finalInstanceInfo.VMUserPasswd = vmReqInfo.VMUserPasswd
			}
			return finalInstanceInfo, nil
		}
		curRetryCnt++
		time.Sleep(1 * time.Second)
		if curRetryCnt > maxRetryCnt {
			err = errors.New(fmt.Sprintf("failed to create VM, exceeded maximum retry count %d", maxRetryCnt))
			removeFloatingIpsErr := removeFloatingIps(finalInstance, vmHandler.VpcService, vmHandler.Ctx)
			// 생성 완료후 running 기다리는중 에러.
			// 제거 로직을 위해 removeFloatingIp
			if removeFloatingIpsErr != nil {
				// 제거 로직을 위해 removeFloatingIp Error => instance에 대한 에러 + removeError + delete error
				newErrText := err.Error() + removeFloatingIpsErr.Error() + "and failed delete VM"
				err = errors.New(newErrText)
				createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
				return irs.VMInfo{}, createErr
			}
			// 제거 로직을 위해 deleteInstance
			deleteErr := deleteInstance(*createInstance.ID, vmHandler.VpcService, vmHandler.Ctx)
			if deleteErr != nil {
				newErrText := err.Error() + deleteErr.Error()
				err = errors.New(newErrText)
				createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
				return irs.VMInfo{}, createErr
			}
			err = errors.New("failed to create VM")
			createErr := errors.New(fmt.Sprintf("Failed to Create VM. err = %s", err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VMInfo{}, createErr
		}
	}
}
func (vmHandler *IbmVMHandler) SuspendVM(vmIID irs.IID) (irs.VMStatus, error) {
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, vmIID.NameId, "SuspendVM()")
	start := call.Start()
	err := checkVmIID(vmIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to SuspendVM. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.Failed, getErr
	}
	instance, err := getRawInstance(vmIID, vmHandler.VpcService, vmHandler.Ctx)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to SuspendVM. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.Failed, getErr
	}
	err = getSuspendVMCheck(*instance.Status)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to SuspendVM. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.Failed, getErr
	}
	instanceActionOptions := &vpcv1.CreateInstanceActionOptions{}
	instanceActionOptions.SetInstanceID(*instance.ID)
	instanceActionOptions.SetType("stop")
	_, _, err = vmHandler.VpcService.CreateInstanceActionWithContext(vmHandler.Ctx, instanceActionOptions)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to SuspendVM. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.Failed, getErr
	}
	LoggingInfo(hiscallInfo, start)
	return irs.Suspending, nil
}
func (vmHandler *IbmVMHandler) ResumeVM(vmIID irs.IID) (irs.VMStatus, error) {
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, vmIID.NameId, "ResumeVM()")
	start := call.Start()
	err := checkVmIID(vmIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to ResumeVM. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.Failed, getErr
	}
	instance, err := getRawInstance(vmIID, vmHandler.VpcService, vmHandler.Ctx)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to ResumeVM. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.Failed, getErr
	}
	err = getResumeVMCheck(*instance.Status)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to ResumeVM. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.Failed, getErr
	}
	instanceActionOptions := &vpcv1.CreateInstanceActionOptions{}
	instanceActionOptions.SetInstanceID(*instance.ID)
	instanceActionOptions.SetType("start")
	instanceActionOptions.SetForce(true)
	_, _, err = vmHandler.VpcService.CreateInstanceActionWithContext(vmHandler.Ctx, instanceActionOptions)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to ResumeVM. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.Failed, getErr
	}
	LoggingInfo(hiscallInfo, start)
	return irs.Resuming, nil
}
func (vmHandler *IbmVMHandler) RebootVM(vmIID irs.IID) (irs.VMStatus, error) {
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, vmIID.NameId, "RebootVM()")
	start := call.Start()
	err := checkVmIID(vmIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to RebootVM. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.Failed, getErr
	}
	instance, err := getRawInstance(vmIID, vmHandler.VpcService, vmHandler.Ctx)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to RebootVM. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.Failed, getErr
	}
	err = getRebootCheck(*instance.Status)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to RebootVM. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.Failed, getErr
	}
	instanceActionOptions := &vpcv1.CreateInstanceActionOptions{}
	instanceActionOptions.SetInstanceID(*instance.ID)
	instanceActionOptions.SetType("reboot")
	instanceActionOptions.SetForce(true)
	_, _, err = vmHandler.VpcService.CreateInstanceActionWithContext(vmHandler.Ctx, instanceActionOptions)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to RebootVM. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.Failed, getErr
	}
	LoggingInfo(hiscallInfo, start)
	return irs.Rebooting, nil
}
func (vmHandler *IbmVMHandler) TerminateVM(vmIID irs.IID) (irs.VMStatus, error) {
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, vmIID.NameId, "TerminateVM()")
	start := call.Start()
	err := checkVmIID(vmIID)
	if err != nil {
		TerminateErr := errors.New(fmt.Sprintf("Failed to Terminate VM err = %s", err.Error()))
		cblogger.Error(TerminateErr.Error())
		LoggingError(hiscallInfo, TerminateErr)
		return irs.Failed, TerminateErr
	}
	instance, err := getRawInstance(vmIID, vmHandler.VpcService, vmHandler.Ctx)
	if err != nil {
		TerminateErr := errors.New(fmt.Sprintf("Failed to Terminate VM err = %s", err.Error()))
		cblogger.Error(TerminateErr.Error())
		LoggingError(hiscallInfo, TerminateErr)
		return irs.Failed, TerminateErr
	}
	err = removeFloatingIps(instance, vmHandler.VpcService, vmHandler.Ctx)
	if err != nil {
		TerminateErr := errors.New(fmt.Sprintf("Failed to Terminate VM err = %s", err.Error()))
		cblogger.Error(TerminateErr.Error())
		LoggingError(hiscallInfo, TerminateErr)
		return irs.Failed, TerminateErr
	}
	err = deleteInstance(*instance.ID, vmHandler.VpcService, vmHandler.Ctx)
	if err != nil {
		TerminateErr := errors.New(fmt.Sprintf("Failed to Terminate VM err = %s", err.Error()))
		cblogger.Error(TerminateErr.Error())
		LoggingError(hiscallInfo, TerminateErr)
		return irs.Failed, TerminateErr
	}
	LoggingInfo(hiscallInfo, start)
	return irs.Terminating, nil
}

func (vmHandler *IbmVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, "VMStatus", "ListVMStatus()")
	start := call.Start()
	options := &vpcv1.ListInstancesOptions{}
	instances, _, err := vmHandler.VpcService.ListInstancesWithContext(vmHandler.Ctx, options)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get ListVMStatus. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}
	var vmStatusList []*irs.VMStatusInfo
	for {
		for _, instance := range instances.Instances {
			vmStatusString, _ := convertInstanceStatus(*instance.Status)
			VMStatusInfo := irs.VMStatusInfo{
				IId: irs.IID{
					NameId:   *instance.Name,
					SystemId: *instance.ID,
				},
				VmStatus: vmStatusString,
			}

			vmStatusList = append(vmStatusList, &VMStatusInfo)
		}
		nextstr, _ := getVMNextHref(instances.Next)
		if nextstr != "" {
			listVpcsOptions2 := &vpcv1.ListInstancesOptions{
				Start: core.StringPtr(nextstr),
			}
			instances, _, err = vmHandler.VpcService.ListInstancesWithContext(vmHandler.Ctx, listVpcsOptions2)
			if err != nil {
				getErr := errors.New(fmt.Sprintf("Failed to Get ListVMStatus. err = %s", err.Error()))
				cblogger.Error(getErr.Error())
				LoggingError(hiscallInfo, getErr)
				return nil, getErr
			}
		} else {
			break
		}
	}
	LoggingInfo(hiscallInfo, start)
	return vmStatusList, nil
}
func (vmHandler *IbmVMHandler) GetVMStatus(vmIID irs.IID) (irs.VMStatus, error) {
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, vmIID.NameId, "GetVMStatus()")
	start := call.Start()
	err := checkVmIID(vmIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VMStatus. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}
	instance, err := getRawInstance(vmIID, vmHandler.VpcService, vmHandler.Ctx)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VMStatus. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return "", getErr
	}
	LoggingInfo(hiscallInfo, start)
	return convertInstanceStatus(*instance.Status)
}

func (vmHandler *IbmVMHandler) ListVM() ([]*irs.VMInfo, error) {
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, "VM", "ListVM()")
	start := call.Start()
	options := &vpcv1.ListInstancesOptions{}
	instances, _, err := vmHandler.VpcService.ListInstancesWithContext(vmHandler.Ctx, options)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VMList. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	var vmInstanceList []vpcv1.Instance

	for {
		vmInstanceList = append(vmInstanceList, instances.Instances...)
		nextstr, _ := getVMNextHref(instances.Next)
		if nextstr != "" {
			listVpcsOptions2 := &vpcv1.ListInstancesOptions{
				Start: core.StringPtr(nextstr),
			}
			instances, _, err = vmHandler.VpcService.ListInstancesWithContext(vmHandler.Ctx, listVpcsOptions2)
			if err != nil {
				getErr := errors.New(fmt.Sprintf("Failed to Get VMList. err = %s", err.Error()))
				cblogger.Error(getErr.Error())
				LoggingError(hiscallInfo, getErr)
				return nil, getErr
				//break
			}
		} else {
			break
		}
	}

	vmList, err := vmHandler.setVMList(vmInstanceList)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VMList. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}
	LoggingInfo(hiscallInfo, start)
	return vmList, nil
}

func (vmHandler *IbmVMHandler) GetVM(vmIID irs.IID) (irs.VMInfo, error) {
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, vmIID.NameId, "GetVM()")
	start := call.Start()
	err := checkVmIID(vmIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VM. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.VMInfo{}, getErr
	}
	instance, err := getRawInstance(vmIID, vmHandler.VpcService, vmHandler.Ctx)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VM. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.VMInfo{}, getErr
	}

	vmInfo, err := vmHandler.setVmInfo(instance)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VM. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.VMInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)
	return vmInfo, nil
}

type IBMIPBindReqInfo struct {
	vmID               string
	floatingIPID       string
	NetworkInterfaceID string
}

func floatingIPBind(IPBindReqInfo IBMIPBindReqInfo, vpcService *vpcv1.VpcV1, ctx context.Context) (vpcv1.FloatingIP, error) {
	if IPBindReqInfo.vmID == "" || IPBindReqInfo.floatingIPID == "" || IPBindReqInfo.NetworkInterfaceID == "" {
		return vpcv1.FloatingIP{}, errors.New("invalid IDs")
	}
	addInstanceNetworkInterfaceFloatingIPOptions := &vpcv1.AddInstanceNetworkInterfaceFloatingIPOptions{}
	addInstanceNetworkInterfaceFloatingIPOptions.SetID(IPBindReqInfo.floatingIPID)
	addInstanceNetworkInterfaceFloatingIPOptions.SetInstanceID(IPBindReqInfo.vmID)
	addInstanceNetworkInterfaceFloatingIPOptions.SetNetworkInterfaceID(IPBindReqInfo.NetworkInterfaceID)
	floatingIP, _, err := vpcService.AddInstanceNetworkInterfaceFloatingIPWithContext(ctx, addInstanceNetworkInterfaceFloatingIPOptions)
	if err != nil {
		return vpcv1.FloatingIP{}, err
	}
	return *floatingIP, nil
}

func floatingIPUnBind(IPBindReqInfo IBMIPBindReqInfo, vpcService *vpcv1.VpcV1, ctx context.Context) (bool, error) {
	if IPBindReqInfo.vmID == "" || IPBindReqInfo.floatingIPID == "" || IPBindReqInfo.NetworkInterfaceID == "" {
		return false, errors.New("invalid IDs")
	}
	removeInstanceNetworkInterfaceFloatingIPOptions := &vpcv1.RemoveInstanceNetworkInterfaceFloatingIPOptions{}
	removeInstanceNetworkInterfaceFloatingIPOptions.SetID(IPBindReqInfo.floatingIPID)
	removeInstanceNetworkInterfaceFloatingIPOptions.SetInstanceID(IPBindReqInfo.vmID)
	removeInstanceNetworkInterfaceFloatingIPOptions.SetNetworkInterfaceID(IPBindReqInfo.NetworkInterfaceID)
	_, err := vpcService.RemoveInstanceNetworkInterfaceFloatingIPWithContext(ctx, removeInstanceNetworkInterfaceFloatingIPOptions)
	if err != nil {
		return false, err
	}
	deleteFloatingIPOptions := vpcService.NewDeleteFloatingIPOptions(IPBindReqInfo.floatingIPID)
	_, err = vpcService.DeleteFloatingIPWithContext(ctx, deleteFloatingIPOptions)
	if err != nil {
		return false, err
	}
	return true, nil
}

func getFloatIPsNextHref(next *vpcv1.FloatingIPCollectionNext) (string, error) {
	if next != nil {
		href := *next.Href
		u, err := url.Parse(href)
		if err != nil {
			return "", err
		}
		paramMap, _ := url.ParseQuery(u.RawQuery)
		if paramMap != nil {
			safe := paramMap["start"]
			if safe != nil && len(safe) > 0 {
				return safe[0], nil
			}
		}
	}
	return "", errors.New("NOT NEXT")
}

func getVMNextHref(next *vpcv1.InstanceCollectionNext) (string, error) {
	if next != nil {
		href := *next.Href
		u, err := url.Parse(href)
		if err != nil {
			return "", err
		}
		paramMap, _ := url.ParseQuery(u.RawQuery)
		if paramMap != nil {
			safe := paramMap["start"]
			if safe != nil && len(safe) > 0 {
				return safe[0], nil
			}
		}
	}
	return "", errors.New("NOT NEXT")
}
func getVolumeNextHref(next *vpcv1.VolumeCollectionNext) (string, error) {
	if next != nil {
		href := *next.Href
		u, err := url.Parse(href)
		if err != nil {
			return "", err
		}
		paramMap, _ := url.ParseQuery(u.RawQuery)
		if paramMap != nil {
			safe := paramMap["start"]
			if safe != nil && len(safe) > 0 {
				return safe[0], nil
			}
		}
	}
	return "", errors.New("NOT NEXT")
}
func checkVmIID(vmIID irs.IID) error {
	if vmIID.SystemId == "" && vmIID.NameId == "" {
		return errors.New("invalid IID")
	}
	return nil
}

func checkVMReqInfo(vmReqInfo irs.VMReqInfo) error {
	err := notSupportRootDiskCustom(vmReqInfo)
	if err != nil {
		return err
	}
	if vmReqInfo.IId.NameId == "" {
		return errors.New("invalid VM IID")
	}
	if vmReqInfo.ImageIID.NameId == "" && vmReqInfo.ImageIID.SystemId == "" {
		return errors.New("invalid VM ImageIID")
	}
	if vmReqInfo.VpcIID.NameId == "" && vmReqInfo.VpcIID.SystemId == "" {
		return errors.New("invalid VM VpcIID")
	}
	if vmReqInfo.SubnetIID.NameId == "" && vmReqInfo.SubnetIID.SystemId == "" {
		return errors.New("invalid VM SubnetIID")
	}
	if vmReqInfo.KeyPairIID.NameId == "" && vmReqInfo.KeyPairIID.SystemId == "" {
		return errors.New("invalid VM KeyPairIID")
	}
	if vmReqInfo.VMSpecName == "" {
		return errors.New("invalid VM VMSpecName")
	}
	return nil
}
func existInstance(vmIID irs.IID, vpcService *vpcv1.VpcV1, ctx context.Context) (bool, error) {
	options := &vpcv1.ListInstancesOptions{}
	instances, _, err := vpcService.ListInstancesWithContext(ctx, options)
	if err != nil {
		return false, err
	}
	for {
		for _, instance := range instances.Instances {
			if *instance.Name == vmIID.NameId {
				return true, nil
			}
		}
		nextstr, _ := getVMNextHref(instances.Next)
		if nextstr != "" {
			listInstanceOptionsNext := &vpcv1.ListInstancesOptions{
				Start: core.StringPtr(nextstr),
			}
			instances, _, err = vpcService.ListInstancesWithContext(ctx, listInstanceOptionsNext)
			if err != nil {
				return false, err
			}
		} else {
			break
		}
	}
	return false, nil
}

func getRawVolume(volumeIId irs.IID, vpcService *vpcv1.VpcV1, ctx context.Context) (vpcv1.Volume, error) {
	if volumeIId.SystemId == "" {
		options := &vpcv1.ListVolumesOptions{}
		volumes, _, err := vpcService.ListVolumesWithContext(ctx, options)
		if err != nil {
			return vpcv1.Volume{}, err
		}
		for {
			for _, volume := range volumes.Volumes {
				if *volume.Name == volumeIId.NameId {
					return volume, nil
				}
			}
			nextstr, _ := getVolumeNextHref(volumes.Next)
			if nextstr != "" {
				listVolumeOptionsNext := &vpcv1.ListVolumesOptions{
					Start: core.StringPtr(nextstr),
				}
				volumes, _, err = vpcService.ListVolumesWithContext(ctx, listVolumeOptionsNext)
				if err != nil {
					return vpcv1.Volume{}, err
				}
			} else {
				break
			}
		}
		err = errors.New(fmt.Sprintf("not found Volume %s", volumeIId.NameId))
		return vpcv1.Volume{}, err
	} else {
		options := &vpcv1.GetVolumeOptions{}
		options.SetID(volumeIId.SystemId)
		volume, _, err := vpcService.GetVolumeWithContext(ctx, options)
		if err != nil {
			return vpcv1.Volume{}, err
		}
		return *volume, err
	}
}

func getRawInstance(vmIID irs.IID, vpcService *vpcv1.VpcV1, ctx context.Context) (vpcv1.Instance, error) {
	if vmIID.SystemId == "" {
		options := &vpcv1.ListInstancesOptions{}
		instances, _, err := vpcService.ListInstancesWithContext(ctx, options)
		if err != nil {
			return vpcv1.Instance{}, err
		}
		for {
			for _, instance := range instances.Instances {
				if *instance.Name == vmIID.NameId {
					return instance, nil
				}
			}
			nextstr, _ := getVMNextHref(instances.Next)
			if nextstr != "" {
				listInstanceOptionsNext := &vpcv1.ListInstancesOptions{
					Start: core.StringPtr(nextstr),
				}
				instances, _, err = vpcService.ListInstancesWithContext(ctx, listInstanceOptionsNext)
				if err != nil {
					// LoggingError(hiscallInfo, err)
					return vpcv1.Instance{}, err
					//break
				}
			} else {
				break
			}
		}
		err = errors.New(fmt.Sprintf("not found VM %s", vmIID.NameId))
		return vpcv1.Instance{}, err
	} else {
		instanceOptions := &vpcv1.GetInstanceOptions{}
		instanceOptions.SetID(vmIID.SystemId)
		instance, _, err := vpcService.GetInstanceWithContext(ctx, instanceOptions)
		if err != nil {
			return vpcv1.Instance{}, err
		}
		return *instance, nil
	}
}
func getSuspendVMCheck(status string) error {
	switch status {
	case "running":
		return nil
	case "pausing", "pending", "stopping", "resuming", "restarting", "failed", "stopped", "paused", "starting":
		status, _ := convertInstanceStatus(status)
		return errors.New(fmt.Sprintf("can't ReBoot VM when your VM Status is %s", status))
	case "deleting":
		return errors.New("can't ReBoot VM when your VM Status is Terminating")
	//case "starting":
	//	return errors.New("can't ReBoot VM when your VM Status is Creating ")
	case "":
		return errors.New("can't ReBoot VM when your VM Status is NotExist")
	default:
		return errors.New("UnKnown STATUS")
	}
}

func getResumeVMCheck(status string) error {
	switch status {
	case "stopped", "paused":
		return nil
	case "pausing", "pending", "stopping", "running", "resuming", "restarting", "failed", "starting":
		status, _ := convertInstanceStatus(status)
		return errors.New(fmt.Sprintf("can't ReBoot VM when your VM Status is %s", status))
	case "deleting":
		return errors.New("can't ReBoot VM when your VM Status is Terminating")
	//case "starting":
	//	return errors.New("can't ReBoot VM when your VM Status is Creating ")
	case "":
		return errors.New("can't ReBoot VM when your VM Status is NotExist")
	default:
		return errors.New("UnKnown STATUS")
	}
}
func getRebootCheck(status string) error {
	switch status {
	case "pausing", "pending", "stopping", "running", "resuming", "restarting", "failed", "stopped", "paused", "starting":
		return nil
	case "deleting":
		return errors.New("can't ReBoot VM when your VM Status is Terminating")
	//case "starting":
	//	return errors.New("can't ReBoot VM when your VM Status is Creating")
	case "":
		return errors.New("can't ReBoot VM when your VM Status is NotExist")
	default:
		return errors.New("UnKnown STATUS")
	}
}

func convertInstanceStatus(status string) (irs.VMStatus, error) {
	switch status {
	case "pausing", "pending", "stopping":
		return irs.Suspending, nil
	case "stopped", "paused":
		return irs.Suspended, nil
	case "failed":
		return irs.Failed, nil
	case "restarting":
		return irs.Rebooting, nil
	case "resuming":
		return irs.Resuming, nil
	case "deleting":
		return irs.Terminating, nil
	case "running":
		return irs.Running, nil
	// TODO: starting and Creating 구분 못함.
	case "starting":
		return irs.Resuming, nil
	case "":
		return irs.NotExist, nil
	default:
		return "", errors.New("UnKnown STATUS")
	}
}

func deleteInstance(instanceId string, vpcService *vpcv1.VpcV1, ctx context.Context) error {
	deleteInstanceOptions := &vpcv1.DeleteInstanceOptions{}
	deleteInstanceOptions.SetID(instanceId)
	_, err := vpcService.DeleteInstanceWithContext(ctx, deleteInstanceOptions)
	return err
}

func removeFloatingIps(instance vpcv1.Instance, vpcService *vpcv1.VpcV1, ctx context.Context) error {
	instanceNetworkInterfaceOptions := &vpcv1.GetInstanceNetworkInterfaceOptions{}
	instanceNetworkInterfaceOptions.SetID(*instance.PrimaryNetworkInterface.ID)
	instanceNetworkInterfaceOptions.SetInstanceID(*instance.ID)
	networkInterface, _, err := vpcService.GetInstanceNetworkInterfaceWithContext(ctx, instanceNetworkInterfaceOptions)
	if err != nil {
		return err
	}
	if networkInterface.FloatingIps != nil {
		for _, floatingIp := range networkInterface.FloatingIps {
			ipBindInfo := IBMIPBindReqInfo{
				vmID:               *instance.ID,
				floatingIPID:       *floatingIp.ID,
				NetworkInterfaceID: *instance.PrimaryNetworkInterface.ID,
			}
			_, err := floatingIPUnBind(ipBindInfo, vpcService, ctx)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (vmHandler *IbmVMHandler) getKeyIId(instance vpcv1.Instance) irs.IID {
	// KeyGet
	var instanceId = ""
	if instance.ID != nil {
		instanceId = *instance.ID
	} else {
		return irs.IID{}
	}
	instanceInitializationOptions := &vpcv1.GetInstanceInitializationOptions{}
	instanceInitializationOptions.SetID(instanceId)
	initData, _, err := vmHandler.VpcService.GetInstanceInitializationWithContext(vmHandler.Ctx, instanceInitializationOptions)
	if err == nil && initData.Keys != nil && len(initData.Keys) > 0 {
		jsonInitDataBytes, err := json.Marshal(initData.Keys[0])
		if err == nil {
			var keyRef vpcv1.KeyReferenceInstanceInitializationContextKeyReference
			err = json.Unmarshal(jsonInitDataBytes, &keyRef)
			if err == nil && keyRef.ID != nil && keyRef.Name != nil {
				return irs.IID{
					NameId:   *keyRef.Name,
					SystemId: *keyRef.ID,
				}
			}
		}
	}
	return irs.IID{}
}

type vmNetworkInfo struct {
	NetworkInterface  string
	PublicIP          string
	SSHAccessPoint    string
	SecurityGroupIIds []irs.IID
}

func (vmHandler *IbmVMHandler) getBootVolumeInfo(instance vpcv1.Instance) (rootDiskSize string) {
	var bootVolumeId = ""
	if instance.BootVolumeAttachment != nil && instance.BootVolumeAttachment.Volume.ID != nil {
		bootVolumeId = *instance.BootVolumeAttachment.Volume.ID
	} else {
		return ""
	}
	volumeIId := irs.IID{SystemId: bootVolumeId}
	rawVolume, err := getRawVolume(volumeIId, vmHandler.VpcService, vmHandler.Ctx)

	if err == nil {
		return strconv.Itoa(int(*rawVolume.Capacity))
	}
	return ""
}

// networkDone <- vmHandler.getNetworkInfo(*instance.ID, *instance.PrimaryNetworkInterface.ID, *instance.VPC.ID)
func (vmHandler *IbmVMHandler) getNetworkInfo(instance vpcv1.Instance) vmNetworkInfo {
	// Network Get
	var instanceId = ""
	var instancePrimaryNIId = ""
	var vpcId = ""
	if instance.ID != nil {
		instanceId = *instance.ID
	}
	if instance.PrimaryNetworkInterface != nil && instance.PrimaryNetworkInterface.ID != nil {
		instancePrimaryNIId = *instance.PrimaryNetworkInterface.ID
	}
	if instance.VPC != nil && instance.VPC.ID != nil {
		vpcId = *instance.VPC.ID
	}
	if instanceId == "" || instancePrimaryNIId == "" {
		return vmNetworkInfo{}
	}
	info := vmNetworkInfo{}
	instanceNetworkInterfaceOptions := &vpcv1.GetInstanceNetworkInterfaceOptions{}
	instanceNetworkInterfaceOptions.SetID(instancePrimaryNIId)
	instanceNetworkInterfaceOptions.SetInstanceID(instanceId)
	networkInterface, _, err := vmHandler.VpcService.GetInstanceNetworkInterfaceWithContext(vmHandler.Ctx, instanceNetworkInterfaceOptions)
	if err == nil {
		// SET IP
		info.NetworkInterface = *networkInterface.Name
		if networkInterface.FloatingIps != nil && len(networkInterface.FloatingIps) > 0 {
			info.PublicIP = *networkInterface.FloatingIps[0].Address
			info.SSHAccessPoint = info.PublicIP + ":22"
		}
		if vpcId == "" {
			info.SecurityGroupIIds = []irs.IID{}
			return info
		}
		// SET SG
		getVpcOptions := &vpcv1.GetVPCOptions{}
		getVpcOptions.SetID(vpcId)
		vpc, _, _ := vmHandler.VpcService.GetVPCWithContext(vmHandler.Ctx, getVpcOptions)
		var sgIIds []irs.IID
		if vpc != nil && vpc.DefaultSecurityGroup != nil {
			defaultSGId := *vpc.DefaultSecurityGroup.ID
			vmSecurityGroups := networkInterface.SecurityGroups
			for _, seg := range vmSecurityGroups {
				if defaultSGId != *seg.ID {
					sgIIds = append(sgIIds, irs.IID{NameId: *seg.Name, SystemId: *seg.ID})
				}
			}
		}
		info.SecurityGroupIIds = sgIIds
	}
	return info
}

func (vmHandler *IbmVMHandler) setVmInfo(instance vpcv1.Instance) (irs.VMInfo, error) {
	var dataDiskIIDs []irs.IID
	for _, rawDataDisk := range instance.VolumeAttachments {
		if *rawDataDisk.Volume.ID != *instance.BootVolumeAttachment.Volume.ID {
			dataDiskIIDs = append(dataDiskIIDs, irs.IID{
				NameId:   *rawDataDisk.Volume.Name,
				SystemId: *rawDataDisk.Volume.ID,
			})
		}
	}
	vmInfo := irs.VMInfo{
		IId: irs.IID{
			NameId:   *instance.Name,
			SystemId: *instance.ID,
		},
		StartTime: time.Time(*instance.CreatedAt).Local(),
		Region: irs.RegionInfo{
			Region: vmHandler.Region.Region,
			Zone:   *instance.Zone.Name,
		},
		VMSpecName: *instance.Profile.Name,
		VpcIID: irs.IID{
			NameId:   *instance.VPC.Name,
			SystemId: *instance.VPC.ID,
		},
		SubnetIID: irs.IID{
			NameId:   *instance.PrimaryNetworkInterface.Subnet.Name,
			SystemId: *instance.PrimaryNetworkInterface.Subnet.ID,
		},
		PrivateIP:      *instance.PrimaryNetworkInterface.PrimaryIpv4Address,
		VMUserId:       CBDefaultVmUserName,
		RootDeviceName: "Not visible in IBMCloud-VPC",
		VMBlockDisk:    "Not visible in IBMCloud-VPC",
		DataDiskIIDs:   dataDiskIIDs,
	}
	chanCount := 0
	// KeyGet
	keyDone := make(chan irs.IID)
	chanCount++
	go func() {
		keyDone <- vmHandler.getKeyIId(instance)
	}()

	networkDone := make(chan vmNetworkInfo)
	chanCount++
	go func() {
		networkDone <- vmHandler.getNetworkInfo(instance)
	}()

	volumeDone := make(chan string)
	chanCount++
	go func() {
		volumeDone <- vmHandler.getBootVolumeInfo(instance)
	}()

	for i := 0; i < chanCount; i++ {
		select {
		case keyIID := <-keyDone:
			vmInfo.KeyPairIId = keyIID
		case netInfo := <-networkDone:
			vmInfo.NetworkInterface = netInfo.NetworkInterface
			vmInfo.PublicIP = netInfo.PublicIP
			vmInfo.SSHAccessPoint = netInfo.SSHAccessPoint
			vmInfo.SecurityGroupIIds = netInfo.SecurityGroupIIds
		case volumeRootDiskSize := <-volumeDone:
			vmInfo.RootDiskSize = volumeRootDiskSize
		}
	}

	vmInfo.RootDiskType = "general-purpose"

	imageId := instance.Image.ID
	imageHandler := IbmImageHandler{
		CredentialInfo: vmHandler.CredentialInfo,
		Region:         vmHandler.Region,
		VpcService:     vmHandler.VpcService,
		Ctx:            vmHandler.Ctx,
	}
	myImageHandler := IbmMyImageHandler{
		CredentialInfo: vmHandler.CredentialInfo,
		Region:         vmHandler.Region,
		VpcService:     vmHandler.VpcService,
		Ctx:            vmHandler.Ctx,
	}

	vmInfo.VMBootDisk = *instance.BootVolumeAttachment.Volume.ID

	sourceImage, getSourceImageErr := imageHandler.GetImage(irs.IID{SystemId: *imageId})
	if getSourceImageErr == nil {
		vmInfo.ImageIId = sourceImage.IId
		vmInfo.ImageType = irs.PublicImage

		image, getImageErr := getRawImage(irs.IID{SystemId: *imageId}, vmHandler.VpcService, vmHandler.Ctx)
		if getImageErr == nil {
			isWindows := strings.Contains(strings.ToLower(*image.OperatingSystem.Name), "windows")
			if isWindows {
				vmInfo.VMUserId = "Administrator"
			} else {
				vmInfo.VMUserId = "cb-user"
			}
		}

		return vmInfo, nil
	}

	sourceMyImage, getSourceMyImageErr := myImageHandler.GetMyImage(irs.IID{SystemId: *imageId})
	if getSourceMyImageErr == nil {
		vmInfo.ImageIId = sourceMyImage.IId
		vmInfo.ImageType = irs.MyImage

		rawSnapshot, _, getRawSnapshotErr := myImageHandler.VpcService.GetSnapshotWithContext(myImageHandler.Ctx, &vpcv1.GetSnapshotOptions{ID: imageId})
		if getRawSnapshotErr == nil {
			isWindows := strings.Contains(strings.ToLower(*rawSnapshot.OperatingSystem.Name), "windows")
			if isWindows {
				vmInfo.VMUserId = "Administrator"
			} else {
				vmInfo.VMUserId = "cb-user"
			}
		}

		return vmInfo, nil
	}

	return vmInfo, nil
}

func (vmHandler *IbmVMHandler) checkFloatingIPName(floatingIPName string) (exist bool, err error) {
	options := &vpcv1.ListFloatingIpsOptions{}
	floatingIPs, _, err := vmHandler.VpcService.ListFloatingIpsWithContext(vmHandler.Ctx, options)
	if err != nil {
		return false, err
	}
	for {
		for _, floatingIP := range floatingIPs.FloatingIps {
			if *floatingIP.Name == floatingIPName {
				return true, nil
			}
		}
		nextstr, _ := getFloatIPsNextHref(floatingIPs.Next)
		if nextstr != "" {
			listFloatingNext := &vpcv1.ListFloatingIpsOptions{
				Start: core.StringPtr(nextstr),
			}
			floatingIPs, _, err = vmHandler.VpcService.ListFloatingIpsWithContext(vmHandler.Ctx, listFloatingNext)
			if err != nil {
				return false, err
			}
		} else {
			break
		}
	}
	return false, nil
}

func notSupportRootDiskCustom(vmReqInfo irs.VMReqInfo) error {
	if vmReqInfo.RootDiskType != "" && strings.ToLower(vmReqInfo.RootDiskType) != "default" {
		return errors.New("IBM-VPC_CANNOT_CHANGE_ROOTDISKTYPE")
	}
	if vmReqInfo.RootDiskSize != "" && strings.ToLower(vmReqInfo.RootDiskSize) != "default" {
		return errors.New("IBM-VPC_CANNOT_CHANGE_ROOTDISKSIZE")
	}
	return nil
}

type vmInfoWithError struct {
	VMInfo irs.VMInfo
	err    error
}

func (vmHandler *IbmVMHandler) setVmInfoWithContext(ctx context.Context, instance vpcv1.Instance) (irs.VMInfo, error) {
	done := make(chan vmInfoWithError)

	go func() {
		vmInfo, err := vmHandler.setVmInfo(instance)
		done <- vmInfoWithError{
			VMInfo: vmInfo,
			err:    err,
		}
	}()
	select {
	case vmInfoWithErrorDone := <-done:
		return vmInfoWithErrorDone.VMInfo, vmInfoWithErrorDone.err
	case <-ctx.Done():
		return irs.VMInfo{}, nil
	}
}

func (vmHandler *IbmVMHandler) setVMList(instanceList []vpcv1.Instance) ([]*irs.VMInfo, error) {
	vmListCount := len(instanceList)
	vmList := make([]*irs.VMInfo, len(instanceList))

	if vmListCount == 0 {
		return vmList, nil
	}
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	var globalErr error

	for i, vmInstance := range instanceList {
		wg.Add(1)
		index := i
		dumpInstance := vmInstance
		go func() {
			defer wg.Done()
			vmInfo, err := vmHandler.setVmInfoWithContext(ctx, dumpInstance)
			if err != nil {
				cancel()
				if globalErr == nil {
					globalErr = err
				}
			}
			vmList[index] = &vmInfo
		}()
	}
	wg.Wait()

	if globalErr != nil {
		return nil, globalErr
	}

	return vmList, nil
}
