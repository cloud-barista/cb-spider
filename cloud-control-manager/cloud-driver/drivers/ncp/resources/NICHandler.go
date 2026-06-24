// Cloud Driver Interface of CB-Spider.
// NCP VPC NIC (Network Interface) Handler
//
// by CB-Spider Team, 2025.06.

package resources

import (
	"fmt"
	"time"

	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vpc"
	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vserver"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NcpVpcNICHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	VMClient       *vserver.APIClient
	VpcClient      *vpc.APIClient
}

func (h *NcpVpcNICHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(h.RegionInfo.Zone, call.NIC, "ListIID", "GetNetworkInterfaceList()")
	start := call.Start()
	req := &vserver.GetNetworkInterfaceListRequest{RegionCode: ncloud.String(h.RegionInfo.Region)}
	resp, err := h.VMClient.V2Api.GetNetworkInterfaceList(req)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil { cblogger.Error(err); LoggingError(hiscallInfo, err); return nil, err }
	LoggingInfo(hiscallInfo, start)
	var list []*irs.IID
	for _, ni := range resp.NetworkInterfaceList {
		list = append(list, &irs.IID{NameId: ncpNICNameId(ni), SystemId: ncloud.StringValue(ni.NetworkInterfaceNo)})
	}
	return list, nil
}

func (h *NcpVpcNICHandler) CreateNIC(req irs.NICReqInfo) (irs.NICInfo, error) {
	hiscallInfo := GetCallLogScheme(h.RegionInfo.Zone, call.NIC, req.IId.NameId, "CreateNetworkInterface()")
	start := call.Start()
	subnetNo := req.SubnetIID.SystemId; if subnetNo == "" { subnetNo = req.SubnetIID.NameId }

	// NCP CreateNetworkInterface requires VpcNo — resolve it from the subnet
	var vpcNo string
	if h.VpcClient != nil {
		subnetResp, subnetErr := h.VpcClient.V2Api.GetSubnetDetail(&vpc.GetSubnetDetailRequest{
			RegionCode: ncloud.String(h.RegionInfo.Region),
			SubnetNo:   ncloud.String(subnetNo),
		})
		if subnetErr != nil || len(subnetResp.SubnetList) == 0 {
			return irs.NICInfo{}, fmt.Errorf("CreateNIC: failed to resolve VpcNo from subnet [%s]: %v", subnetNo, subnetErr)
		}
		if subnetResp.SubnetList[0].VpcNo != nil {
			vpcNo = *subnetResp.SubnetList[0].VpcNo
		}
	}
	if vpcNo == "" {
		return irs.NICInfo{}, fmt.Errorf("CreateNIC: could not determine VpcNo for subnet [%s]", subnetNo)
	}

	var acgNos []*string
	for _, sg := range req.SecurityGroupIIDs { id := sg.SystemId; if id == "" { id = sg.NameId }; acgNos = append(acgNos, ncloud.String(id)) }
	createReq := &vserver.CreateNetworkInterfaceRequest{
		RegionCode:                  ncloud.String(h.RegionInfo.Region),
		VpcNo:                       ncloud.String(vpcNo),
		SubnetNo:                    ncloud.String(subnetNo),
		NetworkInterfaceName:        ncloud.String(req.IId.NameId),
		NetworkInterfaceDescription: ncloud.String(req.IId.NameId),
		AccessControlGroupNoList:    acgNos,
	}
	resp, err := h.VMClient.V2Api.CreateNetworkInterface(createReq)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil { cblogger.Error(err); LoggingError(hiscallInfo, err); return irs.NICInfo{}, err }
	if len(resp.NetworkInterfaceList) == 0 { return irs.NICInfo{}, fmt.Errorf("CreateNetworkInterface returned empty list") }
	LoggingInfo(hiscallInfo, start)
	ni := resp.NetworkInterfaceList[0]
	info := extractNcpNICInfo(ni)
	info.IId.NameId = req.IId.NameId
	h.enrichNICInfo(&info, ni)
	return info, nil
}

