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

//====================================== VPC

// number, VPC Name, VPC CIDR, SUBNET Info, Additional Info, checkbox
func makeVPCTRList_html(bgcolor string, height string, fontSize string, infoList []*cres.VPCInfo) string {
        if bgcolor == "" { bgcolor = "#FFFFFF" }
        if height == "" { height = "30" }
        if fontSize == "" { fontSize = "2" }

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

        strData := ""
        // set data and make TR list
        for i, one := range infoList{
                str := strings.ReplaceAll(strTR, "$$NUM$$", strconv.Itoa(i+1))
                str = strings.ReplaceAll(str, "$$VPCNAME$$", one.IId.NameId)
                str = strings.ReplaceAll(str, "$$VPCCIDR$$", one.IPv4_CIDR)

		// for subnet
		strSubnetList := ""
                for _, one := range one.SubnetInfoList {
                        strSubnetList += one.IId.NameId + ", "
                        strSubnetList += "CIDR:" + one.IPv4_CIDR + ", {"
			for _, kv := range one.KeyValueList {
				strSubnetList += kv.Key + ":" + kv.Value + ", "
			}
                        strSubnetList += "}<br>"
	
                }
                str = strings.ReplaceAll(str, "$$SUBNETINFO$$", strSubnetList)

		// for KeyValueList
		strKeyList := ""
                for _, kv := range one.KeyValueList {
                        strKeyList += kv.Key + ":" + kv.Value + ", "
                }
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
                        xhr.send(sendJson);

			location.reload();
                }
        `
        strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://" + cr.HostIPorName + cr.ServicePort) // cr.ServicePort = ":1024"
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
                                        xhr.open("DELETE", "$$SPIDER_SERVER$$/spider/vpc/" + checkboxes[i].value, false);
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

func VPC(c echo.Context) error {
        cblog.Info("call VPC()")

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
                htmlStr += makePostVPCFunc_js()
                htmlStr += makeDeleteVPCFunc_js()


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
                nameWidthList := []NameWidth {
                    {"VPC Name", "200"},
                    {"VPC CIDR", "200"},
                    {"Subnet Info", "300"},
                    {"Additional Info", "300"},
                }
                htmlStr +=  makeTitleTRList_html("#DDDDDD", "2", nameWidthList, true)


        // (4) make TR list with info list
        // (4-1) get info list 
                resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "vpc")
                if err != nil {
                        cblog.Error(err)
                        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
                }
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
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="2" value="192.168.0.0/16">
                            </td>
                            <td>
                                <textarea style="font-size:12px;text-align:center;" name="text_box" id="3" cols=50>[ { "Name": "subnet-01", "IPv4_CIDR": "192.168.1.0/24"} ]</textarea>
                            <td>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="4" disabled value="N/A">
                            </td>
                            </td>
                            <td>
                                <a href="javascript:postVPC()">
                                    <font size=3><b>+</b></font>
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

//====================================== Security Group

// number, VPC Name, SecurityGroup Name, Security Rules, Additional Info, checkbox
func makeSecurityGroupTRList_html(bgcolor string, height string, fontSize string, infoList []*cres.SecurityInfo) string {
        if bgcolor == "" { bgcolor = "#FFFFFF" }
        if height == "" { height = "30" }
        if fontSize == "" { fontSize = "2" }

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

        strData := ""
        // set data and make TR list
        for i, one := range infoList{
                str := strings.ReplaceAll(strTR, "$$NUM$$", strconv.Itoa(i+1))
                str = strings.ReplaceAll(str, "$$VPCNAME$$", one.VpcIID.NameId)
                str = strings.ReplaceAll(str, "$$SGNAME$$", one.IId.NameId)

        // for security rules info
        strSRList := ""
                for _, one := range *one.SecurityRules {
                        strSRList += "FromPort:" + one.FromPort + ", "
                        strSRList += "ToPort:" + one.ToPort + ", "
                        strSRList += "IPProtocol:" + one.IPProtocol + ", "
                        strSRList += "Direction:" + one.Direction + ", "
                        strSRList += "}<br>"    
                }
                str = strings.ReplaceAll(str, "$$SECURITYRULES$$", strSRList)

        // for KeyValueList
        strKeyList := ""
                for _, kv := range one.KeyValueList {
                        strKeyList += kv.Key + ":" + kv.Value + ", "
                }
                str = strings.ReplaceAll(str, "$$ADDITIONALINFO$$", strKeyList)

                strData += str
        }

        return strData
}

// make the string of javascript function
func makePostSecurityGroupFunc_js() string {

//curl -sX POST http://localhost:1024/spider/securitygroup -H 'Content-Type: application/json' 
//  -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": { "Name": "sg-01", "VPCName": "vpc-01", 
//      "SecurityRules": [ {"FromPort": "1", "ToPort" : "65535", "IPProtocol" : "tcp", "Direction" : "inbound"} ] } }' 

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
                        xhr.send(sendJson);

            location.reload();
                }
        `
        strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://" + cr.HostIPorName + cr.ServicePort) // cr.ServicePort = ":1024"
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
                                        xhr.send(sendJson);
                                }
                        }
            location.reload();
                }
        `
        strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://" + cr.HostIPorName + cr.ServicePort) // cr.ServicePort = ":1024"
        return strFunc
}

func SecurityGroup(c echo.Context) error {
        cblog.Info("call SecurityGroup()")

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
                htmlStr += makePostSecurityGroupFunc_js()
                htmlStr += makeDeleteSecurityGroupFunc_js()


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
                nameWidthList := []NameWidth {
                    {"VPC Name", "200"},
                    {"SecurityGroup Name", "200"},
                    {"Security Rules", "300"},
                    {"Additional Info", "300"},
                }
                htmlStr +=  makeTitleTRList_html("#DDDDDD", "2", nameWidthList, true)


        // (4) make TR list with info list
        // (4-1) get info list 
                resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "securitygroup")
                if err != nil {
                        cblog.Error(err)
                        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
                }
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
				htmlStr += makeSelect_html("onchangeVPC", nameList)

		htmlStr += `
                            </td>
                            <td>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="2" value="sg-01">
                            </td>
                            <td>
                                <textarea style="font-size:12px;text-align:center;" name="text_box" id="3" cols=50>[ {"FromPort": "1", "ToPort" : "65535", "IPProtocol" : "tcp", "Direction" : "inbound"} ]</textarea>
                            <td>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="4" disabled value="N/A">
                            </td>
                            </td>
                            <td>
                                <a href="javascript:postSecurityGroup()">
                                    <font size=3><b>+</b></font>
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
        if bgcolor == "" { bgcolor = "#FFFFFF" }
        if height == "" { height = "30" }
        if fontSize == "" { fontSize = "2" }

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
                        <input type="checkbox" name="check_box" value=$$VPCNAME$$>
                    </td>
                </tr>
                `, bgcolor, height, fontSize, fontSize, fontSize, fontSize, fontSize)

        strData := ""
        // set data and make TR list
        for i, one := range infoList{
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
                        xhr.send(sendJson);

            location.reload();
                }
        `
        strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://" + cr.HostIPorName + cr.ServicePort) // cr.ServicePort = ":1024"
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
                                        xhr.send(sendJson);
                                }
                        }
            location.reload();
                }
        `
        strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://" + cr.HostIPorName + cr.ServicePort) // cr.ServicePort = ":1024"
        return strFunc
}

func KeyPair(c echo.Context) error {
        cblog.Info("call KeyPair()")

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
                nameWidthList := []NameWidth {
                    {"KeyPair Name", "200"},
                    {"KeyPair Info", "300"},
                    {"Key User", "200"},
                    {"Additional Info", "300"},
                }
                htmlStr +=  makeTitleTRList_html("#DDDDDD", "2", nameWidthList, true)


        // (4) make TR list with info list
        // (4-1) get info list 
                resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "keypair")
                if err != nil {
                        cblog.Error(err)
                        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
                }
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
                            <td>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="4" disabled value="N/A">
                            </td>
                            </td>
                            <td>
                                <a href="javascript:postKeyPair()">
                                    <font size=3><b>+</b></font>
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


//====================================== VMSpec

// number, VMSpec Name, VCPU, Memory, GPU, KeyValueList
func makeVMSpecTRList_html(bgcolor string, height string, fontSize string, infoList []*cres.VMSpecInfo) string {
        if bgcolor == "" { bgcolor = "#FFFFFF" }
        if height == "" { height = "30" }
        if fontSize == "" { fontSize = "2" }

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
        for i, one := range infoList{
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
                str = strings.ReplaceAll(str, "$$ADDITIONALINFO$$", strKeyList)

                strData += str
        }

        return strData
}

func VMSpec(c echo.Context) error {
        cblog.Info("call VMSpec()")

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
                </head>

                <body>
		<br>
                    <table border="0" bordercolordark="#F8F8FF" cellpadding="0" cellspacing="1" bgcolor="#FFFFFF"  style="font-size:small;">
                `

        // (3) make Table Header TR
                nameWidthList := []NameWidth {
                    {"VMSpec Name", "200"},
                    {"VCPU", "300"},
                    {"Memory", "200"},
                    {"GPU", "300"},
                    {"Additional Info", "300"},
                }
                htmlStr +=  makeTitleTRList_html("#DDDDDD", "2", nameWidthList, false)


        // (4) make TR list with info list
        // (4-1) get info list 
                resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "vmspec")
                if err != nil {
                        cblog.Error(err)
                        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
                }
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

