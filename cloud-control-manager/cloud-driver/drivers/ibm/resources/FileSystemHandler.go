package resources

import (
	"context"
	"errors"
	"fmt"
	"github.com/IBM/platform-services-go-sdk/globalsearchv2"
	"github.com/IBM/platform-services-go-sdk/globaltaggingv1"
	"strconv"
	"strings"
	"time"

	"github.com/IBM/vpc-go-sdk/vpcv1"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type IbmFileSystemHandler struct {
	Region         idrv.RegionInfo
	CredentialInfo idrv.CredentialInfo
	Ctx            context.Context
	VpcService     *vpcv1.VpcV1
	TaggingService *globaltaggingv1.GlobalTaggingV1
	SearchService  *globalsearchv2.GlobalSearchV2
}

const (
	IBMFileSystemSecurityGroupNamePrefix string = "cbfs-"
)

// generateSGNameForFileSystem generates a security group name for FileSystem
// IBM Cloud requires SG names to be:
// - Maximum 63 characters
// - Lowercase only
// - Only hyphens allowed as special characters
func generateSGNameForFileSystem(vpcID string) string {
	prefix := IBMFileSystemSecurityGroupNamePrefix
	maxLength := 63

	// Calculate available length for VPC ID
	availableLength := maxLength - len(prefix)

	// If vpcID fits within the limit, use it as is
	if len(vpcID) <= availableLength {
		return prefix + vpcID
	}

	// If vpcID is too long, truncate it
	truncatedID := vpcID[:availableLength]
	return prefix + truncatedID
}

func (filesystemHandler *IbmFileSystemHandler) GetMetaInfo() (irs.FileSystemMetaInfo, error) {
	metaInfo := irs.FileSystemMetaInfo{
		SupportsFileSystemType: map[irs.FileSystemType]bool{
			irs.FileSystemType("nfs"): true,
		},

		SupportsVPC: map[irs.RSType]bool{
			irs.RSType("VPC"): true,
		},

		SupportsNFSVersion: []string{"4.1"},

		SupportsCapacity: true,

		CapacityGBOptions: map[string]irs.CapacityGBRange{
			"Standard": {
				Min: 10,
				Max: 32000,
			},
		},

		PerformanceOptions: map[string][]string{
			"IOPS": {"100-1000"},
		},
	}

	return metaInfo, nil
}

func (filesystemHandler *IbmFileSystemHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(filesystemHandler.Region, call.FILESYSTEM, "FILESYSTEM", "ListIID()")
	start := call.Start()

	options := &vpcv1.ListSharesOptions{}
	res, _, err := filesystemHandler.VpcService.ListSharesWithContext(filesystemHandler.Ctx, options)
	if err != nil {
		return nil, err
	}

	var iidList []*irs.IID
	for _, share := range res.Shares {
		if share.Zone != nil && share.Zone.Name != nil && *share.Zone.Name == filesystemHandler.Region.Zone {
			iid := &irs.IID{
				NameId:   *share.Name,
				SystemId: *share.ID,
			}
			iidList = append(iidList, iid)
		}
	}

	LoggingInfo(hiscallInfo, start)
	return iidList, nil
}

func (filesystemHandler *IbmFileSystemHandler) findSubnetIDByName(name string) (string, error) {
	subnets, _, err := filesystemHandler.VpcService.ListSubnetsWithContext(filesystemHandler.Ctx, &vpcv1.ListSubnetsOptions{})
	if err != nil {
		return "", err
	}
	for _, subnet := range subnets.Subnets {
		if subnet.Name != nil && *subnet.Name == name {
			return *subnet.ID, nil
		}
	}
	return "", fmt.Errorf("subnet with name %s not found", name)
}

func (filesystemHandler *IbmFileSystemHandler) findSubnetsByVPC(vpcIID irs.IID) ([]irs.IID, error) {
	subnets, _, err := filesystemHandler.VpcService.ListSubnetsWithContext(filesystemHandler.Ctx, &vpcv1.ListSubnetsOptions{})
	if err != nil {
		return nil, err
	}

	var subnetList []irs.IID
	for _, subnet := range subnets.Subnets {
		if subnet.VPC != nil {
			vpcMatch := false
			if vpcIID.SystemId != "" && subnet.VPC.ID != nil && *subnet.VPC.ID == vpcIID.SystemId {
				vpcMatch = true
			} else if vpcIID.NameId != "" && subnet.VPC.Name != nil && *subnet.VPC.Name == vpcIID.NameId {
				vpcMatch = true
			}

			if vpcMatch && subnet.Zone != nil && subnet.Zone.Name != nil && *subnet.Zone.Name == filesystemHandler.Region.Zone {
				subnetIID := irs.IID{}
				if subnet.Name != nil {
					subnetIID.NameId = *subnet.Name
				}
				if subnet.ID != nil {
					subnetIID.SystemId = *subnet.ID
				}
				subnetList = append(subnetList, subnetIID)
			}
		}
	}
	return subnetList, nil
}
func (filesystemHandler *IbmFileSystemHandler) findOrCreateCbspiderSecurityGroup(vpcID string) (string, error) {
	listSgOptions := &vpcv1.ListSecurityGroupsOptions{
		VPCID: &vpcID,
	}
	securityGroups, _, err := filesystemHandler.VpcService.ListSecurityGroupsWithContext(filesystemHandler.Ctx, listSgOptions)
	if err != nil {
		return "", err
	}

	sgName := generateSGNameForFileSystem(vpcID)

	for _, sg := range securityGroups.SecurityGroups {
		if sg.Name != nil && *sg.Name == sgName {
			return *sg.ID, nil
		}
	}

	createSgOptions := &vpcv1.CreateSecurityGroupOptions{
		VPC: &vpcv1.VPCIdentity{
			ID: &vpcID,
		},
		Name: stringPtr(sgName),
	}

	newSg, _, err := filesystemHandler.VpcService.CreateSecurityGroupWithContext(filesystemHandler.Ctx, createSgOptions)
	if err != nil {
		return "", err
	}

	err = addTag(filesystemHandler.TaggingService, irs.KeyValue{
		Key:   IBMFileSystemSGTagKey,
		Value: vpcID,
	}, *newSg.CRN)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Attach Tag to Security err = %s", err.Error()))
		cblogger.Error(createErr.Error())
	}

	return *newSg.ID, nil
}