func (h *NcpVpcNICHandler) ListNIC() ([]*irs.NICInfo, error) {
	hiscallInfo := GetCallLogScheme(h.RegionInfo.Zone, call.NIC, "All", "GetNetworkInterfaceList()")
	start := call.Start()
	req := &vserver.GetNetworkInterfaceListRequest{RegionCode: ncloud.String(h.RegionInfo.Region)}
	resp, err := h.VMClient.V2Api.GetNetworkInterfaceList(req)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil { cblogger.Error(err); LoggingError(hiscallInfo, err); return nil, err }
	LoggingInfo(hiscallInfo, start)
	var list []*irs.NICInfo
	for _, ni := range resp.NetworkInterfaceList {
		info := extractNcpNICInfo(ni)
		h.enrichNICInfo(&info, ni)
		list = append(list, &info)
	}
	if list == nil { list = []*irs.NICInfo{} }
	return list, nil
}

func (h *NcpVpcNICHandler) GetNIC(iid irs.IID) (irs.NICInfo, error) {
	hiscallInfo := GetCallLogScheme(h.RegionInfo.Zone, call.NIC, iid.NameId, "GetNetworkInterfaceList()")
	start := call.Start()
	req := &vserver.GetNetworkInterfaceListRequest{RegionCode: ncloud.String(h.RegionInfo.Region)}
	if iid.SystemId != "" { req.NetworkInterfaceNoList = []*string{ncloud.String(iid.SystemId)} }
	resp, err := h.VMClient.V2Api.GetNetworkInterfaceList(req)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil { cblogger.Error(err); LoggingError(hiscallInfo, err); return irs.NICInfo{}, err }
	LoggingInfo(hiscallInfo, start)
	for _, ni := range resp.NetworkInterfaceList {
		if iid.SystemId != "" && ncloud.StringValue(ni.NetworkInterfaceNo) == iid.SystemId {
			info := extractNcpNICInfo(ni)
			if iid.NameId != "" { info.IId.NameId = iid.NameId }
			h.enrichNICInfo(&info, ni)
			return info, nil
		}
		if ncloud.StringValue(ni.NetworkInterfaceName) == iid.NameId {
			info := extractNcpNICInfo(ni)
			info.IId.NameId = iid.NameId
			h.enrichNICInfo(&info, ni)
			return info, nil
		}
	}
	return irs.NICInfo{}, fmt.Errorf("NIC not found: %s", iid.NameId)
}

func (h *NcpVpcNICHandler) DeleteNIC(iid irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(h.RegionInfo.Zone, call.NIC, iid.NameId, "DeleteNetworkInterface()")
	start := call.Start()
	info, err := h.GetNIC(iid); if err != nil { return false, err }
	req := &vserver.DeleteNetworkInterfaceRequest{RegionCode: ncloud.String(h.RegionInfo.Region), NetworkInterfaceNo: ncloud.String(info.IId.SystemId)}
	_, err = h.VMClient.V2Api.DeleteNetworkInterface(req)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil { cblogger.Error(err); LoggingError(hiscallInfo, err); return false, err }
	LoggingInfo(hiscallInfo, start); return true, nil
}

func (h *NcpVpcNICHandler) AttachNIC(nicIID irs.IID, vmIID irs.IID) (irs.NICInfo, error) {
	hiscallInfo := GetCallLogScheme(h.RegionInfo.Zone, call.NIC, nicIID.NameId, "AttachNetworkInterface()")
	start := call.Start()
	nicInfo, err := h.GetNIC(nicIID); if err != nil { return irs.NICInfo{}, err }
	serverNo := vmIID.SystemId; if serverNo == "" { serverNo = vmIID.NameId }
	subnetNo := nicInfo.SubnetIID.SystemId; if subnetNo == "" { subnetNo = nicInfo.SubnetIID.NameId }
	req := &vserver.AttachNetworkInterfaceRequest{
		RegionCode: ncloud.String(h.RegionInfo.Region),
		NetworkInterfaceNo: ncloud.String(nicInfo.IId.SystemId),
		ServerInstanceNo: ncloud.String(serverNo),
		SubnetNo: ncloud.String(subnetNo),
	}
	_, err = h.VMClient.V2Api.AttachNetworkInterface(req)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil { cblogger.Error(err); LoggingError(hiscallInfo, err); return irs.NICInfo{}, err }
	LoggingInfo(hiscallInfo, start); return h.GetNIC(nicIID)
}

