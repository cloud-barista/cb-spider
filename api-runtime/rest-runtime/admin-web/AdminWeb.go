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
	"bytes"
        "github.com/cloud-barista/cb-store/config"
        "github.com/sirupsen/logrus"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	cr "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"

	"net/http"
	"io/ioutil"
	"strings"
	"github.com/labstack/echo/v4"
	"encoding/json"
	"regexp"
)

var cblog *logrus.Logger
func init() {
	cblog = config.Cblogger
}

type NameWidth struct {
	Name string
	Width string
}


func cloudosList() []string {
	resBody, err := getResourceList_JsonByte("cloudos")
	if err != nil {
		cblog.Error(err)
	}
	var info struct {
		ResultList []string `json:"cloudos"`
	}
	json.Unmarshal(resBody, &info)

	return info.ResultList
}

func vpcList(connConfig string) []string {
        resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "vpc")
        if err != nil {
                cblog.Error(err)
        }
        var info struct {
                ResultList []cres.VPCInfo `json:"vpc"`
        }
        json.Unmarshal(resBody, &info)

	var nameList []string
	for _, vpc := range info.ResultList {
		nameList = append(nameList, vpc.IId.NameId)
	}
        return nameList
}

func vmList(connConfig string) []string {
        resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "vm")
        if err != nil {
                cblog.Error(err)
        }
        var info struct {
                ResultList []cres.VMInfo `json:"vm"`
        }
        json.Unmarshal(resBody, &info)

        var nameList []string
        for _, vm := range info.ResultList {
                nameList = append(nameList, vm.IId.NameId)
        }
        return nameList
}

func vmStatus(connConfig string, vmName string) string {
        resBody, err := getResource_with_Connection_JsonByte(connConfig, "vmstatus", vmName)
        if err != nil {
                cblog.Error(err)
        }
	//var info cres.VMStatusInfo 
	var info struct {
                Status string
        }
        json.Unmarshal(resBody, &info)
        //return fmt.Sprint(info.Status)
        return info.Status
}

func subnetList(connConfig string, vpcName string) []string {
        resBody, err := getResource_with_Connection_JsonByte(connConfig, "vpc", vpcName)
        if err != nil {
                cblog.Error(err)
        }
        var info cres.VPCInfo
        json.Unmarshal(resBody, &info)

        var nameList []string
        for _, subnetInfo := range info.SubnetInfoList {
                nameList = append(nameList, subnetInfo.IId.NameId)
        }
        return nameList
}


func keyPairList(connConfig string) []string {
        resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "keypair")
        if err != nil {
                cblog.Error(err)
        }
        var info struct {
                ResultList []cres.VPCInfo `json:"keypair"`
        }
        json.Unmarshal(resBody, &info)

        var nameList []string
        for _, keypair := range info.ResultList {
                nameList = append(nameList, keypair.IId.NameId)
        }
        return nameList
}

func availableDataDiskList(connConfig string) []string {
        resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "disk")
        if err != nil {
                cblog.Error(err)
        }
        var info struct {
                ResultList []cres.DiskInfo `json:"disk"`
        }
        json.Unmarshal(resBody, &info)

        var nameList []string
        for _, disk := range info.ResultList {
		if disk.Status == cres.DiskAvailable {
			nameList = append(nameList, disk.IId.NameId)
		}
        }
        return nameList
}

func diskInfo(connConfig string, diskName string) cres.DiskInfo {
        resBody, err := getResource_with_Connection_JsonByte(connConfig, "disk", diskName)
        if err != nil {
                cblog.Error(err)
        }

        var info cres.DiskInfo
        json.Unmarshal(resBody, &info)
        return info
}

func diskTypeList(providerName string) []string {
        // get Provider's Meta Info
        cloudOSMetaInfo, err := cim.GetCloudOSMetaInfo(providerName)
        if err != nil {
                cblog.Error(err)
                return []string{}
        }
	return cloudOSMetaInfo.DiskType
}

func diskTypeSizeList(providerName string) []string {
        // get Provider's Meta Info
        cloudOSMetaInfo, err := cim.GetCloudOSMetaInfo(providerName)
        if err != nil {
                cblog.Error(err)
                return []string{}
        }
        return cloudOSMetaInfo.DiskSize
}

