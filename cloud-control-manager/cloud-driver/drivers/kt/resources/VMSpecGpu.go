// KT Cloud VPC Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista

package resources

import (
	"regexp"
	"strconv"
	"strings"

	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

//====================================== GPU info from KT VPC flavor names
//
// The KT Cloud VPC flavor API returns no GPU details: a flavor has only ID, Name, VCPUs, RAM,
// Disk, Ephemeral, IsPublic and RxTxFactor (no extra_specs). The only source of GPU information
// is the flavor name, which KT composes as "<vCPU>x<RAM GB>.<CPU vendor>.<GPU model>-<count>...".
// Examples of KT flavor names (KR1/DX-M1, 2026-11):
//
//   24x240.itl.a100-2.ai-voucher        => NVIDIA A100 x 2
//   16x96.itl.a100-80g-xe9680-1ea       => NVIDIA A100 80GB x 1 ("<n>ea" is the device count)
//   rockplace_192x928_a100_80G_8        => NVIDIA A100 80GB x 8
//   4x8.itl.v100-1, 24x240.vpc-mss.v100s-3.ai-voucher => NVIDIA V100 x 1, NVIDIA V100S x 3
//   48x192.itl.T4-4.colorverse, aidu_t4x5_72x720.gpu  => NVIDIA T4 x 4, NVIDIA T4 x 5
//   4x8.gpu, 16x128.gp2, 16x64.amd.sdgpu, vpc-gpu-test => a GPU flavor of unknown model
//
// Because this is inferred from names, values that the name does not tell are left as "-1"/"NA"
// rather than guessed. Update ktGpuModels below when KT adds flavors of a new GPU model.

// ktGpuModel describes a GPU model that may appear in KT flavor names.
type ktGpuModel struct {
	token     string // lower-case token in the flavor name, e.g. "a100"
	model     string // model name reported in GpuInfo/AcceleratorInfo
	memSizeGB string // memory per device when the model has a single memory size, else "-1"
}

// ktGpuModels is ordered so that a longer token is matched before its prefix ("v100s" before "v100").
// A100 (40GB or 80GB) and V100 (16GB or 32GB) come in two memory sizes, so their memory is known
// only when the name states it (e.g. "a100-80g").
var ktGpuModels = []ktGpuModel{
	{token: "a100", model: "A100", memSizeGB: "-1"},
	{token: "v100s", model: "V100S", memSizeGB: "32"},
	{token: "v100", model: "V100", memSizeGB: "-1"},
	{token: "t4", model: "T4", memSizeGB: "16"},
}

var (
	// ktGpuMemRegex matches a memory size right after the model token: "-80g", "_80G".
	ktGpuMemRegex = regexp.MustCompile(`^[-_](\d+)g`)
	// ktGpuCountEaRegex matches an explicit device count anywhere after the model: "-8ea".
	ktGpuCountEaRegex = regexp.MustCompile(`[-_](\d+)ea\b`)
	// ktGpuCountRegex matches a device count right after the model (and memory): "-2", "_8", "x5".
	ktGpuCountRegex = regexp.MustCompile(`^[-_x](\d+)(?:$|[^\dg])`)
	// ktGenericGpuRegex matches flavors marked as GPU without a model: "gpu", "gp2", "sdgpu".
	ktGenericGpuRegex = regexp.MustCompile(`gpu|gp\d`)
)

// ktGpuInfoFromFlavorName returns the GPU info inferred from a KT VPC flavor name,
// or ok=false when the name does not indicate a GPU flavor.
func ktGpuInfoFromFlavorName(flavorName string) (gpu irs.GpuInfo, ok bool) {
	name := strings.ToLower(flavorName)

	for _, m := range ktGpuModels {
		rest, found := ktTokenSuffix(name, m.token)
		if !found {
			continue
		}

		gpu = irs.GpuInfo{Mfr: "NVIDIA", Model: m.model, Count: "-1", MemSizeGB: m.memSizeGB, TotalMemSizeGB: "-1"}

		// memory per device stated in the name, e.g. "a100-80g"
		if mem := ktGpuMemRegex.FindStringSubmatch(rest); mem != nil {
			gpu.MemSizeGB = mem[1]
			rest = rest[len(mem[0]):]
		}

		// device count: an explicit "<n>ea" wins, then a number right after the model/memory
		if cnt := ktGpuCountEaRegex.FindStringSubmatch(rest); cnt != nil {
			gpu.Count = cnt[1]
		} else if cnt := ktGpuCountRegex.FindStringSubmatch(rest); cnt != nil {
			gpu.Count = cnt[1]
		}

		gpu.TotalMemSizeGB = ktTotalMemSizeGB(gpu.Count, gpu.MemSizeGB)
		return gpu, true
	}

	if ktGenericGpuRegex.MatchString(name) {
		return irs.GpuInfo{Mfr: "NA", Model: "NA", Count: "-1", MemSizeGB: "-1", TotalMemSizeGB: "-1"}, true
	}
	return irs.GpuInfo{}, false
}

// ktTokenSuffix finds a model token that stands as its own name part (at the start or after
// '.', '-' or '_') and returns the rest of the name after it.
// This keeps e.g. "t4" from matching inside an unrelated word.
func ktTokenSuffix(name, token string) (string, bool) {
	for from := 0; from < len(name); {
		idx := strings.Index(name[from:], token)
		if idx < 0 {
			return "", false
		}
		idx += from
		if idx == 0 || strings.ContainsRune("._-", rune(name[idx-1])) {
			return name[idx+len(token):], true
		}
		from = idx + 1
	}
	return "", false
}

// ktTotalMemSizeGB returns count x memory per device, or "-1" when either is unknown.
func ktTotalMemSizeGB(count, memSizeGB string) string {
	c, err1 := strconv.Atoi(count)
	m, err2 := strconv.Atoi(memSizeGB)
	if err1 != nil || err2 != nil || c < 1 || m < 1 {
		return "-1"
	}
	return strconv.Itoa(c * m)
}

// ktAcceleratorFromGpu converts the inferred GPU info to the AI accelerator form.
func ktAcceleratorFromGpu(gpu irs.GpuInfo) irs.AcceleratorInfo {
	return irs.AcceleratorInfo{
		Type:           irs.AcceleratorGPU,
		Mfr:            gpu.Mfr,
		Model:          gpu.Model,
		Count:          gpu.Count,
		MemSizeGB:      gpu.MemSizeGB,
		TotalMemSizeGB: gpu.TotalMemSizeGB,
	}
}
