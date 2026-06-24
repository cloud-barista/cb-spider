// Cloud Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Alibaba NIC Handler.
//
// by CB-Spider Team, 2025.06.

package resources

import (
	"fmt"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AlibabaNICHandler struct {
	Region    idrv.RegionInfo
	EcsClient *ecs.Client
}

// alibabaStatusToNICStatus converts Alibaba ENI status to CB-Spider NICStatus.
func alibabaStatusToNICStatus(status string) irs.NICStatus {
	switch status {
	case "Available":
		return irs.NICAvailable
	case "InUse":
		return irs.NICAttached
	case "Deleting":
		return irs.NICDeleting
	default:
		return irs.NICError
	}
}

// mappingNICInfo converts an Alibaba NetworkInterfaceSet to CB-Spider NICInfo.
func mappingNICInfo(eni ecs.NetworkInterfaceSet) (irs.NICInfo, error) {
	createdTime, err := time.Parse(time.RFC3339, eni.CreationTime)
	if err != nil {
		createdTime = time.Time{}
	}

	// Fix 7: NameId fallback for VpcIID and SubnetIID - use SystemId when NameId is empty
	vpcIID := irs.IID{NameId: eni.VpcId, SystemId: eni.VpcId}
	subnetIID := irs.IID{NameId: eni.VSwitchId, SystemId: eni.VSwitchId}

	nicInfo := irs.NICInfo{
		IId: irs.IID{
			NameId:   eni.NetworkInterfaceName,
			SystemId: eni.NetworkInterfaceId,
		},
		VpcIID:     vpcIID,
		SubnetIID:  subnetIID,
		PrivateIP:  eni.PrivateIpAddress,
		MACAddress: eni.MacAddress,
		Status:     alibabaStatusToNICStatus(eni.Status),
		CreatedTime: createdTime,
	}

	// Fix 1 (mappingNICInfo side): use ENI ID as NameId fallback when NetworkInterfaceName is empty
	if nicInfo.IId.NameId == "" {
		nicInfo.IId.NameId = eni.NetworkInterfaceId
	}

	// Fix 7: SecurityGroupIIDs - use SystemId as NameId fallback
	for _, sg := range eni.SecurityGroupIds.SecurityGroupId {
		nicInfo.SecurityGroupIIDs = append(nicInfo.SecurityGroupIIDs, irs.IID{
			NameId:   sg,
			SystemId: sg,
		})
	}

	// Fix 5: Build parallel PrivateIPs / PublicIPs arrays
	var privIPs, pubIPs []string
	for _, pip := range eni.PrivateIpSets.PrivateIpSet {
		privIPs = append(privIPs, pip.PrivateIpAddress)
		pubIP := pip.AssociatedPublicIp.PublicIpAddress
		if pubIP != "" && nicInfo.PublicIP == "" {
			nicInfo.PublicIP = pubIP
		}
		pubIPs = append(pubIPs, pubIP)
	}
	nicInfo.PrivateIPs = privIPs
	nicInfo.PublicIPs = pubIPs

	// Owner VM and device index
	if eni.InstanceId != "" {
		// Fix 7: OwnerVM - use SystemId as NameId fallback
		nicInfo.OwnerVM = irs.IID{
			NameId:   eni.InstanceId,
			SystemId: eni.InstanceId,
		}
	}

	// DeviceIndex: read directly from Attachment.DeviceIndex (SDK struct_attachment.go).
	// Falls back to 0 (primary) if NIC is not attached.
	if eni.InstanceId != "" {
		nicInfo.DeviceIndex = eni.Attachment.DeviceIndex
	} else {
		nicInfo.DeviceIndex = 0
	}

	// Tags
	for _, tag := range eni.Tags.Tag {
		nicInfo.TagList = append(nicInfo.TagList, irs.KeyValue{
			Key:   tag.TagKey,
			Value: tag.TagValue,
		})
	}

	return nicInfo, nil
}

// ListIID returns a list of NIC IIDs.
func (nicHandler *AlibabaNICHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(nicHandler.Region, call.NIC, "NIC", "ListIID()")
	start := call.Start()

	request := ecs.CreateDescribeNetworkInterfacesRequest()
	request.RegionId = nicHandler.Region.Region
	request.PageSize = requests.Integer("100")

	var iidList []*irs.IID
	pageNumber := 1
	for {
		request.PageNumber = requests.Integer(fmt.Sprintf("%d", pageNumber))
		result, err := nicHandler.EcsClient.DescribeNetworkInterfaces(request)
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		if err != nil {
			hiscallInfo.ErrorMSG = err.Error()
			calllogger.Error(call.String(hiscallInfo))
			return nil, fmt.Errorf("failed to list NICs: %w", err)
		}

		for _, eni := range result.NetworkInterfaceSets.NetworkInterfaceSet {
			// Fix 1: use ENI ID as NameId fallback when NetworkInterfaceName is empty
			nameId := eni.NetworkInterfaceName
			if nameId == "" {
				nameId = eni.NetworkInterfaceId
			}
			iidList = append(iidList, &irs.IID{
				NameId:   nameId,
				SystemId: eni.NetworkInterfaceId,
			})
		}

		totalFetched := pageNumber * 100
		if result.TotalCount <= totalFetched || len(result.NetworkInterfaceSets.NetworkInterfaceSet) == 0 {
			break
		}
		pageNumber++
	}

	calllogger.Info(call.String(hiscallInfo))
	return iidList, nil
}

// CreateNIC creates a new NIC.
func (nicHandler *AlibabaNICHandler) CreateNIC(nicReqInfo irs.NICReqInfo) (irs.NICInfo, error) {
	hiscallInfo := GetCallLogScheme(nicHandler.Region, call.NIC, nicReqInfo.IId.NameId, "CreateNIC()")
	start := call.Start()

	request := ecs.CreateCreateNetworkInterfaceRequest()
	request.RegionId = nicHandler.Region.Region
	request.VSwitchId = nicReqInfo.SubnetIID.SystemId
	request.NetworkInterfaceName = nicReqInfo.IId.NameId

	// Fix 2: Pass all security groups - primary via SecurityGroupId, additional via SecurityGroupIds
	if len(nicReqInfo.SecurityGroupIIDs) > 0 {
		request.SecurityGroupId = nicReqInfo.SecurityGroupIIDs[0].SystemId
		if len(nicReqInfo.SecurityGroupIIDs) > 1 {
			additionalSGs := make([]string, 0, len(nicReqInfo.SecurityGroupIIDs)-1)
			for _, sg := range nicReqInfo.SecurityGroupIIDs[1:] {
				additionalSGs = append(additionalSGs, sg.SystemId)
			}
			request.SecurityGroupIds = &additionalSGs
		}
	}

	// Fix 3: Apply TagList using Tag repeated parameter in CreateNetworkInterfaceRequest
	if len(nicReqInfo.TagList) > 0 {
		tags := make([]ecs.CreateNetworkInterfaceTag, 0, len(nicReqInfo.TagList))
		for _, kv := range nicReqInfo.TagList {
			tags = append(tags, ecs.CreateNetworkInterfaceTag{
				Key:   kv.Key,
				Value: kv.Value,
			})
		}
		request.Tag = &tags
	}

	result, err := nicHandler.EcsClient.CreateNetworkInterface(request)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		hiscallInfo.ErrorMSG = err.Error()
		calllogger.Error(call.String(hiscallInfo))
		return irs.NICInfo{}, fmt.Errorf("failed to create NIC: %w", err)
	}
	calllogger.Info(call.String(hiscallInfo))

	return nicHandler.GetNIC(irs.IID{SystemId: result.NetworkInterfaceId})
}

