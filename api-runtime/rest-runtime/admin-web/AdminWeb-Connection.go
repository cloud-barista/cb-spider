// Cloud Info Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// Updated: 2024.07.05.
// by CB-Spider Team, 2020.06.

package adminweb

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strconv"

	cr "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	ccim "github.com/cloud-barista/cb-spider/cloud-info-manager/connection-config-info-manager"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager/credential-info-manager"
	dim "github.com/cloud-barista/cb-spider/cloud-info-manager/driver-info-manager"
	rim "github.com/cloud-barista/cb-spider/cloud-info-manager/region-info-manager"

	"encoding/json"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

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
          case "TENCENT":
	    driverNameList = document.getElementsByName('driverName-TENCENT');
	    credentialNameList = document.getElementsByName('credentialName-TENCENT');
	    regionNameList = document.getElementsByName('regionName-TENCENT');
            break;
          case "IBM":
	    driverNameList = document.getElementsByName('driverName-IBM');
	    credentialNameList = document.getElementsByName('credentialName-IBM');
	    regionNameList = document.getElementsByName('regionName-IBM');
            break;
          case "OPENSTACK":
	    driverNameList = document.getElementsByName('driverName-OPENSTACK');
	    credentialNameList = document.getElementsByName('credentialName-OPENSTACK');
	    regionNameList = document.getElementsByName('regionName-OPENSTACK');
            break;

          case "NCP":
	    driverNameList = document.getElementsByName('driverName-NCP');
	    credentialNameList = document.getElementsByName('credentialName-NCP');
	    regionNameList = document.getElementsByName('regionName-NCP');
            break;
          case "NCP":
	    driverNameList = document.getElementsByName('driverName-NCP');
	    credentialNameList = document.getElementsByName('credentialName-NCP');
	    regionNameList = document.getElementsByName('regionName-NCP');
            break;
          case "NHN":
	    driverNameList = document.getElementsByName('driverName-NHN');
	    credentialNameList = document.getElementsByName('credentialName-NHN');
	    regionNameList = document.getElementsByName('regionName-NHN');
            break;
		case "KTCLOUD":
			driverNameList = document.getElementsByName('driverName-KTCLOUD');
			credentialNameList = document.getElementsByName('credentialName-KTCLOUD');
			regionNameList = document.getElementsByName('regionName-KTCLOUD');
				break;
		case "KTCLOUDVPC":
			driverNameList = document.getElementsByName('driverName-KTCLOUDVPC');
			credentialNameList = document.getElementsByName('credentialName-KTCLOUDVPC');
			regionNameList = document.getElementsByName('regionName-KTCLOUDVPC');
				break;

          case "MOCK":
	    driverNameList = document.getElementsByName('driverName-MOCK');
	    credentialNameList = document.getElementsByName('credentialName-MOCK');
	    regionNameList = document.getElementsByName('regionName-MOCK');
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

	//document.getElementById('5').value= providerName.toLowerCase() + "-" +  document.getElementById('4').value + "-connection-config-01";
	document.getElementById('5').value= providerName.toLowerCase() + "-config-01";

              }
        `
	return strFunc
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

			// for vpc
			var a = parent.frames["top_frame"].document.getElementById("vpcHref");
			a.href = "vpc/" + configName
			a = parent.frames["top_frame"].document.getElementById("vpcmgmtHref");
			a.href = "vpcmgmt/" + configName

			// for securitygroup
			a = parent.frames["top_frame"].document.getElementById("securitygroupHref");
			a.href = "securitygroup/" + configName
			a = parent.frames["top_frame"].document.getElementById("securitygroupmgmtHref");
			a.href = "securitygroupmgmt/" + configName

			// for KeyPair
			a = parent.frames["top_frame"].document.getElementById("keypairHref");
			a.href = "keypair/" + configName
			a = parent.frames["top_frame"].document.getElementById("keypairmgmtHref");
			a.href = "keypairmgmt/" + configName

			// for vm
			a = parent.frames["top_frame"].document.getElementById("vmHref");
			a.href = "vm/" + configName
			a = parent.frames["top_frame"].document.getElementById("vmmgmtHref");
			a.href = "vmmgmt/" + configName

			// for nlb 
			a = parent.frames["top_frame"].document.getElementById("nlbHref");
			a.href = "nlb/" + configName
			a = parent.frames["top_frame"].document.getElementById("nlbmgmtHref");
			a.href = "nlbmgmt/" + configName

			// for disk 
			a = parent.frames["top_frame"].document.getElementById("diskHref");
			a.href = "disk/" + configName
			a = parent.frames["top_frame"].document.getElementById("diskmgmtHref");
			a.href = "diskmgmt/" + configName

			// for myimage 
			a = parent.frames["top_frame"].document.getElementById("myimageHref");
			a.href = "myimage/" + configName
			a = parent.frames["top_frame"].document.getElementById("myimagemgmtHref");
			a.href = "myimagemgmt/" + configName

			// for VMImage
			a = parent.frames["top_frame"].document.getElementById("vmimageHref");
			a.href = "vmimage/" + configName

			// for VMSpec
			a = parent.frames["top_frame"].document.getElementById("vmspecHref");
			a.href = "vmspec/" + configName

			// for RegionZone
			a = parent.frames["top_frame"].document.getElementById("regionzoneHref");
			a.href = "regionzone/" + configName

			// for Price
			a = parent.frames["top_frame"].document.getElementById("priceinfoHref");
			a.href = "priceinfo/" + configName

			// for Cluster(PMKS)
			a = parent.frames["top_frame"].document.getElementById("clusterHref");
			a.href = "cluster/" + configName
			a = parent.frames["top_frame"].document.getElementById("clustermgmtHref");
			a.href = "clustermgmt/" + configName
		}
        `
	return strFunc
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

