package request

import (
	"time"

	pb "github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/stub/cbspider"
)

// ===== [ Constants and Variables ] =====

// ===== [ Types ] =====

// CIMRequest -
type CIMRequest struct {
	Client  pb.CIMClient
	Timeout time.Duration

	InType  string
	InData  string
	OutType string
}

// CCMRequest -
type CCMRequest struct {
	Client  pb.CCMClient
	Timeout time.Duration

	InType  string
	InData  string
	OutType string
}

// SSHRequest -
type SSHRequest struct {
	Client  pb.SSHClient
	Timeout time.Duration

	InType  string
	InData  string
	OutType string
}

// ===== [ Implementations ] =====

// ===== [ Private Functions ] =====

// ===== [ Public Functions ] =====
