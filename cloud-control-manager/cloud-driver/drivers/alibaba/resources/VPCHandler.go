// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by zephy@mz.co.kr, 2019.09.

//VPCHandler는 서브넷을 처리하는 핸들러임.
//VPC & Subnet 처리 (AlibabaCloud's Subnet --> VSwitch 임)
//Ver2 - <CB-Virtual Network> 개발 방안에 맞게 VPC기능은 외부에 숨기고 Subnet을 Main으로 함.

package resources

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	//"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	/*
		"github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
		"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
		idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
		irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
		"github.com/davecgh/go-spew/spew"
	*/)

type AlibabaVPCHandler struct {
	Region idrv.RegionInfo
	Client *vpc.Client
}

func (VPCHandler *AlibabaVPCHandler) CreateVPC(vpcReqInfo irs.VPCReqInfo) (irs.VPCInfo, error) {
	cblogger.Info(vpcReqInfo)

	return irs.VPCInfo{}, nil
}

func (VPCHandler *AlibabaVPCHandler) ListVPC() ([]*irs.VPCInfo, error) {
	cblogger.Debug("Start")

	return nil, nil
}

func (VPCHandler *AlibabaVPCHandler) GetVPC(vpcIID irs.IID) (irs.VPCInfo, error) {
	cblogger.Info("VPC IID : ", vpcIID.SystemId)
	return irs.VPCInfo{}, nil

}

func (VPCHandler *AlibabaVPCHandler) DeleteVPC(vpcIID irs.IID) (bool, error) {
	cblogger.Infof("Delete VPC : [%s]", vpcIID.SystemId)

	return true, nil
}
