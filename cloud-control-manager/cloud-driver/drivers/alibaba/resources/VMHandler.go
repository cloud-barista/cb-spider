// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// EC2 Hander (AWS SDK GO Version 1.16.26, Thanks AWS.)
//
// by zephy@mz.co.kr, 2019.09.

package resources

import (
	"errors"
	"fmt"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/sirupsen/logrus"
	/*
		"github.com/sirupsen/logrus"
		"reflect"
		"strings"
		"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
		"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
		cblog "github.com/cloud-barista/cb-log"
		idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
		irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
		"github.com/davecgh/go-spew/spew"
	*/)

type AlibabaVMHandler struct {
	Region idrv.RegionInfo
	Client *ecs.Client
}

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("ALIBABA VMHandler")
}

// func Connect(region string) *ecs.Client {
// 	// setup Region
// 	sess, err := session.NewSession(&aws.Config{
// 		Region: aws.String(region)},
// 	)

// 	if err != nil {
// 		fmt.Println("Could not create instance", err)
// 		return nil
// 	}

// 	// Create EC2 service client
// 	svc := ec2.New(sess)

// 	return svc
// }

// @Todo : SecurityGroupId 배열 처리 방안
// 1개의 VM만 생성되도록 수정 (MinCount / MaxCount 이용 안 함)
//키페어 이름(예:mcloud-barista)은 아래 URL에 나오는 목록 중 "키페어 이름"의 값을 적으면 됨.
//https://ap-northeast-2.console.aws.amazon.com/ec2/v2/home?region=ap-northeast-2#KeyPairs:sort=keyName
func (vmHandler *AlibabaVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (irs.VMInfo, error) {
	return irs.VMInfo{}, nil
	/*
		cblogger.Info(vmReqInfo)
		spew.Dump(vmReqInfo)

		imageID := vmReqInfo.ImageId
		instanceType := vmReqInfo.SpecId // "t2.micro"
		minCount := requests.NewInteger64(1)
		maxCount := requests.NewInteger64(1)
		keyName := vmReqInfo.KeyPairInfo.Name
		securityGroupID := vmReqInfo.SecurityInfo.Id // "sg-0df1c209ea1915e4b" - 미지정시 보안 그룹명이 "default"인 보안 그룹이 사용 됨.
		subnetID := vmReqInfo.VNetworkInfo.Id        // "subnet-cf9ccf83" - 미지정시 기본 VPC의 기본 서브넷이 임의로 이용되며 PublicIP가 할당 됨.
		baseName := vmReqInfo.Name                   //"mcloud-barista-VMHandlerTest"

		cblogger.Info("Create ECS Instance")

		// Specify the details of the instance that you want to create.
		runResult, err := vmHandler.Client.RunInstances(&ec2.RunInstancesInput{
			ImageId:      aws.String(imageID),
			InstanceType: aws.String(instanceType),
			MinCount:     minCount,
			MaxCount:     maxCount,
			KeyName:      aws.String(keyName),

			SecurityGroupIds: []*string{
				aws.String(securityGroupID), // set a security group.
			},

			SubnetId: aws.String(subnetID), // set a subnet.
		})
		if err != nil {
			cblogger.Errorf("Could not create instance", err)
			return irs.VMInfo{}, err
		}

		newVmId := *runResult.Instances[0].InstanceId
		cblogger.Info("Created instance ", newVmId)
		// Tag에 VM Name 설정
		_, errtag := vmHandler.Client.CreateTags(&ec2.CreateTagsInput{
			Resources: []*string{runResult.Instances[0].InstanceId},
			Tags: []*ec2.Tag{
				{
					Key:   aws.String("Name"),
					Value: aws.String(baseName),
				},
			},
		})
		if errtag != nil {
			cblogger.Error("Could not create tags for instance ", newVmId, errtag)
			return irs.VMInfo{}, errtag
		}

		//EC2에 EIP 할당
		cblogger.Infof("[%s] EC2에 [%s] IP 할당 시작", newVmId, vmReqInfo.PublicIPId)
		assocRes, errIp := vmHandler.AssociatePublicIP(vmReqInfo.PublicIPId, newVmId)
		if errIp != nil {
			cblogger.Errorf("Unable to associate IP address with %s, %v", newVmId, err)
			return irs.VMInfo{}, nil
		}

		cblogger.Infof("[%s] EC2에 Public IP 할당 결과 : ", newVmId, assocRes)

		//Public IP및 초신 정보 전달을 위해 부팅이 완료될 때까지 대기했다가 전달하는 것으로 변경 함.
		cblogger.Info("EC2 Running 상태 대기")
		WaitForRun(vmHandler.Client, newVmId)
		cblogger.Info("EC2 Running 상태 완료 : ", runResult.Instances[0].State.Name)

		//최신 정보 조회
		vmInfo := vmHandler.GetVM(newVmId)

		return vmInfo, nil
	*/
}

/*
//VM이 Running 상태일때까지 대기 함.
func WaitForRun(svc *ecs.Client, instanceID string) {
	cblogger.Infof("ECS ID : [%s]", instanceID)

	input := &ecs.DescribeInstancesInput{
		InstanceIds: []*string{
			aws.String(instanceID),
		},
	}
	err := svc.WaitUntilInstanceRunning(input)
	if err != nil {
		cblogger.Errorf("failed to wait until instances exist: %v", err)
	}
	cblogger.Info("=========WaitForRun() 종료")
}
*/

func (vmHandler *AlibabaVMHandler) ResumeVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Infof("vmID : [%s]", vmIID.SystemId)
	return irs.VMStatus("Failed"), nil
	/*
		input := &ec2.StartInstancesInput{
			InstanceIds: []*string{
				aws.String(vmID),
			},
			DryRun: aws.Bool(true),
		}
		result, err := vmHandler.Client.StartInstances(input)
		awsErr, ok := err.(awserr.Error)

		if ok && awsErr.Code() == "DryRunOperation" {
			// Let's now set dry run to be false. This will allow us to start the instances
			input.DryRun = aws.Bool(false)
			result, err = vmHandler.Client.StartInstances(input)
			if err != nil {
				//fmt.Println("Error", err)
				cblogger.Error(err)
			} else {
				//fmt.Println("Success", result.StartingInstances)
				cblogger.Info("Success", result.StartingInstances)
			}
		} else { // This could be due to a lack of permissions
			//fmt.Println("Error", err)
			cblogger.Error(err)
		}
	*/
}

