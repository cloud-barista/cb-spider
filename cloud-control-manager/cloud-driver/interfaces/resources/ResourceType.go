// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by CB-Spider Team, 2024.06.

package resources

import (
	"fmt"
	"strings"
)

type RSType string

const (
	ALL       RSType = "all"
	IMAGE     RSType = "image"
	VPC       RSType = "vpc"
	SUBNET    RSType = "subnet"
	SG        RSType = "sg"
	KEY       RSType = "keypair"
	VM        RSType = "vm"
	NLB       RSType = "nlb"
	DISK      RSType = "disk"
	MYIMAGE   RSType = "myimage"
	CLUSTER   RSType = "cluster"
	NODEGROUP RSType = "nodegroup"

	FILESYSTEM RSType = "filesystem"
)

func RSTypeString(rsType RSType) string {
	switch rsType {
	case ALL:
		return "All Resources"
	case IMAGE:
		return "VM Image"
	case VPC:
		return "VPC"
	case SUBNET:
		return "Subnet"
	case SG:
		return "Security Group"
	case KEY:
		return "VM KeyPair"
	case VM:
		return "VM"
	case NLB:
		return "Network Load Balancer"
	case DISK:
		return "Disk"
	case MYIMAGE:
		return "MyImage"
	case CLUSTER:
		return "Kubernetes Cluster"
	case NODEGROUP:
		return "Kubernetes NodeGroup"
	case FILESYSTEM:
		return "FileSystem"
	default:
		return string(rsType) + " is not supported Resource!!"

	}
}

func StringToRSType(str string) (RSType, error) {

	str = strings.ToLower(str)

	switch str {
	case "all":
		return ALL, nil
	case "image":
		return IMAGE, nil
	case "vpc":
		return VPC, nil
	case "subnet":
		return SUBNET, nil
	case "sg":
		return SG, nil
	case "keypair":
		return KEY, nil
	case "vm":
		return VM, nil
	case "nlb":
		return NLB, nil
	case "disk":
		return DISK, nil
	case "myimage":
		return MYIMAGE, nil
	case "cluster":
		return CLUSTER, nil
	case "nodegroup":
		return NODEGROUP, nil
	case "filesystem":
		return FILESYSTEM, nil
	default:
		return "", fmt.Errorf("%s is not a valid resource type", str)
	}
}