// number, Provider Name, Driver Name, Credential Name, Region Name, Connection Name, checkbox
func makeConnectionConfigTRList_html(bgcolor string, height string, fontSize string, infoList []*ccim.ConnectionConfigInfo) (string, error) {
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
	for i, one := range infoList {
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
	//    -d '{"ProviderName":"AWS", "DriverName":"aws-driver01", "CredentialName":"aws-credential-01", "RegionName":"aws-ohio", "ConfigName":"aws-ohio-config",}'

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
                        xhr.open("POST", "$$SPIDER_SERVER$$/spider/connectionconfig", false);
                        xhr.setRequestHeader('Content-Type', 'application/json');

			// client logging
			parent.frames["log_frame"].Log("curl -sX POST " + "$$SPIDER_SERVER$$/spider/connectionconfig -H 'Content-Type: application/json' -d '" + sendJson + "'");

                        xhr.send(sendJson);

			// client logging
			parent.frames["log_frame"].Log("   => " + xhr.response);

                        // setTimeout(function(){ // when async call
                                location.reload();
                        // }, 400);

                }
        `
	strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://"+cr.ServiceIPorName+cr.ServicePort) // cr.ServicePort = ":1024"
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

                                        // client logging
                                        parent.frames["log_frame"].Log("curl -sX DELETE " + "$$SPIDER_SERVER$$/spider/connectionconfig/" + checkboxes[i].value + " -H 'Content-Type: application/json'" );

                                        xhr.send(null);

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

// create Connection page
func Connectionconfig(c echo.Context) error {
	cblog.Info("call Connectionconfig()")

	// make page header
	htmlStr := `
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
	nameWidthList := []NameWidth{
		{"Provider Name", "200"},
		{"Driver Name", "200"},
		{"Credential Name", "200"},
		{"Region Name", "200"},
		{"Connection Config Name", "200"},
	}
	htmlStr += makeTitleTRList_html("#DDDDDD", "2", nameWidthList, true)

	// (4) make TR list with info list
	// (4-1) get info list @todo if empty list

	// client logging
	htmlStr += genLoggingGETResURL("connectionconfig")

	resBody, err := getResourceList_JsonByte("connectionconfig")
	if err != nil {
		cblog.Error(err)
		// client logging
		htmlStr += genLoggingGETResURL(err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	// client logging
	htmlStr += genLoggingResult(string(resBody[:len(resBody)-1]))

	var info struct {
		ResultList []*ccim.ConnectionConfigInfo `json:"connectionconfig"`
	}
	json.Unmarshal(resBody, &info)

	// (4-2) make TR list with info list
	trStrList, err := makeConnectionConfigTRList_html("", "", "", info.ResultList)
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
	nameList := cloudosList()
	htmlStr += `
                        <tr bgcolor="#FFFFFF" align="center" height="30">
                            <td bgcolor="#FFEFBA">
                                    <font size=2>&nbsp;create:&nbsp;</font>
                            </td>
                            <td>
                <!-- <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="1" value="AWS"> -->
        `
	// Select format of CloudOS  name=text_box, id=1
	htmlStr += makeSelect_html("onchangeProvider", nameList, "1")

	htmlStr += `    
                            </td>
			    <!-- value is set up by '<body onload()=onInitialSetup()>' -->
                            <td>
                                <select style="font-size:12px;text-align:center;" name="text_box" id="2" value="aws-driver-v1.0">
                            </td>
                            <td>
                                <select style="font-size:12px;text-align:center;" name="text_box" id="3" value="aws-credential-01">
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

// ================ Connection Config Info Management
type ConnectionConfig struct {
	ConfigName     string `json:"ConfigName"`
	ProviderName   string `json:"ProviderName"`
	DriverName     string `json:"DriverName"`
	CredentialName string `json:"CredentialName"`
	RegionName     string `json:"RegionName"`
}

type ConnectionConfigs struct {
	ConnectionConfigs []ConnectionConfig `json:"connectionconfig"`
}

type Providers struct {
	Providers []string `json:"cloudos"`
}

type RegionInfo struct {
	RegionName       string `json:"RegionName"`
	ProviderName     string `json:"ProviderName"`
	KeyValueInfoList []struct {
		Key   string `json:"Key"`
		Value string `json:"Value"`
	} `json:"KeyValueInfoList"`
}

type Regions struct {
	Regions []RegionInfo `json:"region"`
}

type DriverInfo struct {
	DriverName        string `json:"DriverName"`
	DriverLibFileName string `json:"DriverLibFileName"`
	ProviderName      string `json:"ProviderName"`
}

type Drivers struct {
	Drivers []DriverInfo `json:"driver"`
}

func fetchConnectionConfigs() (map[string][]ConnectionConfig, error) {
	resp, err := http.Get("http://localhost:1024/spider/connectionconfig")
	if err != nil {
		return nil, fmt.Errorf("error fetching connection configurations: %v", err)
	}
	defer resp.Body.Close()

	var configs ConnectionConfigs
	if err := json.NewDecoder(resp.Body).Decode(&configs); err != nil {
		return nil, fmt.Errorf("error decoding connection configurations: %v", err)
	}

	connectionMap := make(map[string][]ConnectionConfig)
	for _, config := range configs.ConnectionConfigs {
		connectionMap[config.ProviderName] = append(connectionMap[config.ProviderName], config)
	}

	return connectionMap, nil
}

func fetchProviders() ([]string, error) {
	resp, err := http.Get("http://localhost:1024/spider/cloudos")
	if err != nil {
		return nil, fmt.Errorf("error fetching providers: %v", err)
	}
	defer resp.Body.Close()

	var providers Providers
	if err := json.NewDecoder(resp.Body).Decode(&providers); err != nil {
		return nil, fmt.Errorf("error decoding providers: %v", err)
	}

	return providers.Providers, nil
}

func fetchDrivers() (map[string]string, error) {
	resp, err := http.Get("http://localhost:1024/spider/driver")
	if err != nil {
		return nil, fmt.Errorf("error fetching drivers: %v", err)
	}
	defer resp.Body.Close()

	var drivers Drivers
	if err := json.NewDecoder(resp.Body).Decode(&drivers); err != nil {
		return nil, fmt.Errorf("error decoding drivers: %v", err)
	}

	driverMap := make(map[string]string)
	for _, driver := range drivers.Drivers {
		driverMap[driver.DriverName] = driver.DriverLibFileName
	}

	return driverMap, nil
}

func ConnectionManagement(c echo.Context) error {
	connectionConfigs, err := fetchConnectionConfigs()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	providers, err := fetchProviders()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	regions, err := fetchRegions()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	drivers, err := fetchDrivers()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	data := struct {
		ConnectionConfigs map[string][]ConnectionConfig
		Providers         []string
		Regions           map[string]string
		Drivers           map[string]string
	}{
		ConnectionConfigs: connectionConfigs,
		Providers:         providers,
		Regions:           regions,
		Drivers:           drivers,
	}

	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/connection.html")
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error loading template: " + err.Error()})
	}

	return tmpl.Execute(c.Response().Writer, data)
}
