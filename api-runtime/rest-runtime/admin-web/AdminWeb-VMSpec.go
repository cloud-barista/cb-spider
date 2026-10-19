package adminweb

import (
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
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
func classifyVMSpecAccelerator(spec *cres.VMSpecInfo) vmSpecAccelerator {
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
	if len(spec.Gpu) > 0 {
		return vmSpecAccelerator{Class: accClassGPU}
	}
	return vmSpecAccelerator{Class: accClassCPU}
}
