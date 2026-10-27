// GCP Cloud DNS DNSHandler of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This covers only the Public Authoritative DNS baseline (Zone + Record CRUD).
//
// by CB-Spider Team, 2026.10.

package resources

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	dns "google.golang.org/api/dns/v1"
)

type GCPDNSHandler struct {
	Region     idrv.RegionInfo
	Ctx        context.Context
	Client     *dns.Service
	Credential idrv.CredentialInfo
}

// gcpFqdn normalizes a DNS name to the trailing-dot form Cloud DNS uses internally,
// so names coming from the user (with or without a trailing dot) compare equal
// to names coming back from the Cloud DNS API.
func gcpFqdn(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || strings.HasSuffix(name, ".") {
		return name
	}
	return name + "."
}

var gcpZoneNameSanitizer = regexp.MustCompile(`[^a-z0-9-]+`)

// makeManagedZoneName derives a GCP-safe ManagedZone resource name from the request.
// A ManagedZone Name must match [a-z]([-a-z0-9]*[a-z0-9])? and be unique within the project,
// unlike the DomainName(DnsName) which is the actual FQDN.
func makeManagedZoneName(nameId string, domainName string) string {
	base := nameId
	if base == "" {
		base = strings.TrimSuffix(domainName, ".")
	}
	base = strings.ToLower(base)
	base = gcpZoneNameSanitizer.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")
	if base == "" {
		base = "cb-dns-zone"
	}
	if !regexp.MustCompile(`^[a-z]`).MatchString(base) {
		base = "z-" + base
	}
	if len(base) > 63 {
		base = base[:63]
	}
	base = strings.Trim(base, "-")
	return base
}

func (dnsHandler *GCPDNSHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, "DNSZone", "ListIID()")
	start := call.Start()

	zones, err := dnsHandler.listAllManagedZones()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	iidList := []*irs.IID{}
	for _, zone := range zones {
		iidList = append(iidList, &irs.IID{NameId: zone.Name, SystemId: zone.Name})
	}

	LoggingInfo(hiscallInfo, start)
	return iidList, nil
}

func (dnsHandler *GCPDNSHandler) CreateDNSZone(dnsZoneReqInfo irs.DNSZoneReqInfo) (irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, dnsZoneReqInfo.IId.NameId, "CreateDNSZone()")
	start := call.Start()

	if dnsZoneReqInfo.DomainName == "" {
		err := fmt.Errorf("DomainName is required to create a DNS Zone")
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	projectID := dnsHandler.Credential.ProjectID
	zoneName := makeManagedZoneName(dnsZoneReqInfo.IId.NameId, dnsZoneReqInfo.DomainName)

	managedZone := &dns.ManagedZone{
		Name:        zoneName,
		DnsName:     gcpFqdn(dnsZoneReqInfo.DomainName),
		Description: "Created by CB-Spider",
	}

	result, err := dnsHandler.Client.ManagedZones.Create(projectID, managedZone).Context(dnsHandler.Ctx).Do()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	info, err := dnsHandler.getDNSZoneInfo(result)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func (dnsHandler *GCPDNSHandler) ListDNSZone() ([]*irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, "DNSZone", "ListDNSZone()")
	start := call.Start()

	zones, err := dnsHandler.listAllManagedZones()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	infoList := []*irs.DNSZoneInfo{}
	for _, zone := range zones {
		info, err := dnsHandler.getDNSZoneInfo(zone)
		if err != nil {
			LoggingError(hiscallInfo, err)
			return nil, err
		}
		infoList = append(infoList, &info)
	}

	LoggingInfo(hiscallInfo, start)
	return infoList, nil
}

