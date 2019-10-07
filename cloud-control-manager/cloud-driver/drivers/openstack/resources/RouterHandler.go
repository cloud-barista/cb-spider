package resources

import (
	//irs "github.com/cloud-barista/poc-cb-spider/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/rackspace/gophercloud/pagination"
)

/*var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}*/

type OpenStackRouterHandler struct {
	Client *gophercloud.ServiceClient
}

// @TODO: Router 생성 요청 파라미터 정의 필요
type RouterReqInfo struct {
	Name         string
	GateWayId    string
	AdminStateUp bool
}

// @TODO: Router 리소스 프로퍼티 정의 필요
type RouteInfo struct {
	NextHop         string
	DestinationCIDR string
}
type RouterInfo struct {
	Id           string
	Name         string
	TenantId     string
	AdminStateUp bool
	Distributed  bool
	Routes       []RouteInfo
}

// @TODO: Interface 생성 요청 파라미터 정의 필요
type InterfaceReqInfo struct {
	RouterId string
	SubnetId string
}

// @TODO: Interface 리소스 프로퍼티 정의 필요
type InterfaceInfo struct {
	Id   string
	Name string
}

func (routerInfo *RouterInfo) setter(router routers.Router) *RouterInfo {
	routerInfo.Id = router.ID
	routerInfo.Name = router.Name
	router.TenantID = router.TenantID
	router.AdminStateUp = router.AdminStateUp
	router.Distributed = router.Distributed

	var routeArr []RouteInfo

	for _, route := range router.Routes {
		routeInfo := RouteInfo{
			NextHop:         route.NextHop,
			DestinationCIDR: route.DestinationCIDR,
		}
		routeArr = append(routeArr, routeInfo)
	}

	routerInfo.Routes = routeArr

	return routerInfo
}

func (routerHandler *OpenStackRouterHandler) CreateRouter(routerReqInfo RouterReqInfo) (RouterInfo, error) {

	createOpts := routers.CreateOpts{
		Name:         routerReqInfo.Name,
		AdminStateUp: &routerReqInfo.AdminStateUp,
		GatewayInfo: &routers.GatewayInfo{
			NetworkID: routerReqInfo.GateWayId,
		},
	}

	// Create Router
	router, err := routers.Create(routerHandler.Client, createOpts).Extract()
	if err != nil {
		return RouterInfo{}, err
	}

	spew.Dump(router)
	return RouterInfo{Id: router.ID, Name: router.Name}, nil
}

func (routerHandler *OpenStackRouterHandler) ListRouter() ([]*RouterInfo, error) {
	var routerInfoList []*RouterInfo

	pager := routers.List(routerHandler.Client, routers.ListOpts{})
	err := pager.EachPage(func(page pagination.Page) (b bool, e error) {
		// Get Router
		list, err := routers.ExtractRouters(page)
		if err != nil {
			return false, err
		}
		for _, r := range list {
			routerInfo := new(RouterInfo).setter(r)
			routerInfoList = append(routerInfoList, routerInfo)
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	spew.Dump(routerInfoList)
	return nil, nil
}

func (routerHandler *OpenStackRouterHandler) GetRouter(routerID string) (RouterInfo, error) {
	router, err := routers.Get(routerHandler.Client, routerID).Extract()
	if err != nil {
		return RouterInfo{}, err
	}

	routerInfo := new(RouterInfo).setter(*router)

	spew.Dump(routerInfo)
	return RouterInfo{}, nil
}

func (routerHandler *OpenStackRouterHandler) DeleteRouter(routerID string) (bool, error) {
	err := routers.Delete(routerHandler.Client, routerID).ExtractErr()
	if err != nil {
		return false, err
	}
	return true, nil
}

func (routerHandler *OpenStackRouterHandler) AddInterface(interfaceReqInfo InterfaceReqInfo) (InterfaceInfo, error) {
	createOpts := routers.InterfaceOpts{
		SubnetID: interfaceReqInfo.SubnetId,
	}

	// Add Interface
	ir, err := routers.AddInterface(routerHandler.Client, interfaceReqInfo.RouterId, createOpts).Extract()
	if err != nil {
		return InterfaceInfo{}, err
	}

	spew.Dump(ir)
	return InterfaceInfo{}, nil
}

func (routerHandler *OpenStackRouterHandler) DeleteInterface(routerID string, subnetID string) (bool, error) {
	deleteOpts := routers.InterfaceOpts{
		SubnetID: subnetID,
	}

	// Delete Interface
	ir, err := routers.RemoveInterface(routerHandler.Client, routerID, deleteOpts).Extract()
	if err != nil {
		return false, err
	}

	spew.Dump(ir)
	return true, nil
}