func (filesystemHandler *IbmFileSystemHandler) getSubnetCIDR(subnetID string) (string, error) {
	getSubnetOptions := &vpcv1.GetSubnetOptions{
		ID: &subnetID,
	}
	subnet, _, err := filesystemHandler.VpcService.GetSubnetWithContext(filesystemHandler.Ctx, getSubnetOptions)
	if err != nil {
		return "", err
	}
	if subnet.Ipv4CIDRBlock != nil {
		return *subnet.Ipv4CIDRBlock, nil
	}
	return "", fmt.Errorf("subnet CIDR not found")
}

func (filesystemHandler *IbmFileSystemHandler) findSubnetByCIDR(cidr string) (*irs.IID, error) {
	subnets, _, err := filesystemHandler.VpcService.ListSubnetsWithContext(filesystemHandler.Ctx, &vpcv1.ListSubnetsOptions{})
	if err != nil {
		return nil, err
	}

	for _, subnet := range subnets.Subnets {
		if subnet.Ipv4CIDRBlock != nil && *subnet.Ipv4CIDRBlock == cidr {
			subnetIID := &irs.IID{}
			if subnet.Name != nil {
				subnetIID.NameId = *subnet.Name
			}
			if subnet.ID != nil {
				subnetIID.SystemId = *subnet.ID
			}
			return subnetIID, nil
		}
	}
	return nil, fmt.Errorf("subnet with CIDR %s not found", cidr)
}