//================ Frame
func Frame(c echo.Context) error {
	cblog.Info("call Frame()")

        htmlStr :=  `
<html>
  <head>
    <title>CB-Spider Admin Web Tool ....__^..^__....</title>
    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
  </head>
 <!--   <frameset rows="66,*" frameborder="Yes" border=1"> -->
    <frameset rows="100,*" frameborder="Yes" border=1">
        <frame src="adminweb/top" name="top_frame" scrolling="auto" noresize marginwidth="0" marginheight="0"/>
        <frameset rows="*,130" frameborder="Yes" border=2">
            <frame src="adminweb/driver" id="main_frame" name="main_frame" scrolling="auto" /> 
            <frame src="adminweb/log" id="log_frame" name="log_frame" scrolling="auto" /> 
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
func Top(c echo.Context) error {
	cblog.Info("call Top()")

	htmlStr :=  ` 
<html>
<head>
    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
</head>
<body>
    <!-- <table border="0" bordercolordark="#FFFFFF" cellpadding="0" cellspacing="2" bgcolor="#FFFFFF" width="320" style="font-size:small;"> -->
    <table border="0" bordercolordark="#FFFFFF" cellpadding="0" cellspacing="1" bgcolor="#FFFFFF"  style="font-size:small;">      
        <tr bgcolor="#FFFFFF" align="left">
            <td rowspan="2" width="70" bgcolor="#FFFFFF" align="center">
                <!-- CB-Spider Logo -->
                <a href="../adminweb" target="_top">
                  <!-- <img height="45" width="42" src="https://cloud-barista.github.io/assets/img/frameworks/cb-spider.png" border='0' hspace='0' vspace='1' align="middle"> -->
                  <img height="45" width="45" src="./images/logo.png" border='0' hspace='0' vspace='1' align="middle">
                </a>
		<font size=1>$$TIME$$</font>	
            </td>

            <td width="150"> 
                <!-- Drivers Management --> 
                <a href="driver" target="main_frame">            
                    <font size=2>1.driver</font>
                </a>
            </td>
            <td width="190">       
                <!-- Credential Management -->
                <a href="credential" target="main_frame">            
                    <font size=2>1.credential</font>
                </a>
            </td>
            <td width="130">
                <!-- Regions Management -->
                <a href="region" target="main_frame">            
                    <font size=2>1.region</font>
                </a>
            </td>
            <td width="190">
                <!-- Connection Management -->
                <a href="connectionconfig" target="main_frame">            
                    <font size=2>2.CONNECTION</font>
                </a>
            </td>
            <td width="260">
                <!-- Display Connection Config -->
		<label id="connConfig" hidden></label>
		<input style="font-size:11px;font-weight:bold;text-align:center;background-color:#EDF7F9;" type="text" id="connDisplay" name="connDisplay" size = 30 disabled value="CloudOS: Region / Zone">

            </td>
            <td rowspan="2" width="60"> 
                <!-- This CB-Spider Info -->
                <a href="spiderinfo" target="main_frame">            
                    <font size=2>info</font>
                </a>
            </td>
	</tr>

        <tr bgcolor="#FFFFFF" align="left">
            <td width="150">
                <!-- VPC/Subnet Management -->
                <a href="vpc/region not set" target="main_frame" id="vpcHref">
                    <font size=2>1.vpc/subnet</font>
                </a>
		&nbsp;
                <a href="vpcmgmt/region not set" target="main_frame" id="vpcmgmtHref">
                    <font size=2>[mgmt]</font>
                </a>                
            </td>
            <td width="190">
                <!-- SecurityGroup Management -->
                <a href="securitygroup/region not set" target="main_frame" id="securitygroupHref">
                    <font size=2>1.1.security group</font>
                </a>
		&nbsp;
                <a href="securitygroupmgmt/region not set" target="main_frame" id="securitygroupmgmtHref">
                    <font size=2>[mgmt]</font>
                </a>
            </td>
            <td width="130">
                <!-- KeyPair Management -->
                <a href="keypair/region not set" target="main_frame" id="keypairHref">
                    <font size=2>1.keypair</font>
                </a>
		&nbsp;
                <a href="keypairmgmt/region not set" target="main_frame" id="keypairmgmtHref">
                    <font size=2>[mgmt]</font>
                </a>
            </td>
            <td width="190">
                <!-- VM Management -->
                <a href="vm/region not set" target="main_frame" id="vmHref">
                    <font size=2>2.VM</font>
                </a>
                &nbsp;
                <a href="vmmgmt/region not set" target="main_frame" id="vmmgmtHref">
                    <font size=2>[mgmt]</font>
                </a>

                &nbsp;
                &nbsp;

                <!-- Disk Management -->
                <a href="disk/region not set" target="main_frame" id="diskHref">
                    <font size=2>2.Disk</font>
                </a>
                &nbsp;
                <a href="diskmgmt/region not set" target="main_frame" id="diskmgmtHref">
                    <font size=2>[mgmt]</font>
                </a>
            </td>

            <td width="260">

                <!-- NLB Management -->
                <a href="nlb/region not set" target="main_frame" id="nlbHref">
                    <font size=2>3.NLB</font>
                </a>
                &nbsp;
                <a href="nlbmgmt/region not set" target="main_frame" id="nlbmgmtHref">
                    <font size=2>[mgmt]</font>
                </a>

                &nbsp;
                &nbsp;
                &nbsp;
                &nbsp;

                <!-- Image Management -->
                <a href="vmimage/region not set" target="main_frame" id="vmimageHref">
                    <font size=2>vmimage</font>
                </a>

                &nbsp;
                &nbsp;

                <!-- Spec Management -->
                <a href="vmspec/region not set" target="main_frame" id="vmspecHref">
                    <font size=2>vmspec</font>
                </a>
            </td>

        </tr>

    </table>
</body>
</html>
	`

	
	htmlStr = strings.ReplaceAll(htmlStr, "$$TIME$$", cr.ShortStartTime)
	return c.HTML(http.StatusOK, htmlStr)
}

//================ Log
func Log(c echo.Context) error {
	cblog.Info("call Log()")

	htmlStr :=  ` 
<html>
	<head>
		<style>
			.footer {
			   position: fixed;
			   left: 2%;
			   bmeottom:8 0;
			   width: 96%;
			   background-color:lightgray;
			   color: white;
			   text-align: center;
			}
			.clearbutton {
			   position: fixed;
			   left: 0%;
			}

		</style>
		<script>
			function init() {
				var logObject = document.getElementById('printLog');
				logObject.style.width = "100%"; // 800;
				var height = parent.document.getElementById("log_frame").scrollHeight;
				logObject.style.height = height-15;
			}
						
			function main() {
				Log("# Spider Client Log...");
			}

			function Log(s) {
				var logObject = document.getElementById('printLog');
				var curTime = "[" + new Date().toLocaleTimeString() + "] ";
				logObject.value += (curTime + s + '\n');

				if(logObject.selectionStart == logObject.selectionEnd) {
					logObject.scrollTop = logObject.scrollHeight;
				}
			}

			function clearLog() {
				var logObject = document.getElementById('printLog');
				var curTime = "[" + new Date().toLocaleTimeString() + "] ";
				var s = "# Spider Client Log..."
				logObject.value = (curTime + s + '\n');
			}

			function resizeLogArea() {
				init()
			}

		</script>
	</head>

	<body onresize="resizeLogArea()">
		<button class="clearbutton" onclick="clearLog()">X</button>

		<div class="footer">
			<textarea id='printLog' disabled="true" style="overflow:scroll;resize:none;" wrap="off"></textarea>
		</div>
		<script>
			init();
			main();
		</script>

	</body>
</html>
	`

	
	htmlStr = strings.ReplaceAll(htmlStr, "$$TIME$$", cr.ShortStartTime)
	return c.HTML(http.StatusOK, htmlStr)
}

func makeSelect_html(onchangeFunctionName string, strList []string, id string) string {

	strSelect := `<select name="text_box" id="` + id + `" onchange="` + onchangeFunctionName + `(this)">`
	for _, one := range strList {
		if one == "AWS" {
			strSelect += `<option value="` + one + `" selected>` + one + `</option>`
		} else {
			strSelect += `<option value="` + one + `">` + one + `</option>`
		}
	}

	strSelect += `
		</select>
	`


	return strSelect
}

func makeDataDiskSelect_html(onchangeFunctionName string, strList []string, id string) string {

	strResult := "* DataDisk"
	if len(strList) == 0 {
		noDiskStr := `<input style="font-size:12px;text-align:center;" type="text" name="text_box" id="` + 
				id +`" disabled value="N/A">`
		return strResult + noDiskStr
	}
        strSelect := `<select style="width:120px;" name="text_box" id="` + id + `" onchange="` + onchangeFunctionName + `(this)" multiple>`
        for _, one := range strList {
		strSelect += `<option value="` + one + `">` + one + `</option>`
        }

        strSelect += `
                </select>
		<br>
		(Unselect: ctrl + click)
        `


        return strResult + strSelect
}

