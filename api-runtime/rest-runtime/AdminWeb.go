// Cloud Info Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2019.09.

package main

import (
	"fmt"
	dim "github.com/cloud-barista/cb-spider/cloud-info-manager/driver-info-manager"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager/credential-info-manager"
	"strconv"

	//cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
/*

	ccim "github.com/cloud-barista/cb-spider/cloud-info-manager/connection-config-info-manager"
	im "github.com/cloud-barista/cb-spider/cloud-info-manager"
	rim "github.com/cloud-barista/cb-spider/cloud-info-manager/region-info-manager"
*/
	"net/http"
	"io/ioutil"
	"strings"
	"github.com/labstack/echo"
	"encoding/json"
)

//================ Frame
func frame(c echo.Context) error {
	cblog.Info("call frame()")

        htmlStr :=  `
<html>
  <head>
    <title>CB-Spider Admin Web Tool ....__^..^__....</title>
    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
  </head>
    <frameset rows="85,*" frameborder="Yes" border=1">
        <frame src="adminweb/top" name="top_frame" scrolling="auto" noresize marginwidth="0" marginheight="0"/>
        <frameset frameborder="Yes" border=1">
            <frame src="adminweb/driver" name="main_frame" scrolling="auto" noresize marginwidth="5" marginheight="0"/> 
<!--            <frame src="bottom_history.jsp" name="bottom" scrolling="auto" noresize marginwidth="2" marginheight="0"> -->            
        </frameset>
    </frameset>
    <noframes>
    <body>
    
    
    </body>
    </noframes>
</html>
        `

	return c.HTML(http.StatusOK, htmlStr)
}

//================ Top
func top(c echo.Context) error {
	cblog.Info("call top()")

	htmlStr :=  ` 
<html>
<head>
    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
</head>
<body>

    <!-- <table border="0" bordercolordark="#FFFFFF" cellpadding="0" cellspacing="2" bgcolor="#FFFFFF" width="320" style="font-size:small;"> -->
    <table border="0" bordercolordark="#FFFFFF" cellpadding="0" cellspacing="1" bgcolor="#FFFFFF"  style="font-size:small;">      
        <tr bgcolor="#FFFFFF" align="center">
            <td rowspan="2" width="80" bgcolor="#FFFFFF">
                <!-- CB-Spider Logo -->
                <a href="../adminweb" target="_top">
                  <img height="45" width="42" src="https://cloud-barista.github.io/assets/img/frameworks/cb-spider.png" border='0' hspace='0' vspace='1' align="middle">
                </a>
		<font size=1>$$TIME$$</font>	
            </td>

            <td width="100">       
                <!-- Drivers Management --> 
                <a href="driver" target="main_frame">            
                    <font size=2>driver</font>
                </a>
            </td>
            <td width="100">       
                <!-- Credential Management -->
                <a href="credential" target="main_frame">            
                    <font size=2>credential</font>
                </a>
            </td>
            <td width="100">       
                <!-- Regions Management -->
                <a href="region" target="main_frame">            
                    <font size=2>region</font>
                </a>
            </td>
            <td width="100">       
                <!-- Connection Management -->
                <a href="connection" target="main_frame">            
                    <font size=2>connection</font>
                </a>
            </td>
            <td width="100">       
                <!-- This CB-Spider Info -->
                <a href="spiderinfo" target="main_frame">            
                    <font size=2>this spider</font>
                </a>
            </td>
            <td width="100">       
                <!-- CB-Spider Github -->
                <a href="https://github.com/cloud-barista/cb-spider" target="_blank">            
                    <font size=2>github</font>
                </a>
            </td> 
	</tr>

        <tr bgcolor="#FFFFFF" align="center">
            <td width="100">
                <!-- Image Management -->
                <a href="image" target="main_frame">
                    <font size=2>image(tbd)</font>
                </a>
            </td>
            <td width="100">
                <!-- Spec Management -->
                <a href="spec" target="_blank">
                    <font size=2>spec</font>
                </a>
            </td>
            <td width="100">
                <!-- VPC/Subnet Management -->
                <a href="vpc" target="_blank">
                    <font size=2>vpc/subnet</font>
                </a>
            </td>
            <td width="100">
                <!-- SecurityGroup Management -->
                <a href="security" target="_blank">
                    <font size=2>security group</font>
                </a>
            </td>
            <td width="100">
                <!-- KeyPair Management -->
                <a href="keypair" target="_blank">
                    <font size=2>keypair</font>
                </a>
            </td>
            <td width="100">
                <!-- VM Management -->
                <a href="vm">
                    <font size=2>vm</font>
                </a>
            </td>
        </tr>

    </table>
</body>
</html>
	`

	
	htmlStr = strings.ReplaceAll(htmlStr, "$$TIME$$", StartTime)
	return c.HTML(http.StatusOK, htmlStr)
}

