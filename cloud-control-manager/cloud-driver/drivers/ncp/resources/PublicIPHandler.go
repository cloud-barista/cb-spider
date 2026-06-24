// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// NCP VPC Public IP Handler
//
// by CB-Spider Team, 2025.06.

package resources

import (
	"fmt"
	"time"

	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vserver"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NcpVpcPublicIPHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	VMClient       *vserver.APIClient
}

func (h *NcpVpcPublicIPHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(h.RegionInfo.Zone, call.PUBLICIP, "ListIID", "GetPublicIpInstanceList()")
	start := call.Start()

	req := &vserver.GetPublicIpInstanceListRequest{
		RegionCode: ncloud.String(h.RegionInfo.Region),
	}
	resp, err := h.VMClient.V2Api.GetPublicIpInstanceList(req)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	LoggingInfo(hiscallInfo, start)

	var iidList []*irs.IID
	for _, pip := range resp.PublicIpInstanceList {
		nameId := ncpPublicIPNameId(pip)
		iidList = append(iidList, &irs.IID{NameId: nameId, SystemId: ncloud.StringValue(pip.PublicIpInstanceNo)})
	}
	return iidList, nil
}

func (h *NcpVpcPublicIPHandler) CreatePublicIP(reqInfo irs.PublicIPInfo) (irs.PublicIPInfo, error) {
	hiscallInfo := GetCallLogScheme(h.RegionInfo.Zone, call.PUBLICIP, reqInfo.IId.NameId, "CreatePublicIpInstance()")
	start := call.Start()

	req := &vserver.CreatePublicIpInstanceRequest{
		RegionCode:          ncloud.String(h.RegionInfo.Region),
		PublicIpDescription: ncloud.String(reqInfo.IId.NameId),
	}

	resp, err := h.VMClient.V2Api.CreatePublicIpInstance(req)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	if len(resp.PublicIpInstanceList) == 0 {
		return irs.PublicIPInfo{}, fmt.Errorf("CreatePublicIpInstance returned empty list")
	}

	created := resp.PublicIpInstanceList[0]
	systemId := ncloud.StringValue(created.PublicIpInstanceNo)

	// NCP is async: poll until the IP reaches a stable operation state before returning.
	// Attempting DeletePublicIpInstance while the IP is still initializing returns error 1080101.
	for i := 0; i < 30; i++ {
		pollResp, pollErr := h.VMClient.V2Api.GetPublicIpInstanceList(&vserver.GetPublicIpInstanceListRequest{
			RegionCode:             ncloud.String(h.RegionInfo.Region),
			PublicIpInstanceNoList: []*string{ncloud.String(systemId)},
		})
		if pollErr == nil && len(pollResp.PublicIpInstanceList) > 0 {
			statusCode := ""
			if pollResp.PublicIpInstanceList[0].PublicIpInstanceStatus != nil {
				statusCode = ncloud.StringValue(pollResp.PublicIpInstanceList[0].PublicIpInstanceStatus.Code)
			}
			if statusCode != "" && statusCode != "INIT" && statusCode != "CREAT" {
				break
			}
		}
		time.Sleep(2 * time.Second)
	}

	info := h.extractPublicIPInfo(created)
	info.IId.NameId = reqInfo.IId.NameId
	return info, nil
}

func (h *NcpVpcPublicIPHandler) ListPublicIP() ([]*irs.PublicIPInfo, error) {
	hiscallInfo := GetCallLogScheme(h.RegionInfo.Zone, call.PUBLICIP, "All", "GetPublicIpInstanceList()")
	start := call.Start()

	req := &vserver.GetPublicIpInstanceListRequest{
		RegionCode: ncloud.String(h.RegionInfo.Region),
	}
	resp, err := h.VMClient.V2Api.GetPublicIpInstanceList(req)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	LoggingInfo(hiscallInfo, start)

	var infoList []*irs.PublicIPInfo
	for _, pip := range resp.PublicIpInstanceList {
		info := h.extractPublicIPInfo(pip)
		infoList = append(infoList, &info)
	}
	if infoList == nil {
		infoList = []*irs.PublicIPInfo{}
	}
	return infoList, nil
}

func (h *NcpVpcPublicIPHandler) GetPublicIP(publicIPIID irs.IID) (irs.PublicIPInfo, error) {
	hiscallInfo := GetCallLogScheme(h.RegionInfo.Zone, call.PUBLICIP, publicIPIID.NameId, "GetPublicIpInstanceList()")
	start := call.Start()

	req := &vserver.GetPublicIpInstanceListRequest{
		RegionCode: ncloud.String(h.RegionInfo.Region),
	}
	if publicIPIID.SystemId != "" {
		req.PublicIpInstanceNoList = []*string{ncloud.String(publicIPIID.SystemId)}
	}

	resp, err := h.VMClient.V2Api.GetPublicIpInstanceList(req)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	for _, pip := range resp.PublicIpInstanceList {
		if publicIPIID.SystemId != "" && ncloud.StringValue(pip.PublicIpInstanceNo) == publicIPIID.SystemId {
			info := h.extractPublicIPInfo(pip)
			if publicIPIID.NameId != "" {
				info.IId.NameId = publicIPIID.NameId
			}
			return info, nil
		}
		if ncloud.StringValue(pip.PublicIpDescription) == publicIPIID.NameId {
			info := h.extractPublicIPInfo(pip)
			info.IId.NameId = publicIPIID.NameId
			return info, nil
		}
	}

	return irs.PublicIPInfo{}, fmt.Errorf("PublicIP not found: %s", publicIPIID.NameId)
}

