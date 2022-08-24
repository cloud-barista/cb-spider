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

//-------- Info Structure
type NLBInfo struct {
	IId		IID 	// {NameId, SystemId}
	VpcIID		IID	// {NameId, SystemId}

	Type		string	// PUBLIC(V) | INTERNAL
	Scope		string	// REGION(V) | GLOBAL

	//------ Frontend
	Listener	ListenerInfo

	//------ Backend
	VMGroup		VMGroupInfo
	HealthChecker	HealthCheckerInfo

	CreatedTime	time.Time
	KeyValueList []KeyValue
}

type ListenerInfo struct {
	Protocol	string	// TCP|UDP
	IP		string	// Auto Generated and attached
	Port		string	// 1-65535
	DNSName		string	// Optional, Auto Generated and attached

	CspID		string	// Optional, May be Used by Driver.
	KeyValueList []KeyValue
}

type VMGroupInfo struct {
        Protocol        string	// TCP|UDP
        Port            string	// 1-65535
	VMs		*[]IID

	CspID		string	// Optional, May be Used by Driver.
        KeyValueList []KeyValue
}

type HealthCheckerInfo struct {
	Protocol	string	// TCP|HTTP
	Port		string	// Listener Port or 1-65535
	Interval	int	// secs, Interval time between health checks.
	Timeout		int	// secs, Waiting time to decide an unhealthy VM when no response.
	Threshold	int	// num, The number of continuous health checks to change the VM status.

	CspID		string	// Optional, May be Used by Driver.
        KeyValueList	[]KeyValue
}

type HealthInfo struct {
	AllVMs		*[]IID
	HealthyVMs	*[]IID
	UnHealthyVMs	*[]IID
}

//-------- API
type NLBHandler interface {

	//------ NLB Management
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

//---------------------------------------------------//
// @todo  To support or not will be decided later.   //
//---------------------------------------------------//
}
