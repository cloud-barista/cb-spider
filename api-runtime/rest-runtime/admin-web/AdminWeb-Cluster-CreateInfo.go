// Proxy for AdminWeb(Multi-Connection Cluster) of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista

package adminweb

import (
	"sync"

	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/labstack/echo/v4"
)

//====================================== Multi-Connection Cluster: data for the Add Cluster dialog
//
// The Add Cluster dialog needs, for every connection, whether Cluster is supported and its VPCs with
// their subnets, SecurityGroups and KeyPairs. They are fetched on the server side and streamed per
// connection (see streamPerConnection). Lists other than the VPCs are optional: they are sent empty
// when they cannot be fetched.

// clusterCreateInfoSubnet is the part of a subnet the dialog uses (same field names as cres.SubnetInfo).
// The zone matters: an AWS EKS cluster needs subnets in at least 2 zones.
type clusterCreateInfoSubnet struct {
	IId       cres.IID `json:"IId"`
	Zone      string   `json:"Zone"`
	IPv4_CIDR string   `json:"IPv4_CIDR"`
}

// clusterCreateInfoVPC is the part of a VPC the dialog uses (same field names as cres.VPCInfo).
type clusterCreateInfoVPC struct {
	IId            cres.IID                  `json:"IId"`
	SubnetInfoList []clusterCreateInfoSubnet `json:"SubnetInfoList"`
}

// clusterCreateInfo is one streamed line: the Add Cluster dialog data of one connection.
type clusterCreateInfo struct {
	Connection     string                 `json:"connection"`
	Supported      bool                   `json:"supported"` // false: the CSP does not support Cluster
	Zone           string                 `json:"zone"`      // the zone of the connection: NCP NKS takes only subnets of this zone
	VPCs           []clusterCreateInfoVPC `json:"vpcs"`
	SecurityGroups []vmCreateInfoSG       `json:"securitygroups"`
	KeyPairs       []vmCreateInfoKeyPair  `json:"keypairs"`
	Error          string                 `json:"error,omitempty"` // VPC list error: the connection is skipped
}

// fetchClusterCreateInfo gets the Add Cluster dialog data of one connection; the 3 lists are fetched at the same time.
func fetchClusterCreateInfo(connName string) clusterCreateInfo {
	info := clusterCreateInfo{
		Connection:     connName,
		VPCs:           []clusterCreateInfoVPC{},
		SecurityGroups: []vmCreateInfoSG{},
		KeyPairs:       []vmCreateInfoKeyPair{},
	}
	if fetchUnsupportedResources(connName)["cluster"] {
		return info
	}
	info.Supported = true
	if regionName, err := getRegionName(connName); err == nil {
		if _, zone, err := getRegionZone(regionName); err == nil {
			info.Zone = zone
		}
	}

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		vpcs, err := fetchVPCs(connName)
		if err != nil {
			info.Error = err.Error()
			return
		}
		for _, vpc := range vpcs {
			if vpc == nil {
				continue
			}
			item := clusterCreateInfoVPC{IId: vpc.IId, SubnetInfoList: []clusterCreateInfoSubnet{}}
			for _, subnet := range vpc.SubnetInfoList {
				item.SubnetInfoList = append(item.SubnetInfoList, clusterCreateInfoSubnet{IId: subnet.IId, Zone: subnet.Zone, IPv4_CIDR: subnet.IPv4_CIDR})
			}
			info.VPCs = append(info.VPCs, item)
		}
	}()
	go func() {
		defer wg.Done()
		sgs, err := fetchSecurityGroups(connName)
		if err != nil {
			cblog.Info("Cluster create info: SecurityGroups of " + connName + " are not listed: " + err.Error())
			return
		}
		for _, sg := range sgs {
			if sg != nil {
				info.SecurityGroups = append(info.SecurityGroups, vmCreateInfoSG{IId: sg.IId, VpcIID: sg.VpcIID})
			}
		}
	}()
	go func() {
		defer wg.Done()
		keyPairs, err := fetchKeyPairs(connName)
		if err != nil {
			cblog.Info("Cluster create info: KeyPairs of " + connName + " are not listed: " + err.Error())
			return
		}
		for _, keyPair := range keyPairs {
			if keyPair != nil {
				info.KeyPairs = append(info.KeyPairs, vmCreateInfoKeyPair{IId: keyPair.IId})
			}
		}
	}()
	wg.Wait()
	return info
}

// ClusterCreateInfo streams the Add Cluster dialog data of the given connections, one JSON line each,
// in the order the connections finish.
// GET /spider/adminweb/cluster-createinfo?ids=conn1,conn2
func ClusterCreateInfo(c echo.Context) error {
	return streamPerConnection(c, connectionIDs(c), func(connName string) any { return fetchClusterCreateInfo(connName) })
}
