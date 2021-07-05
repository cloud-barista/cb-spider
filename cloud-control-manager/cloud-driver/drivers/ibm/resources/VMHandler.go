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
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

type IbmVMHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	AccountClient  *services.Account
	VirtualGuestClient 		   *services.Virtual_Guest
	ProductPackageClient *services.Product_Package
	LocationDatacenterClient *services.Location_Datacenter
	ProductOrderClient *services.Product_Order
	SecuritySshKeyClient *services.Security_Ssh_Key
}

type IbmInfoParameter struct {
	Domain 				string
	Hostname        	string
	SshKeys             []int
	Flavor             	string
	SecurityGroupIIDs  	[]int
	Image 				string
}

func setterVmIfo(vmInfo *irs.VMInfo,virtualServer *datatypes.Virtual_Guest){
	setIId(vmInfo,virtualServer)
	setImageIId(vmInfo,virtualServer)
	setStartTime(vmInfo,virtualServer)
	setRegion(vmInfo,virtualServer)
	setVMSpecName(vmInfo,virtualServer)
	setSecurityGroup(vmInfo,virtualServer)
	setNetwork(vmInfo,virtualServer)
	setUserInfo(vmInfo,virtualServer)
	setKeyPair(vmInfo,virtualServer)
}

func(vmHandler *IbmVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (irs.VMInfo, error){
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, vmReqInfo.IId.NameId, "StartVM()")

	err := checkVmReqInfo(vmReqInfo)
	if err != nil{
		LoggingError(hiscallInfo, err)
		return irs.VMInfo{}, err
	}

	exist,err :=vmHandler.existCheckVMByName(vmReqInfo.IId.NameId)
	if exist{
		if err == nil{
			err = errors.New(fmt.Sprintf("VM with name %s already exist",vmReqInfo.IId.NameId))
		}
		LoggingError(hiscallInfo, err)
		return irs.VMInfo{}, err
	}
	var productId int
	var locationId string
	var presetId int
	var param IbmInfoParameter

	account, err := vmHandler.AccountClient.Mask("mask[firstName,lastName]").GetObject()
	if err != nil{
		LoggingError(hiscallInfo, err)
		return irs.VMInfo{}, err
	}
	if vmReqInfo.ImageIID.SystemId != "" && vmReqInfo.ImageIID.NameId == ""{
		imageIId,err := vmHandler.getterImageIIDBySystemId(vmReqInfo.ImageIID.SystemId)
		if err !=nil{
			LoggingError(hiscallInfo, err)
			return irs.VMInfo{}, err
		}
		vmReqInfo.ImageIID = imageIId
	}
	if vmReqInfo.KeyPairIID.NameId != "" && vmReqInfo.KeyPairIID.SystemId == ""{
		keyIId,err := vmHandler.getterKeyPairIIDByName(vmReqInfo.KeyPairIID.NameId)
		if err !=nil{
			LoggingError(hiscallInfo, err)
			return irs.VMInfo{}, err
		}
		vmReqInfo.KeyPairIID = keyIId
	}
	if vmReqInfo.VpcIID.NameId != "" && vmReqInfo.VpcIID.SystemId == ""{
		vpcIId,err := vmHandler.getterVpcIIDByName(vmReqInfo.VpcIID.NameId)
		if err !=nil{
			LoggingError(hiscallInfo, err)
			return irs.VMInfo{}, err
		}
		vmReqInfo.VpcIID = vpcIId
	}
	if vmReqInfo.SubnetIID.NameId != "" && vmReqInfo.SubnetIID.SystemId == "" {
		subnetIId, err := vmHandler.getterSubnetIIDByName(vmReqInfo.SubnetIID.NameId,vmReqInfo.VpcIID.SystemId)
		if err !=nil{
			LoggingError(hiscallInfo, err)
			return irs.VMInfo{}, err
		}
		vmReqInfo.SubnetIID = subnetIId
	}
	if len(vmReqInfo.SecurityGroupIIDs) > 0 {
		systemIdFlag := true
		for _, sgiid := range vmReqInfo.SecurityGroupIIDs{
			if sgiid.SystemId == ""{
				systemIdFlag = false
			}
		}
		if !systemIdFlag {
			sgiids ,err := vmHandler.getterSecurityGroupIIDs(vmReqInfo.SecurityGroupIIDs)
			if err != nil{
				LoggingError(hiscallInfo, err)
				return irs.VMInfo{}, err
			}
			vmReqInfo.SecurityGroupIIDs = sgiids
		}
	}
	err = setStartVMParameter(&vmReqInfo,&param,&account)
	if err != nil{
		LoggingError(hiscallInfo, err)
		return irs.VMInfo{}, err
	}

	// baseOption
	baseSettingItems := []string{
		"BANDWIDTH_0_GB_2",
		"1_IP_ADDRESS",
		"REBOOT_REMOTE_CONSOLE",
		"MONITORING_HOST_PING",
		"NOTIFICATION_EMAIL_AND_TICKET",
		"AUTOMATED_NOTIFICATION",
		"UNLIMITED_SSL_VPN_USERS_1_PPTP_VPN_USER_PER_ACCOUNT",
	}
	productFilter := filter.Path("keyName").Eq(productName).Build()
	locationFilter := filter.Path("name").Eq(vmHandler.Region.Region).Build()
	itemMask := "mask[id,keyName,itemCategory[categoryCode],capacity,units,prices[currentPriceFlag,locationGroupId,laborFee,id]]"

	products, err := vmHandler.ProductPackageClient.Filter(productFilter).Mask("id").GetAllObjects()
	if err != nil{
		LoggingError(hiscallInfo, err)
		return irs.VMInfo{}, err
	}
	if products == nil || len(products) == 0 {
		err = errors.New(fmt.Sprintf("Not Exist %s Product Service",productName))
		LoggingError(hiscallInfo, err)
		return irs.VMInfo{}, errors.New(fmt.Sprintf("Not Exist %s Product Service",productName))
	}
	productId = *products[0].Id

	datacenters, err := vmHandler.LocationDatacenterClient.Filter(locationFilter).GetDatacenters()
	if err != nil{
		LoggingError(hiscallInfo, err)
		return irs.VMInfo{}, err
	}
	if  datacenters == nil || len(datacenters) == 0 {
		err = errors.New(fmt.Sprintf("Not Exist %s Product Service",productName))
		LoggingError(hiscallInfo, err)
		return irs.VMInfo{}, err
	}
	locationId =  strconv.Itoa(*datacenters[0].Id)

	allItems, err := vmHandler.ProductPackageClient.Id(productId).Mask(itemMask).GetItems()
	if err != nil{
		LoggingError(hiscallInfo, err)
		return irs.VMInfo{}, err
	}
	allPresets, err :=  vmHandler.ProductPackageClient.Id(productId).GetActivePresets()
	if err != nil{
		LoggingError(hiscallInfo, err)
		return irs.VMInfo{}, err
	}
	var priceItems []datatypes.Product_Item_Price
	networkItemSetFlag := false
	imageItemSetFlag := false
	for _, getItem :=range allItems{
		itemKeyName := *getItem.KeyName
		if *getItem.ItemCategory.CategoryCode == "port_speed" && strings.Contains(itemKeyName,"PUBLIC") && !networkItemSetFlag {
			if *getItem.Units == "Mbps" && *getItem.Capacity == 100 {
				for _, price :=range getItem.Prices{
					if price.LocationGroupId ==nil{
						priceItems = append(priceItems,price)
						break
					}
				}
				networkItemSetFlag = true
				continue
			}
		}
		if itemKeyName == param.Image && !imageItemSetFlag{
			for _, price :=range getItem.Prices{
				if price.LocationGroupId ==nil{
					priceItems = append(priceItems,price)
					break
				}
			}
			imageItemSetFlag = true
			continue
		}
		if networkItemSetFlag && imageItemSetFlag{
			break
		}
	}
	for _, baseSettingItem :=range baseSettingItems{
		for _, getItem :=range allItems{
			itemKeyName := *getItem.KeyName
			if itemKeyName == baseSettingItem{
				for _, price :=range getItem.Prices{
					if price.LocationGroupId ==nil{
						priceItems = append(priceItems,price)
						break
					}
				}
				break
			}
		}
	}
	for _, preset := range allPresets{
		if *preset.KeyName == param.Flavor{
			presetId = *preset.Id
		}
	}
	virtualGuests := []datatypes.Virtual_Guest{
		{
			Hostname: sl.String(param.Hostname),
			Domain:   sl.String(param.Domain),
		},
	}
	var securityGroupBindings []datatypes.Virtual_Network_SecurityGroup_NetworkComponentBinding
	if param.SecurityGroupIIDs != nil && len(param.SecurityGroupIIDs) > 0 {
		for _, securityGroupId :=range param.SecurityGroupIIDs{
			var binding datatypes.Virtual_Network_SecurityGroup_NetworkComponentBinding
			sg := datatypes.Network_SecurityGroup{Id: sl.Int(securityGroupId)}
			binding.SecurityGroup = &sg
			securityGroupBindings = append(securityGroupBindings,binding)
		}
	}
	privateVlanIdString, err := vmHandler.getPrivateVlan()
	privateVlanId, err2 := strconv.Atoi(privateVlanIdString)
	if err != nil || err2 != nil{
		//private vlan 못찾음 => nil
		virtualGuests[0].PrimaryBackendNetworkComponent = &datatypes.Virtual_Guest_Network_Component{
			NetworkVlan: &datatypes.Network_Vlan{
				Id: nil,
			},
			SecurityGroupBindings: securityGroupBindings,
		}
	} else {
		virtualGuests[0].PrimaryBackendNetworkComponent = &datatypes.Virtual_Guest_Network_Component{
			NetworkVlan: &datatypes.Network_Vlan{
				Id: sl.Int(privateVlanId),
				//PrimarySubnet: &datatypes.Network_Subnet{
				//	Id: nil,
				//},
			},
			SecurityGroupBindings: securityGroupBindings,
		}
	}
	publicVlanId, err := strconv.Atoi(vmReqInfo.VpcIID.SystemId)
	if err != nil{
		virtualGuests[0].PrimaryNetworkComponent = &datatypes.Virtual_Guest_Network_Component{
			NetworkVlan: &datatypes.Network_Vlan{
				Id: nil,
				PrimarySubnet: &datatypes.Network_Subnet{
					Id: nil,
				},
			},
			SecurityGroupBindings: securityGroupBindings,
		}
	} else {
		publicVlanSubnetId, err2 := strconv.Atoi(vmReqInfo.SubnetIID.SystemId)
		if err2 != nil{
			virtualGuests[0].PrimaryNetworkComponent = &datatypes.Virtual_Guest_Network_Component{
				NetworkVlan: &datatypes.Network_Vlan{
					Id: sl.Int(publicVlanId),
					PrimarySubnet: &datatypes.Network_Subnet{
						Id: nil,
					},
				},
				SecurityGroupBindings: securityGroupBindings,
			}
		}else{
			var subnetId *int
			subnetFilter := filter.Path("subnets.id").Eq(publicVlanSubnetId).Build()
			subnets ,err := vmHandler.AccountClient.Filter(subnetFilter).Mask("mask[virtualGuestCount,usableIpAddressCount,id]").GetSubnets()
			if err == nil && len(subnets) > 0 {
				count := int(*subnets[0].UsableIpAddressCount) - int(*subnets[0].VirtualGuestCount)
				if count > 0{
					subnetId = sl.Int(publicVlanSubnetId)
				}
			}
			virtualGuests[0].PrimaryNetworkComponent = &datatypes.Virtual_Guest_Network_Component{
				NetworkVlan: &datatypes.Network_Vlan{
					Id: sl.Int(publicVlanId),
					PrimarySubnet: &datatypes.Network_Subnet{
						Id: subnetId,
					},
				},
				SecurityGroupBindings: securityGroupBindings,
			}
		}
	}

	containerOrder := datatypes.Container_Product_Order{
		PackageId:         sl.Int(productId),
		Location:         &locationId,
		VirtualGuests:    virtualGuests,
		Prices:           priceItems,
		UseHourlyPricing: sl.Bool(true),
		PresetId: sl.Int(presetId),
	}
	if len(param.SshKeys) > 0 {
		containerOrder.SshKeys = []datatypes.Container_Product_Order_SshKeys{
			{SshKeyIds: param.SshKeys},
		}
	}
	orderTemplate := datatypes.Container_Product_Order_Virtual_Guest{
		Container_Product_Order_Hardware_Server: datatypes.Container_Product_Order_Hardware_Server{
			Container_Product_Order: containerOrder,
		},
	}

	rootPath := os.Getenv("CBSPIDER_ROOT")
	fileDataCloudInit, err := ioutil.ReadFile(rootPath + CBCloudInitFilePath)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.VMInfo{}, err
	}

	sshKey, err := vmHandler.SecuritySshKeyClient.Id(param.SshKeys[0]).GetObject()

	userData := string(fileDataCloudInit)
	userData = strings.ReplaceAll(userData, "{{username}}", CBDefaultVmUserName)
	userData = strings.ReplaceAll(userData, "{{public_key}}", *sshKey.Key)
	userDataBase64 := sl.String(userData)
	virtualGuests[0].UserData = []datatypes.Virtual_Guest_Attribute{
		{
			Type: &datatypes.Virtual_Guest_Attribute_Type{
				Keyname: sl.String("USER_DATA"),
				Name:    sl.String("User Data"),
			},
			Value: sl.String(*userDataBase64),
		},
	}
	start := call.Start()
	//check, err := vmHandler.ProductOrderClient.VerifyOrder(&orderTemplate)
	check, err := vmHandler.ProductOrderClient.PlaceOrder(&orderTemplate,sl.Bool(false))
	// Create VM ID = check.OrderDetails.VirtualGuests[0].Id
	if err !=nil{
		LoggingInfo(hiscallInfo, start)
		return irs.VMInfo{}, err
	}
	creatingVmId := *check.OrderDetails.VirtualGuests[0].Id

	curRetryCnt := 0
	maxRetryCnt := 120
	newmask := "mask[status,id,powerState[name]]"
	//virtualGuestMask := "mask[id,hostname,users,blockDevices,blockDeviceTemplateGroup,sshKeyCount,sshKeys,softwareComponentCount,softwareComponents[softwareDescription[productItemCount,productItems[itemCategory]],passwords],privateNetworkOnlyFlag,billingItem[orderItem[preset]],fullyQualifiedDomainName,domain,createDate,datacenter,primaryIpAddress,primaryBackendIpAddress,backendNetworkComponents[securityGroupBindings[securityGroup],securityGroupBindingCount],frontendNetworkComponents[primarySubnet,securityGroupBindings[securityGroup],securityGroupBindingCount,networkVlan[primaryRouter[hostname],vlanNumber,id,subnets,networkSpace,name]]]"
	for{
		// fmt.Println(curRetryCnt)
		createVm ,_ := vmHandler.VirtualGuestClient.Id(creatingVmId).Mask(newmask).GetObject()
		if createVm.Status != nil && *createVm.Status.KeyName == "ACTIVE"{
			if createVm.PowerState !=nil && *createVm.PowerState.Name == IbmVmStatusRunning {
				vmInfo, err := vmHandler.GetVM(irs.IID{SystemId: strconv.Itoa(creatingVmId)})
				if err != nil {
					LoggingError(hiscallInfo, err)
					return irs.VMInfo{}, err
				}
				vmInfo.VMUserId = CBDefaultVmUserName
				LoggingInfo(hiscallInfo, start)
				return vmInfo,nil
			}
		}
		curRetryCnt++
		time.Sleep(1 * time.Second)
		if curRetryCnt > maxRetryCnt {
			err = errors.New(fmt.Sprintf("failed to create VM, exceeded maximum retry count %d", maxRetryCnt))
			LoggingError(hiscallInfo, err)
			return irs.VMInfo{}, err
		}
	}
}

