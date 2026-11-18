package adminweb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/labstack/echo/v4"
)

//====================================== VM Accelerator Classification (VM page only)

// The VM page shows the accelerator class(CPU/GPU/TPU/NPU) of each VM by its VM spec.
// VM spec lookups call the CSP API, which can be slow on some CSPs, so the VM page renders
// first and loads the badges afterwards through VMAccelerators. This must not be used by the Dashboard.

type cachedVMSpecAccelerator struct {
	Accel     vmSpecAccelerator
	ExpiresAt time.Time
}

var vmSpecAcceleratorCache sync.Map

// VM spec information rarely changes, so the class is kept for an hour.
const vmSpecAcceleratorCacheTTL = time.Hour

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

// fetchSpecAccelerators returns the accelerator class of each VM spec, keyed by spec name.
// Distinct specs are looked up in parallel. Specs that cannot be looked up are left out,
// and no badge is shown for them.
func fetchSpecAccelerators(connConfig string, specNames []string) map[string]vmSpecAccelerator {
	accels := make(map[string]vmSpecAccelerator, len(specNames))
	var mu sync.Mutex
	var wg sync.WaitGroup
	seen := make(map[string]bool, len(specNames))
	for _, specName := range specNames {
		specName = strings.TrimSpace(specName)
		if specName == "" || seen[specName] {
			continue
		}
		seen[specName] = true
		wg.Add(1)
		go func(specName string) {
			defer wg.Done()
			accel, err := getVMSpecAccelerator(connConfig, specName)
			if err != nil {
				cblog.Info(fmt.Sprintf("skip accelerator class of VM spec %s: %v", specName, err))
				return
			}
			mu.Lock()
			accels[specName] = accel
			mu.Unlock()
		}(specName)
	}
	wg.Wait()
	return accels
}

// VMAccelerators returns the accelerator class of the given VM specs as JSON, keyed by spec name.
// GET /spider/adminweb/vmaccelerator/:ConnectConfig?specs=spec1,spec2
func VMAccelerators(c echo.Context) error {
	connConfig := c.Param("ConnectConfig")
	specs := strings.Split(c.QueryParam("specs"), ",")
	return c.JSON(http.StatusOK, fetchSpecAccelerators(connConfig, specs))
}
