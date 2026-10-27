// IBM Cloud Internet Services(CIS) DNSHandler of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This covers only the Public Authoritative DNS baseline (Zone + Record CRUD).
//
// IMPORTANT product note: IBM Cloud has two different DNS products. This handler
// talks to CIS(Cloud Internet Services, Cloudflare-based) via the zonesv1/
// dnsrecordsv1 SDK packages, which is IBM's PUBLIC authoritative DNS product.
// It deliberately does NOT use the dnszonesv1/dnssvcsv1 "DNS Services" packages,
// which back a PRIVATE-only DNS product that is not resolvable from the public
// internet and is out of scope for this handler.
//
// by CB-Spider Team, 2026.10.

package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/networking-go-sdk/dnsrecordsv1"
	"github.com/IBM/networking-go-sdk/zonesv1"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

// ibmCISPageSize is the page size used when paginating CIS Zones/DNS records list APIs.
const ibmCISPageSize = int64(50)

// ibmCISDefaultTTL is used when the caller does not specify a TTL (<=0).
const ibmCISDefaultTTL = int64(300)

// ibmCISRecordIdSeparator joins the CIS DNS record IDs that back a single cb-spider
// DNSRecordInfo when a DNSRecordReqInfo.Value had more than one entry (see
// changeDNSRecord()/groupDNSRecordsToInfoList()). CIS DNS records(like Alibaba/
// Tencent) are single-value-per-record(one "Content" string each), unlike
// Route53's ResourceRecordSet which natively holds a list of values under one
// Name+Type, so a multi-value DNSRecordReqInfo must be fanned out into several
// individual CIS DNS records sharing the same Name+Type.
const ibmCISRecordIdSeparator = ","

type IbmDNSHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo

	// Authenticator is the same IAM authenticator used to build every other
	// IBM service client in this driver (see getIamAuthenticator() in
	// IBMCloudDriver.go). It cannot be used to build the zonesv1/dnsrecordsv1
	// clients up front in ConnectCloud like the other IBM clients, because
	// those clients additionally require a CIS instance CRN at construction
	// time(see getCISInstanceCRN() below) which is not currently available.
	Authenticator core.Authenticator
	Ctx           context.Context
}

// getCISInstanceCRN resolves the CRN of the CIS instance to operate against.
//
// Every CIS(Cloud Internet Services) API call -- indeed, even constructing the
// zonesv1.ZonesV1/dnsrecordsv1.DnsRecordsV1 SDK clients themselves -- requires
// the CRN of one specific, already-provisioned CIS service instance (CIS is a
// multi-tenant-within-your-account product: a single CIS instance can host many
// zones/domains, so the instance identity cannot be inferred from the account's
// ApiKey alone the way, say, a VPC region/zone can). Unlike every other CRN used
// elsewhere in the IBM driver (VPC/SG/VM/Disk/NLB/RDBMS), which are all discovered
// dynamically from a resource the driver itself just created, a CIS instance must
// already exist and be supplied by the caller -- so it is read from
// idrv.CredentialInfo.CISInstanceCRN, set via the connection's credential config.
func (dnsHandler *IbmDNSHandler) getCISInstanceCRN() (string, error) {
	crn := dnsHandler.CredentialInfo.CISInstanceCRN
	if crn == "" {
		return "", errors.New("IBM Driver: DNSHandler requires CredentialInfo.CISInstanceCRN, the CRN of a " +
			"specific CIS(Cloud Internet Services) service instance, to call IBM's Public DNS APIs " +
			"(zonesv1/dnsrecordsv1) -- both SDK clients mark this CRN as a required field. " +
			"Set CISInstanceCRN in the connection's credential configuration")
	}
	return crn, nil
}

// newZonesService lazily builds the CIS Zones client, scoped to the resolved CIS instance CRN.
func (dnsHandler *IbmDNSHandler) newZonesService() (*zonesv1.ZonesV1, error) {
	crn, err := dnsHandler.getCISInstanceCRN()
	if err != nil {
		return nil, err
	}
	return zonesv1.NewZonesV1(&zonesv1.ZonesV1Options{
		Authenticator: dnsHandler.Authenticator,
		Crn:           core.StringPtr(crn),
	})
}