func(vmHandler *IbmVMHandler) SuspendVM(vmIID irs.IID) (irs.VMStatus, error){
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, vmIID.NameId, "SuspendVM()")

	virtualServer, err := vmHandler.existCheckVMStatus(vmIID)
	if err != nil{
		LoggingError(hiscallInfo, err)
		return irs.Failed, err
	}
	if virtualServer.Id == nil{
		LoggingError(hiscallInfo, err)
		return irs.NotExist, err
	}
	if virtualServer.PowerState != nil{
		if *virtualServer.PowerState.Name == IbmVmStatusRunning {
			start := call.Start()
			_, err = vmHandler.VirtualGuestClient.Id(*virtualServer.Id).PowerOff()
			if err != nil{
				LoggingError(hiscallInfo, err)
				return irs.Failed, err
			}
			LoggingInfo(hiscallInfo, start)
			return irs.Suspending,nil
		} else {
			var status irs.VMStatus
			setVmStatus(&status,&virtualServer)
			err = errors.New(fmt.Sprintf("not Suspend Instance. Instance Status : %s",status))
			LoggingError(hiscallInfo, err)
			return status,err
		}
	}
	err = errors.New("not Found Instance State")
	LoggingError(hiscallInfo, err)
	return irs.Failed,err
}

func(vmHandler *IbmVMHandler) ResumeVM(vmIID irs.IID) (irs.VMStatus, error){
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, vmIID.NameId, "ResumeVM()")
	virtualServer, err := vmHandler.existCheckVMStatus(vmIID)
	if err != nil{
		LoggingError(hiscallInfo, err)
		return irs.Failed, err
	}
	if virtualServer.Id == nil{
		LoggingError(hiscallInfo, err)
		return irs.NotExist, err
	}
	if virtualServer.PowerState != nil{
		start := call.Start()
		switch *virtualServer.PowerState.Name {
			case IbmVmStatusHalted :{
				_, err = vmHandler.VirtualGuestClient.Id(*virtualServer.Id).PowerOn()
				if err != nil{
					LoggingError(hiscallInfo, err)
					return irs.Failed, err
				}
			}
			case IbmVmStatusPaused:{
				_, err = vmHandler.VirtualGuestClient.Id(*virtualServer.Id).Resume()
				if err != nil{
					LoggingError(hiscallInfo, err)
					return irs.Failed, err
				}
			}
			case IbmVmStatusSuspended:{
				err = errors.New("not Resume this Instance Terminating")
				LoggingError(hiscallInfo, err)
				return irs.Failed, err
			}
			default:{
				err = errors.New("already Instance Running")
				LoggingError(hiscallInfo, err)
				return irs.Running, err
			}
		}
		LoggingInfo(hiscallInfo, start)
		return irs.Resuming,nil
	}
	err = errors.New("not Found Instance State")
	LoggingError(hiscallInfo, err)
	return irs.Failed,err
}

