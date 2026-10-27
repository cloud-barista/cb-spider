// Azure DNS DNSHandler of CB-Spider.
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
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

// dnsZoneLocation is the fixed ARM "location" Azure requires on a public DNS zone
// resource even though the zone itself is a global(non-regional) resource.
const dnsZoneLocation = "global"

type AzureDNSHandler struct {
	Region          idrv.RegionInfo
	Ctx             context.Context
	Client          *armdns.ZonesClient
	RecordSetClient *armdns.RecordSetsClient
}

//------------------------- interface methods -------------------------

func (dnsHandler *AzureDNSHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, "DNSZone", "ListIID()")
	start := call.Start()

	zones, err := dnsHandler.listAllZones()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	iidList := []*irs.IID{}
	for _, zone := range zones {
		iidList = append(iidList, &irs.IID{NameId: strVal(zone.Name), SystemId: strVal(zone.ID)})
	}

	LoggingInfo(hiscallInfo, start)
	return iidList, nil
}

func (dnsHandler *AzureDNSHandler) CreateDNSZone(dnsZoneReqInfo irs.DNSZoneReqInfo) (irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, dnsZoneReqInfo.IId.NameId, "CreateDNSZone()")
	start := call.Start()

	if dnsZoneReqInfo.DomainName == "" {
		err := fmt.Errorf("DomainName is required to create a DNS Zone")
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}
	// Azure DNS zone names are not stored with a trailing dot.
	zoneName := strings.TrimSuffix(strings.TrimSpace(dnsZoneReqInfo.DomainName), ".")

	resp, err := dnsHandler.Client.CreateOrUpdate(dnsHandler.Ctx, dnsHandler.Region.Region, zoneName, armdns.Zone{
		Location: toStrPtr(dnsZoneLocation),
		Tags:     setTags(dnsZoneReqInfo.TagList),
	}, nil)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	info, err := dnsHandler.getDNSZoneInfo(&resp.Zone)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func (dnsHandler *AzureDNSHandler) ListDNSZone() ([]*irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, "DNSZone", "ListDNSZone()")
	start := call.Start()

	zones, err := dnsHandler.listAllZones()
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

func (dnsHandler *AzureDNSHandler) GetDNSZone(dnsZoneIID irs.IID) (irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, dnsZoneIID.NameId, "GetDNSZone()")
	start := call.Start()

	zoneName, err := zoneNameFromIID(dnsZoneIID)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	resp, err := dnsHandler.Client.Get(dnsHandler.Ctx, dnsHandler.Region.Region, zoneName, nil)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	info, err := dnsHandler.getDNSZoneInfo(&resp.Zone)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	LoggingInfo(hiscallInfo, start)
	return info, nil
}

// DeleteDNSZone deletes the DNS Zone.
//
// Unlike AWS Route53 (which refuses to delete a hosted zone that still has
// records other than the default apex NS/SOA pair), Azure DNS's zone delete
// cascades: "All DNS records in the zone will also be deleted" (see
// armdns.ZonesClient.BeginDelete doc). So no pre-cleanup of record sets is
// required here.
func (dnsHandler *AzureDNSHandler) DeleteDNSZone(dnsZoneIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, dnsZoneIID.NameId, "DeleteDNSZone()")
	start := call.Start()

	zoneName, err := zoneNameFromIID(dnsZoneIID)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	poller, err := dnsHandler.Client.BeginDelete(dnsHandler.Ctx, dnsHandler.Region.Region, zoneName, nil)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}
	if _, err := poller.PollUntilDone(dnsHandler.Ctx, nil); err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	LoggingInfo(hiscallInfo, start)
	return true, nil
}

func (dnsHandler *AzureDNSHandler) AddDNSRecord(dnsZoneIID irs.IID, recordReqInfo irs.DNSRecordReqInfo) (irs.DNSZoneInfo, error) {
	return dnsHandler.changeDNSRecord(dnsZoneIID, recordReqInfo, "AddDNSRecord()")
}

