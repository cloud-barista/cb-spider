// Cloud Info Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.06.

package adminweb

import (
    "github.com/cloud-barista/cb-store/config"
    "github.com/sirupsen/logrus"
	cr "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"

	"net/http"
	"strings"
	"github.com/labstack/echo/v4"
)

var cblog *logrus.Logger
func init() {
	cblog = config.Cblogger
}

type NameWidth struct {
	Name string
	Width string
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
    <frameset rows="125,*" frameborder="Yes" border=1">
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

//================ Top Page
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
                    <font size=2>1.Driver</font>
                </a>
            </td>
            <td width="190">       
                <!-- Credential Management -->
                <a href="credential" target="main_frame">            
                    <font size=2>1.Credential</font>
                </a>
            </td>
            <td width="130">
                <!-- Regions Management -->
                <a href="region" target="main_frame">            
                    <font size=2>1.Region</font>
                </a>
            </td>
            <td width="190">
                <!-- Connection Management -->
                <a href="connectionconfig" target="main_frame">            
                    <font size=2>2.CONNECTION</font>
                </a>
            </td>
            <td width="240">
                <!-- Display Connection Config -->
		<label id="connConfig" hidden></label>
		<input style="font-size:11px;font-weight:bold;text-align:center;background-color:#EDF7F9;" type="text" id="connDisplay" name="connDisplay" size = 30 disabled value="CloudOS: Region / Zone">

            </td>
	</tr>

        <tr bgcolor="#FFFFFF" align="left">
            <td width="150">

                <br>

                <!-- VPC/Subnet Management -->
                <a href="vpc/region not set" target="main_frame" id="vpcHref">
                    <font size=2>1.VPC/Subnet</font>
                </a>
		&nbsp;
                <a href="vpcmgmt/region not set" target="main_frame" id="vpcmgmtHref">
                    <font size=2>[mgmt]</font>
                </a>                

                <br>
                <br>
                <br>
            </td>
            <td width="190">
            
                <br>

                <!-- SecurityGroup Management -->
                <a href="securitygroup/region not set" target="main_frame" id="securitygroupHref">
                    <font size=2>1.1.SecurityGroup</font>
                </a>
		&nbsp;
                <a href="securitygroupmgmt/region not set" target="main_frame" id="securitygroupmgmtHref">
                    <font size=2>[mgmt]</font>
                </a>

                <br>
                <br>
                <br>

            </td>
            <td width="130">
            
                <br>

                <!-- KeyPair Management -->
                <a href="keypair/region not set" target="main_frame" id="keypairHref">
                    <font size=2>1.KeyPair</font>
                </a>
		&nbsp;
                <a href="keypairmgmt/region not set" target="main_frame" id="keypairmgmtHref">
                    <font size=2>[mgmt]</font>
                </a>

                <br>
                <br>
                <br>

            </td>
            <td width="280">

                <br>

                <!-- NLB Management -->
                <a href="nlb/region not set" target="main_frame" id="nlbHref">
                    <font size=2>3.NLB</font>
                </a>
                &nbsp;
                <a href="nlbmgmt/region not set" target="main_frame" id="nlbmgmtHref">
                    <font size=2>[mgmt]</font>
                </a>

                <br>

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
                &nbsp;
                ⇆
                &nbsp;
                &nbsp;
                <!-- MyImage Management -->
                <a href="myimage/region not set" target="main_frame" id="myimageHref">
                    <font size=2>3.MyImage</font>
                </a>
                &nbsp;
                <a href="nlbmgmt/region not set" target="main_frame" id="myimagemgmtHref">
                    <font size=2>[mgmt]</font>
                </a>            

                <br>

                <!-- Disk Management -->
                <a href="disk/region not set" target="main_frame" id="diskHref">
                    <font size=2>2.Disk</font>
                </a>
                &nbsp;
                <a href="diskmgmt/region not set" target="main_frame" id="diskmgmtHref">
                    <font size=2>[mgmt]</font>
                </a>

                <br>
                <br>

                <!-- PMKS(K8S) Management -->
                <a href="cluster/region not set" target="main_frame" id="clusterHref">
                    <font size=2>2.PMKS</font>
                </a>
                &nbsp;
                <a href="clustermgmt/region not set" target="main_frame" id="clustermgmtHref">
                    <font size=2>[mgmt]</font>
                </a>
                &nbsp;
                <a href="clusterdashboard" target="main_frame" id="clusterdashboardHref">
                    <font size=2>[Dashboard]</font>
                </a>
                <!-- <a href="../adminweb/clusterdashboard" target="main_frame">                  
                  <img height="40" width="40" src="./images/pmks.png" border='0' hspace='0' vspace='1' align="middle">
                </a> -->

            </td>
            <td width="240">
                        
                <br>

                ¦&nbsp;&nbsp;

                <!-- PublicImage Management -->
                <a href="vmimage/region not set" target="main_frame" id="vmimageHref">
                    <font size=2>VM Image</font>
                </a>

                &nbsp;
                &nbsp;

                <!-- Spec Management -->
                <a href="vmspec/region not set" target="main_frame" id="vmspecHref">
                    <font size=2>VM Spec</font>
                </a>

                &nbsp;
                &nbsp;

                <!-- This CB-Spider Info -->
                <a href="spiderinfo" target="main_frame">            
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

//================ Log Page
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
