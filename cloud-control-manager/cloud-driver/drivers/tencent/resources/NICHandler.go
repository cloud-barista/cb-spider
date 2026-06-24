// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// Tencent Cloud NIC (Network Interface Card) Handler
//
// by CB-Spider Team, 2025.06.

package resources

import (
	"fmt"
	"strings"
	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tencentvpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

type TencentNICHandler struct {
	Region    idrv.RegionInfo
	VPCClient *tencentvpc.Client
}

// ListIID returns all NIC IIDs in the region.
// Fix 5: pagination loop using Offset/Limit.
// Fix 1: use NetworkInterfaceId as NameId fallback when NetworkInterfaceName is empty.
func (h *TencentNICHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.NIC, "ListIID", "DescribeNetworkInterfaces()")
	start := call.Start()

	var iidList []*irs.IID
	var offset uint64 = 0
	const limit uint64 = 100

	for {
		req := tencentvpc.NewDescribeNetworkInterfacesRequest()
		req.Offset = common.Uint64Ptr(offset)
		req.Limit = common.Uint64Ptr(limit)

		resp, err := h.VPCClient.DescribeNetworkInterfaces(req)
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		if err != nil {
			cblogger.Error(err)
			LoggingError(hiscallInfo, err)
			return nil, err
		}

		for _, nic := range resp.Response.NetworkInterfaceSet {
			nameId := ""
			if nic.NetworkInterfaceName != nil && *nic.NetworkInterfaceName != "" {
				nameId = *nic.NetworkInterfaceName
			} else if nic.NetworkInterfaceId != nil {
				// Fix 1: fallback to SystemId when name is empty
				nameId = *nic.NetworkInterfaceId
			}
			systemId := ""
			if nic.NetworkInterfaceId != nil {
				systemId = *nic.NetworkInterfaceId
			}
			iidList = append(iidList, &irs.IID{NameId: nameId, SystemId: systemId})
		}

		fetched := offset + uint64(len(resp.Response.NetworkInterfaceSet))
		if resp.Response.TotalCount == nil || fetched >= *resp.Response.TotalCount || len(resp.Response.NetworkInterfaceSet) == 0 {
			break
		}
		offset += limit
	}

	LoggingInfo(hiscallInfo, start)
	return iidList, nil
}

// CreateNIC creates a new network interface.
func (h *TencentNICHandler) CreateNIC(nicReqInfo irs.NICReqInfo) (irs.NICInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.NIC, nicReqInfo.IId.NameId, "CreateNetworkInterface()")
	start := call.Start()

	req := tencentvpc.NewCreateNetworkInterfaceRequest()
	req.VpcId = common.StringPtr(nicReqInfo.VpcIID.SystemId)
	req.SubnetId = common.StringPtr(nicReqInfo.SubnetIID.SystemId)
	req.NetworkInterfaceName = common.StringPtr(nicReqInfo.IId.NameId)

	if len(nicReqInfo.SecurityGroupIIDs) > 0 {
		sgIds := make([]*string, len(nicReqInfo.SecurityGroupIIDs))
		for i, sg := range nicReqInfo.SecurityGroupIIDs {
			sgId := sg.SystemId
			sgIds[i] = common.StringPtr(sgId)
		}
		req.SecurityGroupIds = sgIds
	}

	resp, err := h.VPCClient.CreateNetworkInterface(req)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.NICInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	if resp.Response.NetworkInterface == nil {
		return irs.NICInfo{}, fmt.Errorf("CreateNIC: empty response from Tencent API")
	}

	return h.toNICInfo(resp.Response.NetworkInterface), nil
}

// ListNIC returns all NICs in the region.
// Fix 5: pagination loop using Offset/Limit.
func (h *TencentNICHandler) ListNIC() ([]*irs.NICInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.NIC, "ListNIC", "DescribeNetworkInterfaces()")
	start := call.Start()

	var nicList []*irs.NICInfo
	var offset uint64 = 0
	const limit uint64 = 100

	for {
		req := tencentvpc.NewDescribeNetworkInterfacesRequest()
		req.Offset = common.Uint64Ptr(offset)
		req.Limit = common.Uint64Ptr(limit)

		resp, err := h.VPCClient.DescribeNetworkInterfaces(req)
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		if err != nil {
			cblogger.Error(err)
			LoggingError(hiscallInfo, err)
			return nil, err
		}

		for _, nic := range resp.Response.NetworkInterfaceSet {
			info := h.toNICInfo(nic)
			nicList = append(nicList, &info)
		}

		fetched := offset + uint64(len(resp.Response.NetworkInterfaceSet))
		if resp.Response.TotalCount == nil || fetched >= *resp.Response.TotalCount || len(resp.Response.NetworkInterfaceSet) == 0 {
			break
		}
		offset += limit
	}

	LoggingInfo(hiscallInfo, start)
	return nicList, nil
}