// ListNIC returns a list of all NICs.
// Fix 9: pagination support - fetches all pages with PageSize=100.
func (nicHandler *AlibabaNICHandler) ListNIC() ([]*irs.NICInfo, error) {
	hiscallInfo := GetCallLogScheme(nicHandler.Region, call.NIC, "NIC", "ListNIC()")
	start := call.Start()

	request := ecs.CreateDescribeNetworkInterfacesRequest()
	request.RegionId = nicHandler.Region.Region
	request.PageSize = requests.Integer("100")

	var nicList []*irs.NICInfo
	pageNumber := 1
	for {
		request.PageNumber = requests.Integer(fmt.Sprintf("%d", pageNumber))
		result, err := nicHandler.EcsClient.DescribeNetworkInterfaces(request)
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		if err != nil {
			hiscallInfo.ErrorMSG = err.Error()
			calllogger.Error(call.String(hiscallInfo))
			return nil, fmt.Errorf("failed to list NICs: %w", err)
		}

		for _, eni := range result.NetworkInterfaceSets.NetworkInterfaceSet {
			nicInfo, err := mappingNICInfo(eni)
			if err != nil {
				cblogger.Warnf("Failed to map NIC info for %s: %v", eni.NetworkInterfaceId, err)
				continue
			}
			nicList = append(nicList, &nicInfo)
		}

		totalFetched := pageNumber * 100
		if result.TotalCount <= totalFetched || len(result.NetworkInterfaceSets.NetworkInterfaceSet) == 0 {
			break
		}
		pageNumber++
	}

	calllogger.Info(call.String(hiscallInfo))
	return nicList, nil
}

