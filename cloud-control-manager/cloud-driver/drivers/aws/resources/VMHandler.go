// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// EC2 Hander (AWS SDK GO Version 1.16.26, Thanks AWS.)
//
// by powerkim@powerkim.co.kr, 2019.03.
package resources

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"

	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AwsVMHandler struct {
	Region idrv.RegionInfo
	Client *ec2.EC2
}

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("AWS VMHandler")
}

func Connect(region string) *ec2.EC2 {
	// setup Region
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)

	if err != nil {
		fmt.Println("Could not create instance", err)
		return nil
	}

	// Create EC2 service client
	svc := ec2.New(sess)

	return svc
}

// @Todo : SecurityGroupId 배열 처리 방안
// 1개의 VM만 생성되도록 수정 (MinCount / MaxCount 이용 안 함)
//키페어 이름(예:mcloud-barista)은 아래 URL에 나오는 목록 중 "키페어 이름"의 값을 적으면 됨.
//https://ap-northeast-2.console.aws.amazon.com/ec2/v2/home?region=ap-northeast-2#KeyPairs:sort=keyName
func (vmHandler *AwsVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (irs.VMInfo, error) {
	cblogger.Info(vmReqInfo)
	spew.Dump(vmReqInfo)

	//imageID := vmReqInfo.ImageInfo.Id
	//instanceType := vmReqInfo.SpecID // "t2.micro"
	//keyName := vmReqInfo.KeyPairInfo.Name
	//securityGroupID := vmReqInfo.SecurityInfo.Id // "sg-0df1c209ea1915e4b" - 미지정시 보안 그룹명이 "default"인 보안 그룹이 사용 됨.
	//subnetID := vmReqInfo.VNetworkInfo.Id // "subnet-cf9ccf83" - 미지정시 기본 VPC의 기본 서브넷이 임의로 이용되며 PublicIP가 할당 됨.
	//baseName := vmReqInfo.Name            //"mcloud-barista-VMHandlerTest"

	imageID := vmReqInfo.ImageId
	instanceType := vmReqInfo.VMSpecId // "t2.micro"
	minCount := aws.Int64(1)
	maxCount := aws.Int64(1)
	keyName := vmReqInfo.KeyPairName
	subnetID := vmReqInfo.VirtualNetworkId // "subnet-cf9ccf83" - 미지정시 기본 VPC의 기본 서브넷이 임의로 이용되며 PublicIP가 할당 됨.
	baseName := vmReqInfo.VMName           //"mcloud-barista-VMHandlerTest"

	cblogger.Info("Create EC2 Instance")

	input := &ec2.RunInstancesInput{
		ImageId:      aws.String(imageID),
		InstanceType: aws.String(instanceType),
		MinCount:     minCount,
		MaxCount:     maxCount,
		KeyName:      aws.String(keyName),

		SecurityGroupIds: aws.StringSlice(vmReqInfo.SecurityGroupIds),

		/*SecurityGroupIds: []*string{
			aws.String(securityGroupID), // set a security group.
		},*/

		SubnetId: aws.String(subnetID), // set a subnet.

	}
	cblogger.Info(input)

	// Specify the details of the instance that you want to create.
	runResult, err := vmHandler.Client.RunInstances(input)
	if err != nil {
		cblogger.Errorf("EC2 인스턴스 생성 실패 : ", err)
		return irs.VMInfo{}, err
	}

	if len(runResult.Instances) < 1 {
		return irs.VMInfo{}, errors.New("AWS로부터 전달 받은 VM 정보가 없습니다.")
	}

	newVmId := *runResult.Instances[0].InstanceId
	cblogger.Infof("[%s] VM이 생성되었습니다.", newVmId)
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
		cblogger.Errorf("[%s] VM에 Name Tag 설정 실패", newVmId)
		cblogger.Error(errtag)
		return irs.VMInfo{}, errtag
	}

	//Public IP및 최신 정보 전달을 위해 부팅이 완료될 때까지 대기했다가 전달하는 것으로 변경 함.
	cblogger.Info("Public IP 할당 및 최신 정보 획득을 위해 EC2가 Running 상태가 될때까지 대기")
	WaitForRun(vmHandler.Client, newVmId)
	cblogger.Info("EC2 Running 상태 완료 : ", runResult.Instances[0].State.Name)

	//EC2에 EIP 할당 (펜딩 상태에서는 EIP 할당 불가)
	cblogger.Infof("[%s] EC2에 [%s] IP 할당 시작", newVmId, vmReqInfo.PublicIPId)
	assocRes, errIp := vmHandler.AssociatePublicIP(vmReqInfo.PublicIPId, newVmId)
	if errIp != nil {
		cblogger.Errorf("EC2[%s]에 Public IP Id[%s]를 할당 할 수 없습니다 - %v", newVmId, vmReqInfo.PublicIPId, err)
		return irs.VMInfo{}, nil
	}

	cblogger.Infof("[%s] EC2에 Public IP 할당 결과 : ", newVmId, assocRes)

	//
	//vNic 추가 요청이 있는 경우 전달 받은 vNic을 VM에 추가 함.
	//
	if vmReqInfo.NetworkInterfaceId != "" {
		_, errvNic := vmHandler.AttachNetworkInterface(vmReqInfo.NetworkInterfaceId, newVmId)
		if errvNic != nil {
			cblogger.Errorf("vNic [%s] 추가 실패!", vmReqInfo.NetworkInterfaceId)
			cblogger.Error(errvNic)
		} else {
			cblogger.Infof("vNic [%s] 추가 완료", vmReqInfo.NetworkInterfaceId)
		}
	}

	//최신 정보 조회
	vmInfo, _ := vmHandler.GetVM(newVmId)

	/*
		//빠른 생성을 위해 Running 상태를 대기하지 않고 최소한의 정보만 리턴 함.
		//Running 상태를 대기 후 Public Ip 등의 정보를 추출하려면 GetVM()을 호출해서 최신 정보를 다시 받아와야 함.
		//vmInfo :=GetVM(runResult.Instances[0].InstanceId)

		//cblogger.Info("EC2 Running 상태 대기")
		//WaitForRun(vmHandler.Client, *runResult.Instances[0].InstanceId)
		//cblogger.Info("EC2 Running 상태 완료 : ", runResult.Instances[0].State.Name)

		vmInfo := ExtractDescribeInstances(runResult)
		//속도상 VM 정보를 다시 조회하지 않았기 때문에 Tag 정보가 누락되어서 Name 정보가 설정되어 있지 않음.
		if vmInfo.Name == "" {
			vmInfo.Name = baseName
		}
	*/

	return vmInfo, nil
}