func (dnsHandler *GCPDNSHandler) GetDNSZone(dnsZoneIID irs.IID) (irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, dnsZoneIID.NameId, "GetDNSZone()")
	start := call.Start()

	projectID := dnsHandler.Credential.ProjectID
	result, err := dnsHandler.Client.ManagedZones.Get(projectID, dnsZoneIID.SystemId).Context(dnsHandler.Ctx).Do()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	info, err := dnsHandler.getDNSZoneInfo(result)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func (dnsHandler *GCPDNSHandler) DeleteDNSZone(dnsZoneIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, dnsZoneIID.NameId, "DeleteDNSZone()")
	start := call.Start()

	projectID := dnsHandler.Credential.ProjectID

	zone, err := dnsHandler.Client.ManagedZones.Get(projectID, dnsZoneIID.SystemId).Context(dnsHandler.Ctx).Do()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	// Cloud DNS also refuses to delete a zone that still has records other than the
	// default apex NS/SOA pair, so those must be removed first.
	rrSets, err := dnsHandler.listAllResourceRecordSets(dnsZoneIID.SystemId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	deletions := []*dns.ResourceRecordSet{}
	for _, rrSet := range rrSets {
		if isDefaultApexRecord(rrSet, zone.DnsName) {
			continue
		}
		deletions = append(deletions, rrSet)
	}
	if len(deletions) > 0 {
		_, err = dnsHandler.Client.Changes.Create(projectID, dnsZoneIID.SystemId, &dns.Change{
			Deletions: deletions,
		}).Context(dnsHandler.Ctx).Do()
		if err != nil {
			LoggingError(hiscallInfo, err)
			return false, err
		}
	}

	err = dnsHandler.Client.ManagedZones.Delete(projectID, dnsZoneIID.SystemId).Context(dnsHandler.Ctx).Do()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	LoggingInfo(hiscallInfo, start)
	return true, nil
}

