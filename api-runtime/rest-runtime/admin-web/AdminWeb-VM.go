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

	cr "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	"strconv"

	"encoding/json"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)


//====================================== VM

// number, VM Name/Control, VMStatus/Last Start Time, VMImage/VMSpec, VPC/Subnet/Security Group,
//         Network Interface/IP, DNS, Root Disk/Data Disk, SSH AccessPoint/Access Key/Access User Name, Additional Info, checkbox
func makeVMTRList_html(connConfig string, bgcolor string, height string, fontSize string, infoList []*cres.VMInfo) string {
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
			    <font size=%s>$$VMNAME$$</font>
			    <br>
			    <font size=%s>$$VMCONTROL$$</font>
			    <br>
			    <input style="font-size:12px;text-align:center;" type="text" name="myimage-name" id="myimage-name" value="spider-myimage-$$NUM$$">
			    <button type="button" onclick="postSnapshotVM('$$NUM$$', '$$VMNAME$$')">Snapshot</button>
                    </td>
                    <td>
                            <font size=%s>$$VMSTATUS$$</font>
                            <br>
                            <font size=%s>$$LASTSTARTTIME$$</font>
                    </td>                    
                    <td>
                            <font size=%s>$$IMAGETYPE$$</font>
                            <br>                    
                            <font size=%s>$$IMAGE$$</font>
                            <br>
                            <font size=%s>$$SPEC$$</font>
                    </td>
                    <td>
                            <font size=%s>$$VPC$$</font>
                            <br>
                            <font size=%s>$$SUBNET$$</font>
                            <br>
                            <font size=%s>$$SECURITYGROUP$$</font>
                    </td>
                    <td>
                            <font size=%s>$$NETWORKINTERFACE$$</font>
                            <br>
                            <font size=%s>$$PUBLICIP$$</font>
                            <br>
                            <font size=%s>$$PRIVATEIP$$</font>
                    </td>
                    <td>
                            <font size=%s>$$PUBLICDNS$$</font>
                            <br>
                            <font size=%s>$$PRIVATEDNS$$</font>
                    </td>
                    <td align=left>
                            <font size=%s>$$ROOTDISK$$</font>
                            <br>
                            <font size=%s>$$DATADISK$$</font>
                    </td>
                    <td>
                            <font size=%s>$$SSHACCESSPOINT$$</font>
                            <br>
                            <font size=%s>$$ACCESSKEY$$</font>
                            <br>
                            <font size=%s>$$ACCESSUSER$$</font>
                    </td>
                    <td>
                            <font size=%s>$$ADDITIONALINFO$$</font>
                    </td>
                    <td>
                        <input type="checkbox" name="check_box" value=$$VMNAME$$>
                    </td>
                </tr>
                `, bgcolor, height, fontSize, fontSize, fontSize, fontSize, fontSize, fontSize, fontSize, fontSize, fontSize, 
		  fontSize, fontSize, fontSize, fontSize, fontSize, fontSize, fontSize, fontSize, fontSize, fontSize, fontSize)

	strData := ""
	// set data and make TR list
	for i, one := range infoList {
		str := strings.ReplaceAll(strTR, "$$NUM$$", strconv.Itoa(i+1))
		str = strings.ReplaceAll(str, "$$VMNAME$$", one.IId.NameId)
		status := vmStatus(connConfig, one.IId.NameId)
		str = strings.ReplaceAll(str, "$$VMSTATUS$$", status)

		if cres.VMStatus(status) == cres.Running {
			str = strings.ReplaceAll(str, "$$VMCONTROL$$", `<span id="vmcontrol-`+one.IId.NameId+`">[<a href="javascript:vmControl('`+one.IId.NameId+`','suspend')">Suspend</a> / <a href="javascript:vmControl('`+one.IId.NameId+`','reboot')">Reboot</a>]</span>`)
		} else if cres.VMStatus(status) == cres.Suspended {
			str = strings.ReplaceAll(str, "$$VMCONTROL$$", `<span id="vmcontrol-`+one.IId.NameId+`">[<a href="javascript:vmControl('`+one.IId.NameId+`','resume')">Resume</a>]</span>`)
		} else {
			str = strings.ReplaceAll(str, "$$VMCONTROL$$", `[<span style="color:brown">vm control disabed</span>] <br> you can control when Running / Suspended. <br> try refresh page...`)
		}
		str = strings.ReplaceAll(str, "$$LASTSTARTTIME$$", one.StartTime.Format("2006.01.02 15:04:05 Mon"))

		// for Image & Spec
		str = strings.ReplaceAll(str, "$$IMAGETYPE$$", string(one.ImageType))
		str = strings.ReplaceAll(str, "$$IMAGE$$", one.ImageIId.NameId)
		str = strings.ReplaceAll(str, "$$SPEC$$", one.VMSpecName)

		// for VPC & Subnet
		str = strings.ReplaceAll(str, "$$VPC$$", one.VpcIID.NameId)
		str = strings.ReplaceAll(str, "$$SUBNET$$", one.SubnetIID.NameId)

		// for security rules info
		strSRList := ""
		for _, one := range one.SecurityGroupIIds {
/* mask for list performance
			// client logging
			strSRList += genLoggingOneGETURL(connConfig, "securitygroup", one.NameId)

			resBody, err := getResource_with_Connection_JsonByte(connConfig, "securitygroup", one.NameId)
			if err != nil {
				cblog.Error(err)
				break
			}
			// client logging
			strSRList += genLoggingResult(string(resBody[:len(resBody)-1]))

			var secInfo cres.SecurityInfo
			json.Unmarshal(resBody, &secInfo)

			strSRList += "["
			if secInfo.SecurityRules != nil {
				for _, secRuleInfo := range *secInfo.SecurityRules {
					strSRList += "{FromPort:" + secRuleInfo.FromPort + ", "
					strSRList += "ToPort:" + secRuleInfo.ToPort + ", "
					strSRList += "IPProtocol:" + secRuleInfo.IPProtocol + ", "
					strSRList += "Direction:" + secRuleInfo.Direction + ", "
					strSRList += "CIDR:" + secRuleInfo.CIDR
					strSRList += "},<br>"
				}
			}
			strSRList += "]"
*/

			strSRList += one.NameId + "," 
		}
		strSRList = strings.TrimSuffix(strSRList, ",")
		str = strings.ReplaceAll(str, "$$SECURITYGROUP$$", strSRList)

		// for Network Interface & PublicIP & PrivateIP
		str = strings.ReplaceAll(str, "$$NETWORKINTERFACE$$", one.NetworkInterface)
		str = strings.ReplaceAll(str, "$$PUBLICIP$$", one.PublicIP)
		str = strings.ReplaceAll(str, "$$PRIVATEIP$$", one.PrivateIP)

		// for Public DNS & Private DNS
		str = strings.ReplaceAll(str, "$$PUBLICDNS$$", one.PublicDNS)
		str = strings.ReplaceAll(str, "$$PRIVATEDNS$$", one.PrivateDNS)

		// for Root Disk & Data Disk
		str = strings.ReplaceAll(str, "$$ROOTDISK$$", "&nbsp;* " + one.RootDeviceName + 
			" (" + one.RootDiskType + ":" + one.RootDiskSize + "GB)" )

		dataDiskList := ""
		if len(one.DataDiskIIDs) > 0 {
			dataDiskList = "<br>&nbsp;&nbsp;&nbsp;------ Data Disk ------<br>"
		}else {
			dataDiskList = "<br>&nbsp;&nbsp;&nbsp;------ No Data Disk ------<br><br>"
		}

		for _, disk := range one.DataDiskIIDs {
			diskInfo := diskInfo(connConfig, disk.NameId)
			dataDiskList += "&nbsp;* " + disk.NameId + "(" + diskInfo.DiskType + ":" + diskInfo.DiskSize + "GB)"
			dataDiskList += "<br>"
		}
		str = strings.ReplaceAll(str, "$$DATADISK$$", dataDiskList)

		// for SSH AccessPoint & Access Key & Access User
		str = strings.ReplaceAll(str, "$$SSHACCESSPOINT$$", "<mark>" + one.SSHAccessPoint + "</mark>")
		str = strings.ReplaceAll(str, "$$ACCESSKEY$$", "<mark>" + one.KeyPairIId.NameId + "</mark>")
		str = strings.ReplaceAll(str, "$$ACCESSUSER$$", "<mark>" + one.VMUserId + "</mark>")

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

func genLoggingOneGETURL(connConfig string, rsType string, name string) string {
        /* return example
        <script type="text/javascript">
                parent.frames["log_frame"].Log("curl -sX GET http://localhost:1024/spider/vpc/vpc-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "aws-ohio-config"}'  ");
        </script>
        */

        url := "http://" + "localhost" + cr.ServerPort + "/spider/" + rsType + "/" + name + " -H 'Content-Type: application/json' -d '{\\\"ConnectionName\\\": \\\"" + connConfig  + "\\\"}'"
        htmlStr := `
                <script type="text/javascript">
                `
        htmlStr += `    parent.frames["log_frame"].Log("curl -sX GET ` +  url + `");`
        htmlStr += `
                </script>
                `
        return htmlStr
}

// make the string of javascript function
func makeVMControlFunc_js() string {
	//curl -sX PUT http://localhost:1024/spider/controlvm/vm-01?action=suspend -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}'

	strFunc := `
                function vmControl(vmName, action) {
                        var connConfig = parent.frames["top_frame"].document.getElementById("connConfig").innerHTML;

			document.getElementById("vmcontrol-" + vmName).innerHTML = '<span style="color:red">Waiting...</span>';
			setTimeout(function(){
				var xhr = new XMLHttpRequest();
				xhr.open("PUT", "$$SPIDER_SERVER$$/spider/controlvm/" + vmName + "?action=" + action, false);
				xhr.setRequestHeader('Content-Type', 'application/json');
				sendJson = '{ "ConnectionName": "' + connConfig + '"}'

				// client logging
				parent.frames["log_frame"].Log("PUT> " + "$$SPIDER_SERVER$$/spider/controlvm/" + vmName + "?action=" + action + " -H 'Content-Type: application/json' -d '" + sendJson + "'");

				xhr.send(sendJson);

				// client logging
				parent.frames["log_frame"].Log("   ==> " + xhr.response);

				location.reload();
			}, 10);
                }
        `
	strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://"+cr.ServiceIPorName+cr.ServicePort) // cr.ServicePort = ":1024"
	return strFunc
}

// make the string of javascript function
func makePostVMFunc_js() string {

	// curl -sX POST http://localhost:1024/spider/vm -H 'Content-Type: application/json'
	//  -d '{ "ConnectionName": "'${CONN_CONFIG}'",
	//  "ReqInfo": { "Name": "vm-01", "ImageName": "ami-00978328f54e31526", "VPCName": "vpc-01",
	//  "SubnetName": "subnet-01", "SecurityGroupNames": [ "sg-01" ], "VMSpecName": "t2.micro", "KeyPairName": "keypair-01"} }'

	strFunc := `
                function postVM() {
                        var connConfig = parent.frames["top_frame"].document.getElementById("connConfig").innerHTML;
                        var textboxes = document.getElementsByName('text_box');
                        sendJson = '{ "ConnectionName" : "' + connConfig + '", "ReqInfo" : { "Name" : "$$VMNAME$$", "ImageType" : "$$IMAGETYPE$$",\
                                "ImageName" : "$$IMAGE$$", "VMSpecName" : "$$SPEC$$", "VPCName" : "$$VPC$$", "SubnetName" : "$$SUBNET$$", \
                                "SecurityGroupNames" : $$SECURITYGROUP$$, "DataDiskNames" : [$$DATADISK$$], "KeyPairName" : "$$ACCESSKEY$$", \
                                "VMUserId" : "$$ACCESSUSER$$", "VMUserPasswd" : "$$ACCESSPASSWD$$" }}'

                        for (var i = 0; i < textboxes.length; i++) { // @todo make parallel executions
                                switch (textboxes[i].id) {
                                        case "1":
                                                sendJson = sendJson.replace("$$VMNAME$$", textboxes[i].value);
                                                break;
                                        case "22":
                                                sendJson = sendJson.replace("$$IMAGETYPE$$", textboxes[i].value);
                                                break;

                                        case "3":
                                        case "33":
                                        	if (textboxes[i].hidden==false) {
                                                	sendJson = sendJson.replace("$$IMAGE$$", textboxes[i].value);
                                                }
                                                break;
                                        case "4":
                                                sendJson = sendJson.replace("$$SPEC$$", textboxes[i].value);
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
                                        case "10":
						diskList = getSelectDisk(textboxes[i])
                                                sendJson = sendJson.replace("$$DATADISK$$", diskList);
                                                break;
                                        case "11":
                                                sendJson = sendJson.replace("$$ACCESSKEY$$", textboxes[i].value);
                                                break;
                                        case "12":
                                                sendJson = sendJson.replace("$$ACCESSUSER$$", textboxes[i].value);
                                                break;
                                        case "13":
                                                sendJson = sendJson.replace("$$ACCESSPASSWD$$", textboxes[i].value);
                                                break;

                                        default:
                                                break;
                                }
                        }
                        var xhr = new XMLHttpRequest();
                        xhr.open("POST", "$$SPIDER_SERVER$$/spider/vm", false);
                        xhr.setRequestHeader('Content-Type', 'application/json');

			// client logging
			parent.frames["log_frame"].Log("curl -sX POST " + "$$SPIDER_SERVER$$/spider/vm -H 'Content-Type: application/json' -d '" + sendJson + "'");

                        xhr.send(sendJson);

			// client logging
			parent.frames["log_frame"].Log("   ==> " + xhr.response);

			location.reload();
                }

		function getSelectDisk(select) {
			  if (select.tagName != 'SELECT') {
				  return ""
			  }
			  var result = [];
			  var options = select && select.options;
			  var opt;

			  for (var i=0, iLen=options.length; i<iLen; i++) {
			    opt = options[i];

			    if (opt.selected) {
			      result.push(opt.value || opt.text);
			    }
			  }

			  if (result.length < 1) {
			    return ""
			  }

			  diskList = ""
			  for ( var j=0, jLen=result.length; j<jLen; j++) {
				if (j==0) {
					diskList += '"' + result[j] + '"'
				} else {
					diskList += ', "' + result[j] + '"'
				}
			  }
			  return diskList;
		}

        `
	strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://"+cr.ServiceIPorName+cr.ServicePort) // cr.ServicePort = ":1024"
	return strFunc
}

// make the string of javascript function
func makeDeleteVMFunc_js() string {
	// curl -sX DELETE http://localhost:1024/spider/vm/vm-01 -H 'Content-Type: application/json' -d '{ "ConnectionName": "'${CONN_CONFIG}'"}'

	strFunc := `
                function deleteVM() {
                        var connConfig = parent.frames["top_frame"].document.getElementById("connConfig").innerHTML;
                        var checkboxes = document.getElementsByName('check_box');
                        for (var i = 0; i < checkboxes.length; i++) { // @todo make parallel executions
                                if (checkboxes[i].checked) {
                                        var xhr = new XMLHttpRequest();
                                        xhr.open("DELETE", "$$SPIDER_SERVER$$/spider/vm/" + checkboxes[i].value, false);
                                        xhr.setRequestHeader('Content-Type', 'application/json');
					sendJson = '{ "ConnectionName": "' + connConfig + '"}'

					// client logging
					parent.frames["log_frame"].Log("curl -sX DELETE " + "$$SPIDER_SERVER$$/spider/vm/" + checkboxes[i].value + " -H 'Content-Type: application/json' -d '" + sendJson + "'");

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

func VM(c echo.Context) error {
	cblog.Info("call VM()")

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
	htmlStr += makePostVMFunc_js()
	htmlStr += makeDeleteVMFunc_js()
	htmlStr += makeVMControlFunc_js()
	htmlStr += makeOnchangeImageTypeFunc_js()
	htmlStr += makePostSnapshotVMFunc_js()

	htmlStr += `
                    </script>
                </head>

                <body>
                    <table border="0" bordercolordark="#F8F8FF" cellpadding="0" cellspacing="1" bgcolor="#FFFFFF"  style="font-size:small;">
                `

	// (2) make Table Action TR
	// colspan, f5_href, delete_href, fontSize
	htmlStr += makeActionTR_html("11", "", "deleteVM()", "2")

	// (3) make Table Header TR
	nameWidthList := []NameWidth{
		{"VM Name / Control", "200"},
		{"VM Status / Last Start Time", "200"},
		{"ImageType / Image / VM Spec", "200"},
		{"VPC / Subnet / Security Group", "400"},
		{"NetworkInterface / PublicIP / PrivateIP", "400"},
		{"PublicDNS / PrivateDNS", "350"},
		{"RootDisk / DataDisk", "250"},
		{"SSH AccessPoint / Access Key / Access User", "200"},
		{"Additional Info", "300"},
	}
	htmlStr += makeTitleTRList_html("#DDDDDD", "2", nameWidthList, true)

	// (4) make TR list with info list
	// (4-1) get info list

	// client logging
	htmlStr += genLoggingGETURL(connConfig, "vm")

	resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "vm")
	if err != nil {
		cblog.Error(err)
		// client logging
                htmlStr += genLoggingResult(err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// client logging
	htmlStr += genLoggingResult(string(resBody[:len(resBody)-1]))

	var info struct {
		ResultList []*cres.VMInfo `json:"vm"`
	}
	json.Unmarshal(resBody, &info)

	// (4-2) make TR list with info list
	htmlStr += makeVMTRList_html(connConfig, "", "", "", info.ResultList)

	// (5) make input field and add
	// attach text box for add
	nameList := vpcList(connConfig)
	imageTypeList := []string{"PublicImage", "MyImage"}
	myImageList := myImageList(connConfig)
	keyNameList := keyPairList(connConfig)
	diskNameList := availableDataDiskList(connConfig)
	providerName, _ := getProviderName(connConfig)

	imageName := ""
	specName := ""
	subnetName := "subnet-01"
	sgName := `["sg-01"]`
	vmUser := "Administrator"  // Administrator for Windows GuserOS
	vmPasswd := "cloudbarista123^"  // default pw for Windows GuserOS
	switch providerName {
	case "AWS":
		imageName = "ami-00978328f54e31526"
		specName = "t2.micro"
	case "AZURE":
		imageName = "Canonical:UbuntuServer:18.04-LTS:latest"
		specName = "Standard_B1ls"
	case "GCP":
		imageName = "https://www.googleapis.com/compute/v1/projects/ubuntu-os-cloud/global/images/ubuntu-minimal-1804-bionic-v20191024"
		specName = "f1-micro"
	case "ALIBABA":
		imageName = "ubuntu_18_04_x64_20G_alibase_20220824.vhd"
		specName = "ecs.t5-lc1m2.small"
	case "TENCENT":
		imageName = "img-pi0ii46r"
		specName = "S3.MEDIUM2"
	case "IBM":
		imageName = "r014-a044e2f5-dfe1-416c-8990-5dc895352728"
		specName = "bx2-2x8"
	case "CLOUDIT":
		imageName = "ee441331-0872-49c3-886c-1873a6e32e09"
		specName = "small-2"
	case "OPENSTACK":
		imageName = "ubuntu18.04"
		specName = "DS-Demo"
	case "NCP":
		imageName = "SPSW0LINUX000130"
		specName = "SPSVRHICPUSSD002"
	case "KTCLOUD":
		imageName = "97ef0091-fdf7-44e9-be79-c99dc9b1a0ad"
		specName = "d3530ad2-462b-43ad-97d5-e1087b952b7d!87c0a6f6-c684-4fbe-a393-d8412bcf788d_disk100GB"
	case "NHNCLOUD":
		imageName = "5396655e-166a-4875-80d2-ed8613aa054f"
		specName = "m2.c4m8"
	case "DOCKER":
		imageName = "nginx:latest"
		specName = "NA"
		subnetName = "NA"
		sgName = `["NA"]`
	case "MOCK":
		imageName = "mock-vmimage-01"
		subnetName = "subnet-01"
		sgName = `["sg-01"]`
		specName = "mock-vmspec-01"
	case "CLOUDTWIN":
		imageName = "ubuntu18.04-sshd-systemd"
		subnetName = "subnet-01"
	default:
		imageName = "ami-00978328f54e31526"
		specName = "t2.micro"
	}

	// white color: #FFFFFF
	htmlStr += `
                        <tr bgcolor="#FFFFFF" align="center" height="30">
                            <td bgcolor="#FFEFBA">
                                    <font size=2>&nbsp;create:&nbsp;</font>
                            </td>
                            <td style="vertical-align:top">
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="1" value="vm-01">
                            </td>
                            <td style="vertical-align:top">
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="2" disabled value="N/A">
                            </td>
                            <td style="vertical-align:top">
                            `
	// Select format of VPC  name=text_box, id=5
	htmlStr += makeSelect_html("onchangeImageType", imageTypeList, "22")
	htmlStr += "<br>"

	htmlStr += makeMyImageSelect_html("", myImageList, "33")

	htmlStr += `                            
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="3" value="$$IMAGENAME$$">
			        <br>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="4" value="$$SPECNAME$$">
                            </td>
                            <td style="vertical-align:top">
			    `
	// Select format of VPC  name=text_box, id=5
	htmlStr += makeSelect_html("", nameList, "5")

	htmlStr += `

				<br>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="6" value="$$SUBNETNAME$$">
				<br>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="7" value=$$SGNAME$$>
                            </td>
                            <td style="vertical-align:top">
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="8" disabled value="N/A">
                            </td>
                            <td style="vertical-align:top">
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="9" disabled value="N/A">
                            </td>
                            <td style="vertical-align:top">
                            `
        // Select format of Data  name=text_box, id=10
        htmlStr += makeDataDiskSelect_html("", diskNameList, "10")

        htmlStr += `
                            </td>
                            <td style="vertical-align:top">
			    `
	// Select format of KeyPair  name=text_box, id=11
	htmlStr += makeKeyPairSelect_html("", keyNameList, "11")

	htmlStr += `
				<br>
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="12" value="$$VMUSER$$" disabled>
				<br>
                                <input style="font-size:12px;text-align:center;" type="password" name="text_box" id="13" value="$$VMPASSWD$$">
                            </td>
                            <td style="vertical-align:top">
                                <input style="font-size:12px;text-align:center;" type="text" name="text_box" id="14" disabled value="N/A">
                            </td>
                            <td>
                                <a href="javascript:postVM()">
                                    <font size=4><mark><b>+</b></mark></font>
                                </a>
                            </td>
                        </tr>
                `

	// set imageName & specName & vmUser
	htmlStr = strings.ReplaceAll(htmlStr, "$$IMAGENAME$$", imageName)
	htmlStr = strings.ReplaceAll(htmlStr, "$$SPECNAME$$", specName)
	htmlStr = strings.ReplaceAll(htmlStr, "$$SUBNETNAME$$", subnetName)
	htmlStr = strings.ReplaceAll(htmlStr, "$$SGNAME$$", sgName)
	htmlStr = strings.ReplaceAll(htmlStr, "$$VMUSER$$", vmUser)
	htmlStr = strings.ReplaceAll(htmlStr, "$$VMPASSWD$$", vmPasswd)

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

func makeKeyPairSelect_html(onchangeFunctionName string, strList []string, id string) string {

        strSelect := `<select name="text_box" id="` + id + `" onchange="` + onchangeFunctionName + `(this)">`
        for _, one := range strList {
		strSelect += `<option value="` + one + `">` + one + `</option>`
        }
	// add one more not to use Key but to use password
	// strSelect += `<option value=""</option>`

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

func makeMyImageSelect_html(onchangeFunctionName string, strList []string, id string) string {

	
	if len(strList) == 0 {
		publicImageStr := `<input style="font-size:12px;text-align:center;" type="text" name="text_box" id="` +
				id +`" disabled value="None" hidden>`
		return publicImageStr
	}
        strSelect := `<select style="width:120px;" name="text_box" id="` + id + `" onchange="` + onchangeFunctionName + `(this)" hidden>`
        for _, one := range strList {
		strSelect += `<option value="` + one + `">` + one + `</option>`
        }

        strSelect += `
                </select>
        `
        return strSelect
}

// make the string of javascript function
func makeOnchangeImageTypeFunc_js() string {
        strFunc := `
              function onchangeImageType(source) {
                var imageType = source.value
                if (imageType == 'MyImage') {
                	document.getElementById('3').hidden=true;
                	document.getElementById('33').hidden=false;
                } else {
                	document.getElementById('3').hidden=false;
                	document.getElementById('33').hidden=true;
                }
              }
        `
        return strFunc
}


// make the string of javascript function
func makePostSnapshotVMFunc_js() string {

        //curl -sX POST http://localhost:1024/spider/myimage -H 'Content-Type: application/json'
        //      -d '{ "ConnectionName": "'${CONN_CONFIG}'", "ReqInfo": {
                                //      "Name": "spider-myimage-01",
                                //      "SourceVM": "vm-01"
                        //      } }'

        strFunc := `
                function postSnapshotVM(i, vmName) {
                        var connConfig = parent.frames["top_frame"].document.getElementById("connConfig").innerHTML;

                        var myImages = document.getElementsByName('myimage-name');
                        var idx = parseInt(i)-1;                        
                        var myImageName = myImages[idx].value;

            		sendJson = '{ "ConnectionName" : "' + connConfig + '", "ReqInfo" : { "Name" : "$$MYIMAGENAME$$", "SourceVM" : "$$SOURCEVM$$"}}'
            		sendJson = sendJson.replace("$$MYIMAGENAME$$", myImageName);
            		sendJson = sendJson.replace("$$SOURCEVM$$", vmName);

                        var xhr = new XMLHttpRequest();
                        xhr.open("POST", "$$SPIDER_SERVER$$/spider/myimage", false);
                        xhr.setRequestHeader('Content-Type', 'application/json');

                        // client logging
                        parent.frames["log_frame"].Log("curl -sX POST " + 
                                "$$SPIDER_SERVER$$/spider/myimage -H 'Content-Type: application/json' -d '" + sendJson + "'");

                        xhr.send(sendJson);

                        // client logging
                        parent.frames["log_frame"].Log("   ==> " + xhr.response);
                        var jsonVal = JSON.parse(xhr.response)


            location.reload();
                }
        `
        strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://"+cr.ServiceIPorName+cr.ServicePort) // cr.ServicePort = ":1024"
        return strFunc
}
