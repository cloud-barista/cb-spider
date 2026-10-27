// NCP Global DNS DNSHandler of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This covers only the Public Authoritative DNS baseline (Zone + Record CRUD).
//
// NOTE: NCP's official Go SDK(github.com/NaverCloudPlatform/ncloud-sdk-go-v2) does NOT
// provide a Global DNS service client, so this handler calls the Global DNS REST API
// directly over net/http, using NCP's standard API Gateway request-signing scheme
// (the same scheme used by every other NCP REST API).
//
// by CB-Spider Team, 2026.09.

package resources

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

// ncpDNSBaseURL is the fixed Global DNS API Gateway endpoint (same for all regions).
const ncpDNSBaseURL = "https://globaldns.apigw.ntruss.com"

// ncpDNSPageSize is the page size used for every paginated list call.
const ncpDNSPageSize = 100

type NcpVpcDNSHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
}

//------------------------- wire/JSON types -------------------------

type ncpDNSDomainCreateRequest struct {
	Name string `json:"name"`
}

type ncpDNSDomain struct {
	Id         int64  `json:"id"`
	Name       string `json:"name"`
	CompleteYn bool   `json:"completeYn"`
	DnssecYn   bool   `json:"dnssecYn"`
	Status     string `json:"status"`
}

type ncpDNSDomainListResponse struct {
	Content []ncpDNSDomain `json:"content"`
	Last    bool           `json:"last"`
}

type ncpDNSSoa struct {
	NameServers string `json:"nameServers"`
	DomainName  string `json:"domainName"`
	Serial      int64  `json:"serial"`
	Refresh     int64  `json:"refresh"`
	Retry       int64  `json:"retry"`
	Expire      int64  `json:"expire"`
	Ttl         int64  `json:"ttl"`
	DnssecYn    bool   `json:"dnssecYn"`
}