// GetNIC returns information about a specific NIC.
func (h *TencentNICHandler) GetNIC(nicIID irs.IID) (irs.NICInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.NIC, nicIID.SystemId, "DescribeNetworkInterfaces()")
	start := call.Start()

	req := tencentvpc.NewDescribeNetworkInterfacesRequest()
	req.NetworkInterfaceIds = []*string{common.StringPtr(nicIID.SystemId)}

	resp, err := h.VPCClient.DescribeNetworkInterfaces(req)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.NICInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	if len(resp.Response.NetworkInterfaceSet) == 0 {
		return irs.NICInfo{}, fmt.Errorf("GetNIC: NIC not found: %s", nicIID.SystemId)
	}

	return h.toNICInfo(resp.Response.NetworkInterfaceSet[0]), nil
}

// DeleteNIC deletes a network interface.
func (h *TencentNICHandler) DeleteNIC(nicIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.NIC, nicIID.SystemId, "DeleteNetworkInterface()")
	start := call.Start()

	req := tencentvpc.NewDeleteNetworkInterfaceRequest()
	req.NetworkInterfaceId = common.StringPtr(nicIID.SystemId)

	_, err := h.VPCClient.DeleteNetworkInterface(req)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	LoggingInfo(hiscallInfo, start)

	return true, nil
}

// AttachNIC attaches a NIC to a VM instance.
func (h *TencentNICHandler) AttachNIC(nicIID irs.IID, vmIID irs.IID) (irs.NICInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.NIC, nicIID.SystemId, "AttachNetworkInterface()")
	start := call.Start()

	req := tencentvpc.NewAttachNetworkInterfaceRequest()
	req.NetworkInterfaceId = common.StringPtr(nicIID.SystemId)
	req.InstanceId = common.StringPtr(vmIID.SystemId)

	_, err := h.VPCClient.AttachNetworkInterface(req)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.NICInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	// Tencent AttachNetworkInterface is asynchronous — poll until NIC status becomes ATTACHED.
	for i := 0; i < 30; i++ {
		time.Sleep(3 * time.Second)
		info, pollErr := h.GetNIC(nicIID)
		if pollErr == nil && info.Status == irs.NICAttached {
			return info, nil
		}
	}
	return h.GetNIC(nicIID)
}

// DetachNIC detaches a NIC from its attached VM instance.
func (h *TencentNICHandler) DetachNIC(nicIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.NIC, nicIID.SystemId, "DetachNetworkInterface()")
	start := call.Start()

	// Retrieve the NIC to find the attached instance ID
	nicInfo, err := h.GetNIC(nicIID)
	if err != nil {
		return false, fmt.Errorf("DetachNIC: failed to get NIC info: %w", err)
	}

	if nicInfo.Status != irs.NICAttached {
		return false, fmt.Errorf("DetachNIC: NIC %s is not attached to any VM", nicIID.SystemId)
	}

	req := tencentvpc.NewDetachNetworkInterfaceRequest()
	req.NetworkInterfaceId = common.StringPtr(nicIID.SystemId)
	req.InstanceId = common.StringPtr(nicInfo.OwnerVM.SystemId)

	_, err = h.VPCClient.DetachNetworkInterface(req)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}

	// Tencent DetachNetworkInterface is asynchronous — poll until NIC status becomes AVAILABLE.
	for i := 0; i < 30; i++ {
		time.Sleep(3 * time.Second)
		info, pollErr := h.GetNIC(nicIID)
		if pollErr != nil {
			continue
		}
		if info.Status == irs.NICAvailable {
			break
		}
	}

	LoggingInfo(hiscallInfo, start)
	return true, nil
}

