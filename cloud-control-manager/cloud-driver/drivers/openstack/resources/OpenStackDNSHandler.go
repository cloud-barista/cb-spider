// OpenStack Designate DNSHandler of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This covers only the Public Authoritative DNS baseline (Zone + Record CRUD).
//
// by CB-Spider Team, 2026.09.

package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/dns/v2/recordsets"
	"github.com/gophercloud/gophercloud/v2/openstack/dns/v2/zones"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type OpenStackDNSHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	DNSClient      *gophercloud.ServiceClient
}

// fqdn normalizes a DNS name to the trailing-dot form Designate uses internally,
// so names coming from the user (with or without a trailing dot) compare equal
// to names coming back from the Designate API.
func fqdn(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || strings.HasSuffix(name, ".") {
		return name
	}
	return name + "."
}

// unfqdn strips the trailing dot Designate always returns, mainly used to
// build a reasonable synthetic admin e-mail for the zone's SOA record.
func unfqdn(name string) string {
	return strings.TrimSuffix(name, ".")
}

func (dnsHandler *OpenStackDNSHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.CredentialInfo.IdentityEndpoint, call.DNSZONE, "DNSZone", "ListIID()")
	start := call.Start()

	zoneList, err := dnsHandler.listAllZones()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	iidList := []*irs.IID{}
	for _, zone := range zoneList {
		iidList = append(iidList, &irs.IID{NameId: zone.Name, SystemId: zone.ID})
	}

	LoggingInfo(hiscallInfo, start)
	return iidList, nil
}

func (dnsHandler *OpenStackDNSHandler) CreateDNSZone(dnsZoneReqInfo irs.DNSZoneReqInfo) (irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.CredentialInfo.IdentityEndpoint, call.DNSZONE, dnsZoneReqInfo.IId.NameId, "CreateDNSZone()")
	start := call.Start()

	if dnsZoneReqInfo.DomainName == "" {
		err := fmt.Errorf("DomainName is required to create a DNS Zone")
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	domain := fqdn(dnsZoneReqInfo.DomainName)
	// Designate requires an admin contact e-mail for the zone's SOA record.
	// CB-Spider's DNSZoneReqInfo has no such field, so synthesize a reasonable
	// placeholder from the domain name itself.
	opts := zones.CreateOpts{
		Name:  domain,
		Email: "admin@" + unfqdn(domain),
	}

	zone, err := zones.Create(context.TODO(), dnsHandler.DNSClient, opts).Extract()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	info, err := dnsHandler.getDNSZoneInfo(zone)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func (dnsHandler *OpenStackDNSHandler) ListDNSZone() ([]*irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.CredentialInfo.IdentityEndpoint, call.DNSZONE, "DNSZone", "ListDNSZone()")
	start := call.Start()

	zoneList, err := dnsHandler.listAllZones()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	infoList := []*irs.DNSZoneInfo{}
	for _, zone := range zoneList {
		info, err := dnsHandler.getDNSZoneInfo(&zone)
		if err != nil {
			LoggingError(hiscallInfo, err)
			return nil, err
		}
		infoList = append(infoList, &info)
	}

	LoggingInfo(hiscallInfo, start)
	return infoList, nil
}

func (dnsHandler *OpenStackDNSHandler) GetDNSZone(dnsZoneIID irs.IID) (irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.CredentialInfo.IdentityEndpoint, call.DNSZONE, dnsZoneIID.NameId, "GetDNSZone()")
	start := call.Start()

	zone, err := zones.Get(context.TODO(), dnsHandler.DNSClient, dnsZoneIID.SystemId).Extract()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	info, err := dnsHandler.getDNSZoneInfo(zone)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	LoggingInfo(hiscallInfo, start)
	return info, nil
}

// DeleteDNSZone deletes a Designate zone.
//
// Designate's zone delete is asynchronous: the API call only marks the zone
// PENDING_DELETE and returns immediately (202 Accepted), while the actual
// removal - including cascading deletion of every recordset still inside the
// zone - happens in the background on the Designate workers/pool backend.
// There is no "zone not empty" restriction like Route53's, so unlike the AWS
// driver there is no need to pre-clean recordsets here: firing the delete
// call is sufficient and matches the simplicity of other delete operations
// in this codebase (no polling for completion).
func (dnsHandler *OpenStackDNSHandler) DeleteDNSZone(dnsZoneIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.CredentialInfo.IdentityEndpoint, call.DNSZONE, dnsZoneIID.NameId, "DeleteDNSZone()")
	start := call.Start()

	_, err := zones.Delete(context.TODO(), dnsHandler.DNSClient, dnsZoneIID.SystemId).Extract()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	LoggingInfo(hiscallInfo, start)
	return true, nil
}

