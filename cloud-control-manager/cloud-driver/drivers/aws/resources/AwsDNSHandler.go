// AWS Route53 DNSHandler of CB-Spider.
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
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AwsDNSHandler struct {
	Region idrv.RegionInfo
	Client *route53.Route53
}

// fqdn normalizes a DNS name to the trailing-dot form Route53 uses internally,
// so names coming from the user (with or without a trailing dot) compare equal
// to names coming back from the Route53 API.
func fqdn(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || strings.HasSuffix(name, ".") {
		return name
	}
	return name + "."
}

func (dnsHandler *AwsDNSHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, "DNSZone", "ListIID()")
	start := call.Start()

	zones, err := dnsHandler.listAllHostedZones()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	iidList := []*irs.IID{}
	for _, zone := range zones {
		iidList = append(iidList, &irs.IID{NameId: *zone.Name, SystemId: *zone.Id})
	}

	LoggingInfo(hiscallInfo, start)
	return iidList, nil
}

func (dnsHandler *AwsDNSHandler) CreateDNSZone(dnsZoneReqInfo irs.DNSZoneReqInfo) (irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, dnsZoneReqInfo.IId.NameId, "CreateDNSZone()")
	start := call.Start()

	if dnsZoneReqInfo.DomainName == "" {
		err := fmt.Errorf("DomainName is required to create a DNS Zone")
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	callerRef := fmt.Sprintf("cb-spider-%s-%d", dnsZoneReqInfo.IId.NameId, time.Now().UnixNano())
	req := &route53.CreateHostedZoneInput{
		Name:            aws.String(dnsZoneReqInfo.DomainName),
		CallerReference: aws.String(callerRef),
		HostedZoneConfig: &route53.HostedZoneConfig{
			Comment: aws.String("Created by CB-Spider"),
		},
	}

	result, err := dnsHandler.Client.CreateHostedZone(req)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	info, err := dnsHandler.getDNSZoneInfo(result.HostedZone, result.DelegationSet)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func (dnsHandler *AwsDNSHandler) ListDNSZone() ([]*irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, "DNSZone", "ListDNSZone()")
	start := call.Start()

	zones, err := dnsHandler.listAllHostedZones()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	infoList := []*irs.DNSZoneInfo{}
	for _, zone := range zones {
		info, err := dnsHandler.getDNSZoneInfo(zone, nil)
		if err != nil {
			LoggingError(hiscallInfo, err)
			return nil, err
		}
		infoList = append(infoList, &info)
	}

	LoggingInfo(hiscallInfo, start)
	return infoList, nil
}

func (dnsHandler *AwsDNSHandler) GetDNSZone(dnsZoneIID irs.IID) (irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, dnsZoneIID.NameId, "GetDNSZone()")
	start := call.Start()

	result, err := dnsHandler.Client.GetHostedZone(&route53.GetHostedZoneInput{
		Id: aws.String(dnsZoneIID.SystemId),
	})
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	info, err := dnsHandler.getDNSZoneInfo(result.HostedZone, result.DelegationSet)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.DNSZoneInfo{}, err
	}

	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func (dnsHandler *AwsDNSHandler) DeleteDNSZone(dnsZoneIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, dnsZoneIID.NameId, "DeleteDNSZone()")
	start := call.Start()

	zoneResult, err := dnsHandler.Client.GetHostedZone(&route53.GetHostedZoneInput{
		Id: aws.String(dnsZoneIID.SystemId),
	})
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}
	zoneName := aws.StringValue(zoneResult.HostedZone.Name)

	// Route53 refuses to delete a zone that still has records other than the
	// default apex NS/SOA pair, so those must be removed first.
	rrSets, err := dnsHandler.listAllResourceRecordSets(dnsZoneIID.SystemId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	changes := []*route53.Change{}
	for _, rrSet := range rrSets {
		if isDefaultApexRecord(rrSet, zoneName) {
			continue
		}
		changes = append(changes, &route53.Change{
			Action:            aws.String(route53.ChangeActionDelete),
			ResourceRecordSet: rrSet,
		})
	}
	if len(changes) > 0 {
		_, err = dnsHandler.Client.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
			HostedZoneId: aws.String(dnsZoneIID.SystemId),
			ChangeBatch:  &route53.ChangeBatch{Changes: changes},
		})
		if err != nil {
			LoggingError(hiscallInfo, err)
			return false, err
		}
	}

	_, err = dnsHandler.Client.DeleteHostedZone(&route53.DeleteHostedZoneInput{
		Id: aws.String(dnsZoneIID.SystemId),
	})
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	LoggingInfo(hiscallInfo, start)
	return true, nil
}