func (vmHandler *AlibabaVMHandler) SuspendVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Infof("vmID : [%s]", vmIID.SystemId)
	return irs.VMStatus("Failed"), nil
	/*
		input := &ec2.StopInstancesInput{
			InstanceIds: []*string{
				aws.String(vmID),
			},
			DryRun: aws.Bool(true),
		}
		result, err := vmHandler.Client.StopInstances(input)
		awsErr, ok := err.(awserr.Error)
		if ok && awsErr.Code() == "DryRunOperation" {
			input.DryRun = aws.Bool(false)
			result, err = vmHandler.Client.StopInstances(input)
			if err != nil {
				cblogger.Error(err)
			} else {
				cblogger.Info("Success", result.StoppingInstances)
			}
		} else {
			cblogger.Error("Error", err)
		}
	*/
}

func (vmHandler *AlibabaVMHandler) RebootVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Infof("vmID : [%s]", vmIID.SystemId)
	/*
		input := &ec2.RebootInstancesInput{
			InstanceIds: []*string{
				aws.String(vmID),
			},
			DryRun: aws.Bool(true),
		}
		result, err := vmHandler.Client.RebootInstances(input)
		cblogger.Info("result 값 : ", result)
		cblogger.Info("err 값 : ", err)

		awsErr, ok := err.(awserr.Error)
		cblogger.Info("ok 값 : ", ok)
		cblogger.Info("awsErr 값 : ", awsErr)
		if ok && awsErr.Code() == "DryRunOperation" {
			cblogger.Info("Reboot 권한 있음 - awsErr.Code() : ", awsErr.Code())

			//DryRun 권한 해제 후 리부팅을 요청 함.
			cblogger.Info("DryRun 권한 해제 후 리부팅을 요청 함.")
			input.DryRun = aws.Bool(false)
			result, err = vmHandler.Client.RebootInstances(input)
			cblogger.Info("result 값 : ", result)
			cblogger.Info("err 값 : ", err)
			if err != nil {
				cblogger.Error("Error", err)
			} else {
				cblogger.Info("Success", result)
			}
		} else { // This could be due to a lack of permissions
			cblogger.Info("리부팅 권한이 없는 것같음.")
			cblogger.Error("Error", err)
		}
	*/
	return irs.VMStatus("Failed"), nil
}

