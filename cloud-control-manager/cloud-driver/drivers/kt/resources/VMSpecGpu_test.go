package resources

import (
	"testing"

	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

// Flavor names are taken from KT Cloud VPC (KR1/DX-M1).
func TestKtGpuInfoFromFlavorName(t *testing.T) {
	tests := []struct {
		name string
		ok   bool
		want irs.GpuInfo
	}{
		// model and count
		{"24x240.itl.a100-2.ai-voucher", true, irs.GpuInfo{Mfr: "NVIDIA", Model: "A100", Count: "2", MemSizeGB: "-1", TotalMemSizeGB: "-1"}},
		{"16x90.itl.A100-1", true, irs.GpuInfo{Mfr: "NVIDIA", Model: "A100", Count: "1", MemSizeGB: "-1", TotalMemSizeGB: "-1"}},
		{"32x180.itl.A100-2-", true, irs.GpuInfo{Mfr: "NVIDIA", Model: "A100", Count: "2", MemSizeGB: "-1", TotalMemSizeGB: "-1"}},
		{"8x32.amd.a100-1.hac", true, irs.GpuInfo{Mfr: "NVIDIA", Model: "A100", Count: "1", MemSizeGB: "-1", TotalMemSizeGB: "-1"}},
		{"4x8.itl.v100-1", true, irs.GpuInfo{Mfr: "NVIDIA", Model: "V100", Count: "1", MemSizeGB: "-1", TotalMemSizeGB: "-1"}},
		{"64x256.v100-2", true, irs.GpuInfo{Mfr: "NVIDIA", Model: "V100", Count: "2", MemSizeGB: "-1", TotalMemSizeGB: "-1"}},
		{"24x240.vpc-mss.v100s-3.ai-voucher", true, irs.GpuInfo{Mfr: "NVIDIA", Model: "V100S", Count: "3", MemSizeGB: "32", TotalMemSizeGB: "96"}},
		{"48x192.itl.T4-4.colorverse", true, irs.GpuInfo{Mfr: "NVIDIA", Model: "T4", Count: "4", MemSizeGB: "16", TotalMemSizeGB: "64"}},
		{"aidu_t4x5_72x720.gpu", true, irs.GpuInfo{Mfr: "NVIDIA", Model: "T4", Count: "5", MemSizeGB: "16", TotalMemSizeGB: "80"}},
		{"aidu-gpu-t4-5", true, irs.GpuInfo{Mfr: "NVIDIA", Model: "T4", Count: "5", MemSizeGB: "16", TotalMemSizeGB: "80"}},
		{"210326-t4-5-test", true, irs.GpuInfo{Mfr: "NVIDIA", Model: "T4", Count: "5", MemSizeGB: "16", TotalMemSizeGB: "80"}},
		// memory stated in the name, count as "<n>ea" or a trailing number
		{"128x768.itl.a100-80g-xe9680-8ea", true, irs.GpuInfo{Mfr: "NVIDIA", Model: "A100", Count: "8", MemSizeGB: "80", TotalMemSizeGB: "640"}},
		{"16x96.itl.a100-80g-xe9680-1ea", true, irs.GpuInfo{Mfr: "NVIDIA", Model: "A100", Count: "1", MemSizeGB: "80", TotalMemSizeGB: "80"}},
		{"rockplace_192x928_a100_80G_8", true, irs.GpuInfo{Mfr: "NVIDIA", Model: "A100", Count: "8", MemSizeGB: "80", TotalMemSizeGB: "640"}},
		// model known, count not stated
		{"rockplace_4x16_a100_80G", true, irs.GpuInfo{Mfr: "NVIDIA", Model: "A100", Count: "-1", MemSizeGB: "80", TotalMemSizeGB: "-1"}},
		{"rp.t4.2x2", true, irs.GpuInfo{Mfr: "NVIDIA", Model: "T4", Count: "-1", MemSizeGB: "16", TotalMemSizeGB: "-1"}},
		// GPU flavor of unknown model
		{"4x8.gpu", true, irs.GpuInfo{Mfr: "NA", Model: "NA", Count: "-1", MemSizeGB: "-1", TotalMemSizeGB: "-1"}},
		{"d1_pub_16x128.gp2", true, irs.GpuInfo{Mfr: "NA", Model: "NA", Count: "-1", MemSizeGB: "-1", TotalMemSizeGB: "-1"}},
		{"16x64.amd.sdgpu", true, irs.GpuInfo{Mfr: "NA", Model: "NA", Count: "-1", MemSizeGB: "-1", TotalMemSizeGB: "-1"}},
		{"vpc-gpu-test-9", true, irs.GpuInfo{Mfr: "NA", Model: "NA", Count: "-1", MemSizeGB: "-1", TotalMemSizeGB: "-1"}},
		// not a GPU flavor
		{"4x8.itl", false, irs.GpuInfo{}},
		{"16x16", false, irs.GpuInfo{}},
		{"4x8.datalake", false, irs.GpuInfo{}},
	}
	for _, tt := range tests {
		got, ok := ktGpuInfoFromFlavorName(tt.name)
		if ok != tt.ok || got != tt.want {
			t.Errorf("ktGpuInfoFromFlavorName(%q) = %+v, %v; want %+v, %v", tt.name, got, ok, tt.want, tt.ok)
		}
	}
}