func (dnsHandler *GCPDNSHandler) AddDNSRecord(dnsZoneIID irs.IID, recordReqInfo irs.DNSRecordReqInfo) (irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, recordReqInfo.Name, "AddDNSRecord()")
	start := call.Start()

	if len(recordReqInfo.Value) == 0 {
		err := fmt.Errorf("at least one Value is required for a DNS record")
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	ttl := int64(recordReqInfo.TTL)
	if ttl <= 0 {
		ttl = 300 // CB-Spider default when the caller does not specify one
	}

	newRRSet := &dns.ResourceRecordSet{
		Name:    gcpFqdn(recordReqInfo.Name),
		Type:    string(recordReqInfo.Type),
		Ttl:     ttl,
		Rrdatas: recordReqInfo.Value,
	}

	projectID := dnsHandler.Credential.ProjectID
	_, err := dnsHandler.Client.Changes.Create(projectID, dnsZoneIID.SystemId, &dns.Change{
		Additions: []*dns.ResourceRecordSet{newRRSet},
	}).Context(dnsHandler.Ctx).Do()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	info, err := dnsHandler.GetDNSZone(dnsZoneIID)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func (dnsHandler *GCPDNSHandler) ChangeDNSRecord(dnsZoneIID irs.IID, recordReqInfo irs.DNSRecordReqInfo) (irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, recordReqInfo.Name, "ChangeDNSRecord()")
	start := call.Start()

	if len(recordReqInfo.Value) == 0 {
		err := fmt.Errorf("at least one Value is required for a DNS record")
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	ttl := int64(recordReqInfo.TTL)
	if ttl <= 0 {
		ttl = 300 // CB-Spider default when the caller does not specify one
	}

	projectID := dnsHandler.Credential.ProjectID
	name := gcpFqdn(recordReqInfo.Name)
	recordType := string(recordReqInfo.Type)

	// GCP's Changes API requires the exact existing ResourceRecordSet to be present
	// in Deletions in order to replace it with a new one submitted in Additions,
	// both within the same Change.
	existing, err := dnsHandler.Client.ResourceRecordSets.Get(projectID, dnsZoneIID.SystemId, name, recordType).Context(dnsHandler.Ctx).Do()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	newRRSet := &dns.ResourceRecordSet{
		Name:    name,
		Type:    recordType,
		Ttl:     ttl,
		Rrdatas: recordReqInfo.Value,
	}

	_, err = dnsHandler.Client.Changes.Create(projectID, dnsZoneIID.SystemId, &dns.Change{
		Deletions: []*dns.ResourceRecordSet{existing},
		Additions: []*dns.ResourceRecordSet{newRRSet},
	}).Context(dnsHandler.Ctx).Do()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	info, err := dnsHandler.GetDNSZone(dnsZoneIID)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func (dnsHandler *GCPDNSHandler) RemoveDNSRecord(dnsZoneIID irs.IID, recordIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, recordIID.NameId, "RemoveDNSRecord()")
	start := call.Start()

	name, recordType, err := parseRecordSystemId(recordIID.SystemId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	projectID := dnsHandler.Credential.ProjectID
	target, err := dnsHandler.Client.ResourceRecordSets.Get(projectID, dnsZoneIID.SystemId, gcpFqdn(name), recordType).Context(dnsHandler.Ctx).Do()
	if err != nil {
		err := fmt.Errorf("DNS record '%s'(%s) does not exist", name, recordType)
		LoggingError(hiscallInfo, err)
		return false, err
	}

	_, err = dnsHandler.Client.Changes.Create(projectID, dnsZoneIID.SystemId, &dns.Change{
		Deletions: []*dns.ResourceRecordSet{target},
	}).Context(dnsHandler.Ctx).Do()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	LoggingInfo(hiscallInfo, start)
	return true, nil
}

//------------------------- internal helpers -------------------------

func (dnsHandler *GCPDNSHandler) listAllManagedZones() ([]*dns.ManagedZone, error) {
	projectID := dnsHandler.Credential.ProjectID
	zones := []*dns.ManagedZone{}

	err := dnsHandler.Client.ManagedZones.List(projectID).Pages(dnsHandler.Ctx, func(page *dns.ManagedZonesListResponse) error {
		zones = append(zones, page.ManagedZones...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return zones, nil
}

func (dnsHandler *GCPDNSHandler) listAllResourceRecordSets(zoneName string) ([]*dns.ResourceRecordSet, error) {
	projectID := dnsHandler.Credential.ProjectID
	rrSets := []*dns.ResourceRecordSet{}

	err := dnsHandler.Client.ResourceRecordSets.List(projectID, zoneName).Pages(dnsHandler.Ctx, func(page *dns.ResourceRecordSetsListResponse) error {
		rrSets = append(rrSets, page.Rrsets...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return rrSets, nil
}

func (dnsHandler *GCPDNSHandler) getDNSZoneInfo(zone *dns.ManagedZone) (irs.DNSZoneInfo, error) {
	rrSets, err := dnsHandler.listAllResourceRecordSets(zone.Name)
	if err != nil {
		return irs.DNSZoneInfo{}, err
	}

	nameServers := zone.NameServers

	recordInfoList := []irs.DNSRecordInfo{}
	for _, rrSet := range rrSets {
		recordInfoList = append(recordInfoList, convertToDNSRecordInfo(rrSet))
	}

	return irs.DNSZoneInfo{
		IId:            irs.IID{NameId: zone.Name, SystemId: zone.Name},
		DomainName:     zone.DnsName,
		NameServers:    nameServers,
		RecordInfoList: recordInfoList,
	}, nil
}

func convertToDNSRecordInfo(rrSet *dns.ResourceRecordSet) irs.DNSRecordInfo {
	return irs.DNSRecordInfo{
		IId:   irs.IID{NameId: rrSet.Name, SystemId: makeRecordSystemId(rrSet.Name, rrSet.Type)},
		Type:  irs.DNSRecordType(rrSet.Type),
		Name:  rrSet.Name,
		Value: rrSet.Rrdatas,
		TTL:   int(rrSet.Ttl),
	}
}

// isDefaultApexRecord reports whether the record set is the NS/SOA pair Cloud DNS
// auto-creates for the zone apex. Those cannot be deleted individually; they are
// removed automatically by ManagedZones.Delete.
func isDefaultApexRecord(rrSet *dns.ResourceRecordSet, zoneDnsName string) bool {
	if rrSet.Name != zoneDnsName {
		return false
	}
	return rrSet.Type == "NS" || rrSet.Type == "SOA"
}

func makeRecordSystemId(name string, recordType string) string {
	return name + ":" + recordType
}

func parseRecordSystemId(systemId string) (name string, recordType string, err error) {
	idx := strings.LastIndex(systemId, ":")
	if idx < 0 {
		return "", "", fmt.Errorf("invalid DNS record SystemId: %s", systemId)
	}
	return systemId[:idx], systemId[idx+1:], nil
}
