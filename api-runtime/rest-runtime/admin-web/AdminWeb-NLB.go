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
                            <td bgcolor="#FFEFBA">
                                    <font size=2>&nbsp;create:&nbsp;</font>
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
                                Interval: <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="12" maxlength="5" size="5" value="default">
                                <br> Timeout: <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="13" maxlength="5" size="5" value="default">
                                <br> Threshold: <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="14" maxlength="5" size="5" value="default">
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
