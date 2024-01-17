// Cloud Info Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.06.

package adminweb

import (
	"encoding/json"
	"fmt"
	"strings"

	cr "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager"

	"bytes"
	"io/ioutil"
	"net/http"
)

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

//----------------

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
		ResultList []cres.KeyPairInfo `json:"keypair"`
	}
	json.Unmarshal(resBody, &info)

	var nameList []string
	for _, keypair := range info.ResultList {
		nameList = append(nameList, keypair.IId.NameId)
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

func myImageList(connConfig string) []string {
	resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "myimage")
	if err != nil {
		cblog.Error(err)
	}
	var info struct {
		ResultList []cres.MyImageInfo `json:"myimage"`
	}
	json.Unmarshal(resBody, &info)

	var nameList []string
	for _, myImage := range info.ResultList {
		nameList = append(nameList, myImage.IId.NameId)
	}
	return nameList
}

// -------------

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

func getPriceInfoJsonString(connConfig string, resourceName string, productFamily string, regionName string, filter []cres.KeyValue, target interface{}) error {
	url := fmt.Sprintf("http://localhost:1024/spider/%s/%s/%s", resourceName, productFamily, regionName)

	reqBody := struct {
		ConnectionName string          `json:"ConnectionName"`
		FilterList     []cres.KeyValue `json:"FilterList"`
	}{
		ConnectionName: connConfig,
		FilterList:     filter,
	}

	jsonValue, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	request, err := http.NewRequest("GET", url, bytes.NewBuffer(jsonValue))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	return json.NewDecoder(resp.Body).Decode(target)
}

// F5, X ("5", "driver", "deleteDriver()", "2")
func makeActionTR_html(colspan string, f5_href string, delete_href string, fontSize string) string {
	if fontSize == "" {
		fontSize = "2"
	}

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

//	fieldName-width
//
// number, fieldName0-200, fieldName1-400, ... , checkbox
func makeTitleTRList_html(bgcolor string, fontSize string, nameWidthList []NameWidth, hasCheckBox bool) string {
	if bgcolor == "" {
		bgcolor = "#DDDDDD"
	}
	if fontSize == "" {
		fontSize = "2"
	}

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

// REST URL logging page
func genLoggingGETURL(connConfig string, rsType string) string {
	/* return example
	<script type="text/javascript">
		parent.frames["log_frame"].Log("curl -sX GET http://localhost:1024/spider/vpc -H 'Content-Type: application/json' -d '{"ConnectionName": "aws-ohio-config"}'   ");
	</script>
	*/

	url := "http://" + "localhost" + cr.ServerPort + "/spider/" + rsType + " -H 'Content-Type: application/json' -d '{\\\"ConnectionName\\\": \\\"" + connConfig + "\\\"}'"
	htmlStr := `
                <script type="text/javascript">
                `
	htmlStr += `    parent.frames["log_frame"].Log("curl -sX GET ` + url + `");`
	htmlStr += `
                </script>
                `
	return htmlStr
}

// REST URL logging page
func genLoggingGETURL2(connConfig string, rsType string) string {
	/* return example
	parent.frames["log_frame"].Log("curl -sX GET http://localhost:1024/spider/vpc -H 'Content-Type: application/json' -d '{"ConnectionName": "aws-ohio-config"}'   ");
	*/
	url := "http://" + "localhost" + cr.ServerPort + "/spider/" + rsType + " -H 'Content-Type: application/json' -d '{\\\"ConnectionName\\\": \\\"" + connConfig + "\\\"}'"
	htmlStr := `    parent.frames["log_frame"].Log("curl -sX GET ` + url + `");`
	return htmlStr
}

func genLoggingResult(response string) string {

	/*--------------------
		    {
	               "Key" : "Property",
	               "Value" : "{\"NodeNameType\":\"lan-ip\",\"NetworkType\":\"GR\"}"
	            },
		----------------------*/
	// to escape back-slash in the 'Property' Values
	response = strings.ReplaceAll(response, `\"`, `"`)

	htmlStr := `
                <script type="text/javascript">
                `
	htmlStr += `    parent.frames["log_frame"].Log("   ==> ` + strings.ReplaceAll(response, "\"", "\\\"") + `");`
	htmlStr += `
                </script>
                `
	return htmlStr
}

func genLoggingResult2(response string) string {

	/*--------------------
		    {
	               "Key" : "Property",
	               "Value" : "{\"NodeNameType\":\"lan-ip\",\"NetworkType\":\"GR\"}"
	            },
		----------------------*/
	// to escape back-slash in the 'Property' Values
	response = strings.ReplaceAll(response, `\"`, `"`)

	htmlStr := `    parent.frames["log_frame"].Log("   ==> ` + strings.ReplaceAll(response, "\"", "\\\"") + `");`

	return htmlStr
}