func(vmHandler *IbmVMHandler) RebootVM(vmIID irs.IID) (irs.VMStatus, error){
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, vmIID.NameId, "RebootVM()")
	virtualServer, err := vmHandler.existCheckVMStatus(vmIID)
	if err != nil{
		LoggingError(hiscallInfo, err)
		return irs.Failed, err
	}
	if virtualServer.Id == nil{
		err = errors.New("not Exist this Instance")
		LoggingError(hiscallInfo, err)
		return irs.NotExist, err
	}
	if virtualServer.PowerState != nil{
		start := call.Start()
		switch *virtualServer.PowerState.Name {
			case IbmVmStatusHalted:{
				_, err = vmHandler.VirtualGuestClient.Id(*virtualServer.Id).PowerOn()
				if err != nil{
					return irs.Failed, err
				}
			}
			case IbmVmStatusSuspended:{
				err = errors.New("not Resume this Instance Terminating")
				LoggingError(hiscallInfo, err)
				return irs.Failed, err
			}
			case IbmVmStatusRunning:{
				_, err = vmHandler.VirtualGuestClient.Id(*virtualServer.Id).PowerCycle()
				if err != nil{
					LoggingError(hiscallInfo, err)
					return irs.Failed, err
				}
			}
			default :{
				err = errors.New("not Found Instance State")
				LoggingError(hiscallInfo, err)
				return irs.Failed, err
			}
		}
		LoggingInfo(hiscallInfo, start)
		return irs.Rebooting, err
	}
	err = errors.New("not Found Instance State")
	LoggingError(hiscallInfo, err)
	return irs.Failed,err
}

