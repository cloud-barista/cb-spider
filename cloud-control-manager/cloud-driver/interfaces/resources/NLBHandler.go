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

type NLBReqInfo struct {
	IId		IID // {NameId, SystemId}
	VpcIID		IID    // {NameId, SystemId}

	//------ Frontend
	Listeners	*[]ListenerInfo

	//------ Backend
	ServiceGroup	ServiceGroupInfo
	ServiceVMs	*[]IID
	HealthChecker	HealthCheckerInfo
}

type ListenerInfo struct {
	Protocol	string // TCP|UDP
	Port		string // 1-65535

	CspID		string // Optional, May be Used by Driver.
	KeyValueList []KeyValue
}

type ServiceGroupInfo struct {
        Protocol        string // TCP|UDP|HTTP|HTTPS
        Port            string // 1-65535

	CspID		string // Optional, May be Used by Driver.
        KeyValueList []KeyValue
}

type HealthCheckerInfo struct {
	Protocol	string	// TCP|HTTP|HTTPS
	Port		string	// Service Port or 1-65535
	Interval	int	// secs, Interval time between health checks.
	Timeout		int	// secs, Waiting time to decide an unhealthy VM when no response.
	Threshold	int	// num, The number of continuous health checks to change the VM status.

        KeyValueList	[]KeyValue
}

type HealthyInfo struct {
	AllServiceVMs	*[]IID
	HealthyVMs	*[]IID
	UnHealthyVMs	*[]IID
}

type NLBInfo struct {
        IId		IID	// {NameId, SystemId}
        VpcIID		IID	// {NameId, SystemId}

	//------ Frontend
	FrontendIP	string	// Auto Generated and attached
	FrontendDNSName	string	// Optional, Auto Generated and attached
	Listeners	*[]ListenerInfo

	//------ Backend
	ServiceGroup	ServiceGroupInfo
	ServiceVMs	*[]IID
	HealthChecker	HealthCheckerInfo

        KeyValueList	[]KeyValue
}


type NLBHandler interface {

	//------ NLB Management
	CreateNLB(nlbReqInfo NLBReqInfo) (NLBInfo, error)
	ListNLB() ([]*NLBInfo, error)
	GetNLB(nlbIID IID) (NLBInfo, error)
	DeleteNLB(nlbIID IID) (bool, error)

	//------ Frontend Control
	AddListeners(nlbIID IID, listeners *[]ListenerInfo) (NLBInfo, error)
	RemoveListeners(nlbIID IID, listeners *[]ListenerInfo) (bool, error)

	//------ Backend Control
	ChangeServiceGroupInfo(nlbIID IID, serviceGroup ServiceGroupInfo) (error)
	AddServiceVMs(nlbIID IID, vmIIDs *[]IID) (NLBInfo, error)
	RemoveServiceVMs(nlbIID IID, vmIIDs *[]IID) (bool, error)
	GetServiceVMStatus(nlbIID IID) (HealthyInfo, error)
	ChangeHealthCheckerInfo(nlbIID IID, healthChecker HealthCheckerInfo) (error)
}
