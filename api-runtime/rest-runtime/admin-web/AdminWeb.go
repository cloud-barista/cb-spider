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
	cr "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"

	"net/http"
	"io/ioutil"
	"strings"
	"github.com/labstack/echo"
	"encoding/json"
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

//================ Frame
func Frame(c echo.Context) error {
	cblog.Info("call Frame()")

        htmlStr :=  `
<html>
  <head>
    <title>CB-Spider Admin Web Tool ....__^..^__....</title>
    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
  </head>
    <frameset rows="66,*" frameborder="Yes" border=1">
        <frame src="adminweb/top" name="top_frame" scrolling="auto" noresize marginwidth="0" marginheight="0"/>
        <frameset frameborder="Yes" border=1">
            <frame src="adminweb/driver" name="main_frame" scrolling="auto" noresize marginwidth="5" marginheight="0"/> 
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
                    <font size=2>1.driver</font>
                </a>
            </td>
            <td width="120">       
                <!-- Credential Management -->
                <a href="credential" target="main_frame">            
                    <font size=2>1.credential</font>
                </a>
            </td>
            <td width="80">       
                <!-- Regions Management -->
                <a href="region" target="main_frame">            
                    <font size=2>1.region</font>
                </a>
            </td>
            <td width="120">
                <!-- Connection Management -->
                <a href="connectionconfig" target="main_frame">            
                    <font size=2>2.CONNECTION</font>
                </a>
            </td>
            <td width="210">
                <!-- Display Connection Config -->
		<label id="connConfig" hidden></label>
		<input style="font-size:11px;font-weight:bold;text-align:center;background-color:#EDF7F9;" type="text" id="connDisplay" name="connDisplay" size = 24 disabled value="CloudOS: Region / Zone">

            </td>
            <td rowspan="2" width="60">       
                <!-- This CB-Spider Info -->
                <a href="spiderinfo" target="main_frame">            
                    <font size=2>info</font>
                </a>
            </td>
	</tr>

        <tr bgcolor="#FFFFFF" align="left">
            <td width="100">
                <!-- VPC/Subnet Management -->
                <a href="vpc/region not set" target="main_frame" id="vpcHref">
                    <font size=2>1.vpc/subnet</font>
                </a>
            </td>
            <td width="120">
                <!-- SecurityGroup Management -->
                <a href="security/region not set" target="main_frame" id="securityHref">
                    <font size=2>1.1.security group</font>
                </a>
            </td>
            <td width="80">
                <!-- KeyPair Management -->
                <a href="keypair/region not set" target="main_frame" id="keypairHref">
                    <font size=2>1.keypair</font>
                </a>
            </td>
            <td width="120">
                <!-- VM Management -->
                <a href="vm/region not set" target="main_frame" id="vmHref">
                    <font size=2>2.VM</font>
                </a>
            </td>
            <td width="210">
                <!-- Image Management -->
                <a href="image/region not set" target="main_frame" id="imageHref">
                    <font size=2>image(tbd)</font>
                </a>
		&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
                <!-- Spec Management -->
                <a href="spec/region not set" target="main_frame" id="specHref">
                    <font size=2>spec</font>
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

func makeSelect_html(onchangeFunctionName string) string {
	strList := cloudosList()

	strSelect := `<select name="text_box" id="1" onchange="` + onchangeFunctionName + `(this)">`
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

func getResourceList_JsonByte(resourceName string) ([]byte, error) {
        // cr.ServicePort = ":1024"
	url := "http://localhost" + cr.ServicePort + "/spider/" + resourceName

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
        url := "http://localhost" + cr.ServicePort + "/spider/" + resourceName
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
	url := "http://localhost" + cr.ServicePort + "/spider/" + resourceName + "/" + name

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
			    <font size=%s><b>&nbsp;X</b></font>
			</a>
			&nbsp;
		    </td>
		</tr>
       		`, colspan, f5_href, fontSize, delete_href, fontSize) 

	return strTR
}

//         fieldName-width
// number, fieldName0-200, fieldName1-400, ... , checkbox
func makeTitleTRList_html(bgcolor string, fontSize string, nameWidthList []NameWidth) string {
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
	
	// (3) header checkbox field
        strTR += `
		    <td width="15">
			    <input type="checkbox" onclick="toggle(this);" />
		    </td>
		</tr>
		`
	return strTR
}
