// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This covers only the Public Authoritative DNS baseline (Zone + Record CRUD).
//
// Known gap(follow-up): unlike VPC/Subnet, this manager does not yet support
// PERMISSION_BASED_CONTROL_MODE (cross-connection IID lookup via
// getAuthIIDInfoList/getAuthIIDInfo). When that env var is set, DNS Zone/Record
// calls still behave as if it were unset.
//
// by CB-Spider Team, 2026.09.

package commonruntime

import (
	"fmt"
	"strconv"

	splock "github.com/cloud-barista/cb-spider/api-runtime/common-runtime/sp-lock"
	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	iidm "github.com/cloud-barista/cb-spider/cloud-control-manager/iid-manager"
	infostore "github.com/cloud-barista/cb-spider/info-store"
)

// ====================================================================
// type for GORM
type DNSZoneIIDInfo FirstIIDInfo

func (DNSZoneIIDInfo) TableName() string {
	return "dns_zone_iid_infos"
}

// DNSRecordIIDInfo does not use the usual "SP-XID:CSP-ID" SystemId scheme,
// because a Public DNS record has no CSP-assigned opaque ID of its own -
// it is addressed by (Name, Type) inside its zone, which the driver already
// returns as a deterministic SystemId("Name:Type"). So SystemId here IS the
// driver SystemId, with no translation layer.
type DNSRecordIIDInfo struct {
	ConnectionName   string `gorm:"primaryKey"` // ex) "aws-seoul-config"
	NameId           string `gorm:"primaryKey"` // ex) "my-a-record"
	SystemId         string // ex) "www.example.com.:A"
	OwnerDNSZoneName string `gorm:"primaryKey"` // ex) "my-zone"
}

func (DNSRecordIIDInfo) TableName() string {
	return "dns_record_iid_infos"
}

const OWNER_DNSZONE_NAME_COLUMN = "owner_dns_zone_name"

//====================================================================

func init() {
	db, err := infostore.Open()
	if err != nil {
		cblog.Error(err)
		return
	}
	db.AutoMigrate(&DNSZoneIIDInfo{})
	db.AutoMigrate(&DNSRecordIIDInfo{})
	infostore.Close(db)
}

var dnsZoneSPLock = splock.New()

//================ DNS Zone Handler

