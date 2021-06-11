package resources

import (
	"errors"
	"fmt"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/softlayer/softlayer-go/datatypes"
	"github.com/softlayer/softlayer-go/filter"
	"github.com/softlayer/softlayer-go/services"
	"github.com/softlayer/softlayer-go/sl"
	"strconv"
	"strings"
)

type IbmVPCHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	AccountClient  *services.Account
	NetworkVlanClient *services.Network_Vlan
	ProductPackageClient * services.Product_Package
	ProductOrderClient *services.Product_Order
	NetworkSubnetClient *services.Network_Subnet
	BillingItemClient *services.Billing_Item
	LocationDatacenterClient *services.Location_Datacenter
}

func (vpcHandler *IbmVPCHandler) CreateVPC(vpcReqInfo irs.VPCReqInfo) (irs.VPCInfo, error){
	hiscallInfo := GetCallLogScheme(vpcHandler.Region, call.VPCSUBNET, "VPC", "CreateVPC()")
	var networkVlanPackage datatypes.Product_Package
	var locationId string
	vlanProductPackageFilter := filter.Path("keyName").Eq("NETWORK_VLAN").Build()
	locationFilter := filter.Path("name").Eq(vpcHandler.Region.Region).Build()
	vlanProductPackageMask := "mask[items,keyName,id]"
	networkVlanPackages, err := vpcHandler.ProductPackageClient.Filter(vlanProductPackageFilter).Mask(vlanProductPackageMask).GetAllObjects()
	if err != nil{
		LoggingError(hiscallInfo, err)
		return irs.VPCInfo{},err
	}
	if len(networkVlanPackages) < 1 {
		err = errors.New("not found NETWORK_VLAN Package")
		LoggingError(hiscallInfo, err)
		return irs.VPCInfo{},err
	}
	networkVlanPackage = networkVlanPackages[0]
	// 현재 PublicVlan만을 생성 interface 정책 결정시 그에 따른 Private Vlan 생성 로직 추가
	var publicVlanItemPrice datatypes.Product_Item_Price

	for _,item :=range networkVlanPackage.Items{
		if strings.Contains(*item.KeyName, "PUBLIC"){
			// publicVlanItem = item
			for _, price :=range item.Prices{
				if price.LocationGroupId == nil{
					publicVlanItemPrice = price
					break
				}
			}
		}
		//if strings.Contains(*item.KeyName, "PRIVATE"){
		//	// privateVlanItem = item
		//	for _, price :=range item.Prices{
		//		if price.LocationGroupId == nil{
		//			privateVlanItemPrice = price
		//			break
		//		}
		//	}
		//}
	}
	datacenters, err := vpcHandler.LocationDatacenterClient.Filter(locationFilter).GetDatacenters()
	if err != nil{
		LoggingError(hiscallInfo, err)
		return irs.VPCInfo{},err
	}
	if len(datacenters) < 1 {
		err = errors.New("not found Location Id")
		LoggingError(hiscallInfo, err)
		return irs.VPCInfo{},err
	}
	locationId =  strconv.Itoa(*datacenters[0].Id)
	orderTemplate := datatypes.Container_Product_Order_Network_Vlan {
		Container_Product_Order : datatypes.Container_Product_Order {
			Prices		   : []datatypes.Product_Item_Price {
				publicVlanItemPrice,     // Price for the new Public Network Vlan
			},
			PackageId	   : sl.Int(*networkVlanPackage.Id),
			Location	   : &locationId,
			Quantity	   : sl.Int(1),
		},
	}
	// TODO : for placeOrder
	// preVpcList, err := vpcHandler.ListVPC()
	if err != nil{
		LoggingError(hiscallInfo, err)
		return irs.VPCInfo{},err
	}
	// TODO : for placeOrder
	//start := call.Start()

	// TODO VerifyOrder => placeOrder
	_, err = vpcHandler.ProductOrderClient.VerifyOrder(&orderTemplate)
	if err != nil{
		LoggingError(hiscallInfo, err)
		return irs.VPCInfo{},err
	}
	//fmt.Println(result)
	err = errors.New("DryRun Test Success")
	return irs.VPCInfo{},err


	// TODO : for placeOrder
	//if err != nil{
	//	LoggingError(hiscallInfo, err)
	//	return irs.VPCInfo{},err
	//}
	//curRetryCnt := 0
	//maxRetryCnt := 120
	//for{
	//	checkVPCs, err := vpcHandler.ListVPC()
	//	if err == nil{
	//		if len(checkVPCs) != len(preVpcList){
	//			for _, vpc := range checkVPCs{
	//				newVpcFlag := true
	//				for _, preVPC :=range preVpcList {
	//					if vpc.IId.NameId == preVPC.IId.NameId && vpc.IId.NameId != ""{
	//						newVpcFlag = false
	//						break
	//					}
	//				}
	//				if newVpcFlag{
	//					LoggingInfo(hiscallInfo, start)
	//					return *vpc,nil
	//				}
	//			}
	//		}
	//	}
	//	curRetryCnt++
	//	time.Sleep(1 * time.Second)
	//	if curRetryCnt > maxRetryCnt {
	//		err = errors.New(fmt.Sprintf("failed to create VPC, exceeded maximum retry count %d", maxRetryCnt))
	//		LoggingError(hiscallInfo, err)
	//		return irs.VPCInfo{},err
	//	}
	//}
}