// newDNSRecordsService lazily builds the CIS DNS Records client. Unlike ZonesV1,
// DnsRecordsV1 is scoped to both the CIS instance CRN AND a single zone ID at
// construction time (dnsrecordsv1.DnsRecordsV1Options.ZoneIdentifier is also a
// required field) -- so a new client must be built per zone operated on.
func (dnsHandler *IbmDNSHandler) newDNSRecordsService(zoneId string) (*dnsrecordsv1.DnsRecordsV1, error) {
	crn, err := dnsHandler.getCISInstanceCRN()
	if err != nil {
		return nil, err
	}
	return dnsrecordsv1.NewDnsRecordsV1(&dnsrecordsv1.DnsRecordsV1Options{
		Authenticator:  dnsHandler.Authenticator,
		Crn:            core.StringPtr(crn),
		ZoneIdentifier: core.StringPtr(zoneId),
	})
}

func (dnsHandler *IbmDNSHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, "DNSZone", "ListIID()")
	start := call.Start()

	zonesSvc, err := dnsHandler.newZonesService()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	zoneList, err := dnsHandler.listAllZones(zonesSvc)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	iidList := []*irs.IID{}
	for _, zone := range zoneList {
		iidList = append(iidList, &irs.IID{NameId: ibmDNSStrVal(zone.Name), SystemId: ibmDNSStrVal(zone.ID)})
	}

	LoggingInfo(hiscallInfo, start)
	return iidList, nil
}

func (dnsHandler *IbmDNSHandler) CreateDNSZone(dnsZoneReqInfo irs.DNSZoneReqInfo) (irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, dnsZoneReqInfo.IId.NameId, "CreateDNSZone()")
	start := call.Start()

	if dnsZoneReqInfo.DomainName == "" {
		err := fmt.Errorf("DomainName is required to create a DNS Zone")
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	zonesSvc, err := dnsHandler.newZonesService()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	createOpts := zonesSvc.NewCreateZoneOptions()
	createOpts.SetName(dnsZoneReqInfo.DomainName)

	result, _, err := zonesSvc.CreateZoneWithContext(dnsHandler.Ctx, createOpts)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}
	if result == nil || result.Result == nil {
		err := fmt.Errorf("IBM CIS CreateZone returned no zone details")
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	// NameServers may come back empty immediately after creation -- it can take
	// a short time for Cloudflare/CIS to assign them. A subsequent GetDNSZone
	// will show them once CIS has populated them.
	info, err := dnsHandler.getDNSZoneInfo(result.Result)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func (dnsHandler *IbmDNSHandler) ListDNSZone() ([]*irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, "DNSZone", "ListDNSZone()")
	start := call.Start()

	zonesSvc, err := dnsHandler.newZonesService()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	zoneList, err := dnsHandler.listAllZones(zonesSvc)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	infoList := []*irs.DNSZoneInfo{}
	for i := range zoneList {
		info, err := dnsHandler.getDNSZoneInfo(&zoneList[i])
		if err != nil {
			LoggingError(hiscallInfo, err)
			return nil, err
		}
		infoList = append(infoList, &info)
	}

	LoggingInfo(hiscallInfo, start)
	return infoList, nil
}

