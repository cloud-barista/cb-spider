// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by CB-Spider Team, 2026.09.

package resources

// DNSRecordType is the DNS resource record type (RFC 1035 etc.)
type DNSRecordType string

const (
	DNSRecordTypeA     DNSRecordType = "A"
	DNSRecordTypeAAAA  DNSRecordType = "AAAA"
	DNSRecordTypeCNAME DNSRecordType = "CNAME"
	DNSRecordTypeMX    DNSRecordType = "MX"
	DNSRecordTypeTXT   DNSRecordType = "TXT"
	DNSRecordTypeNS    DNSRecordType = "NS"
	DNSRecordTypeSRV   DNSRecordType = "SRV"
	DNSRecordTypeCAA   DNSRecordType = "CAA"
)

// DNSZoneReqInfo is used to create a Public Authoritative DNS Zone(Hosted Zone).
type DNSZoneReqInfo struct {
	IId IID // {NameId, SystemId}

	DomainName string // ex) "example.com"

	TagList []KeyValue
}

// DNSZoneInfo is the Public Authoritative DNS Zone(Hosted Zone) information,
// including the records currently managed by CB-Spider in this zone.
type DNSZoneInfo struct {
	IId IID `json:"IId" validate:"required"` // {NameId, SystemId}

	DomainName     string          `json:"DomainName" validate:"required" example:"example.com" description:"The domain name of this public hosted zone"`
	NameServers    []string        `json:"NameServers,omitempty" validate:"omitempty" description:"The name servers assigned by the CSP. Set these as the NS records at the domain registrar"`
	RecordInfoList []DNSRecordInfo `json:"RecordInfoList" validate:"required" description:"A list of DNS records registered by CB-Spider in this zone"`

	TagList      []KeyValue `json:"TagList,omitempty" validate:"omitempty" description:"A list of tags associated with this DNS zone"`
	KeyValueList []KeyValue `json:"KeyValueList,omitempty" validate:"omitempty" description:"Additional key-value pairs associated with this DNS zone"`
}

// DNSRecordReqInfo is used to create/change a record in a DNS Zone.
type DNSRecordReqInfo struct {
	IId IID // {NameId, SystemId}

	Type  DNSRecordType `json:"Type" validate:"required" example:"A" description:"Record type, ex) A, AAAA, CNAME, MX, TXT, NS, SRV, CAA"`
	Name  string        `json:"Name" validate:"required" example:"www.example.com" description:"Fully qualified record name"`
	Value []string      `json:"Value" validate:"required" example:"203.0.113.10" description:"Record values, ex) IP addresses for A/AAAA, hostname for CNAME"`
	TTL   int           `json:"TTL,omitempty" validate:"omitempty" example:"300" description:"Time to live in seconds. 0 means use the CSP default"`
}

// DNSRecordInfo is the record information managed by CB-Spider inside a DNS Zone.
type DNSRecordInfo struct {
	IId IID `json:"IId" validate:"required"` // {NameId, SystemId}

	Type  DNSRecordType `json:"Type" validate:"required" example:"A"`
	Name  string        `json:"Name" validate:"required" example:"www.example.com"`
	Value []string      `json:"Value" validate:"required" example:"203.0.113.10"`
	TTL   int           `json:"TTL,omitempty" validate:"omitempty" example:"300"`

	KeyValueList []KeyValue `json:"KeyValueList,omitempty" validate:"omitempty"`
}

// DNSHandler is the CB-Spider abstraction for a CSP's Public Authoritative DNS service
// (ex: AWS Route 53, Azure DNS, GCP Cloud DNS, ...).
//
// Scope note: this handler covers ONLY the Public Authoritative DNS baseline
// (Zone + Record CRUD). Private/VPC DNS, weighted/latency/geo routing policies,
// health-check based failover, and DNSSEC are intentionally out of scope -
// they are optional, CSP-specific capabilities layered on top of this baseline
// and are not required to map a registered domain to a VM's web server.
type DNSHandler interface {
	ListIID() ([]*IID, error)

	CreateDNSZone(dnsZoneReqInfo DNSZoneReqInfo) (DNSZoneInfo, error)
	ListDNSZone() ([]*DNSZoneInfo, error)
	GetDNSZone(dnsZoneIID IID) (DNSZoneInfo, error)
	DeleteDNSZone(dnsZoneIID IID) (bool, error)

	AddDNSRecord(dnsZoneIID IID, recordReqInfo DNSRecordReqInfo) (DNSZoneInfo, error)
	ChangeDNSRecord(dnsZoneIID IID, recordReqInfo DNSRecordReqInfo) (DNSZoneInfo, error)
	RemoveDNSRecord(dnsZoneIID IID, recordIID IID) (bool, error)
}
