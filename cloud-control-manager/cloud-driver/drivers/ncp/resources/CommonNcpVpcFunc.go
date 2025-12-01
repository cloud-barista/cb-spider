// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// NCP VPC Connection Driver
//
// by ETRI, 2020.10.

package resources

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	// "strconv"
	"math/rand"

	cblog "github.com/cloud-barista/cb-log"
	"github.com/sirupsen/logrus"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
)

const (
	usageTypeCodeGen   = "GEN"
	usageTypeCodeLoadb = "LOADB"
	usageTypeCodeBm    = "BM"
	usageTypeCodeNatgw = "NATGW"

	subnetTypeCodePublic  = "PUBLIC"
	subnetTypeCodePrivate = "PRIVATE"

	subnetStatusInitiated   = "INIT"
	subnetStatusCreating    = "CREATING"
	subnetStatusRun         = "RUN"
	subnetStatusTerminating = "TERMTING"
)

var once sync.Once
var cblogger *logrus.Logger
var calllogger *logrus.Logger

func InitLog() {
	once.Do(func() {
		// cblog is a global variable.
		cblogger = cblog.GetLogger("CB-SPIDER")
		calllogger = call.GetLogger("HISCALL")
	})
}

// Convert Cloud Object to JSON String type
func ConvertJsonString(v interface{}) (string, error) {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		newErr := fmt.Errorf("Failed to Convert Json to String. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		return "", newErr
	}
	jsonString := string(jsonBytes)
	return jsonString, nil
}

// int32 to string 변환 : String(), int64 to string 변환 : strconv.Itoa()
func String(n int32) string {
	buf := [11]byte{}
	pos := len(buf)
	i := int64(n)
	signed := i < 0
	if signed {
		i = -i
	}
	for {
		pos--
		buf[pos], i = '0'+byte(i%10), i/10
		if i == 0 {
			if signed {
				pos--
				buf[pos] = '-'
			}
			return string(buf[pos:])
		}
	}
}

func LoggingError(hiscallInfo call.CLOUDLOGSCHEMA, err error) {
	if cblogger == nil || calllogger == nil {
		InitLog()
	}
	cblogger.Error(err.Error())
	hiscallInfo.ErrorMSG = err.Error()
	calllogger.Error(call.String(hiscallInfo))
}

func LoggingInfo(hiscallInfo call.CLOUDLOGSCHEMA, start time.Time) {
	if calllogger == nil {
		InitLog()
	}
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))
}

func GetCallLogScheme(zoneInfo string, resourceType call.RES_TYPE, resourceName string, apiName string) call.CLOUDLOGSCHEMA {
	cblogger.Info(fmt.Sprintf("Call %s %s", call.NCP, apiName))

	return call.CLOUDLOGSCHEMA{
		CloudOS:      call.NCP,
		RegionZone:   zoneInfo,
		ResourceType: resourceType,
		ResourceName: resourceName,
		CloudOSAPI:   apiName,
	}
}

func logAndReturnError(callLogInfo call.CLOUDLOGSCHEMA, givenErrString string, v interface{}) error {
	newErr := fmt.Errorf(givenErrString+" %v", v)
	cblogger.Error(newErr.Error())
	LoggingError(callLogInfo, newErr)
	return newErr
}

func randSeq(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	rand.Seed(time.Now().UnixNano())

	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func convertTimeFormat(inputTime string) (time.Time, error) {
	// Parse the input time using the given layout
	layout := "2006-01-02T15:04:05-0700"
	parsedTime, err := time.Parse(layout, inputTime)
	if err != nil {
		newErr := fmt.Errorf("Failed to Parse the Input Time Format!!")
		cblogger.Error(newErr.Error())
		return time.Time{}, newErr
	}

	return parsedTime, nil
}

// Converts net.IP to uint32
func ipToUint32(ip net.IP) uint32 {
	ip = ip.To4()
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

// Converts uint32 to net.IP
func uint32ToIP(n uint32) net.IP {
	return net.IPv4(byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
}

// Returns true if two CIDRs overlap
func checkOverlap(a, b *net.IPNet) bool {
	return a.Contains(b.IP) || b.Contains(a.IP)
}

// GetReverseSubnetCidrs returns up to maxCount available subnets in reverse order (/prefix) within vpcCIDR
func GetReverseSubnetCidrs(vpcCIDR string, existingSubnets []string, subnetPrefixLength int, maxCount int) ([]string, error) {
	// Parse VPC CIDR
	_, vpcNet, err := net.ParseCIDR(vpcCIDR)
	if err != nil {
		return nil, fmt.Errorf("invalid VPC CIDR: %v", err)
	}

	vpcPrefix, _ := vpcNet.Mask.Size()
	if vpcPrefix > subnetPrefixLength {
		return nil, fmt.Errorf("VPC CIDR too small to accommodate /%d subnets", subnetPrefixLength)
	}
	if subnetPrefixLength > 32 {
		return nil, fmt.Errorf("Invalid subnet prefix length: %d", subnetPrefixLength)
	}

	// Parse existing subnets
	var existing []*net.IPNet
	for _, cidr := range existingSubnets {
		_, sn, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("invalid existing subnet: %v", err)
		}
		existing = append(existing, sn)
	}

	// Convert to IP space
	subnetSize := uint32(1 << (32 - subnetPrefixLength))
	vpcStart := ipToUint32(vpcNet.IP)
	vpcEnd := vpcStart + uint32(1<<(32-vpcPrefix)) - 1

	// Align last subnet to boundary
	lastSubnetStart := (vpcEnd + 1 - subnetSize) & ^(subnetSize - 1)

	var results []string
	for i := uint32(0); ; i++ {
		addr := lastSubnetStart - i*subnetSize

		// Prevent unsigned underflow
		if addr < vpcStart {
			break
		}

		ip := uint32ToIP(addr)
		subnet := &net.IPNet{
			IP:   ip,
			Mask: net.CIDRMask(subnetPrefixLength, 32),
		}

		// Check for overlap
		conflict := false
		for _, exist := range existing {
			if checkOverlap(subnet, exist) {
				conflict = true
				break
			}
		}

		if !conflict {
			results = append(results, subnet.String())
			if len(results) == maxCount {
				break
			}
		}
	}

	return results, nil
}