func(vmHandler *IbmVMHandler) TerminateVM(vmIID irs.IID) (irs.VMStatus, error){
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, vmIID.NameId, "TerminateVM()")
	virtualServer, err := vmHandler.existCheckVMStatus(vmIID)
	if err != nil{
		LoggingError(hiscallInfo, err)
		return irs.Failed, err
	}
	if virtualServer.PowerState != nil{
		start := call.Start()
		switch *virtualServer.PowerState.Name {
			case IbmVmStatusSuspended:{
				err = errors.New("not Resume this Instance Terminating")
				LoggingError(hiscallInfo, err)
				return irs.Failed, err
			}
			default : {
				_, err = vmHandler.VirtualGuestClient.Id(*virtualServer.Id).DeleteObject()
				if err != nil{
					LoggingError(hiscallInfo, err)
					return irs.Failed, err
				}
			}
		}
		LoggingInfo(hiscallInfo, start)
		return irs.Terminating,nil
	}
	err = errors.New("not Found Instance State")
	LoggingError(hiscallInfo, err)
	return irs.Failed,err
}

func(vmHandler *IbmVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error){
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, "VMStatus", "ListVMStatus()")

	vmStatusMask := "mask[id,hostname,powerState,datacenter]"
	LocationFilter := filter.Path("virtualGuests.datacenter.name").Eq(vmHandler.Region.Region).Build()
	start := call.Start()
	vms, err := vmHandler.AccountClient.Mask(vmStatusMask).Filter(LocationFilter).GetVirtualGuests()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return []*irs.VMStatusInfo{}, err
	}
	var vmStatusList []*irs.VMStatusInfo
	for _, vm := range vms{
		vmStatusInfo := irs.VMStatusInfo{}
		setVmStatusInfo(&vmStatusInfo,&vm)
		vmStatusList = append(vmStatusList, &vmStatusInfo)
	}
	LoggingInfo(hiscallInfo, start)
	return vmStatusList, nil
}

func(vmHandler *IbmVMHandler) GetVMStatus(vmIID irs.IID) (irs.VMStatus, error){
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, vmIID.NameId, "GetVMStatus()")
	virtualServer ,err := vmHandler.existCheckVMStatus(vmIID)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.Failed, err
	}
	start := call.Start()

	var status irs.VMStatus
	setVmStatus(&status,&virtualServer)
	LoggingInfo(hiscallInfo, start)
	return status,nil
}

func(vmHandler *IbmVMHandler) ListVM() ([]*irs.VMInfo, error){
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, "VM", "ListVM()")

	var vmInfos []*irs.VMInfo
	LocationFilter := filter.Path("virtualGuests.datacenter.name").Eq(vmHandler.Region.Region).Build()
	virtualGuestMask := "mask[id,hostname,users,blockDevices,blockDeviceTemplateGroup,sshKeyCount,sshKeys,softwareComponentCount,softwareComponents[softwareDescription[productItemCount,productItems[itemCategory]],passwords],privateNetworkOnlyFlag,billingItem[orderItem[preset]],fullyQualifiedDomainName,domain,createDate,datacenter,primaryIpAddress,primaryBackendIpAddress,backendNetworkComponents[securityGroupBindings[securityGroup],securityGroupBindingCount],frontendNetworkComponents[primarySubnet,securityGroupBindings[securityGroup],securityGroupBindingCount,networkVlan[primaryRouter[hostname],vlanNumber,id,subnets,networkSpace,name]]]"
	allVirtualServer, err := vmHandler.AccountClient.Mask(virtualGuestMask).Filter(LocationFilter).GetVirtualGuests()
	start := call.Start()
	if err != nil{
		LoggingError(hiscallInfo, err)
		return []*irs.VMInfo{},err
	}
	if len(allVirtualServer) < 1{
		LoggingInfo(hiscallInfo, start)
		return []*irs.VMInfo{},nil
	}
	for _, virtualServer :=range allVirtualServer{
		vmInfo := irs.VMInfo{}
		setterVmIfo(&vmInfo,&virtualServer)
		vmInfos = append(vmInfos,&vmInfo)
	}
	LoggingInfo(hiscallInfo, start)
	return vmInfos,nil
}

