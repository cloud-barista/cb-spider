// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by powerkim@etri.re.kr, 2019.06.

package resources

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/davecgh/go-spew/spew"
)

// AssociationId 대신 PublicIP로도 가능 함.
func AssociatePublicIP(client *ec2.EC2, allocationId string, instanceId string) (bool, error) {
	cblogger.Infof("EC2에 퍼블릭 IP할당 - AllocationId : [%s], InstanceId : [%s]", allocationId, instanceId)

	// EC2에 할당.
	// Associate the new Elastic IP address with an existing EC2 instance.
	assocRes, err := client.AssociateAddress(&ec2.AssociateAddressInput{
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
}