func (filesystemHandler *IbmFileSystemHandler) deleteCbspiderSecurityGroup(vpcID string) error {
	listSgOptions := &vpcv1.ListSecurityGroupsOptions{
		VPCID: &vpcID,
	}
	securityGroups, _, err := filesystemHandler.VpcService.ListSecurityGroupsWithContext(filesystemHandler.Ctx, listSgOptions)
	if err != nil {
		return err
	}

	sgName := generateSGNameForFileSystem(vpcID)

	var sgID *string
	for _, sg := range securityGroups.SecurityGroups {
		if sg.Name != nil && *sg.Name == sgName {
			sgID = sg.ID
			break
		}
	}

	if sgID == nil {
		// Security group not found - already deleted or never created
		return nil
	}

	// Retry logic for deleting security group
	// Total retry time: 200 retries * 3 seconds = 600 seconds (10 minutes)
	maxRetries := 200
	retryInterval := 3 * time.Second

	for i := 0; i < maxRetries; i++ {
		deleteOptions := &vpcv1.DeleteSecurityGroupOptions{
			ID: sgID,
		}
		_, err := filesystemHandler.VpcService.DeleteSecurityGroupWithContext(filesystemHandler.Ctx, deleteOptions)
		if err == nil {
			cblogger.Infof("Successfully deleted security group %s for FileSystem VPC %s", sgName, vpcID)
			return nil
		}

		// If it's the last retry, return the error
		if i == maxRetries-1 {
			return fmt.Errorf("failed to delete security group %s for FileSystem VPC %s after %d retries (%.0f minutes). err = %s",
				sgName, vpcID, maxRetries, retryInterval.Seconds()*float64(maxRetries)/60, err.Error())
		}

		// Wait before retrying
		if i%10 == 0 && i > 0 {
			cblogger.Infof("Retrying security group deletion (%d/%d) for %s - elapsed: %.1f minutes",
				i+1, maxRetries, sgName, retryInterval.Seconds()*float64(i)/60)
		}
		time.Sleep(retryInterval)
	}

	return nil
}
func (filesystemHandler *IbmFileSystemHandler) CreateFileSystem(reqInfo irs.FileSystemInfo) (irs.FileSystemInfo, error) {
	hiscallInfo := GetCallLogScheme(filesystemHandler.Region, call.FILESYSTEM, filesystemHandler.Region.Region, "CreateFileSystem()")
	start := call.Start()

	if reqInfo.IId.NameId == "" {
		err := fmt.Errorf("invalid request: NameId is required")
		LoggingError(hiscallInfo, err)
		return irs.FileSystemInfo{}, err
	}

	if !strings.EqualFold(reqInfo.NFSVersion, "4.1") {
		err := fmt.Errorf("unsupported NFS version: %q", reqInfo.NFSVersion)
		LoggingError(hiscallInfo, err)
		return irs.FileSystemInfo{}, err
	}

	shareProfile := &vpcv1.ShareProfileIdentity{
		Name: stringPtr("dp2"),
	}

	var iops = 100
	iopsStr, ok := reqInfo.PerformanceInfo["IOPS"]
	if ok {
		var err error
		iops, err = strconv.Atoi(iopsStr)
		if err != nil {
			err = fmt.Errorf("invalid IOPS: %s", reqInfo.PerformanceInfo["IOPS"])
			LoggingError(hiscallInfo, err)
			return irs.FileSystemInfo{}, err
		}
		if iops < 100 || iops > 1000 {
			err = fmt.Errorf("IOPS is out of range: %s", reqInfo.PerformanceInfo["IOPS"])
			LoggingError(hiscallInfo, err)
			return irs.FileSystemInfo{}, err
		}
	}

	zone := &vpcv1.ZoneIdentity{
		Name: &filesystemHandler.Region.Zone,
	}

	sharePrototype := &vpcv1.SharePrototype{
		Name:              &reqInfo.IId.NameId,
		Profile:           shareProfile,
		Iops:              int64Ptr(int64(iops)),
		Zone:              zone,
		AccessControlMode: stringPtr("security_group"),
	}

	capacity := reqInfo.CapacityGB
	if capacity <= 0 {
		capacity = 10
	}
	sharePrototype.Size = int64Ptr(capacity)

	subnetList := reqInfo.AccessSubnetList
	if len(subnetList) == 0 && (reqInfo.VpcIID.NameId != "" || reqInfo.VpcIID.SystemId != "") {
		foundSubnets, err := filesystemHandler.findSubnetsByVPC(reqInfo.VpcIID)
		if err != nil {
			LoggingError(hiscallInfo, err)
			return irs.FileSystemInfo{}, fmt.Errorf("failed to create file share: %v", err)
		}

		if len(foundSubnets) > 0 {
			subnetList = []irs.IID{foundSubnets[0]}
		} else {
			err = errors.New("subnets not found with the provided VPC ID")
			LoggingError(hiscallInfo, err)
			return irs.FileSystemInfo{}, fmt.Errorf("failed to create file share: %v", err)
		}
	}

	var vpcID string
	var securityGroupID string
	if len(subnetList) > 0 {
		subnetIID := subnetList[0]
		var subnetID string
		if subnetIID.SystemId != "" {
			subnetID = subnetIID.SystemId
		} else if subnetIID.NameId != "" {
			id, err := filesystemHandler.findSubnetIDByName(subnetIID.NameId)
			if err != nil {
				LoggingError(hiscallInfo, err)
				return irs.FileSystemInfo{}, fmt.Errorf("failed to find subnet: %v", err)
			}
			subnetID = id
		}

		getSubnetOptions := &vpcv1.GetSubnetOptions{
			ID: &subnetID,
		}
		subnet, _, err := filesystemHandler.VpcService.GetSubnetWithContext(filesystemHandler.Ctx, getSubnetOptions)
		if err != nil {
			LoggingError(hiscallInfo, err)
			return irs.FileSystemInfo{}, fmt.Errorf("failed to get subnet info: %v", err)
		}
		if subnet.VPC != nil && subnet.VPC.ID != nil {
			vpcID = *subnet.VPC.ID
		}

		if vpcID != "" {
			securityGroupID, err = filesystemHandler.findOrCreateCbspiderSecurityGroup(vpcID)
			if err != nil {
				LoggingError(hiscallInfo, err)
				return irs.FileSystemInfo{}, fmt.Errorf("failed to create security group: %v", err)
			} else {
				allSubnets, err := filesystemHandler.findSubnetsByVPC(irs.IID{SystemId: vpcID})
				if err == nil {
					for _, subnet := range allSubnets {
						cidr, err := filesystemHandler.getSubnetCIDR(subnet.SystemId)
						if err == nil {
							createRuleOptions := &vpcv1.CreateSecurityGroupRuleOptions{
								SecurityGroupID: &securityGroupID,
								SecurityGroupRulePrototype: &vpcv1.SecurityGroupRulePrototype{
									Direction: stringPtr("inbound"),
									Protocol:  stringPtr("tcp"),
									PortMin:   int64Ptr(2049),
									PortMax:   int64Ptr(2049),
									Remote: &vpcv1.SecurityGroupRuleRemotePrototype{
										CIDRBlock: &cidr,
									},
								},
							}
							_, _, err := filesystemHandler.VpcService.CreateSecurityGroupRuleWithContext(filesystemHandler.Ctx, createRuleOptions)
							if err != nil {
								LoggingError(hiscallInfo, err)
								return irs.FileSystemInfo{}, fmt.Errorf("failed to create security group's rule: %v", err)
							}
						}
					}
				}
			}
		}

		mountTarget := &vpcv1.ShareMountTargetPrototype{
			VirtualNetworkInterface: &vpcv1.ShareMountTargetVirtualNetworkInterfacePrototype{
				Subnet: &vpcv1.SubnetIdentity{
					ID: &subnetID,
				},
				SecurityGroups: []vpcv1.SecurityGroupIdentityIntf{
					&vpcv1.SecurityGroupIdentity{
						ID: &securityGroupID,
					},
				},
			},
		}
		sharePrototype.MountTargets = []vpcv1.ShareMountTargetPrototypeIntf{mountTarget}
	}

	createShareOptions := &vpcv1.CreateShareOptions{
		SharePrototype: sharePrototype,
	}

	share, _, err := filesystemHandler.VpcService.CreateShareWithContext(filesystemHandler.Ctx, createShareOptions)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.FileSystemInfo{}, fmt.Errorf("failed to create file share: %v", err)
	}

	time.Sleep(10 * time.Second)

	fileSystemInfo, err := filesystemHandler.GetFileSystem(irs.IID{
		NameId:   *share.Name,
		SystemId: *share.ID,
	})
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.FileSystemInfo{}, fmt.Errorf("created but get failed: %v", err)
	}

	LoggingInfo(hiscallInfo, start)
	return fileSystemInfo, nil
}