// toNICInfo converts a Tencent NetworkInterface object to irs.NICInfo.
// Fix 1: use NetworkInterfaceId as NameId fallback when NetworkInterfaceName is empty.
// Fix 2: use SystemId as NameId fallback for VpcIID, SubnetIID, SecurityGroupIIDs, OwnerVM.
// Fix 3: build index-aligned PrivateIPs[] and PublicIPs[] arrays.
func (h *TencentNICHandler) toNICInfo(nic *tencentvpc.NetworkInterface) irs.NICInfo {
	info := irs.NICInfo{}

	if nic.NetworkInterfaceId != nil {
		info.IId.SystemId = *nic.NetworkInterfaceId
	}
	// Fix 1: NameId fallback
	if nic.NetworkInterfaceName != nil && *nic.NetworkInterfaceName != "" {
		info.IId.NameId = *nic.NetworkInterfaceName
	} else {
		info.IId.NameId = info.IId.SystemId
	}

	// Fix 2: VpcIID and SubnetIID - use SystemId as NameId fallback
	if nic.VpcId != nil {
		info.VpcIID = irs.IID{NameId: *nic.VpcId, SystemId: *nic.VpcId}
	}
	if nic.SubnetId != nil {
		info.SubnetIID = irs.IID{NameId: *nic.SubnetId, SystemId: *nic.SubnetId}
	}

	if nic.MacAddress != nil {
		info.MACAddress = *nic.MacAddress
	}

	// Fix 3: Build index-aligned PrivateIPs[] and PublicIPs[] arrays.
	// PrivateIpAddressSpecification has PublicIpAddress field directly.
	for _, pip := range nic.PrivateIpAddressSet {
		privateIP := ""
		if pip.PrivateIpAddress != nil {
			privateIP = *pip.PrivateIpAddress
		}
		info.PrivateIPs = append(info.PrivateIPs, privateIP)

		publicIP := ""
		if pip.PublicIpAddress != nil {
			publicIP = *pip.PublicIpAddress
		}
		info.PublicIPs = append(info.PublicIPs, publicIP)

		// Set primary private IP from first entry
		if info.PrivateIP == "" && privateIP != "" {
			info.PrivateIP = privateIP
		}
		// Set PublicIP from first non-empty public IP
		if info.PublicIP == "" && publicIP != "" {
			info.PublicIP = publicIP
		}
	}

	// Fix 2: SecurityGroupIIDs - use SystemId as NameId fallback
	for _, sg := range nic.GroupSet {
		if sg != nil {
			info.SecurityGroupIIDs = append(info.SecurityGroupIIDs, irs.IID{NameId: *sg, SystemId: *sg})
		}
	}

	// Attachment info
	if nic.Attachment != nil {
		info.Status = irs.NICAttached
		if nic.Attachment.InstanceId != nil {
			// Fix 2: OwnerVM - use SystemId as NameId fallback
			info.OwnerVM = irs.IID{NameId: *nic.Attachment.InstanceId, SystemId: *nic.Attachment.InstanceId}
		}
		if nic.Attachment.DeviceIndex != nil {
			info.DeviceIndex = int(*nic.Attachment.DeviceIndex)
		}
	} else {
		info.Status = irs.NICAvailable
	}

	// CreatedTime
	if nic.CreatedTime != nil {
		t, err := time.Parse("2006-01-02 15:04:05", *nic.CreatedTime)
		if err == nil {
			info.CreatedTime = t
		}
	}

	// Tags
	for _, tag := range nic.TagSet {
		if tag.Key != nil && tag.Value != nil {
			info.TagList = append(info.TagList, irs.KeyValue{Key: *tag.Key, Value: *tag.Value})
		}
	}

	return info
}

// AddPrivateIP assigns a secondary private IP address to a Tencent ENI via VPC AssignPrivateIpAddresses API.
// Fix 4: when privateIP is empty, use SecondaryPrivateIpAddressCount=1 for auto-assign.
func (h *TencentNICHandler) AddPrivateIP(nicIID irs.IID, privateIP string) (irs.NICInfo, error) {
	if nicIID.SystemId == "" {
		return irs.NICInfo{}, fmt.Errorf("TencentNICHandler.AddPrivateIP: nicIID.SystemId is required")
	}
	req := tencentvpc.NewAssignPrivateIpAddressesRequest()
	req.NetworkInterfaceId = common.StringPtr(nicIID.SystemId)
	if privateIP != "" {
		req.PrivateIpAddresses = []*tencentvpc.PrivateIpAddressSpecification{
			{PrivateIpAddress: common.StringPtr(privateIP)},
		}
	} else {
		// Fix 4: auto-assign one secondary private IP
		var count uint64 = 1
		req.SecondaryPrivateIpAddressCount = &count
	}
	_, err := h.VPCClient.AssignPrivateIpAddresses(req)
	if err != nil {
		return irs.NICInfo{}, fmt.Errorf("TencentNICHandler.AddPrivateIP: %v", err)
	}
	return h.GetNIC(nicIID)
}

// RemovePrivateIP unassigns a secondary private IP address from a Tencent ENI via VPC UnassignPrivateIpAddresses API.
func (h *TencentNICHandler) RemovePrivateIP(nicIID irs.IID, privateIP string) (bool, error) {
	if nicIID.SystemId == "" {
		return false, fmt.Errorf("TencentNICHandler.RemovePrivateIP: nicIID.SystemId is required")
	}
	req := tencentvpc.NewUnassignPrivateIpAddressesRequest()
	req.NetworkInterfaceId = common.StringPtr(nicIID.SystemId)
	req.PrivateIpAddresses = []*tencentvpc.PrivateIpAddressSpecification{
		{PrivateIpAddress: common.StringPtr(privateIP)},
	}
	_, err := h.VPCClient.UnassignPrivateIpAddresses(req)
	if err != nil {
		return false, fmt.Errorf("TencentNICHandler.RemovePrivateIP: %v", err)
	}
	return true, nil
}

// GetNICOSConfigScript returns a bash script that must be run inside the Tencent Cloud VM OS
// after a secondary ENI is attached. Tencent Cloud does not auto-configure secondary ENIs;
// the guest OS must bring up the interface and configure policy-based routing.
func (h *TencentNICHandler) GetNICOSConfigScript(nicIID irs.IID) (string, error) {
	nicInfo, err := h.GetNIC(nicIID)
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
