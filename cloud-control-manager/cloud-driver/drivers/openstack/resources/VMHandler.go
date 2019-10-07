// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by hyokyung.kim@innogrid.co.kr, 2019.07.

package resources

import (
	//"fmt"
	cblog "github.com/cloud-barista/cb-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/startstop"
	"github.com/rackspace/gophercloud/openstack/compute/v2/servers"
	"github.com/rackspace/gophercloud/pagination"
	"github.com/sirupsen/logrus"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

// modified by powerkim, 2019.07.29
type OpenStackVMHandler struct {
	Client *gophercloud.ServiceClient
}

// modified by powerkim, 2019.07.29
func (vmHandler *OpenStackVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (irs.VMInfo, error) {

	// Add Server Create Options
	serverCreateOpts := servers.CreateOpts{
		Name:      vmReqInfo.Name,
		ImageRef:  vmReqInfo.ImageInfo.Id,
		FlavorRef: vmReqInfo.SpecID,
		Networks: []servers.Network{
			{UUID: vmReqInfo.VNetworkInfo.Id},
		},
		SecurityGroups: []string{
			vmReqInfo.SecurityInfo.Name,
		},
		//ServiceClient: vmHandler.Client,
	}

	// Add KeyPair
	createOpts := keypairs.CreateOptsExt{
		CreateOptsBuilder: serverCreateOpts,
		KeyName:           vmReqInfo.KeyPairInfo.Name,
	}

	server, err := servers.Create(vmHandler.Client, createOpts).Extract()
	if err != nil {
		return irs.VMInfo{}, err
	}

	vmInfo := mappingServerInfo(*server)
	return vmInfo, nil
}

func (vmHandler *OpenStackVMHandler) SuspendVM(vmID string) {
	err := startstop.Stop(vmHandler.Client, vmID).Err
	if err != nil {
		cblogger.Error(err)
	}
}

func (vmHandler *OpenStackVMHandler) ResumeVM(vmID string) {
	err := startstop.Start(vmHandler.Client, vmID).Err
	if err != nil {
		cblogger.Error(err)
	}
}

func (vmHandler *OpenStackVMHandler) RebootVM(vmID string) {
	/*rebootOpts := servers.RebootOpts{
		Type: servers.SoftReboot,
		//Type: servers.HardReboot,
	}*/
	rebootOpts := servers.SoftReboot
	err := servers.Reboot(vmHandler.Client, vmID, rebootOpts).ExtractErr()
	if err != nil {
		cblogger.Error(err)
	}
}

func (vmHandler *OpenStackVMHandler) TerminateVM(vmID string) {
	err := servers.Delete(vmHandler.Client, vmID).ExtractErr()
	if err != nil {
		cblogger.Error(err)
	}
}

func (vmHandler *OpenStackVMHandler) ListVMStatus() []*irs.VMStatusInfo {
	var vmStatusList []*irs.VMStatusInfo

	pager := servers.List(vmHandler.Client, nil)
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		// Get VM Status
		list, err := servers.ExtractServers(page)
		if err != nil {
			return false, err
		}
		// Add to List
		for _, s := range list {
			vmStatus := irs.VMStatus(s.Status)
			vmStatusInfo := irs.VMStatusInfo{
				VmId:     s.ID,
				VmStatus: vmStatus,
			}
			vmStatusList = append(vmStatusList, &vmStatusInfo)
		}
		return true, nil
	})
	if err != nil {
		cblogger.Error(err)
	}

	return vmStatusList
}

func (vmHandler *OpenStackVMHandler) GetVMStatus(vmID string) irs.VMStatus {
	serverResult, err := servers.Get(vmHandler.Client, vmID).Extract()
	if err != nil {
		cblogger.Error(err)
	}
	return irs.VMStatus(serverResult.Status)
}

func (vmHandler *OpenStackVMHandler) ListVM() []*irs.VMInfo {
	var vmList []*irs.VMInfo

	pager := servers.List(vmHandler.Client, nil)
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		// Get Servers
		list, err := servers.ExtractServers(page)
		if err != nil {
			return false, err
		}
		// Add to List
		for _, s := range list {
			vmInfo := mappingServerInfo(s)
			vmList = append(vmList, &vmInfo)
		}
		return true, nil
	})
	if err != nil {
		cblogger.Error(err)
	}

	return vmList
}

func (vmHandler *OpenStackVMHandler) GetVM(vmID string) irs.VMInfo {
	serverResult, err := servers.Get(vmHandler.Client, vmID).Extract()
	if err != nil {
		cblogger.Info(err)
		return irs.VMInfo{}
	}

	vmInfo := mappingServerInfo(*serverResult)
	return vmInfo
}

func mappingServerInfo(server servers.Server) irs.VMInfo {

	// Get Default VM Info
	vmInfo := irs.VMInfo{
		Name: server.Name,
		Id:   server.ID,
		//StartTime: server.Created,
		KeyPairID: server.KeyName,
	}

	if len(server.Image) != 0 {
		vmInfo.ImageID = server.Image["id"].(string)
	}
	if len(server.Flavor) != 0 {
		vmInfo.SpecID = server.Flavor["id"].(string)
	}

	// Get VM Subnet, Address Info
	for k, subnet := range server.Addresses {
		vmInfo.SubNetworkID = k
		for _, addr := range subnet.([]interface{}) {
			addrMap := addr.(map[string]interface{})
			if addrMap["OS-EXT-IPS:type"] == "floating" {
				vmInfo.PrivateIP = addrMap["addr"].(string)
			} else if addrMap["OS-EXT-IPS:type"] == "fixed" {
				vmInfo.PublicIP = addrMap["addr"].(string)
			}
		}
	}

	return vmInfo
}