func (filesystemHandler *IbmFileSystemHandler) ListFileSystem() ([]*irs.FileSystemInfo, error) {
	hiscallInfo := GetCallLogScheme(filesystemHandler.Region, call.FILESYSTEM, filesystemHandler.Region.Region, "ListFileSystem()")
	start := call.Start()

	options := &vpcv1.ListSharesOptions{}
	res, _, err := filesystemHandler.VpcService.ListSharesWithContext(filesystemHandler.Ctx, options)
	if err != nil {
		return nil, err
	}

	var list []*irs.FileSystemInfo
	for _, share := range res.Shares {
		if share.Zone != nil && share.Zone.Name != nil && *share.Zone.Name == filesystemHandler.Region.Zone {
			info, err := filesystemHandler.setterFileSystemInfo(&share)
			if err != nil {
				cblogger.Warnf("ListFileSystem: setter error for %s: %v", *share.Name, err)
				continue
			}
			list = append(list, info)
		}
	}

	LoggingInfo(hiscallInfo, start)
	return list, nil
}

func (filesystemHandler *IbmFileSystemHandler) GetFileSystem(iid irs.IID) (irs.FileSystemInfo, error) {
	hiscallInfo := GetCallLogScheme(filesystemHandler.Region, call.FILESYSTEM, "FILESYSTEM", "GetFileSystem()")
	start := call.Start()

	var shareID string
	if iid.SystemId != "" {
		shareID = iid.SystemId
	} else if iid.NameId != "" {
		shares, _, err := filesystemHandler.VpcService.ListSharesWithContext(filesystemHandler.Ctx, &vpcv1.ListSharesOptions{})
		if err != nil {
			LoggingError(hiscallInfo, err)
			return irs.FileSystemInfo{}, fmt.Errorf("failed to list shares: %v", err)
		}

		for _, share := range shares.Shares {
			if *share.Name == iid.NameId {
				shareID = *share.ID
				break
			}
		}

		if shareID == "" {
			err := fmt.Errorf("file share with name %s not found", iid.NameId)
			LoggingError(hiscallInfo, err)
			return irs.FileSystemInfo{}, err
		}
	} else {
		err := fmt.Errorf("invalid IID: either NameId or SystemId is required")
		LoggingError(hiscallInfo, err)
		return irs.FileSystemInfo{}, err
	}

	getShareOptions := &vpcv1.GetShareOptions{
		ID: &shareID,
	}

	share, _, err := filesystemHandler.VpcService.GetShareWithContext(filesystemHandler.Ctx, getShareOptions)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.FileSystemInfo{}, fmt.Errorf("failed to get file share: %v", err)
	}

	info, err := filesystemHandler.setterFileSystemInfo(share)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.FileSystemInfo{}, fmt.Errorf("failed to parse file share info: %v", err)
	}

	LoggingInfo(hiscallInfo, start)
	return *info, nil
}