func (vmHandler *AlibabaVMHandler) TerminateVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Infof("vmID : [%s]", vmIID.SystemId)
	return irs.VMStatus("Failed"), nil
	/*
		input := &ec2.TerminateInstancesInput{
			//InstanceIds: instanceIds,
			InstanceIds: []*string{
				aws.String(vmID),
			},
		}

		_, err := vmHandler.Client.TerminateInstances(input)
		if err != nil {
			cblogger.Error("Could not termiate instances", err)
		} else {
			cblogger.Info("Success")
		}
	*/
}

func (vmHandler *AlibabaVMHandler) GetVM(vmIID irs.IID) (irs.VMInfo, error) {
	cblogger.Infof("vmID : [%s]", vmIID.SystemId)

	request := ecs.CreateDescribeInstancesRequest()
	request.Scheme = "https"
	request.InstanceIds = "[\"" + vmIID.SystemId + "\"]"

	response, err := vmHandler.Client.DescribeInstances(request)
	if err != nil {
		fmt.Print(err.Error())
	}

	if response.TotalCount < 1 {
		return irs.VMInfo{}, errors.New("Notfound: '" + vmIID.SystemId + "' VM Not found")
	}

	/*
		vmInfo := irs.VMInfo{}
		for _, curInstance := range response.Instances.Instance {
			vmInfo := ExtractDescribeInstances(result.Reservations[0])
			vmInfo = vmHandler.ExtractDescribeInstances(curInstance)
		}
	*/

	//	vmInfo := vmHandler.ExtractDescribeInstances(response.Instances.Instance[0])
	vmInfo := vmHandler.ExtractDescribeInstances(&response.Instances.Instance[0])
	cblogger.Info("vmInfo", vmInfo)
	return vmInfo, nil
}

//@TODO : 2020-03-26 Ali클라우드 API 구조가 바뀐 것 같아서 임시로 변경해 놓음.
//func (vmHandler *AlibabaVMHandler) ExtractDescribeInstances() irs.VMInfo {
func (vmHandler *AlibabaVMHandler) ExtractDescribeInstances(instancInfo *ecs.Instance) irs.VMInfo {
	cblogger.Info(instancInfo)

	//time.Parse(layout, str)
	vmInfo := irs.VMInfo{
		IId:        irs.IID{NameId: instancInfo.InstanceName, SystemId: instancInfo.InstanceId},
		ImageIId:   irs.IID{SystemId: instancInfo.ImageId},
		VMSpecName: instancInfo.InstanceType,
		KeyPairIId: irs.IID{SystemId: instancInfo.KeyPairName},
		//StartTime:  instancInfo.StartTime,

		//Region            RegionInfo //  ex) {us-east1, us-east1-c} or {ap-northeast-2}
		//VpcIID            irs.IID{SystemId: instancInfo,
		//SubnetIID         IID   // AWS, ex) subnet-8c4a53e4
		//SecurityGroupIIds []IID // AWS, ex) sg-0b7452563e1121bb6
		//NetworkInterface string // ex) eth0
		//PublicIP
		//PublicDNS
		//PrivateIP
		//PrivateDNS

		//VMBootDisk  string // ex) /dev/sda1
		//VMBlockDisk string // ex)

		KeyValueList: []irs.KeyValue{{Key: "", Value: ""}},
	}

	if instancInfo.StartTime != "" {
		cblogger.Infof("Convert StartTime string [%s] to time.time", instancInfo.StartTime)

		layout := "2017-12-10T04:04Z"
		t, _ := time.Parse(layout, instancInfo.StartTime)
		vmInfo.StartTime = t
	}

	return vmInfo
}