func (h *NcpVpcNICHandler) DetachNIC(nicIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(h.RegionInfo.Zone, call.NIC, nicIID.NameId, "DetachNetworkInterface()")
	start := call.Start()
	nicInfo, err := h.GetNIC(nicIID); if err != nil { return false, err }
	if nicInfo.Status != irs.NICAttached { return false, fmt.Errorf("NIC %s is not attached", nicIID.NameId) }
	serverNo := nicInfo.OwnerVM.SystemId; if serverNo == "" { serverNo = nicInfo.OwnerVM.NameId }
	subnetNo := nicInfo.SubnetIID.SystemId; if subnetNo == "" { subnetNo = nicInfo.SubnetIID.NameId }
	req := &vserver.DetachNetworkInterfaceRequest{
		RegionCode: ncloud.String(h.RegionInfo.Region),
		NetworkInterfaceNo: ncloud.String(nicInfo.IId.SystemId),
		ServerInstanceNo: ncloud.String(serverNo),
		SubnetNo: ncloud.String(subnetNo),
	}
	_, err = h.VMClient.V2Api.DetachNetworkInterface(req)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil { cblogger.Error(err); LoggingError(hiscallInfo, err); return false, err }
	LoggingInfo(hiscallInfo, start); return true, nil
}

func extractNcpNICInfo(ni *vserver.NetworkInterface) irs.NICInfo {
	status := irs.NICAvailable
	if ni.InstanceNo != nil && ncloud.StringValue(ni.InstanceNo) != "" { status = irs.NICAttached }
	info := irs.NICInfo{
		IId:         irs.IID{NameId: ncpNICNameId(ni), SystemId: ncloud.StringValue(ni.NetworkInterfaceNo)},
		SubnetIID:   irs.IID{SystemId: ncloud.StringValue(ni.SubnetNo)},
		VpcIID:      irs.IID{SystemId: "NA"},
		PrivateIP:   ncloud.StringValue(ni.Ip),
		MACAddress:  ncloud.StringValue(ni.MacAddress),
		Status:      status,
		CreatedTime: time.Time{},
	}
	if ni.InstanceNo != nil && ncloud.StringValue(ni.InstanceNo) != "" {
		info.OwnerVM = irs.IID{NameId: ncloud.StringValue(ni.InstanceNo), SystemId: ncloud.StringValue(ni.InstanceNo)}
	}
	// PrivateIPs: primary + secondary
	var privateIPs []string
	if ni.Ip != nil && *ni.Ip != "" { privateIPs = append(privateIPs, *ni.Ip) }
	for _, secIP := range ni.SecondaryIpList {
		if secIP != nil && *secIP != "" { privateIPs = append(privateIPs, *secIP) }
	}
	info.PrivateIPs = privateIPs

	var acgs []irs.IID
	for _, acgNo := range ni.AccessControlGroupNoList { acgs = append(acgs, irs.IID{SystemId: ncloud.StringValue(acgNo)}) }
	info.SecurityGroupIIDs = acgs
	info.KeyValueList = []irs.KeyValue{
		{Key: "NetworkInterfaceNo", Value: ncloud.StringValue(ni.NetworkInterfaceNo)},
		{Key: "SubnetNo", Value: ncloud.StringValue(ni.SubnetNo)},
		{Key: "DeviceName", Value: ncloud.StringValue(ni.DeviceName)},
	}
	return info
}

// enrichNICInfo resolves Subnet name, VPC info, and OwnerVM name via API calls.
func (h *NcpVpcNICHandler) enrichNICInfo(info *irs.NICInfo, ni *vserver.NetworkInterface) {
	if info.SubnetIID.SystemId != "" && h.VpcClient != nil {
		subnetResp, err := h.VpcClient.V2Api.GetSubnetDetail(&vpc.GetSubnetDetailRequest{
			RegionCode: ncloud.String(h.RegionInfo.Region),
			SubnetNo:   ncloud.String(info.SubnetIID.SystemId),
		})
		if err == nil && len(subnetResp.SubnetList) > 0 {
			subnet := subnetResp.SubnetList[0]
			if subnet.SubnetName != nil { info.SubnetIID.NameId = *subnet.SubnetName }
			if subnet.VpcNo != nil {
				vpcResp, vpcErr := h.VpcClient.V2Api.GetVpcDetail(&vpc.GetVpcDetailRequest{
					RegionCode: ncloud.String(h.RegionInfo.Region),
					VpcNo:      subnet.VpcNo,
				})
				if vpcErr == nil && len(vpcResp.VpcList) > 0 {
					v := vpcResp.VpcList[0]
					nameId := ""; if v.VpcName != nil { nameId = *v.VpcName }
					sysId := ""; if v.VpcNo != nil { sysId = *v.VpcNo }
					info.VpcIID = irs.IID{NameId: nameId, SystemId: sysId}
				}
			}
		}
	}
	if info.OwnerVM.SystemId != "" {
		vmResp, err := h.VMClient.V2Api.GetServerInstanceList(&vserver.GetServerInstanceListRequest{
			RegionCode:           ncloud.String(h.RegionInfo.Region),
			ServerInstanceNoList: []*string{ncloud.String(info.OwnerVM.SystemId)},
		})
		if err == nil && len(vmResp.ServerInstanceList) > 0 {
			vm := vmResp.ServerInstanceList[0]
			if vm.ServerName != nil { info.OwnerVM.NameId = *vm.ServerName }
		}
	}
}