func (filesystemHandler *IbmFileSystemHandler) setterFileSystemInfo(share *vpcv1.Share) (*irs.FileSystemInfo, error) {
	if share == nil || share.Name == nil || share.ID == nil {
		return nil, fmt.Errorf("invalid Share input")
	}

	info := &irs.FileSystemInfo{
		IId: irs.IID{
			NameId:   *share.Name,
			SystemId: *share.ID,
		},
		Region:           filesystemHandler.Region.Region,
		Zone:             "",
		VpcIID:           irs.IID{},
		AccessSubnetList: nil,
		NFSVersion:       "4.1",
		FileSystemType:   irs.FileSystemType("nfs"),
		Encryption:       true,
		CapacityGB:       0,
		PerformanceInfo:  map[string]string{},
		Status:           irs.FileSystemStatus("Unknown"),
		CreatedTime:      time.Time{},
	}

	if share.Zone != nil && share.Zone.Name != nil {
		info.Zone = *share.Zone.Name
	}

	if share.Size != nil {
		info.CapacityGB = *share.Size
	}

	if share.CreatedAt != nil {
		info.CreatedTime = time.Time(*share.CreatedAt)
	}

	if share.LifecycleState != nil {
		switch *share.LifecycleState {
		case "stable":
			info.Status = irs.FileSystemAvailable
		case "pending":
			info.Status = irs.FileSystemCreating
		case "deleting":
			info.Status = irs.FileSystemDeleting
		case "failed":
			info.Status = irs.FileSystemError
		default:
			info.Status = irs.FileSystemStatus(*share.LifecycleState)
		}
	}

	if share.Iops != nil {
		info.PerformanceInfo["IOPS"] = fmt.Sprintf("%d", *share.Iops)
	}

	if share.MountTargets != nil && len(share.MountTargets) > 0 {
		mountTargetRef := share.MountTargets[0]
		if mountTargetRef.ID != nil {
			getMountTargetOptions := &vpcv1.GetShareMountTargetOptions{
				ShareID: share.ID,
				ID:      mountTargetRef.ID,
			}

			mountTarget, _, err := filesystemHandler.VpcService.GetShareMountTargetWithContext(filesystemHandler.Ctx, getMountTargetOptions)
			if err == nil && mountTarget.VirtualNetworkInterface != nil && mountTarget.VirtualNetworkInterface.ID != nil {
				getVniOptions := &vpcv1.GetVirtualNetworkInterfaceOptions{
					ID: mountTarget.VirtualNetworkInterface.ID,
				}

				vni, _, err := filesystemHandler.VpcService.GetVirtualNetworkInterfaceWithContext(filesystemHandler.Ctx, getVniOptions)
				if err == nil && vni.VPC != nil {
					if vni.VPC.Name != nil {
						info.VpcIID.NameId = *vni.VPC.Name
					}
					if vni.VPC.ID != nil {
						info.VpcIID.SystemId = *vni.VPC.ID
					}

					accessSubnets, err := filesystemHandler.ListAccessSubnet(irs.IID{
						NameId:   *share.Name,
						SystemId: *share.ID,
					})
					if err == nil {
						info.AccessSubnetList = accessSubnets
					}
				}
			}
		}
	}

	baseKVs := irs.StructToKeyValueList(share)
	info.KeyValueList = baseKVs

	return info, nil
}

func (filesystemHandler *IbmFileSystemHandler) DeleteFileSystem(iid irs.IID) (bool, error) {
	shareID := iid.SystemId
	if shareID == "" && iid.NameId != "" {
		shares, _, err := filesystemHandler.VpcService.ListSharesWithContext(filesystemHandler.Ctx, &vpcv1.ListSharesOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to list shares: %v", err)
		}

		for _, share := range shares.Shares {
			if *share.Name == iid.NameId {
				shareID = *share.ID
				break
			}
		}
	}

	if shareID == "" {
		return false, fmt.Errorf("invalid IID: filesystem not found")
	}

	var vpcID string

	getShareOptions := &vpcv1.GetShareOptions{
		ID: &shareID,
	}
	share, _, err := filesystemHandler.VpcService.GetShareWithContext(filesystemHandler.Ctx, getShareOptions)
	if err == nil && share.MountTargets != nil && len(share.MountTargets) > 0 {
		mountTargetRef := share.MountTargets[0]
		if mountTargetRef.ID != nil {
			getMountTargetOptions := &vpcv1.GetShareMountTargetOptions{
				ShareID: &shareID,
				ID:      mountTargetRef.ID,
			}

			mountTarget, _, err := filesystemHandler.VpcService.GetShareMountTargetWithContext(filesystemHandler.Ctx, getMountTargetOptions)
			if err == nil && mountTarget.VirtualNetworkInterface != nil && mountTarget.VirtualNetworkInterface.ID != nil {
				getVniOptions := &vpcv1.GetVirtualNetworkInterfaceOptions{
					ID: mountTarget.VirtualNetworkInterface.ID,
				}

				vni, _, err := filesystemHandler.VpcService.GetVirtualNetworkInterfaceWithContext(filesystemHandler.Ctx, getVniOptions)
				if err == nil && vni.Subnet != nil && vni.Subnet.ID != nil {
					vpcHandler := IbmVPCHandler{
						CredentialInfo: filesystemHandler.CredentialInfo,
						Region:         filesystemHandler.Region,
						VpcService:     filesystemHandler.VpcService,
						Ctx:            filesystemHandler.Ctx,
						TaggingService: filesystemHandler.TaggingService,
						SearchService:  filesystemHandler.SearchService,
					}
					vpcList, err := vpcHandler.ListVPC()
					if err != nil {
						return false, fmt.Errorf("failed to get VPC list: %v", err)
					}

					for _, vpc := range vpcList {
						for _, subnet := range vpc.SubnetInfoList {
							if subnet.IId.SystemId == *vni.Subnet.ID {
								vpcID = vpc.IId.SystemId
								break
							}
						}
					}

					if vpcID == "" {
						return false, fmt.Errorf("failed to get VPC info")
					}
				}
			}
		}
	}

	deleteShareOptions := &vpcv1.DeleteShareOptions{
		ID: &shareID,
	}

	_, _, err = filesystemHandler.VpcService.DeleteShareWithContext(filesystemHandler.Ctx, deleteShareOptions)
	if err != nil {
		return false, fmt.Errorf("failed to delete file share: %v", err)
	}

	// Wait a bit for IBM Cloud to fully release the security group from FileSystem
	time.Sleep(5 * time.Second)

	// Delete the security group with retry logic
	err = filesystemHandler.deleteCbspiderSecurityGroup(vpcID)
	if err != nil {
		cblogger.Errorf("failed to delete file share's security group: %v", err)
		return false, fmt.Errorf("failed to delete file share's security group: %v", err)
	}

	return true, nil
}

