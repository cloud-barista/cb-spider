package adminweb

import (
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/labstack/echo/v4"
)

//====================================== VMSpec

func VMSpec(c echo.Context) error {
	cblog.Info("call VMSpec()")

	connConfig := c.Param("ConnectConfig")
	if connConfig == "region not set" {
		htmlStr := `
            <html>
            <head>
                <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
				<style>
				th {
				  border: 1px solid lightgray;
				}
				td {
				  border: 1px solid lightgray;
				  border-radius: 4px;
				}
				</style>
                <script type="text/javascript">
                alert(connConfig)
                </script>
            </head>
            <body>
                <br>
                <br>
                <label style="font-size:24px;color:#606262;">&nbsp;&nbsp;&nbsp;Please select a Connection Configuration! (MENU: 2.CONNECTION)</label>   
            </body>
        `

		return c.HTML(http.StatusOK, htmlStr)
	}

	resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "vmspec")
	if err != nil {
		cblog.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var info struct {
		ResultList []*cres.VMSpecInfo `json:"vmspec"`
	}
	json.Unmarshal(resBody, &info)

	providerName, _ := getProviderName(connConfig)
	accSupported := isAcceleratorClassSupported(providerName)

	vmSpecs := make([]vmSpecView, 0, len(info.ResultList))
	accCounts := map[string]int{}
	for _, spec := range info.ResultList {
		if spec == nil {
			continue
		}
		view := vmSpecView{VMSpecInfo: spec}
		if accSupported {
			view.Accel = classifyVMSpecAccelerator(spec)
			accCounts[view.Accel.Class]++
		}
		vmSpecs = append(vmSpecs, view)
	}

	data := struct {
		ConnConfig   string
		ProviderName string
		AccSupported bool
		AccCounts    map[string]int
		VMSpecs      []vmSpecView
		APIUsername  string
		APIPassword  string
	}{
		ConnConfig:   connConfig,
		ProviderName: providerName,
		AccSupported: accSupported,
		AccCounts:    accCounts,
		VMSpecs:      vmSpecs,
		APIUsername:  os.Getenv("SPIDER_USERNAME"),
		APIPassword:  os.Getenv("SPIDER_PASSWORD"),
	}

	tmplPath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/vm-spec.html")
	tmpl, err := template.New("vm-spec.html").Funcs(template.FuncMap{
		"inc": func(i int) int {
			return i + 1
		},
		// GPU/accelerator values use "-1" (or empty) when the CSP does not provide them:
		// show "NA" instead of "-1" or a bare unit such as "TotalMemory: GB".
		"orNA":   valueOrNA,
		"gbOrNA": func(v string) string { return withUnitOrNA(v, "GB") },
	}).ParseFiles(tmplPath)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var result strings.Builder
	err = tmpl.Execute(&result, data)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.HTML(http.StatusOK, result.String())
}

//====================================== VMSpec Accelerator Classification

// Accelerator classes shown as badges and filter buttons on the VMSpec page.
const (
	accClassCPU = "CPU"
	accClassGPU = "GPU"
	accClassTPU = "TPU"
	accClassNPU = "NPU"
)

// vmSpecAccelerator describes the AI accelerator class of a VM spec.
// The class and chip come from VMSpecInfo.Accelerator filled by the drivers with the CSP API values.
type vmSpecAccelerator struct {
	Class string // CPU, GPU, TPU, NPU
	Chip  string // e.g. Inferentia2, CT6E, A10 (1/6)
}

type vmSpecView struct {
	*cres.VMSpecInfo
	Accel vmSpecAccelerator
}

// Currently, accelerator classification is provided only for AWS, Azure and GCP.
func isAcceleratorClassSupported(providerName string) bool {
	switch strings.ToUpper(providerName) {
	case "AWS", "AZURE", "GCP":
		return true
	}
	return false
}

// classifyVMSpecAccelerator picks the representative accelerator of a spec (NPU > TPU > GPU).
// Until the Accelerator structure is officially released, both structures are supported:
//   - VMSpecInfo.Accelerator: new structure (GPU/TPU/NPU), used first when provided
//   - VMSpecInfo.Gpu: legacy structure, used when Accelerator is empty
func classifyVMSpecAccelerator(spec *cres.VMSpecInfo) vmSpecAccelerator {
	if spec == nil {
		return vmSpecAccelerator{}
	}

	priority := map[cres.AcceleratorType]int{cres.AcceleratorNPU: 3, cres.AcceleratorTPU: 2, cres.AcceleratorGPU: 1}

	var selected *cres.AcceleratorInfo
	for i := range spec.Accelerator {
		acc := &spec.Accelerator[i]
		if selected == nil || priority[acc.Type] > priority[selected.Type] {
			selected = acc
		}
	}
	if selected != nil && priority[selected.Type] > 0 {
		chip := selected.Model
		if chip == "NA" {
			chip = ""
		}
		if chip != "" && selected.Fraction != "" {
			chip += " (" + selected.Fraction + ")"
		}
		return vmSpecAccelerator{Class: string(selected.Type), Chip: chip}
	}

	// drivers that do not provide Accelerator yet
	return classifyLegacyGpu(spec.Gpu)
}

// classifyLegacyGpu classifies a spec with the legacy VMSpecInfo.Gpu structure.
// Drivers fill the legacy structure differently for specs without GPU,
// e.g. no entry, {Count: "0"} or {Count: "-1", Model: "NA"}, so an entry is a GPU only when
// its count is positive, or when its count is unknown but its model or manufacturer is given.
func classifyLegacyGpu(gpus []cres.GpuInfo) vmSpecAccelerator {
	hasGpu := false
	var models []string
	for _, gpu := range gpus {
		model := knownValue(gpu.Model)
		count, err := strconv.Atoi(strings.TrimSpace(gpu.Count))
		if err == nil && count == 0 {
			continue
		}
		if (err != nil || count < 0) && model == "" && knownValue(gpu.Mfr) == "" {
			continue
		}
		hasGpu = true
		if model != "" && !slices.Contains(models, model) {
			models = append(models, model)
		}
	}
	if !hasGpu {
		return vmSpecAccelerator{Class: accClassCPU}
	}
	return vmSpecAccelerator{Class: accClassGPU, Chip: strings.Join(models, ", ")}
}

// knownValue returns the trimmed value, or "" when the driver marks it as not applicable.
func knownValue(value string) string {
	value = strings.TrimSpace(value)
	if strings.EqualFold(value, "NA") || value == "-1" {
		return ""
	}
	return value
}

// valueOrNA returns "NA" for a value the CSP does not provide ("", "-1", "NA"), else the value.
func valueOrNA(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || v == "-1" || strings.EqualFold(v, "NA") {
		return "NA"
	}
	return v
}

// withUnitOrNA appends the unit to a provided value, and returns "NA" without the unit otherwise.
func withUnitOrNA(v, unit string) string {
	if na := valueOrNA(v); na == "NA" {
		return na
	}
	return strings.TrimSpace(v) + " " + unit
}
