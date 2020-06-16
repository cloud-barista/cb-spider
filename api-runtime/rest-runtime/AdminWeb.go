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
	//cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
/*
	"strconv"

	ccim "github.com/cloud-barista/cb-spider/cloud-info-manager/connection-config-info-manager"
	im "github.com/cloud-barista/cb-spider/cloud-info-manager"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager/credential-info-manager"
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
    <meta name="verify-v1" content="+lxf1HQ/m7PqnfmwSZ1TeHlhMabM8AqXpTX3ZfmYSqM=" />
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
            <td rowspan="2" width="60" bgcolor="#FFFFFF">
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
                <a href="adminweb/credential" target="_blank">            
                    <font size=2>credential</font>
                </a>
            </td>
            <td width="100">       
                <!-- Regions Management -->
                <a href="adminweb/region" target="_blank">            
                    <font size=2>region</font>
                </a>
            </td>
            <td width="100">       
                <!-- Connection Management -->
                <a href="adminweb/connection" target="_blank">            
                    <font size=2>connection</font>
                </a>
            </td>
            <td width="100">       
                <!-- This CB-Spider Info -->
                <a href="adminweb/spider" target="_blank">            
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
                    <font size=2>image</font>
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

//================ Driver Management
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
        //fmt.Printf("%s", resBody)
	//str := fmt.Sprintf("%s", resBody)
	//fmt.Println(str)

        var info struct {
                Result []*dim.CloudDriverInfo `json:"driver"`
        }
	json.Unmarshal(resBody, &info)

fmt.Printf("\n\n =========== powekim: %#v\n\n", info)

	strTR :=  ` 
		<tr bgcolor="#FFFFFF" align="center">
		    <td width="200">
			    <font size=2>$$S1$$</font>
		    </td>
		    <td width="200">
			    <font size=2>$$S2$$</font>
		    </td>
		    <td width="250">
			    <font size=2>$$S3$$</font>
		    </td>
		</tr>
	`

	strData := ""
	for _, driver := range info.Result {
		str := strings.ReplaceAll(strTR, "$$S1$$", driver.DriverName)
		str = strings.ReplaceAll(str, "$$S2$$", driver.ProviderName)
		str = strings.ReplaceAll(str, "$$S3$$", driver.DriverLibFileName)
		strData += str
	}

	htmlStr :=  ` 
<html>
<head>
    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
</head>
<body>
    <table border="0" bordercolordark="#F8F8FF" cellpadding="0" cellspacing="1" bgcolor="#FFFFFF"  style="font-size:small;">      
                <tr bgcolor="#DDDDDD" align="center">
                    <td width="200">
                            <font size=2>Driver Name</font>
                    </td>
                    <td width="200">
                            <font size=2>Provider Name</font>
                    </td>
                    <td width="250">
                            <font size=2>Driver Library Name</font>
                    </td>
                </tr>
		$$DATA$$
    </table>
</body>
</html>
        `

	htmlStr = strings.ReplaceAll(htmlStr, "$$DATA$$", strData)
	return c.HTML(http.StatusOK, htmlStr)
}