func (h *NcpVpcPublicIPHandler) DeletePublicIP(publicIPIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(h.RegionInfo.Zone, call.PUBLICIP, publicIPIID.NameId, "DeletePublicIpInstance()")
	start := call.Start()

	systemId := publicIPIID.SystemId
	if systemId == "" {
		info, err := h.GetPublicIP(publicIPIID)
		if err != nil {
			return false, err
		}
		systemId = info.IId.SystemId
	}

	req := &vserver.DeletePublicIpInstanceRequest{
		RegionCode:         ncloud.String(h.RegionInfo.Region),
		PublicIpInstanceNo: ncloud.String(systemId),
	}

	_, err := h.VMClient.V2Api.DeletePublicIpInstance(req)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	LoggingInfo(hiscallInfo, start)

	return true, nil
}

func (h *NcpVpcPublicIPHandler) extractPublicIPInfo(pip *vserver.PublicIpInstance) irs.PublicIPInfo {
	nameId := ncpPublicIPNameId(pip)
	status := irs.PublicIPAvailable
	if pip.ServerInstanceNo != nil && ncloud.StringValue(pip.ServerInstanceNo) != "" {
		status = irs.PublicIPAssociated
	}

	info := irs.PublicIPInfo{
		IId:             irs.IID{NameId: nameId, SystemId: ncloud.StringValue(pip.PublicIpInstanceNo)},
		PublicIPAddress: ncloud.StringValue(pip.PublicIp),
		Status:          status,
		CreatedTime:     time.Time{},
	}

	if pip.ServerInstanceNo != nil && ncloud.StringValue(pip.ServerInstanceNo) != "" {
		instanceNo := ncloud.StringValue(pip.ServerInstanceNo)
		vmName := instanceNo
		vmResp, err := h.VMClient.V2Api.GetServerInstanceList(&vserver.GetServerInstanceListRequest{
			RegionCode:           ncloud.String(h.RegionInfo.Region),
			ServerInstanceNoList: []*string{ncloud.String(instanceNo)},
		})
		if err == nil && len(vmResp.ServerInstanceList) > 0 && vmResp.ServerInstanceList[0].ServerName != nil {
			vmName = *vmResp.ServerInstanceList[0].ServerName
		}
		info.OwnedVM = irs.IID{NameId: vmName, SystemId: instanceNo}
	}

	kvList := []irs.KeyValue{
		{Key: "PublicIpInstanceNo", Value: ncloud.StringValue(pip.PublicIpInstanceNo)},
		{Key: "PrivateIp", Value: ncloud.StringValue(pip.PrivateIp)},
	}
	if pip.PublicIpInstanceStatus != nil {
		kvList = append(kvList, irs.KeyValue{Key: "Status", Value: ncloud.StringValue(pip.PublicIpInstanceStatus.Code)})
	}
	info.KeyValueList = kvList

	return info
}

func ncpPublicIPNameId(pip *vserver.PublicIpInstance) string {
	if pip.PublicIpDescription != nil && ncloud.StringValue(pip.PublicIpDescription) != "" {
		return ncloud.StringValue(pip.PublicIpDescription)
	}
	return ncloud.StringValue(pip.PublicIpInstanceNo)
}

// AssociatePublicIP associates a Public IP with an NCP server instance.
func (h *NcpVpcPublicIPHandler) AssociatePublicIP(publicIPIID irs.IID, vmIID irs.IID, nicIID irs.IID, privateIP string) (irs.PublicIPInfo, error) {
	hiscallInfo := GetCallLogScheme(h.RegionInfo.Zone, call.PUBLICIP, publicIPIID.NameId, "AssociatePublicIpWithServerInstance()")
	start := call.Start()

	pipInfo, err := h.GetPublicIP(publicIPIID)
	if err != nil {
		return irs.PublicIPInfo{}, err
	}

	serverNo := vmIID.SystemId
	if serverNo == "" {
		serverNo = vmIID.NameId
	}
	if serverNo == "" {
		return irs.PublicIPInfo{}, fmt.Errorf("AssociatePublicIP: vmIID (SystemId or NameId) is required for NCP")
	}

	req := &vserver.AssociatePublicIpWithServerInstanceRequest{
		RegionCode:         ncloud.String(h.RegionInfo.Region),
		PublicIpInstanceNo: ncloud.String(pipInfo.IId.SystemId),
		ServerInstanceNo:   ncloud.String(serverNo),
	}
	_, err = h.VMClient.V2Api.AssociatePublicIpWithServerInstance(req)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	return h.GetPublicIP(publicIPIID)
}

// DisassociatePublicIP disassociates a Public IP from an NCP server instance.
func (h *NcpVpcPublicIPHandler) DisassociatePublicIP(publicIPIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(h.RegionInfo.Zone, call.PUBLICIP, publicIPIID.NameId, "DisassociatePublicIpFromServerInstance()")
	start := call.Start()

	pipInfo, err := h.GetPublicIP(publicIPIID)
	if err != nil {
		return false, err
	}
	if pipInfo.Status != irs.PublicIPAssociated {
		return false, fmt.Errorf("PublicIP %s is not associated", publicIPIID.NameId)
	}

	req := &vserver.DisassociatePublicIpFromServerInstanceRequest{
		RegionCode:         ncloud.String(h.RegionInfo.Region),
		PublicIpInstanceNo: ncloud.String(pipInfo.IId.SystemId),
	}
	_, err = h.VMClient.V2Api.DisassociatePublicIpFromServerInstance(req)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	LoggingInfo(hiscallInfo, start)

	return true, nil
}