func (dnsHandler *AzureDNSHandler) ChangeDNSRecord(dnsZoneIID irs.IID, recordReqInfo irs.DNSRecordReqInfo) (irs.DNSZoneInfo, error) {
	return dnsHandler.changeDNSRecord(dnsZoneIID, recordReqInfo, "ChangeDNSRecord()")
}

// changeDNSRecord implements both AddDNSRecord and ChangeDNSRecord: Azure's RecordSets.CreateOrUpdate
// is already an upsert keyed by (zone, relative name, type), so both CB-Spider operations map to the
// same CSP call.
func (dnsHandler *AzureDNSHandler) changeDNSRecord(dnsZoneIID irs.IID, recordReqInfo irs.DNSRecordReqInfo, apiName string) (irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, recordReqInfo.Name, apiName)
	start := call.Start()

	zoneName, err := zoneNameFromIID(dnsZoneIID)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	if len(recordReqInfo.Value) == 0 {
		err := fmt.Errorf("at least one Value is required for a DNS record")
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	relativeName, err := relativeRecordName(recordReqInfo.Name, zoneName)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	ttl := int64(recordReqInfo.TTL)
	if ttl <= 0 {
		ttl = 300 // CB-Spider default when the caller does not specify one
	}

	props, err := buildRecordSetProperties(recordReqInfo.Type, recordReqInfo.Value, ttl)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	_, err = dnsHandler.RecordSetClient.CreateOrUpdate(dnsHandler.Ctx, dnsHandler.Region.Region, zoneName, relativeName,
		armdns.RecordType(string(recordReqInfo.Type)), armdns.RecordSet{Properties: props}, nil)
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

func (dnsHandler *AzureDNSHandler) RemoveDNSRecord(dnsZoneIID irs.IID, recordIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, recordIID.NameId, "RemoveDNSRecord()")
	start := call.Start()

	zoneName, err := zoneNameFromIID(dnsZoneIID)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	name, recordType, err := parseRecordSystemId(recordIID.SystemId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	relativeName, err := relativeRecordName(name, zoneName)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	_, err = dnsHandler.RecordSetClient.Delete(dnsHandler.Ctx, dnsHandler.Region.Region, zoneName, relativeName, armdns.RecordType(recordType), nil)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	LoggingInfo(hiscallInfo, start)
	return true, nil
}

//------------------------- internal helpers -------------------------

func (dnsHandler *AzureDNSHandler) listAllZones() ([]*armdns.Zone, error) {
	zones := []*armdns.Zone{}
	pager := dnsHandler.Client.NewListByResourceGroupPager(dnsHandler.Region.Region, nil)
	for pager.More() {
		page, err := pager.NextPage(dnsHandler.Ctx)
		if err != nil {
			return nil, err
		}
		zones = append(zones, page.Value...)
	}
	return zones, nil
}

func (dnsHandler *AzureDNSHandler) listAllRecordSets(zoneName string) ([]*armdns.RecordSet, error) {
	rrSets := []*armdns.RecordSet{}
	pager := dnsHandler.RecordSetClient.NewListAllByDNSZonePager(dnsHandler.Region.Region, zoneName, nil)
	for pager.More() {
		page, err := pager.NextPage(dnsHandler.Ctx)
		if err != nil {
			return nil, err
		}
		rrSets = append(rrSets, page.Value...)
	}
	return rrSets, nil
}

func (dnsHandler *AzureDNSHandler) getDNSZoneInfo(zone *armdns.Zone) (irs.DNSZoneInfo, error) {
	zoneName := strVal(zone.Name)
	if zoneName == "" {
		zoneName = zoneNameFromResourceId(strVal(zone.ID))
	}

	rrSets, err := dnsHandler.listAllRecordSets(zoneName)
	if err != nil {
		return irs.DNSZoneInfo{}, err
	}

	nameServers := []string{}
	var tagList []irs.KeyValue
	if zone.Properties != nil {
		for _, ns := range zone.Properties.NameServers {
			nameServers = append(nameServers, strVal(ns))
		}
	}
	if zone.Tags != nil {
		tagList = setTagList(zone.Tags)
	}

	recordInfoList := []irs.DNSRecordInfo{}
	for _, rrSet := range rrSets {
		info, ok := convertRecordSetToInfo(rrSet, zoneName)
		if !ok {
			// Record set type this driver does not map to the common DNSRecordType
			// set(ex: CAA, PTR) - skip it rather than failing the whole zone read.
			continue
		}
		recordInfoList = append(recordInfoList, info)
	}

	return irs.DNSZoneInfo{
		IId:            irs.IID{NameId: zoneName, SystemId: strVal(zone.ID)},
		DomainName:     zoneName,
		NameServers:    nameServers,
		RecordInfoList: recordInfoList,
		TagList:        tagList,
	}, nil
}

// zoneNameFromIID resolves the Azure DNS zone name(ex: "example.com") used by the
// armdns SDK calls from the IID cb-spider's common-runtime layer hands to the driver.
//
// That IID's SystemId is whatever this driver returned from CreateDNSZone/ListIID -
// the zone's ARM resource ID(.../dnsZones/{zoneName}) - so the zone name is simply its
// last path segment. NameId is used only as a fallback for callers that pass the zone
// name directly in NameId with no SystemId set.
func zoneNameFromIID(dnsZoneIID irs.IID) (string, error) {
	if dnsZoneIID.SystemId != "" {
		return zoneNameFromResourceId(dnsZoneIID.SystemId), nil
	}
	if dnsZoneIID.NameId != "" {
		return dnsZoneIID.NameId, nil
	}
	return "", fmt.Errorf("DNS Zone IID must have either NameId or SystemId set")
}

func zoneNameFromResourceId(resourceId string) string {
	resourceId = strings.TrimSuffix(strings.TrimSpace(resourceId), "/")
	idx := strings.LastIndex(resourceId, "/")
	if idx < 0 {
		return resourceId
	}
	return resourceId[idx+1:]
}

// relativeRecordName converts a user-facing FQDN record name(ex: "www.example.com")
// into the name relative to the zone that armdns.RecordSetsClient expects(ex: "www"),
// using "@" for the zone apex itself, per the armdns.RecordSetsClient.CreateOrUpdate contract.
func relativeRecordName(fqdnName string, zoneName string) (string, error) {
	name := strings.TrimSuffix(strings.TrimSpace(fqdnName), ".")
	zone := strings.TrimSuffix(strings.TrimSpace(zoneName), ".")

	if strings.EqualFold(name, zone) {
		return "@", nil
	}

	suffix := "." + zone
	if len(name) > len(suffix) && strings.EqualFold(name[len(name)-len(suffix):], suffix) {
		return name[:len(name)-len(suffix)], nil
	}

	return "", fmt.Errorf("record name '%s' is not part of DNS zone '%s'", fqdnName, zoneName)
}

// fqdnFromRelativeName is the inverse of relativeRecordName: it reconstructs the
// user-facing FQDN from the relative record set name("@" at the zone apex) Azure returns.
func fqdnFromRelativeName(relativeName string, zoneName string) string {
	zone := strings.TrimSuffix(strings.TrimSpace(zoneName), ".")
	if relativeName == "" || relativeName == "@" {
		return zone
	}
	return relativeName + "." + zone
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

// recordTypeFromRecordSet extracts the short record type(ex: "A") from the record
// set's Type field, which Azure returns as a full resource type string
// (ex: "Microsoft.Network/dnsZones/A").
func recordTypeFromRecordSet(rrSet *armdns.RecordSet) string {
	t := strVal(rrSet.Type)
	if idx := strings.LastIndex(t, "/"); idx >= 0 {
		return t[idx+1:]
	}
	return t
}

// convertRecordSetToInfo maps an Azure RecordSet to the common DNSRecordInfo shape.
// ok is false when the record set's type is not one this driver maps to a Value list
// (ex: CAA, PTR) - the caller should skip such record sets rather than fail outright.
func convertRecordSetToInfo(rrSet *armdns.RecordSet, zoneName string) (irs.DNSRecordInfo, bool) {
	recordType := recordTypeFromRecordSet(rrSet)
	name := fqdnFromRelativeName(strVal(rrSet.Name), zoneName)

	var ttl int64
	props := rrSet.Properties
	if props != nil && props.TTL != nil {
		ttl = *props.TTL
	}

	values, ok := valuesFromRecordSetProperties(irs.DNSRecordType(recordType), props)
	if !ok {
		return irs.DNSRecordInfo{}, false
	}

	return irs.DNSRecordInfo{
		IId:   irs.IID{NameId: name, SystemId: makeRecordSystemId(name, recordType)},
		Type:  irs.DNSRecordType(recordType),
		Name:  name,
		Value: values,
		TTL:   int(ttl),
	}, true
}

// valuesFromRecordSetProperties extracts the record values(in CB-Spider's flat []string
// shape) from Azure's strongly-typed per-type RecordSetProperties fields.
func valuesFromRecordSetProperties(recordType irs.DNSRecordType, props *armdns.RecordSetProperties) ([]string, bool) {
	if props == nil {
		return []string{}, true
	}

	values := []string{}
	switch recordType {
	case irs.DNSRecordTypeA:
		for _, r := range props.ARecords {
			values = append(values, strVal(r.IPv4Address))
		}
	case irs.DNSRecordTypeAAAA:
		for _, r := range props.AaaaRecords {
			values = append(values, strVal(r.IPv6Address))
		}
	case irs.DNSRecordTypeCNAME:
		if props.CnameRecord != nil {
			values = append(values, strVal(props.CnameRecord.Cname))
		}
	case irs.DNSRecordTypeTXT:
		for _, r := range props.TxtRecords {
			chunks := make([]string, 0, len(r.Value))
			for _, v := range r.Value {
				chunks = append(chunks, strVal(v))
			}
			values = append(values, strings.Join(chunks, ""))
		}
	case irs.DNSRecordTypeNS:
		for _, r := range props.NsRecords {
			values = append(values, strVal(r.Nsdname))
		}
	case irs.DNSRecordTypeMX:
		for _, r := range props.MxRecords {
			values = append(values, formatMXValue(int32Val(r.Preference), strVal(r.Exchange)))
		}
	case irs.DNSRecordTypeSRV:
		for _, r := range props.SrvRecords {
			values = append(values, formatSRVValue(int32Val(r.Priority), int32Val(r.Weight), int32Val(r.Port), strVal(r.Target)))
		}
	case "SOA":
		if props.SoaRecord != nil {
			values = append(values, formatSOAValue(props.SoaRecord))
		}
	default:
		// CAA, PTR, and any other type this driver does not map.
		return nil, false
	}
	return values, true
}

// buildRecordSetProperties maps CB-Spider's flat []string Value list to Azure's
// strongly-typed per-type RecordSetProperties fields. Supported types: A, AAAA,
// CNAME, TXT, NS, MX, SRV. CAA is not supported(its value packs flag/tag/quoted-value
// into a single string in a CSP-specific way that does not map cleanly onto a plain
// []string) and returns an error.
func buildRecordSetProperties(recordType irs.DNSRecordType, values []string, ttl int64) (*armdns.RecordSetProperties, error) {
	props := &armdns.RecordSetProperties{TTL: &ttl}

	switch recordType {
	case irs.DNSRecordTypeA:
		recs := make([]*armdns.ARecord, 0, len(values))
		for _, v := range values {
			recs = append(recs, &armdns.ARecord{IPv4Address: toStrPtr(v)})
		}
		props.ARecords = recs
	case irs.DNSRecordTypeAAAA:
		recs := make([]*armdns.AaaaRecord, 0, len(values))
		for _, v := range values {
			recs = append(recs, &armdns.AaaaRecord{IPv6Address: toStrPtr(v)})
		}
		props.AaaaRecords = recs
	case irs.DNSRecordTypeCNAME:
		if len(values) != 1 {
			return nil, fmt.Errorf("a CNAME record requires exactly one Value, got %d", len(values))
		}
		props.CnameRecord = &armdns.CnameRecord{Cname: toStrPtr(values[0])}
	case irs.DNSRecordTypeTXT:
		recs := make([]*armdns.TxtRecord, 0, len(values))
		for _, v := range values {
			recs = append(recs, &armdns.TxtRecord{Value: []*string{toStrPtr(v)}})
		}
		props.TxtRecords = recs
	case irs.DNSRecordTypeNS:
		recs := make([]*armdns.NsRecord, 0, len(values))
		for _, v := range values {
			recs = append(recs, &armdns.NsRecord{Nsdname: toStrPtr(v)})
		}
		props.NsRecords = recs
	case irs.DNSRecordTypeMX:
		recs := make([]*armdns.MxRecord, 0, len(values))
		for _, v := range values {
			preference, exchange, err := parseMXValue(v)
			if err != nil {
				return nil, err
			}
			recs = append(recs, &armdns.MxRecord{Preference: &preference, Exchange: toStrPtr(exchange)})
		}
		props.MxRecords = recs
	case irs.DNSRecordTypeSRV:
		recs := make([]*armdns.SrvRecord, 0, len(values))
		for _, v := range values {
			priority, weight, port, target, err := parseSRVValue(v)
			if err != nil {
				return nil, err
			}
			recs = append(recs, &armdns.SrvRecord{Priority: &priority, Weight: &weight, Port: &port, Target: toStrPtr(target)})
		}
		props.SrvRecords = recs
	default:
		return nil, fmt.Errorf("DNS record type '%s' is not supported by the Azure driver (supported: A, AAAA, CNAME, TXT, NS, MX, SRV)", recordType)
	}

	return props, nil
}

// formatMXValue/parseMXValue and formatSRVValue/parseSRVValue use the familiar
// zone-file literal form(ex: "10 mail.example.com") - the same convention CB-Spider's
// AWS driver relies on, since Route53 itself stores MX/SRV values that way.

func formatMXValue(preference int32, exchange string) string {
	return fmt.Sprintf("%d %s", preference, exchange)
}

func parseMXValue(value string) (preference int32, exchange string, err error) {
	fields := strings.Fields(value)
	if len(fields) != 2 {
		return 0, "", fmt.Errorf("invalid MX record value '%s': expected '<preference> <exchange>'", value)
	}
	n, err := strconv.ParseInt(fields[0], 10, 32)
	if err != nil {
		return 0, "", fmt.Errorf("invalid MX record value '%s': preference must be an integer: %w", value, err)
	}
	return int32(n), fields[1], nil
}

func formatSRVValue(priority int32, weight int32, port int32, target string) string {
	return fmt.Sprintf("%d %d %d %s", priority, weight, port, target)
}

func parseSRVValue(value string) (priority int32, weight int32, port int32, target string, err error) {
	fields := strings.Fields(value)
	if len(fields) != 4 {
		return 0, 0, 0, "", fmt.Errorf("invalid SRV record value '%s': expected '<priority> <weight> <port> <target>'", value)
	}
	ints := make([]int32, 3)
	for i := 0; i < 3; i++ {
		n, err := strconv.ParseInt(fields[i], 10, 32)
		if err != nil {
			return 0, 0, 0, "", fmt.Errorf("invalid SRV record value '%s': %w", value, err)
		}
		ints[i] = int32(n)
	}
	return ints[0], ints[1], ints[2], fields[3], nil
}

func formatSOAValue(soa *armdns.SoaRecord) string {
	return fmt.Sprintf("%s %s %d %d %d %d %d",
		strVal(soa.Host), strVal(soa.Email), int64Val(soa.SerialNumber),
		int64Val(soa.RefreshTime), int64Val(soa.RetryTime), int64Val(soa.ExpireTime), int64Val(soa.MinimumTTL))
}

func strVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func int32Val(n *int32) int32 {
	if n == nil {
		return 0
	}
	return *n
}

func int64Val(n *int64) int64 {
	if n == nil {
		return 0
	}
	return *n
}
