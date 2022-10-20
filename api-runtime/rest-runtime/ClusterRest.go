// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2022.10.

package restruntime

import (

        cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
        cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

        // REST API (echo)
        "net/http"

        "github.com/labstack/echo/v4"

        "strconv"
        "strings"
)


//================ Cluster Handler

func GetClusterOwnerVPC(c echo.Context) error {
        cblog.Info("call GetClusterOwnerVPC()")

        var req struct {
                ConnectionName string
                ReqInfo        struct {
                        CSPId          string
                }
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, err := cmrt.GetClusterOwnerVPC(req.ConnectionName, req.ReqInfo.CSPId)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

type ClusterRegisterReq struct {
        ConnectionName string
        ReqInfo        struct {
                VPCName           string
                Name           string
                CSPId          string
        }
}

func RegisterCluster(c echo.Context) error {
        cblog.Info("call RegisterCluster()")

        req := ClusterRegisterReq{}

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // create UserIID
        userIId := cres.IID{req.ReqInfo.Name, req.ReqInfo.CSPId}

        // Call common-runtime API
        result, err := cmrt.RegisterCluster(req.ConnectionName, req.ReqInfo.VPCName, userIId)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

// (1) get args from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func UnregisterCluster(c echo.Context) error {
        cblog.Info("call UnregisterCluster()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, err := cmrt.UnregisterResource(req.ConnectionName, rsCluster, c.Param("Name"))
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

        return c.JSON(http.StatusOK, &resultInfo)
}

type ClusterReq struct {
        NameSpace string
        ConnectionName string
        ReqInfo        struct {
		// (1) Cluster Info
                Name		string
                Version		string

		// (2) Network Info
                VPCName			string
                SubnetNames		[]string
                SecurityGroupNames	[]string

		// (3) NodeGroupInfo List
		NodeGroupList	        []NodeGroupReq
        }
}

type NodeGroupReq struct {
        Name			string 
        ImageName		string 
        VMSpecName		string 
        RootDiskType		string 
        RootDiskSize		string 
        KeyPairName		string 

	// autoscale config.
        OnAutoScaling		string 
        DesiredNodeSize		string 
        MinNodeSize		string 
        MaxNodeSize		string 
}

func CreateCluster(c echo.Context) error {
        cblog.Info("call CreateCluster()")

        req := ClusterReq{}

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Rest RegInfo => Driver ReqInfo
        reqInfo := cres.ClusterInfo{
		// (1) Cluster Info
                IId:           cres.IID{req.ReqInfo.Name, req.ReqInfo.Name}, 
                Version:       req.ReqInfo.Version,

		// (2) Network Info
		Network:	cres.NetworkInfo {
					VpcIID:        		cres.IID{req.ReqInfo.VPCName, ""},
					SubnetIIDs:		convertIIDs(req.ReqInfo.SubnetNames), 
					SecurityGroupIIDs:	convertIIDs(req.ReqInfo.SecurityGroupNames),
				}, 
		// (3) NodeGroup Info List
                NodeGroupList: 	convertNodeGroupList(req.ReqInfo.NodeGroupList),
        }

        // Resource Name has namespace prefix when from Tumblebug
        if req.NameSpace != "" {                
                attachNameSpaceToName(req.NameSpace, &reqInfo)
        }

        // Call common-runtime API
        result, err := cmrt.CreateCluster(req.ConnectionName, rsCluster, reqInfo)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Resource Name has namespace prefix when from Tumblebug
        if req.NameSpace != "" {                
                detachNameSpaceFromName(req.NameSpace, result)
        }

        return c.JSON(http.StatusOK, result)
}

func convertIIDs(names []string) []cres.IID {
	IIDs:= []cres.IID{}
	for _, name := range names {
		IIDs = append(IIDs, cres.IID{name, ""})	
	}
	return IIDs
}

func convertNodeGroupList(nodeGroupReqList []NodeGroupReq) []cres.NodeGroupInfo {
	nodeGroupInfoList := []cres.NodeGroupInfo{}
	for _, ngReq := range nodeGroupReqList {		
		nodeGroupInfoList = append(nodeGroupInfoList, convertNodeGroup(ngReq))
	}
	return nodeGroupInfoList
}

func convertNodeGroup(ngReq NodeGroupReq) cres.NodeGroupInfo {
        
        nodeGroupInfo := cres.NodeGroupInfo {
                                IId:    cres.IID{ngReq.Name, ""},
                                ImageIID:       cres.IID{ngReq.ImageName, ""},
                                VMSpecName:     ngReq.VMSpecName,
                                RootDiskType:   ngReq.RootDiskType,
                                RootDiskSize:   ngReq.RootDiskSize,
                                KeyPairIID:     cres.IID{ngReq.KeyPairName, ""},

                                OnAutoScaling:  func() bool { on, _ := strconv.ParseBool(ngReq.OnAutoScaling); return on }(),
                                DesiredNodeSize: func() int { size, _ := strconv.Atoi(ngReq.DesiredNodeSize); return size }(),
                                MinNodeSize: func() int { size, _ := strconv.Atoi(ngReq.MinNodeSize); return size }(),
                                MaxNodeSize: func() int { size, _ := strconv.Atoi(ngReq.MaxNodeSize); return size }(),
                        }        
        return nodeGroupInfo
}

// Resource Name has namespace prefix when from Tumblebug
func attachNameSpaceToName(nameSpace string, clusterInfo *cres.ClusterInfo) {
        nameSpace += "-"

        // (1) Network's VpcIID
        clusterInfo.Network.VpcIID.NameId = nameSpace + clusterInfo.Network.VpcIID.NameId

        // (2) Network's SubnetIIDs
        for idx, _ := range clusterInfo.Network.SubnetIIDs {
                clusterInfo.Network.SubnetIIDs[idx].NameId = nameSpace + clusterInfo.Network.SubnetIIDs[idx].NameId
        }

        // (3) Network's SecurityGroupsIIDs
        for idx, _ := range clusterInfo.Network.SubnetIIDs {
                clusterInfo.Network.SubnetIIDs[idx].NameId = nameSpace + clusterInfo.Network.SubnetIIDs[idx].NameId
        }

        // (4) NodeGroup's KeyPairIID
        for idx, _ := range clusterInfo.NodeGroupList {
                clusterInfo.NodeGroupList[idx].KeyPairIID.NameId = nameSpace + clusterInfo.NodeGroupList[idx].KeyPairIID.NameId
        }
}

func ListCluster(c echo.Context) error {
        cblog.Info("call ListCluster()")

        var req struct {
                NameSpace string
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // To support for Get-Query Param Type API
        if req.ConnectionName == "" {
                req.ConnectionName = c.QueryParam("ConnectionName")
        }

        // Call common-runtime API
        result, err := cmrt.ListCluster(req.ConnectionName, rsCluster)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // To support for Get-Query Param Type API
        if req.NameSpace == "" {
                req.NameSpace = c.QueryParam("NameSpace")
        }

        // Resource Name has namespace prefix when from Tumblebug
        if req.NameSpace != "" {
                for _, clusterInfo := range result {
                        detachNameSpaceFromName(req.NameSpace, clusterInfo)
                }
        }

        var jsonResult struct {
                Result []*cres.ClusterInfo `json:"cluster"`
        }
        jsonResult.Result = result
        return c.JSON(http.StatusOK, &jsonResult)
}

// Resource Name has namespace prefix when from Tumblebug
func detachNameSpaceFromName(nameSpace string, clusterInfo *cres.ClusterInfo) {
        nameSpace += "-"

        // (1) Network's VpcIID
        clusterInfo.Network.VpcIID.NameId = strings.Replace(clusterInfo.Network.VpcIID.NameId, nameSpace, "", 1)

        // (2) Network's SubnetIIDs
        for idx, _ := range clusterInfo.Network.SubnetIIDs {
                clusterInfo.Network.SubnetIIDs[idx].NameId = 
                        strings.Replace(clusterInfo.Network.SubnetIIDs[idx].NameId, nameSpace, "", 1)
        }

        // (3) Network's SecurityGroupsIIDs
        for idx, _ := range clusterInfo.Network.SecurityGroupIIDs {
                clusterInfo.Network.SecurityGroupIIDs[idx].NameId = 
                        strings.Replace(clusterInfo.Network.SecurityGroupIIDs[idx].NameId, nameSpace, "", 1)
        }

        // (4) NodeGroup's KeyPairIID
        for idx, _ := range clusterInfo.NodeGroupList {
                clusterInfo.NodeGroupList[idx].KeyPairIID.NameId = 
                        strings.Replace(clusterInfo.NodeGroupList[idx].KeyPairIID.NameId, nameSpace,"", 1)
        }
}

// list all Clusters for management
// (1) get args from REST Call
// (2) get all Cluster List by common-runtime API
// (3) return REST Json Format
func ListAllCluster(c echo.Context) error {
        cblog.Info("call ListAllCluster()")

        var req struct {
                NameSpace string                
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // To support for Get-Query Param Type API
        if req.ConnectionName == "" {
                req.ConnectionName = c.QueryParam("ConnectionName")
        }

        // Call common-runtime API
        allResourceList, err := cmrt.ListAllResource(req.ConnectionName, rsCluster)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // To support for Get-Query Param Type API
        if req.NameSpace == "" {
                req.NameSpace = c.QueryParam("NameSpace")
        }

        // Resource Name has namespace prefix when from Tumblebug
        if req.NameSpace != "" {
                nameSpace := req.NameSpace + "-"
                for idx, IID := range allResourceList.AllList.MappedList {
                        if IID.NameId != "" {
                                allResourceList.AllList.MappedList[idx].NameId = strings.Replace(IID.NameId, nameSpace, "", 1)
                        }                        
                }                
                for idx, IID := range allResourceList.AllList.OnlySpiderList {
                        if IID.NameId != "" {
                                allResourceList.AllList.OnlySpiderList[idx].NameId = strings.Replace(IID.NameId, nameSpace, "", 1)
                        }                        
                }
                for idx, IID := range allResourceList.AllList.OnlyCSPList {
                        if IID.NameId != "" {
                                allResourceList.AllList.OnlyCSPList[idx].NameId = strings.Replace(IID.NameId, nameSpace, "", 1)
                        }                        
                }
        }

        return c.JSON(http.StatusOK, &allResourceList)
}

func GetCluster(c echo.Context) error {
        cblog.Info("call GetCluster()")

        var req struct {
                NameSpace string
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // To support for Get-Query Param Type API
        if req.ConnectionName == "" {
                req.ConnectionName = c.QueryParam("ConnectionName")
        }

        // Call common-runtime API
        result, err := cmrt.GetCluster(req.ConnectionName, rsCluster, c.Param("Name"))
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // To support for Get-Query Param Type API
        if req.NameSpace == "" {
                req.NameSpace = c.QueryParam("NameSpace")
        }

        // Resource Name has namespace prefix when from Tumblebug
        if req.NameSpace != "" {
                detachNameSpaceFromName(req.NameSpace, result)
        }

        return c.JSON(http.StatusOK, result)
}

func AddNodeGroup(c echo.Context) error {
        cblog.Info("call AddNodeGroup()")

        var req struct {
                NameSpace string
                ConnectionName string
                ReqInfo        struct {
                        Name                    string 
                        ImageName               string 
                        VMSpecName              string 
                        RootDiskType            string 
                        RootDiskSize            string 
                        KeyPairName             string 

                        // autoscale config.
                        OnAutoScaling           string 
                        DesiredNodeSize         string 
                        MinNodeSize             string 
                        MaxNodeSize             string
                }
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        reqInfo := cres.NodeGroupInfo {
                                IId:    cres.IID{req.ReqInfo.Name, ""},
                                ImageIID:       cres.IID{req.ReqInfo.ImageName, ""},
                                VMSpecName:     req.ReqInfo.VMSpecName,
                                RootDiskType:   req.ReqInfo.RootDiskType,
                                RootDiskSize:   req.ReqInfo.RootDiskSize,
                                KeyPairIID:     cres.IID{req.ReqInfo.KeyPairName, ""},

                                OnAutoScaling:  func() bool { on, _ := strconv.ParseBool(req.ReqInfo.OnAutoScaling); return on }(),
                                DesiredNodeSize: func() int { size, _ := strconv.Atoi(req.ReqInfo.DesiredNodeSize); return size }(),
                                MinNodeSize: func() int { size, _ := strconv.Atoi(req.ReqInfo.MinNodeSize); return size }(),
                                MaxNodeSize: func() int { size, _ := strconv.Atoi(req.ReqInfo.MaxNodeSize); return size }(),
                        }

        // Resource Name has namespace prefix when from Tumblebug
        if req.NameSpace != "" {
                nameSpace := req.NameSpace + "-"
                // NodeGroup's KeyPairIID                
                reqInfo.KeyPairIID.NameId = nameSpace + reqInfo.KeyPairIID.NameId
        }

        // Call common-runtime API
        result, err := cmrt.AddNodeGroup(req.ConnectionName, rsNodeGroup, c.Param("Name"), reqInfo)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Resource Name has namespace prefix when from Tumblebug
        if req.NameSpace != "" {
                detachNameSpaceFromName(req.NameSpace, result)
        }

        return c.JSON(http.StatusOK, result)
}

func RemoveNodeGroup(c echo.Context) error {
        cblog.Info("call RemoveNodeGroup()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, err := cmrt.RemoveNodeGroup(req.ConnectionName, c.Param("Name"), c.Param("NodeGroupName"), c.QueryParam("force"))
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

func SetNodeGroupAutoScaling(c echo.Context) error {
        cblog.Info("call SetNodeGroupAutoScaling()")

        var req struct {
                ConnectionName string
                ReqInfo        struct {
                        OnAutoScaling      string
                }
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }
        
        // Call common-runtime API
        on, _ := strconv.ParseBool(req.ReqInfo.OnAutoScaling)
        result, err := cmrt.SetNodeGroupAutoScaling(req.ConnectionName, c.Param("Name"), 
                        c.Param("NodeGroupName"), on)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

func ChangeNodeGroupScaling(c echo.Context) error {
        cblog.Info("call ChangeNodeGroupScaling()")

        var req struct {
                NameSpace string
                ConnectionName string
                ReqInfo        struct {
                        DesiredNodeSize      string
                        MinNodeSize      string
                        MaxNodeSize      string
                }
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        desiredNodeSize, _ := strconv.Atoi(req.ReqInfo.DesiredNodeSize)
        minNodeSize, _ := strconv.Atoi(req.ReqInfo.MinNodeSize)
        maxNodeSize, _ := strconv.Atoi(req.ReqInfo.MaxNodeSize)
        result, err := cmrt.ChangeNodeGroupScaling(req.ConnectionName, c.Param("Name"), 
                        c.Param("NodeGroupName"), desiredNodeSize, minNodeSize, maxNodeSize)                        
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Resource Name has namespace prefix when from Tumblebug
        if req.NameSpace != "" {
                nameSpace := req.NameSpace + "-"
                result.KeyPairIID.NameId = 
                        strings.Replace(result.KeyPairIID.NameId, nameSpace,"", 1)
        }

        return c.JSON(http.StatusOK, result)
}

// (1) get args from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func DeleteCluster(c echo.Context) error {
        cblog.Info("call DeleteCluster()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, _, err := cmrt.DeleteResource(req.ConnectionName, rsCluster, c.Param("Name"), c.QueryParam("force"))
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

        return c.JSON(http.StatusOK, &resultInfo)
}

// (1) get args from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func DeleteCSPCluster(c echo.Context) error {
        cblog.Info("call DeleteCSPCluster()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, _, err := cmrt.DeleteCSPResource(req.ConnectionName, rsCluster, c.Param("Id"))
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

        return c.JSON(http.StatusOK, &resultInfo)
}


func UpgradeCluster(c echo.Context) error {
        cblog.Info("call UpgradeCluster()")

        var req struct {
                NameSpace string
                ConnectionName string
                ReqInfo        struct {
                        Version      string
                }
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, err := cmrt.UpgradeCluster(req.ConnectionName, c.Param("Name"), req.ReqInfo.Version)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Resource Name has namespace prefix when from Tumblebug
        if req.NameSpace != "" {
                detachNameSpaceFromName(req.NameSpace, &result)
        }
        
        return c.JSON(http.StatusOK, result)
}