func (vmHandler *AlibabaVMHandler) ListVM() ([]*irs.VMInfo, error) {
	cblogger.Infof("Start")
	return nil, nil
	/*
		var vmInfoList []*irs.VMInfo

		input := &ec2.DescribeInstancesInput{
			InstanceIds: []*string{
				nil,
			},
		}

		result, err := vmHandler.Client.DescribeInstances(input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				default:
					cblogger.Error(aerr.Error())
					return vmInfoList
				}
			} else {
				// Print the error, cast err to awserr.Error to get the Code and Message from an error.
				cblogger.Error(err.Error())
				return vmInfoList
			}
			return vmInfoList
		}

		cblogger.Info("Success")

		for _, i := range result.Reservations {
			for _, vm := range i.Instances {
				cblogger.Info("[%s] EC2 정보 조회", *vm.InstanceId)
				vmInfo := vmHandler.GetVM(*vm.InstanceId)
				vmInfoList = append(vmInfoList, &vmInfo)
			}
		}

		return vmInfoList
	*/
}

//SHUTTING-DOWN / TERMINATED
func (vmHandler *AlibabaVMHandler) GetVMStatus(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Infof("vmID : [%s]", vmIID.SystemId)
	/*

		if result.TotalCount < 1 {
			return irs.SubnetInfo{}, errors.New("Notfound: '" + reqSubnetId + "' Subnet Not found")
		}

			//vmStatus := "pending"
			//return irs.VMStatus(vmStatus)

			input := &ec2.DescribeInstancesInput{
				InstanceIds: []*string{
					aws.String(vmID),
				},
			}

			result, err := vmHandler.Client.DescribeInstances(input)
			if err != nil {
				if aerr, ok := err.(awserr.Error); ok {
					switch aerr.Code() {
					default:
						cblogger.Error(aerr.Error())
						return irs.VMStatus("")
					}
				} else {
					// Print the error, cast err to awserr.Error to get the Code and Message from an error.
					cblogger.Error(err.Error())
					return irs.VMStatus("")
				}
				return irs.VMStatus("")
			}

			cblogger.Info("Success", result)
			for _, i := range result.Reservations {
				for _, vm := range i.Instances {
					vmStatus := strings.ToUpper(*vm.State.Name)
					cblogger.Info(vmID, " EC2 Status : ", vmStatus)
					return irs.VMStatus(vmStatus)
				}
			}
	*/

	return irs.VMStatus(""), nil
}

func (vmHandler *AlibabaVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
	cblogger.Infof("Start")
	return nil, nil
	/*
		var vmStatusList []*irs.VMStatusInfo

		input := &ec2.DescribeInstancesInput{
			InstanceIds: []*string{
				nil,
			},
		}

		result, err := vmHandler.Client.DescribeInstances(input)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				default:
					cblogger.Error(aerr.Error())
					return vmStatusList
				}
			} else {
				// Print the error, cast err to awserr.Error to get the Code and Message from an error.
				cblogger.Error(err.Error())
				return vmStatusList
			}
			return vmStatusList
		}

		cblogger.Info("Success")

		for _, i := range result.Reservations {
			for _, vm := range i.Instances {
				//*vm.State.Name
				//*vm.InstanceId
				vmStatusInfo := irs.VMStatusInfo{
					VmId: *vm.InstanceId,
					//VmStatus: vmHandler.GetVMStatus(*vm.InstanceId),
					VmStatus: irs.VMStatus(strings.ToUpper(*vm.State.Name)),
				}
				cblogger.Info(vmStatusInfo.VmId, " EC2 Status : ", vmStatusInfo.VmStatus)
				vmStatusList = append(vmStatusList, &vmStatusInfo)
			}
		}

		return vmStatusList
	*/
}