func (vpcHandler *IbmVPCHandler) ListVPC() ([]*irs.VPCInfo, error){
	hiscallInfo := GetCallLogScheme(vpcHandler.Region, call.VPCSUBNET, "VPC", "ListVPC()")

	var vlanInfos []*irs.VPCInfo
	vlanFilter := filter.Path("networkVlans.primaryRouter.datacenter.name").Eq(vpcHandler.Region.Region).Build()
	vlanMask:= "mask[primaryRouter[datacenter],id,name,subnets,subnetCount,networkSpace,vlanNumber]"
	start := call.Start()
	allVlans, err := vpcHandler.AccountClient.Mask(vlanMask).Filter(vlanFilter).GetNetworkVlans()
	if err != nil{
		LoggingError(hiscallInfo, err)
		return nil,err
	}
	privateVlanId, err :=vpcHandler.getPrivateVlan()
	if err != nil{
		LoggingError(hiscallInfo, err)
		return nil,err
	}
	for _, vlan := range allVlans{
		if *vlan.NetworkSpace == "PUBLIC" {
			var vlanInfo = irs.VPCInfo{}
			vpcHandler.setVlanIId(&vlanInfo,&vlan)
			vpcHandler.setVlanSubnets(&vlanInfo,&vlan)
			vpcHandler.setVlanKeyValues(&vlanInfo,privateVlanId)
			vlanInfos = append(vlanInfos,&vlanInfo)
		}
	}
	LoggingInfo(hiscallInfo, start)
	return vlanInfos,nil
}

func (vpcHandler *IbmVPCHandler) GetVPC(vpcIID irs.IID) (irs.VPCInfo, error){
	hiscallInfo := GetCallLogScheme(vpcHandler.Region, call.VPCSUBNET, vpcIID.NameId, "GetVPC()")

	var vlanInfo = irs.VPCInfo{}
	var vlan datatypes.Network_Vlan
	vlanMask:= "mask[primaryRouter[datacenter],id,name,subnets,subnetCount,networkSpace,vlanNumber]"
	start := call.Start()
	privateVlanId, err :=vpcHandler.getPrivateVlan()
	if err != nil{
		LoggingError(hiscallInfo, err)
		return vlanInfo,err
	}
	vlanId, err := strconv.Atoi(vpcIID.SystemId)
	if err != nil{
		vlan, err = vpcHandler.getVlanByName(vpcIID.NameId)
		if err != nil{
			LoggingError(hiscallInfo, err)
			return vlanInfo,err
		}
	} else {
		vlan, err = vpcHandler.NetworkVlanClient.Mask(vlanMask).Id(vlanId).GetObject()
		if err != nil{
			LoggingError(hiscallInfo, err)
			return vlanInfo,err
		}
	}
	vpcHandler.setVlanIId(&vlanInfo,&vlan)
	vpcHandler.setVlanSubnets(&vlanInfo,&vlan)
	vpcHandler.setVlanKeyValues(&vlanInfo,privateVlanId)
	// setIID, setSubNets, setKeyValue
	LoggingInfo(hiscallInfo, start)
	return vlanInfo,nil
}

