// IID(Integrated ID) Manager of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.03.

package iidmanager

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/rs/xid"

	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager"
	ccim "github.com/cloud-barista/cb-spider/cloud-info-manager/connection-config-info-manager"
	"github.com/cloud-barista/cb-store/config"
	"github.com/sirupsen/logrus"
)

var cblog *logrus.Logger

func init() {
	cblog = config.Cblogger
}

//====================================================================

// ====================================================================
type IIDInfo struct {
	ConnectionName string        // ex) "aws-seoul-config"
	ResourceType   string        // ex) "VM"
	IId            resources.IID // ex) {NameId, SystemId} = {"powerkim_vm_01", "i-0bc7123b7e5cbf79d"}
}

//====================================================================

//----------------

// generate Spider UUID(SP-XID)
func New(cloudConnectName string, rsType string, uid string) (string, error) {
	cccInfo, err := ccim.GetConnectionConfig(cloudConnectName)
	if err != nil {
		return "", err
	}

	// ref) https://github.com/cloud-barista/cb-spider/issues/655
	if cccInfo.ProviderName == "NCP" && rsType == "sg" {
		return uid, nil
	}

	// MaxLength = 12, lower, number, Cannot use '-'
	if cccInfo.ProviderName == "AZURE" && rsType == "nodegroup" {
		retUID := strings.ToLower(strings.ReplaceAll(uid, "-", ""))

		if len(retUID) > 12 {
			// #6 + #6 => #12
			retUID = uid[:6] + xid.New().String()[0:6]
		}
		return retUID, nil
	}

	// MaxLenth = 20, lower, number, '-'
	if cccInfo.ProviderName == "NHNCLOUD" && (rsType == "cluster" || rsType == "nodegroup") {
		retUID := strings.ToLower(uid)

		if len(retUID) > 20 {
			// #10 + #10 => #20
			retUID = uid[:10] + xid.New().String()[0:10]
		}
		return retUID, nil
	}

	// default length: 9 + 21 => 30 (NCP's ID Length, the shortest)
	//   ex) AWS maxLen(VMID)=255, #234 + #1 + #20 <== "{UID}-{XID}", {XID} = #20
	maxLength := 9

	rsMaxLength := getIdMaxLength(cccInfo.ProviderName, rsType)

	if rsMaxLength > 0 && rsMaxLength <= 21 {
		return "", fmt.Errorf("The Minimum ID Length must be greater than 21!")
	}

	if rsMaxLength > 21 {
		maxLength = rsMaxLength - 21
	}

	cookedUID := cookUID(uid, maxLength)
	// cblog.Info("UID: " + uid + " => cookedUID: " + cookedUID)

	guid := xid.New()
	spXID := cookedUID + "-" + guid.String()
	// cblog.Info("SP-XID: " + spXID)

	return convertDashOrUnderScore(cccInfo.ProviderName, spXID)
}

func getIdMaxLength(providerName string, rsType string) int {
	// get Provider's Meta Info
	cloudOSMetaInfo, err := cim.GetCloudOSMetaInfo(providerName)
	if err != nil {
		cblog.Error(err)
		return 0
	}

	if len(cloudOSMetaInfo.IdMaxLength) <= 1 {
		return 0
	}

	/*----- ref) cloud-driver-libs/cloudos_meta.yaml
	  # idmaxlength: VPC / Subnet / SecurityGroup / KeyPair / VM / Disk / NLB / MyImage /Cluster
	    idmaxlength: 255 / 256 / 255 / 255 / 255 / 255 / 255 / 255 / 255
	-----*/
	idx := getIDXNumber(rsType)
	if idx == -1 {
		return 0
	}

	// target CSP's rsType not defined in cloudos_meta.yaml
	if idx >= len(cloudOSMetaInfo.IdMaxLength) {
		return 0
	}

	strMaxLength := cloudOSMetaInfo.IdMaxLength[idx]
	maxLength, _ := strconv.Atoi(strMaxLength)

	return maxLength
}

func getIDXNumber(rsType string) int {
	switch rsType {
	case "vpc":
		return 0
	case "subnet":
		return 1
	case "sg":
		return 2
	case "keypair":
		return 3
	case "vm":
		return 4
	case "disk":
		return 5
	case "nlb":
		return 6
	case "myimage":
		return 7
	case "cluster":
		return 8
	case "nodegroup":
		return 9
	default:
		return -1
	}
}

func convertDashOrUnderScore(providerName string, spXID string) (string, error) {
	var convertedSpXID string
	// Tencent use '_'
	if providerName == "TENCENT" {
		convertedSpXID = strings.ReplaceAll(spXID, "-", "_")
	} else { // other CSP use '-'
		convertedSpXID = strings.ReplaceAll(spXID, "_", "-")
	}

	// AWS SecurityGroup: User can not use 'sg-*' format
	convertedSpXID = strings.ReplaceAll(convertedSpXID, "sg-", "sg")

	return convertedSpXID, nil
}

func cookUID(orgUID string, maxLength int) string {
	runes := []rune(orgUID)
	filteredUID := []byte{}
	for _, char := range runes {
		// (1) Max length is '9' or 4(TENCENT)
		if len(filteredUID) == maxLength { // max length: 9 or 4(TENCENT)
			break
		}
		var matched bool = false
		var err error
		// (2) Check the first character is a lowercase string
		if len(filteredUID) == 0 {
			matched, err = regexp.MatchString("[a-zA-Z]", string(char))
			// (3) Extract filteredUID([a-zA-Z0-9-_])
		} else {
			matched, err = regexp.MatchString("[a-zA-Z0-9-_]", string(char))
		}
		if err != nil {
			cblog.Error(err)
		}
		if matched {
			//fmt.Printf("%s matches\n", string(char))
			filteredUID = append(filteredUID, byte(char))
		}
	}

	// (4) Coverting UID into lowercase
	lowercaseUID := strings.ToLower(string(filteredUID))

	return lowercaseUID
}
