// Proxy for AdminWeb(Multi-Connection VM) of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista

package adminweb

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"sync"

	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/labstack/echo/v4"
)

//====================================== Multi-Connection VM: data for the Add VM dialog
//
// The Add VM dialog needs, for every connection, its VPC/Subnets, SecurityGroups, KeyPairs and the
// recently created Image/Spec of its CSP region: 4 requests per connection from the browser before.
// They are fetched on the server side and streamed per connection (see streamPerConnection).
// Every list is optional, as before: a list that cannot be fetched is sent empty.

// vmCreateInfoSubnet is the part of a subnet the dialog uses (same field names as cres.SubnetInfo).
type vmCreateInfoSubnet struct {
	IId  cres.IID `json:"IId"`
	Zone string   `json:"Zone"`
}

// vmCreateInfoVPC is the part of a VPC the dialog uses (same field names as cres.VPCInfo).
type vmCreateInfoVPC struct {
	IId            cres.IID             `json:"IId"`
	SubnetInfoList []vmCreateInfoSubnet `json:"SubnetInfoList"`
}

// vmCreateInfoSG is the part of a SecurityGroup the dialog uses (same field names as cres.SecurityInfo).
type vmCreateInfoSG struct {
	IId    cres.IID `json:"IId"`
	VpcIID cres.IID `json:"VpcIID"`
}

// vmCreateInfoKeyPair is the part of a KeyPair the dialog uses (same field names as cres.KeyPairInfo).
type vmCreateInfoKeyPair struct {
	IId cres.IID `json:"IId"`
}

// vmCreateInfo is one streamed line: the Add VM dialog data of one connection.
type vmCreateInfo struct {
	Connection     string                `json:"connection"`
	VPCs           []vmCreateInfoVPC     `json:"vpcs"`
	SecurityGroups []vmCreateInfoSG      `json:"securitygroups"`
	KeyPairs       []vmCreateInfoKeyPair `json:"keypairs"`
	Recents        json.RawMessage       `json:"recents"` // recent Image/Spec list, as returned by /spider/vm/imagespecrecent
}

// fetchRecentImageSpecs returns the 10 Image/Spec combinations most recently used to create VMs in the
// CSP region of the connection (same query as the dialog sent before), or an empty JSON list.
// Without a provider or region the list would mix other regions, so it is left empty then.
func fetchRecentImageSpecs(connName string) json.RawMessage {
	empty := json.RawMessage("[]")
	provider, _ := getProviderName(connName)
	regionName, err := getRegionName(connName)
	if provider == "" || err != nil {
		return empty
	}
	region, _, _ := getRegionZone(regionName)
	if region == "" {
		return empty
	}

	params := url.Values{}
	params.Set("CSP", provider)
	params.Set("Region", region)
	params.Set("Limit", "10")
	params.Set("SortBy", "last_created_at")
	params.Set("SortOrder", "desc")
	resp, err := httpGetWithAuth(spiderBaseURL() + "/spider/vm/imagespecrecent?" + params.Encode())
	if err != nil {
		return empty
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil || resp.StatusCode != http.StatusOK || !json.Valid(body) || string(body) == "null" {
		return empty
	}
	return json.RawMessage(body)
}

// fetchVMCreateInfo gets the Add VM dialog data of one connection; the 4 lists are fetched at the same time.
func fetchVMCreateInfo(connName string) vmCreateInfo {
	info := vmCreateInfo{
		Connection:     connName,
		VPCs:           []vmCreateInfoVPC{},
		SecurityGroups: []vmCreateInfoSG{},
		KeyPairs:       []vmCreateInfoKeyPair{},
		Recents:        json.RawMessage("[]"),
	}

	var wg sync.WaitGroup
	wg.Add(4)
	go func() {
		defer wg.Done()
		vpcs, err := fetchVPCs(connName)
		if err != nil {
			cblog.Info("VM create info: VPCs of " + connName + " are not listed: " + err.Error())
			return
		}
		for _, vpc := range vpcs {
			if vpc == nil {
				continue
			}
			item := vmCreateInfoVPC{IId: vpc.IId, SubnetInfoList: []vmCreateInfoSubnet{}}
			for _, subnet := range vpc.SubnetInfoList {
				item.SubnetInfoList = append(item.SubnetInfoList, vmCreateInfoSubnet{IId: subnet.IId, Zone: subnet.Zone})
			}
			info.VPCs = append(info.VPCs, item)
		}
	}()
	go func() {
		defer wg.Done()
		sgs, err := fetchSecurityGroups(connName)
		if err != nil {
			cblog.Info("VM create info: SecurityGroups of " + connName + " are not listed: " + err.Error())
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
			cblog.Info("VM create info: KeyPairs of " + connName + " are not listed: " + err.Error())
			return
		}
		for _, keyPair := range keyPairs {
			if keyPair != nil {
				info.KeyPairs = append(info.KeyPairs, vmCreateInfoKeyPair{IId: keyPair.IId})
			}
		}
	}()
	go func() {
		defer wg.Done()
		info.Recents = fetchRecentImageSpecs(connName)
	}()
	wg.Wait()
	return info
}

// VMCreateInfo streams the Add VM dialog data of the given connections, one JSON line each,
// in the order the connections finish.
// GET /spider/adminweb/vm-createinfo?ids=conn1,conn2
func VMCreateInfo(c echo.Context) error {
	return streamPerConnection(c, connectionIDs(c), func(connName string) any { return fetchVMCreateInfo(connName) })
}