func ncpNICNameId(ni *vserver.NetworkInterface) string {
	if ni.NetworkInterfaceName != nil && ncloud.StringValue(ni.NetworkInterfaceName) != "" { return ncloud.StringValue(ni.NetworkInterfaceName) }
	return ncloud.StringValue(ni.NetworkInterfaceNo)
}

// AddPrivateIP assigns a secondary private IP to a NCP NIC using AssignSecondaryIps.
func (h *NcpVpcNICHandler) AddPrivateIP(nicIID irs.IID, privateIP string) (irs.NICInfo, error) {
	nicInfo, err := h.GetNIC(nicIID)
	if err != nil {
		return irs.NICInfo{}, fmt.Errorf("NcpVpcNICHandler.AddPrivateIP: failed to get NIC: %w", err)
	}
	if privateIP == "" {
		return irs.NICInfo{}, fmt.Errorf("NcpVpcNICHandler.AddPrivateIP: privateIP is required (NCP does not support auto-assignment)")
	}
	req := &vserver.AssignSecondaryIpsRequest{
		RegionCode:         ncloud.String(h.RegionInfo.Region),
		NetworkInterfaceNo: ncloud.String(nicInfo.IId.SystemId),
		SecondaryIpList:    []*string{ncloud.String(privateIP)},
	}
	resp, err := h.VMClient.V2Api.AssignSecondaryIps(req)
	if err != nil {
		return irs.NICInfo{}, fmt.Errorf("NcpVpcNICHandler.AddPrivateIP: AssignSecondaryIps failed: %w", err)
	}
	if len(resp.NetworkInterfaceList) == 0 {
		return irs.NICInfo{}, fmt.Errorf("NcpVpcNICHandler.AddPrivateIP: no NIC returned after assign")
	}
	ni := resp.NetworkInterfaceList[0]
	info := extractNcpNICInfo(ni)
	h.enrichNICInfo(&info, ni)
	return info, nil
}

// RemovePrivateIP unassigns a secondary private IP from a NCP NIC using UnassignSecondaryIps.
func (h *NcpVpcNICHandler) RemovePrivateIP(nicIID irs.IID, privateIP string) (bool, error) {
	nicInfo, err := h.GetNIC(nicIID)
	if err != nil {
		return false, fmt.Errorf("NcpVpcNICHandler.RemovePrivateIP: failed to get NIC: %w", err)
	}
	req := &vserver.UnassignSecondaryIpsRequest{
		RegionCode:         ncloud.String(h.RegionInfo.Region),
		NetworkInterfaceNo: ncloud.String(nicInfo.IId.SystemId),
		SecondaryIpList:    []*string{ncloud.String(privateIP)},
	}
	_, err = h.VMClient.V2Api.UnassignSecondaryIps(req)
	if err != nil {
		return false, fmt.Errorf("NcpVpcNICHandler.RemovePrivateIP: UnassignSecondaryIps failed: %w", err)
	}
	return true, nil
}

// GetNICOSConfigScript returns an empty string for NCP.
// NCP additional NICs are in private subnets with no direct internet route, so no
// policy-based routing table setup is required beyond standard DHCP on the interface.
func (h *NcpVpcNICHandler) GetNICOSConfigScript(nicIID irs.IID) (string, error) {
	return "", nil
}
