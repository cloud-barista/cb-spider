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
)

//================ Cluster Handler

// ClusterGetOwnerVPCRequest represents the request body for retrieving the owner VPC of a Cluster.
type ClusterGetOwnerVPCRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		CSPId string `json:"CSPId" validate:"required" example:"csp-cluster-1234"`
	} `json:"ReqInfo" validate:"required"`
}

// getClusterOwnerVPC godoc
// @ID get-cluster-owner-vpc
// @Summary Get Cluster Owner VPC
// @Description Retrieve the owner VPC of a specified Cluster.
// @Tags [Cluster Management]
// @Accept  json
// @Produce  json
// @Param ClusterGetOwnerVPCRequest body restruntime.ClusterGetOwnerVPCRequest true "Request body for getting Cluster Owner VPC"
// @Success 200 {object} cres.IID "Details of the owner VPC"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /getclusterowner [post]
func GetClusterOwnerVPC(c echo.Context) error {
	cblog.Info("call GetClusterOwnerVPC()")

	var req ClusterGetOwnerVPCRequest

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

// ClusterRegisterRequest represents the request body for registering a Cluster.
type ClusterRegisterRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		VPCName string `json:"VPCName" validate:"required" example:"vpc-01"`
		Name    string `json:"Name" validate:"required" example:"cluster-01"`
		CSPId   string `json:"CSPId" validate:"required" example:"csp-cluster-1234"`
	} `json:"ReqInfo" validate:"required"`
}