func(vmHandler *IbmVMHandler) GetVM(vmIID irs.IID) (irs.VMInfo, error){
	hiscallInfo := GetCallLogScheme(vmHandler.Region, call.VM, vmIID.NameId, "GetVM()")

	vmInfo := irs.VMInfo{}
	var virtualServer datatypes.Virtual_Guest
	numSystemId, err := strconv.Atoi(vmIID.SystemId)
	start := call.Start()
	if err != nil{
		if vmIID.NameId != "" {
			virtualServer, err = vmHandler.getterVMByName(vmIID.NameId)
			if err != nil{
				LoggingError(hiscallInfo, err)
				return irs.VMInfo{}, err
			}
		}
		//LoggingError(hiscallInfo, err)
		//return irs.VMInfo{}, err
	}else{
		virtualGuestMask := "mask[id,hostname,users,blockDevices,blockDeviceTemplateGroup,sshKeyCount,sshKeys,softwareComponentCount,softwareComponents[softwareDescription[productItemCount,productItems[itemCategory]],passwords],privateNetworkOnlyFlag,billingItem[orderItem[preset]],fullyQualifiedDomainName,domain,createDate,datacenter,primaryIpAddress,primaryBackendIpAddress,backendNetworkComponents[securityGroupBindings[securityGroup],securityGroupBindingCount],frontendNetworkComponents[primarySubnet,securityGroupBindings[securityGroup],securityGroupBindingCount,networkVlan[primaryRouter[hostname],vlanNumber,id,subnets,networkSpace,name]]]"
		virtualServer, err = vmHandler.VirtualGuestClient.Mask(virtualGuestMask).Id(numSystemId).GetObject()
		if err != nil {
			LoggingError(hiscallInfo, err)
			return irs.VMInfo{}, err
		}
		err = checkRegion(virtualServer,vmHandler.Region.Region)
		if err != nil{
			LoggingError(hiscallInfo, err)
			return irs.VMInfo{}, err
		}
	}
	setterVmIfo(&vmInfo,&virtualServer)
	LoggingInfo(hiscallInfo, start)
	return vmInfo,nil
}

func (vmHandler *IbmVMHandler) existCheckVMByName(VMName string) (bool, error){
	existFilter := filter.Path("virtualGuests.hostname").Eq(VMName).Build()
	mask := "mask[hostname,datacenter]"
	filterObjects,err := vmHandler.AccountClient.Filter(existFilter).Mask(mask).GetVirtualGuests()
	if err != nil{
		return true, err
	}
	if len(filterObjects) == 0 {
		return false, nil
	} else {
		for _,vm := range filterObjects{
			if *vm.Datacenter.Name == vmHandler.Region.Region{
				return true, errors.New(fmt.Sprintf("VM with name %s already exist", VMName))
			}
		}
		return false, nil
	}
}

func (vmHandler *IbmVMHandler) existCheckVMStatus(vmIID irs.IID) (datatypes.Virtual_Guest, error){
	mask := "mask[hostname,datacenter,powerState,id]"
	if vmIID.SystemId == ""{
		if vmIID.NameId != ""{
			existFilter := filter.Path("virtualGuests.hostname").Eq(vmIID.NameId).Build()
			filterObjects,err := vmHandler.AccountClient.Filter(existFilter).Mask(mask).GetVirtualGuests()
			if err != nil{
				return datatypes.Virtual_Guest{}, err
			}
			if len(filterObjects) == 0 {
				return datatypes.Virtual_Guest{},errors.New(fmt.Sprintf("VM with name %s not exist", vmIID.NameId))
			} else {
				for _,virtualServer := range filterObjects{
					err = checkRegion(virtualServer,vmHandler.Region.Region)
					if err != nil{
						return datatypes.Virtual_Guest{}, errors.New(fmt.Sprintf("VM with name %s not exist", vmIID.NameId))
					}
					return virtualServer, nil
				}
				return datatypes.Virtual_Guest{},errors.New(fmt.Sprintf("VM with name %s not exist", vmIID.NameId))
			}
		}else{
			return datatypes.Virtual_Guest{}, errors.New("invalid VMIId")
		}
	} else {
		numSystemId, err := strconv.Atoi(vmIID.SystemId)
		if err != nil{
			return datatypes.Virtual_Guest{}, err
		}
		virtualServer, err := vmHandler.VirtualGuestClient.Id(numSystemId).Mask(mask).GetObject()
		if err != nil{
			return datatypes.Virtual_Guest{}, err
		}
		err = checkRegion(virtualServer,vmHandler.Region.Region)
		if err != nil{
			return datatypes.Virtual_Guest{}, errors.New(fmt.Sprintf("VM not exist"))
		}
		return virtualServer, nil
	}
}

//func(vmHandler *IbmVMHandler) getVPCAndSubnet(virtaulServerIId irs.IID) (string,error){
//
//}

