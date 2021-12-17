package resources

import (
	"context"
	"errors"
	"fmt"
	"github.com/IBM/go-sdk-core/v4/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"net/url"
	"time"
)

type IbmVPCHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	VpcService     *vpcv1.VpcV1
	Ctx            context.Context
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
		LoggingError(hiscallInfo, err)
		return irs.VPCInfo{}, err
	}

	if vpcHandler.Region.Zone == "" {
		err = errors.New("not exist Zone")
		LoggingError(hiscallInfo, err)
		return irs.VPCInfo{}, err
	}

	exist, err := existVpc(vpcReqInfo.IId, vpcHandler.VpcService, vpcHandler.Ctx)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.VPCInfo{}, err
	}
	if exist {
		err = errors.New(fmt.Sprintf("already exist VPC %s", vpcReqInfo.IId.NameId))
		LoggingError(hiscallInfo, err)
		return irs.VPCInfo{}, err
	}

	// Create VPC
	createVPCOptions := &vpcv1.CreateVPCOptions{}
	createVPCOptions.SetAddressPrefixManagement("manual")
	createVPCOptions.SetName(vpcReqInfo.IId.NameId)
	vpc, _, err := vpcHandler.VpcService.CreateVPCWithContext(vpcHandler.Ctx, createVPCOptions)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.VPCInfo{}, err
	}
	newVpcIId := irs.IID{
		NameId:   *vpc.Name,
		SystemId: *vpc.ID,
	}
	// createVPCAddressPrefix
	createVPCAddressPrefixOptions := &vpcv1.CreateVPCAddressPrefixOptions{}
	createVPCAddressPrefixOptions.SetVPCID(newVpcIId.SystemId)
	createVPCAddressPrefixOptions.SetCIDR(vpcReqInfo.IPv4_CIDR)
	createVPCAddressPrefixOptions.SetName(newVpcIId.NameId)
	createVPCAddressPrefixOptions.SetZone(&vpcv1.ZoneIdentity{
		Name: core.StringPtr(vpcHandler.Region.Zone),
	})
	_, _, err = vpcHandler.VpcService.CreateVPCAddressPrefixWithContext(vpcHandler.Ctx, createVPCAddressPrefixOptions)
	// createVPCAddressPrefix error
	if err != nil {
		LoggingError(hiscallInfo, err)
		_, delErr := vpcHandler.DeleteVPC(newVpcIId)
		if delErr != nil {
			err = errors.New(err.Error() + delErr.Error())
		}
		return irs.VPCInfo{}, err
	}

	if vpcReqInfo.SubnetInfoList != nil && len(vpcReqInfo.SubnetInfoList) > 0 {
		for _, subnetInfo := range vpcReqInfo.SubnetInfoList {
			err = attachSubnet(*vpc, subnetInfo, vpcHandler.VpcService, vpcHandler.Ctx)
			if err != nil {
				_, delErr := vpcHandler.DeleteVPC(newVpcIId)
				if delErr != nil {
					err = errors.New(err.Error() + delErr.Error())
				}
				LoggingError(hiscallInfo, err)
				return irs.VPCInfo{}, err
			}
		}
	}
	vpcInfo, err := setVPCInfo(*vpc, vpcHandler.VpcService, vpcHandler.Ctx)
	if err != nil {
		_, delErr := vpcHandler.DeleteVPC(newVpcIId)
		if delErr != nil {
			err = errors.New(err.Error() + delErr.Error())
		}
		LoggingError(hiscallInfo, err)
		return irs.VPCInfo{}, err
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
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	// Next Check
	for {
		for _, vpc := range vpcs.Vpcs {
			vpcInfo, err := setVPCInfo(vpc, vpcHandler.VpcService, vpcHandler.Ctx)
			if err != nil {
				LoggingError(hiscallInfo, err)
				return nil, err
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
				LoggingError(hiscallInfo, err)
				return nil, err
				//break
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
	vpc, err := getRawVPC(vpcIID, vpcHandler.VpcService, vpcHandler.Ctx)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.VPCInfo{}, err
	}
	vpcInfo, err := setVPCInfo(vpc, vpcHandler.VpcService, vpcHandler.Ctx)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.VPCInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)
	return vpcInfo, nil
}
func (vpcHandler *IbmVPCHandler) DeleteVPC(vpcIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(vpcHandler.Region, call.VPCSUBNET, vpcIID.NameId, "DeleteVPC()")
	start := call.Start()
	vpc, err := getRawVPC(vpcIID, vpcHandler.VpcService, vpcHandler.Ctx)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	// Remove all Subnet
	rawSubnets, err := getVPCRawSubnets(vpc, vpcHandler.VpcService, vpcHandler.Ctx)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}
	if rawSubnets != nil {
		for _, subnet := range rawSubnets {
			options := &vpcv1.DeleteSubnetOptions{}
			options.SetID(*subnet.ID)
			_, err := vpcHandler.VpcService.DeleteSubnetWithContext(vpcHandler.Ctx, options)
			if err != nil {
				LoggingError(hiscallInfo, err)
				return false, err
			}
		}
	}
	// subnets delete Time delay
	curRetryCnt := 0
	maxRetryCnt := 60
	for {
		rawDeleteSubnets, err := getVPCRawSubnets(vpc, vpcHandler.VpcService, vpcHandler.Ctx)
		if err != nil {
			LoggingError(hiscallInfo, err)
			return false, err
		}
		if rawDeleteSubnets == nil {
			break
		}
		curRetryCnt++
		time.Sleep(1 * time.Second)
		if curRetryCnt > maxRetryCnt {
			err = errors.New("failed delete VPC - subnets delete TimeOut")
			LoggingError(hiscallInfo, err)
			return false, err
		}
	}
	// Delete VPC
	deleteVpcOptions := &vpcv1.DeleteVPCOptions{}
	deleteVpcOptions.SetID(*vpc.ID)
	_, err = vpcHandler.VpcService.DeleteVPCWithContext(vpcHandler.Ctx, deleteVpcOptions)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}