func (filesystemHandler *IbmFileSystemHandler) AddAccessSubnet(fsIID irs.IID, subnetIID irs.IID) (irs.FileSystemInfo, error) {
	if fsIID.NameId == "" && fsIID.SystemId == "" {
		return irs.FileSystemInfo{}, fmt.Errorf("invalid filesystem IID: NameId or SystemId is required")
	}

	shareID := fsIID.SystemId
	if shareID == "" {
		list, err := filesystemHandler.ListIID()
		if err != nil {
			return irs.FileSystemInfo{}, err
		}

		for _, iid := range list {
			if iid.NameId == fsIID.NameId {
				shareID = iid.SystemId
				break
			}
		}
	}

	if shareID == "" {
		return irs.FileSystemInfo{}, fmt.Errorf("filesystem not found")
	}

	var targetSubnetID string
	if subnetIID.SystemId != "" {
		targetSubnetID = subnetIID.SystemId
	} else if subnetIID.NameId != "" {
		id, err := filesystemHandler.findSubnetIDByName(subnetIID.NameId)
		if err != nil {
			return irs.FileSystemInfo{}, err
		}
		targetSubnetID = id
	} else {
		return irs.FileSystemInfo{}, fmt.Errorf("invalid subnet IID: either NameId or SystemId is required")
	}

	targetCIDR, err := filesystemHandler.getSubnetCIDR(targetSubnetID)
	if err != nil {
		return irs.FileSystemInfo{}, fmt.Errorf("failed to get target subnet CIDR: %v", err)
	}

	getSubnetOptions := &vpcv1.GetSubnetOptions{
		ID: &targetSubnetID,
	}
	subnet, _, err := filesystemHandler.VpcService.GetSubnetWithContext(filesystemHandler.Ctx, getSubnetOptions)
	if err != nil {
		return irs.FileSystemInfo{}, fmt.Errorf("failed to get subnet info: %v", err)
	}

	if subnet.VPC == nil || subnet.VPC.ID == nil {
		return irs.FileSystemInfo{}, fmt.Errorf("subnet does not have VPC information")
	}

	securityGroupID, err := filesystemHandler.findOrCreateCbspiderSecurityGroup(*subnet.VPC.ID)
	if err != nil {
		return irs.FileSystemInfo{}, fmt.Errorf("failed to find/create filesystem's security group: %v", err)
	}

	listRulesOptions := &vpcv1.ListSecurityGroupRulesOptions{
		SecurityGroupID: &securityGroupID,
	}
	rules, _, err := filesystemHandler.VpcService.ListSecurityGroupRulesWithContext(filesystemHandler.Ctx, listRulesOptions)
	if err != nil {
		return irs.FileSystemInfo{}, fmt.Errorf("failed to check security group rules: %v", err)
	}

	for _, rule := range rules.Rules {
		if sgRule, ok := rule.(*vpcv1.SecurityGroupRuleSecurityGroupRuleProtocolTcpudp); ok {
			if sgRule.Direction != nil && *sgRule.Direction == "inbound" &&
				sgRule.Protocol != nil && *sgRule.Protocol == "tcp" &&
				sgRule.PortMin != nil && *sgRule.PortMin == 2049 &&
				sgRule.PortMax != nil && *sgRule.PortMax == 2049 &&
				sgRule.Remote != nil {
				if remoteCIDR, ok := sgRule.Remote.(*vpcv1.SecurityGroupRuleRemote); ok {
					if remoteCIDR.CIDRBlock != nil && *remoteCIDR.CIDRBlock == targetCIDR {
						return irs.FileSystemInfo{}, fmt.Errorf("provided subnet is already added")
					}
				}
			}
		}
	}

	createRuleOptions := &vpcv1.CreateSecurityGroupRuleOptions{
		SecurityGroupID: &securityGroupID,
		SecurityGroupRulePrototype: &vpcv1.SecurityGroupRulePrototype{
			Direction: stringPtr("inbound"),
			Protocol:  stringPtr("tcp"),
			PortMin:   int64Ptr(2049),
			PortMax:   int64Ptr(2049),
			Remote: &vpcv1.SecurityGroupRuleRemotePrototype{
				CIDRBlock: &targetCIDR,
			},
		},
	}

	_, _, err = filesystemHandler.VpcService.CreateSecurityGroupRuleWithContext(filesystemHandler.Ctx, createRuleOptions)
	if err != nil {
		return irs.FileSystemInfo{}, fmt.Errorf("failed to add security group rule: %v", err)
	}

	time.Sleep(10 * time.Second)

	fsInfo, err := filesystemHandler.GetFileSystem(fsIID)
	if err != nil {
		return irs.FileSystemInfo{}, fmt.Errorf("failed to get updated filesystem info: %v", err)
	}

	return fsInfo, nil
}