//VM이 Running 상태일때까지 대기 함.
func WaitForRun(svc *ec2.EC2, instanceID string) {
	cblogger.Infof("EC2 ID : [%s]", instanceID)

	input := &ec2.DescribeInstancesInput{
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

func (vmHandler *AwsVMHandler) ResumeVM(vmID string) error {
	cblogger.Infof("vmID : [%s]", vmID)
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
			return err
		} else {
			//fmt.Println("Success", result.StartingInstances)
			cblogger.Info("Success", result.StartingInstances)
		}
	} else { // This could be due to a lack of permissions
		//fmt.Println("Error", err)
		cblogger.Error(err)
		return err
	}

	return nil
}

func (vmHandler *AwsVMHandler) SuspendVM(vmID string) error {
	cblogger.Infof("vmID : [%s]", vmID)
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
			return err
		} else {
			cblogger.Info("Success", result.StoppingInstances)
		}
	} else {
		cblogger.Error("Error", err)
		return err
	}

	return nil
}

func (vmHandler *AwsVMHandler) RebootVM(vmID string) error {
	cblogger.Infof("vmID : [%s]", vmID)
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
			return err
		} else {
			cblogger.Info("Success", result)
		}
	} else { // This could be due to a lack of permissions
		cblogger.Info("리부팅 권한이 없는 것같음.")
		cblogger.Error("Error", err)
		return err
	}
	return nil
}