func (vmHandler *IbmVMHandler) getPrivateVlan() (string,error){
	vlanFilter := filter.Path("networkVlans.primaryRouter.datacenter.name").Eq(vmHandler.Region.Region).Build()
	vlanMask:= "mask[primaryRouter[datacenter],id,name,subnets,subnetCount,networkSpace,vlanNumber]"
	vlans, err :=vmHandler.AccountClient.Mask(vlanMask).Filter(vlanFilter).GetNetworkVlans()
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

func (vmHandler *IbmVMHandler) getterVMStatusByName(VMName string) (datatypes.Virtual_Guest, error) {
	existFilter := filter.Path("virtualGuests.hostname").Eq(VMName).Build()
	virtualGuestMask := "mask[powerState,id,datacenter,hostname]"
	filterObjects,err := vmHandler.AccountClient.Filter(existFilter).Mask(virtualGuestMask).GetVirtualGuests()
	if err != nil{
		return datatypes.Virtual_Guest{}, err
	}
	if len(filterObjects) == 0 {
		return datatypes.Virtual_Guest{}, errors.New(fmt.Sprintf("not found %s", VMName))
	} else {
		for _, vm := range filterObjects{
			if *vm.Datacenter.Name == vmHandler.Region.Region{
				return vm, nil
			}
		}
		return datatypes.Virtual_Guest{}, errors.New(fmt.Sprintf("not found %s", VMName))
	}
}

func (vmHandler *IbmVMHandler) getterSubnetIIDByName(name string, vpcId string) (irs.IID, error) {
	if name != "" && vpcId != ""{
		vlanNumId ,err :=strconv.Atoi(vpcId)
		if err != nil{
			return irs.IID{}, errors.New("invalid vpcId")
		}
		vlanIdFilter := filter.Path("networkVlans.id").Eq(vlanNumId).Build()
		vlanMask:= "mask[id,name,subnets]"
		vlans, err := vmHandler.AccountClient.Filter(vlanIdFilter).Mask(vlanMask).GetNetworkVlans()
		if err != nil{
			return irs.IID{},err
		}
		if len(vlans) > 0 {
			for _, subnet := range vlans[0].Subnets{
				if strings.Contains(*subnet.SubnetType,"PRIMARY"){
					cidr := *subnet.NetworkIdentifier +"/"+ strconv.Itoa(*subnet.Cidr)
					if cidr == name{
						return irs.IID{
							SystemId: strconv.Itoa(*subnet.Id),
							NameId: cidr,//???
						},nil
					}
				}
			}
		} else{
			return irs.IID{}, errors.New(fmt.Sprintf("not exist Subnet %s",name))
		}
	}
	return irs.IID{}, errors.New("invalid SubnetName")
}

func (vmHandler *IbmVMHandler) getterSecurityGroupIIDs(sgs []irs.IID) ([]irs.IID, error){
	var newsgs []irs.IID
	allsgs, err := vmHandler.AccountClient.GetSecurityGroups()
	if err != nil{
		return nil, err
	}
	for _, sg := range  sgs{
		if sg.SystemId == ""{
			if sg.NameId == ""{
				return nil, errors.New("invalid SecurityGroupIIDs")
			}
			for _, rawsg := range allsgs{
				if *rawsg.Name == sg.NameId{
					newsgs = append(newsgs,irs.IID{NameId:sg.NameId, SystemId: strconv.Itoa(*rawsg.Id) })
					break
				}
			}
		}else{
			newsgs = append(newsgs,sg)
		}
	}
	return newsgs ,nil
}

func (vmHandler *IbmVMHandler) getterVpcIIDByName(name string) (irs.IID, error){
	if name != ""{
		vlanFilter := filter.Path("networkVlans.primaryRouter.datacenter.name").Eq(vmHandler.Region.Region).Build()
		vlanMask:= "mask[primaryRouter[datacenter,hostname],id,name,networkSpace,vlanNumber]"
		vlans, err := vmHandler.AccountClient.Filter(vlanFilter).Mask(vlanMask).GetNetworkVlans()
		if err != nil{
			return irs.IID{},err
		}
		if len(vlans) > 0 {
			for _,vlan := range vlans {
				if *vlan.NetworkSpace == "PUBLIC"{
					vlanName :=*vlan.PrimaryRouter.Hostname +"."+ strconv.Itoa(*vlan.VlanNumber)
					if vlanName == name {
						var vlanIId = irs.IID{}
						vlanIId.SystemId =  strconv.Itoa(*vlan.Id)
						vlanIId.NameId = *vlan.PrimaryRouter.Hostname +"."+strconv.Itoa(*vlan.VlanNumber)
						return vlanIId, nil
					}
				}
			}
		}else{
			return irs.IID{}, errors.New(fmt.Sprintf("not exist Vpc %s",name))
		}
	}
	return irs.IID{}, errors.New("invalid VpcName")
}

func (vmHandler *IbmVMHandler) getterVMByName(VMName string) (datatypes.Virtual_Guest, error) {
	existFilter := filter.Path("virtualGuests.hostname").Eq(VMName).Build()
	virtualGuestMask := "mask[id,hostname,users,blockDevices,blockDeviceTemplateGroup,sshKeyCount,sshKeys,softwareComponentCount,softwareComponents[softwareDescription[productItemCount,productItems[itemCategory]],passwords],privateNetworkOnlyFlag,billingItem[orderItem[preset]],fullyQualifiedDomainName,domain,createDate,datacenter,primaryIpAddress,primaryBackendIpAddress,backendNetworkComponents[securityGroupBindings[securityGroup],securityGroupBindingCount],frontendNetworkComponents[primarySubnet,securityGroupBindings[securityGroup],securityGroupBindingCount,networkVlan[primaryRouter[hostname],vlanNumber,id,subnets,networkSpace,name]]]"
	filterObjects,err := vmHandler.AccountClient.Filter(existFilter).Mask(virtualGuestMask).GetVirtualGuests()
	if err != nil{
		return datatypes.Virtual_Guest{}, err
	}
	if len(filterObjects) == 0 {
		return datatypes.Virtual_Guest{}, errors.New(fmt.Sprintf("not found %s", VMName))
	} else {
		for _, vm := range filterObjects{
			if *vm.Datacenter.Name == vmHandler.Region.Region{
				return vm, nil
			}
		}
		return datatypes.Virtual_Guest{}, errors.New(fmt.Sprintf("not found %s", VMName))
	}
}

func  (vmHandler *IbmVMHandler) getterImageIIDBySystemId(systemId string) (irs.IID, error){
	if systemId != ""{
		systemIdNum,err := strconv.Atoi(systemId)
		if err != nil {
			return irs.IID{}, errors.New(	fmt.Sprintf("invalid ImageIID %s",systemId))
		}
		productFilter := filter.Path("keyName").Eq(productName).Build()
		products, err:= vmHandler.ProductPackageClient.Filter(productFilter).GetAllObjects()
		if err != nil {
			return irs.IID{}, err
		}
		if !(len(products) > 0){
			err = errors.New(	fmt.Sprintf("not Exist %s Package",productName))
			return irs.IID{}, err
		}
		osItemMask :="mask[itemCategory[categoryCode],activeUsagePriceCount,capacityRestrictedProductFlag]"
		packageSoftwareItems, err := vmHandler.ProductPackageClient.Mask(osItemMask).Id(*products[0].Id).GetActiveSoftwareItems()

		for _, item := range packageSoftwareItems{
			if *item.Id == systemIdNum{
				return irs.IID{NameId: *item.KeyName, SystemId: strconv.Itoa(*item.Id)}, nil
			}
		}

	}
	return irs.IID{}, errors.New("invalid ImageName")
}


func  (vmHandler *IbmVMHandler) getterKeyPairIIDByName(name string) (irs.IID, error){
	if name != ""{
		keyFilter := filter.Path("sshKey.label").Eq(name).Build()
		sshkeys, err := vmHandler.AccountClient.Filter(keyFilter).GetSshKeys()
		if err != nil{
			return irs.IID{},err
		}
		if len(sshkeys) > 0 {
			return irs.IID{
				NameId: name,
				SystemId: strconv.Itoa(*sshkeys[0].Id),
			},nil
		}else{
			return irs.IID{}, errors.New(fmt.Sprintf("not exist sshKey %s",name))
		}
	}
	return irs.IID{}, errors.New("invalid KeyPairName")
}

func setUserInfo(vmInfo *irs.VMInfo,virtualServer *datatypes.Virtual_Guest){
	vmUserId := ""
	vmUserPasswd := ""
	defer func() {
		recover()
		vmInfo.VMUserId = vmUserId
		vmInfo.VMUserPasswd = vmUserPasswd
	}()
	if *virtualServer.SoftwareComponentCount > 0 {
		softwareComponents := virtualServer.SoftwareComponents
		for _, SoftwareComponent := range softwareComponents{
			description := SoftwareComponent.SoftwareDescription
			if description !=nil && *description.OperatingSystem == 1 {
				passwords := SoftwareComponent.Passwords
				if len(passwords) > 0{
					vmUserId = *passwords[0].Username
					vmUserPasswd = *passwords[0].Password
				}
			}
		}
	}
}

func setKeyPair(vmInfo *irs.VMInfo,virtualServer *datatypes.Virtual_Guest){
	vmKeyPairIId := irs.IID{}
	defer func() {
		recover()
		vmInfo.KeyPairIId = vmKeyPairIId
	}()
	if*virtualServer.SshKeyCount > 0{
		vmKeyPairIId.SystemId= fmt.Sprintf("%d",*virtualServer.SshKeys[0].Id)
		vmKeyPairIId.NameId= *virtualServer.SshKeys[0].Label
	}
}

func setNetwork(vmInfo *irs.VMInfo,virtualServer *datatypes.Virtual_Guest){
	vmPrivateIP := ""
	vmPublicIP := ""
	vmNetworkInterface := ""
	vmVPCName := ""
	vmVPCId := ""
	vmSubnetName := ""
	vmSubnetId := ""
	defer func() {
		recover()
		vmInfo.PrivateIP = vmPrivateIP
		vmInfo.PublicIP = vmPublicIP
		vmInfo.NetworkInterface = vmNetworkInterface
		vmInfo.VpcIID.NameId = vmVPCName
		vmInfo.VpcIID.SystemId = vmVPCId
		vmInfo.SubnetIID.NameId = vmSubnetName
		vmInfo.SubnetIID.SystemId = vmSubnetId
	}()
	if *virtualServer.PrivateNetworkOnlyFlag {
		//privateNetwork only
		vmPrivateIP = *virtualServer.BackendNetworkComponents[0].PrimaryIpAddress
		vmNetworkInterface = fmt.Sprintf("%s%x",*virtualServer.BackendNetworkComponents[0].Name,*virtualServer.BackendNetworkComponents[0].Port)
	}else{
		//privateNetwork/publicNetwork
		vlan := virtualServer.FrontendNetworkComponents[0].NetworkVlan
		subnet := *virtualServer.FrontendNetworkComponents[0].PrimarySubnet
		vmPrivateIP = *virtualServer.BackendNetworkComponents[0].PrimaryIpAddress
		vmPublicIP = *virtualServer.FrontendNetworkComponents[0].PrimaryIpAddress
		vmNetworkInterface = fmt.Sprintf("%s%x/%s%x",*virtualServer.BackendNetworkComponents[0].Name,*virtualServer.BackendNetworkComponents[0].Port,*virtualServer.FrontendNetworkComponents[0].Name,*virtualServer.FrontendNetworkComponents[0].Port)
		vmVPCName =  *vlan.PrimaryRouter.Hostname +"."+strconv.Itoa(*vlan.VlanNumber)
		vmVPCId = strconv.Itoa(*vlan.Id)
		vmSubnetId = strconv.Itoa(*subnet.Id)
		vmSubnetName = *subnet.NetworkIdentifier +"/"+ strconv.Itoa(*subnet.Cidr)
	}
}

func setIId(vmInfo *irs.VMInfo,virtualServer *datatypes.Virtual_Guest) {
	vmIId := irs.IID{}
	defer func() {
		recover()
		vmInfo.IId = vmIId
	}()
	vmIId.SystemId = strconv.Itoa(*virtualServer.Id)
	vmIId.NameId = *virtualServer.Hostname
}

func setStartTime(vmInfo *irs.VMInfo,virtualServer *datatypes.Virtual_Guest) {
	vmStartTime := time.Time{}
	defer func() {
		recover()
		vmInfo.StartTime = vmStartTime
	}()
	vmStartTime = virtualServer.CreateDate.Time.Local()
}

func setVMSpecName(vmInfo *irs.VMInfo,virtualServer *datatypes.Virtual_Guest) {
	vmSpecString := ""
	defer func() {
		recover()
		vmInfo.VMSpecName = vmSpecString
	}()
	vmSpecString = *virtualServer.BillingItem.OrderItem.Preset.Name
}

func setRegion(vmInfo *irs.VMInfo,virtualServer *datatypes.Virtual_Guest) {
	vmRegion := ""
	defer func() {
		recover()
		vmInfo.Region.Region = vmRegion
	}()
	vmRegion = *virtualServer.Datacenter.Name
}

func setImageIId(vmInfo *irs.VMInfo,virtualServer *datatypes.Virtual_Guest) {
	vmImageIId := irs.IID{}
	defer func() {
		recover()
		vmInfo.ImageIId = vmImageIId
	}()
	if *virtualServer.SoftwareComponentCount > 0 {
		softwareComponents := virtualServer.SoftwareComponents
		for _, SoftwareComponent := range softwareComponents{
			description := SoftwareComponent.SoftwareDescription
			if description !=nil && *description.OperatingSystem == 1 {
				if *description.ProductItemCount > 0 {
					for _, productItem := range description.ProductItems{
						if *productItem.ItemCategory.CategoryCode == "os" {
							vmImageIId.SystemId = strconv.Itoa(*productItem.Id)
							vmImageIId.NameId = *productItem.KeyName
						}
					}
				}
			}
		}
	}
}

func setSecurityGroup(vmInfo *irs.VMInfo,virtualServer *datatypes.Virtual_Guest) {
	var vmSecurityInfos []irs.IID
	defer func() {
		recover()
		vmInfo.SecurityGroupIIds = vmSecurityInfos
	}()
	if *virtualServer.FrontendNetworkComponents[0].SecurityGroupBindingCount > 0{
		securityGroupBindings := virtualServer.FrontendNetworkComponents[0].SecurityGroupBindings
		if securityGroupBindings != nil{
			for _,securityGroupBinding :=range securityGroupBindings{
				if securityGroupBinding.SecurityGroup != nil{
					securityIId :=irs.IID{
						SystemId: strconv.Itoa(*securityGroupBinding.SecurityGroup.Id),
						NameId : *securityGroupBinding.SecurityGroup.Name,

					}
					vmSecurityInfos = append(vmSecurityInfos,securityIId)
				}
			}
		}
	}
}

func setVmStatus(status *irs.VMStatus, virtualServer *datatypes.Virtual_Guest){
	resultStatus := irs.Failed
	state := *virtualServer.PowerState.Name
	defer func() {
		recover()
		*status = resultStatus
	}()
	switch state {
	case IbmVmStatusRunning:
		resultStatus = irs.Running
	case  IbmVmStatusHalted:
		resultStatus = irs.Suspended
	case IbmVmStatusPaused:
		resultStatus = irs.Suspended
	case IbmVmStatusSuspended:
		resultStatus = irs.Terminating
	default:
		resultStatus = irs.Failed
	}
}

func setVmStatusInfo(statusInfo *irs.VMStatusInfo,virtualServer *datatypes.Virtual_Guest){
	vmStatusInfo := irs.VMStatusInfo{}
	defer func() {
		recover()
		*statusInfo = vmStatusInfo
	}()
	var status irs.VMStatus
	setVmStatus(&status,virtualServer)
	vmName := fmt.Sprintf("%s",*virtualServer.Hostname)
	vmSystemId := strconv.Itoa(*virtualServer.Id)
	vmStatusInfo = irs.VMStatusInfo{
		IId: irs.IID{
			NameId:   vmName,
			SystemId: vmSystemId,
		},
		VmStatus: status,
	}
}
// for KeyValueList
//func getValue(KeyValueList *[]irs.KeyValue, key string) (string, error){
//	if KeyValueList !=nil{
//		for _, KeyValue := range *KeyValueList {
//			if KeyValue.Key == key {
//				return  KeyValue.Value, nil
//			}
//		}
//	}
//	return "", errors.New(fmt.Sprintf("Not Exist Key %s",key))
//}

func checkRegion(virtualServer datatypes.Virtual_Guest,region string) error{
	if virtualServer.Datacenter == nil{
		return errors.New("not Exist Location Information")
	}
	if *virtualServer.Datacenter.Name != region{
		return errors.New("not Exist Virtual Guest in Location")
	}
	return nil
}

func checkVmReqInfo(vmReqInfo irs.VMReqInfo) error{
	if vmReqInfo.IId.NameId == ""{
		return errors.New("VM name Invalid")
	}
	if vmReqInfo.KeyPairIID.NameId == "" &&  vmReqInfo.KeyPairIID.SystemId == "" {
		return errors.New("KeyPairIID Invalid")
	}
	if vmReqInfo.ImageIID.NameId == "" &&  vmReqInfo.ImageIID.SystemId == "" {
		return errors.New("ImageIID Invalid")
	}
	if vmReqInfo.VMSpecName == ""{
		return errors.New("VMSpecName Invalid")
	}
	return nil
}

func setStartVMParameter(vmReqInfo *irs.VMReqInfo, parameters *IbmInfoParameter, account *datatypes.Account) error {
	var param IbmInfoParameter
	var result error
	defer func() {
		v := recover()
		if v != nil {
			result = errors.New("inValid ReqInfo")
		}
		*parameters = param
	}()
	if vmReqInfo.IId.NameId == ""{
		return errors.New("invalid vm Name")
	}
	param.Hostname = vmReqInfo.IId.NameId
	param.Domain = fmt.Sprintf("cb-spider-%s-%s-Account.Cloud",*account.FirstName,*account.LastName)

	if vmReqInfo.KeyPairIID.SystemId != "" {
		sshKeyId, err := strconv.Atoi(vmReqInfo.KeyPairIID.SystemId)
		result = err
		param.SshKeys = []int{ sshKeyId }
	}
	if vmReqInfo.VMSpecName == ""{
		return errors.New("invalid VMSpecName")
	}
	param.Flavor = vmReqInfo.VMSpecName
	if vmReqInfo.ImageIID.NameId == ""{
		return errors.New("invalid Image IId")
	}
	param.Image = vmReqInfo.ImageIID.NameId
	if len(vmReqInfo.SecurityGroupIIDs) > 0 {
		var sgs []int
		for _, sg := range vmReqInfo.SecurityGroupIIDs{
			id, err := strconv.Atoi(sg.SystemId)
			if err != nil{
				panic("")
			}
			sgs = append(sgs,id)
		}
		if sgs != nil && len(sgs) > 0 {
			param.SecurityGroupIIDs = sgs
		}
	}
	return result
}

