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
                            <td>
                                    <font size=2>#</font>
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

func genLoggingGETURL(connConfig string, rsType string) string {
	/* return example
	<script type="text/javascript">
		parent.frames["log_frame"].Log("curl -sX GET http://localhost:1024/spider/vpc -H 'Content-Type: application/json' -d '{"ConnectionName": "aws-ohio-config"}'   ");
	</script>
	*/

        url := "http://" + "localhost" + cr.ServerPort + "/spider/" + rsType + " -H 'Content-Type: application/json' -d '{\\\"ConnectionName\\\": \\\"" + connConfig  + "\\\"}'"
        htmlStr := `
                <script type="text/javascript">
                `
        htmlStr += `    parent.frames["log_frame"].Log("curl -sX GET ` +  url + `");`
        htmlStr += `
                </script>
                `
	return htmlStr
}

func genLoggingResult(response string) string {

        htmlStr := `
                <script type="text/javascript">
                `
        htmlStr += `    parent.frames["log_frame"].Log("   ==> ` + strings.ReplaceAll(response, "\"", "\\\"") + `");`
        htmlStr += `
                </script>
                `
        return htmlStr
}

func genLoggingOneGETURL(connConfig string, rsType string, name string) string {
        /* return example
        <script type="text/javascript">
                parent.frames["log_frame"].Log("curl -sX GET http://localhost:1024/spider/vpc/vpc-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "aws-ohio-config"}'  ");
        </script>
        */

        url := "http://" + "localhost" + cr.ServerPort + "/spider/" + rsType + "/" + name + " -H 'Content-Type: application/json' -d '{\\\"ConnectionName\\\": \\\"" + connConfig  + "\\\"}'"
        htmlStr := `
                <script type="text/javascript">
                `
        htmlStr += `    parent.frames["log_frame"].Log("curl -sX GET ` +  url + `");`
        htmlStr += `
                </script>
                `
        return htmlStr
}

//====================================== Security Group

