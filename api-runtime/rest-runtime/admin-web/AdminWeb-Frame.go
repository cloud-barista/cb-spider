// Cloud Info Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.06.

package adminweb

import (
	cblogger "github.com/cloud-barista/cb-log"
	cr "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	"github.com/sirupsen/logrus"

	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

var cblog *logrus.Logger

func init() {
	cblog = cblogger.GetLogger("CLOUD-BARISTA")
}

type NameWidth struct {
	Name  string
	Width string
}

// ================ Frame
func Frame(c echo.Context) error {
	cblog.Info("call Frame()")

	htmlStr := `
<html>
  <head>
    <title>CB-Spider Admin Web Tool ....__^..^__....</title>
    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
  </head>
    <frameset rows="150,*" frameborder="Yes" border=1">
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

// ================ Top Page
func Top(c echo.Context) error {
	cblog.Info("call Top()")

	htmlStr := ` 
    <html>
    <head>
        <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
        <style>
            .selectedMenu {
                background-color: #f0f0f0;
                color: #000;
                font-weight: bold;
            }
            #menuDashboard {
                display: block;
                margin-top: 8px;
                color: #FFFFFF;
                text-decoration: none;
                background-color: #87CEEB;
                text-align: center;
                padding: 6px 10px;
                border-radius: 4px;
                font-weight: bold;
                font-size: 12px;
                width: 60px;
            }
            #menuDashboard:hover {
                background-color: #B0E0E6;
            }
        </style>
        <script>
            function selectMenu(selectedId) {
                document.querySelectorAll('td a').forEach(function(menu) {
                    menu.classList.remove('selectedMenu');
                });
                var selectedElement = document.getElementById(selectedId);
                if (selectedElement) {
                    selectedElement.classList.add('selectedMenu');
                } else {
                    console.error('Element not found:', selectedId);
                }
            }
            window.onload = function() {
                selectMenu('menuDriver');
            };
        </script>
    </head>
    <body>    
        <table border="0" bordercolordark="#FFFFFF" cellpadding="0" cellspacing="1" bgcolor="#FFFFFF"  style="font-size:small;">      
            <tr bgcolor="#FFFFFF" align="left">
                <td rowspan="2" width="70" bgcolor="#FFFFFF" align="center">
                    <!-- CB-Spider Logo -->
                    <a href="../adminweb" target="_top">
                        <img height="45" width="45" src="./images/logo.png" border='0' hspace='0' vspace='1' align="middle">
                    </a>
                    <font size=1>$$TIME$$</font>
                    <br>
                    <!-- Dashboard Button -->
                    <a href="dashboard" target="main_frame" id="menuDashboard" onclick="selectMenu('menuDashboard')">
                        Dashboard
                    </a>
                </td>
            
            <td width="150"> 
                <!-- Drivers Management --> 
                <a href="driver" target="main_frame" id="menuDriver" onclick="selectMenu('menuDriver')">
                    <font size=2>1.Driver</font>
                </a>
            </td>
            <td width="190">       
                <!-- Credential Management -->
                <a href="credential" target="main_frame" id="menuCredential" onclick="selectMenu('menuCredential')">
                    <font size=2>1.Credential</font>
                </a>
            </td>
            <td width="130">
                <!-- Regions Management -->
                <a href="region" target="main_frame" id="menuRegion" onclick="selectMenu('menuRegion')"> 
                    <font size=2>1.Region</font>
                </a>
            </td>
            <td width="300">
                <!-- Connection Management -->
                <a href="connectionconfig" target="main_frame" id="menuConnection" onclick="selectMenu('menuConnection')">
                    <font size=2>2.CONNECTION</font>
                </a>
            </td>
            <td>
                <br>

                ¦&nbsp;&nbsp;
                <!-- RegionZone Info -->
                <a href="regionzone/region not set" target="main_frame" id="regionzoneHref" onclick="selectMenu('regionzoneHref')">
                    <font size=2>Region/Zone</font>
                </a>

                &nbsp;
                &nbsp;

                <a href="priceinfo/region not set" target="main_frame" id="priceinfoHref" onclick="selectMenu('priceinfoHref')">
                    <font size=2>Price</font>
                </a>
            </td>
            <td width="280">
                <!-- Display Connection Config -->
		<label id="connConfig" hidden></label>
		<input style="font-size:12px;font-weight:bold;text-align:center;background-color:#EDF7F9;" type="text" id="connDisplay" name="connDisplay" size = 35 disabled value="CloudOS: Region / Zone">

            </td>
	</tr>

        <tr bgcolor="#FFFFFF" align="left">
            <td width="150">

                <br>

                <!-- VPC/Subnet Management -->
                <a href="vpc/region not set" target="main_frame" id="vpcHref" onclick="selectMenu('vpcHref')">
                    <font size=2>1.VPC/Subnet</font>
                </a>
		&nbsp;
                <a href="vpcmgmt/region not set" target="main_frame" id="vpcmgmtHref" onclick="selectMenu('vpcmgmtHref')">
                    <font size=2>[mgmt]</font>
                </a>                

                <br>
                <br>
                <br>
            </td>
            <td width="190">
            
                <br>

                <!-- SecurityGroup Management -->
                <a href="securitygroup/region not set" target="main_frame" id="securitygroupHref" onclick="selectMenu('securitygroupHref')">
                    <font size=2>1.1.SecurityGroup</font>
                </a>
		&nbsp;
                <a href="securitygroupmgmt/region not set" target="main_frame" id="securitygroupmgmtHref" onclick="selectMenu('securitygroupmgmtHref')">
                    <font size=2>[mgmt]</font>
                </a>

                <br>
                <br>
                <br>

            </td>
            <td width="130">
            
                <br>

                <!-- KeyPair Management -->
                <a href="keypair/region not set" target="main_frame" id="keypairHref" onclick="selectMenu('keypairHref')">
                    <font size=2>1.KeyPair</font>
                </a>
		&nbsp;
                <a href="keypairmgmt/region not set" target="main_frame" id="keypairmgmtHref" onclick="selectMenu('keypairmgmtHref')">
                    <font size=2>[mgmt]</font>
                </a>

                <br>
                <br>
                <br>

            </td>
            <td width="300">

                <br>

                <!-- NLB Management -->
                <a href="nlb/region not set" target="main_frame" id="nlbHref" onclick="selectMenu('nlbHref')">
                    <font size=2>3.NLB</font>
                </a>
                &nbsp;
                <a href="nlbmgmt/region not set" target="main_frame" id="nlbmgmtHref" onclick="selectMenu('nlbmgmtHref')">
                    <font size=2>[mgmt]</font>
                </a>

                <br>

                <!-- VM Management -->
                <a href="vm/region not set" target="main_frame" id="vmHref" onclick="selectMenu('vmHref')">
                    <font size=2>2.VM</font>
                </a>
                &nbsp;
                <a href="vmmgmt/region not set" target="main_frame" id="vmmgmtHref" onclick="selectMenu('vmmgmtHref')">
                    <font size=2>[mgmt]</font>
                </a>
                &nbsp;
                &nbsp;
                &nbsp;
                ⇆
                &nbsp;
                &nbsp;
                <!-- MyImage Management -->
                <a href="myimage/region not set" target="main_frame" id="myimageHref" onclick="selectMenu('myimageHref')">
                    <font size=2>3.MyImage</font>
                </a>
                &nbsp;
                <a href="myimagemgmt/region not set" target="main_frame" id="myimagemgmtHref" onclick="selectMenu('myimagemgmtHref')">
                    <font size=2>[mgmt]</font>
                </a>            

                <br>

                <!-- Disk Management -->
                <a href="disk/region not set" target="main_frame" id="diskHref" onclick="selectMenu('diskHref')">
                    <font size=2>2.Disk</font>
                </a>
                &nbsp;
                <a href="diskmgmt/region not set" target="main_frame" id="diskmgmtHref" onclick="selectMenu('diskmgmtHref')">
                    <font size=2>[mgmt]</font>
                </a>

                <br>
                <br>

                <!-- PMKS(K8S) Management -->
                <a href="cluster/region not set" target="main_frame" id="clusterHref" onclick="selectMenu('clusterHref')">
                    <font size=2>2.PMKS</font>
                </a>
                &nbsp;
                <a href="clustermgmt/region not set" target="main_frame" id="clustermgmtHref" onclick="selectMenu('clustermgmtHref')">
                    <font size=2>[mgmt]</font>
                </a>
                &nbsp;
                <!-- <a href="clusterdashboard" target="main_frame" id="clusterdashboardHref" onclick="selectMenu('clusterdashboardHref')">
                    <font size=2>[Dashboard]</font>
                </a> -->
                <!-- <a href="../adminweb/clusterdashboard" target="main_frame">                  
                  <img height="40" width="40" src="./images/pmks.png" border='0' hspace='0' vspace='1' align="middle">
                </a> -->

            </td>
            <td width="280">
                        
                <br>

                ¦&nbsp;&nbsp;

                <!-- PublicImage Management -->
                <a href="vmimage/region not set" target="main_frame" id="vmimageHref" onclick="selectMenu('vmimageHref')">
                    <font size=2>VM Image</font>
                </a>

                &nbsp;
                &nbsp;

                <!-- Spec Management -->
                <a href="vmspec/region not set" target="main_frame" id="vmspecHref" onclick="selectMenu('vmspecHref')">
                    <font size=2>VM Spec</font>
                </a>

                &nbsp;
                &nbsp;

                <!-- This CB-Spider Info -->
                <a href="spiderinfo" target="main_frame" id="spiderinfoHref" onclick="selectMenu('spiderinfoHref')">
                    <font size=2>Spider Info</font>
                </a>

                <br>
                <br>
                <br>

            </td>

        </tr>

    </table>
</body>
</html>
	`

	htmlStr = strings.ReplaceAll(htmlStr, "$$TIME$$", cr.ShortStartTime)
	return c.HTML(http.StatusOK, htmlStr)
}

// ================ Log Page
func Log(c echo.Context) error {
	cblog.Info("call Log()")

	htmlStr := ` 
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
