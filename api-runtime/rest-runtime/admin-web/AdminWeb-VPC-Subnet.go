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
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	"strconv"

	"encoding/json"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

//====================================== VPC

// number, VPC Name, VPC CIDR, SUBNET Info, Additional Info, checkbox
func makeVPCTRList_html(bgcolor string, height string, fontSize string, infoList []*cres.VPCInfo) string {
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
                            <font size=%s>$$VPCNAME$$</font>
                    </td>
                    <td>
                            <font size=%s>$$VPCCIDR$$</font>
                    </td>
                    <td>
                            <font size=%s>$$SUBNETINFO$$</font>
                    </td>
                    <td>
                            <font size=%s>$$ADDITIONALINFO$$</font>
                    </td>
                    <td>
                        <input type="checkbox" name="check_box" value=$$VPCNAME$$>
                    </td>
                </tr>
                `, bgcolor, height, fontSize, fontSize, fontSize, fontSize, fontSize)

	strRemoveSubnet := fmt.Sprintf(`
                <a href="javascript:$$REMOVESUBNET$$;">
                        <font color=red size=%s><b>&nbsp;X</b></font>
                </a>
                `, fontSize)

	strAddSubnet := fmt.Sprintf(`
                <textarea style="font-size:12px;text-align:center;" name="subnet_text_box_$$ADDVPC$$" id="subnet_text_box_$$ADDVPC$$" cols=40>{ "Name": "subnet-02-add", "IPv4_CIDR": "10.0.12.0/22"}</textarea>
                <a href="javascript:$$ADDSUBNET$$;">
                        <font size=%s><mark><b>+</b></mark></font>
                </a>
								`, fontSize)

	strData := ""
	// set data and make TR list
	for i, one := range infoList {
		str := strings.ReplaceAll(strTR, "$$NUM$$", strconv.Itoa(i+1))
		str = strings.ReplaceAll(str, "$$VPCNAME$$", one.IId.NameId)
		str = strings.ReplaceAll(str, "$$VPCCIDR$$", one.IPv4_CIDR)

		var vpcName = one.IId.NameId

		// for subnet
		strSubnetList := ""
		for _, one := range one.SubnetInfoList {
			strSubnetList += one.IId.NameId + ", "
			strSubnetList += "CIDR:" + one.IPv4_CIDR + ", {"
			for _, kv := range one.KeyValueList {
				strSubnetList += kv.Key + ":" + kv.Value + ", "
			}
			strSubnetList = strings.TrimRight(strSubnetList, ", ")
			strSubnetList += "}"

			var subnetName = one.IId.NameId
			strSubnetList += strings.ReplaceAll(strRemoveSubnet, "$$REMOVESUBNET$$", "deleteSubnet('"+vpcName+"', '"+subnetName+"')")

			strSubnetList += "<br>"
		}
		vpcAddSubnet := strings.ReplaceAll(strAddSubnet, "$$ADDVPC$$", vpcName)
		strSubnetList += strings.ReplaceAll(vpcAddSubnet, "$$ADDSUBNET$$", "postSubnet('"+vpcName+"')")
		str = strings.ReplaceAll(str, "$$SUBNETINFO$$", strSubnetList)

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
func makePostVPCFunc_js() string {

	//curl -sX POST http://localhost:1024/spider/vpc -H 'Content-Type: application/json'
	//      -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": { "Name": "vpc-01", "IPv4_CIDR": "192.168.0.0/16",
	//              "SubnetInfoList": [ { "Name": "subnet-01", "IPv4_CIDR": "192.168.1.0/24"} ] } }'

	strFunc := `
                function postVPC() {
                        var connConfig = parent.frames["top_frame"].document.getElementById("connConfig").innerHTML;

                        var textboxes = document.getElementsByName('text_box');
            sendJson = '{ "ConnectionName" : "' + connConfig + '", "ReqInfo" : { "Name" : "$$VPCNAME$$", "IPv4_CIDR" : "$$VPCCIDR$$", "SubnetInfoList" : $$SUBNETINFOLIST$$ }}'

                        for (var i = 0; i < textboxes.length; i++) { // @todo make parallel executions
                                switch (textboxes[i].id) {
                                        case "1":
                                                sendJson = sendJson.replace("$$VPCNAME$$", textboxes[i].value);
                                                break;
                                        case "2":
                                                sendJson = sendJson.replace("$$VPCCIDR$$", textboxes[i].value);
                                                break;
                                        case "3":
                                                sendJson = sendJson.replace("$$SUBNETINFOLIST$$", textboxes[i].value);
                                                break;
                                        default:
                                                break;
                                }
                        }
                        var xhr = new XMLHttpRequest();
                        xhr.open("POST", "$$SPIDER_SERVER$$/spider/vpc", false);
                        xhr.setRequestHeader('Content-Type', 'application/json');

			// client logging
			parent.frames["log_frame"].Log("curl -sX POST " + "$$SPIDER_SERVER$$/spider/vpc -H 'Content-Type: application/json' -d '" + sendJson + "'");

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
func makePostSubnetFunc_js() string {

	//curl -sX POST http://localhost:1024/spider/vpc/vpc-01/subnet -H 'Content-Type: application/json'
	//      -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": { "Name": "subnet-02", "IPv4_CIDR": "192.168.2.0/24" } }'

	strFunc := `
                function postSubnet(vpcName) {
                        var connConfig = parent.frames["top_frame"].document.getElementById("connConfig").innerHTML;

                        var textbox = document.getElementById('subnet_text_box_' + vpcName);
                        sendJson = '{ "ConnectionName" : "' + connConfig + '", "ReqInfo" :  $$SUBNETINFO$$ }'

                        sendJson = sendJson.replace("$$SUBNETINFO$$", textbox.value);

                        var xhr = new XMLHttpRequest();
                        xhr.open("POST", "$$SPIDER_SERVER$$/spider/vpc/" + vpcName + "/subnet", false);
                        xhr.setRequestHeader('Content-Type', 'application/json');

			 // client logging
			parent.frames["log_frame"].Log("curl -sX POST " + "$$SPIDER_SERVER$$/spider/vpc/" + vpcName + "/subnet" + " -H 'Content-Type: application/json' -d '" + sendJson + "'");

                        xhr.send(sendJson);

			// client logging
			parent.frames["log_frame"].Log("   => " + xhr.response);

                        location.reload();
                }
        `
	strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://"+cr.ServiceIPorName+cr.ServicePort) // cr.ServicePort = ":1024"
	return strFunc
}

// make the string of javascript function
func makeDeleteVPCFunc_js() string {
	// curl -sX DELETE http://localhost:1024/spider/vpc/vpc-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}'

	strFunc := `
                function deleteVPC() {
                        var connConfig = parent.frames["top_frame"].document.getElementById("connConfig").innerHTML;
                        var checkboxes = document.getElementsByName('check_box');
                        for (var i = 0; i < checkboxes.length; i++) { // @todo make parallel executions
                                if (checkboxes[i].checked) {
                                        var xhr = new XMLHttpRequest();
                                        xhr.open("DELETE", "$$SPIDER_SERVER$$/spider/vpc/" + checkboxes[i].value, false); // synch
                                        xhr.setRequestHeader('Content-Type', 'application/json');
					sendJson = '{ "ConnectionName": "' + connConfig + '"}'

					// client logging
					parent.frames["log_frame"].Log("curl -sX DELETE " + "$$SPIDER_SERVER$$/spider/vpc/" + checkboxes[i].value +" -H 'Content-Type: application/json' -d '" + sendJson + "'");

                                        xhr.send(sendJson);

					// client logging
					parent.frames["log_frame"].Log("   => " + xhr.response);
                                }
                        }
			location.reload();
                }
        `
	strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://"+cr.ServiceIPorName+cr.ServicePort) // cr.ServicePort = ":1024"
	return strFunc
}

// make the string of javascript function
func makeDeleteSubnetFunc_js() string {
	//curl -sX DELETE http://localhost:1024/spider/vpc/vpc-01/subnet/subnet-02 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}'

	strFunc := `
                function deleteSubnet(vpcName, subnetName) {
                        var connConfig = parent.frames["top_frame"].document.getElementById("connConfig").innerHTML;

                        var xhr = new XMLHttpRequest();
                        xhr.open("DELETE", "$$SPIDER_SERVER$$/spider/vpc/" + vpcName + "/subnet/" + subnetName, false);
                        xhr.setRequestHeader('Content-Type', 'application/json');
                        sendJson = '{ "ConnectionName": "' + connConfig + '"}'

			 // client logging
			parent.frames["log_frame"].Log("curl -sX DELETE " + "$$SPIDER_SERVER$$/spider/vpc/" + vpcName + "/subnet/" + subnetName + " -H 'Content-Type: application/json' -d '" + sendJson + "'");

                        xhr.send(sendJson);

			// client logging
			parent.frames["log_frame"].Log("   => " + xhr.response);

                        location.reload();
                }
        `
	strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://"+cr.ServiceIPorName+cr.ServicePort) // cr.ServicePort = ":1024"
	return strFunc
}

func VPC(c echo.Context) error {
	cblog.Info("call VPC()")

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
	htmlStr += makePostVPCFunc_js()
	htmlStr += makeDeleteVPCFunc_js()
	htmlStr += makePostSubnetFunc_js()
	htmlStr += makeDeleteSubnetFunc_js()

	htmlStr += `
                    </script>
                </head>

                <body>
                    <table border="0" bordercolordark="#F8F8FF" cellpadding="0" cellspacing="1" bgcolor="#FFFFFF"  style="font-size:small;">
                `

	// (2) make Table Action TR
	// colspan, f5_href, delete_href, fontSize
	//htmlStr += makeActionTR_html("6", "vpc", "deleteVPC()", "2")
	htmlStr += makeActionTR_html("6", "", "deleteVPC()", "2")

	// (3) make Table Header TR
	nameWidthList := []NameWidth{
		{"VPC Name", "200"},
		{"VPC CIDR", "200"},
		{"Subnet Info", "300"},
		{"Additional Info", "300"},
	}
	htmlStr += makeTitleTRList_html("#DDDDDD", "2", nameWidthList, true)

	// (4) make TR list with info list
	// (4-1) get info list

	// client logging
	htmlStr += genLoggingGETURL(connConfig, "vpc")

	resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "vpc")
	if err != nil {
		cblog.Error(err)
		// client logging
		htmlStr += genLoggingResult(err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	// client logging
	htmlStr += genLoggingResult(string(resBody[:len(resBody)-1]))

	var info struct {
		ResultList []*cres.VPCInfo `json:"vpc"`
	}
	json.Unmarshal(resBody, &info)

	// (4-2) make TR list with info list
	htmlStr += makeVPCTRList_html("", "", "", info.ResultList)

	// (5) make input field and add
	// attach text box for add
	htmlStr += `
                        <tr bgcolor="#FFFFFF" align="center" height="30">
                            <td bgcolor="#FFEFBA">
                                    <font size=2>&nbsp;create:&nbsp;</font>
                            </td>
                            <td>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="1" value="vpc-01">
                            </td>
                            <td>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="2" value="10.0.0.0/16">
                            </td>
                            <td>
                                <textarea style="font-size:12px;text-align:center;" name="text_box" id="3" cols=50>[ { "Name": "subnet-01", "IPv4_CIDR": "10.0.8.0/22"} ]</textarea>
                            </td>
                            <td>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="4" disabled value="N/A">
                            </td>                            
                            <td>
                                <a href="javascript:postVPC()">
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


