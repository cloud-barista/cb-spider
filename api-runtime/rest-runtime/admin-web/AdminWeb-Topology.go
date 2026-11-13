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
	"sort"
	"strings"
	"sync"

	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/labstack/echo/v4"
)

// ---- Graph Node/Edge types ----

type TopoNode struct {
	ID        string            `json:"id"`
	Name      string            `json:"name,omitempty"` // Spider NameId of a registered resource (used by delete); empty for unregistered nodes
	Label     string            `json:"label"`
	Kind      string            `json:"kind"`   // vpc, subnet, sg, vm, nic, privateip, publicip, nlb, rdbms, cluster, nodegroup, k8snode
	Parent    string            `json:"parent"` // compound parent node id
	PrivateIP string            `json:"privateIP,omitempty"`
	PublicIP  string            `json:"publicIP,omitempty"`
	Status    string            `json:"status,omitempty"`
	Info      map[string]string `json:"info,omitempty"`      // extra key/values shown in the detail panel
	Public    bool              `json:"public,omitempty"`    // reachable from the Internet (public NLB, publicly accessible RDBMS)
	Conn      string            `json:"conn,omitempty"`      // connection name (all-connections view only; used by delete)
	Lazy      bool              `json:"lazy,omitempty"`      // connection group whose graph is loaded on first expand
	InfoOrder []string          `json:"infoOrder,omitempty"` // display order of Info keys (others follow, sorted)
}

// infoList builds a node's detail Info while keeping the display order; empty values are skipped.
type infoList struct {
	m     map[string]string
	order []string
}

func (l *infoList) add(k, v string) {
	if strings.TrimSpace(v) == "" {
		return
	}
	if l.m == nil {
		l.m = map[string]string{}
	}
	if _, ok := l.m[k]; !ok {
		l.order = append(l.order, k)
	}
	l.m[k] = v
}

func iidLabels(iids []cres.IID) string {
	out := []string{}
	for _, i := range iids {
		if l := iidLabel(i); l != "" {
			out = append(out, l)
		}
	}
	return strings.Join(out, ", ")
}

func kvValue(kvs []cres.KeyValue, key string) string {
	for _, kv := range kvs {
		if strings.EqualFold(kv.Key, key) {
			return kv.Value
		}
	}
	return ""
}

func rootDiskText(diskType, size, device string) string {
	parts := []string{}
	if diskType != "" && diskType != "default" {
		parts = append(parts, diskType)
	}
	if size != "" && size != "default" {
		parts = append(parts, size+" GB")
	}
	if device != "" {
		parts = append(parts, device)
	}
	return strings.Join(parts, " · ")
}

// vmDetail is the VM detail shown in the topology side panel: only the essentials (about the same
// level as a Kubernetes worker node); the VM page has everything else.
func vmDetail(vm *cres.VMInfo) infoList {
	var l infoList
	l.add("VM Spec", vm.VMSpecName)
	l.add("Image", iidLabel(vm.ImageIId))
	l.add("_imageType", string(vm.ImageType)) // hidden: picks the image API for the Image link (PublicImage | MyImage)
	zone := vm.Region.Zone
	if zone == "" {
		zone = vm.Region.Region
	}
	l.add("Zone", zone)
	privIP := vm.PrivateIP
	if privIP == "" && len(vm.PrivateIPs) > 0 {
		privIP = vm.PrivateIPs[0]
	}
	pubIP := vm.PublicIP
	if pubIP == "" && len(vm.PublicIPs) > 0 {
		pubIP = vm.PublicIPs[0]
	}
	l.add("Private IP", privIP)
	l.add("Public IP", pubIP)
	l.add("Root Disk", rootDiskText(vm.RootDiskType, vm.RootDiskSize, ""))
	l.add("KeyPair", iidLabel(vm.KeyPairIId))
	l.add("System ID", vm.IId.SystemId)
	return l
}

const publicMark = " · 🌐 public" // appended to the 2nd card line of Internet-facing resources

// shortEngineVersion trims a CSP engine version for the card (ex: "8.0.46-rds.20260908" → "8.0");
// the full version stays in the detail panel.
func shortEngineVersion(v string) string {
	v = strings.SplitN(v, "-", 2)[0]
	parts := strings.Split(v, ".")
	if len(parts) > 2 {
		parts = parts[:2]
	}
	return strings.Join(parts, ".")
}

type TopoEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label,omitempty"`
	Kind   string `json:"kind,omitempty"` // vm-nic, nic-sg, nic-pub, nlb-vm, rdbms-sg, cluster-sg, nlb-subnet, rdbms-subnet, cluster-subnet
}

// topoSources holds every resource list used by the topology, fetched in parallel.
type topoSources struct {
	vpcs     []*cres.VPCInfo
	sgs      []*cres.SecurityInfo
	nics     []*cres.NICInfo
	vms      []*cres.VMInfo
	pips     []*cres.PublicIPInfo
	nlbs     []*cres.NLBInfo
	rdbmss   []*cres.RDBMSInfo
	clusters []*cres.ClusterInfo
	disks    []*cres.DiskInfo
	vmStatus map[string]string // VM NameId → status (Running, Suspended, ...)
}

// fetchTopoSources fetches all resource lists concurrently.
// Resources the driver does not support (Driver Capability API) are skipped;
// any fetch error just leaves that list empty, so one failing resource never breaks the page.
func fetchTopoSources(connConfig string) topoSources {
	var src topoSources
	na := fetchUnsupportedResources(connConfig)

	var wg sync.WaitGroup
	run := func(resource string, f func()) {
		if na[resource] {
			return
		}
		wg.Add(1)
		go func() { defer wg.Done(); f() }()
	}
	run("vpc", func() { src.vpcs, _ = fetchVPCs(connConfig) })
	run("securitygroup", func() { src.sgs, _ = fetchSecurityGroups(connConfig) })
	run("nic", func() { src.nics, _ = fetchNICs(connConfig) })
	run("vm", func() { src.vms, _ = fetchVMs(connConfig) })
	run("vm", func() { src.vmStatus, _ = fetchAllVMStatuses(connConfig) })
	run("publicip", func() { src.pips, _ = fetchPublicIPs(connConfig) })
	run("nlb", func() { src.nlbs, _ = fetchNLBs(connConfig) })
	run("rdbms", func() { src.rdbmss, _ = fetchRDBMSs(connConfig) })
	run("cluster", func() { src.clusters, _ = fetchClusters(connConfig) })
	run("disk", func() { src.disks, _ = fetchDisks(connConfig) })
	wg.Wait()
	return src
}

// iidIndex resolves an IID to a node ID by NameId first, then SystemId
// (some CSP responses fill only one of them in references).
type iidIndex struct {
	byName map[string]string
	bySys  map[string]string
}

func newIIDIndex() *iidIndex {
	return &iidIndex{byName: map[string]string{}, bySys: map[string]string{}}
}

func (x *iidIndex) add(iid cres.IID, nodeID string) {
	if iid.NameId != "" {
		x.byName[iid.NameId] = nodeID
	}
	if iid.SystemId != "" {
		x.bySys[iid.SystemId] = nodeID
	}
}

func (x *iidIndex) get(iid cres.IID) string {
	if id, ok := x.byName[iid.NameId]; ok && iid.NameId != "" {
		return id
	}
	if id, ok := x.bySys[iid.SystemId]; ok && iid.SystemId != "" {
		return id
	}
	return ""
}

func iidLabel(iid cres.IID) string {
	if iid.NameId != "" {
		return iid.NameId
	}
	return iid.SystemId
}