func (vmHandler *AwsVMHandler) TerminateVM(vmID string) error {
	cblogger.Infof("vmID : [%s]", vmID)
	input := &ec2.TerminateInstancesInput{
		//InstanceIds: instanceIds,
		InstanceIds: []*string{
			aws.String(vmID),
		},
	}

	_, err := vmHandler.Client.TerminateInstances(input)
	if err != nil {
		cblogger.Error("Could not termiate instances", err)
		return err
	} else {
		cblogger.Info("Success")
	}
	return nil
}

//- 보안그룹의 경우 멀티개 설정이 가능한데 현재는 1개만 입력 받음
// @Todo : SecurityID에 보안그룹 Name을 할당하는게 맞는지 확인 필요
func (vmHandler *AwsVMHandler) GetVM(vmID string) (irs.VMInfo, error) {
	cblogger.Infof("vmID : [%s]", vmID)

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
		return irs.VMInfo{}, err
	}

	cblogger.Info("Success", result)

	/*
		- 보안그룹의 경우 멀티개 설정이 가능한데 현재는 1개만 입력 받음
		- SecurityID에 보안그룹 Name을 할당하는게 맞는지 확인 필요
	*/
	vmInfo := irs.VMInfo{}
	for _, i := range result.Reservations {
		//vmInfo := ExtractDescribeInstances(result.Reservations[0])
		vmInfo = ExtractDescribeInstances(i)
	}

	cblogger.Info("vmInfo", vmInfo)
	return vmInfo, nil
}

