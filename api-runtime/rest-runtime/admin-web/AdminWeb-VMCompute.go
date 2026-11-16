package adminweb

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

//====================================== VM Accelerator Classification (VM page only)

// The VM page shows the accelerator class(CPU/GPU/TPU/NPU) of each VM by its VM spec.
// VM spec lookups call the CSP API, so this must not be used by the Dashboard.

type cachedVMSpecAccelerator struct {
	Accel     vmSpecAccelerator
	ExpiresAt time.Time
}

var vmSpecAcceleratorCache sync.Map

const vmSpecAcceleratorCacheTTL = time.Minute

func getVMSpecAccelerator(connConfig, specName string) (vmSpecAccelerator, error) {
	if strings.TrimSpace(specName) == "" {
		return vmSpecAccelerator{}, fmt.Errorf("VM spec name is empty")
	}

	cacheKey := connConfig + "\x00" + specName
	if cached, ok := vmSpecAcceleratorCache.Load(cacheKey); ok {
		entry := cached.(cachedVMSpecAccelerator)
		if time.Now().Before(entry.ExpiresAt) {
			return entry.Accel, nil
		}
		vmSpecAcceleratorCache.Delete(cacheKey)
	}

	endpoint := fmt.Sprintf("%s/spider/vmspec/%s?ConnectionName=%s", spiderBaseURL(), url.PathEscape(specName), url.QueryEscape(connConfig))
	resp, err := httpGetWithAuth(endpoint)
	if err != nil {
		return vmSpecAccelerator{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return vmSpecAccelerator{}, fmt.Errorf("VM spec lookup returned HTTP %d", resp.StatusCode)
	}

	var spec cres.VMSpecInfo
	if err := json.NewDecoder(resp.Body).Decode(&spec); err != nil {
		return vmSpecAccelerator{}, fmt.Errorf("decode VM spec: %w", err)
	}
	accel := classifyVMSpecAccelerator(&spec)
	vmSpecAcceleratorCache.Store(cacheKey, cachedVMSpecAccelerator{Accel: accel, ExpiresAt: time.Now().Add(vmSpecAcceleratorCacheTTL)})
	return accel, nil
}

// fetchVMAccelerators returns the accelerator class of each VM, keyed by VM NameId.
// VMs whose spec cannot be looked up are left out, and no badge is shown for them.
func fetchVMAccelerators(connConfig string, vms []*cres.VMInfo) map[string]vmSpecAccelerator {
	accels := make(map[string]vmSpecAccelerator, len(vms))
	for _, vm := range vms {
		if vm == nil {
			continue
		}
		accel, err := getVMSpecAccelerator(connConfig, vm.VMSpecName)
		if err != nil {
			cblog.Info(fmt.Sprintf("skip accelerator class of VM %s: %v", vm.IId.NameId, err))
			continue
		}
		accels[vm.IId.NameId] = accel
	}
	return accels
}