func (vpcHandler *IbmVPCHandler) AddSubnet(vpcIID irs.IID, subnetInfo irs.SubnetInfo) (irs.VPCInfo, error) {
	hiscallInfo := GetCallLogScheme(vpcHandler.Region, call.VPCSUBNET, subnetInfo.IId.NameId, "AddSubnet()")
	start := call.Start()
	vpc, err := getRawVPC(vpcIID, vpcHandler.VpcService, vpcHandler.Ctx)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.VPCInfo{}, err
	}
	err = attachSubnet(vpc, subnetInfo, vpcHandler.VpcService, vpcHandler.Ctx)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.VPCInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)
	vpc, err = getRawVPC(vpcIID, vpcHandler.VpcService, vpcHandler.Ctx)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.VPCInfo{}, err
	}
	vpcInfo, err := setVPCInfo(vpc, vpcHandler.VpcService, vpcHandler.Ctx)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.VPCInfo{}, err
	}
	return vpcInfo, nil
}
func (vpcHandler *IbmVPCHandler) RemoveSubnet(vpcIID irs.IID, subnetIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(vpcHandler.Region, call.VPCSUBNET, subnetIID.NameId, "RemoveSubnet()")
	start := call.Start()
	if subnetIID.SystemId != "" {
		options := &vpcv1.DeleteSubnetOptions{}
		options.SetID(subnetIID.SystemId)
		_, err := vpcHandler.VpcService.DeleteSubnetWithContext(vpcHandler.Ctx, options)
		if err != nil {
			LoggingError(hiscallInfo, err)
			return false, err
		}
		LoggingInfo(hiscallInfo, start)
		return true, nil
	}
	vpc, err := getRawVPC(vpcIID, vpcHandler.VpcService, vpcHandler.Ctx)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}
	rawSubnets, err := getVPCRawSubnets(vpc, vpcHandler.VpcService, vpcHandler.Ctx)
	if len(rawSubnets) > 0 {
		for _, subnet := range rawSubnets {
			if *subnet.Name == subnetIID.NameId {
				options := &vpcv1.DeleteSubnetOptions{}
				options.SetID(*subnet.ID)
				_, err := vpcHandler.VpcService.DeleteSubnetWithContext(vpcHandler.Ctx, options)
				if err != nil {
					LoggingError(hiscallInfo, err)
					return false, err
				}
				LoggingInfo(hiscallInfo, start)
				return true, nil
			}
		}
	}
	err = errors.New("not found subnet")
	LoggingError(hiscallInfo, err)
	return false, err

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
	zone, err := getFirstVPCZone(vpc, vpcService, ctx)
	if err != nil {
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
			Name: &zone,
		},
	})
	_, _, err = vpcService.CreateSubnetWithContext(ctx, options)
	return err
}
func getVPCNextHref(next *vpcv1.VPCCollectionNext) (string, error) {
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
func getSubnetNextHref(next *vpcv1.SubnetCollectionNext) (string, error) {
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
func getRawVPC(vpcIID irs.IID, vpcService *vpcv1.VpcV1, ctx context.Context) (vpcv1.VPC, error) {
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

func getFirstVPCZone(vpc vpcv1.VPC, vpcService *vpcv1.VpcV1, ctx context.Context) (string, error) {
	// get first AddressPrefix Zone
	listVpcAddressPrefixesOptions := &vpcv1.ListVPCAddressPrefixesOptions{}
	listVpcAddressPrefixesOptions.SetVPCID(*vpc.ID)
	addressPrefixes, _, err := vpcService.ListVPCAddressPrefixesWithContext(ctx, listVpcAddressPrefixesOptions)
	if err != nil {
		return "", err
	}
	if *addressPrefixes.TotalCount > 0 {
		return *addressPrefixes.AddressPrefixes[0].Zone.Name, nil
	}
	err = errors.New("VPC not found FirstVPCZone")
	return "", err
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

func setVPCInfo(vpc vpcv1.VPC, vpcService *vpcv1.VpcV1, ctx context.Context) (irs.VPCInfo, error) {
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
				IPv4_CIDR: *subnet.Ipv4CIDRBlock,
			}
			newSubnetInfos = append(newSubnetInfos, subnetInfo)
		}
		vpcInfo.SubnetInfoList = newSubnetInfos
	}
	return vpcInfo, nil
}