//================ Driver Info Management
// (1) make info list
// (2) make input field and add
// (3) make page & table frame
func driver(c echo.Context) error {
	cblog.Info("call driver()")

	res, err := http.Get("http://localhost:1024/spider/driver")
        if err != nil {
		cblog.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }
        resBody, err := ioutil.ReadAll(res.Body)
        res.Body.Close()
        if err != nil {
		cblog.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        var info struct {
                Result []*dim.CloudDriverInfo `json:"driver"`
        }
	json.Unmarshal(resBody, &info)


	// (1) make info list
	strTR :=  ` 
		<tr bgcolor="#FFFFFF" align="center" height="30">
		    <td width="15">
			    <font size=2>$$NUM$$</font>
		    </td>
		    <td width="200">
			    <font size=2>$$S1$$</font>
		    </td>
		    <td width="200">
			    <font size=2>$$S2$$</font>
		    </td>
		    <td width="250">
			    <font size=2>$$S3$$</font>
		    </td>
		    <td width="15">
			<input type="checkbox" name="check_box" value=$$S1$$>
		    </td>
		</tr>
	`

	strData := ""
	for i, one := range info.Result {
		str := strings.ReplaceAll(strTR, "$$NUM$$", strconv.Itoa(i+1))
		str = strings.ReplaceAll(str, "$$S1$$", one.DriverName)
		str = strings.ReplaceAll(str, "$$S2$$", one.ProviderName)
		str = strings.ReplaceAll(str, "$$S3$$", one.DriverLibFileName)
		strData += str
	}

	// (2) make input field and add
	// attach text box for add
	strTextBox := `
                <tr bgcolor="#FFFFFF" align="center" height="30">
		    <td width="15">
			    <font size=2>#</font>
		    </td>
                    <td width="200">
                        <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="1" value="aws-driver01">
                    </td>
                    <td width="200">
                        <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="2" value="AWS">
                    </td>
                    <td width="250">
                        <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="3" value="aws-driver-v1.0.so">
                    </td>
                    <td width="15">
			<a href="javascript:postDriver()">
			    <font size=3><b>+</b></font>
			</a>
                    </td>
                </tr>
	`
	strData += strTextBox


	// (3) make page & table frame
	htmlStr :=  ` 
		<html>
		<head>
		    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
		    <script type="text/javascript">
		      function toggle(source) {
		        var checkboxes = document.getElementsByName('check_box');
		        for (var i = 0; i < checkboxes.length; i++) {
		          checkboxes[i].checked = source.checked;
			}
		      }
                        // curl -X POST http://$RESTSERVER:1024/spider/driver -H 'Content-Type: application/json'  -d '{"DriverName":"aws-driver01","ProviderName":"AWS", "DriverLibFileName":"aws-driver-v1.0.so"}'
                        function postDriver() {
                                var textboxes = document.getElementsByName('text_box');
                                for (var i = 0; i < textboxes.length; i++) { // @todo make parallel executions
					switch (textboxes[i].id) {
						case "1": 
							driverName = textboxes[i].value;
							break;
						case "2": 
							providerName = textboxes[i].value;
							break;
						case "3": 
							driverLibFileName = textboxes[i].value;
							break;
						default:
							break;
					}
                                }
				var xhr = new XMLHttpRequest();
				xhr.open("POST", "$$SPIDER_SERVER$$/spider/driver", true);
				xhr.setRequestHeader('Content-Type', 'application/json');
				xhr.send(JSON.stringify({ "DriverName": driverName, "ProviderName": providerName, "DriverLibFileName": driverLibFileName}));

                                setTimeout(function(){
                                        location.reload();
                                }, 500);

                        }

			// curl -X DELETE http://$RESTSERVER:1024/spider/driver/gcp-driver01 -H 'Content-Type: application/json'
			function deleteDriver() {
				var checkboxes = document.getElementsByName('check_box');
				for (var i = 0; i < checkboxes.length; i++) { // @todo make parallel executions
					if (checkboxes[i].checked) {
						var xhr = new XMLHttpRequest();
						xhr.open("DELETE", "$$SPIDER_SERVER$$/spider/driver/" + checkboxes[i].value, true);
						xhr.setRequestHeader('Content-Type', 'application/json');
						xhr.send(null);
					}
				}
				setTimeout(function(){
					location.reload();
				}, 500);

			}
		    </script>
		</head>


		<body>
		    <table border="0" bordercolordark="#F8F8FF" cellpadding="0" cellspacing="1" bgcolor="#FFFFFF"  style="font-size:small;">      
				<tr bgcolor="#FFFFFF" align="right">
				    <td colspan="5">
					<a href="driver">
					    <font size=2><b>&nbsp;F5</b></font>
					</a>
					&nbsp;
					<a href="javascript:deleteDriver();">
					    <font size=2><b>&nbsp;X</b></font>
					</a>
					&nbsp;
				    </td>
				</tr>
				<tr bgcolor="#DDDDDD" align="center">
				    <td width="15">
					    <font size=2><b>&nbsp;#</b></font>
				    </td>
				    <td width="200">
					    <font size=2>Driver Name</font>
				    </td>
				    <td width="200">
					    <font size=2>Provider Name</font>
				    </td>
				    <td width="250">
					    <font size=2>Driver Library Name</font>
				    </td>
				    <td width="15">
					    <input type="checkbox" onclick="toggle(this);" /> 
				    </td>
				</tr>
				$$DATA$$
		    </table>
		</body>
		</html>
		`

	htmlStr = strings.ReplaceAll(htmlStr, "$$SPIDER_SERVER$$", "http://" + HostIPorName + ServicePort) // ServicePort = ":1024"
	htmlStr = strings.ReplaceAll(htmlStr, "$$DATA$$", strData)
fmt.Println(htmlStr)
	return c.HTML(http.StatusOK, htmlStr)
}

//================ Crential Info Management
// (1) make info list
// (2) make input field and add
// (3) make page & table frame
func credential(c echo.Context) error {
        cblog.Info("call credential()")

        res, err := http.Get("http://localhost:1024/spider/credential")
        if err != nil {
                cblog.Error(err)
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }
        resBody, err := ioutil.ReadAll(res.Body)
        res.Body.Close()
        if err != nil {
                cblog.Error(err)
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        var info struct {
                Result []*cim.CredentialInfo `json:"credential"`
        }
        json.Unmarshal(resBody, &info)


        // (1) make info list
        strTR :=  `
                <tr bgcolor="#FFFFFF" align="center" height="30">
                    <td width="15">
                            <font size=2>$$NUM$$</font>
                    </td>
                    <td width="200">
                            <font size=2>$$S1$$</font>
                    </td>
                    <td width="200">
                            <font size=2>$$S2$$</font>
                    </td>
                    <td width="250">
                            <font size=2>$$S3$$</font>
                    </td>
                    <td width="15">
                        <input type="checkbox" name="check_box" value=$$S1$$>
                    </td>
                </tr>
        `

        strData := ""
        for i, one := range info.Result {
                str := strings.ReplaceAll(strTR, "$$NUM$$", strconv.Itoa(i+1))
                str = strings.ReplaceAll(str, "$$S1$$", one.CredentialName)
                str = strings.ReplaceAll(str, "$$S2$$", one.ProviderName)
		strKeyList := ""
		for _, kv := range one.KeyValueInfoList {
			strKeyList += kv.Key + ":xxxx, "	
		}
                str = strings.ReplaceAll(str, "$$S3$$", strKeyList)
                strData += str
        }

        // (2) make input field and add
        // attach text box for add
        strTextBox := `
                <tr bgcolor="#FFFFFF" align="center" height="30">
                    <td width="15">
                            <font size=2>#</font>
                    </td>
                    <td width="200">
                        <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="1" value="aws-credential01">
                    </td>
                    <td width="200">
                        <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="2" value="AWS">
                    </td>
                    <td width="250">
                        <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="3" value='[{"Key":"ClientId", "Value":"XXXXXX"}, {"Key":"ClientSecret", "Value":"XXXXXX"}]'>
                    </td>
                    <td width="15">
                       <!-- <a href="javascript:postCredential()"> -->
                            <font size=3><b>+(tbd)</b></font>
                       <!-- </a> -->
                    </td>
                </tr>
        `
        strData += strTextBox


        // (3) make page & table frame
        htmlStr :=  `
                <html>
                <head>
                    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
                    <script type="text/javascript">
                      function toggle(source) {
                        var checkboxes = document.getElementsByName('check_box');
                        for (var i = 0; i < checkboxes.length; i++) {
                          checkboxes[i].checked = source.checked;
                        }
                      }
                        // curl -X POST http://$RESTSERVER:1024/spider/credential -H 'Content-Type: application/json' -d '{"CredentialName":"aws-credential01","ProviderName":"AWS", "KeyValueInfoList": [{"Key":"ClientId", "Value":"XXXXXX"}, {"Key":"ClientSecret", "Value":"XXXXXX"}]}'

                        function postCredential() {
                                var textboxes = document.getElementsByName('text_box');
                                for (var i = 0; i < textboxes.length; i++) { // @todo make parallel executions
                                        switch (textboxes[i].id) {
                                                case "1":
                                                        crdentialName = textboxes[i].value;
                                                        break;
                                                case "2":
                                                        providerName = textboxes[i].value;
                                                        break;
                                                case "3":
                                                        keyValueInfoList = textboxes[i].value;
                                                        break;
                                                default:
                                                        break;
                                        }
                                }
                                var xhr = new XMLHttpRequest();
                                xhr.open("POST", "$$SPIDER_SERVER$$/spider/driver", true);
                                xhr.setRequestHeader('Content-Type', 'application/json');
                                xhr.send(JSON.stringify({ "DriverName": driverName, "ProviderName": providerName, "DriverLibFileName": driverLibFileName}));

                                setTimeout(function(){
                                        location.reload();
                                }, 500);

                        }

                        // curl -X DELETE http://$RESTSERVER:1024/spider/credential/aws-credential01 -H 'Content-Type: application/json'
                        function deleteCredential() {
                                var checkboxes = document.getElementsByName('check_box');
                                for (var i = 0; i < checkboxes.length; i++) { // @todo make parallel executions
                                        if (checkboxes[i].checked) {
                                                var xhr = new XMLHttpRequest();
                                                xhr.open("DELETE", "$$SPIDER_SERVER$$/spider/credential/" + checkboxes[i].value, true);
                                                xhr.setRequestHeader('Content-Type', 'application/json');
                                                xhr.send(null);
                                        }
                                }
                                setTimeout(function(){
                                        location.reload();
                                }, 500);

                        }
                    </script>
                </head>


                <body>
                    <table border="0" bordercolordark="#F8F8FF" cellpadding="0" cellspacing="1" bgcolor="#FFFFFF"  style="font-size:small;">
                                <tr bgcolor="#FFFFFF" align="right">
                                    <td colspan="4">
                                        <a href="credential">
                                            <font size=2><b>&nbsp;F5</b></font>
                                        </a>
                                        &nbsp;
                                        <a href="javascript:deleteCredential();">
                                            <font size=2><b>&nbsp;X</b></font>
                                        </a>
                                        &nbsp;
                                    </td>
                                </tr>
                                <tr bgcolor="#DDDDDD" align="center">
                                    <td width="15">
                                            <font size=2><b>&nbsp;#</b></font>
                                    </td>
                                    <td width="200">
                                            <font size=2>Credential Name</font>
                                    </td>
                                    <td width="200">
                                            <font size=2>Provider Name</font>
                                    </td>
                                    <td width="250">
                                            <font size=2>Credential Info</font>
                                    </td>
                                    <td width="15">
                                            <input type="checkbox" onclick="toggle(this);" />
                                    </td>
                                </tr>
                                $$DATA$$
                    </table>
                </body>
                </html>
                `

        htmlStr = strings.ReplaceAll(htmlStr, "$$SPIDER_SERVER$$", "http://" + HostIPorName + ServicePort) // ServicePort = ":1024"
        htmlStr = strings.ReplaceAll(htmlStr, "$$DATA$$", strData)
        return c.HTML(http.StatusOK, htmlStr)
}


//================ This Spider Info
func spiderInfo(c echo.Context) error {
        cblog.Info("call spiderInfo()")


        htmlStr :=  `
                <html>
                <head>
                    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
                </head>

                <body>

                    <table border="0" bordercolordark="#F8F8FF" cellpadding="0" cellspacing="1" bgcolor="#FFFFFF"  style="font-size:small;">
                                <tr bgcolor="#DDDDDD" align="center">
                                    <td width="200">
                                            <font size=2>Start Time</font>
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
                                            <font size=2>CB-Spider Version(TBD)</font>
                                    </td>
                                    <td width="220">
                                            <font size=2>API Version</font>
                                    </td>
                                </tr>

                    </table>
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
                                <tr bgcolor="#FFFFFF" align="center" height="30">
                                    <td width="240">
                                            <font size=2>$$APIENDPOINT$$</font>
                                    </td>
                                    <td width="420">
                                            <font size=2><a href='https://cloud-barista.github.io/rest-api/v0.2.0/spider/ccim/' target='_blank'>https://cloud-barista.github.io/rest-api/v0.2.0/spider/ccim/ </a></font>
                                            <font size=2><a href='https://cloud-barista.github.io/rest-api/v0.2.0/spider/cctm/' target='_blank'>https://cloud-barista.github.io/rest-api/v0.2.0/spider/cctm/ </a></font>
                                    </td>
                                </tr>

                    </table>
                </body>
                </html>
                `

        htmlStr = strings.ReplaceAll(htmlStr, "$$STARTTIME$$", StartTime)
        htmlStr = strings.ReplaceAll(htmlStr, "$$APIENDPOINT$$", "http://" + HostIPorName + ServicePort + "/spider") // ServicePort = ":1024"

        return c.HTML(http.StatusOK, htmlStr)
}

