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

	"github.com/labstack/echo/v4"
)

//====================================== MyImage

// number, MyImage Name, MyImage Source VM, MyImage Status, Created Time, Additional Info, checkbox
func makeMyImageTRList_html(bgcolor string, height string, fontSize string, infoList []*cres.MyImageInfo) string {
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
                            <font size=%s>$$MYIMAGENAME$$</font>
                    </td>
                    <td>
                            <font size=%s>$$SOURCEVM$$</font>
                    </td>
                    <td>
                            <font size=%s>$$MYIMAGESTATUS$$</font>
                    </td>
                    <td>
                            <font size=%s>$$CREATEDTIME$$</font>
                    </td>
                    <td>
                            <font size=%s>$$ADDITIONALINFO$$</font>
                    </td>
                    <td>
                        <input type="checkbox" name="check_box" value=$$MYIMAGENAME$$>
                    </td>
                </tr>
                `, bgcolor, height, fontSize, fontSize, fontSize, fontSize, fontSize, fontSize)

        strData := ""
        // set data and make TR list
        for i, one := range infoList {
                str := strings.ReplaceAll(strTR, "$$NUM$$", strconv.Itoa(i+1))
                str = strings.ReplaceAll(str, "$$MYIMAGENAME$$", one.IId.NameId)

                // Source VM
                str = strings.ReplaceAll(str, "$$SOURCEVM$$", one.SourceVM.NameId)

		// Status
                str = strings.ReplaceAll(str, "$$MYIMAGESTATUS$$", string(one.Status))

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
func makePostMyImageFunc_js() string {

        //curl -sX POST http://localhost:1024/spider/myimage -H 'Content-Type: application/json'
        //      -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": {
                                //      "Name": "spider-myimage-01",
                                //      "SourceVM": "vm-01"
                        //      } }'

        strFunc := `
                function postMyImage() {
                        var connConfig = parent.frames["top_frame"].document.getElementById("connConfig").innerHTML;

                        var textboxes = document.getElementsByName('text_box');
            sendJson = '{ "ConnectionName" : "' + connConfig + '", "ReqInfo" : { "Name" : "$$MYIMAGENAME$$", "SourceVM" : "$$SOURCEVM$$"}}'

                        for (var i = 0; i < textboxes.length; i++) { // @todo make parallel executions
                                switch (textboxes[i].id) {
                                        case "1":
                                                sendJson = sendJson.replace("$$MYIMAGENAME$$", textboxes[i].value);
                                                break;
                                        case "2":
                                                sendJson = sendJson.replace("$$SOURCEVM$$", textboxes[i].value);
                                                break;
                                        default:
                                                break;
                                }
                        }
                        var xhr = new XMLHttpRequest();
                        xhr.open("POST", "$$SPIDER_SERVER$$/spider/myimage", false);
                        xhr.setRequestHeader('Content-Type', 'application/json');

                        // client logging
                        parent.frames["log_frame"].Log("curl -sX POST " + 
                                "$$SPIDER_SERVER$$/spider/myimage -H 'Content-Type: application/json' -d '" + sendJson + "'");

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
func makeDeleteMyImageFunc_js() string {
        // curl -sX DELETE http://localhost:1024/spider/myimage/spider-myimage-01 -H 'Content-Type: application/json'
        //           -d '{ "ConnectionName": "'${CONN_CONFIG}'"}'

        strFunc := `
                function deleteMyImage() {
                        var connConfig = parent.frames["top_frame"].document.getElementById("connConfig").innerHTML;
                        var checkboxes = document.getElementsByName('check_box');
                        for (var i = 0; i < checkboxes.length; i++) { // @todo make parallel executions
                                if (checkboxes[i].checked) {
                                        var xhr = new XMLHttpRequest();
                                        xhr.open("DELETE", "$$SPIDER_SERVER$$/spider/myimage/" + checkboxes[i].value, false);
                                        xhr.setRequestHeader('Content-Type', 'application/json');
                                        sendJson = '{ "ConnectionName": "' + connConfig + '"}'

                                        // client logging
                                        parent.frames["log_frame"].Log("curl -sX DELETE " + "$$SPIDER_SERVER$$/spider/myimage/" + 
                                                checkboxes[i].value + " -H 'Content-Type: application/json' -d '" + sendJson + "'");

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

func MyImage(c echo.Context) error {
        cblog.Info("call MyImage()")

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
        htmlStr += makePostMyImageFunc_js()
        htmlStr += makeDeleteMyImageFunc_js()

        htmlStr += `
                    </script>
                </head>

                <body>
                    <table border="0" bordercolordark="#F8F8FF" cellpadding="0" cellspacing="1" bgcolor="#FFFFFF"  style="font-size:small;">
                `

        // (2) make Table Action TR
        // colspan, f5_href, delete_href, fontSize
        htmlStr += makeActionTR_html("8", "", "deleteMyImage()", "2")

        // (3) make Table Header TR
        nameWidthList := []NameWidth{
                {"MyImage Name", "100"},
                {"MyImage SourceVM", "100"},
                {"MyImage Status", "100"},
                {"Created Time", "100"},
                {"Additional Info", "300"},
        }
        htmlStr += makeTitleTRList_html("#DDDDDD", "2", nameWidthList, true)

        // (4) make TR list with info list
        // (4-1) get info list

        // client logging
        htmlStr += genLoggingGETURL(connConfig, "myimage")

        resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "myimage")
        if err != nil {
                cblog.Error(err)
                // client logging
                htmlStr += genLoggingResult(err.Error())
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // client logging
        htmlStr += genLoggingResult(string(resBody[:len(resBody)-1]))

        var info struct {
                ResultList []*cres.MyImageInfo `json:"myimage"`
        }
        json.Unmarshal(resBody, &info)
	
        // (4-2) make TR list with info list
        htmlStr += makeMyImageTRList_html("", "", "", info.ResultList)

        // (5) make input field and add
        // attach text box for add
	
        htmlStr += `
                        <tr bgcolor="#FFFFFF" align="center" height="30">
                            <td bgcolor="#FFEFBA">
                                    <font size=2>&nbsp;create:&nbsp;</font>
                            </td>
                            <td>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="1" value="spider-myimage-01">
                            </td>
                            <td>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="2" value="vm-01">
                            </td>                            
                            <td>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="3" disabled value="N/A">
                            </td>
                            <td>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="4" disabled value="N/A">
                            </td>
                            <td>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="5" disabled value="N/A">
                            </td>
                            <td>
                                <a href="javascript:postMyImage()">
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