// GetNIC returns information of a specific NIC.
// Fix 4: when SystemId is empty but NameId is provided, filter by NetworkInterfaceName.
func (nicHandler *AlibabaNICHandler) GetNIC(nicIID irs.IID) (irs.NICInfo, error) {
	hiscallInfo := GetCallLogScheme(nicHandler.Region, call.NIC, nicIID.SystemId, "GetNIC()")
	start := call.Start()

	request := ecs.CreateDescribeNetworkInterfacesRequest()
	request.RegionId = nicHandler.Region.Region

	if nicIID.SystemId != "" {
		request.NetworkInterfaceId = &[]string{nicIID.SystemId}
	} else if nicIID.NameId != "" {
		request.NetworkInterfaceName = nicIID.NameId
	}

	result, err := nicHandler.EcsClient.DescribeNetworkInterfaces(request)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		hiscallInfo.ErrorMSG = err.Error()
		calllogger.Error(call.String(hiscallInfo))
		return irs.NICInfo{}, fmt.Errorf("failed to get NIC %s: %w", nicIID.SystemId, err)
	}
	calllogger.Info(call.String(hiscallInfo))

	if len(result.NetworkInterfaceSets.NetworkInterfaceSet) == 0 {
		id := nicIID.SystemId
		if id == "" {
			id = nicIID.NameId
		}
		return irs.NICInfo{}, fmt.Errorf("NIC not found: %s", id)
	}

	return mappingNICInfo(result.NetworkInterfaceSets.NetworkInterfaceSet[0])
}

// DeleteNIC deletes a NIC.
func (nicHandler *AlibabaNICHandler) DeleteNIC(nicIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(nicHandler.Region, call.NIC, nicIID.SystemId, "DeleteNIC()")
	start := call.Start()

	request := ecs.CreateDeleteNetworkInterfaceRequest()
	request.RegionId = nicHandler.Region.Region
	request.NetworkInterfaceId = nicIID.SystemId

	_, err := nicHandler.EcsClient.DeleteNetworkInterface(request)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		hiscallInfo.ErrorMSG = err.Error()
		calllogger.Error(call.String(hiscallInfo))
		return false, fmt.Errorf("failed to delete NIC %s: %w", nicIID.SystemId, err)
	}
	calllogger.Info(call.String(hiscallInfo))

	return true, nil
}

// AttachNIC attaches a NIC to a VM.
func (nicHandler *AlibabaNICHandler) AttachNIC(nicIID irs.IID, vmIID irs.IID) (irs.NICInfo, error) {
	hiscallInfo := GetCallLogScheme(nicHandler.Region, call.NIC, nicIID.SystemId, "AttachNIC()")
	start := call.Start()

	request := ecs.CreateAttachNetworkInterfaceRequest()
	request.RegionId = nicHandler.Region.Region
	request.NetworkInterfaceId = nicIID.SystemId
	request.InstanceId = vmIID.SystemId

	_, err := nicHandler.EcsClient.AttachNetworkInterface(request)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		hiscallInfo.ErrorMSG = err.Error()
		calllogger.Error(call.String(hiscallInfo))
		return irs.NICInfo{}, fmt.Errorf("failed to attach NIC %s to VM %s: %w", nicIID.SystemId, vmIID.SystemId, err)
	}
	calllogger.Info(call.String(hiscallInfo))

	return nicHandler.GetNIC(nicIID)
}

// DetachNIC detaches a NIC from its owner VM.
func (nicHandler *AlibabaNICHandler) DetachNIC(nicIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(nicHandler.Region, call.NIC, nicIID.SystemId, "DetachNIC()")
	start := call.Start()

	// First, find the owner VM via GetNIC
	nicInfo, err := nicHandler.GetNIC(nicIID)
	if err != nil {
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		hiscallInfo.ErrorMSG = err.Error()
		calllogger.Error(call.String(hiscallInfo))
		return false, fmt.Errorf("failed to get NIC info for detach: %w", err)
	}

	if nicInfo.OwnerVM.SystemId == "" {
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		hiscallInfo.ErrorMSG = "NIC is not attached to any VM"
		calllogger.Error(call.String(hiscallInfo))
		return false, fmt.Errorf("NIC %s is not attached to any VM", nicIID.SystemId)
	}

	request := ecs.CreateDetachNetworkInterfaceRequest()
	request.RegionId = nicHandler.Region.Region
	request.NetworkInterfaceId = nicIID.SystemId
	request.InstanceId = nicInfo.OwnerVM.SystemId

	_, err = nicHandler.EcsClient.DetachNetworkInterface(request)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		hiscallInfo.ErrorMSG = err.Error()
		calllogger.Error(call.String(hiscallInfo))
		return false, fmt.Errorf("failed to detach NIC %s from VM %s: %w", nicIID.SystemId, nicInfo.OwnerVM.SystemId, err)
	}
	calllogger.Info(call.String(hiscallInfo))

	return true, nil
}