func (dnsHandler *IbmDNSHandler) GetDNSZone(dnsZoneIID irs.IID) (irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, dnsZoneIID.NameId, "GetDNSZone()")
	start := call.Start()

	if dnsZoneIID.SystemId == "" {
		err := fmt.Errorf("DNSZone IId must have a SystemId(the CIS zone ID)")
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	zonesSvc, err := dnsHandler.newZonesService()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	zoneDetails, err := dnsHandler.getZoneDetails(zonesSvc, dnsZoneIID.SystemId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	info, err := dnsHandler.getDNSZoneInfo(zoneDetails)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func (dnsHandler *IbmDNSHandler) DeleteDNSZone(dnsZoneIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, dnsZoneIID.NameId, "DeleteDNSZone()")
	start := call.Start()

	if dnsZoneIID.SystemId == "" {
		err := fmt.Errorf("DNSZone IId must have a SystemId(the CIS zone ID)")
		LoggingError(hiscallInfo, err)
		return false, err
	}

	zonesSvc, err := dnsHandler.newZonesService()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	// NOTE on cascade behavior(per the task's verification requirement): CIS's
	// Zones/DNS Records API is built on Cloudflare's API, whose documented
	// DeleteZone semantics are a full cascading teardown of the zone and
	// everything under it(DNS records, page rules, etc.) -- there is no
	// Route53-style restriction requiring non-apex records to be removed
	// individually first before the zone itself can be deleted. So, unlike
	// AwsDNSHandler.DeleteDNSZone, no pre-emptying ListAllDnsRecords+
	// DeleteDnsRecord loop is performed here. This could not be confirmed by an
	// actual end-to-end run against a live CIS instance because of the CRN gap
	// documented on getCISInstanceCRN(); if that assumption ever proves wrong in
	// practice, add a cleanup loop here mirroring AwsDNSHandler/
	// AlibabaDNSHandler's DeleteDNSZone.
	deleteOpts := zonesSvc.NewDeleteZoneOptions(dnsZoneIID.SystemId)
	_, _, err = zonesSvc.DeleteZoneWithContext(dnsHandler.Ctx, deleteOpts)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	LoggingInfo(hiscallInfo, start)
	return true, nil
}

func (dnsHandler *IbmDNSHandler) AddDNSRecord(dnsZoneIID irs.IID, recordReqInfo irs.DNSRecordReqInfo) (irs.DNSZoneInfo, error) {
	return dnsHandler.changeDNSRecord(dnsZoneIID, recordReqInfo, "AddDNSRecord()", false)
}

func (dnsHandler *IbmDNSHandler) ChangeDNSRecord(dnsZoneIID irs.IID, recordReqInfo irs.DNSRecordReqInfo) (irs.DNSZoneInfo, error) {
	return dnsHandler.changeDNSRecord(dnsZoneIID, recordReqInfo, "ChangeDNSRecord()", true)
}

// changeDNSRecord implements both AddDNSRecord(isUpsert=false, fails if the
// Name+Type already exists) and ChangeDNSRecord(isUpsert=true, replaces the full
// value set of the Name+Type). CIS has no native multi-value record(each
// DnsrecordDetails carries exactly one "Content" string), so each entry of
// recordReqInfo.Value becomes its own CreateDnsRecord call, and(for upsert) any
// pre-existing records sharing the Name+Type are deleted first so the end result
// matches exactly the requested value set -- mirroring AlibabaDNSHandler/
// TencentDNSHandler, which face the same single-value-per-record limitation.
//
// recordReqInfo.Name is passed to CIS as-is(a full FQDN, e.g. "www.example.com")
// with no relative-name conversion: unlike Alibaba's alidns(which requires a
// zone-relative "RR") or Azure/Tencent-style APIs, CIS's dnsrecordsv1 CreateDnsRecord/
// ListAllDnsRecords/UpdateDnsRecord Name field accepts the full FQDN directly(see
// the IBM networking-go-sdk's own dns_records_v1_test.go, which exercises
// SetName("host-1.test-example.com") -- not a bare "host-1").
func (dnsHandler *IbmDNSHandler) changeDNSRecord(dnsZoneIID irs.IID, recordReqInfo irs.DNSRecordReqInfo, apiName string, isUpsert bool) (irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, recordReqInfo.Name, apiName)
	start := call.Start()

	if dnsZoneIID.SystemId == "" {
		err := fmt.Errorf("DNSZone IId must have a SystemId(the CIS zone ID)")
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}
	if len(recordReqInfo.Value) == 0 {
		err := fmt.Errorf("at least one Value is required for a DNS record")
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	recordsSvc, err := dnsHandler.newDNSRecordsService(dnsZoneIID.SystemId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	existing, err := dnsHandler.listAllDNSRecords(recordsSvc)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	var matched []dnsrecordsv1.DnsrecordDetails
	for _, record := range existing {
		if strings.EqualFold(ibmDNSStrVal(record.Name), recordReqInfo.Name) &&
			strings.EqualFold(ibmDNSStrVal(record.Type), string(recordReqInfo.Type)) {
			matched = append(matched, record)
		}
	}

	if isUpsert {
		for _, record := range matched {
			deleteOpts := recordsSvc.NewDeleteDnsRecordOptions(ibmDNSStrVal(record.ID))
			if _, _, err := recordsSvc.DeleteDnsRecordWithContext(dnsHandler.Ctx, deleteOpts); err != nil {
				LoggingError(hiscallInfo, err)
				return irs.DNSZoneInfo{}, err
			}
		}
	} else if len(matched) > 0 {
		err := fmt.Errorf("DNS record '%s'(%s) already exists", recordReqInfo.Name, recordReqInfo.Type)
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	ttl := int64(recordReqInfo.TTL)
	if ttl <= 0 {
		ttl = ibmCISDefaultTTL // CB-Spider default when the caller does not specify one
	}

	for _, value := range recordReqInfo.Value {
		createOpts := recordsSvc.NewCreateDnsRecordOptions()
		createOpts.SetName(recordReqInfo.Name)
		createOpts.SetType(string(recordReqInfo.Type))
		createOpts.SetContent(value)
		createOpts.SetTTL(ttl)

		if _, _, err := recordsSvc.CreateDnsRecordWithContext(dnsHandler.Ctx, createOpts); err != nil {
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

func (dnsHandler *IbmDNSHandler) RemoveDNSRecord(dnsZoneIID irs.IID, recordIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, recordIID.NameId, "RemoveDNSRecord()")
	start := call.Start()

	if dnsZoneIID.SystemId == "" {
		err := fmt.Errorf("DNSZone IId must have a SystemId(the CIS zone ID)")
		LoggingError(hiscallInfo, err)
		return false, err
	}
	if recordIID.SystemId == "" {
		err := fmt.Errorf("DNSRecord IId must have a SystemId(the CIS DNS record ID)")
		LoggingError(hiscallInfo, err)
		return false, err
	}

	recordsSvc, err := dnsHandler.newDNSRecordsService(dnsZoneIID.SystemId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	// recordIID.SystemId may hold several comma-joined CIS DNS record IDs when
	// the record was created from a multi-value DNSRecordReqInfo (see
	// groupDNSRecordsToInfoList()).
	for _, recordId := range strings.Split(recordIID.SystemId, ibmCISRecordIdSeparator) {
		recordId = strings.TrimSpace(recordId)
		if recordId == "" {
			continue
		}
		deleteOpts := recordsSvc.NewDeleteDnsRecordOptions(recordId)
		if _, _, err := recordsSvc.DeleteDnsRecordWithContext(dnsHandler.Ctx, deleteOpts); err != nil {
			LoggingError(hiscallInfo, err)
			return false, err
		}
	}

	LoggingInfo(hiscallInfo, start)
	return true, nil
}

//------------------------- internal helpers -------------------------

func (dnsHandler *IbmDNSHandler) listAllZones(zonesSvc *zonesv1.ZonesV1) ([]zonesv1.ZoneDetails, error) {
	allZones := []zonesv1.ZoneDetails{}
	page := int64(1)
	for {
		listOpts := zonesSvc.NewListZonesOptions()
		listOpts.SetPage(page)
		listOpts.SetPerPage(ibmCISPageSize)

		result, _, err := zonesSvc.ListZonesWithContext(dnsHandler.Ctx, listOpts)
		if err != nil {
			return nil, err
		}
		if result == nil {
			break
		}
		allZones = append(allZones, result.Result...)
		if len(result.Result) == 0 || result.ResultInfo == nil || result.ResultInfo.TotalCount == nil ||
			int64(len(allZones)) >= *result.ResultInfo.TotalCount {
			break
		}
		page++
	}
	return allZones, nil
}

func (dnsHandler *IbmDNSHandler) listAllDNSRecords(recordsSvc *dnsrecordsv1.DnsRecordsV1) ([]dnsrecordsv1.DnsrecordDetails, error) {
	allRecords := []dnsrecordsv1.DnsrecordDetails{}
	page := int64(1)
	for {
		listOpts := recordsSvc.NewListAllDnsRecordsOptions()
		listOpts.SetPage(page)
		listOpts.SetPerPage(ibmCISPageSize)

		result, _, err := recordsSvc.ListAllDnsRecordsWithContext(dnsHandler.Ctx, listOpts)
		if err != nil {
			return nil, err
		}
		if result == nil {
			break
		}
		allRecords = append(allRecords, result.Result...)
		if len(result.Result) == 0 || result.ResultInfo == nil || result.ResultInfo.TotalCount == nil ||
			int64(len(allRecords)) >= *result.ResultInfo.TotalCount {
			break
		}
		page++
	}
	return allRecords, nil
}

func (dnsHandler *IbmDNSHandler) getZoneDetails(zonesSvc *zonesv1.ZonesV1, zoneId string) (*zonesv1.ZoneDetails, error) {
	getOpts := zonesSvc.NewGetZoneOptions(zoneId)
	result, _, err := zonesSvc.GetZoneWithContext(dnsHandler.Ctx, getOpts)
	if err != nil {
		return nil, err
	}
	if result == nil || result.Result == nil {
		return nil, fmt.Errorf("IBM CIS GetZone returned no zone details for zone id '%s'", zoneId)
	}
	return result.Result, nil
}

func (dnsHandler *IbmDNSHandler) getDNSZoneInfo(zoneDetails *zonesv1.ZoneDetails) (irs.DNSZoneInfo, error) {
	if zoneDetails == nil || zoneDetails.ID == nil || *zoneDetails.ID == "" {
		return irs.DNSZoneInfo{}, fmt.Errorf("IBM CIS zone response did not include a zone ID")
	}

	recordsSvc, err := dnsHandler.newDNSRecordsService(*zoneDetails.ID)
	if err != nil {
		return irs.DNSZoneInfo{}, err
	}

	records, err := dnsHandler.listAllDNSRecords(recordsSvc)
	if err != nil {
		return irs.DNSZoneInfo{}, err
	}

	return irs.DNSZoneInfo{
		IId:            irs.IID{NameId: ibmDNSStrVal(zoneDetails.Name), SystemId: ibmDNSStrVal(zoneDetails.ID)},
		DomainName:     ibmDNSStrVal(zoneDetails.Name),
		NameServers:    zoneDetails.NameServers,
		RecordInfoList: groupDNSRecordsToInfoList(records),
	}, nil
}

// groupDNSRecordsToInfoList groups CIS DnsrecordDetails sharing the same
// Name+Type into a single irs.DNSRecordInfo, since cb-spider's DNSRecordInfo
// models a record as one Name/Type with potentially multiple Values, while each
// CIS DnsrecordDetails only ever carries a single Content value and its own ID.
// The grouped DNSRecordInfo's IId.SystemId becomes the comma-joined list of
// those record IDs (see ibmCISRecordIdSeparator), which RemoveDNSRecord/
// changeDNSRecord know how to split/operate on again.
func groupDNSRecordsToInfoList(records []dnsrecordsv1.DnsrecordDetails) []irs.DNSRecordInfo {
	order := []string{}
	grouped := map[string]*irs.DNSRecordInfo{}
	recordIds := map[string][]string{}

	for _, record := range records {
		name := ibmDNSStrVal(record.Name)
		recordType := ibmDNSStrVal(record.Type)
		key := name + "\x00" + recordType
		info, ok := grouped[key]
		if !ok {
			info = &irs.DNSRecordInfo{
				IId:  irs.IID{NameId: name},
				Type: irs.DNSRecordType(recordType),
				Name: name,
				TTL:  int(ibmDNSInt64Val(record.TTL)),
			}
			grouped[key] = info
			order = append(order, key)
		}
		info.Value = append(info.Value, ibmDNSStrVal(record.Content))
		recordIds[key] = append(recordIds[key], ibmDNSStrVal(record.ID))
	}

	infoList := []irs.DNSRecordInfo{}
	for _, key := range order {
		info := grouped[key]
		info.IId.SystemId = strings.Join(recordIds[key], ibmCISRecordIdSeparator)
		infoList = append(infoList, *info)
	}
	return infoList
}

func ibmDNSStrVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ibmDNSInt64Val(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}
