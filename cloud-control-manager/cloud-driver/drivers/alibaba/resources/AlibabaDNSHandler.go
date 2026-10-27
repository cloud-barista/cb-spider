// Alibaba Cloud DNS(alidns) DNSHandler of CB-Spider.
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

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/alidns"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

// AlibabaDNSRegion is the fixed regionId used for the alidns(Alibaba Cloud DNS)
// client. alidns is a global, non-regional product served from a single
// centralized endpoint, so the connection's actual configured region is not
// used when constructing this particular client.
const AlibabaDNSRegion = "cn-hangzhou"

// recordIdSeparator joins the alidns RecordIds that back a single cb-spider
// DNSRecordInfo when a DNSRecordReqInfo.Value had more than one entry (see
// changeDNSRecord()/groupRecordsToDNSRecordInfoList()).
const recordIdSeparator = ","

// alidnsDefaultTTL is used when the caller does not specify a TTL (<=0).
const alidnsDefaultTTL = 600

type AlibabaDNSHandler struct {
	Region idrv.RegionInfo
	Client *alidns.Client
}

func (dnsHandler *AlibabaDNSHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, "DNSZone", "ListIID()")
	start := call.Start()

	domains, err := dnsHandler.listAllDomains()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	iidList := []*irs.IID{}
	for _, domain := range domains {
		iidList = append(iidList, &irs.IID{NameId: domain.DomainName, SystemId: domain.DomainId})
	}

	LoggingInfo(hiscallInfo, start)
	return iidList, nil
}

func (dnsHandler *AlibabaDNSHandler) CreateDNSZone(dnsZoneReqInfo irs.DNSZoneReqInfo) (irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, dnsZoneReqInfo.IId.NameId, "CreateDNSZone()")
	start := call.Start()

	if dnsZoneReqInfo.DomainName == "" {
		err := fmt.Errorf("DomainName is required to create a DNS Zone")
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	req := alidns.CreateAddDomainRequest()
	req.Scheme = "https"
	req.DomainName = dnsZoneReqInfo.DomainName

	if _, err := dnsHandler.Client.AddDomain(req); err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	info, err := dnsHandler.getDNSZoneInfo(dnsZoneReqInfo.DomainName)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func (dnsHandler *AlibabaDNSHandler) ListDNSZone() ([]*irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, "DNSZone", "ListDNSZone()")
	start := call.Start()

	domains, err := dnsHandler.listAllDomains()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	infoList := []*irs.DNSZoneInfo{}
	for _, domain := range domains {
		info, err := dnsHandler.getDNSZoneInfo(domain.DomainName)
		if err != nil {
			LoggingError(hiscallInfo, err)
			return nil, err
		}
		infoList = append(infoList, &info)
	}

	LoggingInfo(hiscallInfo, start)
	return infoList, nil
}

func (dnsHandler *AlibabaDNSHandler) GetDNSZone(dnsZoneIID irs.IID) (irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, dnsZoneIID.NameId, "GetDNSZone()")
	start := call.Start()

	domainName := dnsZoneDomainName(dnsZoneIID)
	if domainName == "" {
		err := fmt.Errorf("DNSZone IId must have a NameId(the zone's DomainName)")
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	info, err := dnsHandler.getDNSZoneInfo(domainName)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func (dnsHandler *AlibabaDNSHandler) DeleteDNSZone(dnsZoneIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, dnsZoneIID.NameId, "DeleteDNSZone()")
	start := call.Start()

	domainName := dnsZoneDomainName(dnsZoneIID)
	if domainName == "" {
		err := fmt.Errorf("DNSZone IId must have a NameId(the zone's DomainName)")
		LoggingError(hiscallInfo, err)
		return false, err
	}

	// NOTE on cascade behavior: Alibaba's alidns API documentation does not
	// clearly guarantee that DeleteDomain cascades to remove the domain's
	// DNS records (unlike, say, deleting a whole hosted zone in some other
	// CSPs). To be safe regardless of actual cascading behavior, all of the
	// domain's records are removed individually first, then the domain
	// itself is deleted.
	records, err := dnsHandler.listAllDomainRecords(domainName)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}
	for _, record := range records {
		delReq := alidns.CreateDeleteDomainRecordRequest()
		delReq.Scheme = "https"
		delReq.RecordId = record.RecordId
		if _, err := dnsHandler.Client.DeleteDomainRecord(delReq); err != nil {
			LoggingError(hiscallInfo, err)
			return false, err
		}
	}

	req := alidns.CreateDeleteDomainRequest()
	req.Scheme = "https"
	req.DomainName = domainName

	if _, err := dnsHandler.Client.DeleteDomain(req); err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	LoggingInfo(hiscallInfo, start)
	return true, nil
}

func (dnsHandler *AlibabaDNSHandler) AddDNSRecord(dnsZoneIID irs.IID, recordReqInfo irs.DNSRecordReqInfo) (irs.DNSZoneInfo, error) {
	return dnsHandler.changeDNSRecord(dnsZoneIID, recordReqInfo, "AddDNSRecord()", false)
}

