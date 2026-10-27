// NHN Cloud DNS Plus(Designate-based) DNSHandler of CB-Spider.
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
	"fmt"
	"strings"

	nhnsdk "github.com/cloud-barista/nhncloud-sdk-go"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/dns/v2/recordsets"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/dns/v2/zones"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NhnDNSHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	DNSClient      *nhnsdk.ServiceClient
}

// dnsFqdn normalizes a DNS name to the trailing-dot form Designate(NHN DNS Plus)
// uses internally, so names coming from the user (with or without a trailing
// dot) compare equal to names coming back from the DNS API.
func dnsFqdn(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || strings.HasSuffix(name, ".") {
		return name
	}
	return name + "."
}

// dnsAdminEmail synthesizes a placeholder admin-contact e-mail for the zone's
// SOA record. CB-Spider's DNSZoneReqInfo has no corresponding field, and
// Designate's zones.CreateOpts.Email is a required field, so a reasonable
// default is derived from the domain name itself(ex: "admin@example.com").
func dnsAdminEmail(domainName string) string {
	return "admin@" + strings.TrimSuffix(dnsFqdn(domainName), ".")
}

func (dnsHandler *NhnDNSHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := getCallLogScheme(dnsHandler.RegionInfo.Zone, call.DNSZONE, "DNSZone", "ListIID()")
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

func (dnsHandler *NhnDNSHandler) CreateDNSZone(dnsZoneReqInfo irs.DNSZoneReqInfo) (irs.DNSZoneInfo, error) {
	hiscallInfo := getCallLogScheme(dnsHandler.RegionInfo.Zone, call.DNSZONE, dnsZoneReqInfo.IId.NameId, "CreateDNSZone()")
	start := call.Start()

	if dnsZoneReqInfo.DomainName == "" {
		err := fmt.Errorf("DomainName is required to create a DNS Zone")
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	createOpts := zones.CreateOpts{
		Name:  dnsFqdn(dnsZoneReqInfo.DomainName),
		Email: dnsAdminEmail(dnsZoneReqInfo.DomainName),
		Type:  "PRIMARY",
	}

	zone, err := zones.Create(dnsHandler.DNSClient, createOpts).Extract()
	if err != nil {
		newErr := fmt.Errorf("failed to Create DNS Zone. [%v]", err)
		LoggingError(hiscallInfo, newErr)
		return irs.DNSZoneInfo{}, newErr
	}

	info, err := dnsHandler.getDNSZoneInfo(zone)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func (dnsHandler *NhnDNSHandler) ListDNSZone() ([]*irs.DNSZoneInfo, error) {
	hiscallInfo := getCallLogScheme(dnsHandler.RegionInfo.Zone, call.DNSZONE, "DNSZone", "ListDNSZone()")
	start := call.Start()

	zoneList, err := dnsHandler.listAllZones()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	infoList := []*irs.DNSZoneInfo{}
	for _, zone := range zoneList {
		z := zone
		info, err := dnsHandler.getDNSZoneInfo(&z)
		if err != nil {
			LoggingError(hiscallInfo, err)
			return nil, err
		}
		infoList = append(infoList, &info)
	}

	LoggingInfo(hiscallInfo, start)
	return infoList, nil
}

func (dnsHandler *NhnDNSHandler) GetDNSZone(dnsZoneIID irs.IID) (irs.DNSZoneInfo, error) {
	hiscallInfo := getCallLogScheme(dnsHandler.RegionInfo.Zone, call.DNSZONE, dnsZoneIID.NameId, "GetDNSZone()")
	start := call.Start()

	zone, err := dnsHandler.getZoneBySystemId(dnsZoneIID.SystemId)
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

// DeleteDNSZone deletes the zone.
//
// Note: NHN Cloud DNS Plus(Designate) cascades a zone delete - all recordsets
// that belong to the zone(including the apex SOA/NS pair) are automatically
// removed by the server when the zone itself is deleted. There is no need(and
// no supported way) to delete the recordsets individually beforehand.
func (dnsHandler *NhnDNSHandler) DeleteDNSZone(dnsZoneIID irs.IID) (bool, error) {
	hiscallInfo := getCallLogScheme(dnsHandler.RegionInfo.Zone, call.DNSZONE, dnsZoneIID.NameId, "DeleteDNSZone()")
	start := call.Start()

	zone, err := dnsHandler.getZoneBySystemId(dnsZoneIID.SystemId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	_, err = zones.Delete(dnsHandler.DNSClient, zone.ID).Extract()
	if err != nil {
		newErr := fmt.Errorf("failed to Delete DNS Zone. [%v]", err)
		LoggingError(hiscallInfo, newErr)
		return false, newErr
	}

	LoggingInfo(hiscallInfo, start)
	return true, nil
}

func (dnsHandler *NhnDNSHandler) AddDNSRecord(dnsZoneIID irs.IID, recordReqInfo irs.DNSRecordReqInfo) (irs.DNSZoneInfo, error) {
	hiscallInfo := getCallLogScheme(dnsHandler.RegionInfo.Zone, call.DNSZONE, recordReqInfo.Name, "AddDNSRecord()")
	start := call.Start()

	if len(recordReqInfo.Value) == 0 {
		err := fmt.Errorf("at least one Value is required for a DNS record")
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	zone, err := dnsHandler.getZoneBySystemId(dnsZoneIID.SystemId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	createOpts := recordsets.CreateOpts{
		Name:    dnsFqdn(recordReqInfo.Name),
		Type:    string(recordReqInfo.Type),
		Records: recordReqInfo.Value,
		TTL:     recordReqInfo.TTL,
	}

	_, err = recordsets.Create(dnsHandler.DNSClient, zone.ID, createOpts).Extract()
	if err != nil {
		newErr := fmt.Errorf("failed to Add DNS Record. [%v]", err)
		LoggingError(hiscallInfo, newErr)
		return irs.DNSZoneInfo{}, newErr
	}

	info, err := dnsHandler.getDNSZoneInfo(zone)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func (dnsHandler *NhnDNSHandler) ChangeDNSRecord(dnsZoneIID irs.IID, recordReqInfo irs.DNSRecordReqInfo) (irs.DNSZoneInfo, error) {
	hiscallInfo := getCallLogScheme(dnsHandler.RegionInfo.Zone, call.DNSZONE, recordReqInfo.Name, "ChangeDNSRecord()")
	start := call.Start()

	if len(recordReqInfo.Value) == 0 {
		err := fmt.Errorf("at least one Value is required for a DNS record")
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	zone, err := dnsHandler.getZoneBySystemId(dnsZoneIID.SystemId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	rrSets, err := dnsHandler.listAllRecordSets(zone.ID)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	var target *recordsets.RecordSet
	for i, rrSet := range rrSets {
		if rrSet.Name == dnsFqdn(recordReqInfo.Name) && rrSet.Type == string(recordReqInfo.Type) {
			target = &rrSets[i]
			break
		}
	}
	if target == nil {
		err := fmt.Errorf("DNS record '%s'(%s) does not exist", recordReqInfo.Name, recordReqInfo.Type)
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	updateOpts := recordsets.UpdateOpts{
		Records: recordReqInfo.Value,
	}
	if recordReqInfo.TTL > 0 {
		ttl := recordReqInfo.TTL
		updateOpts.TTL = &ttl
	}

	_, err = recordsets.Update(dnsHandler.DNSClient, zone.ID, target.ID, updateOpts).Extract()
	if err != nil {
		newErr := fmt.Errorf("failed to Change DNS Record. [%v]", err)
		LoggingError(hiscallInfo, newErr)
		return irs.DNSZoneInfo{}, newErr
	}

	info, err := dnsHandler.getDNSZoneInfo(zone)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func (dnsHandler *NhnDNSHandler) RemoveDNSRecord(dnsZoneIID irs.IID, recordIID irs.IID) (bool, error) {
	hiscallInfo := getCallLogScheme(dnsHandler.RegionInfo.Zone, call.DNSZONE, recordIID.NameId, "RemoveDNSRecord()")
	start := call.Start()

	zone, err := dnsHandler.getZoneBySystemId(dnsZoneIID.SystemId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	if recordIID.SystemId == "" {
		err := fmt.Errorf("SystemId is required to remove a DNS record")
		LoggingError(hiscallInfo, err)
		return false, err
	}

	err = recordsets.Delete(dnsHandler.DNSClient, zone.ID, recordIID.SystemId).ExtractErr()
	if err != nil {
		newErr := fmt.Errorf("failed to Remove DNS Record. [%v]", err)
		LoggingError(hiscallInfo, newErr)
		return false, newErr
	}

	LoggingInfo(hiscallInfo, start)
	return true, nil
}

//------------------------- internal helpers -------------------------

func (dnsHandler *NhnDNSHandler) listAllZones() ([]zones.Zone, error) {
	allPages, err := zones.List(dnsHandler.DNSClient, nil).AllPages()
	if err != nil {
		return nil, err
	}
	return zones.ExtractZones(allPages)
}

func (dnsHandler *NhnDNSHandler) getZoneBySystemId(systemId string) (*zones.Zone, error) {
	if systemId == "" {
		return nil, fmt.Errorf("SystemId is required to look up a DNS Zone")
	}
	zone, err := zones.Get(dnsHandler.DNSClient, systemId).Extract()
	if err != nil {
		return nil, fmt.Errorf("failed to Get DNS Zone. [%v]", err)
	}
	return zone, nil
}

func (dnsHandler *NhnDNSHandler) listAllRecordSets(zoneId string) ([]recordsets.RecordSet, error) {
	allPages, err := recordsets.ListByZone(dnsHandler.DNSClient, zoneId, nil).AllPages()
	if err != nil {
		return nil, err
	}
	return recordsets.ExtractRecordSets(allPages)
}

func (dnsHandler *NhnDNSHandler) getDNSZoneInfo(zone *zones.Zone) (irs.DNSZoneInfo, error) {
	rrSets, err := dnsHandler.listAllRecordSets(zone.ID)
	if err != nil {
		return irs.DNSZoneInfo{}, fmt.Errorf("failed to List DNS Record Sets. [%v]", err)
	}

	// NHN Cloud's DNS Plus(Designate-based) API does not expose a separate
	// "nameservers" endpoint in this SDK; the assigned name servers are
	// carried as the values of the NS recordset at the zone apex.
	nameServers := []string{}
	recordInfoList := []irs.DNSRecordInfo{}
	for _, rrSet := range rrSets {
		recordInfoList = append(recordInfoList, convertToDNSRecordInfo(rrSet))
		if rrSet.Type == string(irs.DNSRecordTypeNS) && rrSet.Name == zone.Name {
			nameServers = append(nameServers, rrSet.Records...)
		}
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
