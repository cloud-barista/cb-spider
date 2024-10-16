// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by CB-Spider Team, 2022.05.

package resources

import "time"

// -------- Info Structure
// NLBInfo represents the details of a Network Load Balancer (NLB).
// @description Network Load Balancer (NLB) Information
type NLBInfo struct {
	IId    IID `json:"IId" validate:"required"`
	VpcIID IID `json:"VpcIID" validate:"required"` // Owner VPC IID

	Type  string `json:"Type" validate:"required" example:"PUBLIC"`  // PUBLIC(V) | INTERNAL
	Scope string `json:"Scope" validate:"required" example:"REGION"` // REGION(V) | GLOBAL

	//------ Frontend
	Listener ListenerInfo `json:"Listener" validate:"required"`

	//------ Backend
	VMGroup       VMGroupInfo       `json:"VMGroup" validate:"required"`
	HealthChecker HealthCheckerInfo `json:"HealthChecker" validate:"required"`

	CreatedTime  time.Time  `json:"CreatedTime" validate:"required" example:"2024-08-27T10:00:00Z"`
	TagList      []KeyValue `json:"TagList,omitempty" validate:"omitempty"`
	KeyValueList []KeyValue `json:"KeyValueList,omitempty" validate:"omitempty"`
}

// ListenerInfo represents the frontend listener configuration for an NLB.
// @description Listener Information for a Network Load Balancer (NLB)
type ListenerInfo struct {
	Protocol string `json:"Protocol" validate:"required" example:"TCP"` // TCP|UDP
	IP       string `json:"IP" validate:"omitempty" example:"192.168.0.1"`
	Port     string `json:"Port" validate:"required" example:"80"` // 1-65535
	DNSName  string `json:"DNSName" validate:"omitempty" example:"nlb.example.com"`

	CspID        string     `json:"CspID,omitempty" validate:"omitempty"`
	KeyValueList []KeyValue `json:"KeyValueList,omitempty" validate:"omitempty"`
}

// VMGroupInfo represents the backend VM group configuration for an NLB.
// @description VM Group Information for a Network Load Balancer (NLB)
type VMGroupInfo struct {
	Protocol string `json:"Protocol" validate:"required" example:"TCP"` // TCP|UDP
	Port     string `json:"Port" validate:"required" example:"8080"`    // 1-65535
	VMs      *[]IID `json:"VMs" validate:"required"`

	CspID        string     `json:"CspID,omitempty" validate:"omitempty"`
	KeyValueList []KeyValue `json:"KeyValueList,omitempty" validate:"omitempty"`
}

// HealthCheckerInfo represents the health check configuration for an NLB.
// @description Health Checker Information for a Network Load Balancer (NLB)
type HealthCheckerInfo struct {
	Protocol  string `json:"Protocol" validate:"required" example:"TCP"` // TCP|HTTP
	Port      string `json:"Port" validate:"required" example:"80"`      // Listener Port or 1-65535
	Interval  int    `json:"Interval" validate:"required" example:"30"`  // secs, Interval time between health checks.
	Timeout   int    `json:"Timeout" validate:"required" example:"5"`    // secs, Waiting time to decide an unhealthy VM when no response.
	Threshold int    `json:"Threshold" validate:"required" example:"3"`  // num, The number of continuous health checks to change the VM status.

	CspID        string     `json:"CspID,omitempty" validate:"omitempty"`
	KeyValueList []KeyValue `json:"KeyValueList,omitempty" validate:"omitempty"`
}

// HealthInfo represents the health status of the VM group in an NLB.
// @description Health Information for a Network Load Balancer (NLB)
type HealthInfo struct {
	AllVMs       *[]IID `json:"AllVMs" validate:"required"`
	HealthyVMs   *[]IID `json:"HealthyVMs" validate:"required"`
	UnHealthyVMs *[]IID `json:"UnHealthyVMs" validate:"required"`
}

// -------- API
type NLBHandler interface {

	//------ NLB Management
	ListIID() ([]*IID, error)
	CreateNLB(nlbReqInfo NLBInfo) (NLBInfo, error)
	ListNLB() ([]*NLBInfo, error)
	GetNLB(nlbIID IID) (NLBInfo, error)
	DeleteNLB(nlbIID IID) (bool, error)

	GetVMGroupHealthInfo(nlbIID IID) (HealthInfo, error)
	AddVMs(nlbIID IID, vmIIDs *[]IID) (VMGroupInfo, error)
	RemoveVMs(nlbIID IID, vmIIDs *[]IID) (bool, error)

	//---------------------------------------------------//
	// @todo  To support or not will be decided later.   //
	//---------------------------------------------------//

	//------ Frontend Control
	ChangeListener(nlbIID IID, listener ListenerInfo) (ListenerInfo, error)
	//------ Backend Control
	ChangeVMGroupInfo(nlbIID IID, vmGroup VMGroupInfo) (VMGroupInfo, error)
	ChangeHealthCheckerInfo(nlbIID IID, healthChecker HealthCheckerInfo) (HealthCheckerInfo, error)

	// ---------------------------------------------------//
	// @todo  To support or not will be decided later.   //
	// ---------------------------------------------------//
}
