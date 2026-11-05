package resources

import (
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/eks"
)

func TestMain(m *testing.M) {
	InitLog()
	os.Exit(m.Run())
}

func describeNodegroupOutput(amiType *string) *eks.DescribeNodegroupOutput {
	return &eks.DescribeNodegroupOutput{
		Nodegroup: &eks.Nodegroup{
			NodegroupName: aws.String("g1w"),
			AmiType:       amiType,
			Status:        aws.String(eks.NodegroupStatusActive),
			InstanceTypes: []*string{aws.String("c6g.2xlarge")},
			DiskSize:      aws.Int64(20),
			ScalingConfig: &eks.NodegroupScalingConfig{
				DesiredSize: aws.Int64(1), MinSize: aws.Int64(1), MaxSize: aws.Int64(1),
			},
		},
	}
}

// The runtime copies ImageIID.SystemId into the response NameId, so SystemId must carry the AMI type.
func TestConvertNodeGroup_ImageIIDCarriesAmiType(t *testing.T) {
	h := &AwsClusterHandler{}
	info, err := h.convertNodeGroup(describeNodegroupOutput(aws.String("AL2023_ARM_64_STANDARD")))
	if err != nil {
		t.Fatalf("convertNodeGroup: %v", err)
	}
	if info.ImageIID.SystemId != "AL2023_ARM_64_STANDARD" || info.ImageIID.NameId != "AL2023_ARM_64_STANDARD" {
		t.Fatalf("ImageIID = %+v, want NameId/SystemId AL2023_ARM_64_STANDARD", info.ImageIID)
	}
}

func TestConvertNodeGroup_NilAmiType(t *testing.T) {
	h := &AwsClusterHandler{}
	info, err := h.convertNodeGroup(describeNodegroupOutput(nil))
	if err != nil {
		t.Fatalf("convertNodeGroup: %v", err)
	}
	if info.ImageIID.SystemId != "" || info.ImageIID.NameId != "" {
		t.Fatalf("ImageIID = %+v, want empty", info.ImageIID)
	}
}