// AddPrivateIP assigns a secondary private IP address to an ENI using Alibaba ECS AssignPrivateIpAddresses API.
// Fix 8: when privateIP is empty, use SecondaryPrivateIpAddressCount=1 for auto-assign.
func (h *AlibabaNICHandler) AddPrivateIP(nicIID irs.IID, privateIP string) (irs.NICInfo, error) {
	if nicIID.SystemId == "" {
		return irs.NICInfo{}, fmt.Errorf("AlibabaNICHandler.AddPrivateIP: nicIID.SystemId is required")
	}
	req := ecs.CreateAssignPrivateIpAddressesRequest()
	req.NetworkInterfaceId = nicIID.SystemId
	if privateIP != "" {
		req.PrivateIpAddress = &[]string{privateIP}
	} else {
		// Auto-assign one secondary private IP
		req.SecondaryPrivateIpAddressCount = requests.Integer("1")
	}
	_, err := h.EcsClient.AssignPrivateIpAddresses(req)
	if err != nil {
		return irs.NICInfo{}, fmt.Errorf("AlibabaNICHandler.AddPrivateIP: %v", err)
	}
	return h.GetNIC(nicIID)
}

// RemovePrivateIP unassigns a secondary private IP address from an ENI using Alibaba ECS UnassignPrivateIpAddresses API.
func (h *AlibabaNICHandler) RemovePrivateIP(nicIID irs.IID, privateIP string) (bool, error) {
	if nicIID.SystemId == "" {
		return false, fmt.Errorf("AlibabaNICHandler.RemovePrivateIP: nicIID.SystemId is required")
	}
	req := ecs.CreateUnassignPrivateIpAddressesRequest()
	req.NetworkInterfaceId = nicIID.SystemId
	req.PrivateIpAddress = &[]string{privateIP}
	_, err := h.EcsClient.UnassignPrivateIpAddresses(req)
	if err != nil {
		return false, fmt.Errorf("AlibabaNICHandler.RemovePrivateIP: %v", err)
	}
	return true, nil
}

// GetNICOSConfigScript returns a bash script that must be run inside the Alibaba Cloud VM OS
// after a secondary ENI is attached. Alibaba Cloud does not auto-configure secondary ENIs;
// the OS must bring up the interface and configure policy-based routing.
func (nicHandler *AlibabaNICHandler) GetNICOSConfigScript(nicIID irs.IID) (string, error) {
	nicInfo, err := nicHandler.GetNIC(nicIID)
	if err != nil {
		return "", fmt.Errorf("GetNICOSConfigScript: failed to get NIC info: %w", err)
	}

	privateIPs := nicInfo.PrivateIPs
	if len(privateIPs) == 0 && nicInfo.PrivateIP != "" {
		privateIPs = []string{nicInfo.PrivateIP}
	}
	if len(privateIPs) == 0 {
		return "", fmt.Errorf("GetNICOSConfigScript: NIC has no private IPs")
	}

	mac := strings.ToLower(nicInfo.MACAddress)
	ip0 := privateIPs[0]
	lastDot := strings.LastIndex(ip0, ".")
	gateway := ip0[:lastDot] + ".1"
	cidr := ip0[:lastDot] + ".0/24"

	script := "# 1. Identify interface name by MAC\n" +
		"IFACE=$(ip link | grep -B1 \"" + mac + "\" | head -1 | awk -F': ' '{print $2}')\n" +
		"echo \"Target interface: $IFACE\"\n"

	script += "\n# 2. Bring up secondary NIC (do NOT use dhclient — it adds a default route\n" +
		"#    to the main table and breaks primary NIC SSH)\n" +
		"sudo ip link set $IFACE up\n"
	for _, ip := range privateIPs {
		script += "sudo ip addr add " + ip + "/24 dev $IFACE 2>/dev/null || true\n"
	}

	script += "\n# 3. Configure Policy-based Routing (PBR)\n"
	script += "sudo ip route flush table 101 2>/dev/null || true\n"
	for _, ip := range privateIPs {
		script += "sudo ip rule del from " + ip + " lookup 101 2>/dev/null || true\n"
	}
	script += "sudo ip route add " + cidr + " dev $IFACE src " + ip0 + " table 101\n"
	script += "sudo ip route add default via " + gateway + " dev $IFACE table 101\n"
	for _, ip := range privateIPs {
		script += "sudo ip rule add from " + ip + " lookup 101\n"
	}

	return script, nil
}
