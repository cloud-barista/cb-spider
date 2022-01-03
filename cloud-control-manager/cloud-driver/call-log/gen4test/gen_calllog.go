// Call-Log: calling logger of Cloud & VM in CB-Spider
//           Referred to cb-log
//
//      * Cloud-Barista: https://github.com/cloud-barista
//      * CB-Spider: https://github.com/cloud-barista/cb-spider
//      * cb-log: https://github.com/cloud-barista/cb-log
//
// load and set config file
//
// ref) https://github.com/go-yaml/yaml/tree/v3
//      https://godoc.org/gopkg.in/yaml.v3
//
// by CB-Spider Team, 2020.09.

package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
)

func main() {
	callogger := call.GetLogger("HISCALL")

	info := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   "us-east1/us-east1-c",
		ResourceType: call.VPCSUBNET,
		ResourceName: "aws-vpc-01",
	}

	//for i:=0;i<10;i++ {
	for {
		setRandom(&info)
		start := call.Start()
		err := ListVPC() // just example for simple test
		info.ElapsedTime = call.Elapsed(start)
		if err != nil {
			info.ErrorMSG = err.Error()
		}
		callogger.Info(call.String(info))
	}
}

func ListVPC() error {
	//r = random()%len(1000)
	r := time.Duration(rand.Int63n(1000))
	time.Sleep(time.Millisecond * r)
	return nil
}

var cloudOSList = []call.CLOUD_OS{
	call.AWS,
	call.AZURE,
	call.GCP,
	call.ALIBABA,
	call.TENCENT,
	call.IBM,
	call.OPENSTACK,
	call.CLOUDIT,
	call.NCP,
	call.NCPVPC,
	call.KTCLOUD,
	call.NHNCLOUD,
	call.DOCKER,
	call.MOCK,
	call.CLOUDTWIN,
}

var resTypeList = []call.RES_TYPE{
	call.VMIMAGE,
	call.VMSPEC,
	call.VPCSUBNET,
	call.SECURITYGROUP,
	call.VMKEYPAIR,
	call.VM,
}

func setRandom(info *call.CLOUDLOGSCHEMA) {
	//r = random()%len(cloudOSList)
	r := rand.Intn(len(cloudOSList))
	info.CloudOS = cloudOSList[r]
	str := fmt.Sprintf("%s-region%d/zone-%d", strings.ToLower(string(cloudOSList[r])), r, r)
	info.RegionZone = str

	//r = random()%len(resTypeList)
	r = rand.Intn(len(resTypeList))
	info.ResourceType = resTypeList[r]
	str = fmt.Sprintf("%s-%d", strings.ToLower(string(resTypeList[r])), r)
	info.ResourceName = str
	info.CloudOSAPI = "List" + string(resTypeList[r]) + "()"
}
