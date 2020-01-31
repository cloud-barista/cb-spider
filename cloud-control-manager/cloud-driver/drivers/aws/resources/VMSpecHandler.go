package resources

import (
	"fmt"

	//sdk2 "github.com/aws/aws-sdk-go-v2"
	"github.com/aws/aws-sdk-go/service/ec2"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

//https://docs.aws.amazon.com/sdk-for-go/api/service/ec2/#EC2.DescribeInstanceTypes
type AwsVmSpecHandler struct {
	Region idrv.RegionInfo
	Client *ec2.EC2
}

func (vmSpecHandler *AwsVmSpecHandler) ListVMSpec(Region string) ([]*irs.VMSpecInfo, error) {

	cblogger.Infof("Start ListVMSpec(%s)", Region)

	var vMSpecInfoList []*irs.VMSpecInfo

	// Example sending a request using the DescribeInstanceTypesRequest method.
	req, resp := vmSpecHandler.Client.DescribeInstanceTypesRequest(nil)

	err := req.Send()
	if err == nil { // resp is now filled
		fmt.Println(resp)
	}

	input := &ec2.DescribeKeyPairsInput{
		KeyNames: []*string{
			nil,
		},
	}

	//  Returns a list of key pairs
	result, err := vmSpecHandler.Client.DescribeKeyPairs(input)
	cblogger.Info(result)
	if err != nil {
		cblogger.Errorf("Unable to get key pairs, %v", err)
		return vMSpecInfoList, err
	}

	/*
		//cblogger.Debugf("Key Pairs:")
		for _, pair := range result.KeyPairs {
				//cblogger.Debugf("%s: %s\n", *pair.KeyName, *pair.KeyFingerprint)
				//keyPairInfo := new(irs.KeyPairInfo)
				//keyPairInfo.Name = *pair.KeyName
				//keyPairInfo.Fingerprint = *pair.KeyFingerprint
			//keyPairInfo := ExtractKeyPairDescribeInfo(pair)
			//keyPairList = append(keyPairList, &keyPairInfo)
		}

		cblogger.Info(keyPairList)
		spew.Dump(keyPairList)
		return keyPairList, nil
	*/

	return vMSpecInfoList, nil
}

func (vmSpecHandler *AwsVmSpecHandler) GetVMSpec(Region string, Name string) (irs.VMSpecInfo, error) {

	return irs.VMSpecInfo{}, nil
}

func (vmSpecHandler *AwsVmSpecHandler) ListOrgVMSpec(Region string) (string, error) {
	return "", nil
}

func (vmSpecHandler *AwsVmSpecHandler) GetOrgVMSpec(Region string, Name string) (string, error) {
	return "", nil
}