// buildTopologyData builds nodes and edges from all cloud resources.
func buildTopologyData(connConfig string) ([]TopoNode, []TopoEdge, error) {
	nodes := []TopoNode{}
	edges := []TopoEdge{}
	src := fetchTopoSources(connConfig)

	vpcIdx, subnetIdx, sgIdx, vmIdx := newIIDIndex(), newIIDIndex(), newIIDIndex(), newIIDIndex()

	// ---- VPCs + Subnets ----
	for _, vpc := range src.vpcs {
		vpcID := "vpc-" + vpc.IId.NameId
		vpcIdx.add(vpc.IId, vpcID)
		nodes = append(nodes, TopoNode{
			ID:    vpcID,
			Name:  vpc.IId.NameId,
			Label: vpc.IId.NameId,
			Kind:  "vpc",
		})
		for _, sn := range vpc.SubnetInfoList {
			snID := "subnet-" + sn.IId.NameId
			subnetIdx.add(sn.IId, snID)
			nodes = append(nodes, TopoNode{
				ID:     snID,
				Name:   sn.IId.NameId,
				Label:  sn.IId.NameId,
				Kind:   "subnet",
				Parent: vpcID,
			})
		}
	}

	// ---- Security Groups ----
	for _, sg := range src.sgs {
		sgID := "sg-" + sg.IId.NameId
		sgIdx.add(sg.IId, sgID)
		// find parent vpc
		parentVPC := ""
		if sg.VpcIID.NameId != "" {
			parentVPC = "vpc-" + sg.VpcIID.NameId
		}
		nodes = append(nodes, TopoNode{
			ID:     sgID,
			Name:   sg.IId.NameId,
			Label:  sg.IId.NameId,
			Kind:   "sg",
			Parent: parentVPC,
		})
	}

	// ---- NICs (registered in Spider) ----
	// Use SystemId as node key so VM's NIC edges always reference the same node.
	nics := src.nics
	nicSysIDMap := make(map[string]string) // systemId → nodeID  (for VM edge lookup)
	nicNodeIDs := make(map[string]bool)    // all existing NIC node IDs

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
		if label == "" {
			label = sysID
		}
		if nic.PrivateIP != "" {
			label += "\n" + nic.PrivateIP
		}
		nodes = append(nodes, TopoNode{
			ID:        nicID,
			Name:      nic.IId.NameId,
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
	vms := src.vms
	for _, vm := range vms {
		vmID := "vm-" + vm.IId.NameId
		parentSubnet := ""
		if vm.SubnetIID.NameId != "" {
			parentSubnet = "subnet-" + vm.SubnetIID.NameId
		}
		vmIdx.add(vm.IId, vmID)
		det := vmDetail(vm)
		nodes = append(nodes, TopoNode{ID: vmID, Name: vm.IId.NameId, Label: vm.IId.NameId, Kind: "vm", Parent: parentSubnet,
			Status: src.vmStatus[vm.IId.NameId], Info: det.m, InfoOrder: det.order})

		for _, nic := range vm.NICs {
			sysID := nic.IId.SystemId
			if sysID == "" {
				continue
			}

			// Prefer SystemId-based node ID for exact match
			nicNodeID := "nic-" + sysID
			if !nicNodeIDs[nicNodeID] {
				// Not in registry — create ghost node
				subnetParent := ""
				if vm.SubnetIID.NameId != "" {
					subnetParent = "subnet-" + vm.SubnetIID.NameId
				}
				label := fmt.Sprintf("eth%d", nic.DeviceIndex)
				if len(nic.PrivateIPs) > 0 {
					label += "\n" + nic.PrivateIPs[0]
				}
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
						if j < len(nic.PrivateIPs) {
							privLabel = nic.PrivateIPs[j] + " →"
						}
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
	pips := src.pips
	pubIPNodeIDs := make(map[string]bool)
	for _, pip := range pips {
		pubIPNodeID := "pub-" + pip.PublicIPAddress
		pubIPNodeIDs[pubIPNodeID] = true
		nodes = append(nodes, TopoNode{
			ID:    pubIPNodeID,
			Name:  pip.IId.NameId,
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

	// Edges from a VPC-level service (NLB, RDBMS, Cluster) to the subnets it uses.
	addSubnetEdges := func(srcID, kind string, subnets []cres.IID) {
		seenSN := map[string]bool{}
		for _, sn := range subnets {
			if snID := subnetIdx.get(sn); snID != "" && !seenSN[snID] {
				seenSN[snID] = true
				edges = append(edges, TopoEdge{Source: srcID, Target: snID, Kind: kind})
			}
		}
	}

	// ---- NLBs: VPC-level node, edges to backend VMs ----
	for _, nlb := range src.nlbs {
		nlbID := "nlb-" + nlb.IId.NameId
		lsn := nlb.Listener
		listener := lsn.Protocol + ":" + lsn.Port
		info := map[string]string{"Type": nlb.Type, "Listener": listener}
		if lsn.IP != "" {
			info["Listener IP"] = lsn.IP
		}
		if lsn.DNSName != "" {
			info["DNS Name"] = lsn.DNSName
		}
		if nlb.VMGroup.Port != "" {
			info["Backend"] = nlb.VMGroup.Protocol + ":" + nlb.VMGroup.Port
		}
		nlbPublic := strings.EqualFold(nlb.Type, "PUBLIC")
		line2 := listener
		if nlbPublic {
			line2 += publicMark
		}
		nodes = append(nodes, TopoNode{
			ID:     nlbID,
			Name:   nlb.IId.NameId,
			Label:  nlb.IId.NameId + "\n" + line2,
			Kind:   "nlb",
			Parent: vpcIdx.get(nlb.VpcIID),
			Info:   info,
			Public: nlbPublic,
		})
		nlbSubnets := []cres.IID{}
		for sysID := range subnetIdx.bySys {
			if len(sysID) < 6 {
				continue
			}
			for _, kv := range nlb.KeyValueList {
				if strings.Contains(kv.Value, sysID) {
					nlbSubnets = append(nlbSubnets, cres.IID{SystemId: sysID})
					break
				}
			}
		}
		sort.Slice(nlbSubnets, func(i, j int) bool { return nlbSubnets[i].SystemId < nlbSubnets[j].SystemId })
		addSubnetEdges(nlbID, "nlb-subnet", nlbSubnets)
		if len(nlbSubnets) > 0 {
			names := []string{}
			for _, sn := range nlbSubnets {
				names = append(names, strings.TrimPrefix(subnetIdx.get(sn), "subnet-"))
			}
			info["Subnets"] = strings.Join(names, ", ")
		}
		if nlb.VMGroup.VMs != nil {
			for _, vmIID := range *nlb.VMGroup.VMs {
				if vmID := vmIdx.get(vmIID); vmID != "" {
					edges = append(edges, TopoEdge{Source: nlbID, Target: vmID, Label: nlb.VMGroup.Port, Kind: "nlb-vm"})
				}
			}
		}
	}

	// ---- RDBMS: placed in its subnet when it uses exactly one, otherwise in the VPC ----
	for _, db := range src.rdbmss {
		dbID := "rdbms-" + db.IId.NameId
		parent := vpcIdx.get(db.VpcIID)
		subnetNames := []string{}
		for _, sn := range db.SubnetIIDs {
			subnetNames = append(subnetNames, iidLabel(sn))
		}
		if len(db.SubnetIIDs) == 1 {
			if snID := subnetIdx.get(db.SubnetIIDs[0]); snID != "" {
				parent = snID
			}
		}
		engine := db.DBEngine
		if db.DBEngineVersion != "" {
			engine += " " + db.DBEngineVersion
		}
		info := map[string]string{"Engine": engine, "Spec": db.DBSpec, "Access": "Private"}
		line2 := db.DBEngine
		if db.DBEngineVersion != "" {
			line2 += " " + shortEngineVersion(db.DBEngineVersion)
		}
		if db.PublicAccess {
			info["Access"] = "Public"
			line2 += publicMark
		}
		if db.Endpoint != "" {
			info["Endpoint"] = db.Endpoint
		}
		if len(subnetNames) > 0 {
			info["Subnets"] = strings.Join(subnetNames, ", ")
		}
		if db.HighAvailability {
			info["HA"] = "enabled"
		}
		nodes = append(nodes, TopoNode{
			ID:     dbID,
			Name:   db.IId.NameId,
			Label:  db.IId.NameId + "\n" + line2,
			Kind:   "rdbms",
			Parent: parent,
			Status: string(db.Status),
			Info:   info,
			Public: db.PublicAccess,
		})
		if !strings.HasPrefix(parent, "subnet-") {
			addSubnetEdges(dbID, "rdbms-subnet", db.SubnetIIDs)
		}
		for _, sg := range db.SecurityGroupIIDs {
			if sgID := sgIdx.get(sg); sgID != "" {
				edges = append(edges, TopoEdge{Source: dbID, Target: sgID, Kind: "rdbms-sg"})
			}
		}
	}

	// ---- Clusters: VPC-level node, edges to SGs ----
	for _, cl := range src.clusters {
		clID := "cluster-" + cl.IId.NameId
		ngs := []string{}
		for _, ng := range cl.NodeGroupList {
			ngs = append(ngs, fmt.Sprintf("%s (%d nodes)", iidLabel(ng.IId), len(ng.Nodes)))
		}
		subnetNames := []string{}
		for _, sn := range cl.Network.SubnetIIDs {
			subnetNames = append(subnetNames, iidLabel(sn))
		}
		info := map[string]string{"Version": cl.Version}
		if len(ngs) > 0 {
			info["Node Groups"] = strings.Join(ngs, ", ")
		}
		if len(subnetNames) > 0 {
			info["Subnets"] = strings.Join(subnetNames, ", ")
		}
		label := cl.IId.NameId
		if cl.Version != "" {
			label += "\nk8s " + cl.Version
		}
		nodes = append(nodes, TopoNode{
			ID:     clID,
			Name:   cl.IId.NameId,
			Label:  label,
			Kind:   "cluster",
			Parent: vpcIdx.get(cl.Network.VpcIID),
			Status: string(cl.Status),
			Info:   info,
		})
		addSubnetEdges(clID, "cluster-subnet", cl.Network.SubnetIIDs)
		for _, sg := range cl.Network.SecurityGroupIIDs {
			if sgID := sgIdx.get(sg); sgID != "" {
				edges = append(edges, TopoEdge{Source: clID, Target: sgID, Kind: "cluster-sg"})
			}
		}

		// Node groups (inside the cluster) and their worker nodes (inside each node group).
		// The page shows the cluster and node groups as collapsible groups, collapsed by default.
		for _, ng := range cl.NodeGroupList {
			ngID := clID + "/ng-" + ng.IId.NameId
			ngLabel := iidLabel(ng.IId)
			line2 := []string{}
			if ng.VMSpecName != "" {
				line2 = append(line2, ng.VMSpecName)
			}
			line2 = append(line2, fmt.Sprintf("%d node(s)", len(ng.Nodes)))
			var ngi infoList
			ngi.add("VM Spec", ng.VMSpecName)
			ngi.add("Image", iidLabel(ng.ImageIID))
			ngi.add("Root Disk", rootDiskText(ng.RootDiskType, ng.RootDiskSize, ""))
			ngi.add("KeyPair", iidLabel(ng.KeyPairIID))
			autoScaling := "Off"
			if ng.OnAutoScaling {
				autoScaling = "On"
			}
			ngi.add("Auto Scaling", autoScaling)
			ngi.add("Node Size", fmt.Sprintf("desired %d · min %d · max %d", ng.DesiredNodeSize, ng.MinNodeSize, ng.MaxNodeSize))
			ngi.add("Nodes", fmt.Sprintf("%d", len(ng.Nodes)))
			ngi.add("Cluster", cl.IId.NameId)
			ngi.add("System ID", ng.IId.SystemId)
			nodes = append(nodes, TopoNode{
				ID:        ngID,
				Name:      ng.IId.NameId, // used by delete (DELETE /spider/cluster/{cluster}/nodegroup/{name})
				Label:     ngLabel + "\n" + strings.Join(line2, " · "),
				Kind:      "nodegroup",
				Parent:    clID,
				Status:    string(ng.Status),
				Info:      ngi.m,
				InfoOrder: ngi.order,
			})
			for i, wn := range ng.Nodes {
				key := wn.SystemId
				if key == "" {
					key = wn.NameId
				}
				if key == "" {
					key = fmt.Sprintf("%d", i)
				}
				// A worker node is a VM created by the node group: show the node group's VM settings,
				// and the Spider VM's details when the same instance is also registered in Spider.
				var wi infoList
				var vmOf *cres.VMInfo
				for _, vm := range src.vms {
					if wn.SystemId != "" && vm.IId.SystemId == wn.SystemId {
						vmOf = vm
						break
					}
				}
				wi.add("Node Group", ngLabel)
				clText := cl.IId.NameId
				if cl.Version != "" {
					clText += " (k8s " + cl.Version + ")"
				}
				wi.add("Cluster", clText)
				wi.add("VM Spec", ng.VMSpecName)
				wi.add("Image", iidLabel(ng.ImageIID))
				wi.add("Root Disk", rootDiskText(ng.RootDiskType, ng.RootDiskSize, ""))
				wi.add("KeyPair", iidLabel(ng.KeyPairIID))
				wi.add("Node Group Status", string(ng.Status))
				if vmOf != nil {
					wi.add("Spider VM", vmOf.IId.NameId)
					wi.add("Private IP", vmOf.PrivateIP)
					wi.add("Public IP", vmOf.PublicIP)
				}
				wi.add("System ID", wn.SystemId)
				label := iidLabel(wn)
				if ng.VMSpecName != "" {
					label += "\n" + ng.VMSpecName
				}
				nodes = append(nodes, TopoNode{
					ID:        ngID + "/node-" + key,
					Label:     label,
					Kind:      "k8snode",
					Parent:    ngID,
					Info:      wi.m,
					InfoOrder: wi.order,
				})
			}
		}
	}

	// ---- Disks: not separate nodes; shown on the owner VM card (count) and in its detail ----
	diskIdx := map[string]*cres.DiskInfo{}
	for _, d := range src.disks {
		if d.IId.NameId != "" {
			diskIdx[d.IId.NameId] = d
		}
		if d.IId.SystemId != "" {
			diskIdx[d.IId.SystemId] = d
		}
	}
	vmDisks := map[string][]string{} // vm node ID → disk descriptions
	seen := map[string]bool{}        // vmID|disk → already listed
	addDisk := func(vmID string, iid cres.IID) {
		name := iidLabel(iid)
		if vmID == "" || name == "" || seen[vmID+"|"+name] {
			return
		}
		seen[vmID+"|"+name] = true
		desc := name
		d := diskIdx[iid.NameId]
		if d == nil {
			d = diskIdx[iid.SystemId]
		}
		if d != nil && d.DiskSize != "" {
			desc += " (" + d.DiskSize + "GB)"
		}
		vmDisks[vmID] = append(vmDisks[vmID], desc)
	}
	for _, vm := range src.vms {
		for _, dIID := range vm.DataDiskIIDs {
			addDisk(vmIdx.get(vm.IId), dIID)
		}
	}
	for _, d := range src.disks {
		if d.OwnerVM.NameId != "" || d.OwnerVM.SystemId != "" {
			addDisk(vmIdx.get(d.OwnerVM), d.IId)
		}
	}
	for i := range nodes {
		n := &nodes[i]
		if n.Kind != "vm" {
			continue
		}
		if ds := vmDisks[n.ID]; len(ds) > 0 {
			n.Label += fmt.Sprintf("\n💾 %d data disk", len(ds))
			if len(ds) > 1 {
				n.Label += "s"
			}
			if n.Info == nil {
				n.Info = map[string]string{}
			}
			n.Info["Data Disks"] = strings.Join(ds, ", ")
			order := []string{}
			for _, k := range n.InfoOrder {
				order = append(order, k)
				if k == "Root Disk" {
					order = append(order, "Data Disks")
				}
			}
			if len(order) == len(n.InfoOrder) {
				order = append(order, "Data Disks")
			}
			n.InfoOrder = order
		}
	}

	// Drop edges whose endpoints are missing (e.g. a reference to an unregistered or deleted resource);
	// Cytoscape rejects such edges.
	nodeIDs := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		nodeIDs[n.ID] = true
	}
	valid := edges[:0]
	for _, e := range edges {
		if nodeIDs[e.Source] && nodeIDs[e.Target] {
			valid = append(valid, e)
		}
	}
	edges = valid

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
	return renderTopology(c, connConfig, nodes, edges)
}

// allConnectionsKey is the ConnectionConfig value of the all-connections view
// (the page detects the mode with it, and saved layouts are stored under it).
const allConnectionsKey = "__all__"

// topoConn is a connection shown in the all-connections topology / connection selector.
type topoConn struct {
	provider, name, region string
}

// connectionsWithResources returns the connections shown on the Dashboard (at least one resource),
// sorted by provider and name. Uses the same counting and filter as the Dashboard.
func connectionsWithResources(connectionConfigs map[string][]ConnectionConfig) []topoConn {
	// Same filter as the Dashboard: count resources, keep connections that have any
	var wg sync.WaitGroup
	total := 0
	for _, configs := range connectionConfigs {
		total += len(configs)
	}
	countsChan := make(chan struct {
		Provider string
		Counts   ResourceCounts
	}, total)
	errorChan := make(chan error, total)
	for provider, configs := range connectionConfigs {
		for _, config := range configs {
			wg.Add(1)
			go fetchResourceCounts(config, provider, &wg, countsChan, errorChan)
		}
	}
	wg.Wait()
	close(countsChan)
	close(errorChan)
	resourceCounts := make(map[string][]ResourceCounts)
	for r := range countsChan {
		resourceCounts[r.Provider] = append(resourceCounts[r.Provider], r.Counts)
	}
	for err := range errorChan {
		fmt.Println("Error:", err)
	}
	resourceCounts = filterEmptyConnections(resourceCounts)

	var conns []topoConn
	for provider, counts := range resourceCounts {
		for _, rc := range counts {
			conns = append(conns, topoConn{provider, rc.ConnectionName, rc.RegionName})
		}
	}
	sort.Slice(conns, func(i, j int) bool {
		if conns[i].provider != conns[j].provider {
			return conns[i].provider < conns[j].provider
		}
		return conns[i].name < conns[j].name
	})

	return conns
}

// TopologyConnections returns the connections with resources for the topology page's selector:
// [{name, provider, region: "region/zone"}].
func TopologyConnections(c echo.Context) error {
	connectionConfigs, err := fetchConnectionConfigs()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	regionMap, _ := fetchRegions()
	type item struct {
		Name     string `json:"name"`
		Provider string `json:"provider"`
		Region   string `json:"region"`
	}
	list := []item{}
	for _, cc := range connectionsWithResources(connectionConfigs) {
		region := cc.region
		if rz, ok := regionMap[cc.region]; ok {
			region = rz
		}
		list = append(list, item{cc.name, cc.provider, region})
	}
	return c.JSON(http.StatusOK, list)
}

// TopologyAllManagement renders one topology of every connection shown on the Dashboard
// (connections that have at least one resource). Each connection becomes a top-level group
// ("conn" node). Only the first connection's graph is fetched here; the others are sent as
// collapsed, not-yet-loaded groups (Lazy) and fetched by the page when they are first expanded
// (TopologyConnData), since querying every CSP up front can take a long time.
func TopologyAllManagement(c echo.Context) error {
	connectionConfigs, err := fetchConnectionConfigs()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	nodes := []TopoNode{}
	edges := []TopoEdge{}
	for i, cc := range connectionsWithResources(connectionConfigs) {
		group := TopoNode{
			ID: connGroupID(cc.name), Label: cc.name, Kind: "conn", Status: cc.provider,
			Info: map[string]string{"Provider": cc.provider, "Region": cc.region},
		}
		if i > 0 {
			group.Lazy = true
			nodes = append(nodes, group)
			continue
		}
		n, e, err := buildTopologyData(cc.name)
		if err != nil {
			group.Info["Error"] = err.Error()
		}
		nodes = append(nodes, group)
		n, e = prefixConnGraph(cc.name, n, e)
		nodes = append(nodes, n...)
		edges = append(edges, e...)
	}
	return renderTopology(c, allConnectionsKey, nodes, edges)
}

// TopologyConnData returns one connection's graph for the all-connections view
// (ids prefixed and roots placed under the connection group): {nodes, edges, error}.
func TopologyConnData(c echo.Context) error {
	connConfig := c.Param("ConnectConfig")
	nodes, edges, err := buildTopologyData(connConfig)
	res := map[string]interface{}{}
	if err != nil {
		res["error"] = err.Error()
		nodes, edges = []TopoNode{}, []TopoEdge{}
	}
	res["nodes"], res["edges"] = prefixConnGraph(connConfig, nodes, edges)
	return c.JSON(http.StatusOK, res)
}

func connGroupID(connName string) string { return "conn:" + connName }

// prefixConnGraph makes a connection's node/edge ids unique in the all-connections view and puts
// its top-level nodes under the connection group.
func prefixConnGraph(connName string, nodes []TopoNode, edges []TopoEdge) ([]TopoNode, []TopoEdge) {
	prefix := connName + "/"
	outN := make([]TopoNode, 0, len(nodes))
	for _, n := range nodes {
		n.ID = prefix + n.ID
		if n.Parent == "" {
			n.Parent = connGroupID(connName)
		} else {
			n.Parent = prefix + n.Parent
		}
		n.Conn = connName
		outN = append(outN, n)
	}
	outE := make([]TopoEdge, 0, len(edges))
	for _, e := range edges {
		e.Source = prefix + e.Source
		e.Target = prefix + e.Target
		outE = append(outE, e)
	}
	return outN, outE
}

// renderTopology executes the topology page template with the given graph.
func renderTopology(c echo.Context, connConfig string, nodes []TopoNode, edges []TopoEdge) error {
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
