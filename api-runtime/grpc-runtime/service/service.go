package service

// ===== [ Constants and Variables ] =====

// define string of resource types
const (
	rsImage string = "image"
	rsVPC   string = "vpc"
	rsSG    string = "sg"
	rsKey   string = "keypair"
	rsVM    string = "vm"
)

const rsSubnetPrefix string = "subnet:"
const sgDELIMITER string = "-delimiter-"

// ===== [ Types ] =====

// CIMService -
type CIMService struct {
}

// CCMService -
type CCMService struct {
}

// SSHService -
type SSHService struct {
}

// ===== [ Implementations ] =====

// ===== [ Private Functions ] =====

// ===== [ Public Functions ] =====