// RegisterDNSZone registers an existing(CSP-created) Public DNS Zone into CB-Spider.
// UserIID{UserID, CSP-ID} => SpiderIID{UserID, SP-XID:CSP-ID}
func RegisterDNSZone(connectionName string, userIID cres.IID) (*cres.DNSZoneInfo, error) {
	cblog.Info("call RegisterDNSZone()")

	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	emptyPermissionList := []string{}
	err = ValidateStruct(userIID, emptyPermissionList)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateDNSHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	dnsZoneSPLock.Lock(connectionName, userIID.NameId)
	defer dnsZoneSPLock.Unlock(connectionName, userIID.NameId)

	bool_ret, err := infostore.HasByConditions(&DNSZoneIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, userIID.NameId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	if bool_ret {
		err := fmt.Errorf("%s '%s' already exists in connection '%s'", RSTypeString(DNS), userIID.NameId, connectionName)
		cblog.Error(err)
		return nil, err
	}

	getInfo, err := handler.GetDNSZone(cres.IID{NameId: userIID.SystemId, SystemId: userIID.SystemId})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	systemId := getMSShortID(getInfo.IId.SystemId)
	spiderIId := cres.IID{NameId: userIID.NameId, SystemId: systemId + ":" + getInfo.IId.SystemId}

	err = infostore.Insert(&DNSZoneIIDInfo{ConnectionName: connectionName, NameId: spiderIId.NameId, SystemId: spiderIId.SystemId})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// register pre-existing records(except the zone-apex NS/SOA pair) under generated NameIds
	recordInfoList := []cres.DNSRecordInfo{}
	count := 0
	for _, recordInfo := range getInfo.RecordInfoList {
		if recordInfo.Name == getInfo.DomainName && (recordInfo.Type == cres.DNSRecordTypeNS || recordInfo.Type == "SOA") {
			continue
		}
		recordUserId := userIID.NameId + "-record-" + strconv.Itoa(count)
		count++
		err = infostore.Insert(&DNSRecordIIDInfo{ConnectionName: connectionName, NameId: recordUserId, SystemId: recordInfo.IId.SystemId, OwnerDNSZoneName: userIID.NameId})
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		recordInfo.IId = cres.IID{NameId: recordUserId, SystemId: recordInfo.IId.SystemId}
		recordInfoList = append(recordInfoList, recordInfo)
	}

	getInfo.IId = userIID
	getInfo.RecordInfoList = recordInfoList

	return &getInfo, nil
}

func CreateDNSZone(connectionName string, rsType string, reqInfo cres.DNSZoneReqInfo, IDTransformMode string) (*cres.DNSZoneInfo, error) {
	cblog.Info("call CreateDNSZone()")

	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	emptyPermissionList := []string{
		"resources.IID:SystemId",
		"resources.DNSZoneReqInfo:TagList",
	}
	err = ValidateStruct(reqInfo, emptyPermissionList)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateDNSHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	dnsZoneSPLock.Lock(connectionName, reqInfo.IId.NameId)
	defer dnsZoneSPLock.Unlock(connectionName, reqInfo.IId.NameId)

	bool_ret, err := infostore.HasByConditions(&DNSZoneIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, reqInfo.IId.NameId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	if bool_ret {
		err := fmt.Errorf("%s '%s' already exists in connection '%s'", RSTypeString(rsType), reqInfo.IId.NameId, connectionName)
		cblog.Error(err)
		return nil, err
	}

	spUUID := ""
	if GetID_MGMT(IDTransformMode) == "ON" {
		spUUID, err = iidm.New(connectionName, rsType, reqInfo.IId.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	} else {
		spUUID = reqInfo.IId.NameId
	}

	reqIId := cres.IID{NameId: reqInfo.IId.NameId, SystemId: spUUID}
	driverIId := cres.IID{NameId: spUUID, SystemId: ""}
	reqInfo.IId = driverIId

	info, err := handler.CreateDNSZone(reqInfo)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	spiderIId := cres.IID{NameId: reqIId.NameId, SystemId: spUUID + ":" + info.IId.SystemId}
	err = infostore.Insert(&DNSZoneIIDInfo{ConnectionName: connectionName, NameId: spiderIId.NameId, SystemId: spiderIId.SystemId})
	if err != nil {
		cblog.Error(err)
		// rollback
		_, err2 := handler.DeleteDNSZone(info.IId)
		if err2 != nil {
			cblog.Error(err2)
			return nil, fmt.Errorf("%s", err.Error()+", "+err2.Error())
		}
		return nil, err
	}

	info.IId = cres.IID{NameId: reqIId.NameId, SystemId: info.IId.SystemId}
	info.RecordInfoList = []cres.DNSRecordInfo{} // the zone's default apex NS/SOA records are CSP-managed, not CB-Spider-managed

	return &info, nil
}

type ResultDNSZoneInfo struct {
	dnsZoneInfo cres.DNSZoneInfo
	err         error
}

func ListDNSZone(connectionName string, rsType string) ([]*cres.DNSZoneInfo, error) {
	cblog.Info("call ListDNSZone()")

	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateDNSHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	var iidInfoList []*DNSZoneIIDInfo
	err = infostore.ListByCondition(&iidInfoList, CONNECTION_NAME_COLUMN, connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	if iidInfoList == nil || len(iidInfoList) <= 0 {
		return []*cres.DNSZoneInfo{}, nil
	}

	resultInfoList := []*cres.DNSZoneInfo{}
	var errList []string
	for _, iidInfo := range iidInfoList {
		info, err := getDNSZoneInfoWithRecords(connectionName, handler, cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})
		if err != nil {
			if checkNotFoundError(err) {
				cblog.Error(err)
				resultInfoList = append(resultInfoList, &cres.DNSZoneInfo{IId: cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}})
				continue
			}
			errList = append(errList, connectionName+":DNSZone:"+iidInfo.NameId+" # "+err.Error())
			continue
		}
		resultInfoList = append(resultInfoList, info)
	}

	if len(errList) > 0 {
		cblog.Error(errList)
		return nil, fmt.Errorf("%v", errList)
	}

	return resultInfoList, nil
}

func GetDNSZone(connectionName string, rsType string, nameID string) (*cres.DNSZoneInfo, error) {
	cblog.Info("call GetDNSZone()")

	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	nameID, err = EmptyCheckAndTrim("nameID", nameID)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateDNSHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	dnsZoneSPLock.RLock(connectionName, nameID)
	defer dnsZoneSPLock.RUnlock(connectionName, nameID)

	var iidInfo DNSZoneIIDInfo
	err = infostore.GetByConditions(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	return getDNSZoneInfoWithRecords(connectionName, handler, cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})
}

func getDNSZoneInfoWithRecords(connectionName string, handler cres.DNSHandler, spiderIId cres.IID) (*cres.DNSZoneInfo, error) {
	info, err := handler.GetDNSZone(getDriverIID(spiderIId))
	if err != nil {
		return nil, err
	}

	info.IId = getUserIID(spiderIId)
	info.RecordInfoList = attachRecordNameIds(connectionName, info.IId.NameId, info.RecordInfoList)

	return &info, nil
}

// attachRecordNameIds keeps only the records CB-Spider itself registered for
// this zone(dropping CSP-default or other-tool-managed ones, exactly like
// VPC/Subnet does), and sets each one's IId.NameId to its CB-Spider alias.
func attachRecordNameIds(connectionName string, zoneUserName string, recordInfoList []cres.DNSRecordInfo) []cres.DNSRecordInfo {
	filtered := []cres.DNSRecordInfo{}
	for _, recordInfo := range recordInfoList {
		var recIIDInfo DNSRecordIIDInfo
		err := infostore.GetBy3Conditions(&recIIDInfo, CONNECTION_NAME_COLUMN, connectionName, OWNER_DNSZONE_NAME_COLUMN, zoneUserName, SYSTEM_ID_COLUMN, recordInfo.IId.SystemId)
		if err != nil {
			if checkNotFoundError(err) {
				continue
			}
			cblog.Info(err)
			continue
		}
		recordInfo.IId = cres.IID{NameId: recIIDInfo.NameId, SystemId: recordInfo.IId.SystemId}
		filtered = append(filtered, recordInfo)
	}
	return filtered
}

func DeleteDNSZone(connectionName string, rsType string, nameID string, force string) (bool, error) {
	cblog.Info("call DeleteDNSZone()")

	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}
	nameID, err = EmptyCheckAndTrim("nameID", nameID)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	handler, err := cldConn.CreateDNSHandler()
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	dnsZoneSPLock.Lock(connectionName, nameID)
	defer dnsZoneSPLock.Unlock(connectionName, nameID)

	var iidInfo DNSZoneIIDInfo
	err = infostore.GetByConditions(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	driverIId := getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId})

	result, err := handler.DeleteDNSZone(driverIId)
	if err != nil {
		cblog.Error(err)
		if checkNotFoundError(err) {
			force = "true"
		} else if force != "true" {
			return false, err
		}
	}
	if force != "true" {
		if !result {
			return result, nil
		}
	}

	_, err = infostore.DeleteByConditions(&DNSZoneIIDInfo{}, CONNECTION_NAME_COLUMN, iidInfo.ConnectionName, NAME_ID_COLUMN, iidInfo.NameId)
	if err != nil {
		cblog.Error(err)
		if force != "true" {
			return false, err
		}
	}
	_, err = infostore.DeleteByConditions(&DNSRecordIIDInfo{}, CONNECTION_NAME_COLUMN, iidInfo.ConnectionName, OWNER_DNSZONE_NAME_COLUMN, nameID)
	if err != nil {
		cblog.Error(err)
		if force != "true" {
			return false, err
		}
	}

	return result, nil
}

