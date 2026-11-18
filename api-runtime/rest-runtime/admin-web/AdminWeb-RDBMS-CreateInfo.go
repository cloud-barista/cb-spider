// Proxy for AdminWeb(Multi-Connection RDBMS) of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista

package adminweb

import (
	"encoding/json"
	"net/url"
	"sync"

	"github.com/labstack/echo/v4"
)

//====================================== Multi-Connection RDBMS: data for the Add RDBMS dialog
//
// The Add RDBMS dialog needs, for every connection, whether RDBMS (and which DB engine) is supported
// and its VPCs with their subnets and SecurityGroups. They are fetched on the server side and streamed
// per connection (see streamPerConnection).
//
// The RDBMS MetaInfo (engine versions, DB specs, storage ranges) is not fetched: it takes long on some
// CSPs (ex: about 80s on Alibaba, 10s on AWS). The dialog fills the values verified by the RDBMS tests
// (test/rdbms-mysql-test, test/rdbms-mariadb-test) instead, and they can be edited.

// rdbmsCreateInfo is one streamed line: the Add RDBMS dialog data of one connection.
type rdbmsCreateInfo struct {
	Connection     string                 `json:"connection"`
	Supported      bool                   `json:"supported"` // false: the CSP does not support RDBMS
	Engines        map[string]bool        `json:"engines"`   // mysql, mariadb, postgresql
	VPCs           []clusterCreateInfoVPC `json:"vpcs"`      // VPCs with their subnets (and zones)
	SecurityGroups []vmCreateInfoSG       `json:"securitygroups"`
	Error          string                 `json:"error,omitempty"` // VPC list error: the connection is skipped
}

// rdbmsCapability is the part of the driver capability the dialog uses.
type rdbmsCapability struct {
	RDBMSHandler           bool `json:"RDBMSHandler"`
	RDBMSMySQLHandler      bool `json:"RDBMSMySQLHandler"`
	RDBMSMariaDBHandler    bool `json:"RDBMSMariaDBHandler"`
	RDBMSPostgreSQLHandler bool `json:"RDBMSPostgreSQLHandler"`
}

// fetchRDBMSCreateInfo gets the Add RDBMS dialog data of one connection; VPCs and SecurityGroups are
// fetched at the same time. If the capability cannot be read, the connection is treated as supported
// with every engine, so a hiccup there does not hide the connection (the creation reports the error).
func fetchRDBMSCreateInfo(connName string) rdbmsCreateInfo {
	info := rdbmsCreateInfo{
		Connection:     connName,
		Supported:      true,
		Engines:        map[string]bool{"mysql": true, "mariadb": true, "postgresql": true},
		VPCs:           []clusterCreateInfoVPC{},
		SecurityGroups: []vmCreateInfoSG{},
	}
	if resBody, err := getResourceList_JsonByte("driver/capability?ConnectionName=" + url.QueryEscape(connName)); err == nil {
		var capability rdbmsCapability
		if json.Unmarshal(resBody, &capability) == nil {
			info.Supported = capability.RDBMSHandler
			info.Engines = map[string]bool{
				"mysql":      capability.RDBMSMySQLHandler,
				"mariadb":    capability.RDBMSMariaDBHandler,
				"postgresql": capability.RDBMSPostgreSQLHandler,
			}
		}
	}
	if !info.Supported {
		return info
	}

	var wg sync.WaitGroup
	wg.Add(2)
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
			cblog.Info("RDBMS create info: SecurityGroups of " + connName + " are not listed: " + err.Error())
			return
		}
		for _, sg := range sgs {
			if sg != nil {
				info.SecurityGroups = append(info.SecurityGroups, vmCreateInfoSG{IId: sg.IId, VpcIID: sg.VpcIID})
			}
		}
	}()
	wg.Wait()
	return info
}

// RDBMSCreateInfo streams the Add RDBMS dialog data of the given connections, one JSON line each,
// in the order the connections finish.
// GET /spider/adminweb/rdbms-createinfo?ids=conn1,conn2
func RDBMSCreateInfo(c echo.Context) error {
	return streamPerConnection(c, connectionIDs(c), func(connName string) any { return fetchRDBMSCreateInfo(connName) })
}
