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

	h.waitForPublicIPStable(systemId)

	info := h.extractPublicIPInfo(created)
	info.IId.NameId = reqInfo.IId.NameId
	return info, nil
}

// waitForPublicIPStable polls until the given Public IP (a) leaves the async
// INIT/CREAT state NCP puts it in right after creation or after being
// auto-assigned at VM-creation time, AND (b) has no in-flight
// PublicIpInstanceOperation ("Public IP instance operation", per the NCP SDK
// field doc) - e.g. a just-completed
// Associate/Disassociate is still being applied. Any operation against the
// IP while either condition holds (Disassociate, Delete, ...) fails with NCP
// error 1080101 ("This is not an authorized IP in operation") - the
// "operation" in that message is this very field, not a caller-IP ACL.
// Best-effort - a poll error or timeout is silently ignored and the caller
// proceeds anyway, same as CreatePublicIP already did before this was
// extracted into a shared helper.
func (h *NcpVpcPublicIPHandler) waitForPublicIPStable(systemId string) {
	for i := 0; i < 30; i++ {
		pollResp, pollErr := h.VMClient.V2Api.GetPublicIpInstanceList(&vserver.GetPublicIpInstanceListRequest{
			RegionCode:             ncloud.String(h.RegionInfo.Region),
			PublicIpInstanceNoList: []*string{ncloud.String(systemId)},
		})
		if pollErr == nil && len(pollResp.PublicIpInstanceList) > 0 {
			pip := pollResp.PublicIpInstanceList[0]
			statusCode := ""
			if pip.PublicIpInstanceStatus != nil {
				statusCode = ncloud.StringValue(pip.PublicIpInstanceStatus.Code)
			}
			statusStable := statusCode != "" && statusCode != "INIT" && statusCode != "CREAT"
			opClear := pip.PublicIpInstanceOperation == nil || ncloud.StringValue(pip.PublicIpInstanceOperation.Code) == ""
			if statusStable && opClear {
				return
			}
		}
		time.Sleep(2 * time.Second)
	}
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

// RemoveDefaultPublicIP removes whatever Public IP Instance is currently
// associated with the VM, discovered live via GetPublicIpInstanceList
// (regardless of whether it was ever tracked as a separate CB-Spider
// PublicIP resource) - then disassociates and deletes it via the existing
// DisassociatePublicIP/DeletePublicIP methods. Works on a running VM - no
// stop/restart required.
//
// vmIID here is a driver-level IID rebuilt by getDriverIID() from the VM's
// stored SystemId, so vmIID.NameId is NOT the VM's real server name (it is
// derived from SystemId, see api-runtime/common-runtime/CommonManager.go).
// Matching by ServerName would therefore filter on a bogus value, so the
// association is resolved by ServerInstanceNo instead, same as
// AssociatePublicIP/DisassociatePublicIP above.
func (h *NcpVpcPublicIPHandler) RemoveDefaultPublicIP(vmIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(h.RegionInfo.Zone, call.PUBLICIP, vmIID.NameId, "RemoveDefaultPublicIP()")
	start := call.Start()

	serverInstanceNo := vmIID.SystemId
	if serverInstanceNo == "" {
		serverInstanceNo = vmIID.NameId
	}
	if serverInstanceNo == "" {
		err := fmt.Errorf("RemoveDefaultPublicIP: vmIID (SystemId or NameId) is required for NCP")
		cblogger.Error(err)
		return false, err
	}

	req := &vserver.GetPublicIpInstanceListRequest{
		RegionCode:   ncloud.String(h.RegionInfo.Region),
		IsAssociated: ncloud.Bool(true),
	}
	resp, err := h.VMClient.V2Api.GetPublicIpInstanceList(req)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}

	var attached []*vserver.PublicIpInstance
	for _, pip := range resp.PublicIpInstanceList {
		if ncloud.StringValue(pip.ServerInstanceNo) == serverInstanceNo {
			attached = append(attached, pip)
		}
	}
	if len(attached) == 0 {
		err := fmt.Errorf("no PublicIP found attached to VM %s", vmIID.NameId)
		cblogger.Error(err)
		return false, err
	}

	for _, pip := range attached {
		systemId := ncloud.StringValue(pip.PublicIpInstanceNo)
		// A PublicIP auto-assigned at VM-creation time (AssociateWithPublicIp)
		// is commonly still mid-async-setup (INIT/CREAT) by the time a caller
		// can react to the VM going Running - see waitForPublicIPStable.
		h.waitForPublicIPStable(systemId)

		pipIID := irs.IID{SystemId: systemId}
		if _, err := h.DisassociatePublicIP(pipIID); err != nil {
			cblogger.Error(err)
			LoggingError(hiscallInfo, err)
			return false, err
		}

		// Disassociating triggers its own async state transition, so the IP
		// can be back in a non-stable operation state by the time Delete
		// runs - wait again rather than assuming the earlier wait still
		// covers it (this is the exact case CreatePublicIP's comment above
		// documents: "Attempting DeletePublicIpInstance while the IP is
		// still initializing returns error 1080101").
		h.waitForPublicIPStable(systemId)

		if _, err := h.DeletePublicIP(pipIID); err != nil {
			cblogger.Error(err)
			LoggingError(hiscallInfo, err)
			return false, err
		}
	}
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	LoggingInfo(hiscallInfo, start)

	return true, nil
}
