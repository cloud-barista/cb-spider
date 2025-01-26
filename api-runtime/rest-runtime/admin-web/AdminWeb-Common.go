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
		try {
			parent.frames["log_frame"].Log("curl -sX GET ` + url + `");
		} catch (e) {
			// Do nothing if error occurs
		}
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
	htmlStr := `
<script type="text/javascript">
    try {
        parent.frames["log_frame"].Log("curl -sX GET ` + url + `");
    } catch (e) {
        // Do nothing if error occurs
    }
</script>
`

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
		try {
			parent.frames["log_frame"].Log("   ==> ` + strings.ReplaceAll(response, "\"", "\\\"") + `");
		} catch (e) {
			// Do nothing if error occurs
		}
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

	htmlStr := `
	<script type="text/javascript">
		try {
			parent.frames["log_frame"].Log("   ==> ` + strings.ReplaceAll(response, "\"", "\\\"") + `");;
		} catch (e) {
			// Do nothing if error occurs
		}
	</script>
	`

	return htmlStr
}

// Fetch regions and map them to RegionName -> "Region / Zone"
func fetchRegions() (map[string]string, error) {
	resp, err := http.Get("http://localhost:1024/spider/region")
	if err != nil {
		return nil, fmt.Errorf("error fetching regions: %v", err)
	}
	defer resp.Body.Close()

	var regions struct {
		Regions []struct {
			RegionName       string `json:"RegionName"`
			KeyValueInfoList []struct {
				Key   string `json:"Key"`
				Value string `json:"Value"`
			} `json:"KeyValueInfoList"`
		} `json:"region"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&regions); err != nil {
		return nil, fmt.Errorf("error decoding regions: %v", err)
	}

	regionMap := make(map[string]string)
	for _, region := range regions.Regions {
		var regionValue, zoneValue string
		for _, kv := range region.KeyValueInfoList {
			if kv.Key == "Region" {
				regionValue = kv.Value
			} else if kv.Key == "Zone" {
				zoneValue = kv.Value
			}
		}
		if zoneValue == "" {
			zoneValue = "N/A"
		}
		regionMap[region.RegionName] = fmt.Sprintf("%s / %s", regionValue, zoneValue)
	}
	return regionMap, nil
}