func (dnsHandler *AwsDNSHandler) AddDNSRecord(dnsZoneIID irs.IID, recordReqInfo irs.DNSRecordReqInfo) (irs.DNSZoneInfo, error) {
	return dnsHandler.changeDNSRecord(dnsZoneIID, recordReqInfo, route53.ChangeActionCreate, "AddDNSRecord()")
}

func (dnsHandler *AwsDNSHandler) ChangeDNSRecord(dnsZoneIID irs.IID, recordReqInfo irs.DNSRecordReqInfo) (irs.DNSZoneInfo, error) {
	return dnsHandler.changeDNSRecord(dnsZoneIID, recordReqInfo, route53.ChangeActionUpsert, "ChangeDNSRecord()")
}

func (dnsHandler *AwsDNSHandler) changeDNSRecord(dnsZoneIID irs.IID, recordReqInfo irs.DNSRecordReqInfo, action string, apiName string) (irs.DNSZoneInfo, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, recordReqInfo.Name, apiName)
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

	resourceRecords := []*route53.ResourceRecord{}
	for _, v := range recordReqInfo.Value {
		resourceRecords = append(resourceRecords, &route53.ResourceRecord{Value: aws.String(v)})
	}

	_, err := dnsHandler.Client.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(dnsZoneIID.SystemId),
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action: aws.String(action),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name:            aws.String(recordReqInfo.Name),
						Type:            aws.String(string(recordReqInfo.Type)),
						TTL:             aws.Int64(ttl),
						ResourceRecords: resourceRecords,
					},
				},
			},
		},
	})
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

func (dnsHandler *AwsDNSHandler) RemoveDNSRecord(dnsZoneIID irs.IID, recordIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(dnsHandler.Region, call.DNSZONE, recordIID.NameId, "RemoveDNSRecord()")
	start := call.Start()

	name, recordType, err := parseRecordSystemId(recordIID.SystemId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	rrSets, err := dnsHandler.listAllResourceRecordSets(dnsZoneIID.SystemId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	var target *route53.ResourceRecordSet
	for _, rrSet := range rrSets {
		if aws.StringValue(rrSet.Name) == fqdn(name) && aws.StringValue(rrSet.Type) == recordType {
			target = rrSet
			break
		}
	}
	if target == nil {
		err := fmt.Errorf("DNS record '%s'(%s) does not exist", name, recordType)
		LoggingError(hiscallInfo, err)
		return false, err
	}

	_, err = dnsHandler.Client.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(dnsZoneIID.SystemId),
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action:            aws.String(route53.ChangeActionDelete),
					ResourceRecordSet: target,
				},
			},
		},
	})
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	LoggingInfo(hiscallInfo, start)
	return true, nil
}

//------------------------- internal helpers -------------------------

func (dnsHandler *AwsDNSHandler) listAllHostedZones() ([]*route53.HostedZone, error) {
	zones := []*route53.HostedZone{}
	marker := ""
	for {
		input := &route53.ListHostedZonesInput{}
		if marker != "" {
			input.Marker = aws.String(marker)
		}
		result, err := dnsHandler.Client.ListHostedZones(input)
		if err != nil {
			return nil, err
		}
		zones = append(zones, result.HostedZones...)
		if !aws.BoolValue(result.IsTruncated) {
			break
		}
		marker = aws.StringValue(result.NextMarker)
	}
	return zones, nil
}

