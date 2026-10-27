// Tencent Cloud DNSPod DNSHandler of CB-Spider.
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
	"fmt"
	"strconv"
	"strings"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	dnspod "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/dnspod/v20210323"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type TencentDNSHandler struct {
	Region idrv.RegionInfo
	Client *dnspod.Client
}

// defaultRecordLine is the DNSPod "line"(ISP routing) value required on every
// record create/modify call. CB-Spider only supports the baseline, single
// answer-set DNS model, so every record is created on the default line.
//
// DNSPod exposes this product on two API domains: the mainland-China
// endpoint(dnspod.tencentcloudapi.com), whose line names are the Chinese
// literal "默认", and the international endpoint
// (dnspod.intl.tencentcloudapi.com), whose line names are English, e.g.
// "Default". This driver talks to the international endpoint(see
// getDNSPodClient() in TencentDriver.go) and therefore uses "Default".
const defaultRecordLine = "Default"

// listMaxLimit is the page size used for DescribeDomainList/DescribeRecordList.
const listMaxLimit = 3000

func (dnsHandler *TencentDNSHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, "DNSZone", "ListIID()")
	start := call.Start()

	domains, err := dnsHandler.listAllDomains()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	iidList := []*irs.IID{}
	for _, domain := range domains {
		iidList = append(iidList, &irs.IID{
			NameId:   strVal(domain.Name),
			SystemId: u64Str(domain.DomainId),
		})
	}

	LoggingInfo(hiscallInfo, start)
	return iidList, nil
}

func (dnsHandler *TencentDNSHandler) CreateDNSZone(dnsZoneReqInfo irs.DNSZoneReqInfo) (irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, dnsZoneReqInfo.IId.NameId, "CreateDNSZone()")
	start := call.Start()

	if dnsZoneReqInfo.DomainName == "" {
		err := fmt.Errorf("DomainName is required to create a DNS Zone")
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	req := dnspod.NewCreateDomainRequest()
	req.Domain = common.StringPtr(dnsZoneReqInfo.DomainName)

	result, err := dnsHandler.Client.CreateDomain(req)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}
	if result.Response == nil || result.Response.DomainInfo == nil {
		err := fmt.Errorf("CreateDomain returned no DomainInfo for '%s'", dnsZoneReqInfo.DomainName)
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	domainId := u64Val(result.Response.DomainInfo.Id)
	info, err := dnsHandler.getDNSZoneInfo(domainId, strVal(result.Response.DomainInfo.Domain))
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	// CreateDomain's own response already carries the assigned name servers
	// (GradeNsList); prefer it in case a freshly created zone's DescribeDomain
	// call has not yet been fully consistent.
	if len(info.NameServers) == 0 {
		info.NameServers = strSliceVal(result.Response.DomainInfo.GradeNsList)
	}

	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func (dnsHandler *TencentDNSHandler) ListDNSZone() ([]*irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, "DNSZone", "ListDNSZone()")
	start := call.Start()

	domains, err := dnsHandler.listAllDomains()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	infoList := []*irs.DNSZoneInfo{}
	for _, domain := range domains {
		info, err := dnsHandler.getDNSZoneInfo(u64Val(domain.DomainId), strVal(domain.Name))
		if err != nil {
			LoggingError(hiscallInfo, err)
			return nil, err
		}
		infoList = append(infoList, &info)
	}

	LoggingInfo(hiscallInfo, start)
	return infoList, nil
}