// DescribeInstances결과에서 EC2 세부 정보 추출
// VM 생성 시에는 Running 이전 상태의 정보가 넘어오기 때문에
// 최종 정보 기반으로 리턴 받고 싶으면 GetVM에 통합해야 할 듯.
func ExtractDescribeInstances(reservation *ec2.Reservation) irs.VMInfo {
	//cblogger.Info("ExtractDescribeInstances", reservation)
	cblogger.Debug("Instances[0]", reservation.Instances[0])

	//"stopped" / "terminated" / "running" ...
	var state string
	state = *reservation.Instances[0].State.Name
	cblogger.Infof("EC2 상태 : [%s]", state)

	//VM상태와 무관하게 항상 값이 존재하는 항목들만 초기화
	vmInfo := irs.VMInfo{
		Id:          *reservation.Instances[0].InstanceId,
		ImageId:     *reservation.Instances[0].ImageId,
		VMSpecId:    *reservation.Instances[0].InstanceType,
		KeyPairName: *reservation.Instances[0].KeyName,
		//GuestUserID:    "",
		//AdditionalInfo: "State:" + *reservation.Instances[0].State.Name,
	}

	keyValueList := []irs.KeyValue{
		{Key: "State", Value: *reservation.Instances[0].State.Name},
		{Key: "Architecture", Value: *reservation.Instances[0].Architecture},
	}

	//cblogger.Info("=======>타입 : ", reflect.TypeOf(*reservation.Instances[0]))
	//cblogger.Info("===> PublicIpAddress TypeOf : ", reflect.TypeOf(reservation.Instances[0].PublicIpAddress))
	//cblogger.Info("===> PublicIpAddress ValueOf : ", reflect.ValueOf(reservation.Instances[0].PublicIpAddress))

	//vmInfo.PublicIP = *reservation.Instances[0].NetworkInterfaces[0].Association.PublicIp
	//vmInfo.PublicDNS = *reservation.Instances[0].NetworkInterfaces[0].Association.PublicDnsName

	// 특정 항목(예:EIP)은 VM 상태와 무관하게 동작하므로 VM 상태와 무관하게 Nil처리로 모든 필드를 처리 함.
	if !reflect.ValueOf(reservation.Instances[0].PublicIpAddress).IsNil() {
		vmInfo.PublicIP = *reservation.Instances[0].PublicIpAddress
	}

	if !reflect.ValueOf(reservation.Instances[0].PublicDnsName).IsNil() {
		vmInfo.PublicDNS = *reservation.Instances[0].PublicDnsName
	}

	cblogger.Info("===> BlockDeviceMappings ValueOf : ", reflect.ValueOf(reservation.Instances[0].BlockDeviceMappings))
	if !reflect.ValueOf(reservation.Instances[0].BlockDeviceMappings).IsNil() {
		if !reflect.ValueOf(reservation.Instances[0].BlockDeviceMappings[0].DeviceName).IsNil() {
			vmInfo.VMBlockDisk = *reservation.Instances[0].BlockDeviceMappings[0].DeviceName
		}
	}

	if !reflect.ValueOf(reservation.Instances[0].Placement.AvailabilityZone).IsNil() {
		vmInfo.Region = irs.RegionInfo{
			Zone: *reservation.Instances[0].Placement.AvailabilityZone,
		}
	}

	//NetworkInterfaces 배열 값들
	if !reflect.ValueOf(reservation.Instances[0].NetworkInterfaces).IsNil() {
		if !reflect.ValueOf(reservation.Instances[0].NetworkInterfaces[0].VpcId).IsNil() {
			vmInfo.VirtualNetworkId = *reservation.Instances[0].NetworkInterfaces[0].VpcId
			keyValueList = append(keyValueList, irs.KeyValue{Key: "VpcId", Value: *reservation.Instances[0].NetworkInterfaces[0].VpcId})
		}

		if !reflect.ValueOf(reservation.Instances[0].NetworkInterfaces[0].SubnetId).IsNil() {
			keyValueList = append(keyValueList, irs.KeyValue{Key: "SubnetId", Value: *reservation.Instances[0].NetworkInterfaces[0].SubnetId})
		}

		if !reflect.ValueOf(reservation.Instances[0].NetworkInterfaces[0].Attachment).IsNil() {
			vmInfo.NetworkInterfaceId = *reservation.Instances[0].NetworkInterfaces[0].Attachment.AttachmentId
		}

		for _, security := range reservation.Instances[0].NetworkInterfaces[0].Groups {
			vmInfo.SecurityGroupIds = append(vmInfo.SecurityGroupIds, *security.GroupId)
		}

		/*
			if !reflect.ValueOf(reservation.Instances[0].NetworkInterfaces[0].Groups).IsNil() {
				vmInfo.SecurityGroupIds = *reservation.Instances[0].NetworkInterfaces[0].Groups[0]
				if !reflect.ValueOf(reservation.Instances[0].NetworkInterfaces[0].Groups[0].GroupId).IsNil() {
					vmInfo.SecurityID = *reservation.Instances[0].NetworkInterfaces[0].Groups[0].GroupId
				}
			}
		*/
	}

	//SecurityName: *reservation.Instances[0].NetworkInterfaces[0].Groups[0].GroupName,
	//vmInfo.VNIC = "eth0 - 값 위치 확인 필요"

	//vmInfo.PrivateIP = *reservation.Instances[0].NetworkInterfaces[0].PrivateIpAddress	//없는 경우 존재해서 Instances[0].PrivateIpAddress로 대체 - i-0b75cac73c4575386
	if !reflect.ValueOf(reservation.Instances[0].PrivateIpAddress).IsNil() {
		vmInfo.PrivateIP = *reservation.Instances[0].PrivateIpAddress
	}

	//vmInfo.PrivateDNS = *reservation.Instances[0].NetworkInterfaces[0].PrivateDnsName		//없는 경우 존재해서 Instances[0].PrivateDnsName로 대체 - i-0b75cac73c4575386
	if !reflect.ValueOf(reservation.Instances[0].PrivateDnsName).IsNil() {
		vmInfo.PrivateDNS = *reservation.Instances[0].PrivateDnsName
	}

	if !reflect.ValueOf(reservation.Instances[0].RootDeviceName).IsNil() {
		vmInfo.VMBootDisk = *reservation.Instances[0].RootDeviceName
	}

	if !reflect.ValueOf(reservation.Instances[0].KeyName).IsNil() {
		keyValueList = append(keyValueList, irs.KeyValue{Key: "KeyName", Value: *reservation.Instances[0].KeyName})
	}

	//Name은 Tag의 "Name" 속성에만 저장됨
	cblogger.Debug("Name Tag 찾기")
	for _, t := range reservation.Instances[0].Tags {
		if *t.Key == "Name" {
			vmInfo.Name = *t.Value
			cblogger.Debug("EC2 명칭 : ", vmInfo.Name)
			break
		}
	}

	vmInfo.KeyValueList = keyValueList
	return vmInfo
}