func (dnsHandler *AlibabaDNSHandler) ChangeDNSRecord(dnsZoneIID irs.IID, recordReqInfo irs.DNSRecordReqInfo) (irs.DNSZoneInfo, error) {
	return dnsHandler.changeDNSRecord(dnsZoneIID, recordReqInfo, "ChangeDNSRecord()", true)
}

// changeDNSRecord implements both AddDNSRecord(isUpsert=false, fails if the
// RR+Type already exists) and ChangeDNSRecord(isUpsert=true, replaces the
// full value set of the RR+Type). alidns has no native multi-value record,
// so each entry of recordReqInfo.Value becomes its own AddDomainRecord call,
// and (for upsert) any pre-existing records sharing the RR+Type are deleted
// first so the end result matches exactly the requested value set.
func (dnsHandler *AlibabaDNSHandler) changeDNSRecord(dnsZoneIID irs.IID, recordReqInfo irs.DNSRecordReqInfo, apiName string, isUpsert bool) (irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, recordReqInfo.Name, apiName)
	start := call.Start()

	domainName := dnsZoneDomainName(dnsZoneIID)
	if domainName == "" {
		err := fmt.Errorf("DNSZone IId must have a NameId(the zone's DomainName)")
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}
	if len(recordReqInfo.Value) == 0 {
		err := fmt.Errorf("at least one Value is required for a DNS record")
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	rr, err := toRR(recordReqInfo.Name, domainName)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	existing, err := dnsHandler.listAllDomainRecords(domainName)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	var matched []alidns.Record
	for _, record := range existing {
		if record.RR == rr && record.Type == string(recordReqInfo.Type) {
			matched = append(matched, record)
		}
	}

	if isUpsert {
		for _, record := range matched {
			delReq := alidns.CreateDeleteDomainRecordRequest()
			delReq.Scheme = "https"
			delReq.RecordId = record.RecordId
			if _, err := dnsHandler.Client.DeleteDomainRecord(delReq); err != nil {
				LoggingError(hiscallInfo, err)
				return irs.DNSZoneInfo{}, err
			}
		}
	} else if len(matched) > 0 {
		err := fmt.Errorf("DNS record '%s'(%s) already exists", recordReqInfo.Name, recordReqInfo.Type)
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	ttl := recordReqInfo.TTL
	if ttl <= 0 {
		ttl = alidnsDefaultTTL // CB-Spider default when the caller does not specify one
	}

	for _, value := range recordReqInfo.Value {
		addReq := alidns.CreateAddDomainRecordRequest()
		addReq.Scheme = "https"
		addReq.DomainName = domainName
		addReq.RR = rr
		addReq.Type = string(recordReqInfo.Type)
		addReq.Value = value
		addReq.TTL = requests.NewInteger(ttl)

		if _, err := dnsHandler.Client.AddDomainRecord(addReq); err != nil {
			LoggingError(hiscallInfo, err)
			return irs.DNSZoneInfo{}, err
		}
	}

	info, err := dnsHandler.getDNSZoneInfo(domainName)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func (dnsHandler *AlibabaDNSHandler) RemoveDNSRecord(dnsZoneIID irs.IID, recordIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, recordIID.NameId, "RemoveDNSRecord()")
	start := call.Start()

	if recordIID.SystemId == "" {
		err := fmt.Errorf("DNSRecord IId must have a SystemId(the alidns RecordId)")
		LoggingError(hiscallInfo, err)
		return false, err
	}

	// recordIID.SystemId may hold several comma-joined alidns RecordIds when
	// the record was created from a multi-value DNSRecordReqInfo.
	for _, recordId := range strings.Split(recordIID.SystemId, recordIdSeparator) {
		recordId = strings.TrimSpace(recordId)
		if recordId == "" {
			continue
		}
		delReq := alidns.CreateDeleteDomainRecordRequest()
		delReq.Scheme = "https"
		delReq.RecordId = recordId
		if _, err := dnsHandler.Client.DeleteDomainRecord(delReq); err != nil {
			LoggingError(hiscallInfo, err)
			return false, err
		}
	}

	LoggingInfo(hiscallInfo, start)
	return true, nil
}

//------------------------- internal helpers -------------------------

func (dnsHandler *AlibabaDNSHandler) listAllDomains() ([]alidns.DomainInDescribeDomains, error) {
	domains := []alidns.DomainInDescribeDomains{}
	curPage := 1
	for {
		req := alidns.CreateDescribeDomainsRequest()
		req.Scheme = "https"
		req.PageNumber = requests.NewInteger(curPage)
		req.PageSize = requests.NewInteger(CBPageSize)

		result, err := dnsHandler.Client.DescribeDomains(req)
		if err != nil {
			return nil, err
		}
		domains = append(domains, result.Domains.Domain...)
		if len(result.Domains.Domain) == 0 || int64(len(domains)) >= result.TotalCount {
			break
		}
		curPage++
	}
	return domains, nil
}

func (dnsHandler *AlibabaDNSHandler) listAllDomainRecords(domainName string) ([]alidns.Record, error) {
	records := []alidns.Record{}
	curPage := 1
	for {
		req := alidns.CreateDescribeDomainRecordsRequest()
		req.Scheme = "https"
		req.DomainName = domainName
		req.PageNumber = requests.NewInteger(curPage)
		req.PageSize = requests.NewInteger(CBPageSize)

		result, err := dnsHandler.Client.DescribeDomainRecords(req)
		if err != nil {
			return nil, err
		}
		records = append(records, result.DomainRecords.Record...)
		if len(result.DomainRecords.Record) == 0 || int64(len(records)) >= result.TotalCount {
			break
		}
		curPage++
	}
	return records, nil
}

func (dnsHandler *AlibabaDNSHandler) getDNSZoneInfo(domainName string) (irs.DNSZoneInfo, error) {
	infoReq := alidns.CreateDescribeDomainInfoRequest()
	infoReq.Scheme = "https"
	infoReq.DomainName = domainName

	domainInfo, err := dnsHandler.Client.DescribeDomainInfo(infoReq)
	if err != nil {
		return irs.DNSZoneInfo{}, err
	}

	records, err := dnsHandler.listAllDomainRecords(domainName)
	if err != nil {
		return irs.DNSZoneInfo{}, err
	}

	return irs.DNSZoneInfo{
		IId:            irs.IID{NameId: domainName, SystemId: domainInfo.DomainId},
		DomainName:     domainName,
		NameServers:    domainInfo.DnsServers.DnsServer,
		RecordInfoList: groupRecordsToDNSRecordInfoList(records, domainName),
	}, nil
}

// groupRecordsToDNSRecordInfoList groups alidns Records sharing the same
// RR+Type into a single irs.DNSRecordInfo, since cb-spider's DNSRecordInfo
// models a record as one Name/Type with potentially multiple Values, while
// each alidns Record only ever carries a single Value and its own RecordId.
// The grouped DNSRecordInfo's IId.SystemId becomes the comma-joined list of
// those RecordIds (see recordIdSeparator), which RemoveDNSRecord/
// changeDNSRecord know how to split/operate on again.
func groupRecordsToDNSRecordInfoList(records []alidns.Record, domainName string) []irs.DNSRecordInfo {
	order := []string{}
	grouped := map[string]*irs.DNSRecordInfo{}
	recordIds := map[string][]string{}

	for _, record := range records {
		key := record.RR + "\x00" + record.Type
		info, ok := grouped[key]
		if !ok {
			name := toFQDN(record.RR, domainName)
			info = &irs.DNSRecordInfo{
				IId:  irs.IID{NameId: name},
				Type: irs.DNSRecordType(record.Type),
				Name: name,
				TTL:  int(record.TTL),
			}
			grouped[key] = info
			order = append(order, key)
		}
		info.Value = append(info.Value, record.Value)
		recordIds[key] = append(recordIds[key], record.RecordId)
	}

	infoList := []irs.DNSRecordInfo{}
	for _, key := range order {
		info := grouped[key]
		info.IId.SystemId = strings.Join(recordIds[key], recordIdSeparator)
		infoList = append(infoList, *info)
	}
	return infoList
}

// toRR converts a user-facing FQDN(e.g. "www.example.com") into the alidns
// "RR"(resource record: the host/subdomain portion relative to the zone's
// DomainName) -- "@" for the zone apex record itself.
func toRR(name string, domainName string) (string, error) {
	name = strings.TrimSuffix(strings.TrimSpace(name), ".")
	domainName = strings.TrimSuffix(strings.TrimSpace(domainName), ".")

	if strings.EqualFold(name, domainName) {
		return "@", nil
	}
	suffix := "." + domainName
	if len(name) > len(suffix) && strings.HasSuffix(strings.ToLower(name), strings.ToLower(suffix)) {
		return name[:len(name)-len(suffix)], nil
	}
	return "", fmt.Errorf("record Name '%s' is not the zone's DomainName('%s') nor a subdomain of it", name, domainName)
}

// toFQDN is the inverse of toRR: it reconstructs the full domain name that
// cb-spider's DNSRecordInfo.Name expects, from an alidns RR plus the zone's
// DomainName.
func toFQDN(rr string, domainName string) string {
	if rr == "@" {
		return domainName
	}
	return rr + "." + domainName
}

// dnsZoneDomainName resolves the alidns DomainName to operate on from a DNS
// Zone IId. alidns identifies a zone by its DomainName string itself -- not
// by the DomainId stored in IId.SystemId, which the zone/record APIs used
// here do not accept as an input -- so IId.NameId(holding the DomainName) is
// used, falling back to SystemId only if NameId was left empty.
func dnsZoneDomainName(dnsZoneIID irs.IID) string {
	if dnsZoneIID.NameId != "" {
		return dnsZoneIID.NameId
	}
	return dnsZoneIID.SystemId
}