func (dnsHandler *OpenStackDNSHandler) AddDNSRecord(dnsZoneIID irs.IID, recordReqInfo irs.DNSRecordReqInfo) (irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.CredentialInfo.IdentityEndpoint, call.DNSZONE, recordReqInfo.Name, "AddDNSRecord()")
	start := call.Start()

	if len(recordReqInfo.Value) == 0 {
		err := fmt.Errorf("at least one Value is required for a DNS record")
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	ttl := recordReqInfo.TTL
	if ttl <= 0 {
		ttl = 300 // CB-Spider default when the caller does not specify one
	}

	opts := recordsets.CreateOpts{
		Name:    fqdn(recordReqInfo.Name),
		Type:    string(recordReqInfo.Type),
		TTL:     ttl,
		Records: recordReqInfo.Value,
	}

	_, err := recordsets.Create(context.TODO(), dnsHandler.DNSClient, dnsZoneIID.SystemId, opts).Extract()
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

// ChangeDNSRecord behaves as an upsert: it updates the recordset matching
// Name+Type if one already exists in the zone, or creates a new one otherwise.
// Designate recordsets are addressed by a real UUID (PUT /recordsets/{id}),
// unlike Route53's Name+Type based Upsert action, so the existing recordset
// must be looked up first.
func (dnsHandler *OpenStackDNSHandler) ChangeDNSRecord(dnsZoneIID irs.IID, recordReqInfo irs.DNSRecordReqInfo) (irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.CredentialInfo.IdentityEndpoint, call.DNSZONE, recordReqInfo.Name, "ChangeDNSRecord()")
	start := call.Start()

	if len(recordReqInfo.Value) == 0 {
		err := fmt.Errorf("at least one Value is required for a DNS record")
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	ttl := recordReqInfo.TTL
	if ttl <= 0 {
		ttl = 300 // CB-Spider default when the caller does not specify one
	}

	rrSets, err := dnsHandler.listAllRecordSets(dnsZoneIID.SystemId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	name := fqdn(recordReqInfo.Name)
	recordType := string(recordReqInfo.Type)

	var target *recordsets.RecordSet
	for i := range rrSets {
		if rrSets[i].Name == name && rrSets[i].Type == recordType {
			target = &rrSets[i]
			break
		}
	}

	if target != nil {
		updateOpts := recordsets.UpdateOpts{
			TTL:     &ttl,
			Records: recordReqInfo.Value,
		}
		_, err = recordsets.Update(context.TODO(), dnsHandler.DNSClient, dnsZoneIID.SystemId, target.ID, updateOpts).Extract()
		if err != nil {
			LoggingError(hiscallInfo, err)
			return irs.DNSZoneInfo{}, err
		}
	} else {
		createOpts := recordsets.CreateOpts{
			Name:    name,
			Type:    recordType,
			TTL:     ttl,
			Records: recordReqInfo.Value,
		}
		_, err = recordsets.Create(context.TODO(), dnsHandler.DNSClient, dnsZoneIID.SystemId, createOpts).Extract()
		if err != nil {
			LoggingError(hiscallInfo, err)
			return irs.DNSZoneInfo{}, err
		}
	}

	info, err := dnsHandler.GetDNSZone(dnsZoneIID)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func (dnsHandler *OpenStackDNSHandler) RemoveDNSRecord(dnsZoneIID irs.IID, recordIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.CredentialInfo.IdentityEndpoint, call.DNSZONE, recordIID.NameId, "RemoveDNSRecord()")
	start := call.Start()

	if recordIID.SystemId == "" {
		err := fmt.Errorf("SystemId is required to remove a DNS record")
		LoggingError(hiscallInfo, err)
		return false, err
	}

	err := recordsets.Delete(context.TODO(), dnsHandler.DNSClient, dnsZoneIID.SystemId, recordIID.SystemId).ExtractErr()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	LoggingInfo(hiscallInfo, start)
	return true, nil
}

//------------------------- internal helpers -------------------------

func (dnsHandler *OpenStackDNSHandler) listAllZones() ([]zones.Zone, error) {
	pages, err := zones.List(dnsHandler.DNSClient, nil).AllPages(context.TODO())
	if err != nil {
		return nil, err
	}
	return zones.ExtractZones(pages)
}

func (dnsHandler *OpenStackDNSHandler) listAllRecordSets(zoneID string) ([]recordsets.RecordSet, error) {
	pages, err := recordsets.ListByZone(dnsHandler.DNSClient, zoneID, nil).AllPages(context.TODO())
	if err != nil {
		return nil, err
	}
	return recordsets.ExtractRecordSets(pages)
}

// getDNSZoneInfo builds a DNSZoneInfo from a Designate zone, filling in the
// record list and the zone's authoritative name servers.
//
// Note: this gophercloud/v2 version of the Designate DNS v2 API binding does
// not expose a "GET /zones/{id}/nameservers" call, and zones.Zone.Masters is
// only meaningful for secondary (slave) zones - it is empty for the primary
// zones CB-Spider creates here. The reliable, CSP-agnostic way to surface the
// name servers CB-Spider must hand to the domain registrar is therefore the
// same one used as a fallback in the AWS driver: read them off the NS
// recordset Designate auto-creates at the zone apex.
func (dnsHandler *OpenStackDNSHandler) getDNSZoneInfo(zone *zones.Zone) (irs.DNSZoneInfo, error) {
	rrSets, err := dnsHandler.listAllRecordSets(zone.ID)
	if err != nil {
		return irs.DNSZoneInfo{}, err
	}

	nameServers := []string{}
	recordInfoList := []irs.DNSRecordInfo{}
	for _, rrSet := range rrSets {
		if rrSet.Type == "NS" && rrSet.Name == zone.Name {
			nameServers = append(nameServers, rrSet.Records...)
		}
		recordInfoList = append(recordInfoList, convertToDNSRecordInfo(rrSet))
	}

	return irs.DNSZoneInfo{
		IId:            irs.IID{NameId: zone.Name, SystemId: zone.ID},
		DomainName:     zone.Name,
		NameServers:    nameServers,
		RecordInfoList: recordInfoList,
	}, nil
}

func convertToDNSRecordInfo(rrSet recordsets.RecordSet) irs.DNSRecordInfo {
	return irs.DNSRecordInfo{
		IId:   irs.IID{NameId: rrSet.Name, SystemId: rrSet.ID},
		Type:  irs.DNSRecordType(rrSet.Type),
		Name:  rrSet.Name,
		Value: rrSet.Records,
		TTL:   rrSet.TTL,
	}
}