type ncpDNSDomainDetail struct {
	Id          int64     `json:"id"`
	Name        string    `json:"name"`
	CompleteYn  bool      `json:"completeYn"`
	DnssecYn    bool      `json:"dnssecYn"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	RecordCount int       `json:"recordCount"`
	Soa         ncpDNSSoa `json:"soa"`
}

type ncpDNSRecord struct {
	Id           int64  `json:"id"`
	Name         string `json:"name"`
	Host         string `json:"host"`
	Type         string `json:"type"`
	Content      string `json:"content"`
	Ttl          int64  `json:"ttl"`
	DomainName   string `json:"domainName"`
	AliasYn      bool   `json:"aliasYn"`
	LbYn         bool   `json:"lbYn"`
	LbRegionCode string `json:"lbRegionCode"`
	ApplyYn      bool   `json:"applyYn"`
	DefaultYn    bool   `json:"defaultYn"`
	DelYn        bool   `json:"delYn"`
}

type ncpDNSRecordListResponse struct {
	Content []ncpDNSRecord `json:"content"`
	Last    bool           `json:"last"`
}

// ncpDNSRecordCreateItem is one element of the JSON array the
// POST /dns/v1/ncpdns/record/{domainId} API expects as its request body.
type ncpDNSRecordCreateItem struct {
	Host    string `json:"host"`
	Type    string `json:"type"`
	Content string `json:"content"`
	Ttl     int64  `json:"ttl"`
}

type ncpDNSErrorEnvelope struct {
	Result string `json:"result"`
	Error  struct {
		ErrorCode  string `json:"errorCode"`
		Message    string `json:"message"`
		DevMessage string `json:"devMessage"`
	} `json:"error"`
}

//------------------------- interface methods -------------------------

func (dnsHandler *NcpVpcDNSHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.RegionInfo.Zone, call.DNSZONE, "DNSZone", "ListIID()")
	start := call.Start()

	domains, err := dnsHandler.listAllDomains()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	iidList := []*irs.IID{}
	for _, d := range domains {
		iidList = append(iidList, &irs.IID{NameId: d.Name, SystemId: strconv.FormatInt(d.Id, 10)})
	}

	LoggingInfo(hiscallInfo, start)
	return iidList, nil
}

func (dnsHandler *NcpVpcDNSHandler) CreateDNSZone(dnsZoneReqInfo irs.DNSZoneReqInfo) (irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.RegionInfo.Zone, call.DNSZONE, dnsZoneReqInfo.IId.NameId, "CreateDNSZone()")
	start := call.Start()

	if dnsZoneReqInfo.DomainName == "" {
		err := fmt.Errorf("DomainName is required to create a DNS Zone")
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	domainName := strings.TrimSuffix(dnsZoneReqInfo.DomainName, ".")
	domainId, err := dnsHandler.createDomain(domainName)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	info, err := dnsHandler.getDNSZoneInfo(strconv.FormatInt(domainId, 10))
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func (dnsHandler *NcpVpcDNSHandler) ListDNSZone() ([]*irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.RegionInfo.Zone, call.DNSZONE, "DNSZone", "ListDNSZone()")
	start := call.Start()

	domains, err := dnsHandler.listAllDomains()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	infoList := []*irs.DNSZoneInfo{}
	for _, d := range domains {
		info, err := dnsHandler.getDNSZoneInfo(strconv.FormatInt(d.Id, 10))
		if err != nil {
			LoggingError(hiscallInfo, err)
			return nil, err
		}
		infoList = append(infoList, &info)
	}

	LoggingInfo(hiscallInfo, start)
	return infoList, nil
}

func (dnsHandler *NcpVpcDNSHandler) GetDNSZone(dnsZoneIID irs.IID) (irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.RegionInfo.Zone, call.DNSZONE, dnsZoneIID.NameId, "GetDNSZone()")
	start := call.Start()

	info, err := dnsHandler.getDNSZoneInfo(dnsZoneIID.SystemId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	LoggingInfo(hiscallInfo, start)
	return info, nil
}

// DeleteDNSZone removes all non-default records first(NCP's domain-delete API refuses to
// delete a domain that still has records other than the default apex SOA/NS pair), applies
// that record change, and only then deletes the domain itself. Same defensive pattern as
// the AWS Route53 reference implementation.
func (dnsHandler *NcpVpcDNSHandler) DeleteDNSZone(dnsZoneIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.RegionInfo.Zone, call.DNSZONE, dnsZoneIID.NameId, "DeleteDNSZone()")
	start := call.Start()

	records, err := dnsHandler.listAllRecords(dnsZoneIID.SystemId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	ids := []int64{}
	for _, r := range records {
		if isDefaultApexNcpRecord(r) {
			continue
		}
		ids = append(ids, r.Id)
	}

	if len(ids) > 0 {
		if err := dnsHandler.deleteRecords(dnsZoneIID.SystemId, ids); err != nil {
			LoggingError(hiscallInfo, err)
			return false, err
		}
		if err := dnsHandler.applyDomain(dnsZoneIID.SystemId); err != nil {
			LoggingError(hiscallInfo, err)
			return false, err
		}
	}

	if err := dnsHandler.deleteDomain(dnsZoneIID.SystemId); err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	LoggingInfo(hiscallInfo, start)
	return true, nil
}

func (dnsHandler *NcpVpcDNSHandler) AddDNSRecord(dnsZoneIID irs.IID, recordReqInfo irs.DNSRecordReqInfo) (irs.DNSZoneInfo, error) {
	return dnsHandler.changeDNSRecord(dnsZoneIID, recordReqInfo, false, "AddDNSRecord()")
}

func (dnsHandler *NcpVpcDNSHandler) ChangeDNSRecord(dnsZoneIID irs.IID, recordReqInfo irs.DNSRecordReqInfo) (irs.DNSZoneInfo, error) {
	return dnsHandler.changeDNSRecord(dnsZoneIID, recordReqInfo, true, "ChangeDNSRecord()")
}

// changeDNSRecord implements both Add(create-only) and Change(replace existing values for
// this Name+Type, i.e. UPSERT semantics) on top of NCP's per-value record rows: NCP assigns
// one numeric record ID per (host, type, content) row, not one ID per rrset, so "replace"
// is implemented as delete-the-matching-rows-then-recreate rather than an in-place update.
func (dnsHandler *NcpVpcDNSHandler) changeDNSRecord(dnsZoneIID irs.IID, recordReqInfo irs.DNSRecordReqInfo, replaceExisting bool, apiName string) (irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.RegionInfo.Zone, call.DNSZONE, recordReqInfo.Name, apiName)
	start := call.Start()

	if len(recordReqInfo.Value) == 0 {
		err := fmt.Errorf("at least one Value is required for a DNS record")
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	detail, err := dnsHandler.getDomainDetail(dnsZoneIID.SystemId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	host := ncpDNSHostFromFQDN(recordReqInfo.Name, detail.Name)
	recordType := string(recordReqInfo.Type)

	ttl := int64(recordReqInfo.TTL)
	if ttl <= 0 {
		ttl = 300 // CB-Spider default when the caller does not specify one
	}

	if replaceExisting {
		existing, err := dnsHandler.listAllRecords(dnsZoneIID.SystemId)
		if err != nil {
			LoggingError(hiscallInfo, err)
			return irs.DNSZoneInfo{}, err
		}
		staleIds := []int64{}
		for _, r := range existing {
			if strings.EqualFold(r.Host, host) && strings.EqualFold(r.Type, recordType) {
				staleIds = append(staleIds, r.Id)
			}
		}
		if len(staleIds) > 0 {
			if err := dnsHandler.deleteRecords(dnsZoneIID.SystemId, staleIds); err != nil {
				LoggingError(hiscallInfo, err)
				return irs.DNSZoneInfo{}, err
			}
		}
	}

	items := []ncpDNSRecordCreateItem{}
	for _, v := range recordReqInfo.Value {
		items = append(items, ncpDNSRecordCreateItem{
			Host:    host,
			Type:    recordType,
			Content: v,
			Ttl:     ttl,
		})
	}
	if err := dnsHandler.createRecords(dnsZoneIID.SystemId, items); err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	if err := dnsHandler.applyDomain(dnsZoneIID.SystemId); err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	info, err := dnsHandler.getDNSZoneInfo(dnsZoneIID.SystemId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func (dnsHandler *NcpVpcDNSHandler) RemoveDNSRecord(dnsZoneIID irs.IID, recordIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.RegionInfo.Zone, call.DNSZONE, recordIID.NameId, "RemoveDNSRecord()")
	start := call.Start()

	name, recordType, err := parseNcpRecordSystemId(recordIID.SystemId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	detail, err := dnsHandler.getDomainDetail(dnsZoneIID.SystemId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}
	host := ncpDNSHostFromFQDN(name, detail.Name)

	records, err := dnsHandler.listAllRecords(dnsZoneIID.SystemId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	ids := []int64{}
	for _, r := range records {
		if strings.EqualFold(r.Host, host) && strings.EqualFold(r.Type, recordType) {
			ids = append(ids, r.Id)
		}
	}
	if len(ids) == 0 {
		err := fmt.Errorf("DNS record '%s'(%s) does not exist", name, recordType)
		LoggingError(hiscallInfo, err)
		return false, err
	}

	if err := dnsHandler.deleteRecords(dnsZoneIID.SystemId, ids); err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	if err := dnsHandler.applyDomain(dnsZoneIID.SystemId); err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	LoggingInfo(hiscallInfo, start)
	return true, nil
}

//------------------------- internal helpers -------------------------

// getDNSZoneInfo fetches the domain detail(for NameServers) and the full record list(for
// RecordInfoList) and assembles the common-interface DNSZoneInfo.
func (dnsHandler *NcpVpcDNSHandler) getDNSZoneInfo(domainId string) (irs.DNSZoneInfo, error) {
	detail, err := dnsHandler.getDomainDetail(domainId)
	if err != nil {
		return irs.DNSZoneInfo{}, err
	}

	records, err := dnsHandler.listAllRecords(domainId)
	if err != nil {
		return irs.DNSZoneInfo{}, err
	}

	nameServers := []string{}
	for _, ns := range strings.Fields(detail.Soa.NameServers) {
		nameServers = append(nameServers, strings.TrimSuffix(ns, "."))
	}

	return irs.DNSZoneInfo{
		IId:            irs.IID{NameId: detail.Name, SystemId: strconv.FormatInt(detail.Id, 10)},
		DomainName:     detail.Name,
		NameServers:    nameServers,
		RecordInfoList: ncpDNSBuildRecordInfoList(detail.Name, records),
	}, nil
}

// ncpDNSBuildRecordInfoList groups NCP's per-value record rows(one numeric ID per
// host+type+content) into the common interface's per-(Name,Type) DNSRecordInfo entries
// (Value is a list), the same grouping granularity the AWS Route53 reference uses.
func ncpDNSBuildRecordInfoList(domainName string, records []ncpDNSRecord) []irs.DNSRecordInfo {
	order := []string{}
	groups := map[string]*irs.DNSRecordInfo{}

	for _, r := range records {
		key := strings.ToUpper(r.Host) + "|" + strings.ToUpper(r.Type)
		g, ok := groups[key]
		if !ok {
			fqdn := ncpDNSFQDNFromHost(r.Host, domainName)
			newInfo := irs.DNSRecordInfo{
				IId:   irs.IID{NameId: fqdn, SystemId: makeNcpRecordSystemId(fqdn, r.Type)},
				Type:  irs.DNSRecordType(strings.ToUpper(r.Type)),
				Name:  fqdn,
				Value: []string{},
				TTL:   int(r.Ttl),
			}
			groups[key] = &newInfo
			order = append(order, key)
			g = &newInfo
		}
		g.Value = append(g.Value, r.Content)
	}

	result := []irs.DNSRecordInfo{}
	for _, k := range order {
		result = append(result, *groups[k])
	}
	return result
}

// isDefaultApexNcpRecord reports whether the record row is part of the SOA/NS pair NCP
// auto-creates at the zone apex(host "@") when the domain is created. Those are not
// deletable as ordinary records and must stay behind for DeleteDNSZone's cleanup pass.
func isDefaultApexNcpRecord(r ncpDNSRecord) bool {
	recordType := strings.ToUpper(r.Type)
	if recordType == "SOA" {
		return true
	}
	if recordType == "NS" && (r.Host == "" || r.Host == "@") {
		return true
	}
	return false
}

// ncpDNSHostFromFQDN converts a user-facing FQDN(ex: "www.example.com") to the relative
// "host" field NCP's record API expects(ex: "www"), or "@" for the zone apex.
func ncpDNSHostFromFQDN(name string, domainName string) string {
	trimmedName := strings.TrimSuffix(strings.TrimSpace(name), ".")
	trimmedDomain := strings.TrimSuffix(strings.TrimSpace(domainName), ".")

	if strings.EqualFold(trimmedName, trimmedDomain) {
		return "@"
	}

	suffix := "." + trimmedDomain
	if len(trimmedName) > len(suffix) && strings.EqualFold(trimmedName[len(trimmedName)-len(suffix):], suffix) {
		return trimmedName[:len(trimmedName)-len(suffix)]
	}

	// Fallback: name did not carry the zone's domain suffix(unexpected); treat as already relative.
	return trimmedName
}

// ncpDNSFQDNFromHost is the inverse of ncpDNSHostFromFQDN.
func ncpDNSFQDNFromHost(host string, domainName string) string {
	trimmedDomain := strings.TrimSuffix(domainName, ".")
	if host == "" || host == "@" {
		return trimmedDomain
	}
	return host + "." + trimmedDomain
}

func makeNcpRecordSystemId(fqdn string, recordType string) string {
	return fqdn + ":" + strings.ToUpper(recordType)
}

func parseNcpRecordSystemId(systemId string) (name string, recordType string, err error) {
	idx := strings.LastIndex(systemId, ":")
	if idx < 0 {
		return "", "", fmt.Errorf("invalid DNS record SystemId: %s", systemId)
	}
	return systemId[:idx], systemId[idx+1:], nil
}

//------------------------- NCP Global DNS REST client -------------------------

func (dnsHandler *NcpVpcDNSHandler) listAllDomains() ([]ncpDNSDomain, error) {
	all := []ncpDNSDomain{}
	for page := 0; ; page++ {
		query := url.Values{}
		query.Set("page", strconv.Itoa(page))
		query.Set("size", strconv.Itoa(ncpDNSPageSize))

		respBytes, err := dnsHandler.callAPI(http.MethodGet, "/dns/v1/ncpdns/domain", query, nil)
		if err != nil {
			return nil, err
		}
		var parsed ncpDNSDomainListResponse
		if err := json.Unmarshal(respBytes, &parsed); err != nil {
			return nil, fmt.Errorf("failed to parse Global DNS domain list response: %w", err)
		}
		all = append(all, parsed.Content...)
		if parsed.Last || len(parsed.Content) == 0 {
			break
		}
	}
	return all, nil
}

func (dnsHandler *NcpVpcDNSHandler) createDomain(name string) (int64, error) {
	reqBody := ncpDNSDomainCreateRequest{Name: name}
	respBytes, err := dnsHandler.callAPI(http.MethodPost, "/dns/v1/ncpdns/domain", nil, reqBody)
	if err != nil {
		return 0, err
	}

	// The official docs do not fully detail CreateDomain's success response schema;
	// try to parse it as a DomainResponse first, and fall back to looking the domain
	// up by name if that fails or the id comes back empty.
	var created ncpDNSDomain
	if jsonErr := json.Unmarshal(respBytes, &created); jsonErr == nil && created.Id != 0 {
		return created.Id, nil
	}

	domains, listErr := dnsHandler.listAllDomains()
	if listErr != nil {
		return 0, fmt.Errorf("domain '%s' was created but its ID could not be resolved: %w", name, listErr)
	}
	for _, d := range domains {
		if strings.EqualFold(d.Name, name) {
			return d.Id, nil
		}
	}
	return 0, fmt.Errorf("domain '%s' was created but its ID could not be resolved from the domain list", name)
}

func (dnsHandler *NcpVpcDNSHandler) getDomainDetail(domainId string) (ncpDNSDomainDetail, error) {
	respBytes, err := dnsHandler.callAPI(http.MethodGet, "/dns/v1/ncpdns/domain/"+domainId, nil, nil)
	if err != nil {
		return ncpDNSDomainDetail{}, err
	}
	var detail ncpDNSDomainDetail
	if err := json.Unmarshal(respBytes, &detail); err != nil {
		return ncpDNSDomainDetail{}, fmt.Errorf("failed to parse Global DNS domain detail response: %w", err)
	}
	return detail, nil
}

func (dnsHandler *NcpVpcDNSHandler) deleteDomain(domainId string) error {
	_, err := dnsHandler.callAPI(http.MethodDelete, "/dns/v1/ncpdns/domain/"+domainId, nil, nil)
	return err
}

func (dnsHandler *NcpVpcDNSHandler) listAllRecords(domainId string) ([]ncpDNSRecord, error) {
	all := []ncpDNSRecord{}
	for page := 0; ; page++ {
		query := url.Values{}
		query.Set("page", strconv.Itoa(page))
		query.Set("size", strconv.Itoa(ncpDNSPageSize))

		respBytes, err := dnsHandler.callAPI(http.MethodGet, "/dns/v1/ncpdns/record/"+domainId, query, nil)
		if err != nil {
			return nil, err
		}
		var parsed ncpDNSRecordListResponse
		if err := json.Unmarshal(respBytes, &parsed); err != nil {
			return nil, fmt.Errorf("failed to parse Global DNS record list response: %w", err)
		}
		all = append(all, parsed.Content...)
		if parsed.Last || len(parsed.Content) == 0 {
			break
		}
	}
	return all, nil
}

// createRecords calls POST /dns/v1/ncpdns/record/{domainId}, whose body is a JSON array of
// record items(bulk-create), per the official API example.
func (dnsHandler *NcpVpcDNSHandler) createRecords(domainId string, items []ncpDNSRecordCreateItem) error {
	_, err := dnsHandler.callAPI(http.MethodPost, "/dns/v1/ncpdns/record/"+domainId, nil, items)
	return err
}

// deleteRecords calls DELETE /dns/v1/ncpdns/record/{domainId}, whose body is a plain JSON
// array of numeric record IDs to remove, per the official API example(ex: "[26123]").
func (dnsHandler *NcpVpcDNSHandler) deleteRecords(domainId string, ids []int64) error {
	_, err := dnsHandler.callAPI(http.MethodDelete, "/dns/v1/ncpdns/record/"+domainId, nil, ids)
	return err
}

// applyDomain calls PUT /dns/v1/ncpdns/record/apply/{domainId}, which publishes pending
// record changes to the live name servers. Without this call, record create/update/delete
// operations are accepted by the API but never take effect on DNS resolution.
func (dnsHandler *NcpVpcDNSHandler) applyDomain(domainId string) error {
	_, err := dnsHandler.callAPI(http.MethodPut, "/dns/v1/ncpdns/record/apply/"+domainId, nil, nil)
	return err
}

// callAPI performs one signed HTTP call against the Global DNS REST API and returns the
// raw response body on a 2xx response.
func (dnsHandler *NcpVpcDNSHandler) callAPI(method string, path string, query url.Values, body interface{}) ([]byte, error) {
	reqURI := path
	if len(query) > 0 {
		reqURI = path + "?" + query.Encode()
	}

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal Global DNS request body: %w", err)
		}
	}

	accessKey := dnsHandler.CredentialInfo.ClientId
	secretKey := dnsHandler.CredentialInfo.ClientSecret
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	signature := ncpDNSSign(method, reqURI, timestamp, accessKey, secretKey)

	httpReq, err := http.NewRequest(method, ncpDNSBaseURL+reqURI, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to build Global DNS request: %w", err)
	}
	httpReq.Header.Set("x-ncp-apigw-timestamp", timestamp)
	httpReq.Header.Set("x-ncp-iam-access-key", accessKey)
	httpReq.Header.Set("x-ncp-apigw-signature-v2", signature)
	if len(bodyBytes) > 0 {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Global DNS API call failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Global DNS API response: %w", err)
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return nil, parseNcpDNSError(httpResp.StatusCode, respBytes)
	}
	return respBytes, nil
}

// ncpDNSSign implements NCP's API Gateway v2 signing scheme, shared identically across all
// of NCP's REST APIs: base64(HMAC-SHA256("METHOD URI\nTIMESTAMP\nACCESS_KEY", SECRET_KEY)).
func ncpDNSSign(method string, uri string, timestamp string, accessKey string, secretKey string) string {
	message := strings.ToUpper(method) + " " + uri + "\n" + timestamp + "\n" + accessKey
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func parseNcpDNSError(statusCode int, body []byte) error {
	var envelope ncpDNSErrorEnvelope
	if jsonErr := json.Unmarshal(body, &envelope); jsonErr == nil && envelope.Error.ErrorCode != "" {
		return fmt.Errorf("NCP Global DNS API error(HTTP %d): [%s] %s", statusCode, envelope.Error.ErrorCode, envelope.Error.Message)
	}
	return fmt.Errorf("NCP Global DNS API error(HTTP %d): %s", statusCode, string(body))
}