func (filesystemHandler *IbmFileSystemHandler) RemoveAccessSubnet(fsIID irs.IID, subnetIID irs.IID) (bool, error) {
	if fsIID.NameId == "" && fsIID.SystemId == "" {
		return false, fmt.Errorf("invalid filesystem IID: NameId or SystemId is required")
	}

	shareID := fsIID.SystemId
	if shareID == "" {
		list, err := filesystemHandler.ListIID()
		if err != nil {
			return false, err
		}

		for _, iid := range list {
			if iid.NameId == fsIID.NameId {
				shareID = iid.SystemId
				break
			}
		}
	}

	if shareID == "" {
		return false, fmt.Errorf("filesystem not found")
	}

	var targetSubnetID string
	if subnetIID.SystemId != "" {
		targetSubnetID = subnetIID.SystemId
	} else if subnetIID.NameId != "" {
		id, err := filesystemHandler.findSubnetIDByName(subnetIID.NameId)
		if err != nil {
			return false, err
		}
		targetSubnetID = id
	} else {
		return false, fmt.Errorf("invalid subnet IID: either NameId or SystemId is required")
	}

	targetCIDR, err := filesystemHandler.getSubnetCIDR(targetSubnetID)
	if err != nil {
		return false, fmt.Errorf("failed to get target subnet CIDR: %v", err)
	}

	getSubnetOptions := &vpcv1.GetSubnetOptions{
		ID: &targetSubnetID,
	}
	subnet, _, err := filesystemHandler.VpcService.GetSubnetWithContext(filesystemHandler.Ctx, getSubnetOptions)
	if err != nil {
		return false, fmt.Errorf("failed to get subnet info: %v", err)
	}

	if subnet.VPC == nil || subnet.VPC.ID == nil {
		return false, fmt.Errorf("subnet does not have VPC information")
	}

	listSgOptions := &vpcv1.ListSecurityGroupsOptions{
		VPCID: subnet.VPC.ID,
	}
	securityGroups, _, err := filesystemHandler.VpcService.ListSecurityGroupsWithContext(filesystemHandler.Ctx, listSgOptions)
	if err != nil {
		return false, fmt.Errorf("failed to list security groups: %v", err)
	}

	sgName := generateSGNameForFileSystem(*subnet.VPC.ID)

	var securityGroupID string
	for _, sg := range securityGroups.SecurityGroups {
		if sg.Name != nil && *sg.Name == sgName {
			securityGroupID = *sg.ID
			break
		}
	}

	if securityGroupID == "" {
		return false, fmt.Errorf("filesystem's security group not found")
	}

	listRulesOptions := &vpcv1.ListSecurityGroupRulesOptions{
		SecurityGroupID: &securityGroupID,
	}
	rules, _, err := filesystemHandler.VpcService.ListSecurityGroupRulesWithContext(filesystemHandler.Ctx, listRulesOptions)
	if err != nil {
		return false, fmt.Errorf("failed to list security group rules: %v", err)
	}

	var ruleIDToDelete string
	for _, rule := range rules.Rules {
		if sgRule, ok := rule.(*vpcv1.SecurityGroupRuleSecurityGroupRuleProtocolTcpudp); ok {
			if sgRule.Direction != nil && *sgRule.Direction == "inbound" &&
				sgRule.Protocol != nil && *sgRule.Protocol == "tcp" &&
				sgRule.PortMin != nil && *sgRule.PortMin == 2049 &&
				sgRule.PortMax != nil && *sgRule.PortMax == 2049 &&
				sgRule.Remote != nil {
				if remoteCIDR, ok := sgRule.Remote.(*vpcv1.SecurityGroupRuleRemote); ok {
					if remoteCIDR.CIDRBlock != nil && *remoteCIDR.CIDRBlock == targetCIDR {
						ruleIDToDelete = *sgRule.ID
						break
					}
				}
			}
		}
	}

	if ruleIDToDelete == "" {
		return false, fmt.Errorf("security group rule for subnet CIDR %s not found", targetCIDR)
	}

	deleteRuleOptions := &vpcv1.DeleteSecurityGroupRuleOptions{
		SecurityGroupID: &securityGroupID,
		ID:              &ruleIDToDelete,
	}

	_, err = filesystemHandler.VpcService.DeleteSecurityGroupRuleWithContext(filesystemHandler.Ctx, deleteRuleOptions)
	if err != nil {
		return false, fmt.Errorf("failed to delete security group rule: %v", err)
	}

	return true, nil
}