//================ DNS Record Handler

func AddDNSRecord(connectionName string, rsType string, zoneName string, reqInfo cres.DNSRecordReqInfo) (*cres.DNSZoneInfo, error) {
	cblog.Info("call AddDNSRecord()")

	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	zoneName, err = EmptyCheckAndTrim("zoneName", zoneName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	emptyPermissionList := []string{"resources.IID:SystemId"}
	err = ValidateStruct(reqInfo, emptyPermissionList)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateDNSHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	dnsZoneSPLock.Lock(connectionName, zoneName)
	defer dnsZoneSPLock.Unlock(connectionName, zoneName)

	bool_ret, err := infostore.HasBy3Conditions(&DNSRecordIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, OWNER_DNSZONE_NAME_COLUMN, zoneName, NAME_ID_COLUMN, reqInfo.IId.NameId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	if bool_ret {
		err := fmt.Errorf("%s '%s' already exists in DNS Zone '%s'", RSTypeString(DNSRECORD), reqInfo.IId.NameId, zoneName)
		cblog.Error(err)
		return nil, err
	}

	var iidZoneInfo DNSZoneIIDInfo
	err = infostore.GetByConditions(&iidZoneInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, zoneName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	zoneDriverIId := getDriverIID(cres.IID{NameId: iidZoneInfo.NameId, SystemId: iidZoneInfo.SystemId})

	recordNameId := reqInfo.IId.NameId
	info, err := handler.AddDNSRecord(zoneDriverIId, reqInfo)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	systemId := findRecordSystemId(info.RecordInfoList, reqInfo.Name, reqInfo.Type)
	if systemId == "" {
		err := fmt.Errorf("the created DNS record was not found in the driver's response")
		cblog.Error(err)
		return nil, err
	}

	err = infostore.Insert(&DNSRecordIIDInfo{ConnectionName: connectionName, NameId: recordNameId, SystemId: systemId, OwnerDNSZoneName: zoneName})
	if err != nil {
		cblog.Error(err)
		// rollback
		_, err2 := handler.RemoveDNSRecord(zoneDriverIId, cres.IID{SystemId: systemId})
		if err2 != nil {
			cblog.Error(err2)
			return nil, fmt.Errorf("%s", err.Error()+", "+err2.Error())
		}
		return nil, err
	}

	info.IId = getUserIID(cres.IID{NameId: iidZoneInfo.NameId, SystemId: iidZoneInfo.SystemId})
	info.RecordInfoList = attachRecordNameIds(connectionName, zoneName, info.RecordInfoList)

	return &info, nil
}

func ChangeDNSRecord(connectionName string, zoneName string, reqInfo cres.DNSRecordReqInfo) (*cres.DNSZoneInfo, error) {
	cblog.Info("call ChangeDNSRecord()")

	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	zoneName, err = EmptyCheckAndTrim("zoneName", zoneName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	emptyPermissionList := []string{"resources.IID:SystemId"}
	err = ValidateStruct(reqInfo, emptyPermissionList)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateDNSHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	dnsZoneSPLock.Lock(connectionName, zoneName)
	defer dnsZoneSPLock.Unlock(connectionName, zoneName)

	var recIIDInfo DNSRecordIIDInfo
	err = infostore.GetBy3Conditions(&recIIDInfo, CONNECTION_NAME_COLUMN, connectionName, OWNER_DNSZONE_NAME_COLUMN, zoneName, NAME_ID_COLUMN, reqInfo.IId.NameId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	var iidZoneInfo DNSZoneIIDInfo
	err = infostore.GetByConditions(&iidZoneInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, zoneName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	zoneDriverIId := getDriverIID(cres.IID{NameId: iidZoneInfo.NameId, SystemId: iidZoneInfo.SystemId})

	info, err := handler.ChangeDNSRecord(zoneDriverIId, reqInfo)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// ChangeDNSRecord normally keeps the record's (Name, Type) identity, so its
	// deterministic SystemId("Name:Type") does not change. Re-pin it defensively
	// in case the caller supplied a differently-cased/dotted Name for the same record.
	newSystemId := findRecordSystemId(info.RecordInfoList, reqInfo.Name, reqInfo.Type)
	if newSystemId != "" && newSystemId != recIIDInfo.SystemId {
		_, err = infostore.DeleteBy3Conditions(&DNSRecordIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, OWNER_DNSZONE_NAME_COLUMN, zoneName, NAME_ID_COLUMN, recIIDInfo.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		err = infostore.Insert(&DNSRecordIIDInfo{ConnectionName: connectionName, NameId: recIIDInfo.NameId, SystemId: newSystemId, OwnerDNSZoneName: zoneName})
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
	}

	info.IId = getUserIID(cres.IID{NameId: iidZoneInfo.NameId, SystemId: iidZoneInfo.SystemId})
	info.RecordInfoList = attachRecordNameIds(connectionName, zoneName, info.RecordInfoList)

	return &info, nil
}

func RemoveDNSRecord(connectionName string, zoneName string, nameID string, force string) (bool, error) {
	cblog.Info("call RemoveDNSRecord()")

	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}
	zoneName, err = EmptyCheckAndTrim("zoneName", zoneName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}
	nameID, err = EmptyCheckAndTrim("nameID", nameID)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	dnsZoneSPLock.Lock(connectionName, zoneName)
	defer dnsZoneSPLock.Unlock(connectionName, zoneName)

	var recIIDInfo DNSRecordIIDInfo
	err = infostore.GetBy3Conditions(&recIIDInfo, CONNECTION_NAME_COLUMN, connectionName, OWNER_DNSZONE_NAME_COLUMN, zoneName, NAME_ID_COLUMN, nameID)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	var iidZoneInfo DNSZoneIIDInfo
	err = infostore.GetByConditions(&iidZoneInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, zoneName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}
	handler, err := cldConn.CreateDNSHandler()
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	zoneDriverIId := getDriverIID(cres.IID{NameId: iidZoneInfo.NameId, SystemId: iidZoneInfo.SystemId})

	result, err := handler.RemoveDNSRecord(zoneDriverIId, cres.IID{NameId: recIIDInfo.NameId, SystemId: recIIDInfo.SystemId})
	if err != nil {
		cblog.Error(err)
		if checkNotFoundError(err) {
			force = "true"
		} else if force != "true" {
			return false, err
		}
	}
	if force != "true" {
		if !result {
			return result, nil
		}
	}

	_, err = infostore.DeleteBy3Conditions(&DNSRecordIIDInfo{}, CONNECTION_NAME_COLUMN, connectionName, OWNER_DNSZONE_NAME_COLUMN, zoneName, NAME_ID_COLUMN, nameID)
	if err != nil {
		cblog.Error(err)
		if force != "true" {
			return false, err
		}
	}

	return result, nil
}

func findRecordSystemId(recordInfoList []cres.DNSRecordInfo, name string, recordType cres.DNSRecordType) string {
	normalized := fqdn(name)
	for _, r := range recordInfoList {
		if r.Type == recordType && (r.Name == name || r.Name == normalized) {
			return r.IId.SystemId
		}
	}
	return ""
}

func fqdn(name string) string {
	if name == "" || name[len(name)-1] == '.' {
		return name
	}
	return name + "."
}

func CountAllDNSZones() (int64, error) {
	var info DNSZoneIIDInfo
	return infostore.CountAllNameIDs(&info)
}

func CountDNSZonesByConnection(connectionName string) (int64, error) {
	var info DNSZoneIIDInfo
	return infostore.CountNameIDsByConnection(&info, connectionName)
}

func CountAllDNSRecords() (int64, error) {
	var info DNSRecordIIDInfo
	return infostore.CountAllNameIDs(&info)
}

func CountDNSRecordsByConnection(connectionName string) (int64, error) {
	var info DNSRecordIIDInfo
	return infostore.CountNameIDsByConnection(&info, connectionName)
}