func (vmHandler *AwsVMHandler) ListVM() ([]*irs.VMInfo, error) {
	cblogger.Infof("Start")
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
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and Message from an error.
			cblogger.Error(err.Error())
		}
		return nil, err
	}

	cblogger.Info("Success")

	for _, i := range result.Reservations {
		for _, vm := range i.Instances {
			cblogger.Info("[%s] EC2 정보 조회", *vm.InstanceId)
			vmInfo, _ := vmHandler.GetVM(*vm.InstanceId)
			vmInfoList = append(vmInfoList, &vmInfo)
		}
	}

	return vmInfoList, nil
}

//SHUTTING-DOWN / TERMINATED
func (vmHandler *AwsVMHandler) GetVMStatus(vmID string) (irs.VMStatus, error) {
	cblogger.Infof("vmID : [%s]", vmID)

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
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and Message from an error.
			cblogger.Error(err.Error())
		}
		return irs.VMStatus(""), err
	}

	cblogger.Info("Success", result)
	for _, i := range result.Reservations {
		for _, vm := range i.Instances {
			vmStatus := strings.ToUpper(*vm.State.Name)
			cblogger.Info(vmID, " EC2 Status : ", vmStatus)
			return irs.VMStatus(vmStatus), nil
		}
	}

	return irs.VMStatus(""), errors.New("상태 정보를 찾을 수 없습니다.")
}

func (vmHandler *AwsVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
	cblogger.Infof("Start")
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
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and Message from an error.
			cblogger.Error(err.Error())
		}
		return nil, err
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

	return vmStatusList, nil
}

// AssociationId 대신 PublicIP로도 가능 함.
func (vmHandler *AwsVMHandler) AssociatePublicIP(allocationId string, instanceId string) (bool, error) {
	cblogger.Infof("EC2에 퍼블릭 IP할당 - AllocationId : [%s], InstanceId : [%s]", allocationId, instanceId)

	// EC2에 할당.
	// Associate the new Elastic IP address with an existing EC2 instance.
	assocRes, err := vmHandler.Client.AssociateAddress(&ec2.AssociateAddressInput{
		AllocationId: aws.String(allocationId),
		InstanceId:   aws.String(instanceId),
	})

	spew.Dump(assocRes)
	//cblogger.Infof("[%s] EC2에 EIP(AllocationId : [%s]) 할당 완료 - AssociationId Id : [%s]", instanceId, allocationId, *assocRes.AssociationId)

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
}

// 전달 받은 vNic을 VM에 추가함.
func (vmHandler *AwsVMHandler) AttachNetworkInterface(vNicId string, instanceId string) (bool, error) {
	cblogger.Infof("EC2[%s] VM에 vNic[%s] 추가 시작", vNicId, instanceId)

	input := &ec2.AttachNetworkInterfaceInput{
		DeviceIndex:        aws.Int64(1),
		InstanceId:         aws.String(instanceId),
		NetworkInterfaceId: aws.String(vNicId),
	}

	result, err := vmHandler.Client.AttachNetworkInterface(input)
	cblogger.Info(result)

	if err != nil {
		cblogger.Errorf("EC2[%s] VM에 vNic[%s] 추가 실패", vNicId, instanceId)
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Errorf(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and Message from an error.
			cblogger.Errorf(err.Error())
		}
		return false, err
	}

	return true, nil
}