func (dnsHandler *TencentDNSHandler) GetDNSZone(dnsZoneIID irs.IID) (irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, dnsZoneIID.NameId, "GetDNSZone()")
	start := call.Start()

	domainId, err := parseDomainId(dnsZoneIID.SystemId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	info, err := dnsHandler.getDNSZoneInfo(domainId, dnsZoneIID.NameId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func (dnsHandler *TencentDNSHandler) DeleteDNSZone(dnsZoneIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, dnsZoneIID.NameId, "DeleteDNSZone()")
	start := call.Start()

	domainId, err := parseDomainId(dnsZoneIID.SystemId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	// NOTE: Tencent's DNSPod API documentation does not explicitly state
	// whether DeleteDomain cascades and removes the domain's DNS records, or
	// whether records must be removed individually beforehand. To be safe
	// (and consistent with the AWS Route53 reference implementation, which
	// must clean up records before deleting the zone), this driver always
	// deletes every record first via DescribeRecordList + DeleteRecord, and
	// only then calls DeleteDomain. This is correct regardless of whether
	// DeleteDomain would have cascaded on its own.
	records, err := dnsHandler.listAllRecords(domainId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}
	for _, record := range records {
		delReq := dnspod.NewDeleteRecordRequest()
		delReq.DomainId = common.Uint64Ptr(domainId)
		delReq.RecordId = record.RecordId
		if _, err := dnsHandler.Client.DeleteRecord(delReq); err != nil {
			LoggingError(hiscallInfo, err)
			return false, err
		}
	}

	req := dnspod.NewDeleteDomainRequest()
	req.DomainId = common.Uint64Ptr(domainId)
	if _, err := dnsHandler.Client.DeleteDomain(req); err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	LoggingInfo(hiscallInfo, start)
	return true, nil
}

func (dnsHandler *TencentDNSHandler) AddDNSRecord(dnsZoneIID irs.IID, recordReqInfo irs.DNSRecordReqInfo) (irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, recordReqInfo.Name, "AddDNSRecord()")
	start := call.Start()

	domainId, err := parseDomainId(dnsZoneIID.SystemId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}
	if len(recordReqInfo.Value) == 0 {
		err := fmt.Errorf("at least one Value is required for a DNS record")
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	zoneDomainName := dnsZoneIID.NameId
	if zoneDomainName == "" {
		zone, err := dnsHandler.describeDomain(domainId, "")
		if err != nil {
			LoggingError(hiscallInfo, err)
			return irs.DNSZoneInfo{}, err
		}
		zoneDomainName = strVal(zone.Domain)
	}
	subDomain := subDomainFromFQDN(recordReqInfo.Name, zoneDomainName)

	// DNSPod models each record value as a single record; CB-Spider's
	// DNSRecordReqInfo.Value can carry multiple values(ex: round-robin A
	// records), so one DNSPod record is created per value and all of the
	// resulting RecordIds back the same CB-Spider logical record set under
	// the same Name+Type.
	for _, value := range recordReqInfo.Value {
		req := dnspod.NewCreateRecordRequest()
		req.DomainId = common.Uint64Ptr(domainId)
		req.RecordType = common.StringPtr(string(recordReqInfo.Type))
		req.RecordLine = common.StringPtr(defaultRecordLine)
		req.Value = common.StringPtr(value)
		req.SubDomain = common.StringPtr(subDomain)
		if recordReqInfo.TTL > 0 {
			req.TTL = common.Uint64Ptr(uint64(recordReqInfo.TTL))
		}

		if _, err := dnsHandler.Client.CreateRecord(req); err != nil {
			LoggingError(hiscallInfo, err)
			return irs.DNSZoneInfo{}, err
		}
	}

	info, err := dnsHandler.getDNSZoneInfo(domainId, zoneDomainName)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func (dnsHandler *TencentDNSHandler) ChangeDNSRecord(dnsZoneIID irs.IID, recordReqInfo irs.DNSRecordReqInfo) (irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, recordReqInfo.Name, "ChangeDNSRecord()")
	start := call.Start()

	domainId, err := parseDomainId(dnsZoneIID.SystemId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}
	if len(recordReqInfo.Value) == 0 {
		err := fmt.Errorf("at least one Value is required for a DNS record")
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	zoneDomainName := dnsZoneIID.NameId
	if zoneDomainName == "" {
		zone, err := dnsHandler.describeDomain(domainId, "")
		if err != nil {
			LoggingError(hiscallInfo, err)
			return irs.DNSZoneInfo{}, err
		}
		zoneDomainName = strVal(zone.Domain)
	}
	subDomain := subDomainFromFQDN(recordReqInfo.Name, zoneDomainName)

	// Find every existing DNSPod record that backs this Name+Type CB-Spider
	// record (there may be more than one when the record set is multi-value).
	records, err := dnsHandler.listAllRecords(domainId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}
	existing := []*dnspod.RecordListItem{}
	for _, record := range records {
		if strVal(record.Name) == subDomain && strVal(record.Type) == string(recordReqInfo.Type) {
			existing = append(existing, record)
		}
	}

	ttl := uint64(recordReqInfo.TTL)

	// Reuse existing RecordIds for as many new values as possible(so
	// RecordId/SystemId stays stable across a Change call when counts match
	// or shrink), then create new records for any extra values, and delete
	// any existing records left over when the new value count is smaller.
	for i, value := range recordReqInfo.Value {
		if i < len(existing) {
			req := dnspod.NewModifyRecordRequest()
			req.DomainId = common.Uint64Ptr(domainId)
			req.RecordId = existing[i].RecordId
			req.RecordType = common.StringPtr(string(recordReqInfo.Type))
			req.RecordLine = common.StringPtr(defaultRecordLine)
			req.Value = common.StringPtr(value)
			req.SubDomain = common.StringPtr(subDomain)
			if ttl > 0 {
				req.TTL = common.Uint64Ptr(ttl)
			}
			if _, err := dnsHandler.Client.ModifyRecord(req); err != nil {
				LoggingError(hiscallInfo, err)
				return irs.DNSZoneInfo{}, err
			}
			continue
		}

		req := dnspod.NewCreateRecordRequest()
		req.DomainId = common.Uint64Ptr(domainId)
		req.RecordType = common.StringPtr(string(recordReqInfo.Type))
		req.RecordLine = common.StringPtr(defaultRecordLine)
		req.Value = common.StringPtr(value)
		req.SubDomain = common.StringPtr(subDomain)
		if ttl > 0 {
			req.TTL = common.Uint64Ptr(ttl)
		}
		if _, err := dnsHandler.Client.CreateRecord(req); err != nil {
			LoggingError(hiscallInfo, err)
			return irs.DNSZoneInfo{}, err
		}
	}
	for i := len(recordReqInfo.Value); i < len(existing); i++ {
		delReq := dnspod.NewDeleteRecordRequest()
		delReq.DomainId = common.Uint64Ptr(domainId)
		delReq.RecordId = existing[i].RecordId
		if _, err := dnsHandler.Client.DeleteRecord(delReq); err != nil {
			LoggingError(hiscallInfo, err)
			return irs.DNSZoneInfo{}, err
		}
	}

	info, err := dnsHandler.getDNSZoneInfo(domainId, zoneDomainName)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func (dnsHandler *TencentDNSHandler) RemoveDNSRecord(dnsZoneIID irs.IID, recordIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, recordIID.NameId, "RemoveDNSRecord()")
	start := call.Start()

	domainId, err := parseDomainId(dnsZoneIID.SystemId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}
	recordId, err := strconv.ParseUint(recordIID.SystemId, 10, 64)
	if err != nil {
		err := fmt.Errorf("invalid DNS record SystemId: %s", recordIID.SystemId)
		LoggingError(hiscallInfo, err)
		return false, err
	}

	req := dnspod.NewDeleteRecordRequest()
	req.DomainId = common.Uint64Ptr(domainId)
	req.RecordId = common.Uint64Ptr(recordId)

	if _, err := dnsHandler.Client.DeleteRecord(req); err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	LoggingInfo(hiscallInfo, start)
	return true, nil
}

//------------------------- internal helpers -------------------------

func (dnsHandler *TencentDNSHandler) listAllDomains() ([]*dnspod.DomainListItem, error) {
	domains := []*dnspod.DomainListItem{}
	var offset int64 = 0
	for {
		req := dnspod.NewDescribeDomainListRequest()
		req.Offset = common.Int64Ptr(offset)
		req.Limit = common.Int64Ptr(int64(listMaxLimit))

		result, err := dnsHandler.Client.DescribeDomainList(req)
		if err != nil {
			return nil, err
		}
		if result.Response == nil {
			break
		}
		domains = append(domains, result.Response.DomainList...)

		total := uint64(0)
		if result.Response.DomainCountInfo != nil {
			total = u64Val(result.Response.DomainCountInfo.DomainTotal)
		}
		offset += int64(len(result.Response.DomainList))
		if len(result.Response.DomainList) == 0 || uint64(offset) >= total {
			break
		}
	}
	return domains, nil
}

func (dnsHandler *TencentDNSHandler) listAllRecords(domainId uint64) ([]*dnspod.RecordListItem, error) {
	records := []*dnspod.RecordListItem{}
	var offset uint64 = 0
	for {
		req := dnspod.NewDescribeRecordListRequest()
		req.DomainId = common.Uint64Ptr(domainId)
		req.Offset = common.Uint64Ptr(offset)
		req.Limit = common.Uint64Ptr(uint64(listMaxLimit))
		// Do not error out for a brand-new zone that has no records yet.
		req.ErrorOnEmpty = common.StringPtr("no")

		result, err := dnsHandler.Client.DescribeRecordList(req)
		if err != nil {
			return nil, err
		}
		if result.Response == nil {
			break
		}
		records = append(records, result.Response.RecordList...)

		total := uint64(0)
		if result.Response.RecordCountInfo != nil {
			total = u64Val(result.Response.RecordCountInfo.TotalCount)
		}
		offset += uint64(len(result.Response.RecordList))
		if len(result.Response.RecordList) == 0 || offset >= total {
			break
		}
	}
	return records, nil
}

func (dnsHandler *TencentDNSHandler) describeDomain(domainId uint64, domainName string) (*dnspod.DomainInfo, error) {
	req := dnspod.NewDescribeDomainRequest()
	if domainId > 0 {
		req.DomainId = common.Uint64Ptr(domainId)
	} else {
		req.Domain = common.StringPtr(domainName)
	}

	result, err := dnsHandler.Client.DescribeDomain(req)
	if err != nil {
		return nil, err
	}
	if result.Response == nil || result.Response.DomainInfo == nil {
		return nil, fmt.Errorf("DescribeDomain returned no DomainInfo")
	}
	return result.Response.DomainInfo, nil
}

func (dnsHandler *TencentDNSHandler) getDNSZoneInfo(domainId uint64, domainNameHint string) (irs.DNSZoneInfo, error) {
	domain, err := dnsHandler.describeDomain(domainId, domainNameHint)
	if err != nil {
		return irs.DNSZoneInfo{}, err
	}
	domainName := strVal(domain.Domain)
	if domainName == "" {
		domainName = domainNameHint
	}

	nameServers := strSliceVal(domain.DnspodNsList)
	if len(nameServers) == 0 {
		nameServers = strSliceVal(domain.ActualNsList)
	}

	records, err := dnsHandler.listAllRecords(domainId)
	if err != nil {
		return irs.DNSZoneInfo{}, err
	}

	recordInfoList := []irs.DNSRecordInfo{}
	for _, record := range records {
		recordInfoList = append(recordInfoList, convertToDNSRecordInfo(record, domainName))
	}

	return irs.DNSZoneInfo{
		IId:            irs.IID{NameId: domainName, SystemId: strconv.FormatUint(domainId, 10)},
		DomainName:     domainName,
		NameServers:    nameServers,
		RecordInfoList: recordInfoList,
	}, nil
}

func convertToDNSRecordInfo(record *dnspod.RecordListItem, zoneDomainName string) irs.DNSRecordInfo {
	subDomain := strVal(record.Name)
	fqdn := fqdnFromSubDomain(subDomain, zoneDomainName)
	recordId := u64Str(record.RecordId)

	return irs.DNSRecordInfo{
		IId:   irs.IID{NameId: fqdn, SystemId: recordId},
		Type:  irs.DNSRecordType(strVal(record.Type)),
		Name:  fqdn,
		Value: []string{strVal(record.Value)},
		TTL:   int(u64Val(record.TTL)),
	}
}

// subDomainFromFQDN converts a CB-Spider user-facing FQDN(ex: "www.example.com")
// into the DNSPod "SubDomain"(host portion, ex: "www") relative to the zone's
// domain(ex: "example.com"). The zone apex itself maps to "@", DNSPod's
// convention for "no host part".
func subDomainFromFQDN(fqdnName string, zoneDomainName string) string {
	name := strings.TrimSuffix(strings.TrimSpace(fqdnName), ".")
	zone := strings.TrimSuffix(strings.TrimSpace(zoneDomainName), ".")

	if name == zone || name == "" {
		return "@"
	}
	suffix := "." + zone
	if strings.HasSuffix(name, suffix) {
		return strings.TrimSuffix(name, suffix)
	}
	// Not under the zone's domain(shouldn't normally happen); pass through
	// as-is so DNSPod can report a clear error rather than CB-Spider silently
	// mangling it.
	return name
}

// fqdnFromSubDomain is the inverse of subDomainFromFQDN: it reconstructs the
// user-facing FQDN from the DNSPod "SubDomain" and the zone's domain name.
func fqdnFromSubDomain(subDomain string, zoneDomainName string) string {
	zone := strings.TrimSuffix(strings.TrimSpace(zoneDomainName), ".")
	if subDomain == "" || subDomain == "@" {
		return zone
	}
	return subDomain + "." + zone
}

func parseDomainId(systemId string) (uint64, error) {
	domainId, err := strconv.ParseUint(systemId, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid DNS zone SystemId: %s", systemId)
	}
	return domainId, nil
}

func strVal(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func strSliceVal(p []*string) []string {
	values := []string{}
	for _, v := range p {
		values = append(values, strVal(v))
	}
	return values
}

func u64Val(p *uint64) uint64 {
	if p == nil {
		return 0
	}
	return *p
}

func u64Str(p *uint64) string {
	if p == nil {
		return ""
	}
	return strconv.FormatUint(*p, 10)
}
