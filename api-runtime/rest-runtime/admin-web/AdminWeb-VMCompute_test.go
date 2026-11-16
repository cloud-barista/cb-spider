package adminweb

import (
	"testing"

	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

func TestClassifyVMSpecAccelerator(t *testing.T) {
	tests := []struct {
		name string
		spec *cres.VMSpecInfo
		want vmSpecAccelerator
	}{
		// new Accelerator structure
		{name: "accelerator GPU", spec: &cres.VMSpecInfo{Accelerator: []cres.AcceleratorInfo{{Type: cres.AcceleratorGPU, Model: "H100", Count: "8"}}}, want: vmSpecAccelerator{Class: accClassGPU, Chip: "H100"}},
		{name: "accelerator fractional GPU", spec: &cres.VMSpecInfo{Accelerator: []cres.AcceleratorInfo{{Type: cres.AcceleratorGPU, Model: "A10", Count: "1", Fraction: "1/6"}}}, want: vmSpecAccelerator{Class: accClassGPU, Chip: "A10 (1/6)"}},
		{name: "accelerator NPU over GPU", spec: &cres.VMSpecInfo{Accelerator: []cres.AcceleratorInfo{{Type: cres.AcceleratorGPU, Model: "T4"}, {Type: cres.AcceleratorNPU, Model: "Inferentia2"}}}, want: vmSpecAccelerator{Class: accClassNPU, Chip: "Inferentia2"}},
		{name: "accelerator first, legacy ignored", spec: &cres.VMSpecInfo{Accelerator: []cres.AcceleratorInfo{{Type: cres.AcceleratorTPU, Model: "CT6E"}}, Gpu: []cres.GpuInfo{{Count: "4", Model: "T4"}}}, want: vmSpecAccelerator{Class: accClassTPU, Chip: "CT6E"}},

		// legacy Gpu structure
		{name: "legacy no entry", spec: &cres.VMSpecInfo{}, want: vmSpecAccelerator{Class: accClassCPU}},
		{name: "legacy zero count", spec: &cres.VMSpecInfo{Gpu: []cres.GpuInfo{{Count: "0"}}}, want: vmSpecAccelerator{Class: accClassCPU}},
		{name: "legacy NA entry", spec: &cres.VMSpecInfo{Gpu: []cres.GpuInfo{{Count: "-1", Mfr: "NA", Model: "NA"}}}, want: vmSpecAccelerator{Class: accClassCPU}},
		{name: "legacy GPU", spec: &cres.VMSpecInfo{Gpu: []cres.GpuInfo{{Count: "8", Mfr: "NVIDIA", Model: "H100"}}}, want: vmSpecAccelerator{Class: accClassGPU, Chip: "H100"}},
		{name: "legacy GPU without model", spec: &cres.VMSpecInfo{Gpu: []cres.GpuInfo{{Count: "2", Model: "NA"}}}, want: vmSpecAccelerator{Class: accClassGPU}},
		{name: "legacy unknown count with model", spec: &cres.VMSpecInfo{Gpu: []cres.GpuInfo{{Count: "-1", Model: "T4"}}}, want: vmSpecAccelerator{Class: accClassGPU, Chip: "T4"}},
		{name: "legacy duplicated models", spec: &cres.VMSpecInfo{Gpu: []cres.GpuInfo{{Count: "1", Model: "T4"}, {Count: "1", Model: "T4"}}}, want: vmSpecAccelerator{Class: accClassGPU, Chip: "T4"}},

		{name: "nil spec", spec: nil, want: vmSpecAccelerator{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := classifyVMSpecAccelerator(test.spec); got != test.want {
				t.Fatalf("classifyVMSpecAccelerator() = %+v, want %+v", got, test.want)
			}
		})
	}
}