func (vpcHandler *IbmVPCHandler) DeleteVPC(vpcIID irs.IID) (bool, error){
	hiscallInfo := GetCallLogScheme(vpcHandler.Region, call.VPCSUBNET, vpcIID.NameId, "DeleteVPC()")
	vpcId ,err := strconv.Atoi(vpcIID.SystemId)
	var vlan datatypes.Network_Vlan
	// exist Check
	// NameIdBy
	if err != nil {
		vlan, err = vpcHandler.getVlanByName(vpcIID.NameId)
		if err != nil {
			LoggingError(hiscallInfo, err)
			return false,err
		}
		// systemIdBy
	} else {
		vlan, err = vpcHandler.NetworkVlanClient.Id(vpcId).Mask("id").GetObject()
		if err != nil {
			LoggingError(hiscallInfo, err)
			return false,err
		}
	}
	if vlan.Id == nil{
		err = errors.New(fmt.Sprintf("Not exist VPC %s",vpcIID.NameId))
		LoggingError(hiscallInfo, err)
		return false,err
	}
	billingItem, err := vpcHandler.NetworkVlanClient.Id(*vlan.Id).GetBillingItem()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false,err
	}
	billingItemId := billingItem.Id
	if billingItemId == nil{
		err = errors.New(fmt.Sprintf("cannot delete Vlan %s , automatically assigned and removed by IBM",vpcIID.NameId))
		LoggingError(hiscallInfo, err)
		return false,err
	}
	start := call.Start()
	result, err := vpcHandler.BillingItemClient.Id(*billingItemId).CancelService()
	if err != nil {
		return false,err
	}
		LoggingInfo(hiscallInfo, start)
	return result,nil
	//err = errors.New(fmt.Sprintf("Protect VLan... for Test"))
	//return false, err
}

func (vpcHandler *IbmVPCHandler) AddSubnet(vpcIID irs.IID, subnetInfo irs.SubnetInfo) (irs.VPCInfo, error){
	hiscallInfo := GetCallLogScheme(vpcHandler.Region, call.VPCSUBNET, subnetInfo.IId.NameId, "AddSubnet()")

	//vlanMask:= "mask[primaryRouter[datacenter],id,name,subnets,subnetCount,networkSpace,vlanNumber]"
	//subnetIpItemMask := "mask[items,itemCount]"
	//portableIpAddressesProductFilter := filter.Path("keyName").Eq(potableSubnetPackageKeyName).Build()
	//vlanId, err := strconv.Atoi(vpcIID.SystemId)
	//var vlan datatypes.Network_Vlan
	//if err != nil {
	//	vlan, err = vpcHandler.getVlanByName(vpcIID.NameId)
	//	if err != nil {
	//		return irs.VPCInfo{},err
	//	}
	//	// systemIdBy
	//} else {
	//	vlan, err = vpcHandler.NetworkVlanClient.Id(vlanId).Mask(vlanMask).GetObject()
	//	if err != nil {
	//		return irs.VPCInfo{},err
	//	}
	//}
	//vlanId = *vlan.Id
	//// get ProductPackage PORTABLE_IP_ADDRESSES
	//portableIpAddressesProducts, err := vpcHandler.ProductPackageClient.Filter(portableIpAddressesProductFilter).Mask(subnetIpItemMask).GetAllObjects()
	//if err != nil{
	//	return irs.VPCInfo{}, err
	//}
	//if len(portableIpAddressesProducts) < 1 {
	//	return irs.VPCInfo{},errors.New("not Found PORTABLE_IP_ADDRESSES Product")
	//}
	//potableIPProduct := portableIpAddressesProducts[0]
	//var subnetProductItemIP4vlansSpace []datatypes.Product_Item
	//for _, item := range potableIPProduct.Items{
	//	if item.Units != nil && *item.Units == "IPs"{ // IPs => IPv4 nil => IPv6
	//		// NetworkSpace : PUBLIC / PRIVATE item select
	//		if strings.Contains(*item.KeyName,*vlan.NetworkSpace){
	//			subnetProductItemIP4vlansSpace = append(subnetProductItemIP4vlansSpace,item)
	//		}
	//	}
	//}
	//cidrSplits :=strings.Split(subnetInfo.IPv4_CIDR,"/")
	//if len(cidrSplits) != 2 {
	//	return irs.VPCInfo{},errors.New(fmt.Sprintf("%s invalid IPv4_CIDR",subnetInfo.IPv4_CIDR))
	//}
	//netMask,err := strconv.Atoi(cidrSplits[1])
	//if err != nil{
	//	return irs.VPCInfo{},errors.New(fmt.Sprintf("%s invalid IPv4_CIDR",subnetInfo.IPv4_CIDR))
	//}
	//// IBM Subnet Product Item Order restrict
	//if netMask < 24 || netMask > 30 {
	//	return irs.VPCInfo{},errors.New("only cidr 24~30")
	//}
	//addressCapacity := int(math.Pow(float64(2),float64(int(32-netMask))))
	//var subnetItem datatypes.Product_Item
	//// get Product Item with addressCapacity
	//for _, item := range subnetProductItemIP4vlansSpace{
	//	if int(*item.Capacity) == addressCapacity{
	//		subnetItem = item
	//		break
	//	}
	//}
	//prices := subnetItem.Prices
	//location := *vlan.PrimaryRouter.Datacenter.LongName
	//
	//if subnetItem.Id == nil {
	//	return irs.VPCInfo{},errors.New("unavail Subnet")
	//}
	//
	//orderTemplate := datatypes.Container_Product_Order_Network_Subnet {
	//	Container_Product_Order : datatypes.Container_Product_Order {
	//		PackageId : sl.Int(*potableIPProduct.Id), // 281
	//		Location  : sl.String(location),
	//		Quantity  : sl.Int(1),
	//		Prices    : prices,
	//	},
	//
	//	EndPointVlanId : sl.Int(vlanId),
	//}
	//save := false
	//preVPC, err := vpcHandler.GetVPC(vpcIID)
	//_, err = vpcHandler.ProductOrderClient.PlaceOrder(&orderTemplate,&save)
	//if err != nil{
	//	return irs.VPCInfo{},err
	//}
	//curRetryCnt := 0
	//maxRetryCnt := 120
	//for{
	//	fmt.Println(curRetryCnt)
	//	checkVPC, err := vpcHandler.GetVPC(vpcIID)
	//	if err != nil {
	//		return irs.VPCInfo{}, err
	//	}
	//	for _, checkSubnet := range  checkVPC.SubnetInfoList{
	//		newVpcFlag := true
	//		for _, presubnet := range preVPC.SubnetInfoList{
	//			if checkSubnet.IId.NameId == presubnet.IId.NameId && presubnet.IId.NameId != ""{
	//				newVpcFlag = false
	//			}
	//		}
	//		if newVpcFlag{
	//			return checkVPC,nil
	//		}
	//	}
	//	curRetryCnt++
	//	time.Sleep(1 * time.Second)
	//	if curRetryCnt > maxRetryCnt {
	//		return irs.VPCInfo{}, errors.New(fmt.Sprintf("failed to add Subnet, exceeded maximum retry count %d", maxRetryCnt))
	//	}
	//}
	err := errors.New(fmt.Sprintf("Ibm Not Provide PRIMARY Subnet Add, automatically assigned and removed by IBM"))
	LoggingError(hiscallInfo, err)
	return  irs.VPCInfo{}, err
}