func makeDataDiskTypeSelect_html(onchangeFunctionName string, strList []string, id string) string {

        strResult := ""
        if len(strList) == 0 {
                noDiskStr := `<input style="font-size:12px;text-align:center;" type="text" name="text_box" id="` +
                                id +`" value="default">`
                return strResult + noDiskStr
        }
        strSelect := `<select style="width:120px;" name="text_box" id="` + id + `" onchange="` + onchangeFunctionName + `(this)">`
                strSelect += `<option value="default">default</option>`
        for _, one := range strList {
                strSelect += `<option value="` + one + `">` + one + `</option>`
        }

        strSelect += `
                </select>
        `

        return strResult + strSelect
}

func makeDataDiskTypeSize_html(strList []string) string {

	strResult := ""
        for _, one := range strList {
		// one = "cloud|5|2000|GB"
		splits := strings.Split(one, "|")
		rangeStr := ""
		if len(splits) == 4 {
			rangeStr = fmt.Sprintf("&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;[%s]&nbsp;&nbsp; %s~%s %s<br>", 
					strings.TrimSpace(splits[0]), insertComma(strings.TrimSpace(splits[1])), 
				 	insertComma(strings.TrimSpace(splits[2])), strings.TrimSpace(splits[3]))
		} else {
			rangeStr = one // keep origin string
		}
		strResult += rangeStr
        }

        strInput := `<p style="font-size:12px;color:gray;text-align:left;">` + strResult + `</p>`

        return strInput
}

