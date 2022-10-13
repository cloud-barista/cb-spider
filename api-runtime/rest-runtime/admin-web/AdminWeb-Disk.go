// Cloud Info Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2022.09.

package adminweb

import (
	"fmt"

	cr "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	"strconv"

	"encoding/json"
	"net/http"
	"strings"
	"regexp"

	"github.com/labstack/echo/v4"
)

//====================================== Disk

// number, Disk Name, Disk Type, Disk Size, Disk Status, Attach/Detach, Created Time, Additional Info, checkbox
func makeDiskTRList_html(bgcolor string, height string, fontSize string, infoList []*cres.DiskInfo, vmList []string) string {
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
                            <font size=%s>$$DISKNAME$$</font>
                    </td>
                    <td>
                            <font size=%s>$$DISKTYPE$$</font>
                    </td>
                    <td>
                            <font size=%s>$$DISKSIZE$$</font>
                    </td>
                    <td>
                            <font size=%s>$$DISKSTATUS$$</font>
                    </td>
                    <td>
                            <font size=%s>$$ATTACHDETACH$$</font>
                    </td>
                    <td>
                            <font size=%s>$$CREATEDTIME$$</font>
                    </td>
                    <td>
                            <font size=%s>$$ADDITIONALINFO$$</font>
                    </td>
                    <td>
                        <input type="checkbox" name="check_box" value=$$DISKNAME$$>
                    </td>
                </tr>
                `, bgcolor, height, fontSize, fontSize, fontSize, fontSize, fontSize, fontSize, fontSize, fontSize)

        strData := ""
        // set data and make TR list
        for i, one := range infoList {
		// to select a VM in VMList
		selectHtml := makeSelect_html("", vmList, "select_box_" + one.IId.NameId) // <select name="text_box" id=select_box_{diskName} onchangeVM(this)>

                str := strings.ReplaceAll(strTR, "$$NUM$$", strconv.Itoa(i+1))
                str = strings.ReplaceAll(str, "$$DISKNAME$$", one.IId.NameId)

                // Disk Type
                str = strings.ReplaceAll(str, "$$DISKTYPE$$", one.DiskType)

		// Size
		sizeInputText := `<input style="font-size:12px;text-align:center;" type="text" name="size_box" size=6 id="size_input_text_` + 
			one.IId.NameId + `" value="`+ one.DiskSize + `">&nbsp;GB &nbsp;`

			sizeUpButton := `<button style="font-size:10px;" type="button" onclick="diskSizeUp('` + one.IId.NameId + `')">Upsize</button>`
                str = strings.ReplaceAll(str, "$$DISKSIZE$$", sizeInputText + sizeUpButton)

		// Status
                str = strings.ReplaceAll(str, "$$DISKSTATUS$$", string(one.Status))

		// Attach/Detach
		if one.Status == cres.DiskAvailable {
			attachButton := `<button style="font-size:10px;" type="button" onclick="diskAttach('` + 
				one.IId.NameId+ `')">Attach</button>`
			str = strings.ReplaceAll(str, "$$ATTACHDETACH$$", selectHtml + "&nbsp;&nbsp;" + attachButton)
		} else if one.Status == cres.DiskAttached {
			detachButton := `<button style="font-size:10px;" type="button" onclick="diskDetach('` + 
				one.IId.NameId+ `', '` + one.OwnerVM.NameId+ `')">Detach</button>`
			str = strings.ReplaceAll(str, "$$ATTACHDETACH$$", one.OwnerVM.NameId + "&nbsp;&nbsp;" + detachButton)
		} else {
			str = strings.ReplaceAll(str, "$$ATTACHDETACH$$", "N/A")
		}

		// Created Time
                str = strings.ReplaceAll(str, "$$CREATEDTIME$$", one.CreatedTime.Format("2006.01.02 15:04:05 Mon"))

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

// make the string of javascript function
func makePostDiskFunc_js() string {

        //curl -sX POST http://localhost:1024/spider/disk -H 'Content-Type: application/json'
        //      -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": {
                                //      "Name": "spider-disk-01",
                                //      "DiskType": "",
                                //      "DiskSize": ""
                        //      } }'

        strFunc := `
                function postDisk() {
                        var connConfig = parent.frames["top_frame"].document.getElementById("connConfig").innerHTML;

                        var textboxes = document.getElementsByName('text_box');
            sendJson = '{ "ConnectionName" : "' + connConfig + '", "ReqInfo" : { "Name" : "$$DISKNAME$$", "DiskType" : "$$DISKTYPE$$", "DiskSize" : "$$DISKSIZE$$"}}'

                        for (var i = 0; i < textboxes.length; i++) { // @todo make parallel executions
                                switch (textboxes[i].id) {
                                        case "1":
                                                sendJson = sendJson.replace("$$DISKNAME$$", textboxes[i].value);
                                                break;
                                        case "2":
                                                sendJson = sendJson.replace("$$DISKTYPE$$", textboxes[i].value);
                                                break;
                                        case "3":
                                                sendJson = sendJson.replace("$$DISKSIZE$$", textboxes[i].value);
                                                break;
                                        default:
                                                break;
                                }
                        }
                        var xhr = new XMLHttpRequest();
                        xhr.open("POST", "$$SPIDER_SERVER$$/spider/disk", false);
                        xhr.setRequestHeader('Content-Type', 'application/json');

                        // client logging
                        parent.frames["log_frame"].Log("curl -sX POST " + "$$SPIDER_SERVER$$/spider/disk -H 'Content-Type: application/json' -d '" + sendJson + "'");

                        xhr.send(sendJson);

                        // client logging
                        parent.frames["log_frame"].Log("   ==> " + xhr.response);
                        var jsonVal = JSON.parse(xhr.response)


            location.reload();
                }
        `
        strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://"+cr.ServiceIPorName+cr.ServicePort) // cr.ServicePort = ":1024"
        return strFunc
}

// make the string of javascript function
func makeDiskSizeUpFunc_js() string {
	/*
	curl -sX PUT http://localhost:1024/spider/disk/spider-disk-01/size -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName": "mock-config01",
                "ReqInfo": {
                        "Size" : "128"
                }
        }'
	*/

        strFunc := `
                function diskSizeUp(diskName, sizeInputTextId) {
                        var connConfig = parent.frames["top_frame"].document.getElementById("connConfig").innerHTML;

                        var textbox = document.getElementById('size_input_text_'+diskName);
                        upSize = textbox.value

                        var xhr = new XMLHttpRequest();
                        xhr.open("PUT", "$$SPIDER_SERVER$$/spider/disk/" + diskName + "/size", false);
                        xhr.setRequestHeader('Content-Type', 'application/json');
                        sendJson = '{ "ConnectionName": "' + connConfig + '", "ReqInfo": { "Size" : "' + upSize + '" } }'

                        // client logging
                        parent.frames["log_frame"].Log("PUT> " + "$$SPIDER_SERVER$$/spider/disk/" + diskName + "/size" + " -H 'Content-Type: application/json' -d '" + sendJson + "'");

                        xhr.send(sendJson);

                        // client logging
                        parent.frames["log_frame"].Log("   ==> " + xhr.response);

                        location.reload();
                }
        `
        strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://"+cr.ServiceIPorName+cr.ServicePort) // cr.ServicePort = ":1024"
        return strFunc
}


// make the string of javascript function
func makeAttachDiskFunc_js() string {
        /*curl -sX PUT http://localhost:1024/spider/disk/spider-disk-01/attach -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName": "mock-config01",
                "ReqInfo": {
                        "VMName" : "vm-01"
                }
        }'
        */

        strFunc := `
                function diskAttach(diskName) {
                        var connConfig = parent.frames["top_frame"].document.getElementById("connConfig").innerHTML;

			var textbox = document.getElementById('select_box_'+diskName);
			vmName = textbox.value

                        var xhr = new XMLHttpRequest();
                        xhr.open("PUT", "$$SPIDER_SERVER$$/spider/disk/" + diskName + "/attach", false);
                        xhr.setRequestHeader('Content-Type', 'application/json');
                        sendJson = '{ "ConnectionName": "' + connConfig + '", "ReqInfo": { "VMName" : "' + vmName + '" } }'

                        // client logging
                        parent.frames["log_frame"].Log("PUT> " + "$$SPIDER_SERVER$$/spider/disk/" + diskName + "/attach" + " -H 'Content-Type: application/json' -d '" + sendJson + "'");

                        xhr.send(sendJson);

                        // client logging
                        parent.frames["log_frame"].Log("   ==> " + xhr.response);

                        location.reload();
                }
        `
        strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://"+cr.ServiceIPorName+cr.ServicePort) // cr.ServicePort = ":1024"
        return strFunc
}

// make the string of javascript function
func makeDetachDiskFunc_js() string {
        /*curl -sX PUT http://localhost:1024/spider/disk/spider-disk-01/detach -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName": "mock-config01",
                "ReqInfo": {
                        "VMName" : "vm-01"
                }
        }'
        */

        strFunc := `
                function diskDetach(diskName, vmName) {
                        var connConfig = parent.frames["top_frame"].document.getElementById("connConfig").innerHTML;

                        var xhr = new XMLHttpRequest();
                        xhr.open("PUT", "$$SPIDER_SERVER$$/spider/disk/" + diskName + "/detach", false);
                        xhr.setRequestHeader('Content-Type', 'application/json');
                        sendJson = '{ "ConnectionName": "' + connConfig + '", "ReqInfo": { "VMName" : "' + vmName + '" } }'

                        // client logging
                        parent.frames["log_frame"].Log("PUT> " + "$$SPIDER_SERVER$$/spider/disk/" + diskName + "/detach" + " -H 'Content-Type: application/json' -d '" + sendJson + "'");

                        xhr.send(sendJson);

                        // client logging
                        parent.frames["log_frame"].Log("   ==> " + xhr.response);

                        location.reload();
                }
        `
        strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://"+cr.ServiceIPorName+cr.ServicePort) // cr.ServicePort = ":1024"
        return strFunc
}

// make the string of javascript function
func makeDeleteDiskFunc_js() string {
        // curl -sX DELETE http://localhost:1024/spider/disk/spider-disk-01 -H 'Content-Type: application/json'
        //           -d '{ "ConnectionName": "'${CONN_CONFIG}'"}'

        strFunc := `
                function deleteDisk() {
                        var connConfig = parent.frames["top_frame"].document.getElementById("connConfig").innerHTML;
                        var checkboxes = document.getElementsByName('check_box');
                        for (var i = 0; i < checkboxes.length; i++) { // @todo make parallel executions
                                if (checkboxes[i].checked) {
                                        var xhr = new XMLHttpRequest();
                                        xhr.open("DELETE", "$$SPIDER_SERVER$$/spider/disk/" + checkboxes[i].value, false);
                                        xhr.setRequestHeader('Content-Type', 'application/json');
                                        sendJson = '{ "ConnectionName": "' + connConfig + '"}'

                                        // client logging
                                        parent.frames["log_frame"].Log("curl -sX DELETE " + "$$SPIDER_SERVER$$/spider/disk/" + checkboxes[i].value + " -H 'Content-Type: application/json' -d '" + sendJson + "'");

                                        xhr.send(sendJson);

                                        // client logging
                                        parent.frames["log_frame"].Log("   ==> " + xhr.response);
                                }
                        }
            location.reload();
                }
        `
        strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://"+cr.ServiceIPorName+cr.ServicePort) // cr.ServicePort = ":1024"
        return strFunc
}

func Disk(c echo.Context) error {
        cblog.Info("call Disk()")

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
                    <script type="text/javascript">
                `
        // (1) make Javascript Function
        htmlStr += makeCheckBoxToggleFunc_js()
        htmlStr += makePostDiskFunc_js()
        htmlStr += makeDiskSizeUpFunc_js()
        htmlStr += makeAttachDiskFunc_js()
        htmlStr += makeDetachDiskFunc_js()
        htmlStr += makeDeleteDiskFunc_js()

        htmlStr += `
                    </script>
                </head>

                <body>
                    <table border="0" bordercolordark="#F8F8FF" cellpadding="0" cellspacing="1" bgcolor="#FFFFFF"  style="font-size:small;">
                `

        // (2) make Table Action TR
        // colspan, f5_href, delete_href, fontSize
        htmlStr += makeActionTR_html("8", "", "deleteDisk()", "2")

        // (3) make Table Header TR
        nameWidthList := []NameWidth{
                {"Disk Name", "100"},
                {"Disk Type", "100"},
                {"Disk Size(GB)", "100"},
                {"Disk Status", "100"},
                {"Disk Attach|Detach", "200"},
                {"Created Time", "100"},
                {"Additional Info", "300"},
        }
        htmlStr += makeTitleTRList_html("#DDDDDD", "2", nameWidthList, true)

        // (4) make TR list with info list
        // (4-1) get info list

        // client logging
        htmlStr += genLoggingGETURL(connConfig, "disk")

        resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "disk")
        if err != nil {
                cblog.Error(err)
                // client logging
                htmlStr += genLoggingResult(err.Error())
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // client logging
        htmlStr += genLoggingResult(string(resBody[:len(resBody)-1]))

        var info struct {
                ResultList []*cres.DiskInfo `json:"disk"`
        }
        json.Unmarshal(resBody, &info)

	// get VM List
	vmList := vmList(connConfig)

        // (4-2) make TR list with info list
        htmlStr += makeDiskTRList_html("", "", "", info.ResultList, vmList)

	providerName, _ := getProviderName(connConfig)
	diskTypeList := diskTypeList(providerName)
	diskTypeSizeList := diskTypeSizeList(providerName)


        // (5) make input field and add
        // attach text box for add
	
        htmlStr += `
                        <tr bgcolor="#FFFFFF" align="center" height="30">
                            <td bgcolor="#FFEFBA">
                                    <font size=2>&nbsp;create:&nbsp;</font>
                            </td>
                            <td>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="1" value="spider-disk-01">
                            </td>
                            <td>

                            `
        // Select format of Disk Type  name=text_box, id=2
        htmlStr += makeDataDiskTypeSelect_html("", diskTypeList, "2")

	htmlStr += `

                            </td>
                            <td>

                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="3" value="default">
			    `
				htmlStr += makeDataDiskTypeSize_html(diskTypeSizeList)

				htmlStr += `

                            </td>
                            <td>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="4" disabled value="N/A">
                            </td>
                            <td>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="5" disabled value="N/A">
                            </td>
                            <td>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="6" disabled value="N/A">
                            </td>
                            <td>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="7" disabled value="N/A">
                            </td>
                            <td>
                                <a href="javascript:postDisk()">
                                    <font size=4><mark><b>+</b></mark></font>
                                </a>
                            </td>
                        </tr>
                `
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


func makeDataDiskTypeSelect_html(onchangeFunctionName string, strList []string, id string) string {

        strResult := ""
        if len(strList) == 0 {
                noDiskStr := `<input style="font-size:12px;text-align:center;" type="text" name="text_box" id="` +
                                id +`" value="default">`
                return strResult + noDiskStr
        }
        strSelect := `<select style="width:120px;" name="text_box" id="` + id + `" onchange="` + onchangeFunctionName + `(this)">`
                strSelect += `<option value="default">default</option>`
        for _, one := range strList {
                strSelect += `<option value="` + one + `">` + one + `</option>`
        }

        strSelect += `
                </select>
        `

        return strResult + strSelect
}

func makeDataDiskTypeSize_html(strList []string) string {

	strResult := ""
        for _, one := range strList {
		// one = "cloud|5|2000|GB"
		splits := strings.Split(one, "|")
		rangeStr := ""
		if len(splits) == 4 {
			rangeStr = fmt.Sprintf("&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;[%s]&nbsp;&nbsp; %s~%s %s<br>",
					strings.TrimSpace(splits[0]), insertComma(strings.TrimSpace(splits[1])),
				 	insertComma(strings.TrimSpace(splits[2])), strings.TrimSpace(splits[3]))
		} else {
			rangeStr = one // keep origin string
		}
		strResult += rangeStr
        }

        strInput := `<p style="font-size:12px;color:gray;text-align:left;">` + strResult + `</p>`

        return strInput
}

// ref) https://stackoverflow.com/a/39185719/17474800
func insertComma(str string) string {
    re := regexp.MustCompile("(\\d+)(\\d{3})")
    for n := ""; n != str; {
        n = str
        str = re.ReplaceAllString(str, "$1,$2")
    }
    return str
}