func (vpcHandler *IbmVPCHandler) RemoveSubnet(vpcIID irs.IID, subnetIID irs.IID) (bool, error){
	hiscallInfo := GetCallLogScheme(vpcHandler.Region, call.VPCSUBNET, subnetIID.NameId, "AddSubnet()")

	//subnetId ,err := strconv.Atoi(subnetIID.SystemId)
	//if err != nil {
	//	return false,err
	//}
	//billingItem, err := vpcHandler.NetworkSubnetClient.Id(subnetId).GetBillingItem()
	//if err != nil {
	//	return false,err
	//}
	//billingItemId := billingItem.Id
	//if billingItemId == nil{
	//	return false,errors.New("Not Exist Subnet Billing")
	//}
	//result, err := vpcHandler.BillingItemClient.Id(*billingItemId).CancelService()
	//if err != nil {
	//	return false,err
	//}
	//return result,nil
	err := errors.New(fmt.Sprintf("Ibm Not Provide PRIMARY Subnet Remove, automatically assigned and removed by IBM"))
	LoggingError(hiscallInfo, err)
	return  false, err
}

func (vpcHandler *IbmVPCHandler) getPrivateVlan() (string,error){
	vlanFilter := filter.Path("networkVlans.primaryRouter.datacenter.name").Eq(vpcHandler.Region.Region).Build()
	vlanMask:= "mask[primaryRouter[datacenter],id,name,subnets,subnetCount,networkSpace,vlanNumber]"
	vlans, err :=vpcHandler.AccountClient.Mask(vlanMask).Filter(vlanFilter).GetNetworkVlans()
	if err != nil{
		return "",err
	}
	for _, vlan := range vlans{
		if *vlan.NetworkSpace == "PRIVATE"{
			return strconv.Itoa(*vlan.Id),nil
		}
	}
	return "",errors.New("not Exist Private Vlan")
}

