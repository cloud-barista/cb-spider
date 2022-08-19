package resources

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/davecgh/go-spew/spew"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

//// AWS API 1:1로 대응

//WaitUntilVolumeAvailable
func WaitUntilVolumeAvailable(svc *ec2.EC2, volumeID string) error {
	input := &ec2.DescribeVolumesInput{
		VolumeIds: []*string{
			aws.String(volumeID),
		},
	}
	err := svc.WaitUntilVolumeAvailable(input)
	if err != nil {
		cblogger.Errorf("failed to wait until volume available: %v", err)
		return err
	}
	cblogger.Info("=========WaitUntilVolumeAvailable() 종료")
	return nil
}

//WaitUntilVolumeDeleted
func WaitUntilVolumeDeleted(svc *ec2.EC2, volumeID string) error {
	input := &ec2.DescribeVolumesInput{
		VolumeIds: []*string{
			aws.String(volumeID),
		},
	}
	err := svc.WaitUntilVolumeDeleted(input)
	if err != nil {
		cblogger.Errorf("failed to wait until volume deleted: %v", err)
		return err
	}
	cblogger.Info("=========WaitUntilVolumeDeleted() 종료")
	return nil
}

//WaitUntilVolumeInUse : attached
func WaitUntilVolumeInUse(svc *ec2.EC2, volumeID string) error {
	input := &ec2.DescribeVolumesInput{
		VolumeIds: []*string{
			aws.String(volumeID),
		},
	}
	err := svc.WaitUntilVolumeInUse(input)
	if err != nil {
		cblogger.Errorf("failed to wait until volume in use: %v", err)
		return err
	}
	cblogger.Info("=========WaitUntilVolumeInUse() 종료")
	return nil
}

/*
	Instance 정보조회.
	기본은 목록 조회이며 filter조건이 있으면 해당 filter 조건으로 검색하도록
*/
func DescribeInstances(svc *ec2.EC2, vmIIDs []irs.IID) (*ec2.DescribeInstancesOutput, error) {
	input := &ec2.DescribeInstancesInput{}
	var instanceIds []*string

	if vmIIDs != nil {
		for _, vmIID := range vmIIDs {
			instanceIds = append(instanceIds, aws.String(vmIID.SystemId))
		}
		input.InstanceIds = instanceIds
	}

	result, err := svc.DescribeInstances(input)

	return result, err
}

/*
	1개 인스턴스의 정보 조회
*/
func DescribeInstanceById(svc *ec2.EC2, vmIID irs.IID) (*ec2.Instance, error) {
	var vmIIDs []irs.IID
	var iid irs.IID

	if vmIID == iid {
		return nil, errors.New("instanceID is empty.)")
	}

	vmIIDs = append(vmIIDs, vmIID)

	result, err := DescribeInstances(svc, vmIIDs)
	if err != nil {
		return nil, err
	}
	instance := result.Reservations[0].Instances[0]
	return instance, err
}

/*
	1개 인스턴스의 상태조회
*/
func DescribeInstanceStatus(svc *ec2.EC2, vmIID irs.IID) (string, error) {

	instance, err := DescribeInstanceById(svc, vmIID)
	if err != nil {
		return "", err
	}
	//type InstanceState struct {
	//	_    struct{} `type:"structure"`
	//	Code *int64   `locationName:"code" type:"integer"`
	//	Name *string  `locationName:"name" type:"string" enum:"InstanceStateName"`
	//}
	status := instance.State.Name

	return *status, err
}

/*
	1개 인스턴스에서 사용중인 disk 와 device 목록
*/
func DescribeInstanceDiskDeviceList(svc *ec2.EC2, vmIID irs.IID) ([]*ec2.InstanceBlockDeviceMapping, error) {

	instance, err := DescribeInstanceById(svc, vmIID)
	if err != nil {
		return nil, err
	}

	//device := instance.BlockDeviceMappings[0].DeviceName
	//blockDevice := instance.BlockDeviceMappings[0].Ebs
	return instance.BlockDeviceMappings, err
}

/*
	List 와 Get 이 같은 API 호출
	filter 조건으로 VolumeId 를 넣도록하고
	return은 그대로 DescribeVolumesOutput.
	Get에서는 1개만 들어있으므로 [0]번째 사용

	각 항목을 irs.DiskInfo로 변환하는 convertVolumeInfoToDiskInfo 로 필요Data 생성
*/
func DescribeVolumnes(svc *ec2.EC2, volumeIdList []*string) (*ec2.DescribeVolumesOutput, error) {

	input := &ec2.DescribeVolumesInput{}

	if volumeIdList != nil {
		input.VolumeIds = volumeIdList
	}

	result, err := svc.DescribeVolumes(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return nil, err
	}
	if cblogger.Level.String() == "debug" {
		spew.Dump(result)
	}
	return result, nil
}

func DescribeVolumneById(svc *ec2.EC2, volumeId string) (*ec2.Volume, error) {
	volumeIdList := []*string{}
	input := &ec2.DescribeVolumesInput{}

	if volumeId != "" {
		volumeIdList = append(volumeIdList, aws.String(volumeId))
		input.VolumeIds = volumeIdList
	}

	result, err := svc.DescribeVolumes(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return nil, err
	}
	if cblogger.Level.String() == "debug" {
		spew.Dump(result)
	}

	for _, volume := range result.Volumes {
		if strings.EqualFold(volumeId, *volume.VolumeId) {
			//break
			return volume, nil
		}
	}

	return nil, awserr.New("404", "["+volumeId+"] 볼륨 정보가 존재하지 않습니다.", nil)
}
