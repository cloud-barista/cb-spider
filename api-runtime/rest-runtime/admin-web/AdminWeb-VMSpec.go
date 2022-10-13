// Cloud Info Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.06.

package adminweb

import (
	"fmt"

	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	"strconv"

	"encoding/json"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)


//====================================== VMSpec

// number, VMSpec Name, VCPU, Memory, GPU, KeyValueList
func makeVMSpecTRList_html(bgcolor string, height string, fontSize string, infoList []*cres.VMSpecInfo) string {
	if bgcolor == "" {
		bgcolor = "#FFFFFF"
	}
	if height == "" {
		height = "30"
	}
	if fontSize == "" {
		fontSize = "2"
	}

	// make base TR frame for info list
	strTR := fmt.Sprintf(`
                <tr bgcolor="%s" align="center" height="%s">
                    <td>
                            <font size=%s>$$NUM$$</font>
                    </td>
                    <td>
                            <font size=%s>$$VMSPECNAME$$</font>
                    </td>
                    <td align="left">
                            <font size=%s>$$VCPUINFO$$</font>
                    </td>
                    <td>
                            <font size=%s>$$MEMINFO$$ MB</font>
                    </td>
                    <td align="left">
                            <font size=%s>$$GPUINFO$$</font>
                    </td>
                    <td align="left">
                            <font size=%s>$$ADDITIONALINFO$$</font>
                    </td>
                </tr>
                `, bgcolor, height, fontSize, fontSize, fontSize, fontSize, fontSize, fontSize)

	strData := ""
	// set data and make TR list
	for i, one := range infoList {
		str := strings.ReplaceAll(strTR, "$$NUM$$", strconv.Itoa(i+1))
		str = strings.ReplaceAll(str, "$$VMSPECNAME$$", one.Name)
		// VCPU Info: count, GHz
		vcpuInfo := "&nbsp;* Count: " + one.VCpu.Count + "<br>"
		vcpuInfo += "&nbsp;* Clock: " + one.VCpu.Clock + "GHz" + "<br>"
		str = strings.ReplaceAll(str, "$$VCPUINFO$$", vcpuInfo)

		// Mem Info
		str = strings.ReplaceAll(str, "$$MEMINFO$$", one.Mem)

		// GPU Info: Mfr, Model, Mem, Count
		gpuInfo := ""
		for _, gpu := range one.Gpu {
			gpuInfo += "&nbsp;* Mfr: " + gpu.Mfr + "<br>"
			gpuInfo += "&nbsp;* Model: " + gpu.Model + "<br>"
			gpuInfo += "&nbsp;* Memory: " + gpu.Mem + " MB" + "<br>"
			gpuInfo += "&nbsp;* Count: " + gpu.Count + "<br><br>"
		}
		str = strings.ReplaceAll(str, "$$GPUINFO$$", gpuInfo)

		// for KeyValueList
		strKeyList := ""
		for _, kv := range one.KeyValueList {
			strKeyList += kv.Key + ":" + kv.Value + ", "
		}
		strKeyList = strings.TrimRight(strKeyList, ", ")
		str = strings.ReplaceAll(str, "$$ADDITIONALINFO$$", strKeyList)

		strData += str
	}

	return strData
}

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

	// make page header
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
                </head>

                <body>
        <br>
                    <table border="0" bordercolordark="#F8F8FF" cellpadding="0" cellspacing="1" bgcolor="#FFFFFF"  style="font-size:small;">
                `

	// (3) make Table Header TR
	nameWidthList := []NameWidth{
		{"VMSpec Name", "200"},
		{"VCPU", "300"},
		{"Memory", "200"},
		{"GPU", "300"},
		{"Additional Info", "300"},
	}
	htmlStr += makeTitleTRList_html("#DDDDDD", "2", nameWidthList, false)

	// (4) make TR list with info list
	// (4-1) get info list

	// client logging
	htmlStr += genLoggingGETURL(connConfig, "vmspec")

	resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "vmspec")
	if err != nil {
		cblog.Error(err)
		// client logging
                htmlStr += genLoggingResult(err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// client logging
	htmlStr += genLoggingResult(string(resBody[:len(resBody)-1]))

	var info struct {
		ResultList []*cres.VMSpecInfo `json:"vmspec"`
	}
	json.Unmarshal(resBody, &info)

	// (4-2) make TR list with info list
	htmlStr += makeVMSpecTRList_html("", "", "", info.ResultList)

	// make page tail
	htmlStr += `
                    </table>
            <hr>
                </body>
                </html>
        `

	//fmt.Println(htmlStr)
	return c.HTML(http.StatusOK, htmlStr)
}
