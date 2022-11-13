// Cloud Info Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2022.11.

package adminweb

import (
	"fmt"

	cr "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	"strconv"

	"encoding/json"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)


//====================================== Cluster: Provider Managed Kubernetes(PMKS)

// number, CLUSTERNAME/Version, Status/CreatedTime, NetworkInfo(VPC/Sub/SG), AccessInfo(AccessPoint/KubeConfig), Addons, 
// NodeGroups,  
// Additional Info, checkbox
func makeClusterTRList_html(bgcolor string, height string, fontSize string, providerName string, connConfig string, infoList []*cres.ClusterInfo) string {
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
                            <font size=%s>$$CLUSTERNAME$$</font>
                            <br>
                            <font size=%s>$$VERSION$$</font>
                    </td>
                    <td>
                            <font size=%s><mark>$$STATUS$$</mark></font>
                            <br>
                            <font size=%s>$$CREATEDTIME$$</font>
                    </td>                    
                    <td>
                            <font size=%s>$$VPC$$</font>
                            <br>
                            <font size=%s>$$SUBNET$$</font>
                            <br>
                            <font size=%s>$$SECURITYGROUP$$</font>
                    </td>
                    <td align="left">
                            <font size=%s>$$ENDPOINT$$</font>
                            <br>
                            ----------
                            <br>
                            <textarea style="font-size:12px;text-align:left;" disabled rows=13 cols=40>
$$KUBECONFIG$$
                            </textarea>
                    </td>
		    <td>
                            <font size=%s>$$ADDONS$$</font>
                    </td>
                    <td align="left">
                            <font size=%s>$$NODEGROUPS$$</font>
                    </td>
                    <td>
                            <br>
                            <textarea style="font-size:12px;text-align:left;" disabled rows=13 cols=40>
$$ADDITIONALINFO$$
                            </textarea>
                            <br>
                            <br>
                            <br>
                    </td>
                    <td>
                        <input type="checkbox" name="check_box" value=$$CLUSTERNAME$$>
                    </td>
                </tr>
                `, bgcolor, height, fontSize, fontSize, fontSize, fontSize, fontSize, 
                fontSize, fontSize, fontSize, fontSize, fontSize, fontSize)

	strRemoveNodeGroup := fmt.Sprintf(`
                <a href="javascript:$$REMOVENODEGROUP$$;">
                        <font color=red size=%s><b>&nbsp;X</b></font>
                </a>
                `, fontSize)

	strAddNodegroup := fmt.Sprintf(`
                <textarea style="font-size:12px;text-align:left;" name="nodegroup_text_box_$$ADDCLUSTER$$" id="nodegroup_text_box_$$ADDCLUSTER$$" rows=13 cols=50>
%s
                </textarea>
                <a href="javascript:$$ADDNODEGROUP$$;">
                        <font size=%s><mark><b>+</b></mark></font>
                </a>
		`, generateNodeGroupReqString(providerName), fontSize)



	strData := ""
	// set data and make TR list
	for i, one := range infoList {
		str := strings.ReplaceAll(strTR, "$$NUM$$", strconv.Itoa(i+1))
		str = strings.ReplaceAll(str, "$$CLUSTERNAME$$", one.IId.NameId)
		str = strings.ReplaceAll(str, "$$VERSION$$", one.Version)

		str = strings.ReplaceAll(str, "$$STATUS$$", string(one.Status))
		str = strings.ReplaceAll(str, "$$CREATEDTIME$$", one.CreatedTime.Format("2006.01.02 15:04:05 Mon"))

		// for VPC
		str = strings.ReplaceAll(str, "$$VPC$$", one.Network.VpcIID.NameId)

		// for Subnet
		strSRList := ""
		for _, one := range one.Network.SubnetIIDs {
			strSRList += one.NameId + "," 
		}
		strSRList = strings.TrimSuffix(strSRList, ",")
		strSRList = "[" + strSRList +  "]"
		str = strings.ReplaceAll(str, "$$SUBNET$$", strSRList)

		// for security rules info
		strSRList = ""
		for _, one := range one.Network.SecurityGroupIIDs {
			strSRList += one.NameId + "," 
		}
		strSRList = strings.TrimSuffix(strSRList, ",")
		strSRList = "[" + strSRList +  "]"
		str = strings.ReplaceAll(str, "$$SECURITYGROUP$$", strSRList)

		str = strings.ReplaceAll(str, "$$ENDPOINT$$", one.AccessInfo.Endpoint)

		str = strings.ReplaceAll(str, "$$KUBECONFIG$$", one.AccessInfo.Kubeconfig)

		// for Addons
		strAddonList := ""
		for _, kv := range one.Addons.KeyValueList {
			strAddonList += kv.Key + ":" + kv.Value + ", "
		}
		strAddonList = strings.TrimRight(strAddonList, ", ")
		str = strings.ReplaceAll(str, "$$ADDONS$$", strAddonList)

		var clusterName = one.IId.NameId
		// for NodeGroups
		strNodeGroupList := ""
		for _, one := range one.NodeGroupList {
			strNodeGroupList += "{<br>"
			strNodeGroupList += "&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;Name: " + one.IId.NameId + ", <br>"
			strNodeGroupList += "&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;ImageName:" + one.ImageIID.NameId + ", <br>"
			strNodeGroupList += "&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;VMSpecName:" + one.VMSpecName + ", <br>"
			strNodeGroupList += "&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;RootDiskType:" + one.RootDiskType + ", <br>"
			strNodeGroupList += "&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;RootDiskSize:" + one.RootDiskSize + ", <br>"
			strNodeGroupList += "&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;KeyPairName:" + one.KeyPairIID.NameId + ", <br>"
			strNodeGroupList += "&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;OnAutoScaling:" + strconv.FormatBool(one.OnAutoScaling) + ", <br>"
			strNodeGroupList += "&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;DesiredNodeSize:" + strconv.Itoa(one.DesiredNodeSize) + ", <br>"
			strNodeGroupList += "&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;MinNodeSize:" + strconv.Itoa(one.MinNodeSize) + ", <br>"
			strNodeGroupList += "&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;MaxNodeSize:" + strconv.Itoa(one.MaxNodeSize) + ", <br>"

			strNodeGroupList += "&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;Status:<mark> " + string(one.Status) + " </mark>, <br>"

			// Nodes
			strVMList := ""
			for idx, vmIID := range one.Nodes {
				strVMList += generateNodeInfoHyperlinkNodeString(idx, connConfig, vmIID.SystemId) + ", "
			}
			strVMList = strings.TrimRight(strVMList, ", ")
			
			strNodeGroupList += "&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;Nodes: <mark> [ " + strVMList + " ] </mark>" + "<br>"
			strNodeGroupList += "}"

			// masking to avoid complexity
			// KeyValueList
			// strNodeGroupList += "{"
			// for _, kv := range one.KeyValueList {
			// 	strNodeGroupList += kv.Key + ":" + kv.Value + ", "
			// }
			// strNodeGroupList = strings.TrimRight(strNodeGroupList, ", ")
			// strNodeGroupList += "}"

			var nodegroupName = one.IId.NameId
			strNodeGroupList += strings.ReplaceAll(strRemoveNodeGroup, "$$REMOVENODEGROUP$$", "removeNodeGroup('"+clusterName+"', '"+nodegroupName+"')")

			strNodeGroupList += "<br>----------"
			strNodeGroupList += "<br>"
		}
		clusterAddNodeGroup := strings.ReplaceAll(strAddNodegroup, "$$ADDCLUSTER$$", clusterName)
		strNodeGroupList += strings.ReplaceAll(clusterAddNodeGroup, "$$ADDNODEGROUP$$", "postNodeGroup('"+clusterName+"')")
		str = strings.ReplaceAll(str, "$$NODEGROUPS$$", strNodeGroupList)

		// for KeyValueList
		strKeyList := ""
		for _, kv := range one.KeyValueList {
			strKeyList += kv.Key + ":" + kv.Value + ", "
		}
		strKeyList = strings.TrimRight(strKeyList, ", ")
		str = strings.ReplaceAll(str, "$$ADDITIONALINFO$$", strKeyList)

		strData += str
	}

	return strData
}

func generateNodeInfoHyperlinkNodeString(idx int, connConfig string, nodeCSPName string) string {
	return fmt.Sprintf(`
                <a href="javascript:openNodeInfoWindow('%s', '%s');">
                        <font size=2>node-%d</font>
                </a>
		`, connConfig, nodeCSPName, idx+1)
}

// make the string of javascript function
func makeOpenNodeInfoFunc_js() string {
	//curl -sX GET http://localhost:1024/spider/cspvm/"i-6we0n1kv4s13ncj4pfc3"?ConnectionName=alibaba-tokyo-config

	strFunc := `
                function openNodeInfoWindow(connConfig, nodeCSPName) {

                        var xhr = new XMLHttpRequest();
                        xhr.open("GET", "$$SPIDER_SERVER$$/spider/cspvm/" + nodeCSPName + "?ConnectionName=" + connConfig, false);

			 // client logging
			parent.frames["log_frame"].Log("curl -sX GET " + "$$SPIDER_SERVER$$/spider/cspvm" + nodeCSPName + "?ConnectionName=" + connConfig);

                        xhr.send(null);

			// client logging
			parent.frames["log_frame"].Log("   => " + xhr.response);

                        var win = window.open("", "_blank", "width=500,height=690,location=no,scrollbars=no,menubar=no,status=no,titlebar=no,toolbar=no,resizable=no,top=300,left=500,");
                        var jsonPretty = JSON.stringify(JSON.parse(xhr.response),null,2);  
                        var textArea = '<textarea style="font-size:12px;text-align:left;resize:none;" disabled rows=47 cols=66>' + jsonPretty + '</textarea>'
                        win.document.write(textArea);
                }
        `
	strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://"+cr.ServiceIPorName+cr.ServicePort) // cr.ServicePort = ":1024"
	return strFunc
}

func generateNodeGroupReqString(providerName string) string {
                
        Name := "economy"
        ImageName := "" 
        VMSpecName := "" 
        RootDiskType := ""
        RootDiskSize := "60" 
        KeyPairName := "keypair-01"
        OnAutoScaling := "true" 
        DesiredNodeSize := "2" 
        MinNodeSize := "1" 
        MaxNodeSize := "3"
        

        switch providerName {
        case "AWS":
        case "AZURE":
                VMSpecName = "Standard_B2s"
        case "GCP":
        case "ALIBABA":
                VMSpecName = "ecs.c6.xlarge"
                RootDiskType = "cloud_essd" 
        case "TENCENT":
                ImageName = "tlinux3.1x86_64"
                VMSpecName = "S3.MEDIUM8"
                RootDiskType = "CLOUD_BSSD" 

        case "IBM":
        case "CLOUDIT":
        case "OPENSTACK":
        case "NCP":
        case "KTCLOUD":
        case "NHNCLOUD":
        case "DOCKER":
        case "MOCK":
        case "CLOUDTWIN":
        default:
        }

        reqString := fmt.Sprintf(`{
        "Name" :            "%s", 
        "ImageName" :       "%s", 
        "VMSpecName" :      "%s", 
        "RootDiskType" :    "%s", 
        "RootDiskSize" :    "%s", 
        "KeyPairName" :     "%s",
        "OnAutoScaling" :   "%s", 
        "DesiredNodeSize" : "%s", 
        "MinNodeSize" :     "%s", 
        "MaxNodeSize" :     "%s"
}`, Name, ImageName, VMSpecName, RootDiskType, RootDiskSize, KeyPairName, 
 	OnAutoScaling, DesiredNodeSize, MinNodeSize, MaxNodeSize)

	return reqString
}
// make the string of javascript function
func makePostClusterFunc_js() string {

// curl -sX POST http://localhost:1024/spider/cluster -H 'Content-Type: application/json' -d \
//         '{
//                 "ConnectionName": "alibaba-tokyo-config",

//                 "ReqInfo": {
//                         "Name": "spider-cluser-01",
//                         "Version": "1.22.15-aliyun.1",

//                          "VPCName": "vpc-01",
//                         "SubnetNames": ["subnet-01"],
//                         "SecurityGroupNames": ["sg-01"]
//                 }
//         }' |json_pp

	strFunc := `
                function postCluster() {
                        var connConfig = parent.frames["top_frame"].document.getElementById("connConfig").innerHTML;
                        var textboxes = document.getElementsByName('text_box');

                        sendJson = '{ "ConnectionName" : "' + connConfig + '", "ReqInfo" : \
                        			{ \
                        				"Name" : "$$CLUSTERNAME$$", \
                        				"Version" : "$$VERSION$$", \
                        				"VPCName" : "$$VPC$$", \
                        				"SubnetNames" : $$SUBNET$$, \
                        				"SecurityGroupNames" : $$SECURITYGROUP$$, \
                        				"NodeGroupList": [ $$NODEGROUPLIST$$ ] \
                        			} \
                        		}'

                        for (var i = 0; i < textboxes.length; i++) {
                                switch (textboxes[i].id) {
                                        case "1":
                                                sendJson = sendJson.replace("$$CLUSTERNAME$$", textboxes[i].value);
                                                break;
                                        case "2":
                                                sendJson = sendJson.replace("$$VERSION$$", textboxes[i].value);
                                                break;
                                        case "5":
                                                sendJson = sendJson.replace("$$VPC$$", textboxes[i].value);
                                                break;
                                        case "6":
                                                sendJson = sendJson.replace("$$SUBNET$$", textboxes[i].value);
                                                break;
                                        case "7":
                                                sendJson = sendJson.replace("$$SECURITYGROUP$$", textboxes[i].value);
                                                break;
                                        case "11":
                                        	nodeGroupListReq = textboxes[i].value;
                                        	if (nodeGroupListReq.trim() == "N/A") {
                                        		nodeGroupListReq = ""
                                        	}
                                                sendJson = sendJson.replace("$$NODEGROUPLIST$$", nodeGroupListReq);
                                                break;
                                        default:
                                                break;
                                }
                        }
                        var xhr = new XMLHttpRequest();
                        xhr.open("POST", "$$SPIDER_SERVER$$/spider/cluster", false);
                        xhr.setRequestHeader('Content-Type', 'application/json');

			// client logging
			parent.frames["log_frame"].Log("curl -sX POST " + "$$SPIDER_SERVER$$/spider/cluster -H 'Content-Type: application/json' -d '" + sendJson + "'");

                        xhr.send(sendJson);

			// client logging
			parent.frames["log_frame"].Log("   ==> " + xhr.response);

            location.reload();
                }
        `
	strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://"+cr.ServiceIPorName+cr.ServicePort) // cr.ServicePort = ":1024"
	return strFunc
}

// make the string of javascript function
func makeDeleteClusterFunc_js() string {
// curl -sX DELETE http://localhost:1024/spider/cluster/spider-cluser-01 -H 'Content-Type: application/json' -d \
//         '{ 
//                 "ConnectionName": "alibaba-tokyo-config"
//         }' |json_pp

	strFunc := `
                function deleteCluster() {
                        var connConfig = parent.frames["top_frame"].document.getElementById("connConfig").innerHTML;
                        var checkboxes = document.getElementsByName('check_box');

                        for (var i = 0; i < checkboxes.length; i++) { // @todo make parallel executions
                                if (checkboxes[i].checked) {
                                        var xhr = new XMLHttpRequest();
                                        xhr.open("DELETE", "$$SPIDER_SERVER$$/spider/cluster/" + checkboxes[i].value, false);
                                        xhr.setRequestHeader('Content-Type', 'application/json');
					sendJson = '{ "ConnectionName": "' + connConfig + '"}'

					// client logging
					parent.frames["log_frame"].Log("curl -sX DELETE " + "$$SPIDER_SERVER$$/spider/cluster/" + checkboxes[i].value + " -H 'Content-Type: application/json' -d '" + sendJson + "'");

                                        xhr.send(sendJson);

					// client logging
					parent.frames["log_frame"].Log("   ==> " + xhr.response);
                                }
                        }
            location.reload();
                }
        `
	strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://"+cr.ServiceIPorName+cr.ServicePort) // cr.ServicePort = ":1024"
	return strFunc
}

// make the string of javascript function
func makePostNodeGroupFunc_js() string {

// curl -sX POST http://localhost:1024/spider/cluster/spider-cluser-01/nodegroup -H 'Content-Type: application/json' -d \
//         '{
//                 "ConnectionName": "alibaba-tokyo-config",

//                 "ReqInfo": {
//                         "Name": "Economy", 
//                         "ImageName": "ubuntu_18_04_x64_20G_alibase_20220322.vhd", 
//                         "VMSpecName": "ecs.c6.xlarge", 
//                         "RootDiskType": "cloud_essd", 
//                         "RootDiskSize": "70", 
//                         "KeyPairName": "keypair-01",
//                         "OnAutoScaling": "true", 
//                         "DesiredNodeSize": "2", 
//                         "MinNodeSize": "2", 
//                         "MaxNodeSize": "2"
//                 }
//         }' |json_pp

	strFunc := `
                function postNodeGroup(clusterName) {
                        var connConfig = parent.frames["top_frame"].document.getElementById("connConfig").innerHTML;

                        var textbox = document.getElementById('nodegroup_text_box_' + clusterName);
                        sendJson = '{ "ConnectionName" : "' + connConfig + '", "ReqInfo" :  $$NODEGROUPINFO$$ }'

                        sendJson = sendJson.replace("$$NODEGROUPINFO$$", textbox.value);

                        var xhr = new XMLHttpRequest();
                        xhr.open("POST", "$$SPIDER_SERVER$$/spider/cluster/" + clusterName + "/nodegroup", false);
                        xhr.setRequestHeader('Content-Type', 'application/json');

			 // client logging
			parent.frames["log_frame"].Log("curl -sX POST " + "$$SPIDER_SERVER$$/spider/cluster/" + clusterName + "/nodegroup" + " -H 'Content-Type: application/json' -d '" + sendJson + "'");

                        xhr.send(sendJson);

			// client logging
			parent.frames["log_frame"].Log("   => " + xhr.response);

                        location.reload();
                }
        `
	strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://"+cr.ServiceIPorName+cr.ServicePort) // cr.ServicePort = ":1024"
	return strFunc
}

// make the string of javascript function
func makeRemoveNodeGroupFunc_js() string {
	//curl -sX DELETE http://localhost:1024/spider/cluster/spider-cluser-01/nodegroup/Economy -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}'

	strFunc := `
                function removeNodeGroup(clusterName, nodegroupName) {
                        var connConfig = parent.frames["top_frame"].document.getElementById("connConfig").innerHTML;

                        var xhr = new XMLHttpRequest();
                        xhr.open("DELETE", "$$SPIDER_SERVER$$/spider/cluster/" + clusterName + "/nodegroup/" + nodegroupName, false);
                        xhr.setRequestHeader('Content-Type', 'application/json');
                        sendJson = '{ "ConnectionName": "' + connConfig + '"}'

			 // client logging
			parent.frames["log_frame"].Log("curl -sX DELETE " + "$$SPIDER_SERVER$$/spider/cluster/" + clusterName + "/nodegroup/" + nodegroupName + " -H 'Content-Type: application/json' -d '" + sendJson + "'");

                        xhr.send(sendJson);

			// client logging
			parent.frames["log_frame"].Log("   => " + xhr.response);

                        location.reload();
                }
        `
	strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://"+cr.ServiceIPorName+cr.ServicePort) // cr.ServicePort = ":1024"
	return strFunc
}


func Cluster(c echo.Context) error {
	cblog.Info("call Cluster()")

	connConfig := c.Param("ConnectConfig")
	if connConfig == "region not set" {
		htmlStr := `
            <html>
            <head>
                <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
		<style>
		th {
		  border: 1px solid lightgray;
		}
		td {
		  border: 1px solid lightgray;
		  border-radius: 4px;
		}
		</style>
                <script type="text/javascript">
                alert(connConfig)
                </script>
            </head>
            <body>
                <br>
                <br>
                <label style="font-size:24px;color:#606262;">&nbsp;&nbsp;&nbsp;Please select a Connection Configuration! (MENU: 2.CONNECTION)</label>   
            </body>
        `

		return c.HTML(http.StatusOK, htmlStr)
	}

	// make page header
	htmlStr := `
                <html>
                <head>
                    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
			<style>
			th {
			  border: 1px solid lightgray;
			}
			td {
			  border: 1px solid lightgray;
			  border-radius: 4px;
			}
			</style>
                    <script type="text/javascript">
                `
	// (1) make Javascript Function
	htmlStr += makeCheckBoxToggleFunc_js()
	htmlStr += makePostClusterFunc_js()
	htmlStr += makeDeleteClusterFunc_js()	
	htmlStr += makePostNodeGroupFunc_js()
	htmlStr += makeRemoveNodeGroupFunc_js()
	htmlStr += makeOpenNodeInfoFunc_js()

	htmlStr += `
                    </script>
                </head>

                <body>
                    <table bordercolordark="#F8F8FF" cellpadding="0" cellspacing="1" bgcolor="#FFFFFF"  style="font-size:small;">
                `

	// (2) make Table Action TR
	// colspan, f5_href, delete_href, fontSize
	htmlStr += makeActionTR_html("10", "", "deleteCluster()", "2")

	// (3) make Table Header TR
	nameWidthList := []NameWidth{
		{"Cluster Name / Version", "200"},
		{"Cluster Status / Created Time", "200"},
		{"VPC / Subnet / Security Group", "400"},
		{"Endpoint / Kubeconfig", "250"},
		{"Addons", "200"},
		{"NodeGroups", "300"},
		{"Additional Info", "300"},
	}
	htmlStr += makeTitleTRList_html("#DDDDDD", "2", nameWidthList, true)

	// (4) make TR list with info list
	// (4-1) get info list

	// client logging
	htmlStr += genLoggingGETURL(connConfig, "cluster")

	resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "cluster")
	if err != nil {
		cblog.Error(err)
		// client logging
                htmlStr += genLoggingResult(err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// client logging
	htmlStr += genLoggingResult(string(resBody[:len(resBody)-1]))
	var info struct {
		Connection string
		ClusterInfoList []*cres.ClusterInfo
	}
	json.Unmarshal(resBody, &info)

	providerName, _ := getProviderName(connConfig)

	// (4-2) make TR list with info list
	htmlStr += makeClusterTRList_html("", "", "", providerName, connConfig, info.ClusterInfoList)

	// (5) make input field and add	
	vpcList := vpcList(connConfig)

	version := ""
	subnetName := `["subnet-01"]`
	sgName := `["sg-01"]`

	nodegroupList := ""

	switch providerName {
	case "AWS":
	case "AZURE":
		version = "1.22.11"
		nodegroupList = generateNodeGroupReqString(providerName)
	case "GCP":
	case "ALIBABA":
		version = "1.22.15-aliyun.1"
	case "TENCENT":
		version = "1.22.5"
	case "IBM":
	case "CLOUDIT":
	case "OPENSTACK":
	case "NCP":
	case "KTCLOUD":
	case "NHNCLOUD":
	case "DOCKER":
	case "MOCK":
	case "CLOUDTWIN":
	default:
	}

	// create message field
	htmlStr += `
                        <tr bgcolor="#FFFFFF" align="center" height="30">
                            <td bgcolor="#FFEFBA">
                                    <font size=2>&nbsp;create:&nbsp;</font>
                            </td>
		`

	// Cluster Name/Version, Status/CreatedTime
	htmlStr += `
                            <td>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="1" value="spider-cluster-01">
                                <br>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="2" value="##VERSION##">
                            </td>
                            <td style="vertical-align:top">
				<input style="font-size:12px;text-align:center;" type="text" name="text_box" id="3" disabled value="N/A">
                            </td>
                            <td style="vertical-align:top">
                   `

	// Select format of VPC name=text_box, id=5
	htmlStr += makeSelect_html("", vpcList, "5")

	// Subnet/SG  name=text_box, id=6/7
	htmlStr += `

				<br>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="6" value=##SUBNET##>
				<br>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="7" value=##SECURITYGROUP##>
                            </td>                            
                   `
        // AccessInfo and Addons 
        htmlStr += `
                            <td style="vertical-align:top">
				<input style="font-size:12px;text-align:center;" type="text" name="text_box" id="8" disabled value="N/A">
                            </td>
                            <td style="vertical-align:top">
				<input style="font-size:12px;text-align:center;" type="text" name="text_box" id="10" disabled value="N/A">
                            </td>
        	`

	// NodeGroup
	if nodegroupList == "" { // Tencent, Alibaba
        	htmlStr += `
                            <td style="vertical-align:top">
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="11" disabled value="N/A">
                            </td>
        	`
        } else {	// Azure, NHN
        	htmlStr += fmt.Sprintf(`
                            <td style="vertical-align:top">
                		<textarea style="font-size:12px;text-align:left;" name="text_box" id="11" rows=13 cols=50>
%s
                		</textarea>
                            </td>
		`, nodegroupList)
        }

	// AdditionalInfo
        htmlStr += `
                            <td style="vertical-align:top">
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="12" disabled value="N/A">
                            </td>
        	`


	// create button with '+'
	htmlStr += `
                            <td>
                                <a href="javascript:postCluster()">
                                    <font size=4><mark><b>+</b></mark></font>
                                </a>
                            </td>
                        </tr>
                `

	htmlStr = strings.ReplaceAll(htmlStr, "##VERSION##", version)
	htmlStr = strings.ReplaceAll(htmlStr, "##SUBNET##", subnetName)
	htmlStr = strings.ReplaceAll(htmlStr, "##SECURITYGROUP##", sgName)

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
