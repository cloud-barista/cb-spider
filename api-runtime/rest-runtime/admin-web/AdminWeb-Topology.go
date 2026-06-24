// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// AdminWeb Topology Visualization (Cytoscape.js, MIT License)
//
// by CB-Spider Team, 2025.06.

package adminweb

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"

	"github.com/labstack/echo/v4"
)

// ---- Graph Node/Edge types ----

type TopoNode struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Kind     string `json:"kind"`     // vpc, subnet, sg, vm, nic, privateip, publicip
	Parent   string `json:"parent"`   // compound parent node id
	PrivateIP string `json:"privateIP,omitempty"`
	PublicIP  string `json:"publicIP,omitempty"`
	Status   string `json:"status,omitempty"`
}

type TopoEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label,omitempty"`
	Kind   string `json:"kind,omitempty"` // nic-vm, nic-sg, nic-pub, vm-pub
}

// buildTopologyData builds nodes and edges from all cloud resources.
func buildTopologyData(connConfig string) ([]TopoNode, []TopoEdge, error) {
	var nodes []TopoNode
	var edges []TopoEdge

	// ---- VPCs + Subnets ----
	vpcs, _ := fetchVPCs(connConfig)
	for _, vpc := range vpcs {
		vpcID := "vpc-" + vpc.IId.NameId
		nodes = append(nodes, TopoNode{
			ID:    vpcID,
			Label: vpc.IId.NameId,
			Kind:  "vpc",
		})
		for _, sn := range vpc.SubnetInfoList {
			snID := "subnet-" + sn.IId.NameId
			nodes = append(nodes, TopoNode{
				ID:     snID,
				Label:  sn.IId.NameId,
				Kind:   "subnet",
				Parent: vpcID,
			})
		}
	}

	// ---- Security Groups ----
	sgs, _ := fetchSecurityGroups(connConfig)
	for _, sg := range sgs {
		sgID := "sg-" + sg.IId.NameId
		// find parent vpc
		parentVPC := ""
		if sg.VpcIID.NameId != "" {
			parentVPC = "vpc-" + sg.VpcIID.NameId
		}
		nodes = append(nodes, TopoNode{
			ID:     sgID,
			Label:  sg.IId.NameId,
			Kind:   "sg",
			Parent: parentVPC,
		})
	}

	// ---- NICs (registered in Spider) ----
	// Use SystemId as node key so VM's NIC edges always reference the same node.
	nics, _ := fetchNICs(connConfig)
	nicSysIDMap := make(map[string]string) // systemId → nodeID  (for VM edge lookup)
	nicNodeIDs  := make(map[string]bool)   // all existing NIC node IDs

	for _, nic := range nics {
		// Key by SystemId (CSP ENI ID) for reliable matching
		sysID := nic.IId.SystemId
		nicID := "nic-" + sysID
		nicNodeIDs[nicID] = true
		nicSysIDMap[sysID] = nicID

		parentSubnet := ""
		if nic.SubnetIID.NameId != "" {
			parentSubnet = "subnet-" + nic.SubnetIID.NameId
		}
		label := nic.IId.NameId
		if label == "" { label = sysID }
		if nic.PrivateIP != "" {
			label += "\n" + nic.PrivateIP
		}
		nodes = append(nodes, TopoNode{
			ID:        nicID,
			Label:     label,
			Kind:      "nic",
			Parent:    parentSubnet,
			PrivateIP: nic.PrivateIP,
			Status:    string(nic.Status),
		})
		for _, sg := range nic.SecurityGroupIIDs {
			if sg.NameId != "" {
				edges = append(edges, TopoEdge{Source: nicID, Target: "sg-" + sg.NameId, Kind: "nic-sg"})
			}
		}
		for i, privIP := range nic.PrivateIPs {
			if i < len(nic.PublicIPs) && nic.PublicIPs[i] != "" {
				edges = append(edges, TopoEdge{
					Source: nicID, Target: "pub-" + nic.PublicIPs[i],
					Label: privIP + " →", Kind: "nic-pub",
				})
			}
		}
	}

	// ---- VMs ----
	vms, _ := fetchVMs(connConfig)
	for _, vm := range vms {
		vmID := "vm-" + vm.IId.NameId
		parentSubnet := ""
		if vm.SubnetIID.NameId != "" {
			parentSubnet = "subnet-" + vm.SubnetIID.NameId
		}
		nodes = append(nodes, TopoNode{ID: vmID, Label: vm.IId.NameId, Kind: "vm", Parent: parentSubnet})

		for _, nic := range vm.NICs {
			sysID := nic.IId.SystemId
			if sysID == "" { continue }

			// Prefer SystemId-based node ID for exact match
			nicNodeID := "nic-" + sysID
			if !nicNodeIDs[nicNodeID] {
				// Not in registry — create ghost node
				subnetParent := ""
				if vm.SubnetIID.NameId != "" {
					subnetParent = "subnet-" + vm.SubnetIID.NameId
				}
				label := fmt.Sprintf("eth%d", nic.DeviceIndex)
				if len(nic.PrivateIPs) > 0 { label += "\n" + nic.PrivateIPs[0] }
				nodes = append(nodes, TopoNode{
					ID: nicNodeID, Label: label, Kind: "nic",
					Parent: subnetParent, Status: "unregistered",
				})
				nicNodeIDs[nicNodeID] = true
				// Ghost NIC inherits VM's SecurityGroups (SGs specified at VM creation apply to all NICs)
				for _, sg := range vm.SecurityGroupIIds {
					if sg.NameId != "" {
						edges = append(edges, TopoEdge{
							Source: nicNodeID, Target: "sg-" + sg.NameId, Kind: "nic-sg",
						})
					}
				}
				for j, pub := range nic.PublicIPs {
					if pub != "" {
						privLabel := ""
						if j < len(nic.PrivateIPs) { privLabel = nic.PrivateIPs[j] + " →" }
						edges = append(edges, TopoEdge{
							Source: nicNodeID, Target: "pub-" + pub,
							Label: privLabel, Kind: "nic-pub",
						})
					}
				}
			}
			edges = append(edges, TopoEdge{
				Source: vmID, Target: nicNodeID,
				Label: fmt.Sprintf("eth%d", nic.DeviceIndex), Kind: "vm-nic",
			})
		}
	}

	// ---- Public IPs (registered in Spider) ----
	pips, _ := fetchPublicIPs(connConfig)
	pubIPNodeIDs := make(map[string]bool)
	for _, pip := range pips {
		pubIPNodeID := "pub-" + pip.PublicIPAddress
		pubIPNodeIDs[pubIPNodeID] = true
		nodes = append(nodes, TopoNode{
			ID:    pubIPNodeID,
			Label: pip.IId.NameId + "\n" + pip.PublicIPAddress,
			Kind:  "publicip",
		})
	}

	// Ensure all PublicIP nodes referenced in edges exist (ghost nodes for unregistered IPs)
	for i := range edges {
		e := &edges[i]
		if len(e.Target) > 4 && e.Target[:4] == "pub-" {
			if !pubIPNodeIDs[e.Target] {
				addr := e.Target[4:]
				nodes = append(nodes, TopoNode{
					ID:    e.Target,
					Label: addr,
					Kind:  "publicip",
				})
				pubIPNodeIDs[e.Target] = true
			}
		}
	}

	return nodes, edges, nil
}