//====================================== VMImage

// number, VMImage Name, GuestOS, VMImage Status, KeyValueList
func makeVMImageTRList_html(bgcolor string, height string, fontSize string, infoList []*cres.ImageInfo) string {
        if bgcolor == "" { bgcolor = "#FFFFFF" }
        if height == "" { height = "30" }
        if fontSize == "" { fontSize = "2" }

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
                            <font size=%s>$$VMIMAGESTATUS$$ MB</font>
                    </td>
                    <td align="left">
                            <font size=%s>$$ADDITIONALINFO$$</font>
                    </td>
                </tr>
                `, bgcolor, height, fontSize, fontSize, fontSize, fontSize, fontSize, fontSize)

        strData := ""
        // set data and make TR list
        for i, one := range infoList{
                str := strings.ReplaceAll(strTR, "$$NUM$$", strconv.Itoa(i+1))
                str = strings.ReplaceAll(str, "$$VMIMAGENAME$$", one.IID.NameId)
                str = strings.ReplaceAll(str, "$$GUESTOS$$", one.GuestOS)
                str = strings.ReplaceAll(str, "$$VMIMAGESTATUS$$", one.Status)

        // for KeyValueList
        strKeyList := ""
                for _, kv := range one.KeyValueList {
                        strKeyList += kv.Key + ":" + kv.Value + ", "
                }
                str = strings.ReplaceAll(str, "$$ADDITIONALINFO$$", strKeyList)

                strData += str
        }

        return strData
}

func VMImage(c echo.Context) error {
        cblog.Info("call VMImage()")

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
                </head>

                <body>
        <br>
                    <table border="0" bordercolordark="#F8F8FF" cellpadding="0" cellspacing="1" bgcolor="#FFFFFF"  style="font-size:small;">
                `

        // (3) make Table Header TR
                nameWidthList := []NameWidth {
                    {"VMImage Name", "200"},
                    {"GuestOS", "300"},
                    {"VMImage Status", "200"},
                    {"Additional Info", "400"},
                }
                htmlStr +=  makeTitleTRList_html("#DDDDDD", "2", nameWidthList, false)


        // (4) make TR list with info list
        // (4-1) get info list 
                resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "vmimage")
                if err != nil {
                        cblog.Error(err)
                        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
                }
                var info struct {
                        ResultList []*cres.ImageInfo `json:"vmimage"`
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