// ref) https://stackoverflow.com/a/39185719/17474800
func insertComma(str string) string {
    re := regexp.MustCompile("(\\d+)(\\d{3})")
    for n := ""; n != str; {
        n = str
        str = re.ReplaceAllString(str, "$1,$2")
    }
    return str
}

func makeKeyPairSelect_html(onchangeFunctionName string, strList []string, id string) string {

        strSelect := `<select name="text_box" id="` + id + `" onchange="` + onchangeFunctionName + `(this)">`
        for _, one := range strList {
		strSelect += `<option value="` + one + `">` + one + `</option>`
        }
	// add one more not to use Key but to use password
	strSelect += `<option value=""</option>`

        strSelect += `
                </select>
        `


        return strSelect
}


func getResourceList_JsonByte(resourceName string) ([]byte, error) {
        // cr.ServicePort = ":1024"
	url := "http://" + "localhost" + cr.ServerPort + "/spider/" + resourceName

        // get object list
        res, err := http.Get(url)
        if err != nil {
                return nil, err
        }
        resBody, err := ioutil.ReadAll(res.Body)
        res.Body.Close()
        if err != nil {
                return nil, err
        }
	return resBody, err
}

func getResourceList_with_Connection_JsonByte(connConfig string, resourceName string) ([]byte, error) {
        // cr.ServicePort = ":1024"
	url := "http://" + "localhost" + cr.ServerPort + "/spider/" + resourceName
        // get object list
	var reqBody struct {
		Value string `json:"ConnectionName"`
	}	
	reqBody.Value = connConfig

	jsonValue, _ := json.Marshal(reqBody)
        request, err := http.NewRequest("GET", url, bytes.NewBuffer(jsonValue))
        if err != nil {
                return nil, err
        }
        request.Header.Set("Content-Type", "application/json")

	client := http.Client{}
	resp, err := client.Do(request)
        if err != nil {
                return nil, err
        }
	
        resBody, err := ioutil.ReadAll(resp.Body)
        resp.Body.Close()
        if err != nil {
                return nil, err
        }
        return resBody, err
}

