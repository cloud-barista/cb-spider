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
	cr "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"

/*
        "github.com/cloud-barista/cb-store/config"
        "github.com/sirupsen/logrus"
*/
	"strconv"

	"net/http"
	"strings"
	"github.com/labstack/echo"
	"encoding/json"
)

// number, VPC Name, VPC CIDR, SUBNET Info, Additional Info, checkbox
func makeVPCMgmtTRList_html(bgcolor string, height string, fontSize string, infoList cr.AllResourceList) string {
        if bgcolor == "" { bgcolor = "#FFFFFF" }
        if height == "" { height = "30" }
        if fontSize == "" { fontSize = "2" }

        // make base TR frame for info list
        strTR := fmt.Sprintf(`
                <tr bgcolor="%s" align="center" height="%s">
                    <td>
                            <font size=%s>$$NUM$$</font>
                    </td>
                    <td $$NAMEIDSTYLE$$>
                            <font size=%s>$$NAMEID$$</font>
                    </td>
                    <td $$SYTEMIDSTYLE$$>
                            <font size=%s>$$SYTEMID$$</font>
                    </td>
                    <td>
                        <input type="checkbox" name="check_box" value=$$IID$$>
                    </td>
                </tr>
                `, bgcolor, height, fontSize, fontSize, fontSize)

        strData := ""
        // set data and make TR list
        for i, one := range infoList.AllList.MappedList{
                str := strings.ReplaceAll(strTR, "$$NUM$$", strconv.Itoa(i+1))
                str = strings.ReplaceAll(str, "$$NAMEID$$", one.NameId)
                str = strings.ReplaceAll(str, "$$SYTEMID$$", one.SystemId)
                str = strings.ReplaceAll(str, "$$IID$$", one.NameId + ":" + one.SystemId) // MappedList: contain ":"
                str = strings.ReplaceAll(str, "$$NAMEIDSTYLE$$", `style="background-color:#F0F3FF;"`)
                str = strings.ReplaceAll(str, "$$SYTEMIDSTYLE$$", `style="background-color:#F0F3FF;"`)
                strData += str
        }
        for i, one := range infoList.AllList.OnlySpiderList{
                str := strings.ReplaceAll(strTR, "$$NUM$$", strconv.Itoa(i+1))
                str = strings.ReplaceAll(str, "$$NAMEID$$", one.NameId)
                str = strings.ReplaceAll(str, "$$SYTEMID$$", "(" + one.SystemId + ")")
                str = strings.ReplaceAll(str, "$$IID$$", one.NameId + ":" + one.SystemId) // OnlySpiderList: contain ":"

                str = strings.ReplaceAll(str, "$$NAMEIDSTYLE$$", `style="background-color:#F0F3FF;"`)
                str = strings.ReplaceAll(str, "$$SYTEMIDSTYLE$$", ``)
                strData += str
        }
        for i, one := range infoList.AllList.OnlyCSPList{
                str := strings.ReplaceAll(strTR, "$$NUM$$", strconv.Itoa(i+1))
                str = strings.ReplaceAll(str, "$$NAMEID$$", "(" + one.NameId + ")")
                str = strings.ReplaceAll(str, "$$SYTEMID$$", one.SystemId)
                str = strings.ReplaceAll(str, "$$IID$$", one.SystemId) // OnlyCSPList: not contain ":"

                str = strings.ReplaceAll(str, "$$NAMEIDSTYLE$$", ``)
                str = strings.ReplaceAll(str, "$$SYTEMIDSTYLE$$", `style="background-color:#F0F3FF;"`)
                strData += str
        }        

        return strData
}

// make the string of javascript function
func makeDeleteVPCMgmtFunc_js() string {
// delete for MappedList & OnlySpiderList
// curl -sX DELETE http://localhost:1024/spider/vpc/vpc-01?force=true -H 'Content-Type: application/json' -d '{ "ConnectionName": "aws-ohio-config"}
// delete for OnlyCSPList
// curl -sX DELETE http://localhost:1024/spider/cspvpc/vpc-0b0d0d30794eab379 -H 'Content-Type: application/json' -d '{ "ConnectionName": "aws-ohio-config"}' |json_pp

        strFunc := `
                function deleteVPCMgmt() {
                        var connConfig = parent.frames["top_frame"].document.getElementById("connConfig").innerHTML;
                        var checkboxes = document.getElementsByName('check_box');
                        for (var i = 0; i < checkboxes.length; i++) { // @todo make parallel executions
                                if (checkboxes[i].checked) {
                                        var xhr = new XMLHttpRequest();
                                        if(checkboxes[i].value.includes(":")) { // MappedList & OnlySpiderList
                                            xhr.open("DELETE", "$$SPIDER_SERVER$$/spider/vpc/" + checkboxes[i].value + "?force=true", false);
                                        }else { // OnlyCSPList
                                            xhr.open("DELETE", "$$SPIDER_SERVER$$/spider/cspvpc/" + checkboxes[i].value, false);
                                        }

                                        xhr.setRequestHeader('Content-Type', 'application/json');
					sendJson = '{ "ConnectionName": "' + connConfig + '"}'
                                        xhr.send(sendJson);
                                }
                        }
			location.reload();
                }
        `
        strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://" + cr.HostIPorName + cr.ServicePort) // cr.ServicePort = ":1024"
        return strFunc
}

func VPCMgmt(c echo.Context) error {
        cblog.Info("call VPCMgmt()")

	connConfig := c.Param("ConnectConfig")
	if connConfig == "region not set" {
		htmlStr :=  `
			<html>
			<head>
			    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
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
        htmlStr :=  `
                <html>
                <head>
                    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
                    <script type="text/javascript">
                `
        // (1) make Javascript Function
                htmlStr += makeCheckBoxToggleFunc_js()                
                htmlStr += makeDeleteVPCMgmtFunc_js()


        htmlStr += `
                    </script>
                </head>

                <body>
                    <table border="0" bordercolordark="#F8F8FF" cellpadding="0" cellspacing="1" bgcolor="#FFFFFF"  style="font-size:small;">
                `

        // (2) make Table Action TR
                // colspan, f5_href, delete_href, fontSize
                //htmlStr += makeActionTR_html("4", "vpc", "deleteVPCMgmt()", "2")
                htmlStr += makeActionTR_html("4", "", "deleteVPCMgmt()", "2")


        // (3) make Table Header TR
                nameWidthList := []NameWidth {
                    {"Spider's NameId", "300"},
                    {"CSP's SystemId", "300"},                    
                }
                htmlStr +=  makeTitleTRList_html("#DDDDDD", "2", nameWidthList)


        // (4) make TR list with info list
        // (4-1) get info list 
                resBody, err := getAllResourceList_with_Connection_JsonByte(connConfig, "vpc")
                if err != nil {
                        cblog.Error(err)
                        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
                }
                
                var info cr.AllResourceList
                
                json.Unmarshal(resBody, &info)

        // (4-2) make TR list with info list
                htmlStr += makeVPCMgmtTRList_html("", "", "", info)

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
