package resources

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
)

func TestSetterSubnetWithOnlyAddressPrefixes(t *testing.T) {
	handler := &AzureVPCHandler{}
	subnet := &armnetwork.Subnet{
		Name: toStrPtr("subnet-multi"),
		ID:   toStrPtr("/subscriptions/x/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/vnet/subnets/subnet-multi"),
		Properties: &armnetwork.SubnetPropertiesFormat{
			AddressPrefixes: []*string{toStrPtr("198.51.100.0/25"), toStrPtr("198.51.100.128/25")},
		},
	}

	info := handler.setterSubnet(subnet)
	if info.IPv4_CIDR != "198.51.100.0/25" {
		t.Fatalf("IPv4_CIDR = %q, want %q", info.IPv4_CIDR, "198.51.100.0/25")
	}
}

func TestSetterVPCWithoutAddressSpace(t *testing.T) {
	handler := &AzureVPCHandler{}
	network := &armnetwork.VirtualNetwork{
		Name:       toStrPtr("vnet"),
		ID:         toStrPtr("/subscriptions/x/resourceGroups/rg/providers/Microsoft.Network/virtualNetworks/vnet"),
		Properties: &armnetwork.VirtualNetworkPropertiesFormat{},
	}

	info := handler.setterVPC(network)
	if info.IPv4_CIDR != "" {
		t.Fatalf("IPv4_CIDR = %q, want empty", info.IPv4_CIDR)
	}
}

func TestConvertRuleInfoWithOnlyPluralFields(t *testing.T) {
	protocol := armnetwork.SecurityRuleProtocolTCP
	access := armnetwork.SecurityRuleAccessAllow
	direction := armnetwork.SecurityRuleDirectionInbound
	rule := &armnetwork.SecurityRule{
		Name: toStrPtr("rule-multi"),
		Properties: &armnetwork.SecurityRulePropertiesFormat{
			Protocol:                   &protocol,
			Access:                     &access,
			Direction:                  &direction,
			SourceAddressPrefixes:      []*string{toStrPtr("203.0.113.0/25"), toStrPtr("203.0.113.128/25")},
			DestinationAddressPrefixes: []*string{toStrPtr("192.0.2.0/24")},
			DestinationPortRanges:      []*string{toStrPtr("8080-8090"), toStrPtr("443")},
		},
	}

	info, err := convertRuleInfoAZToCB(rule)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.CIDR != "203.0.113.0/25" {
		t.Errorf("CIDR = %q, want %q", info.CIDR, "203.0.113.0/25")
	}
	if info.FromPort != "8080" || info.ToPort != "8090" {
		t.Errorf("ports = %s-%s, want 8080-8090", info.FromPort, info.ToPort)
	}
	if info.Direction != "inbound" || info.IPProtocol != "tcp" {
		t.Errorf("direction/protocol = %s/%s, want inbound/tcp", info.Direction, info.IPProtocol)
	}
}

func TestSetterSecSkipsRuleWithoutAccess(t *testing.T) {
	handler := &AzureSecurityHandler{}
	sg := &armnetwork.SecurityGroup{
		Name: toStrPtr("nsg"),
		ID:   toStrPtr("/subscriptions/x/resourceGroups/rg/providers/Microsoft.Network/networkSecurityGroups/nsg"),
		Properties: &armnetwork.SecurityGroupPropertiesFormat{
			SecurityRules: []*armnetwork.SecurityRule{
				{Name: toStrPtr("no-props")},
				{Name: toStrPtr("no-access"), Properties: &armnetwork.SecurityRulePropertiesFormat{}},
			},
		},
	}

	info := handler.setterSec(sg)
	if info.SecurityRules == nil || len(*info.SecurityRules) != 0 {
		t.Fatalf("SecurityRules = %v, want empty", info.SecurityRules)
	}
}
