package resources

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/IBM/platform-services-go-sdk/globalsearchv2"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/globaltaggingv1"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type IbmVPCHandler struct {
	Region         idrv.RegionInfo
	CredentialInfo idrv.CredentialInfo
	VpcService     *vpcv1.VpcV1
	Ctx            context.Context
	TaggingService *globaltaggingv1.GlobalTaggingV1
	SearchService  *globalsearchv2.GlobalSearchV2
}

func checkValidVpcReqInfo(vpcReqInfo irs.VPCReqInfo) error {
	if vpcReqInfo.IId.NameId == "" {
		return errors.New("invalid VPCReqInfo NameId")
	}
	if vpcReqInfo.IPv4_CIDR == "" {
		return errors.New("invalid VPCReqInfo IPv4_CIDR")
	}
	if vpcReqInfo.SubnetInfoList != nil {
		for _, subnetInfo := range vpcReqInfo.SubnetInfoList {
			if subnetInfo.IPv4_CIDR == "" || subnetInfo.IId.NameId == "" {
				return errors.New("invalid subnetInfo")
			}
		}
	}
	return nil
}

func (vpcHandler *IbmVPCHandler) CreateVPC(vpcReqInfo irs.VPCReqInfo) (irs.VPCInfo, error) {
	hiscallInfo := GetCallLogScheme(vpcHandler.Region, call.VPCSUBNET, "VPC", "CreateVPC()")
	start := call.Start()
	err := checkValidVpcReqInfo(vpcReqInfo)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VPC err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}

	if vpcHandler.Region.Zone == "" {
		createErr := errors.New(fmt.Sprintf("Failed to Create VPC err = Zone is not provided"))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}

	exist, err := existVpc(vpcReqInfo.IId, vpcHandler.VpcService, vpcHandler.Ctx)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VPC err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}
	if exist {
		createErr := errors.New(fmt.Sprintf("Failed to Create VPC err = VPC is already exist"))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}

	if len(vpcReqInfo.SubnetInfoList) == 0 {
		createErr := errors.New("Failed to Create VPC err = Subnet info list is not provided")
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}

	// Create VPC
	createVPCOptions := &vpcv1.CreateVPCOptions{}
	createVPCOptions.SetAddressPrefixManagement("manual")
	createVPCOptions.SetName(vpcReqInfo.IId.NameId)
	vpc, _, err := vpcHandler.VpcService.CreateVPCWithContext(vpcHandler.Ctx, createVPCOptions)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VPC err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}
	newVpcIId := irs.IID{
		NameId:   *vpc.Name,
		SystemId: *vpc.ID,
	}

	// If the zone is not specified in the subnet, use the zone of the connection.
	for i, subnetInfo := range vpcReqInfo.SubnetInfoList {
		if subnetInfo.Zone == "" {
			vpcReqInfo.SubnetInfoList[i].Zone = vpcHandler.Region.Zone
		}
	}

	for _, si := range vpcReqInfo.SubnetInfoList {

		opt := &vpcv1.CreateVPCAddressPrefixOptions{}
		opt.SetVPCID(newVpcIId.SystemId)
		opt.SetCIDR(vpcReqInfo.IPv4_CIDR)
		opt.SetName(fmt.Sprintf("%s-%s-prefix", newVpcIId.NameId, si.Zone))
		opt.SetZone(&vpcv1.ZoneIdentity{Name: core.StringPtr(si.Zone)})

		_, _, err := vpcHandler.VpcService.
			CreateVPCAddressPrefixWithContext(vpcHandler.Ctx, opt)
		if err != nil {
			_, _ = vpcHandler.DeleteVPC(newVpcIId)
			createErr := fmt.Errorf("Failed to create Address Prefix in zone %s: %w", si.Zone, err)
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VPCInfo{}, createErr
		}
	}

	if vpcReqInfo.SubnetInfoList != nil && len(vpcReqInfo.SubnetInfoList) > 0 {
		for _, subnetInfo := range vpcReqInfo.SubnetInfoList {
			err = attachSubnet(*vpc, subnetInfo, vpcHandler.VpcService, vpcHandler.Ctx)
			if err != nil {
				_, delErr := vpcHandler.DeleteVPC(newVpcIId)
				if delErr != nil {
					err = errors.New(err.Error() + delErr.Error())
				}
				createErr := errors.New(fmt.Sprintf("Failed to Create VPC err = %s", err.Error()))
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
				return irs.VPCInfo{}, createErr
			}

			// Attach Tag
			if subnetInfo.TagList != nil && len(subnetInfo.TagList) > 0 {
				for _, tag := range subnetInfo.TagList {
					subnet, err := getRawSubnet(subnetInfo.IId, vpcHandler.VpcService, vpcHandler.Ctx)
					if err != nil {
						createErr := errors.New(fmt.Sprintf("Failed to get created Subnet info err = %s", err.Error()))
						cblogger.Error(createErr.Error())
						LoggingError(hiscallInfo, createErr)
						return irs.VPCInfo{}, createErr
					}

					if subnet.CRN == nil {
						createErr := errors.New(fmt.Sprintf("Failed to get created Subnet's CRN"))
						cblogger.Error(createErr.Error())
						LoggingError(hiscallInfo, createErr)
						return irs.VPCInfo{}, createErr
					}

					err = addTag(vpcHandler.TaggingService, tag, *subnet.CRN)
					if err != nil {
						createErr := errors.New(fmt.Sprintf("Failed to Attach Tag on Subnet err = %s", err.Error()))
						cblogger.Error(createErr.Error())
						LoggingError(hiscallInfo, createErr)
					}
				}
			}
		}
	}
	//// default SecurityGroup modify
	options := &vpcv1.GetSecurityGroupOptions{}
	options.SetID(*vpc.DefaultSecurityGroup.ID)
	sg, _, err := vpcHandler.VpcService.GetSecurityGroupWithContext(vpcHandler.Ctx, options)

	if err != nil {
		_, delErr := vpcHandler.DeleteVPC(newVpcIId)
		if delErr != nil {
			err = errors.New(err.Error() + delErr.Error())
		}
		createErr := errors.New(fmt.Sprintf("Failed to Create VPC err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}

	err = ModifyVPCDefaultRule(sg.Rules, irs.IID{NameId: *sg.Name, SystemId: *sg.ID}, vpcHandler.VpcService, vpcHandler.Ctx)

	if err != nil {
		_, delErr := vpcHandler.DeleteVPC(newVpcIId)
		if delErr != nil {
			err = errors.New(err.Error() + delErr.Error())
		}
		createErr := errors.New(fmt.Sprintf("Failed to Create VPC err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}

	// Attach Tag
	if vpcReqInfo.TagList != nil && len(vpcReqInfo.TagList) > 0 {
		if vpc.CRN == nil {
			createErr := errors.New(fmt.Sprintf("Failed to get created VPC's CRN"))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VPCInfo{}, createErr
		}

		for _, tag := range vpcReqInfo.TagList {
			err = addTag(vpcHandler.TaggingService, tag, *vpc.CRN)
			if err != nil {
				createErr := errors.New(fmt.Sprintf("Failed to Attach Tag to VPC err = %s", err.Error()))
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
			}
		}
	}

	vpcInfo, err := vpcHandler.setVPCInfo(*vpc, vpcHandler.VpcService, vpcHandler.Ctx)
	if err != nil {
		_, delErr := vpcHandler.DeleteVPC(newVpcIId)
		if delErr != nil {
			err = errors.New(err.Error() + delErr.Error())
		}
		createErr := errors.New(fmt.Sprintf("Failed to Create VPC err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}

	LoggingInfo(hiscallInfo, start)

	return vpcInfo, nil
}
func (vpcHandler *IbmVPCHandler) ListVPC() ([]*irs.VPCInfo, error) {
	hiscallInfo := GetCallLogScheme(vpcHandler.Region, call.VPCSUBNET, "VPC", "ListVPC()")
	listVpcsOptions := &vpcv1.ListVpcsOptions{}
	var vpcInfos []*irs.VPCInfo
	start := call.Start()
	vpcs, _, err := vpcHandler.VpcService.ListVpcsWithContext(vpcHandler.Ctx, listVpcsOptions)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List VPC err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}
	// Next Check
	for {
		for _, vpc := range vpcs.Vpcs {
			vpcInfo, err := vpcHandler.setVPCInfo(vpc, vpcHandler.VpcService, vpcHandler.Ctx)
			if err != nil {
				getErr := errors.New(fmt.Sprintf("Failed to List VPC err = %s", err.Error()))
				cblogger.Error(getErr.Error())
				LoggingError(hiscallInfo, getErr)
				return nil, getErr
			}
			vpcInfos = append(vpcInfos, &vpcInfo)
		}
		nextstr, _ := getVPCNextHref(vpcs.Next)
		if nextstr != "" {
			listVpcsOptions2 := &vpcv1.ListVpcsOptions{
				Start: core.StringPtr(nextstr),
			}
			vpcs, _, err = vpcHandler.VpcService.ListVpcsWithContext(vpcHandler.Ctx, listVpcsOptions2)
			if err != nil {
				getErr := errors.New(fmt.Sprintf("Failed to List VPC err = %s", err.Error()))
				cblogger.Error(getErr.Error())
				LoggingError(hiscallInfo, getErr)
				return nil, getErr
			}
		} else {
			break
		}
	}
	LoggingInfo(hiscallInfo, start)
	return vpcInfos, nil
}
func (vpcHandler *IbmVPCHandler) GetVPC(vpcIID irs.IID) (irs.VPCInfo, error) {
	hiscallInfo := GetCallLogScheme(vpcHandler.Region, call.VPCSUBNET, "VPC", "GetVPC()")
	start := call.Start()
	vpc, err := GetRawVPC(vpcIID, vpcHandler.VpcService, vpcHandler.Ctx)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VPC err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.VPCInfo{}, getErr
	}
	// default SecurityGroup modify

	options := &vpcv1.GetSecurityGroupOptions{}
	options.SetID(*vpc.DefaultSecurityGroup.ID)
	sg, _, err := vpcHandler.VpcService.GetSecurityGroupWithContext(vpcHandler.Ctx, options)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VPC err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.VPCInfo{}, getErr
	}
	err = ModifyVPCDefaultRule(sg.Rules, irs.IID{NameId: *sg.Name, SystemId: *sg.ID}, vpcHandler.VpcService, vpcHandler.Ctx)

	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VPC err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.VPCInfo{}, getErr
	}
	vpcInfo, err := vpcHandler.setVPCInfo(vpc, vpcHandler.VpcService, vpcHandler.Ctx)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VPC err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.VPCInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)
	return vpcInfo, nil
}
func (vpcHandler *IbmVPCHandler) DeleteVPC(vpcIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(vpcHandler.Region, call.VPCSUBNET, vpcIID.NameId, "DeleteVPC()")
	start := call.Start()
	vpc, err := GetRawVPC(vpcIID, vpcHandler.VpcService, vpcHandler.Ctx)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete VPC err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}

	// Remove all Subnet
	rawSubnets, err := getVPCRawSubnets(vpc, vpcHandler.VpcService, vpcHandler.Ctx)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete VPC err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	if rawSubnets != nil {
		for _, subnet := range rawSubnets {
			// Clean up public gateway before deleting subnet
			err := vpcHandler.cleanupSubnetPublicGateway(*subnet.ID)
			if err != nil {
				cblogger.Warnf("Failed to cleanup public gateway for subnet %s: %v", *subnet.ID, err)
				// Continue with subnet deletion even if public gateway cleanup fails
			}

			options := &vpcv1.DeleteSubnetOptions{}
			options.SetID(*subnet.ID)
			_, err = vpcHandler.VpcService.DeleteSubnetWithContext(vpcHandler.Ctx, options)
			if err != nil {
				delErr := errors.New(fmt.Sprintf("Failed to Delete VPC err = %s", err.Error()))
				cblogger.Error(delErr.Error())
				LoggingError(hiscallInfo, delErr)
				return false, delErr
			}
		}
	}
	// subnets delete Time delay
	curRetryCnt := 0
	maxRetryCnt := 60
	for {
		rawDeleteSubnets, err := getVPCRawSubnets(vpc, vpcHandler.VpcService, vpcHandler.Ctx)
		if err != nil {
			delErr := errors.New(fmt.Sprintf("Failed to Delete VPC err = %s", err.Error()))
			cblogger.Error(delErr.Error())
			LoggingError(hiscallInfo, delErr)
			return false, delErr
		}
		if rawDeleteSubnets == nil {
			break
		}
		curRetryCnt++
		time.Sleep(1 * time.Second)
		if curRetryCnt > maxRetryCnt {
			err = errors.New("failed delete VPC - subnets delete TimeOut")
			delErr := errors.New(fmt.Sprintf("Failed to Delete VPC err = %s", err.Error()))
			cblogger.Error(delErr.Error())
			LoggingError(hiscallInfo, delErr)
			return false, delErr
		}
	}
	// Delete VPC
	deleteVpcOptions := &vpcv1.DeleteVPCOptions{}
	deleteVpcOptions.SetID(*vpc.ID)
	_, err = vpcHandler.VpcService.DeleteVPCWithContext(vpcHandler.Ctx, deleteVpcOptions)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete VPC err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	LoggingInfo(hiscallInfo, start)

	deleteUnusedTags(vpcHandler.TaggingService)

	return true, nil
}
func (vpcHandler *IbmVPCHandler) AddSubnet(vpcIID irs.IID, subnetInfo irs.SubnetInfo) (irs.VPCInfo, error) {
	hiscallInfo := GetCallLogScheme(vpcHandler.Region, call.VPCSUBNET, subnetInfo.IId.NameId, "AddSubnet()")
	start := call.Start()

	if subnetInfo.Zone == "" {
		subnetInfo.Zone = vpcHandler.Region.Zone
	}

	vpc, err := GetRawVPC(vpcIID, vpcHandler.VpcService, vpcHandler.Ctx)
	if err != nil {
		addSubnetErr := errors.New(fmt.Sprintf("Failed to Add Subnet err = %s", err.Error()))
		cblogger.Error(addSubnetErr.Error())
		LoggingError(hiscallInfo, addSubnetErr)
		return irs.VPCInfo{}, addSubnetErr
	}

	if subnetInfo.Zone == "" {
		subnetInfo.Zone = vpcHandler.Region.Zone
	}

	err = attachSubnet(vpc, subnetInfo, vpcHandler.VpcService, vpcHandler.Ctx)
	if err != nil {
		addSubnetErr := errors.New(fmt.Sprintf("Failed to Add Subnet err = %s", err.Error()))
		cblogger.Error(addSubnetErr.Error())
		LoggingError(hiscallInfo, addSubnetErr)
		return irs.VPCInfo{}, addSubnetErr
	}
	LoggingInfo(hiscallInfo, start)
	vpc, err = GetRawVPC(vpcIID, vpcHandler.VpcService, vpcHandler.Ctx)
	if err != nil {
		addSubnetErr := errors.New(fmt.Sprintf("Failed to Add Subnet err = %s", err.Error()))
		cblogger.Error(addSubnetErr.Error())
		LoggingError(hiscallInfo, addSubnetErr)
		return irs.VPCInfo{}, addSubnetErr
	}
	vpcInfo, err := vpcHandler.setVPCInfo(vpc, vpcHandler.VpcService, vpcHandler.Ctx)
	if err != nil {
		addSubnetErr := errors.New(fmt.Sprintf("Failed to Add Subnet err = %s", err.Error()))
		cblogger.Error(addSubnetErr.Error())
		LoggingError(hiscallInfo, addSubnetErr)
		return irs.VPCInfo{}, addSubnetErr
	}
	return vpcInfo, nil
}
func (vpcHandler *IbmVPCHandler) RemoveSubnet(vpcIID irs.IID, subnetIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(vpcHandler.Region, call.VPCSUBNET, subnetIID.NameId, "RemoveSubnet()")
	start := call.Start()
	if subnetIID.SystemId != "" {
		// Clean up public gateway before deleting subnet
		err := vpcHandler.cleanupSubnetPublicGateway(subnetIID.SystemId)
		if err != nil {
			cblogger.Warnf("Failed to cleanup public gateway for subnet %s: %v", subnetIID.SystemId, err)
			// Continue with subnet deletion even if public gateway cleanup fails
		}

		options := &vpcv1.DeleteSubnetOptions{}
		options.SetID(subnetIID.SystemId)
		_, err = vpcHandler.VpcService.DeleteSubnetWithContext(vpcHandler.Ctx, options)
		if err != nil {
			delErr := errors.New(fmt.Sprintf("Failed to Remove Subnet err = %s", err.Error()))
			cblogger.Error(delErr.Error())
			LoggingError(hiscallInfo, delErr)
			return false, delErr
		}
		LoggingInfo(hiscallInfo, start)
		return true, nil
	}
	vpc, err := GetRawVPC(vpcIID, vpcHandler.VpcService, vpcHandler.Ctx)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Remove Subnet err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	rawSubnets, err := getVPCRawSubnets(vpc, vpcHandler.VpcService, vpcHandler.Ctx)
	if len(rawSubnets) > 0 {
		for _, subnet := range rawSubnets {
			if *subnet.Name == subnetIID.NameId {
				// Clean up public gateway before deleting subnet
				err := vpcHandler.cleanupSubnetPublicGateway(*subnet.ID)
				if err != nil {
					cblogger.Warnf("Failed to cleanup public gateway for subnet %s: %v", *subnet.ID, err)
					// Continue with subnet deletion even if public gateway cleanup fails
				}

				options := &vpcv1.DeleteSubnetOptions{}
				options.SetID(*subnet.ID)
				_, err = vpcHandler.VpcService.DeleteSubnetWithContext(vpcHandler.Ctx, options)
				if err != nil {
					delErr := errors.New(fmt.Sprintf("Failed to Remove Subnet err = %s", err.Error()))
					cblogger.Error(delErr.Error())
					LoggingError(hiscallInfo, delErr)
					return false, delErr
				}
				LoggingInfo(hiscallInfo, start)
				return true, nil
			}
		}
	}
	err = errors.New("not found subnet")
	delErr := errors.New(fmt.Sprintf("Failed to Remove Subnet err = %s", err.Error()))
	cblogger.Error(delErr.Error())
	LoggingError(hiscallInfo, delErr)

	deleteUnusedTags(vpcHandler.TaggingService)

	return false, delErr
}

func (vpcHandler *IbmVPCHandler) setVPCInfo(vpc vpcv1.VPC, vpcService *vpcv1.VpcV1, ctx context.Context) (irs.VPCInfo, error) {
	vpcInfo := irs.VPCInfo{
		IId: irs.IID{
			NameId:   *vpc.Name,
			SystemId: *vpc.ID,
		},
	}
	listVpcAddressPrefixesOptions := &vpcv1.ListVPCAddressPrefixesOptions{}
	listVpcAddressPrefixesOptions.SetVPCID(*vpc.ID)
	addressPrefixes, _, err := vpcService.ListVPCAddressPrefixesWithContext(ctx, listVpcAddressPrefixesOptions)
	if err != nil {
		return irs.VPCInfo{}, err
	}
	if *addressPrefixes.TotalCount > 0 {
		cidr := *addressPrefixes.AddressPrefixes[0].CIDR
		vpcInfo.IPv4_CIDR = cidr
	}

	tagHandler := IbmTagHandler{
		Region:         vpcHandler.Region,
		CredentialInfo: vpcHandler.CredentialInfo,
		VpcService:     vpcHandler.VpcService,
		Ctx:            vpcHandler.Ctx,
		SearchService:  vpcHandler.SearchService,
	}

	tags, err := tagHandler.ListTag(irs.VPC, vpcInfo.IId)
	if err != nil {
		cblogger.Warn("Failed to get tags of the VPC (" + vpcInfo.IId.NameId + "). err = " + err.Error())
	}
	vpcInfo.TagList = tags

	vpcInfo.KeyValueList = irs.StructToKeyValueList(vpc)

	rawSubnets, err := getVPCRawSubnets(vpc, vpcService, ctx)
	if err != nil {
		return irs.VPCInfo{}, err
	}
	if len(rawSubnets) > 0 {
		var newSubnetInfos []irs.SubnetInfo
		for _, subnet := range rawSubnets {
			subnetInfo := irs.SubnetInfo{
				IId: irs.IID{
					NameId:   *subnet.Name,
					SystemId: *subnet.ID,
				},
				Zone:      *subnet.Zone.Name,
				IPv4_CIDR: *subnet.Ipv4CIDRBlock,
			}

			subnetInfo.KeyValueList = irs.StructToKeyValueList(subnet)

			tags, err := tagHandler.ListTag(irs.SUBNET, subnetInfo.IId)
			if err != nil {
				cblogger.Warn("failed to get tags of the subnet (" + subnetInfo.IId.NameId + ")")
			}
			subnetInfo.TagList = tags

			newSubnetInfos = append(newSubnetInfos, subnetInfo)
		}
		vpcInfo.SubnetInfoList = newSubnetInfos

		// VPC의 key-value 리스트와 별도로 서브넷들의 key-value 리스트 생성
		var subnetKeyValueList []irs.KeyValue
		for _, subnetInfo := range newSubnetInfos {
			subnetKeyValueList = append(subnetKeyValueList, subnetInfo.KeyValueList...)
		}
		// 필요에 따라 VPCInfo.KeyValueList와 합치기
		vpcInfo.KeyValueList = append(vpcInfo.KeyValueList, subnetKeyValueList...)
	}

	return vpcInfo, nil
}

func attachSubnet(vpc vpcv1.VPC, subnetInfo irs.SubnetInfo, vpcService *vpcv1.VpcV1, ctx context.Context) error {
	if subnetInfo.IPv4_CIDR == "" || subnetInfo.IId.NameId == "" {
		return errors.New("invalid subnetInfo")
	}
	exist, err := existSubnet(subnetInfo.IId, vpc, vpcService, ctx)
	if err != nil {
		return err
	} else if exist {
		err = errors.New(fmt.Sprintf("already exist %s", subnetInfo.IId.NameId))
		return err
	}

	options := &vpcv1.CreateSubnetOptions{}
	options.SetSubnetPrototype(&vpcv1.SubnetPrototype{
		Ipv4CIDRBlock: core.StringPtr(subnetInfo.IPv4_CIDR),
		Name:          core.StringPtr(subnetInfo.IId.NameId),
		VPC: &vpcv1.VPCIdentity{
			ID: vpc.ID,
		},
		Zone: &vpcv1.ZoneIdentity{
			Name: &subnetInfo.Zone,
		},
	})
	_, _, err = vpcService.CreateSubnetWithContext(ctx, options)
	return err
}
func getVPCNextHref(next *vpcv1.PageLink) (string, error) {
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
func getSubnetNextHref(next *vpcv1.PageLink) (string, error) {
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

func existSubnet(subnetIID irs.IID, vpc vpcv1.VPC, vpcService *vpcv1.VpcV1, ctx context.Context) (bool, error) {
	options := &vpcv1.ListSubnetsOptions{}
	subnets, _, err := vpcService.ListSubnetsWithContext(ctx, options)
	if err != nil {
		return false, err
	}
	for {
		if *subnets.TotalCount > 0 {
			for _, subnet := range subnets.Subnets {
				if *subnet.VPC.ID == *vpc.ID {
					if subnetIID.NameId == *subnet.Name || *subnet.ID == subnetIID.SystemId {
						return true, nil
					}
				}
			}
		}
		nextstr, _ := getSubnetNextHref(subnets.Next)
		if nextstr != "" {
			options := &vpcv1.ListSubnetsOptions{
				Start: core.StringPtr(nextstr),
			}
			subnets, _, err = vpcService.ListSubnetsWithContext(ctx, options)
			if err != nil {
				return false, errors.New("failed Get SubnetList")
			}
		} else {
			break
		}
	}
	return false, nil
}

func existVpc(vpcIID irs.IID, vpcService *vpcv1.VpcV1, ctx context.Context) (bool, error) {
	if vpcIID.NameId == "" {
		return false, errors.New("inValid Name")
	} else {
		listVpcsOptions := &vpcv1.ListVpcsOptions{}
		vpcs, _, err := vpcService.ListVpcsWithContext(ctx, listVpcsOptions)
		if err != nil {
			return false, err
		}
		for {
			if vpcs.Vpcs != nil {
				for _, vpc := range vpcs.Vpcs {
					if *vpc.Name == vpcIID.NameId {
						return true, nil
					}
				}
			}
			nextstr, _ := getVPCNextHref(vpcs.Next)
			if nextstr != "" {
				listVpcsOptions2 := &vpcv1.ListVpcsOptions{
					Start: core.StringPtr(nextstr),
				}
				vpcs, _, err = vpcService.ListVpcsWithContext(ctx, listVpcsOptions2)
				if err != nil {
					return false, errors.New("failed Get VPCList")
				}
			} else {
				break
			}
		}
		return false, nil
	}
}
func GetRawVPC(vpcIID irs.IID, vpcService *vpcv1.VpcV1, ctx context.Context) (vpcv1.VPC, error) {
	if vpcIID.SystemId == "" {
		listVpcsOptions := &vpcv1.ListVpcsOptions{}
		vpcs, _, err := vpcService.ListVpcsWithContext(ctx, listVpcsOptions)
		if err != nil {
			return vpcv1.VPC{}, err
		}
		for {
			if vpcs.Vpcs != nil {
				for _, vpc := range vpcs.Vpcs {
					if *vpc.Name == vpcIID.NameId {
						return vpc, nil
					}
				}
			}
			nextstr, _ := getVPCNextHref(vpcs.Next)
			if nextstr != "" {
				listVpcsOptions2 := &vpcv1.ListVpcsOptions{
					Start: core.StringPtr(nextstr),
				}
				vpcs, _, err = vpcService.ListVpcsWithContext(ctx, listVpcsOptions2)
				if err != nil {
					break
				}
			} else {
				break
			}
		}
		// NOT EXIST!
		if vpcIID.NameId != "" {
			err = errors.New(fmt.Sprintf("VPC not found %s", vpcIID.NameId))
		} else {
			err = errors.New("VPC not found")
		}
		return vpcv1.VPC{}, err
	} else {
		getVpcOptions := &vpcv1.GetVPCOptions{
			ID: &vpcIID.SystemId,
		}
		vpc, _, err := vpcService.GetVPCWithContext(ctx, getVpcOptions)
		if err != nil {
			return vpcv1.VPC{}, err
		}
		return *vpc, nil
	}
}

func getVPCRawSubnets(vpc vpcv1.VPC, vpcService *vpcv1.VpcV1, ctx context.Context) ([]vpcv1.Subnet, error) {
	options := &vpcv1.ListSubnetsOptions{}
	subnets, _, err := vpcService.ListSubnetsWithContext(ctx, options)
	if err != nil {
		return nil, err
	}
	var newSubnetInfos []vpcv1.Subnet
	for {
		if *subnets.TotalCount > 0 {
			for _, subnet := range subnets.Subnets {
				if *subnet.VPC.ID == *vpc.ID {
					newSubnetInfos = append(newSubnetInfos, subnet)
				}
			}
		}
		nextstr, _ := getSubnetNextHref(subnets.Next)
		if nextstr != "" {
			options := &vpcv1.ListSubnetsOptions{
				Start: core.StringPtr(nextstr),
			}
			subnets, _, err = vpcService.ListSubnetsWithContext(ctx, options)
			if err != nil {
				break
			}
		} else {
			break
		}
	}
	return newSubnetInfos, nil
}

func getRawSubnet(subnetIID irs.IID, vpcService *vpcv1.VpcV1, ctx context.Context) (vpcv1.Subnet, error) {
	options := &vpcv1.ListSubnetsOptions{}
	subnets, _, err := vpcService.ListSubnetsWithContext(ctx, options)
	if err != nil {
		return vpcv1.Subnet{}, err
	}
	var subnetFoundByName bool
	var foundedSubnetByName vpcv1.Subnet
	for {
		if *subnets.TotalCount > 0 {
			for _, subnet := range subnets.Subnets {
				if subnetIID.SystemId != "" && *subnet.ID == subnetIID.SystemId {
					return subnet, nil
				}
				if subnetIID.NameId == *subnet.Name {
					if subnetFoundByName {
						return vpcv1.Subnet{}, errors.New("found multiple subnets")
					}
					subnetFoundByName = true
					foundedSubnetByName = subnet
				}
			}
		}
		nextstr, _ := getSubnetNextHref(subnets.Next)
		if nextstr != "" {
			options := &vpcv1.ListSubnetsOptions{
				Start: core.StringPtr(nextstr),
			}
			subnets, _, err = vpcService.ListSubnetsWithContext(ctx, options)
			if err != nil {
				break
			}
		} else {
			break
		}
	}

	if subnetFoundByName {
		return foundedSubnetByName, nil
	}

	return vpcv1.Subnet{}, err
}

func getVPCRawSubnet(vpc vpcv1.VPC, subnetIID irs.IID, vpcService *vpcv1.VpcV1, ctx context.Context) (vpcv1.Subnet, error) {
	options := &vpcv1.ListSubnetsOptions{}
	subnets, _, err := vpcService.ListSubnetsWithContext(ctx, options)
	if err != nil {
		return vpcv1.Subnet{}, err
	}
	for {
		if *subnets.TotalCount > 0 {
			for _, subnet := range subnets.Subnets {
				if *subnet.VPC.ID == *vpc.ID {
					if subnetIID.NameId == *subnet.Name || *subnet.ID == subnetIID.SystemId {
						return subnet, nil
					}
				}
			}
		}
		nextstr, _ := getSubnetNextHref(subnets.Next)
		if nextstr != "" {
			options := &vpcv1.ListSubnetsOptions{
				Start: core.StringPtr(nextstr),
			}
			subnets, _, err = vpcService.ListSubnetsWithContext(ctx, options)
			if err != nil {
				break
			}
		} else {
			break
		}
	}
	return vpcv1.Subnet{}, err
}

func (vpcHandler *IbmVPCHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(vpcHandler.Region, call.VPCSUBNET, "VPC", "ListIID()")
	listVpcsOptions := &vpcv1.ListVpcsOptions{}

	var iidList []*irs.IID

	start := call.Start()
	vpcs, _, err := vpcHandler.VpcService.ListVpcsWithContext(vpcHandler.Ctx, listVpcsOptions)
	if err != nil {
		err = errors.New(fmt.Sprintf("Failed to List VPC err = %s", err.Error()))
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return make([]*irs.IID, 0), err

	}
	// Next Check
	for {
		for _, vpc := range vpcs.Vpcs {
			var iid irs.IID

			if vpc.ID != nil {
				iid.SystemId = *vpc.ID
			}
			if vpc.Name != nil {
				iid.NameId = *vpc.Name
			}

			iidList = append(iidList, &iid)
		}
		nextstr, _ := getVPCNextHref(vpcs.Next)
		if nextstr != "" {
			listVpcsOptions2 := &vpcv1.ListVpcsOptions{
				Start: core.StringPtr(nextstr),
			}
			vpcs, _, err = vpcHandler.VpcService.ListVpcsWithContext(vpcHandler.Ctx, listVpcsOptions2)
			if err != nil {
				err = errors.New(fmt.Sprintf("Failed to List VPC err = %s", err.Error()))
				cblogger.Error(err.Error())
				LoggingError(hiscallInfo, err)
				return make([]*irs.IID, 0), err
			}
		} else {
			break
		}
	}

	LoggingInfo(hiscallInfo, start)

	return iidList, nil
}

// cleanupSubnetPublicGateway detaches public gateway from subnet and cleans up unused gateway
func (vpcHandler *IbmVPCHandler) cleanupSubnetPublicGateway(subnetId string) error {
	// Check if subnet has a public gateway attached
	getSubnetPublicGatewayOptions := vpcHandler.VpcService.NewGetSubnetPublicGatewayOptions(subnetId)
	publicGateway, response, err := vpcHandler.VpcService.GetSubnetPublicGatewayWithContext(vpcHandler.Ctx, getSubnetPublicGatewayOptions)

	// If no public gateway is attached, nothing to clean up
	if err != nil {
		if response != nil && response.StatusCode == 404 {
			cblogger.Infof("Subnet %s has no public gateway attached", subnetId)
			return nil
		}
		cblogger.Errorf("Failed to check subnet public gateway: %v", err)
		return err
	}

	cblogger.Infof("Detaching public gateway %s from subnet %s", *publicGateway.ID, subnetId)

	// Detach the public gateway from the subnet
	unsetSubnetPublicGatewayOptions := vpcHandler.VpcService.NewUnsetSubnetPublicGatewayOptions(subnetId)
	_, err = vpcHandler.VpcService.UnsetSubnetPublicGatewayWithContext(vpcHandler.Ctx, unsetSubnetPublicGatewayOptions)
	if err != nil {
		cblogger.Errorf("Failed to detach public gateway from subnet: %v", err)
		return fmt.Errorf("failed to detach public gateway from subnet: %w", err)
	}

	cblogger.Infof("Successfully detached public gateway %s from subnet %s", *publicGateway.ID, subnetId)

	// Check if the public gateway is used by other subnets
	err = vpcHandler.cleanupUnusedPublicGateway(*publicGateway.ID, *publicGateway.VPC.ID)
	if err != nil {
		cblogger.Warnf("Failed to cleanup unused public gateway: %v", err)
		// Don't return error here as the main operation (detaching) succeeded
	}

	return nil
}

// cleanupUnusedPublicGateway deletes public gateway if it's not used by any subnets
func (vpcHandler *IbmVPCHandler) cleanupUnusedPublicGateway(publicGatewayId, vpcId string) error {
	// List all subnets in the VPC to check if any still use this public gateway
	listSubnetsOptions := vpcHandler.VpcService.NewListSubnetsOptions()
	subnets, _, err := vpcHandler.VpcService.ListSubnetsWithContext(vpcHandler.Ctx, listSubnetsOptions)
	if err != nil {
		return fmt.Errorf("failed to list subnets: %w", err)
	}

	// Check if any subnet in the same VPC is still using this public gateway
	for _, subnet := range subnets.Subnets {
		if subnet.VPC != nil && subnet.VPC.ID != nil && *subnet.VPC.ID == vpcId {
			getSubnetPublicGatewayOptions := vpcHandler.VpcService.NewGetSubnetPublicGatewayOptions(*subnet.ID)
			subnetPublicGateway, response, err := vpcHandler.VpcService.GetSubnetPublicGatewayWithContext(vpcHandler.Ctx, getSubnetPublicGatewayOptions)
			if err == nil && subnetPublicGateway.ID != nil && *subnetPublicGateway.ID == publicGatewayId {
				cblogger.Infof("Public gateway %s is still used by subnet %s, not deleting", publicGatewayId, *subnet.ID)
				return nil
			} else if err != nil && (response == nil || response.StatusCode != 404) {
				cblogger.Warnf("Failed to check subnet %s public gateway: %v", *subnet.ID, err)
			}
		}
	}

	// If no subnet is using this public gateway, delete it
	cblogger.Infof("Public gateway %s is no longer used, deleting it", publicGatewayId)
	deletePublicGatewayOptions := vpcHandler.VpcService.NewDeletePublicGatewayOptions(publicGatewayId)
	_, err = vpcHandler.VpcService.DeletePublicGatewayWithContext(vpcHandler.Ctx, deletePublicGatewayOptions)
	if err != nil {
		return fmt.Errorf("failed to delete unused public gateway: %w", err)
	}

	cblogger.Infof("Successfully deleted unused public gateway %s", publicGatewayId)
	return nil
}
