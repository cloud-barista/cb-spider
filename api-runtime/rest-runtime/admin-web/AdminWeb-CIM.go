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
	dim "github.com/cloud-barista/cb-spider/cloud-info-manager/driver-info-manager"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager/credential-info-manager"
	rim "github.com/cloud-barista/cb-spider/cloud-info-manager/region-info-manager"
	ccim "github.com/cloud-barista/cb-spider/cloud-info-manager/connection-config-info-manager"
	"strconv"

	"net/http"
	"strings"
	"github.com/labstack/echo"
	"encoding/json"
)

// number, Provider Name, Driver File, Driver Name, checkbox
func makeDriverTRList_html(bgcolor string, height string, fontSize string, infoList []*dim.CloudDriverInfo) string {
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
                            <font size=%s>$$S1$$</font>
                    </td>
                    <td>
                            <font size=%s>$$S2$$</font>
                    </td>
                    <td>
                            <font size=%s>$$S3$$</font>
                    </td>
                    <td>
                        <input type="checkbox" name="check_box" value=$$S3$$>
                    </td>
                </tr>
       		`, bgcolor, height, fontSize, fontSize, fontSize, fontSize) 

        strData := ""
	// set data and make TR list
        for i, one := range infoList{
                str := strings.ReplaceAll(strTR, "$$NUM$$", strconv.Itoa(i+1))
                str = strings.ReplaceAll(str, "$$S1$$", one.ProviderName)
                str = strings.ReplaceAll(str, "$$S2$$", one.DriverLibFileName)
                str = strings.ReplaceAll(str, "$$S3$$", one.DriverName)
                strData += str
        }

	return strData
}

// make the string of javascript function
func makeOnchangeDriverProviderFunc_js() string {
        strFunc := `
              function onchangeProvider(source) {
                var providerName = source.value
                document.getElementById('2').value= providerName.toLowerCase() + "-driver-v1.0.so";
                document.getElementById('3').value= providerName.toLowerCase() + "-driver-01";
              }
        `

        return strFunc
}

// make the string of javascript function
func makeCheckBoxToggleFunc_js() string {

        strFunc := `
              function toggle(source) {
                var checkboxes = document.getElementsByName('check_box');
                for (var i = 0; i < checkboxes.length; i++) {
                  checkboxes[i].checked = source.checked;
                }
              }
        `

        return strFunc
}

// make the string of javascript function
func makePostDriverFunc_js() string {

// curl -X POST http://$RESTSERVER:1024/spider/driver -H 'Content-Type: application/json'  -d '{"DriverName":"aws-driver01","ProviderName":"AWS", "DriverLibFileName":"aws-driver-v1.0.so"}'

        strFunc := `
                function postDriver() {
                        var textboxes = document.getElementsByName('text_box');
			sendJson = '{ "ProviderName" : "$$PROVIDER$$", "DriverLibFileName" : "$$$DRVFILE$$", "DriverName" : "$$NAME$$" }'
                        for (var i = 0; i < textboxes.length; i++) { // @todo make parallel executions
                                switch (textboxes[i].id) {
                                        case "1":
                                                sendJson = sendJson.replace("$$PROVIDER$$", textboxes[i].value);
                                                break;
                                        case "2":
                                                sendJson = sendJson.replace("$$$DRVFILE$$", textboxes[i].value);
                                                break;
                                        case "3":
                                                sendJson = sendJson.replace("$$NAME$$", textboxes[i].value);
                                                break;
                                        default:
                                                break;
                                }
                        }
                        var xhr = new XMLHttpRequest();
                        xhr.open("POST", "$$SPIDER_SERVER$$/spider/driver", true);
                        xhr.setRequestHeader('Content-Type', 'application/json');
                        //xhr.send(JSON.stringify({ "DriverName": driverName, "ProviderName": providerName, "DriverLibFileName": driverLibFileName}));
			//xhr.send(JSON.stringify(sendJson));
			xhr.send(sendJson);

                        setTimeout(function(){
                                location.reload();
                        }, 400);

                }
        `
        strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://" + cr.HostIPorName + cr.ServicePort) // cr.ServicePort = ":1024"
        return strFunc
}

// make the string of javascript function
func makeDeleteDriverFunc_js() string {
// curl -X DELETE http://$RESTSERVER:1024/spider/driver/gcp-driver01 -H 'Content-Type: application/json'

        strFunc := `
                function deleteDriver() {
                        var checkboxes = document.getElementsByName('check_box');
                        for (var i = 0; i < checkboxes.length; i++) { // @todo make parallel executions
                                if (checkboxes[i].checked) {
                                        var xhr = new XMLHttpRequest();
                                        xhr.open("DELETE", "$$SPIDER_SERVER$$/spider/driver/" + checkboxes[i].value, false);
                                        xhr.setRequestHeader('Content-Type', 'application/json');
                                        xhr.send(null);
                                }
                        }
			location.reload();
                }
        `
        strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://" + cr.HostIPorName + cr.ServicePort) // cr.ServicePort = ":1024"
        return strFunc
}

//================ Driver Info Management
// create driver page
func Driver(c echo.Context) error {
	cblog.Info("call Driver()")

	// make page header
	htmlStr :=  ` 
		<html>
		<head>
		    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
		    <script type="text/javascript">
		`
	// (1) make Javascript Function
		htmlStr += makeOnchangeDriverProviderFunc_js()
		htmlStr += makeCheckBoxToggleFunc_js()
		htmlStr += makePostDriverFunc_js()
		htmlStr += makeDeleteDriverFunc_js()


	htmlStr += `
		    </script>
		</head>

		<body>
		    <table border="0" bordercolordark="#F8F8FF" cellpadding="0" cellspacing="1" bgcolor="#FFFFFF"  style="font-size:small;">      
		`

	// (2) make Table Action TR
		// colspan, f5_href, delete_href, fontSize
		htmlStr += makeActionTR_html("5", "driver", "deleteDriver()", "2")


	// (3) make Table Header TR
		
		nameWidthList := []NameWidth {
		    {"Provider Name", "200"},
		    {"Driver Library Name", "300"},
		    {"Driver Name", "200"},
		}	
		htmlStr +=  makeTitleTRList_html("#DDDDDD", "2", nameWidthList)


	// (4) make TR list with info list
        // (4-1) get info list @todo if empty list
		resBody, err := getResourceList_JsonByte("driver")
		if err != nil {
			cblog.Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		var info struct {
			ResultList []*dim.CloudDriverInfo `json:"driver"`
		}
		json.Unmarshal(resBody, &info)

        // (4-2) make TR list with info list
		htmlStr += makeDriverTRList_html("", "", "", info.ResultList)


        // (5) make input field and add
        // attach text box for add
		htmlStr += `
			<tr bgcolor="#FFFFFF" align="center" height="30">
			    <td>
				    <font size=2>#</font>
			    </td>
			    <td>
				<!-- <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="1" value="AWS"> -->
		`
		// Select format of CloudOS  name=text_box, id=1
		htmlStr += makeSelect_html("onchangeProvider")

		htmlStr += `
			    </td>
			    <td>
				<input style="font-size:12px;text-align:center;" type="text" name="text_box" id="2" value="aws-driver-v1.0.so">
			    </td>
			    <td>
				<input style="font-size:12px;text-align:center;" type="text" name="text_box" id="3" value="aws-driver01">
			    </td>
			    <td>
				<a href="javascript:postDriver()">
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

// make the string of javascript function
func makeOnchangeCredentialProviderFunc_js() string {
        strFunc := `
              function onchangeProvider(source) {
                var providerName = source.value
		// for credential info
		switch(providerName) {
		  case "AWS":
			credentialInfo = '[{"Key":"ClientId", "Value":"XXXXXX"}, {"Key":"ClientSecret", "Value":"XXXXXX"}]'
		    break;
		  case "AZURE":
			credentialInfo = '[{"Key":"ClientId", "Value":"XXXX-XXXX"}, {"Key":"ClientSecret", "Value":"xxxx-xxxx"}, {"Key":"TenantId", "Value":"xxxx-xxxx"}, {"Key":"SubscriptionId", "Value":"xxxx-xxxx"}]'
		    break;
		  case "GCP":
			credentialInfo = '[{"Key":"PrivateKey", "Value":"-----BEGIN PRIVATE KEY-----\nXXXX\n-----END PRIVATE KEY-----\n"},{"Key":"ProjectID", "Value":"powerkimhub"}, {"Key":"ClientEmail", "Value":"xxxx@xxxx.iam.gserviceaccount.com"}]'
		    break;
		  case "ALIBABA":
			credentialInfo = '[{"Key":"ClientId", "Value":"XXXXXX"}, {"Key":"ClientSecret", "Value":"XXXXXX"}]'
		    break;
		  case "CLOUDIT":
			credentialInfo = '[{"Key":"IdentityEndpoint", "Value":"http://xxx.xxx.co.kr:9090"}, {"Key":"AuthToken", "Value":"xxxx"}, {"Key":"Username", "Value":"xxxx"}, {"Key":"Password", "Value":"xxxx"}, {"Key":"TenantId", "Value":"tnt0009"}]'
		    break;
		  case "OPENSTACK":
			credentialInfo = '[{"Key":"IdentityEndpoint", "Value":"http://182.252.xxx.xxx:5000/v3"}, {"Key":"Username", "Value":"etri"}, {"Key":"Password", "Value":"xxxx"}, {"Key":"DomainName", "Value":"default"}, {"Key":"ProjectID", "Value":"xxxx"}]'
		    break;
		  case "DOCKER":
			credentialInfo = '[{"Key":"Host", "Value":"http://18.191.xxx.xxx:1004"}, {"Key":"APIVersion", "Value":"v1.38"}]'
		    break;
		  case "CLOUDTWIN":
			credentialInfo = '[{"Key":"ClientId", "Value":"XXXXXX"}, {"Key":"ClientSecret", "Value":"XXXXXX"}]'
		    break;
		  default:
			credentialInfo = '[{"Key":"ClientId", "Value":"XXXXXX"}, {"Key":"ClientSecret", "Value":"XXXXXX"}]'
		}
                document.getElementById('2').value= credentialInfo

		// for credential name
                document.getElementById('3').value= providerName.toLowerCase() + "-credential-01";
              }
        `
        return strFunc
}

// number, Provider Name, Credential Info, Credential Name, checkbox
func makeCredentialTRList_html(bgcolor string, height string, fontSize string, infoList []*cim.CredentialInfo) string {
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
                            <font size=%s>$$S1$$</font>
                    </td>
                    <td>
                            <font size=%s>$$S2$$</font>
                    </td>
                    <td>
                            <font size=%s>$$S3$$</font>
                    </td>
                    <td>
                        <input type="checkbox" name="check_box" value=$$S3$$>
                    </td>
                </tr>
                `, bgcolor, height, fontSize, fontSize, fontSize, fontSize)

        strData := ""
        // set data and make TR list
        for i, one := range infoList{
                str := strings.ReplaceAll(strTR, "$$NUM$$", strconv.Itoa(i+1))
                str = strings.ReplaceAll(str, "$$S1$$", one.ProviderName)
		strKeyList := ""
                for _, kv := range one.KeyValueInfoList {
                        strKeyList += kv.Key + ":xxxx, "
                }
                str = strings.ReplaceAll(str, "$$S2$$", strKeyList)
                str = strings.ReplaceAll(str, "$$S3$$", one.CredentialName)
                strData += str
        }

        return strData
}

// make the string of javascript function
func makePostCredentialFunc_js() string {

// curl -X POST http://$RESTSERVER:1024/spider/credential -H 'Content-Type: application/json' '{"CredentialName":"aws-credential01","ProviderName":"AWS", "KeyValueInfoList": [{"Key":"ClientId", "Value":"XXXXXX"}, {"Key":"ClientSecret", "Value":"XXXXXX"}]}'

        strFunc := `
                function postCredential() {
                        var textboxes = document.getElementsByName('text_box');
			sendJson = '{ "ProviderName" : "$$PROVIDER$$", "KeyValueInfoList" : $$CREDENTIALINFO$$, "CredentialName" : "$$NAME$$" }'

                        for (var i = 0; i < textboxes.length; i++) { // @todo make parallel executions
                                switch (textboxes[i].id) {
                                        case "1":
                                                sendJson = sendJson.replace("$$PROVIDER$$", textboxes[i].value);
                                                break;
                                        case "2":
                                                sendJson = sendJson.replace("$$CREDENTIALINFO$$", textboxes[i].value);
                                                break;
                                        case "3":
                                                sendJson = sendJson.replace("$$NAME$$", textboxes[i].value);
                                                break;
                                        default:
                                                break;
                                }
                        }
                        var xhr = new XMLHttpRequest();
                        xhr.open("POST", "$$SPIDER_SERVER$$/spider/credential", true);
                        xhr.setRequestHeader('Content-Type', 'application/json');
                        //xhr.send(JSON.stringify({ "CredentialName": credentialName, "ProviderName": providerName, "KeyValueInfoList": credentialInfo}));
                        //xhr.send(JSON.stringify(sendJson));
                        xhr.send(sendJson);

                        setTimeout(function(){
                                location.reload();
                        }, 400);

                }
        `
        strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://" + cr.HostIPorName + cr.ServicePort) // cr.ServicePort = ":1024"
        return strFunc
}

// make the string of javascript function
func makeDeleteCredentialFunc_js() string {
// curl -X DELETE http://$RESTSERVER:1024/spider/credential/aws-credential01 -H 'Content-Type: application/json'

        strFunc := `
                function deleteCredential() {
                        var checkboxes = document.getElementsByName('check_box');
                        for (var i = 0; i < checkboxes.length; i++) { // @todo make parallel executions
                                if (checkboxes[i].checked) {
                                        var xhr = new XMLHttpRequest();
                                        xhr.open("DELETE", "$$SPIDER_SERVER$$/spider/credential/" + checkboxes[i].value, false);
                                        xhr.setRequestHeader('Content-Type', 'application/json');
                                        xhr.send(null);
                                }
                        }
			location.reload();
                }
        `
        strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://" + cr.HostIPorName + cr.ServicePort) // cr.ServicePort = ":1024"
        return strFunc
}

//================ Credential Info Management
// create credential page
func Credential(c echo.Context) error {
        cblog.Info("call Credential()")

        // make page header
        htmlStr :=  `
                <html>
                <head>
                    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
                    <script type="text/javascript">
                `
        // (1) make Javascript Function
		htmlStr += makeOnchangeCredentialProviderFunc_js()
                htmlStr += makeCheckBoxToggleFunc_js()
                htmlStr += makePostCredentialFunc_js()
                htmlStr += makeDeleteCredentialFunc_js()


        htmlStr += `
                    </script>
                </head>

                <body>
                    <table border="0" bordercolordark="#F8F8FF" cellpadding="0" cellspacing="1" bgcolor="#FFFFFF"  style="font-size:small;">
                `

        // (2) make Table Action TR
                // colspan, f5_href, delete_href, fontSize
                htmlStr += makeActionTR_html("5", "credential", "deleteCredential()", "2")


        // (3) make Table Header TR
                nameWidthList := []NameWidth {
                    {"Provider Name", "200"},
                    {"Credential Info", "300"},
                    {"Credential Name", "200"},
                }
                htmlStr +=  makeTitleTRList_html("#DDDDDD", "2", nameWidthList)


        // (4) make TR list with info list
        // (4-1) get info list @todo if empty list
                resBody, err := getResourceList_JsonByte("credential")
                if err != nil {
                        cblog.Error(err)
                        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
                }
                var info struct {
                        ResultList []*cim.CredentialInfo `json:"credential"`
                }
                json.Unmarshal(resBody, &info)

        // (4-2) make TR list with info list
                htmlStr += makeCredentialTRList_html("", "", "", info.ResultList)


        // (5) make input field and add
        // attach text box for add
                htmlStr += `
                        <tr bgcolor="#FFFFFF" align="center" height="30">
                            <td>
                                    <font size=2>#</font>
                            </td>
                            <td>
				<!-- <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="1" value="AWS"> -->
		`
                // Select format of CloudOS  name=text_box, id=1
                htmlStr += makeSelect_html("onchangeProvider")
			
		htmlStr += `	
                            </td>
                            <td>
                                <textarea style="font-size:12px;text-align:center;" name="text_box" id="2" cols=50>[{"Key":"ClientId", "Value":"XXXXXX"}, {"Key":"ClientSecret", "Value":"XXXXXX"}]</textarea>
                            </td>
                            <td>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="3" value="aws-credential01">
                            </td>
                            <td>
                                <a href="javascript:postCredential()">
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

// make the string of javascript function
func makeOnchangeRegionProviderFunc_js() string {
        strFunc := `
              function onchangeProvider(source) {
                var providerName = source.value
        // for region info
        switch(providerName) {
          case "AWS":
            regionInfo = '[{"Key":"Region", "Value":"us-east-2"}]'
            region = '(ohio)us-east-2'
            zone = ''
            break;
          case "AZURE":
            regionInfo = '[{"Key":"location", "Value":"northeurope"}, {"Key":"ResourceGroup", "Value":"CB-GROUP-POWERKIM"}]'
            region = 'northeurope'
            zone = ''            
            break;
          case "GCP":
            regionInfo = '[{"Key":"Region", "Value":"us-central1"},{"Key":"Zone", "Value":"us-central1-a"}]'
            region = 'us-central1'
            zone = 'us-central1-a'             
            break;
          case "ALIBABA":
            regionInfo = '[{"Key":"Region", "Value":"ap-northeast-1"}, {"Key":"Zone", "Value":"ap-northeast-1a"}]'
            region = 'ap-northeast-1'
            zone = 'ap-northeast-1a'             
            break;
          case "CLOUDIT":
            regionInfo = '[{"Key":"Region", "Value":"default"}]'
            region = 'default'
            zone = ''            
            break;
          case "OPENSTACK":
            regionInfo = '[{"Key":"Region", "Value":"RegionOne"}]'
            region = 'RegionOne'
            zone = 'RegionOne'            
            break;
          case "DOCKER":
            regionInfo = '[{"Key":"Region", "Value":"default"}]'
            region = 'default'
            zone = ''             
            break;
          case "CLOUDTWIN":
            regionInfo = '[{"Key":"Region", "Value":"default"}]'
            region = 'default'
            zone = '' 
            break;
          default:
            regionInfo = '[{"Key":"Region", "Value":"us-east-2"}]'
            region = '(ohio)us-east-2'
            zone = ''
        }
                document.getElementById('2').value= regionInfo

        // for region-zone name
                document.getElementById('3').value= providerName.toLowerCase() + "-" + region + "-" + zone;
              }
        `
        return strFunc
}

// number, Provider Name, Region Info, Region Name, checkbox
func makeRegionTRList_html(bgcolor string, height string, fontSize string, infoList []*rim.RegionInfo) string {
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
                            <font size=%s>$$S1$$</font>
                    </td>
                    <td>
                            <font size=%s>$$S2$$</font>
                    </td>
                    <td>
                            <font size=%s>$$S3$$</font>
                    </td>
                    <td>
                        <input type="checkbox" name="check_box" value=$$S3$$>
                    </td>
                </tr>
                `, bgcolor, height, fontSize, fontSize, fontSize, fontSize)

        strData := ""
        // set data and make TR list
        for i, one := range infoList{
                str := strings.ReplaceAll(strTR, "$$NUM$$", strconv.Itoa(i+1))
                str = strings.ReplaceAll(str, "$$S1$$", one.ProviderName)
        strKeyList := ""
                for _, kv := range one.KeyValueInfoList {
                        strKeyList += kv.Key + ":" + kv.Value + ", "
                }
                str = strings.ReplaceAll(str, "$$S2$$", strKeyList)
                str = strings.ReplaceAll(str, "$$S3$$", one.RegionName)
                strData += str
        }

        return strData
}

// make the string of javascript function
func makePostRegionFunc_js() string {

// curl -X POST http://$RESTSERVER:1024/spider/region -H 'Content-Type: application/json' -d '{"RegionName":"aws-(ohio)us-east-2","ProviderName":"AWS", "KeyValueInfoList": [{"Key":"Region", "Value":"us-east-2"}]}'

        strFunc := `
                function postRegion() {
                        var textboxes = document.getElementsByName('text_box');
            sendJson = '{ "ProviderName" : "$$PROVIDER$$", "KeyValueInfoList" : $$REGIONINFO$$, "RegionName" : "$$NAME$$" }'

                        for (var i = 0; i < textboxes.length; i++) { // @todo make parallel executions
                                switch (textboxes[i].id) {
                                        case "1":
                                                sendJson = sendJson.replace("$$PROVIDER$$", textboxes[i].value);
                                                break;
                                        case "2":
                                                sendJson = sendJson.replace("$$REGIONINFO$$", textboxes[i].value);
                                                break;
                                        case "3":
                                                sendJson = sendJson.replace("$$NAME$$", textboxes[i].value);
                                                break;
                                        default:
                                                break;
                                }
                        }
                        var xhr = new XMLHttpRequest();
                        xhr.open("POST", "$$SPIDER_SERVER$$/spider/region", true);
                        xhr.setRequestHeader('Content-Type', 'application/json');
                        //xhr.send(JSON.stringify({ "RegionName": regionName, "ProviderName": providerName, "KeyValueInfoList": regionInfo}));
                        //xhr.send(JSON.stringify(sendJson));
                        xhr.send(sendJson);

                        setTimeout(function(){
                                location.reload();
                        }, 400);

                }
        `
        strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://" + cr.HostIPorName + cr.ServicePort) // cr.ServicePort = ":1024"
        return strFunc
}

// make the string of javascript function
func makeDeleteRegionFunc_js() string {
// curl -X DELETE http://$RESTSERVER:1024/spider/region/aws-(ohio)us-east-2 -H 'Content-Type: application/json'

        strFunc := `
                function deleteRegion() {
                        var checkboxes = document.getElementsByName('check_box');
                        for (var i = 0; i < checkboxes.length; i++) { // @todo make parallel executions
                                if (checkboxes[i].checked) {
                                        var xhr = new XMLHttpRequest();
                                        xhr.open("DELETE", "$$SPIDER_SERVER$$/spider/region/" + checkboxes[i].value, false);
                                        xhr.setRequestHeader('Content-Type', 'application/json');
                                        xhr.send(null);
                                }
                        }
			location.reload();
                }
        `
        strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://" + cr.HostIPorName + cr.ServicePort) // cr.ServicePort = ":1024"
        return strFunc
}

//================ Region Info Management
// create region page
func Region(c echo.Context) error {
        cblog.Info("call Region()")

        // make page header
        htmlStr :=  `
                <html>
                <head>
                    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
                    <script type="text/javascript">
                `
        // (1) make Javascript Function
        htmlStr += makeOnchangeRegionProviderFunc_js()
                htmlStr += makeCheckBoxToggleFunc_js()
                htmlStr += makePostRegionFunc_js()
                htmlStr += makeDeleteRegionFunc_js()


        htmlStr += `
                    </script>
                </head>

                <body>
                    <table border="0" bordercolordark="#F8F8FF" cellpadding="0" cellspacing="1" bgcolor="#FFFFFF"  style="font-size:small;">
                `

        // (2) make Table Action TR
                // colspan, f5_href, delete_href, fontSize
                htmlStr += makeActionTR_html("5", "region", "deleteRegion()", "2")


        // (3) make Table Header TR
                nameWidthList := []NameWidth {
                    {"Provider Name", "200"},
                    {"Region Info", "300"},
                    {"Region Name", "200"},
                }
                htmlStr +=  makeTitleTRList_html("#DDDDDD", "2", nameWidthList)


        // (4) make TR list with info list
        // (4-1) get info list @todo if empty list
                resBody, err := getResourceList_JsonByte("region")
                if err != nil {
                        cblog.Error(err)
                        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
                }
                var info struct {
                        ResultList []*rim.RegionInfo `json:"region"`
                }
                json.Unmarshal(resBody, &info)

        // (4-2) make TR list with info list
                htmlStr += makeRegionTRList_html("", "", "", info.ResultList)


        // (5) make input field and add
        // attach text box for add
                htmlStr += `
                        <tr bgcolor="#FFFFFF" align="center" height="30">
                            <td>
                                    <font size=2>#</font>
                            </td>
                            <td>
                <!-- <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="1" value="AWS"> -->
        `
                // Select format of CloudOS  name=text_box, id=1
                htmlStr += makeSelect_html("onchangeProvider")
            
        htmlStr += `    
                            </td>
                            <td>
                                <textarea style="font-size:12px;text-align:center;" name="text_box" id="2" cols=50>[{"Key":"Region", "Value":"us-east-2"}]</textarea>
                            </td>
                            <td>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="3" value="aws-(ohio)us-east-2">
                            </td>
                            <td>
                                <a href="javascript:postRegion()">
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

// make the string of javascript function
func makeOnInitialInputBoxSetup_js() string {
        strFunc := `
              function onInitialSetup() {
		 cspSelect = document.getElementById('1')
		 onchangeProvider(cspSelect) 
	      }
	`
        return strFunc
}

// make the string of javascript function
func makeOnchangeConnectionConfigProviderFunc_js() string {
        strFunc := `
              function onchangeProvider(source) {
                var providerName = source.value
        // for credential info
	var driverNameList = []
	var credentialNameList
	var regionNameList
        switch(providerName) {
          case "AWS":
	    driverNameList = document.getElementsByName('driverName-AWS');
	    credentialNameList = document.getElementsByName('credentialName-AWS');
	    regionNameList = document.getElementsByName('regionName-AWS');
            break;
          case "AZURE":
	    driverNameList = document.getElementsByName('driverName-AZURE');
	    credentialNameList = document.getElementsByName('credentialName-AZURE');
	    regionNameList = document.getElementsByName('regionName-AZURE');
            break;
          case "GCP":
	    driverNameList = document.getElementsByName('driverName-GCP');
	    credentialNameList = document.getElementsByName('credentialName-GCP');
	    regionNameList = document.getElementsByName('regionName-GCP');
            break;
          case "ALIBABA":
	    driverNameList = document.getElementsByName('driverName-ALIBABA');
	    credentialNameList = document.getElementsByName('credentialName-ALIBABA');
	    regionNameList = document.getElementsByName('regionName-ALIBABA');
            break;
          case "CLOUDIT":
	    driverNameList = document.getElementsByName('driverName-CLOUDIT');
	    credentialNameList = document.getElementsByName('credentialName-CLOUDIT');
	    regionNameList = document.getElementsByName('regionName-CLOUDIT');
            break;
          case "OPENSTACK":
	    driverNameList = document.getElementsByName('driverName-OPENSTACK');
	    credentialNameList = document.getElementsByName('credentialName-OPENSTACK');
	    regionNameList = document.getElementsByName('regionName-OPENSTACK');
            break;
          case "DOCKER":
	    driverNameList = document.getElementsByName('driverName-DOCKER');
	    credentialNameList = document.getElementsByName('credentialName-DOCKER');
	    regionNameList = document.getElementsByName('regionName-DOCKER');
            break;
          case "CLOUDTWIN":
	    driverNameList = document.getElementsByName('driverName-CLOUDTWIN');
	    credentialNameList = document.getElementsByName('credentialName-CLOUDTWIN');
	    regionNameList = document.getElementsByName('regionName-CLOUDTWIN');
            break;
          default:
	    driverNameList = document.getElementsByName('driverName-AWS');
	    credentialNameList = document.getElementsByName('credentialName-AWS');
	    regionNameList = document.getElementsByName('regionName-AWS');
        }

	// Select Tag for drivers
	//  options remove & create
	var len = document.getElementById('2').options.length
	for (var i=0; i < len; i++) {
		document.getElementById('2').remove(0);
	}
	for (var i=0; i < driverNameList.length; i++) {
		document.getElementById('2').options.add(new Option(driverNameList[i].innerHTML, driverNameList[i].innerHTML));
	}

        // Select Tag for Credentials
        //  options remove & create
        var len = document.getElementById('3').options.length
        for (var i=0; i < len; i++) {
                document.getElementById('3').remove(0);
        }
        for (var i=0; i < credentialNameList.length; i++) {
                document.getElementById('3').options.add(new Option(credentialNameList[i].innerHTML, credentialNameList[i].innerHTML));
        }

        // Select Tag for Regions
        //  options remove & create
        var len = document.getElementById('4').options.length
        for (var i=0; i < len; i++) {
                document.getElementById('4').remove(0);
        }
        for (var i=0; i < regionNameList.length; i++) {
                document.getElementById('4').options.add(new Option(regionNameList[i].innerHTML, regionNameList[i].innerHTML));
        }

	document.getElementById('5').value= providerName.toLowerCase() + "-" +  document.getElementById('4').value + "-connection-config-01";

              }
        `
        return strFunc
}

func getRegionZone(regionName string) (string, string, error) {
	// Region Name List
	resBody, err := getResource_JsonByte("region", regionName)
	if err != nil {
		cblog.Error(err)
		return "", "", err 
	}
	var regionInfo rim.RegionInfo
	json.Unmarshal(resBody, &regionInfo)

	region := ""
	zone := ""
	// get the region & zone
	for _, one := range regionInfo.KeyValueInfoList {
		if one.Key == "Region" {
			region = one.Value
		}
		if one.Key == "location" {
			region = one.Value
		}
		if one.Key == "Zone" {
			zone = one.Value
		}
		
	}
	return region, zone, nil
}

// make the string of javascript function
func makeSetupConnectionConfigFunc_js() string {

        strFunc := `
                function setupConnectionConfig(configName, providerName, region, zone) {
                        var connConfigLabel = parent.frames["top_frame"].document.getElementById("connConfig");
			connConfigLabel.innerHTML = configName

                        var cspText = parent.frames["top_frame"].document.getElementById("connDisplay");
			if (zone) {
				cspText.value = providerName + ": " + region + " / " + zone
			} else {
				cspText.value = providerName + ": " + region
			}

                        var a = parent.frames["top_frame"].document.getElementById("vpcHref");
			a.href = "vpc/" + configName

                        var a = parent.frames["top_frame"].document.getElementById("securitygroupHref");
			a.href = "securitygroup/" + configName
                }
        `
        return strFunc
}

// number, Provider Name, Driver Name, Credential Name, Region Name, Connection Name, checkbox
func makeConnectionConfigTRList_html(bgcolor string, height string, fontSize string, infoList []*ccim.ConnectionConfigInfo) (string, error) {
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
                            <font size=%s>$$PROVIDERNAME$$</font>
                    </td>
                    <td>
                            <font size=%s>$$S2$$</font>
                    </td>
                    <td>
                            <font size=%s>$$S3$$</font>
                    </td>
                    <td>
                            <font size=%s>$$S4$$</font>
                    </td>
		    <td>                                       <!-- configName, CSP, Region, Zone -->
			<a href="javascript:setupConnectionConfig('$$CONFIGNAME$$', '$$PROVIDERNAME$$', '$$REGION$$', '$$ZONE$$')">
                            <font size=%s>$$CONFIGNAME$$</font>
			</a>
                    </td>
                    <td>
                        <input type="checkbox" name="check_box" value=$$CONFIGNAME$$>
                    </td>
                </tr>
                `, bgcolor, height, fontSize, fontSize, fontSize, fontSize, fontSize, fontSize)

        strData := ""
        // set data and make TR list
        for i, one := range infoList{
                str := strings.ReplaceAll(strTR, "$$NUM$$", strconv.Itoa(i+1))
                str = strings.ReplaceAll(str, "$$PROVIDERNAME$$", one.ProviderName)
                str = strings.ReplaceAll(str, "$$S2$$", one.DriverName)
                str = strings.ReplaceAll(str, "$$S3$$", one.CredentialName)
                str = strings.ReplaceAll(str, "$$S4$$", one.RegionName)
                str = strings.ReplaceAll(str, "$$CONFIGNAME$$", one.ConfigName)

		region, zone, err := getRegionZone(one.RegionName)
		if err != nil {
			cblog.Error(err)
			return "", err
		}
                str = strings.ReplaceAll(str, "$$REGION$$", region)
                str = strings.ReplaceAll(str, "$$ZONE$$", zone)
	
                strData += str
        }

        return strData, nil
}

// make the string of javascript function
func makePostConnectionConfigFunc_js() string {

// curl -X POST http://$RESTSERVER:1024/spider/connectionconfig -H 'Content-Type: application/json' 
//    -d '{"ProviderName":"AWS", "DriverName":"aws-driver01", "CredentialName":"aws-credential01", "RegionName":"aws-ohio", "ConfigName":"aws-ohio-config",}'

        strFunc := `
                function postConnectionConfig() {
                        var textboxes = document.getElementsByName('text_box');
            sendJson = '{ "ProviderName" : "$$PROVIDER$$", "DriverName" : "$$DRIVERNAME$$", "CredentialName" : "$$CREDENTIALNAME$$", \
                                                "RegionName" : "$$REGIONNAME$$", "ConfigName" : "$$NAME$$" }'

                        for (var i = 0; i < textboxes.length; i++) { // @todo make parallel executions
                                switch (textboxes[i].id) {
                                        case "1":
                                                sendJson = sendJson.replace("$$PROVIDER$$", textboxes[i].value);
                                                break;
                                        case "2":
                                                sendJson = sendJson.replace("$$DRIVERNAME$$", textboxes[i].value);
                                                break;
                                        case "3":
                                                sendJson = sendJson.replace("$$CREDENTIALNAME$$", textboxes[i].value);
                                                break;
                                        case "4":
                                                sendJson = sendJson.replace("$$REGIONNAME$$", textboxes[i].value);
                                                break;                                                
                                        case "5":
                                                sendJson = sendJson.replace("$$NAME$$", textboxes[i].value);
                                                break;
                                        default:
                                                break;
                                }
                        }
                        var xhr = new XMLHttpRequest();
                        xhr.open("POST", "$$SPIDER_SERVER$$/spider/connectionconfig", true);
                        xhr.setRequestHeader('Content-Type', 'application/json');
                        xhr.send(sendJson);

                        setTimeout(function(){
                                location.reload();
                        }, 400);

                }
        `
        strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://" + cr.HostIPorName + cr.ServicePort) // cr.ServicePort = ":1024"
        return strFunc
}

// make the string of javascript function
func makeDeleteConnectionConfigFunc_js() string {
// curl -X DELETE http://$RESTSERVER:1024/spider/connectionconfig/aws-connection01 -H 'Content-Type: application/json'

        strFunc := `
                function deleteConnectionConfig() {
                        var checkboxes = document.getElementsByName('check_box');
                        for (var i = 0; i < checkboxes.length; i++) { // @todo make parallel executions
                                if (checkboxes[i].checked) {
                                        var xhr = new XMLHttpRequest();
                                        xhr.open("DELETE", "$$SPIDER_SERVER$$/spider/connectionconfig/" + checkboxes[i].value, false);
                                        xhr.setRequestHeader('Content-Type', 'application/json');
                                        xhr.send(null);
                                }
                        }
			location.reload();
                }
        `
        strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://" + cr.HostIPorName + cr.ServicePort) // cr.ServicePort = ":1024"
        return strFunc
}

func makeDriverNameHiddenTRList_html(infoList []*dim.CloudDriverInfo) string {

        // make base Label frame for info list
        strTR := `<label name="driverName-$$CSP$$" hidden>$$DRIVERNAME$$</label>`

        strData := ""
        // set data and make TR list
        for _, one := range infoList{
                str := strings.ReplaceAll(strTR, "$$CSP$$", one.ProviderName)
                str = strings.ReplaceAll(str, "$$DRIVERNAME$$", one.DriverName)
                strData += str
        }

        return strData
}

func makeCredentialNameHiddenTRList_html(infoList []*cim.CredentialInfo) string {

        // make base Label frame for info list
        strTR := `<label name="credentialName-$$CSP$$" hidden>$$CREDENTIALNAME$$</label>`

        strData := ""
        // set data and make TR list
        for _, one := range infoList{
                str := strings.ReplaceAll(strTR, "$$CSP$$", one.ProviderName)
                str = strings.ReplaceAll(str, "$$CREDENTIALNAME$$", one.CredentialName)
                strData += str
        }

        return strData
}

func makeRegionNameHiddenTRList_html(infoList []*rim.RegionInfo) string {

        // make base Label frame for info list
        strTR := `<label name="regionName-$$CSP$$" hidden>$$REGIONNAME$$</label>`

        strData := ""
        // set data and make TR list
        for _, one := range infoList{
                str := strings.ReplaceAll(strTR, "$$CSP$$", one.ProviderName)
                str = strings.ReplaceAll(str, "$$REGIONNAME$$", one.RegionName)
                strData += str
        }

        return strData
}

//================ Connection Config Info Management
// create Connection page
func Connectionconfig(c echo.Context) error {
        cblog.Info("call Connectionconfig()")

        // make page header
        htmlStr :=  `
                <html>
                <head>
                    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
                    <script type="text/javascript">
                `
        // (1) make Javascript Function
        htmlStr += makeOnchangeConnectionConfigProviderFunc_js()
                htmlStr += makeSetupConnectionConfigFunc_js()
                htmlStr += makeOnInitialInputBoxSetup_js()
                htmlStr += makeCheckBoxToggleFunc_js()
                htmlStr += makePostConnectionConfigFunc_js()
                htmlStr += makeDeleteConnectionConfigFunc_js()


        htmlStr += `
                    </script>
                </head>

                <body onload=onInitialSetup()>
                    <table border="0" bordercolordark="#F8F8FF" cellpadding="0" cellspacing="1" bgcolor="#FFFFFF"  style="font-size:small;">
                `

        // (2) make Table Action TR
                // colspan, f5_href, delete_href, fontSize
                htmlStr += makeActionTR_html("7", "connectionconfig", "deleteConnectionConfig()", "2")


        // (3) make Table Header TR
                nameWidthList := []NameWidth {
                    {"Provider Name", "200"},
                    {"Driver Name", "200"},
                    {"Credential Name", "200"},
                    {"Region Name", "200"},
                    {"Connection Config Name", "200"},
                }
                htmlStr +=  makeTitleTRList_html("#DDDDDD", "2", nameWidthList)

        // (4) make TR list with info list
        // (4-1) get info list @todo if empty list
                resBody, err := getResourceList_JsonByte("connectionconfig")
                if err != nil {
                        cblog.Error(err)
                        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
                }
                var info struct {
                        ResultList []*ccim.ConnectionConfigInfo `json:"connectionconfig"`
                }
                json.Unmarshal(resBody, &info)

        // (4-2) make TR list with info list
                trStrList, err :=  makeConnectionConfigTRList_html("", "", "", info.ResultList)
                if err != nil {
                        cblog.Error(err)
                        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
                }
                htmlStr += trStrList

        // (4-3) make hidden TR list with info list
		// (a) Driver Name Hidden List
		resBody, err = getResourceList_JsonByte("driver")
		if err != nil {
			cblog.Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		var driverInfo struct {
			ResultList []*dim.CloudDriverInfo `json:"driver"`
		}
                json.Unmarshal(resBody, &driverInfo)
                htmlStr += makeDriverNameHiddenTRList_html(driverInfo.ResultList)

		// (b) Credential Name Hidden List
                resBody, err = getResourceList_JsonByte("credential")
                if err != nil {
                        cblog.Error(err)
                        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
                }
                var credentialInfo struct {
                        ResultList []*cim.CredentialInfo `json:"credential"`
                }
                json.Unmarshal(resBody, &credentialInfo)
                htmlStr += makeCredentialNameHiddenTRList_html(credentialInfo.ResultList)

		// (c) Region Name Hidden List
                resBody, err = getResourceList_JsonByte("region")
                if err != nil {
                        cblog.Error(err)
                        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
                }
                var regionInfo struct {
                        ResultList []*rim.RegionInfo `json:"region"`
                }
                json.Unmarshal(resBody, &regionInfo)
                htmlStr += makeRegionNameHiddenTRList_html(regionInfo.ResultList)


        // (5) make input field and add
        // attach text box for add
                htmlStr += `
                        <tr bgcolor="#FFFFFF" align="center" height="30">
                            <td>
                                    <font size=2>#</font>
                            </td>
                            <td>
                <!-- <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="1" value="AWS"> -->
        `
                // Select format of CloudOS  name=text_box, id=1
                htmlStr += makeSelect_html("onchangeProvider")
            
        htmlStr += `    
                            </td>
			    <!-- value is set up by '<body onload()=onInitialSetup()>' -->
                            <td>
                                <select style="font-size:12px;text-align:center;" name="text_box" id="2" value="aws-driver-v1.0">
                            </td>
                            <td>
                                <select style="font-size:12px;text-align:center;" name="text_box" id="3" value="aws-credential01">
                            </td>
                            <td>
                                <select style="font-size:12px;text-align:center;" name="text_box" id="4" value="aws-region01">
                            </td>
                            <td>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="5" value="aws-connection-config01">
                            </td>

                            <td>
                                <a href="javascript:postConnectionConfig()">
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

//================ This Spider Info
func SpiderInfo(c echo.Context) error {
        cblog.Info("call SpiderInfo()")


        htmlStr :=  `
                <html>
                <head>
                    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
                </head>

                <body>

                    <table border="0" bordercolordark="#F8F8FF" cellpadding="0" cellspacing="1" bgcolor="#FFFFFF"  style="font-size:small;">
                                <tr bgcolor="#DDDDDD" align="center">
                                    <td width="200">
                                            <font size=2>Server Start Time</font>
                                    </td>
                                    <td width="220">
                                            <font size=2>Server Version</font>
                                    </td>
                                    <td width="220">
                                            <font size=2>API Version</font>
                                    </td>
                                </tr>
                                <tr bgcolor="#FFFFFF" align="center" height="30">
                                    <td width="220">
                                            <font size=2>$$STARTTIME$$</font>
                                    </td>
                                    <td width="220">
                                            <font size=2>CB-Spider v0.2.0 (Cappuccino)</font>
                                    </td>
                                    <td width="220">
                                            <font size=2>REST API v0.2.0 (Cappuccino)</font>
                                    </td>
                                </tr>

                    </table>
		    <hr>
		<br>
		<br>
		<br>
                    <table border="0" bordercolordark="#F8F8FF" cellpadding="0" cellspacing="1" bgcolor="#FFFFFF"  style="font-size:small;">
                                <tr bgcolor="#DDDDDD" align="center">
                                    <td width="240">
                                            <font size=2>API EndPoint</font>
                                    </td>
                                    <td width="420">
                                            <font size=2>API Docs</font>
                                    </td>
                                </tr>
                                <tr bgcolor="#FFFFFF" align="left" height="30">
                                    <td width="240">
                                            <font size=2>$$APIENDPOINT$$</font>
                                    </td>
                                    <td width="420">
                                            <font size=2>
                                            * Cloud Connection Info Mgmt:
                                                <br>
                                                &nbsp;&nbsp;&nbsp;&nbsp;<a href='https://cloud-barista.github.io/rest-api/v0.2.0/spider/ccim/' target='_blank'>
                                                    https://cloud-barista.github.io/rest-api/v0.2.0/spider/ccim/
                                                </a>
                                            </font>
                                            <p>
                                            <font size=2>
                                                * Cloud Resource Control Mgmt: 
                                                <br>
                                                &nbsp;&nbsp;&nbsp;&nbsp;<a href='https://cloud-barista.github.io/rest-api/v0.2.0/spider/cctm/' target='_blank'>
                                                    https://cloud-barista.github.io/rest-api/v0.2.0/spider/cctm/
                                                </a>
                                            </font>
                                    </td>
                                </tr>

                    </table>
		    <hr>
                </body>
                </html>
                `

        htmlStr = strings.ReplaceAll(htmlStr, "$$STARTTIME$$", cr.StartTime)
        htmlStr = strings.ReplaceAll(htmlStr, "$$APIENDPOINT$$", "http://" + cr.HostIPorName + cr.ServicePort + "/spider") // cr.ServicePort = ":1024"

        return c.HTML(http.StatusOK, htmlStr)
}