func (filesystemHandler *IbmFileSystemHandler) ListAccessSubnet(fsIID irs.IID) ([]irs.IID, error) {
	if fsIID.NameId == "" && fsIID.SystemId == "" {
		return nil, fmt.Errorf("invalid filesystem IID: NameId or SystemId is required")
	}

	shareID := fsIID.SystemId
	if shareID == "" {
		list, err := filesystemHandler.ListIID()
		if err != nil {
			return nil, err
		}

		for _, iid := range list {
			if iid.NameId == fsIID.NameId {
				shareID = iid.SystemId
				break
			}
		}
	}

	if shareID == "" {
		return nil, fmt.Errorf("filesystem not found")
	}

	getShareOptions := &vpcv1.GetShareOptions{
		ID: &shareID,
	}
	share, _, err := filesystemHandler.VpcService.GetShareWithContext(filesystemHandler.Ctx, getShareOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to get share: %v", err)
	}

	if share.MountTargets == nil || len(share.MountTargets) == 0 {
		return []irs.IID{}, nil
	}

	mountTargetRef := share.MountTargets[0]
	getMountTargetOptions := &vpcv1.GetShareMountTargetOptions{
		ShareID: &shareID,
		ID:      mountTargetRef.ID,
	}

	mountTarget, _, err := filesystemHandler.VpcService.GetShareMountTargetWithContext(filesystemHandler.Ctx, getMountTargetOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to get mount target: %v", err)
	}

	if mountTarget.VirtualNetworkInterface == nil || mountTarget.VirtualNetworkInterface.ID == nil {
		return nil, fmt.Errorf("mount target does not have virtual network interface")
	}

	getVniOptions := &vpcv1.GetVirtualNetworkInterfaceOptions{
		ID: mountTarget.VirtualNetworkInterface.ID,
	}

	vni, _, err := filesystemHandler.VpcService.GetVirtualNetworkInterfaceWithContext(filesystemHandler.Ctx, getVniOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual network interface: %v", err)
	}

	if vni.Subnet == nil || vni.Subnet.ID == nil {
		return nil, fmt.Errorf("virtual network interface does not have subnet")
	}

	vpcHandler := IbmVPCHandler{
		CredentialInfo: filesystemHandler.CredentialInfo,
		Region:         filesystemHandler.Region,
		VpcService:     filesystemHandler.VpcService,
		Ctx:            filesystemHandler.Ctx,
		TaggingService: filesystemHandler.TaggingService,
		SearchService:  filesystemHandler.SearchService,
	}
	vpcList, err := vpcHandler.ListVPC()
	if err != nil {
		return nil, fmt.Errorf("failed to get VPC list: %v", err)
	}

	var vpcID string
	for _, vpc := range vpcList {
		for _, subnet := range vpc.SubnetInfoList {
			if subnet.IId.SystemId == *vni.Subnet.ID {
				vpcID = vpc.IId.SystemId
				break
			}
		}
	}

	if vpcID == "" {
		return nil, fmt.Errorf("failed to get VPC info")
	}

	securityHandler := IbmSecurityHandler{
		CredentialInfo: filesystemHandler.CredentialInfo,
		Region:         filesystemHandler.Region,
		VpcService:     filesystemHandler.VpcService,
		Ctx:            filesystemHandler.Ctx,
		TaggingService: filesystemHandler.TaggingService,
		SearchService:  filesystemHandler.SearchService,
	}
	securityList, err := securityHandler.ListSecurity()
	if err != nil {
		return nil, fmt.Errorf("failed to get security list: %v", err)
	}

	sgName := generateSGNameForFileSystem(vpcID)
	var sgID string
	for _, sg := range securityList {
		if sg.IId.NameId == sgName {
			sgID = sg.IId.SystemId
			break
		}
	}

	if sgID == "" {
		return nil, fmt.Errorf("failed to get security info")
	}

	listRulesOptions := &vpcv1.ListSecurityGroupRulesOptions{
		SecurityGroupID: &sgID,
	}
	rules, _, err := filesystemHandler.VpcService.ListSecurityGroupRulesWithContext(filesystemHandler.Ctx, listRulesOptions)
	if err != nil {
		return []irs.IID{}, nil
	}

	var subnetList []irs.IID
	for _, rule := range rules.Rules {
		if sgRule, ok := rule.(*vpcv1.SecurityGroupRuleSecurityGroupRuleProtocolTcpudp); ok {
			if sgRule.Direction != nil && *sgRule.Direction == "inbound" &&
				sgRule.Protocol != nil && *sgRule.Protocol == "tcp" &&
				sgRule.PortMin != nil && *sgRule.PortMin == 2049 &&
				sgRule.PortMax != nil && *sgRule.PortMax == 2049 &&
				sgRule.Remote != nil {
				if remoteCIDR, ok := sgRule.Remote.(*vpcv1.SecurityGroupRuleRemote); ok {
					if remoteCIDR.CIDRBlock != nil {
						subnetIID, err := filesystemHandler.findSubnetByCIDR(*remoteCIDR.CIDRBlock)
						if err == nil && subnetIID != nil {
							subnetList = append(subnetList, *subnetIID)
						}
					}
				}
			}
		}
	}

	return subnetList, nil
}

func (filesystemHandler *IbmFileSystemHandler) ScheduleBackup(reqInfo irs.FileSystemBackupInfo) (irs.FileSystemBackupInfo, error) {
	return irs.FileSystemBackupInfo{}, fmt.Errorf("backup scheduling is not supported in IBM Cloud")
}

func (filesystemHandler *IbmFileSystemHandler) OnDemandBackup(fsIID irs.IID) (irs.FileSystemBackupInfo, error) {
	return irs.FileSystemBackupInfo{}, fmt.Errorf("on-demand backup is not supported in IBM Cloud")
}

func (filesystemHandler *IbmFileSystemHandler) ListBackup(fsIID irs.IID) ([]irs.FileSystemBackupInfo, error) {
	return nil, fmt.Errorf("backup listing is not supported in IBM Cloud")
}

func (filesystemHandler *IbmFileSystemHandler) GetBackup(fsIID irs.IID, backupID string) (irs.FileSystemBackupInfo, error) {
	return irs.FileSystemBackupInfo{}, fmt.Errorf("backup retrieval is not supported in IBM Cloud")
}

func (filesystemHandler *IbmFileSystemHandler) DeleteBackup(fsIID irs.IID, backupID string) (bool, error) {
	return false, fmt.Errorf("backup deletion is not supported in IBM Cloud")
}

func stringPtr(s string) *string {
	return &s
}

func int64Ptr(i int64) *int64 {
	return &i
}
