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
                            <td bgcolor="#FFEFBA">
                                    <font size=2>&nbsp;create:&nbsp;</font>
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