// registerCluster godoc
// @ID register-cluster
// @Summary Register Cluster
// @Description Register a new Cluster with the specified VPC and CSP ID.
// @Tags [Cluster Management]
// @Accept  json
// @Produce  json
// @Param ClusterRegisterRequest body restruntime.ClusterRegisterRequest true "Request body for registering a Cluster"
// @Success 200 {object} cres.ClusterInfo "Details of the registered Cluster"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /regcluster [post]
func RegisterCluster(c echo.Context) error {
	cblog.Info("call RegisterCluster()")

	req := ClusterRegisterRequest{}

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

// unregisterCluster godoc
// @ID unregister-cluster
// @Summary Unregister Cluster
// @Description Unregister a Cluster with the specified name.
// @Tags [Cluster Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for unregistering a Cluster"
// @Param Name path string true "The name of the Cluster to unregister"
// @Success 200 {object} BooleanInfo "Result of the unregister operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /regcluster/{Name} [delete]
func UnregisterCluster(c echo.Context) error {
	cblog.Info("call UnregisterCluster()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, err := cmrt.UnregisterResource(req.ConnectionName, CLUSTER, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// ClusterCreateRequest represents the request body for creating a Cluster.
type ClusterCreateRequest struct {
	ConnectionName  string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	IDTransformMode string `json:"IDTransformMode,omitempty" validate:"omitempty" example:"ON"` // ON: transform CSP ID, OFF: no-transform CSP ID
	ReqInfo         struct {
		Name               string                    `json:"Name" validate:"required" example:"cluster-01"`
		Version            string                    `json:"Version,omitempty" validate:"omitempty" example:"1.30"` // Some CSPs may not support or limit versions.
		VPCName            string                    `json:"VPCName" validate:"required" example:"vpc-01"`
		SubnetNames        []string                  `json:"SubnetNames" validate:"required" example:"subnet-01,subnet-02"`
		SecurityGroupNames []string                  `json:"SecurityGroupNames" validate:"required" example:"sg-01,sg-02"`
		NodeGroupList      []ClusterNodeGroupRequest `json:"NodeGroupList" validate:"omitempty"`
		TagList            []cres.KeyValue           `json:"TagList,omitempty" validate:"omitempty"`
	} `json:"ReqInfo" validate:"required"`
}

// ClusterNodeGroupRequest represents the request body for a Node Group in a Cluster.
type ClusterNodeGroupRequest struct {
	Name            string `json:"Name" validate:"required" example:"nodegroup-01"`
	ImageName       string `json:"ImageName" validate:"omitempty"`              // Some CSPs may not support or limit images. [Ref](https://docs.google.com/spreadsheets/d/1mPmfnfmyszYimVzplZMzsqO3WsBmOdes/edit?usp=sharing&ouid=108635813398159139552&rtpof=true&sd=true)
	VMSpecName      string `json:"VMSpecName" validate:"omitempty"`             // Some CSPs may not support or limit specs. [Ref](https://docs.google.com/spreadsheets/d/1mPmfnfmyszYimVzplZMzsqO3WsBmOdes/edit?usp=sharing&ouid=108635813398159139552&rtpof=true&sd=true)
	RootDiskType    string `json:"RootDiskType,omitempty" validate:"omitempty"` // Some CSPs may not support or limit types. [Ref](https://docs.google.com/spreadsheets/d/1mPmfnfmyszYimVzplZMzsqO3WsBmOdes/edit?usp=sharing&ouid=108635813398159139552&rtpof=true&sd=true)
	RootDiskSize    string `json:"RootDiskSize,omitempty" validate:"omitempty"` // Some CSPs may not support or limit sizes. [Ref](https://docs.google.com/spreadsheets/d/1mPmfnfmyszYimVzplZMzsqO3WsBmOdes/edit?usp=sharing&ouid=108635813398159139552&rtpof=true&sd=true)
	KeyPairName     string `json:"KeyPairName" validate:"required" example:"keypair-01"`
	OnAutoScaling   string `json:"OnAutoScaling" validate:"required" example:"true"`
	DesiredNodeSize string `json:"DesiredNodeSize" validate:"required" example:"2"`
	MinNodeSize     string `json:"MinNodeSize" validate:"required" example:"1"`
	MaxNodeSize     string `json:"MaxNodeSize" validate:"required" example:"3"`
}

// createCluster godoc
// @ID create-cluster
// @Summary Create Cluster
// @Description Create a new Cluster with specified configurations. 🕷️ [[Concept Guide](https://github.com/cloud-barista/cb-spider/wiki/Provider-Managed-Kubernetes-and-Driver-API)] <br> * NodeGroupList is optional, depends on CSP type: <br> &nbsp;- Type-I (e.g., Tencent, Alibaba): requires separate Node Group addition after Cluster creation. <br> &nbsp;- Type-II (e.g., Azure, NHN): mandates at least one Node Group during initial Cluster creation.
// @Tags [Cluster Management]
// @Accept  json
// @Produce  json
// @Param ClusterCreateRequest body restruntime.ClusterCreateRequest true "Request body for creating a Cluster"
// @Success 200 {object} cres.ClusterInfo "Details of the created Cluster"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /cluster [post]
func CreateCluster(c echo.Context) error {
	cblog.Info("call CreateCluster()")

	req := ClusterCreateRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Rest RegInfo => Driver ReqInfo
	reqInfo := cres.ClusterInfo{
		IId:     cres.IID{req.ReqInfo.Name, req.ReqInfo.Name},
		Version: req.ReqInfo.Version,
		Network: cres.NetworkInfo{
			VpcIID:            cres.IID{req.ReqInfo.VPCName, ""},
			SubnetIIDs:        convertIIDs(req.ReqInfo.SubnetNames),
			SecurityGroupIIDs: convertIIDs(req.ReqInfo.SecurityGroupNames),
		},
		NodeGroupList: convertNodeGroupList(req.ReqInfo.NodeGroupList),
		TagList:       req.ReqInfo.TagList,
	}

	// Call common-runtime API
	result, err := cmrt.CreateCluster(req.ConnectionName, CLUSTER, reqInfo, req.IDTransformMode)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// ClusterListResponse represents the response body for listing Clusters.
type ClusterListResponse struct {
	Result []*cres.ClusterInfo `json:"cluster" validate:"required"`
}

// listCluster godoc
// @ID list-cluster
// @Summary List Clusters
// @Description Retrieve a list of Clusters associated with a specific connection.
// @Tags [Cluster Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to list Clusters for"
// @Success 200 {object} ClusterListResponse "List of Clusters"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /cluster [get]
func ListCluster(c echo.Context) error {
	cblog.Info("call ListCluster()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	result, err := cmrt.ListCluster(req.ConnectionName, CLUSTER)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	jsonResult := ClusterListResponse{
		Result: result,
	}

	return c.JSON(http.StatusOK, &jsonResult)
}

// listAllCluster godoc
// @ID list-all-cluster
// @Summary List All Clusters in a Connection
// @Description Retrieve a comprehensive list of all Clusters associated with a specific connection, <br> including those mapped between CB-Spider and the CSP, <br> only registered in CB-Spider's metadata, <br> and only existing in the CSP.
// @Tags [Cluster Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to list Clusters for"
// @Success 200 {object} AllResourceListResponse "List of all Clusters within the specified connection, including clusters in CB-Spider only, CSP only, and mapped between both."
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /allcluster [get]
func ListAllCluster(c echo.Context) error {
	cblog.Info("call ListAllCluster()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	allResourceList, err := cmrt.ListAllResource(req.ConnectionName, CLUSTER)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &allResourceList)
}

// getCluster godoc
// @ID get-cluster
// @Summary Get Cluster
// @Description Retrieve details of a specific Cluster.
// @Tags [Cluster Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to get a Cluster for"
// @Param Name path string true "The name of the Cluster to retrieve"
// @Success 200 {object} cres.ClusterInfo "Details of the Cluster"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /cluster/{Name} [get]
func GetCluster(c echo.Context) error {
	cblog.Info("call GetCluster()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	clusterName := c.Param("Name")

	// Call common-runtime API
	result, err := cmrt.GetCluster(req.ConnectionName, CLUSTER, clusterName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// ClusterAddNodeGroupRequest represents the request body for adding a Node Group to a Cluster.
type ClusterAddNodeGroupRequest struct {
	ConnectionName  string                  `json:"ConnectionName" validate:"required" example:"aws-connection"`
	IDTransformMode string                  `json:"IDTransformMode,omitempty" validate:"omitempty" example:"ON"` // ON: transform CSP ID, OFF: no-transform CSP ID
	ReqInfo         ClusterNodeGroupRequest `json:"ReqInfo" validate:"required"`
}

// addNodeGroup godoc
// @ID add-nodegroup
// @Summary Add Node Group
// @Description Add a new Node Group to an existing Cluster.
// @Tags [Cluster Management]
// @Accept  json
// @Produce  json
// @Param Name path string true "The name of the Cluster to add the Node Group to"
// @Param ClusterAddNodeGroupRequest body restruntime.ClusterAddNodeGroupRequest true "Request body for adding a Node Group"
// @Success 200 {object} cres.ClusterInfo "Details of the Cluster including the added Node Group"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /cluster/{Name}/nodegroup [post]
func AddNodeGroup(c echo.Context) error {
	cblog.Info("call AddNodeGroup()")

	req := ClusterAddNodeGroupRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	reqInfo := cres.NodeGroupInfo{
		IId:          cres.IID{req.ReqInfo.Name, ""},
		ImageIID:     cres.IID{req.ReqInfo.ImageName, ""},
		VMSpecName:   req.ReqInfo.VMSpecName,
		RootDiskType: req.ReqInfo.RootDiskType,
		RootDiskSize: req.ReqInfo.RootDiskSize,
		KeyPairIID:   cres.IID{req.ReqInfo.KeyPairName, ""},

		OnAutoScaling:   func() bool { on, _ := strconv.ParseBool(req.ReqInfo.OnAutoScaling); return on }(),
		DesiredNodeSize: func() int { size, _ := strconv.Atoi(req.ReqInfo.DesiredNodeSize); return size }(),
		MinNodeSize:     func() int { size, _ := strconv.Atoi(req.ReqInfo.MinNodeSize); return size }(),
		MaxNodeSize:     func() int { size, _ := strconv.Atoi(req.ReqInfo.MaxNodeSize); return size }(),
	}

	clusterName := c.Param("Name")

	// Call common-runtime API
	result, err := cmrt.AddNodeGroup(req.ConnectionName, NODEGROUP, clusterName, reqInfo, req.IDTransformMode)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// removeNodeGroup godoc
// @ID remove-nodegroup
// @Summary Remove Node Group
// @Description Remove an existing Node Group from a Cluster.
// @Tags [Cluster Management]
// @Accept  json
// @Produce  json
// @Param Name path string true "The name of the Cluster to remove the Node Group to"
// @Param NodeGroupName path string true "The name of the Node Group to remove"
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for removing a Node Group"
// @Success 200 {object} BooleanInfo "Result of the remove operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /cluster/{Name}/nodegroup/{NodeGroupName} [delete]
func RemoveNodeGroup(c echo.Context) error {
	cblog.Info("call RemoveNodeGroup()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	clusterName := c.Param("Name")

	// Call common-runtime API
	result, err := cmrt.RemoveNodeGroup(req.ConnectionName, clusterName, c.Param("NodeGroupName"), c.QueryParam("force"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// ClusterSetNodeGroupAutoScalingRequest represents the request body for setting auto-scaling for a Node Group in a Cluster.
type ClusterSetNodeGroupAutoScalingRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		OnAutoScaling string `json:"OnAutoScaling" validate:"required" example:"true"`
	} `json:"ReqInfo" validate:"required"`
}

// setNodeGroupAutoScaling godoc
// @ID set-nodegroup-autoscaling
// @Summary Set Node Group Auto Scaling
// @Description Enable or disable auto scaling for a Node Group in a Cluster.
// @Tags [Cluster Management]
// @Accept  json
// @Produce  json
// @Param Name path string true "The name of the Cluster to set Node Group Auto Scaling"
// @Param NodeGroupName path string true "The name of the Node Group"
// @Param ClusterSetNodeGroupAutoScalingRequest body restruntime.ClusterSetNodeGroupAutoScalingRequest true "Request body for setting auto scaling for a Node Group"
// @Success 200 {object} BooleanInfo "Result of the auto scaling operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /cluster/{Name}/nodegroup/{NodeGroupName}/onautoscaling [put]
func SetNodeGroupAutoScaling(c echo.Context) error {
	cblog.Info("call SetNodeGroupAutoScaling()")

	req := ClusterSetNodeGroupAutoScalingRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	clusterName := c.Param("Name")

	// Call common-runtime API
	on, _ := strconv.ParseBool(req.ReqInfo.OnAutoScaling)
	result, err := cmrt.SetNodeGroupAutoScaling(req.ConnectionName, clusterName,
		c.Param("NodeGroupName"), on)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// ClusterChangeNodeGroupScalingRequest represents the request body for changing the scaling settings of a Node Group in a Cluster.
type ClusterChangeNodeGroupScalingRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		DesiredNodeSize string `json:"DesiredNodeSize" validate:"required" example:"3"`
		MinNodeSize     string `json:"MinNodeSize" validate:"required" example:"1"`
		MaxNodeSize     string `json:"MaxNodeSize" validate:"required" example:"5"`
	} `json:"ReqInfo" validate:"required"`
}

// changeNodeGroupScaling godoc
// @ID change-nodegroup-scaling
// @Summary Change Node Group Scaling
// @Description Change the scaling settings for a Node Group in a Cluster.
// @Tags [Cluster Management]
// @Accept  json
// @Produce  json
// @Param Name path string true "The name of the Cluster to change Node Group Scaling"
// @Param NodeGroupName path string true "The name of the Node Group"
// @Param ClusterChangeNodeGroupScalingRequest body restruntime.ClusterChangeNodeGroupScalingRequest true "Request body for changing Node Group scaling"
// @Success 200 {object} cres.NodeGroupInfo "Details of the updated Node Group"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /cluster/{Name}/nodegroup/{NodeGroupName}/autoscalesize [put]
func ChangeNodeGroupScaling(c echo.Context) error {
	cblog.Info("call ChangeNodeGroupScaling()")

	req := ClusterChangeNodeGroupScalingRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	clusterName := c.Param("Name")

	// Call common-runtime API
	desiredNodeSize, _ := strconv.Atoi(req.ReqInfo.DesiredNodeSize)
	minNodeSize, _ := strconv.Atoi(req.ReqInfo.MinNodeSize)
	maxNodeSize, _ := strconv.Atoi(req.ReqInfo.MaxNodeSize)
	result, err := cmrt.ChangeNodeGroupScaling(req.ConnectionName, clusterName,
		c.Param("NodeGroupName"), desiredNodeSize, minNodeSize, maxNodeSize)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// deleteCluster godoc
// @ID delete-cluster
// @Summary Delete Cluster
// @Description Delete a specified Cluster.
// @Tags [Cluster Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for deleting a Cluster"
// @Param Name path string true "The name of the Cluster to delete"
// @Param force query string false "Force delete the Cluster. ex) true or false(default: false)"
// @Success 200 {object} BooleanInfo "Result of the delete operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /cluster/{Name} [delete]
func DeleteCluster(c echo.Context) error {
	cblog.Info("call DeleteCluster()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	clusterName := c.Param("Name")

	// Call common-runtime API
	result, err := cmrt.DeleteCluster(req.ConnectionName, CLUSTER, clusterName, c.QueryParam("force"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// deleteCSPCluster godoc
// @ID delete-csp-cluster
// @Summary Delete CSP Cluster
// @Description Delete a specified CSP Cluster.
// @Tags [Cluster Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for deleting a CSP Cluster"
// @Param Id path string true "The CSP Cluster ID to delete"
// @Success 200 {object} BooleanInfo "Result of the delete operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /cspcluster/{Id} [delete]
func DeleteCSPCluster(c echo.Context) error {
	cblog.Info("call DeleteCSPCluster()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, _, err := cmrt.DeleteCSPResource(req.ConnectionName, CLUSTER, c.Param("Id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// ClusterUpgradeRequest represents the request body for upgrading a Cluster to a specified version.
type ClusterUpgradeRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		Version string `json:"Version" validate:"required" example:"1.30"`
	} `json:"ReqInfo" validate:"required"`
}

// upgradeCluster godoc
// @ID upgrade-cluster
// @Summary Upgrade Cluster
// @Description Upgrade a Cluster to a specified version.
// @Tags [Cluster Management]
// @Accept  json
// @Produce  json
// @Param Name path string true "The name of the Cluster to upgrade"
// @Param ClusterUpgradeRequest body restruntime.ClusterUpgradeRequest true "Request body for upgrading a Cluster"
// @Success 200 {object} cres.ClusterInfo "Details of the upgraded Cluster"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /cluster/{Name}/upgrade [put]
func UpgradeCluster(c echo.Context) error {
	cblog.Info("call UpgradeCluster()")

	req := ClusterUpgradeRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	clusterName := c.Param("Name")

	// Call common-runtime API
	result, err := cmrt.UpgradeCluster(req.ConnectionName, clusterName, req.ReqInfo.Version)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// countAllClusters godoc
// @ID count-all-cluster
// @Summary Count All Clusters
// @Description Get the total number of Clusters across all connections.
// @Tags [Cluster Management]
// @Produce  json
// @Success 200 {object} CountResponse "Total count of Clusters"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /countcluster [get]
func CountAllClusters(c echo.Context) error {
	// Call common-runtime API to get count of Clusters
	count, err := cmrt.CountAllClusters()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	jsonResult := CountResponse{
		Count: int(count),
	}

	// Return JSON response
	return c.JSON(http.StatusOK, jsonResult)
}

// countClustersByConnection godoc
// @ID count-cluster-by-connection
// @Summary Count Clusters by Connection
// @Description Get the total number of Clusters for a specific connection.
// @Tags [Cluster Management]
// @Produce  json
// @Param ConnectionName path string true "The name of the Connection"
// @Success 200 {object} CountResponse "Total count of Clusters for the connection"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /countcluster/{ConnectionName} [get]
func CountClustersByConnection(c echo.Context) error {
	// Call common-runtime API to get count of Clusters
	count, err := cmrt.CountClustersByConnection(c.Param("ConnectionName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	jsonResult := CountResponse{
		Count: int(count),
	}

	// Return JSON response
	return c.JSON(http.StatusOK, jsonResult)
}

//================ Helper Functions

func convertIIDs(names []string) []cres.IID {
	IIDs := []cres.IID{}
	for _, name := range names {
		IIDs = append(IIDs, cres.IID{name, ""})
	}
	return IIDs
}

func convertNodeGroupList(nodeGroupReqList []ClusterNodeGroupRequest) []cres.NodeGroupInfo {
	nodeGroupInfoList := []cres.NodeGroupInfo{}
	for _, ngReq := range nodeGroupReqList {
		nodeGroupInfoList = append(nodeGroupInfoList, convertNodeGroup(ngReq))
	}
	return nodeGroupInfoList
}

func convertNodeGroup(ngReq ClusterNodeGroupRequest) cres.NodeGroupInfo {

	nodeGroupInfo := cres.NodeGroupInfo{
		IId:          cres.IID{ngReq.Name, ""},
		ImageIID:     cres.IID{ngReq.ImageName, ""},
		VMSpecName:   ngReq.VMSpecName,
		RootDiskType: ngReq.RootDiskType,
		RootDiskSize: ngReq.RootDiskSize,
		KeyPairIID:   cres.IID{ngReq.KeyPairName, ""},

		OnAutoScaling:   func() bool { on, _ := strconv.ParseBool(ngReq.OnAutoScaling); return on }(),
		DesiredNodeSize: func() int { size, _ := strconv.Atoi(ngReq.DesiredNodeSize); return size }(),
		MinNodeSize:     func() int { size, _ := strconv.Atoi(ngReq.MinNodeSize); return size }(),
		MaxNodeSize:     func() int { size, _ := strconv.Atoi(ngReq.MaxNodeSize); return size }(),
	}
	return nodeGroupInfo
}