// number, VPC Name, SecurityGroup Name, Security Rules, Additional Info, checkbox
func makeSecurityGroupTRList_html(bgcolor string, height string, fontSize string, infoList []*cres.SecurityInfo) string {
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
                            <font size=%s>$$SGNAME$$</font>
                    </td>                    
                    <td>
                            <font size=%s>$$SECURITYRULES$$</font>
                    </td>
                    <td>
                            <font size=%s>$$ADDITIONALINFO$$</font>
                    </td>
                    <td>
                        <input type="checkbox" name="check_box" value=$$SGNAME$$>
                    </td>
                </tr>
                `, bgcolor, height, fontSize, fontSize, fontSize, fontSize, fontSize)


        strRemoveRule := fmt.Sprintf(`
                <a href="javascript:$$REMOVERULE$$;">
                        <font color=red size=%s><b>&nbsp;X</b></font>
                </a>
                `, fontSize)

	strAddRule := fmt.Sprintf(`
	<textarea style="font-size:12px;text-align:center;" name="security_text_box_$$ADDSG$$" id="security_text_box_$$ADDSG$$" cols=40>{"FromPort": "1", "ToPort" : "65535", "IPProtocol" : "udp", "Direction" : "inbound", "CIDR" : "0.0.0.0/0" }</textarea>
                <a href="javascript:$$ADDRULE$$;">
                        <font size=%s><mark><b>+</b></mark></font>
                </a>
                                                                `, fontSize)


	strData := ""
	// set data and make TR list
	for i, one := range infoList {
		str := strings.ReplaceAll(strTR, "$$NUM$$", strconv.Itoa(i+1))
		str = strings.ReplaceAll(str, "$$VPCNAME$$", one.VpcIID.NameId)
		str = strings.ReplaceAll(str, "$$SGNAME$$", one.IId.NameId)

		sgName := one.IId.NameId

		// for security rules info
		strSRList := ""
		if one.SecurityRules != nil {
			for _, rule := range *one.SecurityRules {
				oneSR := fmt.Sprintf("{ \"FromPort\" : \"%s\", \"ToPort\" : \"%s\", \"IPProtocol\" : \"%s\", \"Direction\" : \"%s\", \"CIDR\" : \"%s\" }", 
						rule.FromPort, rule.ToPort, rule.IPProtocol, rule.Direction, rule.CIDR)

				strSRList += oneSR
				strDelete := "deleteRule('"+sgName+"', '"+rule.FromPort+"', '"+rule.ToPort+"', '"+rule.IPProtocol+"', '"+rule.Direction+"', '"+rule.CIDR+"')"
				strSRList += strings.ReplaceAll(strRemoveRule, "$$REMOVERULE$$", strDelete)

				strSRList += "<br>"

			}
		}

                SGAddRule := strings.ReplaceAll(strAddRule, "$$ADDSG$$", sgName)
                strSRList += strings.ReplaceAll(SGAddRule, "$$ADDRULE$$", "postRule('"+sgName+"')")

		str = strings.ReplaceAll(str, "$$SECURITYRULES$$", strSRList)

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
func makePostSecurityGroupFunc_js() string {

	//curl -sX POST http://localhost:1024/spider/securitygroup -H 'Content-Type: application/json'
	//  -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": { "Name": "sg-01", "VPCName": "vpc-01",
	//      "SecurityRules": [ {"FromPort": "1", "ToPort" : "65535", "IPProtocol" : "tcp", "Direction" : "inbound", "CIDR" : "0.0.0.0/0" } ] } }'

	strFunc := `
                function postSecurityGroup() {
                        var connConfig = parent.frames["top_frame"].document.getElementById("connConfig").innerHTML;

                        var textboxes = document.getElementsByName('text_box');
                        sendJson = '{ "ConnectionName" : "' + connConfig + '", "ReqInfo" : { "Name" : "$$SGNAME$$", "VPCName" : "$$VPCNAME$$", "SecurityRules" : $$SECURITYRULES$$ }}'

                        for (var i = 0; i < textboxes.length; i++) { // @todo make parallel executions
                                switch (textboxes[i].id) {
                                        case "1":
                                                sendJson = sendJson.replace("$$VPCNAME$$", textboxes[i].value);
                                                break;
                                        case "2":
                                                sendJson = sendJson.replace("$$SGNAME$$", textboxes[i].value);
                                                break;
                                        case "3":
                                                sendJson = sendJson.replace("$$SECURITYRULES$$", textboxes[i].value);
                                                break;
                                        default:
                                                break;
                                }
                        }
                        var xhr = new XMLHttpRequest();
                        xhr.open("POST", "$$SPIDER_SERVER$$/spider/securitygroup", false);
                        xhr.setRequestHeader('Content-Type', 'application/json');

			// client logging
			parent.frames["log_frame"].Log("curl -sX POST " + "$$SPIDER_SERVER$$/spider/securitygroup -H 'Content-Type: application/json' -d '" + sendJson + "'");

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
func makeDeleteSecurityGroupFunc_js() string {
	// curl -sX DELETE http://localhost:1024/spider/securitygroup/sg-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}'

	strFunc := `
                function deleteSecurityGroup() {
                        var connConfig = parent.frames["top_frame"].document.getElementById("connConfig").innerHTML;
                        var checkboxes = document.getElementsByName('check_box');
                        for (var i = 0; i < checkboxes.length; i++) { // @todo make parallel executions
                                if (checkboxes[i].checked) {
                                        var xhr = new XMLHttpRequest();
                                        xhr.open("DELETE", "$$SPIDER_SERVER$$/spider/securitygroup/" + checkboxes[i].value, false);
                                        xhr.setRequestHeader('Content-Type', 'application/json');
					sendJson = '{ "ConnectionName": "' + connConfig + '"}'

					// client logging
					parent.frames["log_frame"].Log("curl -sX DELETE " + "$$SPIDER_SERVER$$/spider/securitygroup/" + checkboxes[i].value + " -H 'Content-Type: application/json' -d '" + sendJson + "'");

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

// make the string of javascript function
func makeDeleteRuleFunc_js() string {
	/* 
	curl -sX DELETE http://localhost:1024/spider/securitygroup/${SG_NAME}/rules -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName": "'${CONN_CONFIG}'",
                "ReqInfo": {
                "RuleInfoList" :
                        [
                                {
                                        "Direction": "inbound",
                                        "IPProtocol": "ALL",
                                        "FromPort": "-1",
                                        "ToPort": "-1",
                                        "CIDR" : "0.0.0.0/0"
                                }
                        ]
                }
        }'
	*/

        strFunc := `
                function deleteRule(sgName, fromPort, toPort, protocol, direction, cidr) {

                        var connConfig = parent.frames["top_frame"].document.getElementById("connConfig").innerHTML;

                        var xhr = new XMLHttpRequest();
                        xhr.open("DELETE", "$$SPIDER_SERVER$$/spider/securitygroup/" + sgName + "/rules", false);
                        xhr.setRequestHeader('Content-Type', 'application/json');
                        sendJson = '{ "ConnectionName": "' + connConfig + '",'
                        sendJson += ' "ReqInfo": {'
                        sendJson += ' "RuleInfoList" : '
                        sendJson += '       [  '
			sendJson += '         { "FromPort": "' + fromPort + '", '
			sendJson += '           "ToPort": "' + toPort + '", '
			sendJson += '           "IPProtocol": "' + protocol + '", '
			sendJson += '           "Direction": "' + direction + '", '
			sendJson += '           "CIDR": "' + cidr + '"'
			sendJson += '         }'
                        sendJson += '       ]  '
                        sendJson += '   } '
                        sendJson += '}'

                         // client logging
                        parent.frames["log_frame"].Log("curl -sX DELETE " + "$$SPIDER_SERVER$$/spider/securitygroup/" + sgName + "/rules" + " -H 'Content-Type: application/json' -d '" + sendJson + "'");

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
func makePostRuleFunc_js() string {
        /*
        curl -sX POST http://localhost:1024/spider/securitygroup/${SG_NAME}/rules -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName": "'${CONN_CONFIG}'",
                "ReqInfo": {
                "RuleInfoList" :
                        [
                                {
                                        "Direction": "inbound",
                                        "IPProtocol": "ALL",
                                        "FromPort": "-1",
                                        "ToPort": "-1",
                                        "CIDR" : "0.0.0.0/0"
                                }
                        ]
                }
        }'
        */

        strFunc := `
                function postRule(sgName, rule) {
                        var connConfig = parent.frames["top_frame"].document.getElementById("connConfig").innerHTML;
                        var textbox = document.getElementById('security_text_box_' + sgName);

                        var xhr = new XMLHttpRequest();
                        xhr.open("POST", "$$SPIDER_SERVER$$/spider/securitygroup/" + sgName + "/rules", false);
                        xhr.setRequestHeader('Content-Type', 'application/json');
                        sendJson = '{ "ConnectionName": "' + connConfig + '",'
                        sendJson += ' "ReqInfo": {'
                        sendJson += ' "RuleInfoList" : '
                        sendJson += '       [  '
                        sendJson += textbox.value
                        sendJson += '       ]  '
                        sendJson += '   } '
                        sendJson += '}'


                         // client logging
                        parent.frames["log_frame"].Log("curl -sX POST " + "$$SPIDER_SERVER$$/spider/securitygroup/" + sgName + "/rules" + " -H 'Content-Type: application/json' -d '" + sendJson + "'");

                        xhr.send(sendJson);

                        // client logging
                        parent.frames["log_frame"].Log("   => " + xhr.response);

                        location.reload();
                }
        `
        strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://"+cr.ServiceIPorName+cr.ServicePort) // cr.ServicePort = ":1024"
        return strFunc
}

func SecurityGroup(c echo.Context) error {
	cblog.Info("call SecurityGroup()")

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
	htmlStr += makePostSecurityGroupFunc_js()
	htmlStr += makeDeleteSecurityGroupFunc_js()
	htmlStr += makePostRuleFunc_js()
	htmlStr += makeDeleteRuleFunc_js()

	htmlStr += `
                    </script>
                </head>

                <body>
                    <table border="0" bordercolordark="#F8F8FF" cellpadding="0" cellspacing="1" bgcolor="#FFFFFF"  style="font-size:small;">
                `

	// (2) make Table Action TR
	// colspan, f5_href, delete_href, fontSize
	//htmlStr += makeActionTR_html("6", "securitygroup", "deleteSecurityGroup()", "2")
	htmlStr += makeActionTR_html("6", "", "deleteSecurityGroup()", "2")

	// (3) make Table Header TR
	nameWidthList := []NameWidth{
		{"VPC Name", "200"},
		{"SecurityGroup Name", "200"},
		{"Security Rules", "300"},
		{"Additional Info", "300"},
	}
	htmlStr += makeTitleTRList_html("#DDDDDD", "2", nameWidthList, true)

	// (4) make TR list with info list
	// (4-1) get info list

	// client logging
	htmlStr += genLoggingGETURL(connConfig, "securitygroup")

	resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "securitygroup")
	if err != nil {
		cblog.Error(err)
		// client logging
                htmlStr += genLoggingResult(err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// client logging
	htmlStr += genLoggingResult(string(resBody[:len(resBody)-1]))

	var info struct {
		ResultList []*cres.SecurityInfo `json:"securitygroup"`
	}
	json.Unmarshal(resBody, &info)

	// (4-2) make TR list with info list
	htmlStr += makeSecurityGroupTRList_html("", "", "", info.ResultList)

	// (5) make input field and add
	// attach text box for add
	nameList := vpcList(connConfig)

	htmlStr += `
                        <tr bgcolor="#FFFFFF" align="center" height="30">
                            <td>
                                    <font size=2>#</font>
                            </td>
                            <td>
		`
	// Select format of CloudOS  name=text_box, id=1
	htmlStr += makeSelect_html("", nameList, "1")

	htmlStr += `
                            </td>
                            <td>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="2" value="sg-01">
                            </td>
                            <td>
                                <textarea style="font-size:12px;text-align:center;" name="text_box" id="3" cols=50 rows=3>[ {"FromPort": "1", "ToPort" : "65535", "IPProtocol" : "tcp", "Direction" : "inbound", "CIDR" : "0.0.0.0/0" } ]</textarea>
                            </td>
                            <td>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="4" disabled value="N/A">                            
                            </td>
                            <td>
                                <a href="javascript:postSecurityGroup()">
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

//====================================== KeyPair

// number, KeyPair Name, KeyPair Info, Key User, Additional Info, checkbox
func makeKeyPairTRList_html(bgcolor string, height string, fontSize string, infoList []*cres.KeyPairInfo) string {
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
                            <font size=%s>$$KEYPAIRNAME$$</font>
                    </td>
                    <td align="left">
                            <font size=%s>$$KEYINFO$$</font>
                    </td>
                    <td>
                            <font size=%s>$$KEYUSER$$</font>
                    </td>
                    <td>
                            <font size=%s>$$ADDITIONALINFO$$</font>
                    </td>
                    <td>
                        <input type="checkbox" name="check_box" value=$$KEYPAIRNAME$$>
                    </td>
                </tr>
                `, bgcolor, height, fontSize, fontSize, fontSize, fontSize, fontSize)

	strData := ""
	// set data and make TR list
	for i, one := range infoList {
		str := strings.ReplaceAll(strTR, "$$NUM$$", strconv.Itoa(i+1))
		str = strings.ReplaceAll(str, "$$KEYPAIRNAME$$", one.IId.NameId)
		// KeyPair Info: Fingerprint, PrivateKey, PublicKey
		runes := []rune(one.Fingerprint)
		fingerPrint := string(runes[0:12]) + "XXXXXXXXXXX"
		runes = []rune(one.PrivateKey)
		privateKey := string(runes[0:12]) + "XXXXXXXXXXX"
		runes = []rune(one.PublicKey)
		publicKey := string(runes[0:12]) + "XXXXXXXXXXX"
		keyInfo := "&nbsp;* Fingerprint: " + fingerPrint + "<br>"
		keyInfo += "&nbsp;* PrivateKey: " + privateKey + "<br>"
		keyInfo += "&nbsp;* PublicKey: " + publicKey
		str = strings.ReplaceAll(str, "$$KEYINFO$$", keyInfo)
		str = strings.ReplaceAll(str, "$$KEYUSER$$", one.VMUserID)

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
func makePostKeyPairFunc_js() string {

	//curl -sX POST http://localhost:1024/spider/keypair -H 'Content-Type: application/json'
	//      -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": { "Name": "keypair-01" } }'

	strFunc := `
                function postKeyPair() {
                        var connConfig = parent.frames["top_frame"].document.getElementById("connConfig").innerHTML;

                        var textboxes = document.getElementsByName('text_box');
            sendJson = '{ "ConnectionName" : "' + connConfig + '", "ReqInfo" : { "Name" : "$$KEYPAIRNAME$$"}}'

                        for (var i = 0; i < textboxes.length; i++) { // @todo make parallel executions
                                switch (textboxes[i].id) {
                                        case "1":
                                                sendJson = sendJson.replace("$$KEYPAIRNAME$$", textboxes[i].value);
                                                break;
                                        default:
                                                break;
                                }
                        }
                        var xhr = new XMLHttpRequest();
                        xhr.open("POST", "$$SPIDER_SERVER$$/spider/keypair", false);
                        xhr.setRequestHeader('Content-Type', 'application/json');

			// client logging
			parent.frames["log_frame"].Log("curl -sX POST " + "$$SPIDER_SERVER$$/spider/keypair -H 'Content-Type: application/json' -d '" + sendJson + "'");

                        xhr.send(sendJson);

			// client logging
			parent.frames["log_frame"].Log("   ==> " + xhr.response);
			var jsonVal = JSON.parse(xhr.response)

//---------------- download this private key 
		  var keyFileName = jsonVal.IId.NameId + ".pem";
		  var keyValue = jsonVal.PrivateKey;
                  var tempElement = document.createElement('a');
                  //tempElement.setAttribute('href','data:text/plain;charset=utf-8, ' + encodeURIComponent(keyValue));
                  tempElement.setAttribute('href','data:text/plain;charset=utf-8,' + encodeURIComponent(keyValue));
                  tempElement.setAttribute('download', keyFileName);
                  document.body.appendChild(tempElement);
                  tempElement.click();
                  document.body.removeChild(tempElement);
//---------------- download this private key 

            location.reload();
                }
        `
	strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://"+cr.ServiceIPorName+cr.ServicePort) // cr.ServicePort = ":1024"
	return strFunc
}

// make the string of javascript function
func makeDeleteKeyPairFunc_js() string {
	// curl -sX DELETE http://localhost:1024/spider/keypair/keypair-01 -H 'Content-Type: application/json'
	//           -d '{ "ConnectionName": "'${CONN_CONFIG}'"}'

	strFunc := `
                function deleteKeyPair() {
                        var connConfig = parent.frames["top_frame"].document.getElementById("connConfig").innerHTML;
                        var checkboxes = document.getElementsByName('check_box');
                        for (var i = 0; i < checkboxes.length; i++) { // @todo make parallel executions
                                if (checkboxes[i].checked) {
                                        var xhr = new XMLHttpRequest();
                                        xhr.open("DELETE", "$$SPIDER_SERVER$$/spider/keypair/" + checkboxes[i].value, false);
                                        xhr.setRequestHeader('Content-Type', 'application/json');
					sendJson = '{ "ConnectionName": "' + connConfig + '"}'

					// client logging
					parent.frames["log_frame"].Log("curl -sX DELETE " + "$$SPIDER_SERVER$$/spider/keypair/" + checkboxes[i].value + " -H 'Content-Type: application/json' -d '" + sendJson + "'");

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

func KeyPair(c echo.Context) error {
	cblog.Info("call KeyPair()")

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
	htmlStr += makePostKeyPairFunc_js()
	htmlStr += makeDeleteKeyPairFunc_js()

	htmlStr += `
                    </script>
                </head>

                <body>
                    <table border="0" bordercolordark="#F8F8FF" cellpadding="0" cellspacing="1" bgcolor="#FFFFFF"  style="font-size:small;">
                `

	// (2) make Table Action TR
	// colspan, f5_href, delete_href, fontSize
	//htmlStr += makeActionTR_html("6", "keypair", "deleteKeyPair()", "2")
	htmlStr += makeActionTR_html("6", "", "deleteKeyPair()", "2")

	// (3) make Table Header TR
	nameWidthList := []NameWidth{
		{"KeyPair Name", "200"},
		{"KeyPair Info", "300"},
		{"Key User", "200"},
		{"Additional Info", "300"},
	}
	htmlStr += makeTitleTRList_html("#DDDDDD", "2", nameWidthList, true)

	// (4) make TR list with info list
	// (4-1) get info list

	// client logging
	htmlStr += genLoggingGETURL(connConfig, "keypair")

	resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "keypair")
	if err != nil {
		cblog.Error(err)
		// client logging
                htmlStr += genLoggingResult(err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// client logging
	htmlStr += genLoggingResult(string(resBody[:len(resBody)-1]))

	var info struct {
		ResultList []*cres.KeyPairInfo `json:"keypair"`
	}
	json.Unmarshal(resBody, &info)

	// (4-2) make TR list with info list
	htmlStr += makeKeyPairTRList_html("", "", "", info.ResultList)

	// (5) make input field and add
	// attach text box for add
	htmlStr += `
                        <tr bgcolor="#FFFFFF" align="center" height="30">
                            <td>
                                    <font size=2>#</font>
                            </td>
                            <td>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="1" value="keypair-01">
                            </td>
                            <td>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="2" disabled value="N/A">
                            </td>
                            <td>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="3" disabled value="N/A">
                            </td>
                            <td>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="4" disabled value="N/A">
                            </td>
                            <td>
                                <a href="javascript:postKeyPair()">
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

//====================================== VM

// number, VM Name/Control, VMStatus/Last Start Time, VMImage/VMSpec, VPC/Subnet/Security Group,
//         Network Interface/IP, DNS, Root Disk/Data Disk, SSH AccessPoint/Access Key/Access User Name, Additional Info, checkbox
func makeVMTRList_html(connConfig string, bgcolor string, height string, fontSize string, infoList []*cres.VMInfo) string {
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
														<font size=%s>$$VMNAME$$</font>
														<br>
														<font size=%s>$$VMCONTROL$$</font>
                    </td>
                    <td>
                            <font size=%s>$$VMSTATUS$$</font>
                            <br>
                            <font size=%s>$$LASTSTARTTIME$$</font>
                    </td>                    
                    <td>
                            <font size=%s>$$IMAGE$$</font>
                            <br>
                            <font size=%s>$$SPEC$$</font>
                    </td>
                    <td>
                            <font size=%s>$$VPC$$</font>
                            <br>
                            <font size=%s>$$SUBNET$$</font>
                            <br>
                            <font size=%s>$$SECURITYGROUP$$</font>
                    </td>
                    <td>
                            <font size=%s>$$NETWORKINTERFACE$$</font>
                            <br>
                            <font size=%s>$$PUBLICIP$$</font>
                            <br>
                            <font size=%s>$$PRIVATEIP$$</font>
                    </td>
                    <td>
                            <font size=%s>$$PUBLICDNS$$</font>
                            <br>
                            <font size=%s>$$PRIVATEDNS$$</font>
                    </td>
                    <td align=left>
                            <font size=%s>$$ROOTDISK$$</font>
                            <br>
                            <font size=%s>$$DATADISK$$</font>
                    </td>
                    <td>
                            <font size=%s>$$SSHACCESSPOINT$$</font>
                            <br>
                            <font size=%s>$$ACCESSKEY$$</font>
                            <br>
                            <font size=%s>$$ACCESSUSER$$</font>
                    </td>
                    <td>
                            <font size=%s>$$ADDITIONALINFO$$</font>
                    </td>
                    <td>
                        <input type="checkbox" name="check_box" value=$$VMNAME$$>
                    </td>
                </tr>
                `, bgcolor, height, fontSize, fontSize, fontSize, fontSize, fontSize, fontSize, fontSize, fontSize, fontSize, fontSize,
		fontSize, fontSize, fontSize, fontSize, fontSize, fontSize, fontSize, fontSize, fontSize)

	strData := ""
	// set data and make TR list
	for i, one := range infoList {
		str := strings.ReplaceAll(strTR, "$$NUM$$", strconv.Itoa(i+1))
		str = strings.ReplaceAll(str, "$$VMNAME$$", one.IId.NameId)
		status := vmStatus(connConfig, one.IId.NameId)
		str = strings.ReplaceAll(str, "$$VMSTATUS$$", status)

		if cres.VMStatus(status) == cres.Running {
			str = strings.ReplaceAll(str, "$$VMCONTROL$$", `<span id="vmcontrol-`+one.IId.NameId+`">[<a href="javascript:vmControl('`+one.IId.NameId+`','suspend')">Suspend</a> / <a href="javascript:vmControl('`+one.IId.NameId+`','reboot')">Reboot</a>]</span>`)
		} else if cres.VMStatus(status) == cres.Suspended {
			str = strings.ReplaceAll(str, "$$VMCONTROL$$", `<span id="vmcontrol-`+one.IId.NameId+`">[<a href="javascript:vmControl('`+one.IId.NameId+`','resume')">Resume</a>]</span>`)
		} else {
			str = strings.ReplaceAll(str, "$$VMCONTROL$$", `[<span style="color:brown">vm control disabed</span>] <br> you can control when Running / Suspended. <br> try refresh page...`)
		}
		str = strings.ReplaceAll(str, "$$LASTSTARTTIME$$", one.StartTime.Format("2006.01.02 15:04:05 Mon"))

		// for Image & Spec
		str = strings.ReplaceAll(str, "$$IMAGE$$", one.ImageIId.NameId)
		str = strings.ReplaceAll(str, "$$SPEC$$", one.VMSpecName)

		// for VPC & Subnet
		str = strings.ReplaceAll(str, "$$VPC$$", one.VpcIID.NameId)
		str = strings.ReplaceAll(str, "$$SUBNET$$", one.SubnetIID.NameId)

		// for security rules info
		strSRList := ""
		for _, one := range one.SecurityGroupIIds {
/* mask for list performance
			// client logging
			strSRList += genLoggingOneGETURL(connConfig, "securitygroup", one.NameId)

			resBody, err := getResource_with_Connection_JsonByte(connConfig, "securitygroup", one.NameId)
			if err != nil {
				cblog.Error(err)
				break
			}
			// client logging
			strSRList += genLoggingResult(string(resBody[:len(resBody)-1]))

			var secInfo cres.SecurityInfo
			json.Unmarshal(resBody, &secInfo)

			strSRList += "["
			if secInfo.SecurityRules != nil {
				for _, secRuleInfo := range *secInfo.SecurityRules {
					strSRList += "{FromPort:" + secRuleInfo.FromPort + ", "
					strSRList += "ToPort:" + secRuleInfo.ToPort + ", "
					strSRList += "IPProtocol:" + secRuleInfo.IPProtocol + ", "
					strSRList += "Direction:" + secRuleInfo.Direction + ", "
					strSRList += "CIDR:" + secRuleInfo.CIDR
					strSRList += "},<br>"
				}
			}
			strSRList += "]"
*/

			strSRList += one.NameId + "," 
		}
		strSRList = strings.TrimSuffix(strSRList, ",")
		str = strings.ReplaceAll(str, "$$SECURITYGROUP$$", strSRList)

		// for Network Interface & PublicIP & PrivateIP
		str = strings.ReplaceAll(str, "$$NETWORKINTERFACE$$", one.NetworkInterface)
		str = strings.ReplaceAll(str, "$$PUBLICIP$$", one.PublicIP)
		str = strings.ReplaceAll(str, "$$PRIVATEIP$$", one.PrivateIP)

		// for Public DNS & Private DNS
		str = strings.ReplaceAll(str, "$$PUBLICDNS$$", one.PublicDNS)
		str = strings.ReplaceAll(str, "$$PRIVATEDNS$$", one.PrivateDNS)

		// for Root Disk & Data Disk
		str = strings.ReplaceAll(str, "$$ROOTDISK$$", "&nbsp;* " + one.RootDeviceName + 
			" (" + one.RootDiskType + ":" + one.RootDiskSize + "GB)" )

		dataDiskList := ""
		if len(one.DataDiskIIDs) > 0 {
			dataDiskList = "<br>&nbsp;&nbsp;&nbsp;------ Data Disk ------<br>"
		}else {
			dataDiskList = "<br>&nbsp;&nbsp;&nbsp;------ No Data Disk ------<br><br>"
		}

		for _, disk := range one.DataDiskIIDs {
			diskInfo := diskInfo(connConfig, disk.NameId)
			dataDiskList += "&nbsp;* " + disk.NameId + "(" + diskInfo.DiskType + ":" + diskInfo.DiskSize + "GB)"
			dataDiskList += "<br>"
		}
		str = strings.ReplaceAll(str, "$$DATADISK$$", dataDiskList)

		// for SSH AccessPoint & Access Key & Access User
		str = strings.ReplaceAll(str, "$$SSHACCESSPOINT$$", "<mark>" + one.SSHAccessPoint + "</mark>")
		str = strings.ReplaceAll(str, "$$ACCESSKEY$$", "<mark>" + one.KeyPairIId.NameId + "</mark>")
		str = strings.ReplaceAll(str, "$$ACCESSUSER$$", "<mark>" + one.VMUserId + "</mark>")

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
func makeVMControlFunc_js() string {
	//curl -sX PUT http://localhost:1024/spider/controlvm/vm-01?action=suspend -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}'

	strFunc := `
                function vmControl(vmName, action) {
                        var connConfig = parent.frames["top_frame"].document.getElementById("connConfig").innerHTML;

			document.getElementById("vmcontrol-" + vmName).innerHTML = '<span style="color:red">Waiting...</span>';
			setTimeout(function(){
				var xhr = new XMLHttpRequest();
				xhr.open("PUT", "$$SPIDER_SERVER$$/spider/controlvm/" + vmName + "?action=" + action, false);
				xhr.setRequestHeader('Content-Type', 'application/json');
				sendJson = '{ "ConnectionName": "' + connConfig + '"}'

				// client logging
				parent.frames["log_frame"].Log("PUT> " + "$$SPIDER_SERVER$$/spider/controlvm/" + vmName + "?action=" + action + " -H 'Content-Type: application/json' -d '" + sendJson + "'");

				xhr.send(sendJson);

				// client logging
				parent.frames["log_frame"].Log("   ==> " + xhr.response);

				location.reload();
			}, 10);
                }
        `
	strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://"+cr.ServiceIPorName+cr.ServicePort) // cr.ServicePort = ":1024"
	return strFunc
}

// make the string of javascript function
func makePostVMFunc_js() string {

	// curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json'
	//  -d '{ "ConnectionName": "'${CONN_CONFIG}'",
	//  "ReqInfo": { "Name": "vm-01", "ImageName": "ami-00978328f54e31526", "VPCName": "vpc-01",
	//  "SubnetName": "subnet-01", "SecurityGroupNames": [ "sg-01" ], "VMSpecName": "t2.micro", "KeyPairName": "keypair-01"} }'

	strFunc := `
                function postVM() {
                        var connConfig = parent.frames["top_frame"].document.getElementById("connConfig").innerHTML;
                        var textboxes = document.getElementsByName('text_box');
                        sendJson = '{ "ConnectionName" : "' + connConfig + '", "ReqInfo" : { "Name" : "$$VMNAME$$", \
                                "ImageName" : "$$IMAGE$$", "VMSpecName" : "$$SPEC$$", "VPCName" : "$$VPC$$", "SubnetName" : "$$SUBNET$$", \
                                "SecurityGroupNames" : $$SECURITYGROUP$$, "DataDiskNames" : [$$DATADISK$$], "KeyPairName" : "$$ACCESSKEY$$", "VMUserId" : "$$ACCESSUSER$$", "VMUserPasswd" : "$$ACCESSPASSWD$$" }}'

                        for (var i = 0; i < textboxes.length; i++) { // @todo make parallel executions
                                switch (textboxes[i].id) {
                                        case "1":
                                                sendJson = sendJson.replace("$$VMNAME$$", textboxes[i].value);
                                                break;
                                        case "3":
                                                sendJson = sendJson.replace("$$IMAGE$$", textboxes[i].value);
                                                break;
                                        case "4":
                                                sendJson = sendJson.replace("$$SPEC$$", textboxes[i].value);
                                                break;
                                        case "5":
                                                sendJson = sendJson.replace("$$VPC$$", textboxes[i].value);
                                                break;
                                        case "6":
                                                sendJson = sendJson.replace("$$SUBNET$$", textboxes[i].value);
                                                break;
                                        case "7":
                                                sendJson = sendJson.replace("$$SECURITYGROUP$$", textboxes[i].value);
                                                break;
                                        case "10":
						diskList = getSelectDisk(textboxes[i])
                                                sendJson = sendJson.replace("$$DATADISK$$", diskList);
                                                break;
                                        case "11":
                                                sendJson = sendJson.replace("$$ACCESSKEY$$", textboxes[i].value);
                                                break;
                                        case "12":
                                                sendJson = sendJson.replace("$$ACCESSUSER$$", textboxes[i].value);
                                                break;
                                        case "13":
                                                sendJson = sendJson.replace("$$ACCESSPASSWD$$", textboxes[i].value);
                                                break;

                                        default:
                                                break;
                                }
                        }
                        var xhr = new XMLHttpRequest();
                        xhr.open("POST", "$$SPIDER_SERVER$$/spider/vm", false);
                        xhr.setRequestHeader('Content-Type', 'application/json');

			// client logging
			parent.frames["log_frame"].Log("curl -sX POST " + "$$SPIDER_SERVER$$/spider/vm -H 'Content-Type: application/json' -d '" + sendJson + "'");

                        xhr.send(sendJson);

			// client logging
			parent.frames["log_frame"].Log("   ==> " + xhr.response);

			location.reload();
                }

		function getSelectDisk(select) {
			  if (select.tagName != 'SELECT') {
				  return ""
			  }
			  var result = [];
			  var options = select && select.options;
			  var opt;

			  for (var i=0, iLen=options.length; i<iLen; i++) {
			    opt = options[i];

			    if (opt.selected) {
			      result.push(opt.value || opt.text);
			    }
			  }

			  if (result.length < 1) {
			    return ""
			  }

			  diskList = ""
			  for ( var j=0, jLen=result.length; j<jLen; j++) {
				if (j==0) {
					diskList += '"' + result[j] + '"'
				} else {
					diskList += ', "' + result[j] + '"'
				}
			  }
			  return diskList;
		}

        `
	strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://"+cr.ServiceIPorName+cr.ServicePort) // cr.ServicePort = ":1024"
	return strFunc
}

// make the string of javascript function
func makeDeleteVMFunc_js() string {
	// curl -sX DELETE http://localhost:1024/spider/vm/vm-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}'

	strFunc := `
                function deleteVM() {
                        var connConfig = parent.frames["top_frame"].document.getElementById("connConfig").innerHTML;
                        var checkboxes = document.getElementsByName('check_box');
                        for (var i = 0; i < checkboxes.length; i++) { // @todo make parallel executions
                                if (checkboxes[i].checked) {
                                        var xhr = new XMLHttpRequest();
                                        xhr.open("DELETE", "$$SPIDER_SERVER$$/spider/vm/" + checkboxes[i].value, false);
                                        xhr.setRequestHeader('Content-Type', 'application/json');
					sendJson = '{ "ConnectionName": "' + connConfig + '"}'

					// client logging
					parent.frames["log_frame"].Log("curl -sX DELETE " + "$$SPIDER_SERVER$$/spider/vm/" + checkboxes[i].value + " -H 'Content-Type: application/json' -d '" + sendJson + "'");

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

func VM(c echo.Context) error {
	cblog.Info("call VM()")

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
	htmlStr += makePostVMFunc_js()
	htmlStr += makeDeleteVMFunc_js()
	htmlStr += makeVMControlFunc_js()

	htmlStr += `
                    </script>
                </head>

                <body>
                    <table border="0" bordercolordark="#F8F8FF" cellpadding="0" cellspacing="1" bgcolor="#FFFFFF"  style="font-size:small;">
                `

	// (2) make Table Action TR
	// colspan, f5_href, delete_href, fontSize
	htmlStr += makeActionTR_html("11", "", "deleteVM()", "2")

	// (3) make Table Header TR
	nameWidthList := []NameWidth{
		{"VM Name / Control", "200"},
		{"VM Status / Last Start Time", "200"},
		{"VM Image / VM Spec", "200"},
		{"VPC / Subnet / Security Group", "400"},
		{"NetworkInterface / PublicIP / PrivateIP", "400"},
		{"PublicDNS / PrivateDNS", "350"},
		{"RootDisk / DataDisk", "250"},
		{"SSH AccessPoint / Access Key / Access User", "200"},
		{"Additional Info", "300"},
	}
	htmlStr += makeTitleTRList_html("#DDDDDD", "2", nameWidthList, true)

	// (4) make TR list with info list
	// (4-1) get info list

	// client logging
	htmlStr += genLoggingGETURL(connConfig, "vm")

	resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "vm")
	if err != nil {
		cblog.Error(err)
		// client logging
                htmlStr += genLoggingResult(err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// client logging
	htmlStr += genLoggingResult(string(resBody[:len(resBody)-1]))

	var info struct {
		ResultList []*cres.VMInfo `json:"vm"`
	}
	json.Unmarshal(resBody, &info)

	// (4-2) make TR list with info list
	htmlStr += makeVMTRList_html(connConfig, "", "", "", info.ResultList)

	// (5) make input field and add
	// attach text box for add
	nameList := vpcList(connConfig)
	keyNameList := keyPairList(connConfig)
	diskNameList := availableDataDiskList(connConfig)
	providerName, _ := getProviderName(connConfig)

	imageName := ""
	specName := ""
	subnetName := ""
	sgName := ""
	vmUser := "" // AWS:ec2-user, Azure&GCP:cb-user, Alibaba&Cloudit:root, OpenStack: ubuntu
	switch providerName {
	case "AWS":
		imageName = "ami-00978328f54e31526"
		specName = "t2.micro"
		subnetName = "subnet-01"
		sgName = `["sg-01"]`
		vmUser = "cb-user"
	case "AZURE":
		imageName = "Canonical:UbuntuServer:18.04-LTS:latest"
		specName = "Standard_B1ls"
		subnetName = "subnet-01"
		sgName = `["sg-01"]`
		vmUser = "cb-user"
	case "GCP":
		imageName = "https://www.googleapis.com/compute/v1/projects/ubuntu-os-cloud/global/images/ubuntu-minimal-1804-bionic-v20191024"
		specName = "f1-micro"
		subnetName = "subnet-01"
		sgName = `["sg-01"]`
		vmUser = "cb-user"
	case "ALIBABA":
		imageName = "ubuntu_18_04_x64_20G_alibase_20220322.vhd"
		specName = "ecs.t5-lc1m2.small"
		subnetName = "subnet-01"
		sgName = `["sg-01"]`
		vmUser = "cb-user"
	case "TENCENT":
		imageName = "img-pi0ii46r"
		specName = "S5.MEDIUM8"
		subnetName = "subnet-01"
		sgName = `["sg-01"]`
		vmUser = "cb-user"
	case "IBM":
		imageName = "r014-a044e2f5-dfe1-416c-8990-5dc895352728"
		specName = "bx2-2x8"
		subnetName = "subnet-01"
		sgName = `["sg-01"]`
		vmUser = "cb-user"

	case "CLOUDIT":
		imageName = "ee441331-0872-49c3-886c-1873a6e32e09"
		specName = "small-2"
		subnetName = "subnet-01"
		sgName = `["sg-01"]`
		vmUser = "cb-user"
	case "OPENSTACK":
		imageName = "ubuntu18.04"
		specName = "DS-Demo"
		subnetName = "subnet-01"
		sgName = `["sg-01"]`
		vmUser = "cb-user"
	case "NCP":
		imageName = "SPSW0LINUX000130"
		specName = "SPSVRHICPUSSD002"
		subnetName = "subnet-01"
		sgName = `["sg-01"]`
		vmUser = "cb-user"
	case "KTCLOUD":
		imageName = "97ef0091-fdf7-44e9-be79-c99dc9b1a0ad"
		specName = "d3530ad2-462b-43ad-97d5-e1087b952b7d!87c0a6f6-c684-4fbe-a393-d8412bcf788d_disk100GB"
		subnetName = "subnet-01"
		sgName = `["sg-01"]`
		vmUser = "cb-user"
	case "NHNCLOUD":
		imageName = "5396655e-166a-4875-80d2-ed8613aa054f"
		specName = "m2.c4m8"
		subnetName = "subnet-01"
		sgName = `["sg-01"]`
		vmUser = "cb-user"

	case "DOCKER":
		imageName = "nginx:latest"
		specName = "NA"
		subnetName = "NA"
		sgName = `["NA"]`
		vmUser = "cb-user"
	case "MOCK":
		imageName = "mock-vmimage-01"
		subnetName = "subnet-01"
		sgName = `["sg-01"]`
		specName = "mock-vmspec-01"
		vmUser = "cb-user"
	case "CLOUDTWIN":
		imageName = "ubuntu18.04-sshd-systemd"
		subnetName = "subnet-01"
		sgName = `["sg-01"]`
		specName = "spec-1"
		vmUser = "cb-user"
	default:
		imageName = "ami-00978328f54e31526"
		specName = "t2.micro"
		subnetName = "subnet-01"
		sgName = `["sg-01"]`
		vmUser = "cb-user"
	}

	htmlStr += `
                        <tr bgcolor="#FFFFFF" align="center" height="30">
                            <td>
                                    <font size=2>#</font>
                            </td>
                            <td style="vertical-align:top">
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="1" value="vm-01">
                            </td>
                            <td style="vertical-align:top">
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="2" disabled value="N/A">
                            </td>
                            <td style="vertical-align:top">
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="3" value="$$IMAGENAME$$">
			        <br>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="4" value="$$SPECNAME$$">
                            </td>
                            <td style="vertical-align:top">
			    `
	// Select format of VPC  name=text_box, id=5
	htmlStr += makeSelect_html("", nameList, "5")

	htmlStr += `

				<br>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="6" value="$$SUBNETNAME$$">
				<br>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="7" value=$$SGNAME$$>
                            </td>
                            <td style="vertical-align:top">
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="8" disabled value="N/A">
                            </td>
                            <td style="vertical-align:top">
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="9" disabled value="N/A">
                            </td>
                            <td style="vertical-align:top">
                            `
        // Select format of Data  name=text_box, id=10
        htmlStr += makeDataDiskSelect_html("", diskNameList, "10")

        htmlStr += `
                            </td>
                            <td style="vertical-align:top">
			    `
	// Select format of KeyPair  name=text_box, id=11
	htmlStr += makeKeyPairSelect_html("", keyNameList, "11")

	htmlStr += `
				<br>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="12" value="$$VMUSER$$" disabled>
				<br>
                                <input style="font-size:12px;text-align:center;" type="password" name="text_box" id="13" value="" disabled>
                            </td>
                            <td style="vertical-align:top">
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="14" disabled value="N/A">
                            </td>
                            <td>
                                <a href="javascript:postVM()">
                                    <font size=4><mark><b>+</b></mark></font>
                                </a>
                            </td>
                        </tr>
                `

	// set imageName & specName & vmUser
	htmlStr = strings.ReplaceAll(htmlStr, "$$IMAGENAME$$", imageName)
	htmlStr = strings.ReplaceAll(htmlStr, "$$SPECNAME$$", specName)
	htmlStr = strings.ReplaceAll(htmlStr, "$$SUBNETNAME$$", subnetName)
	htmlStr = strings.ReplaceAll(htmlStr, "$$SGNAME$$", sgName)
	htmlStr = strings.ReplaceAll(htmlStr, "$$VMUSER$$", vmUser)

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


//====================================== NLB: Network Load Balancer

// number, VPC Name, NLB Name, Type, Scope, 
// Listner(IP/Protocol/Port), VMGroup(Protocol/Port/VMs), HealthChecker(Protocol/Port/Interval/Timeoute/Threshold),
// Additional Info, checkbox
func makeNLBTRList_html(bgcolor string, height string, fontSize string, infoList []*cres.NLBInfo) string {
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
                            <font size=%s>$$NLBNAME$$</font>
                    </td>                    
                    <td>
                            <font size=%s>$$TYPE$$</font>
                    </td>
                    <td>
                            <font size=%s>$$SCOPE$$</font>
                    </td>
		    <td>
                            <font size=%s>$$LISTENER$$</font>
                    </td>
                    <td>
                            <font size=%s>$$VMGROUP$$</font>
                    </td>
		    <td>
                            <font size=%s>$$HEALTHCHECKER$$</font>
                    </td>
                    <td>
                            <font size=%s>$$ADDITIONALINFO$$</font>
                    </td>
                    <td>
                        <input type="checkbox" name="check_box" value=$$NLBNAME$$>
                    </td>
                </tr>
                `, bgcolor, height, fontSize, fontSize, fontSize, fontSize, fontSize, fontSize, fontSize, fontSize, fontSize)

	strData := ""
	// set data and make TR list
	for i, one := range infoList {
		str := strings.ReplaceAll(strTR, "$$NUM$$", strconv.Itoa(i+1))
		str = strings.ReplaceAll(str, "$$VPCNAME$$", one.VpcIID.NameId)
		str = strings.ReplaceAll(str, "$$NLBNAME$$", one.IId.NameId)
		str = strings.ReplaceAll(str, "$$TYPE$$", one.Type)
		str = strings.ReplaceAll(str, "$$SCOPE$$", one.Scope)

		// for Listener info
		// for Listener KeyValueList
		strKeyList := ""
		for _, kv := range one.Listener.KeyValueList {
			strKeyList += kv.Key + ":" + kv.Value + ", "
		}
		strKeyList = strings.TrimRight(strKeyList, ", ")
		strListener := ""
		strListener += "<b> => <mark>" + one.Listener.IP + ":" + one.Listener.Port + "</b> </mark> <br>"
		if one.Listener.DNSName != "" {
			strListener += "=> <mark> <b>" + one.Listener.DNSName + ":" + one.Listener.Port + "</b> </mark> <br>"
		}
		strListener += "--------------------------------<br>"
		//if one.Listener.CspID != "" {
		//	strListener += "CspID:" + one.Listener.CspID + ", "
		//}
		/* complicated to see 
		if strKeyList != "" {
			strListener += "(etc) " + strKeyList + "<br>"
			strListener += "--------------------------------<br>"
		}
		*/
		strListener += one.Listener.Protocol + "<br>"
                strListener += "--------------------------------<br>"
                strListener += `
                        <input disabled='true' type='button' style="font-size:11px;color:gray" onclick="javascript:setListener('` + one.IId.NameId + `');" value='edit'/>
                        <br>
			`
		
		str = strings.ReplaceAll(str, "$$LISTENER$$", strListener)

		// for VMGroup info
		// for VMGroup KeyValueList
		strKeyList = ""
		for _, kv := range one.VMGroup.KeyValueList {
			strKeyList += kv.Key + ":" + kv.Value + ", "
		}
		strKeyList = strings.TrimRight(strKeyList, ", ")
		strVMList := ""
		for _, vmIID := range *one.VMGroup.VMs {
			strVMList += vmIID.NameId + ", "
		}
		strVMList = strings.TrimRight(strVMList, ", ")
		strVMGroup := ""
		strVMGroup += "<b> => <mark>" + one.VMGroup.Port + "</b> </mark> <br>"
		strVMGroup += "--------------------------------<br>"
		strVMGroup += "<mark> [ " + strVMList + " ] </mark>" + "<br>"
		strVMGroup += "--------------------------------<br>"
		//if one.VMGroup.CspID != "" {
		//	strVMGroup += "CspID:" + one.VMGroup.CspID + ", "
		//}
		/* complicated to see 
		if strKeyList != "" {
			strVMGroup += "(etc) " + strKeyList + "<br>"
			strVMGroup += "--------------------------------<br>"
		}
		*/
		strVMGroup += one.VMGroup.Protocol + "<br>"
		strVMGroup += "--------------------------------<br>"
		strVMGroup += `
			<input disabled='true' type='button' style="font-size:11px;color:gray" onclick="javascript:setVMGroup('` + one.IId.NameId + `');" value='edit'/>
			<input disabled='true' type='button' style="font-size:11px;color:gray" onclick="javascript:addVMs('` + one.IId.NameId + `');" value='+'/>
			<input disabled='true' type='button' style="font-size:11px;color:gray" onclick="javascript:removeVMs('` + one.IId.NameId + `');" value='-'/>
			<br>
			`
		
		str = strings.ReplaceAll(str, "$$VMGROUP$$", strVMGroup)

		// for HealthChecker info
		// for HealthChecker KeyValueList
		strKeyList = ""
		for _, kv := range one.HealthChecker.KeyValueList {
			strKeyList += kv.Key + ":" + kv.Value + ", "
		}
		strKeyList = strings.TrimRight(strKeyList, ", ")

		strHealthChecker := ""
/* 
		strHealthChecker := `
			<div class="displayStatus">
				<textarea id='displayStatus' hidden disabled="true" style="overflow:scroll;" wrap="off"></textarea>
			</div>
		`
*/
		strHealthChecker += "<b> <= <mark>" + one.HealthChecker.Port + "</b> </mark><br>"
		strHealthChecker += "--------------------------------<br>"
		strHealthChecker += "Interval:   " + strconv.Itoa(one.HealthChecker.Interval) + "<br>"
		strHealthChecker += "Timeout:    " + strconv.Itoa(one.HealthChecker.Timeout) + "<br>"
		strHealthChecker += "Threshold:  " + strconv.Itoa(one.HealthChecker.Threshold) + "<br>"
		strHealthChecker += "--------------------------------<br>"
		//if one.HealthChecker.CspID != "" {
		//	strHealthChecker += "CspID:" + one.HealthChecker.CspID + ", "
		//}
		/* complicated to see 
		if strKeyList != "" {
			strHealthChecker += "(etc) " + strKeyList + "<br>"
			strHealthChecker += "------------------------<br>"
		}
		*/
		strHealthChecker += one.HealthChecker.Protocol + "<br>"
		strHealthChecker += "--------------------------------<br>"
/*
		strHealthChecker += `
			<a href="javascript:healthStatus('` + one.IId.NameId + `');">
			    <font color=blue size=2><b><div class='displayText'>Status</div></b></font>
			</a><br>`
*/
		strHealthChecker += `
			<input disabled='true' type='button' style="font-size:11px;color:gray" onclick="javascript:setHealthChecker('` + one.IId.NameId + `');" value='edit'/>
			<input type='button' style="font-size:11px;color:blue" onclick="javascript:healthStatus('` + one.IId.NameId + `');" value='Status'/>
			<br>`
		
		str = strings.ReplaceAll(str, "$$HEALTHCHECKER$$", strHealthChecker)


		// for KeyValueList
		strKeyList = ""
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
func makePostNLBFunc_js() string {

// curl -sX POST http://localhost:1024/spider/nlb -H 'Content-Type: application/json' -d \
//         '{
//                 "ConnectionName": "'${CONN_CONFIG}'",
//                 "ReqInfo": {
//                         "Name": "spider-nlb-01",
//                         "VPCName": "vpc-01",
//                         "Type": "PUBLIC",
//                         "Scope": "REGION",
//                         "Listener": {
//                                 "Protocol" : "TCP",
//                                 "Port" : "80"
//                         },
//                         "VMGroup": {
//                                 "Protocol" : "TCP",
//                                 "Port" : "80",
//                                 "VMs" : ["vm-01", "vm-02"]
//                         },
//                         "HealthChecker": {
//                                 "Protocol" : "TCP",
//                                 "Port" : "80",
//                                 "Interval" : "10",
//                                 "Timeout" : "10",
//                                 "Threshold" : "3"
//                         }
//                 }
//         }'

	strFunc := `
                function postNLB() {
                        var connConfig = parent.frames["top_frame"].document.getElementById("connConfig").innerHTML;

                        var textboxes = document.getElementsByName('text_box');
                        sendJson = '{ "ConnectionName" : "' + connConfig + '", "ReqInfo" : \
                        			{ \
                        				"Name" : "$$NLBNAME$$", \
                        				"VPCName" : "$$VPCNAME$$", \
                        				"Type" : "$$TYPE$$", \
                        				"Scope" : "$$SCOPE$$", \
                        				"Listener" : { \
                        					"Protocol" : "$$L_PROTOCOL$$", \
                        					"Port" : "$$L_PORT$$" \
                        				}, \
                        				"VMGroup" : { \
                        					"Protocol" : "$$V_PROTOCOL$$", \
                        					"Port" : "$$V_PORT$$", \
                        					"VMs" : $$VMS$$ \
                        				}, \
                        				"HealthChecker" : { \
                        					"Protocol" : "$$H_PROTOCOL$$", \
                        					"Port" : "$$H_PORT$$", \
                        					"Interval" : "$$INTERVAL$$", \
                        					"Timeout" : "$$TIMEOUT$$", \
                        					"Threshold" : "$$THRESHOLD$$" \
                        				} \
                        			} \
                        		}'

                        for (var i = 0; i < textboxes.length; i++) { // @todo make parallel executions
                                switch (textboxes[i].id) {
                                        case "1":
                                                sendJson = sendJson.replace("$$VPCNAME$$", textboxes[i].value);
                                                break;
                                        case "2":
                                                sendJson = sendJson.replace("$$NLBNAME$$", textboxes[i].value);
                                                break;
                                        case "3":
                                                sendJson = sendJson.replace("$$TYPE$$", textboxes[i].value);
                                                break;
                                        case "4":
                                                sendJson = sendJson.replace("$$SCOPE$$", textboxes[i].value);
                                                break;
                                        case "5":
                                                sendJson = sendJson.replace("$$L_PROTOCOL$$", textboxes[i].value);
                                                break;
                                        case "6":
                                                sendJson = sendJson.replace("$$L_PORT$$", textboxes[i].value);
                                                break;
                                        case "7":
                                                sendJson = sendJson.replace("$$V_PROTOCOL$$", textboxes[i].value);
                                                break;
                                        case "8":
                                                sendJson = sendJson.replace("$$V_PORT$$", textboxes[i].value);
                                                break;
                                        case "9":
                                                sendJson = sendJson.replace("$$VMS$$", textboxes[i].value);
                                                break;
                                        case "10":
                                                sendJson = sendJson.replace("$$H_PROTOCOL$$", textboxes[i].value);
                                                break;
                                        case "11":
                                                sendJson = sendJson.replace("$$H_PORT$$", textboxes[i].value);
                                                break;
                                        case "12":
                                                sendJson = sendJson.replace("$$INTERVAL$$", textboxes[i].value);
                                                break;
                                        case "13":
                                                sendJson = sendJson.replace("$$TIMEOUT$$", textboxes[i].value);
                                                break;
                                        case "14":
                                                sendJson = sendJson.replace("$$THRESHOLD$$", textboxes[i].value);
                                                break;                                              
                                        default:
                                                break;
                                }
                        }
                        var xhr = new XMLHttpRequest();
                        xhr.open("POST", "$$SPIDER_SERVER$$/spider/nlb", false);
                        xhr.setRequestHeader('Content-Type', 'application/json');

			// client logging
			parent.frames["log_frame"].Log("curl -sX POST " + "$$SPIDER_SERVER$$/spider/nlb -H 'Content-Type: application/json' -d '" + sendJson + "'");

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
func makeDeleteNLBFunc_js() string {
	// curl -sX DELETE http://localhost:1024/spider/nlb/spider-nlb-01 -H 'Content-Type: application/json' -d \
 //        '{
 //                "ConnectionName": "'${CONN_CONFIG}'"
 //        }'

	strFunc := `
                function deleteNLB() {
                        var connConfig = parent.frames["top_frame"].document.getElementById("connConfig").innerHTML;
                        var checkboxes = document.getElementsByName('check_box');
                        for (var i = 0; i < checkboxes.length; i++) { // @todo make parallel executions
                                if (checkboxes[i].checked) {
                                        var xhr = new XMLHttpRequest();
                                        xhr.open("DELETE", "$$SPIDER_SERVER$$/spider/nlb/" + checkboxes[i].value, false);
                                        xhr.setRequestHeader('Content-Type', 'application/json');
					sendJson = '{ "ConnectionName": "' + connConfig + '"}'

					// client logging
					parent.frames["log_frame"].Log("curl -sX DELETE " + "$$SPIDER_SERVER$$/spider/nlb/" + checkboxes[i].value + " -H 'Content-Type: application/json' -d '" + sendJson + "'");

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

// make the string of javascript function
func makeGetHealthStatusNLBFunc_js() string {
	// curl -sX GET http://localhost:1024/spider/nlb/spider-nlb-01/health -H 'Content-Type: application/json' -d \
        // '{
        //        "ConnectionName": "'${CONN_CONFIG}'"
        // }'

        strFunc := `
		function convertHealthyInfo(org) {
			const obj = JSON.parse(org);
			var healthinfo = obj.healthinfo		
			var all = healthinfo.AllVMs		
			var text = "[All VMs]\n"
			for (let i=0; i< all.length; i++) {
				text += "\t" + all[i].NameId + "\n";
			}
			text += "\n"	
			text += "[Healthy VMs]\n"
			var healthy = healthinfo.HealthyVMs		
			for (let i=0; i< healthy.length; i++) {
				text += "\t" + healthy[i].NameId + "\n";
			}
			text += "\n"	
			text += "[UnHealthy VMs]\n"
			var unHealthy = healthinfo.UnHealthyVMs		
			for (let i=0; i< unHealthy.length; i++) {
				text += "\t" + unHealthy[i].NameId + "\n";
			}

			return text
		}

                function healthStatus(nlbName) {
                        var connConfig = parent.frames["top_frame"].document.getElementById("connConfig").innerHTML;
			var xhr = new XMLHttpRequest();
			xhr.open("GET", "$$SPIDER_SERVER$$/spider/nlb/" + nlbName + "/health?ConnectionName=" + connConfig, false);

			// client logging
			parent.frames["log_frame"].Log("curl -sX GET " + "$$SPIDER_SERVER$$/spider/nlb/" + nlbName + "/health?ConnectionName=" + connConfig);

			xhr.send();

			// client logging
			parent.frames["log_frame"].Log("   ==> " + xhr.response);

			var healthy = convertHealthyInfo(xhr.response);
			alert(healthy);
		}

        `
        strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://"+cr.ServiceIPorName+cr.ServicePort) // cr.ServicePort = ":1024"
        return strFunc
}

func NLB(c echo.Context) error {
	cblog.Info("call NLB()")

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
	htmlStr += makePostNLBFunc_js()
	htmlStr += makeDeleteNLBFunc_js()
	htmlStr += makeGetHealthStatusNLBFunc_js()

	htmlStr += `
                    </script>
                </head>

                <body>
                    <table bordercolordark="#F8F8FF" cellpadding="0" cellspacing="1" bgcolor="#FFFFFF"  style="font-size:small;">
                `

	// (2) make Table Action TR
	// colspan, f5_href, delete_href, fontSize
	htmlStr += makeActionTR_html("10", "", "deleteNLB()", "2")

	// (3) make Table Header TR
	nameWidthList := []NameWidth{
		{"VPC Name", "100"},
		{"NLB Name", "100"},
		{"NLB Type", "50"},
		{"NLB Scope", "50"},
		{"Listener", "200"},
		{"VMGroup", "200"},
		{"HealthChecker", "200"},
		{"Additional Info", "200"},
	}
	htmlStr += makeTitleTRList_html("#DDDDDD", "2", nameWidthList, true)

	// (4) make TR list with info list
	// (4-1) get info list

	// client logging
	htmlStr += genLoggingGETURL(connConfig, "nlb")

	resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "nlb")
	if err != nil {
		cblog.Error(err)
		// client logging
                htmlStr += genLoggingResult(err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// client logging
	htmlStr += genLoggingResult(string(resBody[:len(resBody)-1]))

	var info struct {
		ResultList []*cres.NLBInfo `json:"nlb"`
	}
	json.Unmarshal(resBody, &info)

	// (4-2) make TR list with info list
	htmlStr += makeNLBTRList_html("", "", "", info.ResultList)

	// (5) make input field and add
	// attach text box for add
	nameList := vpcList(connConfig)

	htmlStr += `
                        <tr bgcolor="#FFFFFF" align="center" height="30">
                            <td>
                                    <font size=2>#</font>
                            </td>
                            <td>
		`
	// Select format of VPC  name=text_box, id=1
	htmlStr += makeSelect_html("", nameList, "1")

	htmlStr += `
                            </td>
                            <td>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="2" value="spider-nlb-01">
                            </td>
                            <td>
				<select style="font-size:12px;text-align:center;"  name="text_box" id="3">
					<option value="PUBLIC">PUBLIC</option>
					<option value="INTERNAL">INTERNAL</option>
				</select>
                            </td>
                            <td>
				<select style="font-size:12px;text-align:center;"  name="text_box" id="4">
					<option value="REGION">REGION</option>
					<option value="GLOBAL">GLOBAL</option>
				</select>
                            </td>
                            <td>
                                <!--Port:--> => <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="6" maxlength="5" size="5" value="22">
				<br>--------------------------------<br>
                                <!--Protocol:-->
					<select style="font-size:12px;text-align:center;"  name="text_box" id="5">
						<option value="TCP">TCP</option>
						<option value="UDP">UDP</option>
					</select>
                            </td>
                            <td>
                                <!--Port:--> => <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="8" maxlength="5" size="5" value="22">
				<br>--------------------------------<br>
                                <!--VM:--> <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="9" value="[ &quot;vm-01&quot;, &quot;vm-02&quot; ]">
				<br>--------------------------------<br>
                                <!--Protocol:-->
					<select style="font-size:12px;text-align:center;"  name="text_box" id="7" >
						<option value="TCP">TCP</option>
						<option value="UDP">UDP</option>
						<option value="HTTP">HTTP</option>
						<option value="HTTPS">HTTPS</option>
					</select>
                            </td>
                            <td>
                                <!--Port:--> <= <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="11" maxlength="5" size="5" value="22">
				<br>--------------------------------<br>
                                Interval: <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="12" maxlength="5" size="5" value="10">
                                <br> Timeout: <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="13" maxlength="5" size="5" value="10">
                                <br> Threshold: <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="14" maxlength="5" size="5" value="3">
				<br>--------------------------------<br>
                                <!--Protocol:-->
					<select style="font-size:12px;text-align:center;"  name="text_box" id="10" >
						<option value="TCP">TCP</option>
						<option value="HTTP">HTTP</option>
						<option value="HTTPS">HTTPS</option>
					</select>
                            </td>
                            <td>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="4" disabled value="N/A">                            
                            </td>
                            <td>
                                <a href="javascript:postNLB()">
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

//====================================== VMImage

// number, VMImage Name, GuestOS, VMImage Status, KeyValueList
func makeVMImageTRList_html(bgcolor string, height string, fontSize string, infoList []*cres.ImageInfo) string {
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
                            <font size=%s>$$VMIMAGENAME$$</font>
                    </td>
                    <td align="left">
                            <font size=%s>$$GUESTOS$$</font>
                    </td>
                    <td>
                            <font size=%s>$$VMIMAGESTATUS$$</font>
                    </td>
                    <td align="left">
                            <font size=%s>$$ADDITIONALINFO$$</font>
                    </td>
                </tr>
                `, bgcolor, height, fontSize, fontSize, fontSize, fontSize, fontSize)

	strData := ""
	// set data and make TR list
	for i, one := range infoList {
		str := strings.ReplaceAll(strTR, "$$NUM$$", strconv.Itoa(i+1))
		str = strings.ReplaceAll(str, "$$VMIMAGENAME$$", one.IId.NameId)
		str = strings.ReplaceAll(str, "$$GUESTOS$$", one.GuestOS)
		str = strings.ReplaceAll(str, "$$VMIMAGESTATUS$$", one.Status)

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

func VMImage(c echo.Context) error {
	cblog.Info("call VMImage()")

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
		{"VMImage Name", "200"},
		{"GuestOS", "300"},
		{"VMImage Status", "200"},
		{"Additional Info", "400"},
	}
	htmlStr += makeTitleTRList_html("#DDDDDD", "2", nameWidthList, false)

	// (4) make TR list with info list
	// (4-1) get info list

	// client logging
	htmlStr += genLoggingGETURL(connConfig, "vmimage")

	resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "vmimage")
	if err != nil {
		cblog.Error(err)
		// client logging
                htmlStr += genLoggingResult(err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// client logging
	htmlStr += genLoggingResult(string(resBody[:len(resBody)-1]))

	var info struct {
		ResultList []*cres.ImageInfo `json:"image"`
	}
	json.Unmarshal(resBody, &info)

	// (4-2) make TR list with info list
	htmlStr += makeVMImageTRList_html("", "", "", info.ResultList)

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
                `, bgcolor, height, fontSize, fontSize, fontSize, fontSize, fontSize, fontSize, fontSize)

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
                            <td>
                                    <font size=2>#</font>
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