func (vpcHandler *IbmVPCHandler) existCheckVlan(vlanName string) error {
	// Vlan Name = *vlan.PrimaryRouter.Hostname +"."+strconv.Itoa(*vlan.VlanNumber)
	mask := "mask[primaryRouter[hostname,datacenter],vlanNumber]"
	vlanFilter := filter.Path("networkVlans.primaryRouter.datacenter.name").Eq(vpcHandler.Region.Region).Build()
	vlans, err:=vpcHandler.AccountClient.Mask(mask).Filter(vlanFilter).GetNetworkVlans()
	if err != nil {
		return err
	}
	if len(vlans) > 0 {
		for _, vlan := range vlans {
			name := *vlan.PrimaryRouter.Hostname +"."+strconv.Itoa(*vlan.VlanNumber)
			if vlanName == name {
				return errors.New(fmt.Sprintf("VPC with name %s already exist", vlanName))
			}
		}
	}
	return nil
}

func (vpcHandler *IbmVPCHandler) getVlanByName(vlanName string) (datatypes.Network_Vlan, error) {
	if vlanName != ""{
		vlanFilter := filter.Path("networkVlans.primaryRouter.datacenter.name").Eq(vpcHandler.Region.Region).Build()
		vlanMask:= "mask[primaryRouter[datacenter],id,name,subnets,subnetCount,networkSpace,vlanNumber]"
		vlans, err:=vpcHandler.AccountClient.Mask(vlanMask).Filter(vlanFilter).GetNetworkVlans()
		if err != nil {
			return datatypes.Network_Vlan{},err
		}
		if len(vlans) > 0 {
			for _, vlan := range vlans {
				name := *vlan.PrimaryRouter.Hostname +"."+strconv.Itoa(*vlan.VlanNumber)
				if vlanName == name {
					return vlan, nil
				}
			}
		}
		return datatypes.Network_Vlan{},errors.New(fmt.Sprintf("VPC with name %s Not exist", vlanName))
	}
	return datatypes.Network_Vlan{},errors.New(fmt.Sprintf("VPC name invalid"))
}

func (vpcHandler *IbmVPCHandler) setVlanIId(vpcInfo *irs.VPCInfo, vlan *datatypes.Network_Vlan){
	var vlanIId = irs.IID{}
	defer func() {
		recover()
		vpcInfo.IId = vlanIId
	}()
	vlanIId.SystemId =  strconv.Itoa(*vlan.Id)
	vlanIId.NameId = *vlan.PrimaryRouter.Hostname +"."+strconv.Itoa(*vlan.VlanNumber)
}

func (vpcHandler *IbmVPCHandler) setVlanKeyValues(vpcInfo *irs.VPCInfo, privateVlanId string)  {
	var vlanKeyValueList []irs.KeyValue
	defer func() {
		recover()
		vpcInfo.KeyValueList = vlanKeyValueList
	}()
	//vlanNetworkSpace := irs.KeyValue{
	//	Key: "NetworkSpace",
	//	Value: *vlan.NetworkSpace,
	//}
	//vlanKeyValueList = append(vlanKeyValueList,vlanNetworkSpace)
	if privateVlanId != ""{
		privateVlan := irs.KeyValue{
			Key: "PrivateVlanId",
			Value: privateVlanId,
		}
		vlanKeyValueList = append(vlanKeyValueList, privateVlan)
	}
}

func (vpcHandler *IbmVPCHandler) setVlanSubnets(vpcInfo *irs.VPCInfo, vlan *datatypes.Network_Vlan){
	var vlanSubnetInfoList []irs.SubnetInfo
	defer func() {
		recover()
		vpcInfo.SubnetInfoList = vlanSubnetInfoList
	}()
	if vlan.SubnetCount != nil && *vlan.SubnetCount > 0{
		for _, subnet := range vlan.Subnets {
			if strings.Contains(*subnet.SubnetType,"PRIMARY"){
				cidr := *subnet.NetworkIdentifier +"/"+ strconv.Itoa(*subnet.Cidr)
				subnetInfo := irs.SubnetInfo{
					IId: irs.IID{
						SystemId: strconv.Itoa(*subnet.Id),
						NameId: cidr,//???
					},
					IPv4_CIDR: cidr,
				}
				vlanSubnetInfoList= append(vlanSubnetInfoList, subnetInfo)
			}
		}
	}
}

