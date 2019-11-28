// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// EC2 Hander (AWS SDK GO Version 1.16.26, Thanks AWS.)
//
// by zephy@mz.co.kr, 2019.09.

package resources

import (
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

func (vmHandler *AlibabaVMHandler) ResumeVM(vmID string) (irs.VMStatus, error) {
	cblogger.Infof("vmID : [%s]", vmID)
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

func (vmHandler *AlibabaVMHandler) SuspendVM(vmID string) (irs.VMStatus, error) {
	cblogger.Infof("vmID : [%s]", vmID)
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

func (vmHandler *AlibabaVMHandler) RebootVM(vmID string) (irs.VMStatus, error) {
	cblogger.Infof("vmID : [%s]", vmID)
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

func (vmHandler *AlibabaVMHandler) TerminateVM(vmID string) (irs.VMStatus, error) {
	cblogger.Infof("vmID : [%s]", vmID)
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

//- 보안그룹의 경우 멀티개 설정이 가능한데 현재는 1개만 입력 받음
// @Todo : SecurityID에 보안그룹 Name을 할당하는게 맞는지 확인 필요
func (vmHandler *AlibabaVMHandler) GetVM(vmID string) (irs.VMInfo, error) {
	cblogger.Infof("vmID : [%s]", vmID)
	return irs.VMInfo{}, nil
	/*

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
				}
			} else {
				// Print the error, cast err to awserr.Error to get the Code and Message from an error.
				cblogger.Error(err.Error())
			}
			return irs.VMInfo{}
		}

		cblogger.Info("Success", result)

		vmInfo := irs.VMInfo{}
		for _, i := range result.Reservations {
			//vmInfo := ExtractDescribeInstances(result.Reservations[0])
			vmInfo = ExtractDescribeInstances(i)
		}

		cblogger.Info("vmInfo", vmInfo)
		return vmInfo
	*/
}

// DescribeInstances결과에서 EC2 세부 정보 추출
// VM 생성 시에는 Running 이전 상태의 정보가 넘어오기 때문에
// 최종 정보 기반으로 리턴 받고 싶으면 GetVM에 통합해야 할 듯.
func ExtractDescribeInstances(instancInfo *ecs.Instance) irs.VMInfo {
	return irs.VMInfo{}
	/*
		//cblogger.Info("ExtractDescribeInstances", instancInfo)
		cblogger.Debug("Instances[0]", instancInfo.Instances[0])

		//"stopped" / "terminated" / "running" ...
		var state string
		state = *instancInfo.Instances[0].Status
		cblogger.Info("EC2 상태 : [%s]", state)

		//VM상태와 무관하게 항상 값이 존재하는 항목들만 초기화
		vmInfo := irs.VMInfo{
			Name:           *instancInfo.Instances[0].InstanceName,
			Id:             *instancInfo.Instances[0].InstanceId,
			ImageID:        *instancInfo.Instances[0].ImageId,
			SpecID:         *instancInfo.Instances[0].InstanceType,
			KeyPairID:      *instancInfo.Instances[0].KeyName,
			GuestUserID:    "",
			AdditionalInfo: "State:" + *instancInfo.Instances[0].State.Name,
		}

		// vmInfo.StartTime =  // StartTime time.Time

		//cblogger.Info("=======>타입 : ", reflect.TypeOf(*instancInfo.Instances[0]))
		//cblogger.Info("===> PublicIpAddress TypeOf : ", reflect.TypeOf(instancInfo.Instances[0].PublicIpAddress))
		//cblogger.Info("===> PublicIpAddress ValueOf : ", reflect.ValueOf(instancInfo.Instances[0].PublicIpAddress))

		//vmInfo.PublicIP = *instancInfo.Instances[0].NetworkInterfaces[0].Association.PublicIp
		//vmInfo.PublicDNS = *instancInfo.Instances[0].NetworkInterfaces[0].Association.PublicDnsName

		// 특정 항목(예:EIP)은 VM 상태와 무관하게 동작하므로 VM 상태와 무관하게 Nil처리로 모든 필드를 처리 함.
		if !reflect.ValueOf(instancInfo.Instances[0].PublicIpAddress).IsNil() {
			vmInfo.PublicIP = *instancInfo.Instances[0].PublicIpAddress
		}

		if !reflect.ValueOf(instancInfo.Instances[0].PublicDnsName).IsNil() {
			vmInfo.PublicDNS = *instancInfo.Instances[0].PublicDnsName
		}

		cblogger.Info("===> BlockDeviceMappings ValueOf : ", reflect.ValueOf(instancInfo.Instances[0].BlockDeviceMappings))
		if !reflect.ValueOf(instancInfo.Instances[0].BlockDeviceMappings).IsNil() {
			if !reflect.ValueOf(instancInfo.Instances[0].BlockDeviceMappings[0].DeviceName).IsNil() {
				vmInfo.GuestBlockDisk = *instancInfo.Instances[0].BlockDeviceMappings[0].DeviceName
			}
		}

		if !reflect.ValueOf(instancInfo.Instances[0].Placement.AvailabilityZone).IsNil() {
			vmInfo.Region = irs.RegionInfo{
				Region: *instancInfo.Instances[0].Placement.AvailabilityZone,
			}
		}

		//NetworkInterfaces 배열 값들
		if !reflect.ValueOf(instancInfo.Instances[0].NetworkInterfaces).IsNil() {
			if !reflect.ValueOf(instancInfo.Instances[0].NetworkInterfaces[0].VpcId).IsNil() {
				vmInfo.VNetworkID = *instancInfo.Instances[0].NetworkInterfaces[0].VpcId
			}

			if !reflect.ValueOf(instancInfo.Instances[0].NetworkInterfaces[0].SubnetId).IsNil() {
				vmInfo.SubNetworkID = *instancInfo.Instances[0].NetworkInterfaces[0].SubnetId
			}

			if !reflect.ValueOf(instancInfo.Instances[0].NetworkInterfaces[0].Groups).IsNil() {
				if !reflect.ValueOf(instancInfo.Instances[0].NetworkInterfaces[0].Groups[0].GroupId).IsNil() {
					vmInfo.SecurityID = *instancInfo.Instances[0].NetworkInterfaces[0].Groups[0].GroupId
				}
			}
		}

		//SecurityName: *instancInfo.Instances[0].NetworkInterfaces[0].Groups[0].GroupName,
		vmInfo.VNIC = "eth0 - 값 위치 확인 필요"

		//vmInfo.PrivateIP = *instancInfo.Instances[0].NetworkInterfaces[0].PrivateIpAddress	//없는 경우 존재해서 Instances[0].PrivateIpAddress로 대체 - i-0b75cac73c4575386
		if !reflect.ValueOf(instancInfo.Instances[0].PrivateIpAddress).IsNil() {
			vmInfo.PrivateIP = *instancInfo.Instances[0].PrivateIpAddress
		}

		//vmInfo.PrivateDNS = *instancInfo.Instances[0].NetworkInterfaces[0].PrivateDnsName		//없는 경우 존재해서 Instances[0].PrivateDnsName로 대체 - i-0b75cac73c4575386
		if !reflect.ValueOf(instancInfo.Instances[0].PrivateDnsName).IsNil() {
			vmInfo.PrivateDNS = *instancInfo.Instances[0].PrivateDnsName
		}

		if !reflect.ValueOf(instancInfo.Instances[0].RootDeviceName).IsNil() {
			vmInfo.GuestBootDisk = *instancInfo.Instances[0].RootDeviceName
		}

		//Name은 Tag의 "Name" 속성에만 저장됨
		cblogger.Debug("Name Tag 찾기")
		for _, t := range instancInfo.Instances[0].Tags {
			if *t.Key == "Name" {
				vmInfo.Name = *t.Value
				cblogger.Debug("EC2 명칭 : ", vmInfo.Name)
				break
			}
		}

		return vmInfo
	*/
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
func (vmHandler *AlibabaVMHandler) GetVMStatus(vmID string) (irs.VMStatus, error) {
	cblogger.Infof("vmID : [%s]", vmID)
	/*

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

// AssociationId 대신 PublicIP로도 가능 함.
func (vmHandler *AlibabaVMHandler) AssociatePublicIP(allocationId string, instanceId string) (bool, error) {
	cblogger.Infof("EC2에 퍼블릭 IP할당 - AllocationId : [%s], InstanceId : [%s]", allocationId, instanceId)
	return true, nil

	/*

		// EC2에 할당.
		// Associate the new Elastic IP address with an existing EC2 instance.
		assocRes, err := vmHandler.Client.AssociateAddress(&ec2.AssociateAddressInput{
			AllocationId: aws.String(allocationId),
			InstanceId:   aws.String(instanceId),
		})

		spew.Dump(assocRes)
		cblogger.Infof("[%s] EC2에 EIP(AllocationId : [%s]) 할당 완료 - AssociationId Id : [%s]", instanceId, allocationId, *assocRes.AssociationId)

		if err != nil {
			cblogger.Errorf("Unable to associate IP address with %s, %v", instanceId, err)
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				default:
					cblogger.Errorf(aerr.Error())
				}
			} else {
				// Print the error, cast err to awserr.Error to get the Code and
				// Message from an error.
				cblogger.Errorf(err.Error())
			}
			return false, err
		}

		cblogger.Info(assocRes)
		return true, nil
	*/
}