func getAllResourceList_with_Connection_JsonByte(connConfig string, resourceName string) ([]byte, error) {
        // cr.ServicePort = ":1024"
	url := "http://" + "localhost" + cr.ServerPort + "/spider/all" + resourceName
        // get object list
        var reqBody struct {
                Value string `json:"ConnectionName"`
        }
        reqBody.Value = connConfig

        jsonValue, _ := json.Marshal(reqBody)
        request, err := http.NewRequest("GET", url, bytes.NewBuffer(jsonValue))
        if err != nil {
                return nil, err
        }
        request.Header.Set("Content-Type", "application/json")

        client := http.Client{}
        resp, err := client.Do(request)
        if err != nil {
                return nil, err
        }

        resBody, err := ioutil.ReadAll(resp.Body)
        resp.Body.Close()
        if err != nil {
                return nil, err
        }
        return resBody, err
}

func getResource_JsonByte(resourceName string, name string) ([]byte, error) {
        // cr.ServicePort = ":1024"
	url := "http://" + "localhost" + cr.ServerPort + "/spider/" + resourceName + "/" + name

        // get object list
        res, err := http.Get(url)
        if err != nil {
                return nil, err
        }
        resBody, err := ioutil.ReadAll(res.Body)
        res.Body.Close()
        if err != nil {
                return nil, err
        }
	return resBody, err
}

func getResource_with_Connection_JsonByte(connConfig string, resourceName string, name string) ([]byte, error) {
        // cr.ServicePort = ":1024"
	url := "http://" + "localhost" + cr.ServerPort + "/spider/" + resourceName + "/" + name
        // get object list
        var reqBody struct {
                Value string `json:"ConnectionName"`
        }
        reqBody.Value = connConfig

        jsonValue, _ := json.Marshal(reqBody)
        request, err := http.NewRequest("GET", url, bytes.NewBuffer(jsonValue))
        if err != nil {
                return nil, err
        }
        request.Header.Set("Content-Type", "application/json")

        client := http.Client{}
        resp, err := client.Do(request)
        if err != nil {
                return nil, err
        }

        resBody, err := ioutil.ReadAll(resp.Body)
        resp.Body.Close()
        if err != nil {
                return nil, err
        }
        return resBody, err
}

// F5, X ("5", "driver", "deleteDriver()", "2")
func makeActionTR_html(colspan string, f5_href string,  delete_href string, fontSize string) string {
	if fontSize == "" { fontSize = "2" }

        strTR := fmt.Sprintf(`
		<tr bgcolor="#FFFFFF" align="right">
		    <td colspan="%s">
			<a href="%s">
			    <font size=%s><b>&nbsp;F5</b></font>
			</a>
			&nbsp;
			<a href="javascript:%s;">
			    <font color=red size=%s><b>&nbsp;X</b></font>
			</a>
			&nbsp;
		    </td>
		</tr>
       		`, colspan, f5_href, fontSize, delete_href, fontSize) 

	return strTR
}

//         fieldName-width
// number, fieldName0-200, fieldName1-400, ... , checkbox
func makeTitleTRList_html(bgcolor string, fontSize string, nameWidthList []NameWidth, hasCheckBox bool) string {
	if bgcolor == "" { bgcolor = "#DDDDDD" }
	if fontSize == "" { fontSize = "2" }

	// (1) header number field
        strTR := fmt.Sprintf(`
		<tr bgcolor="%s" align="center">
		    <td width="15">
			    <font size=%s><b>&nbsp;#</b></font>
		    </td>
		`, bgcolor, fontSize)

	// (2) header title field
	for _, one := range nameWidthList {
		str := fmt.Sprintf(`
			    <td width="%s">
				    <font size=2>%s</font>
			    </td>
			`, one.Width, one.Name)
		strTR += str
	}

	if hasCheckBox {	
		// (3) header checkbox field
		strTR += `
			    <td width="15">
				    <input type="checkbox" onclick="toggle(this);" />
			    </td>
			</tr>
			`
	}
	return strTR
}