// TopologyManagement renders the topology visualization page.
func TopologyManagement(c echo.Context) error {
	connConfig := c.Param("ConnectConfig")
	if connConfig == "region not set" {
		return c.HTML(http.StatusOK, `<html><body><br><br>
			<label style="font-size:24px;color:#606262;">&nbsp;&nbsp;&nbsp;Please select a Connection Configuration! (MENU: 2.CONNECTION)</label>
			</body></html>`)
	}

	nodes, edges, err := buildTopologyData(connConfig)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	nodesJSON, _ := json.Marshal(nodes)
	edgesJSON, _ := json.Marshal(edges)

	data := struct {
		ConnectionConfig string
		NodesJSON        template.JS
		EdgesJSON        template.JS
		APIUsername      string
		APIPassword      string
	}{
		ConnectionConfig: connConfig,
		NodesJSON:        template.JS(nodesJSON),
		EdgesJSON:        template.JS(edgesJSON),
		APIUsername:      os.Getenv("SPIDER_USERNAME"),
		APIPassword:      os.Getenv("SPIDER_PASSWORD"),
	}

	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/topology.html")
	tmpl, err := template.New("topology.html").Funcs(template.FuncMap{
		"inc": func(i int) int { return i + 1 },
	}).ParseFiles(templatePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error loading template: " + err.Error()})
	}

	c.Response().WriteHeader(http.StatusOK)
	if err := tmpl.Execute(c.Response().Writer, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}
	return nil
}