func (dnsHandler *AwsDNSHandler) listAllResourceRecordSets(hostedZoneId string) ([]*route53.ResourceRecordSet, error) {
	rrSets := []*route53.ResourceRecordSet{}
	input := &route53.ListResourceRecordSetsInput{HostedZoneId: aws.String(hostedZoneId)}
	for {
		result, err := dnsHandler.Client.ListResourceRecordSets(input)
		if err != nil {
			return nil, err
		}
		rrSets = append(rrSets, result.ResourceRecordSets...)
		if !aws.BoolValue(result.IsTruncated) {
			break
		}
		input = &route53.ListResourceRecordSetsInput{
			HostedZoneId:          aws.String(hostedZoneId),
			StartRecordName:       result.NextRecordName,
			StartRecordType:       result.NextRecordType,
			StartRecordIdentifier: result.NextRecordIdentifier,
		}
	}
	return rrSets, nil
}

func (dnsHandler *AwsDNSHandler) getDNSZoneInfo(zone *route53.HostedZone, delegationSet *route53.DelegationSet) (irs.DNSZoneInfo, error) {
	rrSets, err := dnsHandler.listAllResourceRecordSets(aws.StringValue(zone.Id))
	if err != nil {
		return irs.DNSZoneInfo{}, err
	}

	nameServers := []string{}
	if delegationSet != nil {
		nameServers = aws.StringValueSlice(delegationSet.NameServers)
	} else {
		// NS records of the zone apex carry the same name servers when the
		// DelegationSet was not returned by the calling API(ex: GetHostedZone).
		zoneName := aws.StringValue(zone.Name)
		for _, rrSet := range rrSets {
			if aws.StringValue(rrSet.Type) == route53.RRTypeNs && aws.StringValue(rrSet.Name) == zoneName {
				for _, rr := range rrSet.ResourceRecords {
					nameServers = append(nameServers, aws.StringValue(rr.Value))
				}
			}
		}
	}

	recordInfoList := []irs.DNSRecordInfo{}
	for _, rrSet := range rrSets {
		recordInfoList = append(recordInfoList, convertToDNSRecordInfo(rrSet))
	}

	return irs.DNSZoneInfo{
		IId:            irs.IID{NameId: aws.StringValue(zone.Name), SystemId: aws.StringValue(zone.Id)},
		DomainName:     aws.StringValue(zone.Name),
		NameServers:    nameServers,
		RecordInfoList: recordInfoList,
	}, nil
}

func convertToDNSRecordInfo(rrSet *route53.ResourceRecordSet) irs.DNSRecordInfo {
	values := []string{}
	for _, rr := range rrSet.ResourceRecords {
		values = append(values, aws.StringValue(rr.Value))
	}
	name := aws.StringValue(rrSet.Name)
	recordType := aws.StringValue(rrSet.Type)
	return irs.DNSRecordInfo{
		IId:   irs.IID{NameId: name, SystemId: makeRecordSystemId(name, recordType)},
		Type:  irs.DNSRecordType(recordType),
		Name:  name,
		Value: values,
		TTL:   int(aws.Int64Value(rrSet.TTL)),
	}
}

// isDefaultApexRecord reports whether the record set is the NS/SOA pair Route53
// auto-creates for the zone apex. Those cannot be deleted individually; they are
// removed automatically by DeleteHostedZone.
func isDefaultApexRecord(rrSet *route53.ResourceRecordSet, zoneName string) bool {
	if aws.StringValue(rrSet.Name) != zoneName {
		return false
	}
	recordType := aws.StringValue(rrSet.Type)
	return recordType == route53.RRTypeNs || recordType == route53.RRTypeSoa
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
