package resources

import (
	"testing"

	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vpc"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

// subnet builds irs.SubnetInfo the same way as the VPC handler (irs.StructToKeyValueList(vpc.Subnet)).
func lbTestSubnet(id, name, subnetType, usageType string) *irs.SubnetInfo {
	s := vpc.Subnet{
		SubnetNo:   ncloud.String(id),
		SubnetName: ncloud.String(name),
		SubnetType: &vpc.CommonCode{Code: ncloud.String(subnetType), CodeName: ncloud.String(subnetType)},
		UsageType:  &vpc.CommonCode{Code: ncloud.String(usageType), CodeName: ncloud.String(usageType)},
	}
	return &irs.SubnetInfo{IId: irs.IID{NameId: name, SystemId: id}, KeyValueList: irs.StructToKeyValueList(s)}
}

func TestSelectLbTypeSubnetId(t *testing.T) {
	gen := lbTestSubnet("s-gen", "spider-vm-ssh-test-subnet-01", "PUBLIC", "GEN")
	nksPriv := lbTestSubnet("s-nks-priv", "ncp-lb-priv-12345", "PRIVATE", "LOADB")
	nksPub := lbTestSubnet("s-nks-pub", "ncp-lb-pub-12345", "PUBLIC", "LOADB")
	nlbOwn := lbTestSubnet("s-nlb", LbTypeSubnetDefaultName+"-abcd1234", "PUBLIC", "LOADB")

	tests := []struct {
		name      string
		list      []*irs.SubnetInfo
		lbNetType string
		owned     bool
		want      string
	}{
		// the reported case: VPC with NKS Cluster LB subnets (PRIVATE listed last) => PUBLIC NLB must not get the PRIVATE one
		{"public NLB with NKS subnets", []*irs.SubnetInfo{gen, nksPub, nksPriv}, NcpPublicLbType, false, "s-nks-pub"},
		{"public NLB, only private LB subnet", []*irs.SubnetInfo{gen, nksPriv}, NcpPublicLbType, false, ""}, // => driver creates a new PUBLIC LB subnet
		{"public NLB, own subnet", []*irs.SubnetInfo{gen, nlbOwn}, NcpPublicLbType, false, "s-nlb"},
		{"no LB subnet", []*irs.SubnetInfo{gen}, NcpPublicLbType, false, ""},
		{"private NLB prefers PRIVATE", []*irs.SubnetInfo{nksPub, nksPriv}, NcpInternalLbType, false, "s-nks-priv"},
		{"private NLB falls back to any LB subnet", []*irs.SubnetInfo{gen, nlbOwn}, NcpInternalLbType, false, "s-nlb"},
		// delete path: never pick LB subnets of other services
		{"delete: only NLB-owned", []*irs.SubnetInfo{nksPriv, nksPub, nlbOwn}, "", true, "s-nlb"},
		{"delete: NKS subnets only => none", []*irs.SubnetInfo{nksPriv, nksPub}, "", true, ""},
		{"nil entry is ignored", []*irs.SubnetInfo{nil, nksPub}, NcpPublicLbType, false, "s-nks-pub"},
	}
	for _, tc := range tests {
		if got := selectLbTypeSubnetId(tc.list, tc.lbNetType, tc.owned); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}
