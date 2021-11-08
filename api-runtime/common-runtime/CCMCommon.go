// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.06.

package commonruntime

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	iidm "github.com/cloud-barista/cb-spider/cloud-control-manager/iid-manager"
	"github.com/cloud-barista/cb-store/config"
	"github.com/sirupsen/logrus"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
)

// define string of resource types
const (
	rsImage string = "image"
	rsVPC   string = "vpc"
	rsSubnet string = "subnet"	
	rsSG  string = "sg"
	rsKey string = "keypair"
	rsVM  string = "vm"
)

// definition of RWLock for each Resource Ops
var imgRWLock = new(sync.RWMutex)
var vpcRWLock = new(sync.RWMutex)
var sgRWLock = new(sync.RWMutex)
var keyRWLock = new(sync.RWMutex)
//var vmRWLock = new(sync.RWMutex)

// definition of IIDManager RWLock
var iidRWLock = new(iidm.IIDRWLOCK)

var cblog *logrus.Logger
var callogger *logrus.Logger

func init() {
	cblog = config.Cblogger
	// logger for HisCall
        callogger = call.GetLogger("HISCALL")
}

type AllResourceList struct {
	AllList struct {
		MappedList     []*cres.IID `json:"MappedList"`
		OnlySpiderList []*cres.IID `json:"OnlySpiderList"`
		OnlyCSPList    []*cres.IID `json:"OnlyCSPList"`
	}
}

//================ Image Handler
// @todo
// (1) check exist(NameID)
// (2) gen SP-XID and create userIID, driverIID
// (3) create Resource
// (4) create spiderIID: {UserNameID, "DriverNameID:CSPSystemID"}
// (5) insert spiderIID
func CreateImage(connectionName string, rsType string, reqInfo cres.ImageReqInfo) (*cres.ImageInfo, error) {
	cblog.Info("call CreateImage()")

	// check empty and trim user inputs
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

	handler, err := cldConn.CreateImageHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	imgRWLock.Lock()
	defer imgRWLock.Unlock()
	// (1) check exist(NameID)
	bool_ret, err := iidRWLock.IsExistIID(iidm.IIDSGROUP, connectionName, rsType, reqInfo.IId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	if bool_ret == true {
		return nil, fmt.Errorf(reqInfo.IId.NameId + " already exists!")
	}

	// (2) gen SP-XID and create userIID, driverIID
	//     ex) SP-XID{"vm-01-9m4e2mr0ui3e8a215n4g"}
	//     ex) userIID{"seoul-service", "vm-01-9m4e2mr0ui3e8a215n4g"}, 
	//     ex) driverIID{"vm-01-9m4e2mr0ui3e8a215n4g", ""}
	spiderUUID, err := iidm.New(connectionName, reqInfo.IId.NameId)
	if err != nil {
                cblog.Error(err)
                return nil, err
        }

	userIId := cres.IID{reqInfo.IId.NameId, spiderUUID}
	driverIId := cres.IID{spiderUUID, ""}
	reqInfo.IId = driverIId 

	// (3) create Resource
	info, err := handler.CreateImage(reqInfo)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (4) create spiderIID: {UserNameID, "DriverNameID:CSPSystemID"}
	//     ex) driverIID{"vm-01-9m4e2mr0ui3e8a215n4g", "i-0bc7123b7e5cbf79d"}
	//     ex) spiderIID{"seoul-service", "vm-01-9m4e2mr0ui3e8a215n4g:i-0bc7123b7e5cbf79d"}	
	spiderIId := cres.IID{userIId.NameId, info.IId.NameId + ":" + info.IId.SystemId}

	// (5) insert spiderIID
	iidInfo, err := iidRWLock.CreateIID(iidm.IIDSGROUP, connectionName, rsType, spiderIId)
	if err != nil {
		cblog.Error(err)
		// rollback
		_, err2 := handler.DeleteImage(iidInfo.IId)
		if err2 != nil {
			cblog.Error(err2)
			return nil, err2
		}
		return nil, err
	}

	return &info, nil
}

// (1) get IID:list
// (2) get CSP:list
// (3) filtering CSP-list by IID-list
func ListImage(connectionName string, rsType string) ([]*cres.ImageInfo, error) {
	cblog.Info("call ListImage()")

	// check empty and trim user inputs
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

	handler, err := cldConn.CreateImageHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	infoList, err := handler.ListImage()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	if infoList == nil || len(infoList) <= 0 {
		infoList = []*cres.ImageInfo{}
	}

	return infoList, nil
}

// (1) get spiderIID:list
// (2) get CSP:list
// (3) filtering CSP-list by spiderIID-list
func ListRegisterImage(connectionName string, rsType string) ([]*cres.ImageInfo, error) {
	cblog.Info("call ListImage()")

	// check empty and trim user inputs
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

	handler, err := cldConn.CreateImageHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	imgRWLock.RLock()
	defer imgRWLock.RUnlock()
	// (1) get spiderIID:list
	iidInfoList, err := iidRWLock.ListIID(iidm.IIDSGROUP, connectionName, rsType)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	var infoList []*cres.ImageInfo
	if iidInfoList == nil || len(iidInfoList) <= 0 {
		infoList = []*cres.ImageInfo{}
		return infoList, nil
	}

	// (2) get CSP:list
	infoList, err = handler.ListImage()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	if infoList == nil { // if iidInfoList not null, then infoList has any list.
		return nil, fmt.Errorf("<IID-CSP mismatch> " + rsType + " IID List has " + strconv.Itoa(len(iidInfoList)) + ", but " + connectionName + " Resource list has nothing!")
	}

	// (3) filtering CSP-list by spiderIID-list
	infoList2 := []*cres.ImageInfo{}
	for _, iidInfo := range iidInfoList {
		exist := false
		driverIId := getDriverIID(iidInfo.IId)
		for _, info := range infoList {			
			if driverIId.SystemId == info.IId.SystemId {
				info.IId.NameId = iidInfo.IId.NameId
				infoList2 = append(infoList2, info)
				exist = true
			}
		}
		if exist == false {
			cblog.Info("<IID-CSP mismatch> " + rsType + "-" + iidInfo.IId.NameId + ":" + driverIId.SystemId + " exsits. but " + connectionName + " does not have!")
			//return nil, fmt.Errorf("<IID-CSP mismatch> " + rsType + "-" + iidInfo.IId.NameId + ":" + driverIId.SystemId + " exsits. but " + connectionName + " does not have!")
		}
	}

	return infoList2, nil
}

// (1) get resource(SystemId)
// (2) set ResourceInfo(IID.NameId)
func GetImage(connectionName string, rsType string, nameID string) (*cres.ImageInfo, error) {
	cblog.Info("call GetImage()")

	// check empty and trim user inputs
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

	handler, err := cldConn.CreateImageHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// now, NameID = SystemID
	info, err := handler.GetImage(cres.IID{nameID, nameID})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	return &info, nil
}

// (1) get spiderIID(NameId)
// (2) extract driverIID from spiderIID
// (3) get resource(SystemId)
// (4) set ResourceInfo(IID.NameId)
func GetRegisterImage(connectionName string, rsType string, nameID string) (*cres.ImageInfo, error) {
	cblog.Info("call GetImage()")

	// check empty and trim user inputs
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

	handler, err := cldConn.CreateImageHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	imgRWLock.RLock()
	defer imgRWLock.RUnlock()
	// (1) get spiderIID(NameId)
	iidInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsType, cres.IID{nameID, ""})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (2) extract driverIID from spiderIID
	driverIId := getDriverIID(iidInfo.IId)

	// (3) get resource(SystemId)
	start := time.Now()
	info, err := handler.GetImage(driverIId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	elapsed := time.Since(start)
	cblog.Infof(connectionName+" : elapsed %d", elapsed.Nanoseconds()/1000000) // msec

	// (4) set ResourceInfo(IID.NameId)
	info.IId.NameId = iidInfo.IId.NameId

	return &info, nil
}

// (1) get spiderIID(NameId)
// (2) extract driverIID from spiderIID
// (3) delete Resource(SystemId)
// (4) delete spiderIID
func DeleteImage(connectionName string, rsType string, nameID string) (bool, error) {
	cblog.Info("call DeleteImage()")

	// check empty and trim user inputs
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

	handler, err := cldConn.CreateImageHandler()
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	imgRWLock.Lock()
	defer imgRWLock.Unlock()
	// (1) get spiderIID(NameId)
	iidInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsType, cres.IID{nameID, ""})
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	// (2) extract driverIID from spiderIID
	driverIId := getDriverIID(iidInfo.IId)


	// keeping for rollback
	info, err := handler.GetImage(driverIId)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	// (3) delete Resource(SystemId)
	result, err := handler.DeleteImage(driverIId)
	if err != nil {
		cblog.Error(err)
		return false, err
	}
	if result == false {
		return result, nil
	}

	// (4) delete spiderIID
	_, err = iidRWLock.DeleteIID(iidm.IIDSGROUP, connectionName, rsType, iidInfo.IId)
	if err != nil {
		cblog.Error(err)
		// rollback
		reqInfo := cres.ImageReqInfo{info.IId} // @todo
		_, err2 := handler.CreateImage(reqInfo)
		if err2 != nil {
			cblog.Error(err2)
			return false, fmt.Errorf(err.Error() + ", " + err2.Error())
		}
		return false, err
	}

	return result, nil
}

//================ VMSpec Handler
func ListVMSpec(connectionName string) ([]*cres.VMSpecInfo, error) {
	cblog.Info("call ListVMSpec()")

	// check empty and trim user inputs
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

	regionName, _, err := ccm.GetRegionNameByConnectionName(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateVMSpecHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	infoList, err := handler.ListVMSpec(regionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	if infoList == nil || len(infoList) <= 0 {
		infoList = []*cres.VMSpecInfo{}
	}

	return infoList, nil
}

func GetVMSpec(connectionName string, nameID string) (*cres.VMSpecInfo, error) {
	cblog.Info("call GetVMSpec()")

	// check empty and trim user inputs
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

	regionName, _, err := ccm.GetRegionNameByConnectionName(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateVMSpecHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	info, err := handler.GetVMSpec(regionName, nameID)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	return &info, nil
}

func ListOrgVMSpec(connectionName string) (string, error) {
	cblog.Info("call ListOrgVMSpec()")

	// check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
		cblog.Error(err)
                return "", err
        }

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	regionName, _, err := ccm.GetRegionNameByConnectionName(connectionName)
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	handler, err := cldConn.CreateVMSpecHandler()
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	infoList, err := handler.ListOrgVMSpec(regionName)
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	return infoList, nil
}

func GetOrgVMSpec(connectionName string, nameID string) (string, error) {
	cblog.Info("call GetOrgVMSpec()")

	// check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
		cblog.Error(err)
                return "", err
        }

        nameID, err = EmptyCheckAndTrim("nameID", nameID)
        if err != nil {
		cblog.Error(err)
                return "", err
        }

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	regionName, _, err := ccm.GetRegionNameByConnectionName(connectionName)
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	handler, err := cldConn.CreateVMSpecHandler()
	if err != nil {
		cblog.Error(err)
		return "", err
	}
	info, err := handler.GetOrgVMSpec(regionName, nameID)
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	return info, nil
}


//================ VPC Handler

// UserIID{UserID, CSP-ID} => SpiderIID{UserID, SP-XID:CSP-ID}
// (1) check existence(UserID)
// (2) get resource info(CSP-ID)
// (3) create spiderIID: {UserID, SP-XID:CSP-ID}
// (4) insert spiderIID
func RegisterVPC(connectionName string, userIID cres.IID) (*cres.VPCInfo, error) {
        cblog.Info("call RegisterVPC()")

	// check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
		cblog.Error(err)
                return nil, err
        }

        rsType := rsVPC

        cldConn, err := ccm.GetCloudConnection(connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

	handler, err := cldConn.CreateVPCHandler()
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        vpcRWLock.Lock()
        defer vpcRWLock.Unlock()

        // (1) check existence(UserID)
        bool_ret, err := iidRWLock.IsExistIID(iidm.IIDSGROUP, connectionName, rsType, userIID)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        if bool_ret == true {
                return nil, fmt.Errorf(rsType + "-" + userIID.NameId + " already exists!")
        }

        // (2) get resource info(CSP-ID)
        // check existence and get info of this resouce in the CSP
	// Do not user NamieId, because Azure driver use it like SystemId
        getInfo, err := handler.GetVPC( cres.IID{userIID.SystemId, userIID.SystemId} )
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // (3) create spiderIID: {UserID, SP-XID:CSP-ID}
        //     ex) spiderIID {"vpc-01", "vpc-01-9m4e2mr0ui3e8a215n4g:i-0bc7123b7e5cbf79d"}
	// Do not user NamieId, because Azure driver use it like SystemId
	systemId := getMSShortID(getInfo.IId.SystemId)
        spiderIId := cres.IID{userIID.NameId, systemId + ":" + getInfo.IId.SystemId}

        // (4) insert spiderIID
        // insert VPC SpiderIID to metadb
        _, err = iidRWLock.CreateIID(iidm.IIDSGROUP, connectionName, rsType, spiderIId)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // insert subnet's spiderIIDs to metadb and setup subnet IID for return info
        for count, subnetInfo := range getInfo.SubnetInfoList {
                // generate subnet's UserID
                subnetUserId := userIID.NameId + "-subnet-" + strconv.Itoa(count)


                // insert a subnet SpiderIID to metadb
		// Do not user NamieId, because Azure driver use it like SystemId
		systemId := getMSShortID(subnetInfo.IId.SystemId)
		subnetSpiderIId := cres.IID{subnetUserId, systemId + ":" + subnetInfo.IId.SystemId}
                _, err = iidRWLock.CreateIID(iidm.SUBNETGROUP, connectionName, userIID.NameId, subnetSpiderIId)
                if err != nil {
                        cblog.Error(err)
                        return nil, err
                }

                // setup subnet IID for return info
                subnetInfo.IId = cres.IID{subnetUserId, subnetInfo.IId.SystemId}
                getInfo.SubnetInfoList[count] = subnetInfo
        } // end of for _, info

        // set up VPC User IID for return info
        getInfo.IId = userIID

        return &getInfo, nil
}

func getMSShortID(inID string) string {
	// /subscriptions/a20fed83~/Microsoft.Network/~/sg01-c5n27e2ba5ofr0fnbck0
        // ==> sg01-c5n27e2ba5ofr0fnbck0
	var shortID string
        if strings.Contains(inID, "/Microsoft.") {
                strList := strings.Split(inID, "/")
                shortID = strList[len(strList)-1]
        } else {
                return inID
        }
	return shortID
}

// UnregisterResource API does not delete the real resource.
// This API just unregister the resource from Spider.
// (1) check exist(NameID)
// (2) delete SpiderIID
func UnregisterResource(connectionName string, rsType string, nameId string) (bool, error) {
        cblog.Info("call UnregisterResource()")

	// check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
		cblog.Error(err)
                return false, err
        }

        nameId, err = EmptyCheckAndTrim("nameId", nameId)
        if err != nil {
		cblog.Error(err)
                return false, err
        }

	switch rsType {
        case rsVPC:
                vpcRWLock.Lock()
                defer vpcRWLock.Unlock()
        case rsSG:
                sgRWLock.Lock()
                defer sgRWLock.Unlock()
        case rsKey:
                keyRWLock.Lock()
                defer keyRWLock.Unlock()
        case rsVM:
        //        vmRWLock.Lock()
        //        defer vmRWLock.Unlock()
        default:
                return false, fmt.Errorf(rsType + " is not supported Resource!!")
        }


        // (1) check existence(UserID)
	var isExist bool=false
	var vpcName string 
	switch rsType {
        case rsSG:
		iidInfoList, err := getAllSGIIDInfoList(connectionName)
		if err != nil {
			cblog.Error(err)
			return false, err
		}
		for _, OneIIdInfo := range iidInfoList {
			if OneIIdInfo.IId.NameId == nameId {
				vpcName = OneIIdInfo.ResourceType/*vpcName*/  // ---------- Don't forget
				isExist = true
				break
			}
		}
	default:
		// (1) check exist(NameID)
		var err error
		isExist, err = iidRWLock.IsExistIID(iidm.IIDSGROUP, connectionName, rsType, cres.IID{nameId, ""})
		if err != nil {
			cblog.Error(err)
			return false, err
		}
	} // end of switch

	if isExist == false {
		return false, fmt.Errorf(rsType + "-" + nameId + " does not exist!")
	}

	// (2) delete the IID from Metadb
	switch rsType {
        case rsVPC:
		// if vpc, delete all subnet meta data
                // (a) for vPC
		_, err := iidRWLock.DeleteIID(iidm.IIDSGROUP, connectionName, rsType, cres.IID{nameId, ""})
                if err != nil {
                        cblog.Error(err)
			return false, err
                }

                // (b) for Subnet list
                // key-value structure: ~/{SUBNETGROUP}/{ConnectionName}/{VPC-NameId}/{Subnet-reqNameId} [subnet-driverNameId:subnet-driverSystemId]  # VPC NameId => rsType
                subnetIIdInfoList, err2 := iidRWLock.ListIID(iidm.SUBNETGROUP, connectionName, nameId/*vpcName*/)
                if err2 != nil {
                        cblog.Error(err)
			return false, err
                }
                for _, subnetIIdInfo := range subnetIIdInfoList {
                        // key-value structure: ~/{SUBNETGROUP}/{ConnectionName}/{VPC-NameId}/{Subnet-reqNameId} [subnet-driverNameId:subnet-driverSystemId]  # VPC NameId => rsType
                        _, err := iidRWLock.DeleteIID(iidm.SUBNETGROUP, connectionName, nameId/*vpcName*/, subnetIIdInfo.IId)
                        if err != nil {
                                cblog.Error(err)
				return false, err
                        }
                }

                // @todo Should we also delete the SG list of this VPC ?


        case rsSG:
		_, err := iidRWLock.DeleteIID(iidm.SGGROUP, connectionName, vpcName/*rsType*/, cres.IID{nameId, ""})
		if err != nil {
			cblog.Error(err)
			return false, err
		}



	default: // other resources(key, vm, ...)
		_, err := iidRWLock.DeleteIID(iidm.IIDSGROUP, connectionName, rsType, cres.IID{nameId, ""})
		if err != nil {
			cblog.Error(err)
			return false, err
		}
	} // end of switch

	return true, nil
}


// (1) check exist(NameID)
// (2) generate SP-XID and create reqIID, driverIID
// (3) create Resource
// (4) create spiderIID: {reqNameID, "driverNameID:driverSystemID"}
// (5) insert spiderIID
// (6) create userIID
func CreateVPC(connectionName string, rsType string, reqInfo cres.VPCReqInfo) (*cres.VPCInfo, error) {
	cblog.Info("call CreateVPC()")

	// check empty and trim user inputs
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

	handler, err := cldConn.CreateVPCHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	vpcRWLock.Lock()
	defer vpcRWLock.Unlock()
	// (1) check exist(NameID)
	bool_ret, err := iidRWLock.IsExistIID(iidm.IIDSGROUP, connectionName, rsType, reqInfo.IId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	if bool_ret == true {
		return nil, fmt.Errorf(rsType + "-" + reqInfo.IId.NameId + " already exists!")
	}

	// (2) generate SP-XID and create reqIID, driverIID
	//     ex) SP-XID {"vm-01-9m4e2mr0ui3e8a215n4g"}
	//
	//     create reqIID: {reqNameID, reqSystemID}   # reqSystemID=SP-XID
	//         ex) reqIID {"seoul-service", "vm-01-9m4e2mr0ui3e8a215n4g"} 
	//
	//     create driverIID: {driverNameID, driverSystemID}   # driverNameID=SP-XID, driverSystemID=csp's ID
	//         ex) driverIID {"vm-01-9m4e2mr0ui3e8a215n4g", "i-0bc7123b7e5cbf79d"}
	spUUID, err := iidm.New(connectionName, reqInfo.IId.NameId)
	if err != nil {
                cblog.Error(err)
                return nil, err
        }

	// reqIID
	reqIId := cres.IID{reqInfo.IId.NameId, spUUID}
	// driverIID
	driverIId := cres.IID{spUUID, ""}
	reqInfo.IId = driverIId

	// for subnet list
	subnetReqIIdList := []cres.IID{}
	subnetInfoList := []cres.SubnetInfo{}
	for _, info := range reqInfo.SubnetInfoList {
		subnetUUID, err := iidm.New(connectionName, info.IId.NameId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}

		// reqIID
		subnetReqIId := cres.IID{info.IId.NameId, subnetUUID}
		subnetReqIIdList = append(subnetReqIIdList, subnetReqIId)
		// driverIID
		subnetDriverIId := cres.IID{subnetUUID, ""}
		info.IId = subnetDriverIId
		subnetInfoList = append(subnetInfoList, info)
	} // end of for _, info
	reqInfo.SubnetInfoList = subnetInfoList

	// (3) create Resource
	// VPC: driverIId, Subnet: driverIId List
	info, err := handler.CreateVPC(reqInfo)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (4) create spiderIID: {reqNameID, driverNameID:driverSystemID}
	//     ex) spiderIID {"seoul-service", "vm-01-9m4e2mr0ui3e8a215n4g:i-0bc7123b7e5cbf79d"}
	spiderIId := cres.IID{reqIId.NameId, info.IId.NameId + ":" + info.IId.SystemId}

	// (5) insert IID
	// for VPC
	iidInfo, err := iidRWLock.CreateIID(iidm.IIDSGROUP, connectionName, rsType, spiderIId)
	if err != nil {
		cblog.Error(err)
		// rollback
		_, err2 := handler.DeleteVPC(info.IId)
		if err2 != nil {
			cblog.Error(err2)
			return nil, fmt.Errorf(err.Error() + ", " + err2.Error())
		}
		return nil, err
	}
	// for Subnet list
	for _, subnetInfo := range info.SubnetInfoList {
		// key-value structure: ~/{SUBNETGROUP}/{ConnectionName}/{VPC-NameId}/{Subnet-reqNameId} [subnet-driverNameId:subnet-driverSystemId]  # VPC NameId => rsType
		subnetReqNameId := getReqNameId(subnetReqIIdList, subnetInfo.IId.NameId)
		if subnetReqNameId == "" {
			cblog.Error(subnetInfo.IId.NameId + "is not requested Subnet.")
			continue;
		}
		subnetSpiderIId := cres.IID{subnetReqNameId, subnetInfo.IId.NameId + ":" + subnetInfo.IId.SystemId}
		_, err := iidRWLock.CreateIID(iidm.SUBNETGROUP, connectionName, reqIId.NameId, subnetSpiderIId)
		if err != nil {
			cblog.Error(err)
			// rollback
			// (1) for resource
			cblog.Info("<<ROLLBACK:TRY:VPC-CSP>> " + info.IId.SystemId)
			_, err2 := handler.DeleteVPC(info.IId)
			if err2 != nil {
				cblog.Error(err2)
				return nil, fmt.Errorf(err.Error() + ", " + err2.Error())
			}
			// (2) for VPC IID
			cblog.Info("<<ROLLBACK:TRY:VPC-IID>> " + info.IId.NameId)
			_, err3 := iidRWLock.DeleteIID(iidm.IIDSGROUP, connectionName, rsType, iidInfo.IId)
			if err3 != nil {
				cblog.Error(err3)
				return nil, fmt.Errorf(err.Error() + ", " + err3.Error())
			}
			// (3) for Subnet IID
			tmpIIdInfoList, err := iidRWLock.ListIID(iidm.SUBNETGROUP, connectionName, info.IId.NameId) // VPC info.IId.NameId => rsType
			for _, subnetIIdInfo := range tmpIIdInfoList {
				_, err := iidRWLock.DeleteIID(iidm.SUBNETGROUP, connectionName, info.IId.NameId, subnetIIdInfo.IId) // VPC info.IId.NameId => rsType
				if err != nil {
					cblog.Error(err)
					return nil, err
				}
			}
			return nil, err
		}
	}

	// (6) create userIID: {reqNameID, driverSystemID}
	//     ex) userIID {"seoul-service", "i-0bc7123b7e5cbf79d"}
	// for VPC
	userIId := cres.IID{reqIId.NameId, info.IId.SystemId}
	info.IId = userIId

	// for Subnet list
	subnetUserInfoList := []cres.SubnetInfo{}
	for _, subnetInfo := range info.SubnetInfoList {
		subnetReqNameId := getReqNameId(subnetReqIIdList, subnetInfo.IId.NameId)
		userIId := cres.IID{subnetReqNameId, subnetInfo.IId.SystemId}
		subnetInfo.IId = userIId
		subnetUserInfoList = append(subnetUserInfoList, subnetInfo)
	}
	info.SubnetInfoList = subnetUserInfoList

	return &info, nil
}

// Get reqNameId from reqIIdList whith driver NameId
func getReqNameId(reqIIdList []cres.IID, driverNameId string) string {
	for _, iid := range reqIIdList {
		if iid.SystemId == driverNameId {
			return iid.NameId
		}
	}
	return ""
}

// (1) get IID:list
// (2) get CSP:list
// (3) filtering CSP-list by IID-list
// (4) set userIID
func ListVPC(connectionName string, rsType string) ([]*cres.VPCInfo, error) {
	cblog.Info("call ListVPC()")

	// check empty and trim user inputs
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

	handler, err := cldConn.CreateVPCHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	vpcRWLock.RLock()
	defer vpcRWLock.RUnlock()
	// (1) get IID:list
	iidInfoList, err := iidRWLock.ListIID(iidm.IIDSGROUP, connectionName, rsType)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	var infoList []*cres.VPCInfo
	if iidInfoList == nil || len(iidInfoList) <= 0 {
		infoList = []*cres.VPCInfo{}
		return infoList, nil
	}

	// (2) get CSP:list
	infoList, err = handler.ListVPC()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	if infoList == nil { // if iidInfoList not null, then infoList has any list.
		return nil, fmt.Errorf("<IID-CSP mismatch> " + rsType + " IID List has " + strconv.Itoa(len(iidInfoList)) + ", but " + connectionName + " Resource list has nothing!")
	}

	// (3) filtering CSP-list by IID-list
	resultInfoList := []*cres.VPCInfo{}
	for _, iidInfo := range iidInfoList {
		exist := false
		for _, info := range infoList {
			if getDriverSystemId(iidInfo.IId) == info.IId.SystemId {

				// (4) set UserIID{reqNameID, driverSystemID}				
				// for VPC				
				info.IId = getUserIID(iidInfo.IId)

				// for Subnet list
				// set userIID for SubnetInfo List
				// create new SubnetInfo List
				subnetInfoList := []cres.SubnetInfo{}
				for _, subnetInfo := range info.SubnetInfoList {					
					subnetIIDInfo, err := iidRWLock.GetIIDbySystemID(iidm.SUBNETGROUP, connectionName, iidInfo.IId.NameId, subnetInfo.IId) // VPC iidInfo.IId.NameId => rsType
					if err != nil {
						cblog.Error(err)
						return nil, err
					}
					if subnetIIDInfo.IId.NameId != "" { // insert only this user created.						
						subnetInfo.IId = getUserIID(subnetIIDInfo.IId)
						subnetInfoList = append(subnetInfoList, subnetInfo)
					}
				}
				info.SubnetInfoList = subnetInfoList

				resultInfoList = append(resultInfoList, info)
				exist = true
			}
		}
		if exist == false {
			cblog.Info("<IID-CSP mismatch> " + rsType + "-" + iidInfo.IId.NameId + ":" + iidInfo.IId.SystemId + " exsits. but " + connectionName + " does not have!")
			//return nil, fmt.Errorf("<IID-CSP mismatch> " + rsType + "-" + iidInfo.IId.NameId + ":" + iidInfo.IId.SystemId + " exsits. but " + connectionName + " does not have!")
		}
	}

	return resultInfoList, nil
}


// Get driverSystemId from SpiderIID
func getDriverSystemId(spiderIId cres.IID) string {
	strArray := strings.Split(spiderIId.SystemId, ":")
	return strArray[1]
}

// Get driverIID from SpiderIID
func getDriverIID(spiderIId cres.IID) cres.IID {
	strArray := strings.Split(spiderIId.SystemId, ":")
	driverIId := cres.IID{strArray[0], strArray[1]}
	return driverIId
}

// Get userIID from SpiderIID
func getUserIID(spiderIId cres.IID) cres.IID {
	strArray := strings.Split(spiderIId.SystemId, ":")
	userIId := cres.IID{spiderIId.NameId, strArray[1]}
	return userIId
}

// (1) get spiderIID(NameId)
// (2) get resource(driverIID)
// (3) set ResourceInfo(userIID)
func GetVPC(connectionName string, rsType string, nameID string) (*cres.VPCInfo, error) {
	cblog.Info("call GetVPC()")

	// check empty and trim user inputs
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

	handler, err := cldConn.CreateVPCHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	vpcRWLock.RLock()
	defer vpcRWLock.RUnlock()
	// (1) get spiderIID(NameId)
	iidInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsType, cres.IID{nameID, ""})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (2) get resource(driverIID)
	info, err := handler.GetVPC(getDriverIID(iidInfo.IId))
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	// (3) set ResourceInfo(userIID)
	info.IId = getUserIID(iidInfo.IId)

	// set NameId for SubnetInfo List
	// create new SubnetInfo List
	subnetInfoList := []cres.SubnetInfo{}
	for _, subnetInfo := range info.SubnetInfoList {		
		subnetIIDInfo, err := iidRWLock.GetIIDbySystemID(iidm.SUBNETGROUP, connectionName, info.IId.NameId, subnetInfo.IId) // VPC info.IId.NameId => rsType
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		if subnetIIDInfo.IId.NameId != "" { // insert only this user created.
			subnetInfo.IId = getUserIID(subnetIIDInfo.IId)
			subnetInfoList = append(subnetInfoList, subnetInfo)
		}
	}
	info.SubnetInfoList = subnetInfoList

	return &info, nil
}

// (1) check exist(NameID)
// (2) create Resource
// (3) insert IID
func AddSubnet(connectionName string, rsType string, vpcName string, reqInfo cres.SubnetInfo) (*cres.VPCInfo, error) {
	cblog.Info("call AddSubnet()")

	// check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
		cblog.Error(err)
                return nil, err
        }

        vpcName, err = EmptyCheckAndTrim("vpcName", vpcName)
        if err != nil {
		cblog.Error(err)
                return nil, err
        }

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateVPCHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	vpcRWLock.Lock()
	defer vpcRWLock.Unlock()
	// (1) check exist(NameID)
	bool_ret, err := iidRWLock.IsExistIID(iidm.SUBNETGROUP, connectionName, vpcName, reqInfo.IId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	if bool_ret == true {
		return nil, fmt.Errorf(rsType + "-" + reqInfo.IId.NameId + " already exists!")
	}
	// (2) create Resource
	iidVPCInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsVPC, cres.IID{vpcName, ""})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	subnetUUID, err := iidm.New(connectionName, reqInfo.IId.NameId)
	if err != nil {
                cblog.Error(err)
                return nil, err
        }

	// driverIID for driver
	subnetReqNameId := reqInfo.IId.NameId
	reqInfo.IId = cres.IID{subnetUUID, ""}
	info, err := handler.AddSubnet(getDriverIID(iidVPCInfo.IId), reqInfo)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (3) insert IID
	// for Subnet list
	for _, subnetInfo := range info.SubnetInfoList {		
		if subnetInfo.IId.NameId == reqInfo.IId.NameId {  // NameId => SS-UUID
			// key-value structure: ~/{SUBNETGROUP}/{ConnectionName}/{VPC-NameId}/{Subnet-reqNameId} [subnet-driverNameId:subnet-driverSystemId]  # VPC NameId => rsType
			subnetSpiderIId := cres.IID{subnetReqNameId, subnetInfo.IId.NameId + ":" + subnetInfo.IId.SystemId}
			_, err := iidRWLock.CreateIID(iidm.SUBNETGROUP, connectionName, vpcName, subnetSpiderIId)
			if err != nil {
				cblog.Error(err)
				// rollback
				// (1) for resource
				cblog.Info("<<ROLLBACK:TRY:VPC-SUBNET-CSP>> " + subnetInfo.IId.SystemId)
				_, err2 := handler.RemoveSubnet(getDriverIID(iidVPCInfo.IId), subnetInfo.IId)
				if err2 != nil {
					cblog.Error(err2)
					return nil, fmt.Errorf(err.Error() + ", " + err2.Error())
				}
				// (2) for Subnet IID
				cblog.Info("<<ROLLBACK:TRY:VPC-SUBNET-IID>> " + subnetInfo.IId.NameId)
				_, err3 := iidRWLock.DeleteIID(iidm.SUBNETGROUP, connectionName, vpcName, subnetSpiderIId) // vpcName => rsType
				if err3 != nil {
					cblog.Error(err3)
					return nil, fmt.Errorf(err.Error() + ", " + err3.Error())
				}
				return nil, err
			}
		}
	}

	// (3) set ResourceInfo(userIID)
	info.IId = getUserIID(iidVPCInfo.IId)

	// set NameId for SubnetInfo List
	// create new SubnetInfo List
	subnetInfoList := []cres.SubnetInfo{}
	for _, subnetInfo := range info.SubnetInfoList {		
		subnetIIDInfo, err := iidRWLock.GetIIDbySystemID(iidm.SUBNETGROUP, connectionName, vpcName, subnetInfo.IId) // vpcName => rsType
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		if subnetIIDInfo.IId.NameId != "" { // insert only this user created.
			subnetInfo.IId = getUserIID(subnetIIDInfo.IId)
			subnetInfoList = append(subnetInfoList, subnetInfo)
		}
	}
	info.SubnetInfoList = subnetInfoList

	return &info, nil
}


//================ SecurityGroup Handler

// UserIID{UserID, CSP-ID} => SpiderIID{UserID, SP-XID:CSP-ID}
// (0) check VPC existence(VPC UserID)
// (1) check existence(UserID)
// (2) get resource info(CSP-ID)
// (3) create spiderIID: {UserID, SP-XID:CSP-ID}
// (4) insert spiderIID
func RegisterSecurity(connectionName string, vpcUserID string, userIID cres.IID) (*cres.SecurityInfo, error) {
        cblog.Info("call RegisterSecurity()")

	// check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
		cblog.Error(err)
                return nil, err
        }

        vpcUserID, err = EmptyCheckAndTrim("vpcUserID", vpcUserID)
        if err != nil {
		cblog.Error(err)
                return nil, err
        }

        rsType := rsSG

        cldConn, err := ccm.GetCloudConnection(connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

	handler, err := cldConn.CreateSecurityHandler()
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        sgRWLock.Lock()
        defer sgRWLock.Unlock()

        // (0) check VPC existence(VPC UserID)
        bool_ret, err := iidRWLock.IsExistIID(iidm.IIDSGROUP, connectionName, rsVPC, cres.IID{vpcUserID, ""})
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        if bool_ret == false {
                return nil, fmt.Errorf(rsVPC + "-" + vpcUserID + " does not exist!")
        }

        // (1) check existence(UserID)
        iidInfoList, err := getAllSGIIDInfoList(connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        var isExist bool=false
        for _, OneIIdInfo := range iidInfoList {
                if OneIIdInfo.IId.NameId == userIID.NameId {
                        isExist = true
			break
                }
        }

        if isExist == true {
                err :=  fmt.Errorf(rsType + "-" + userIID.NameId + " already exists!")
                cblog.Error(err)
                return nil, err
        }


        // (2) get resource info(CSP-ID)
        // check existence and get info of this resouce in the CSP
	// Do not user NamieId, because Azure driver use it like SystemId
        getInfo, err := handler.GetSecurity( cres.IID{userIID.SystemId, userIID.SystemId} )
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // (3) create spiderIID: {UserID, SP-XID:CSP-ID}
        //     ex) spiderIID {"vpc-01", "vpc-01-9m4e2mr0ui3e8a215n4g:i-0bc7123b7e5cbf79d"}
	// Do not user NamieId, because Azure driver use it like SystemId
	systemId := getMSShortID(getInfo.IId.SystemId)
        spiderIId := cres.IID{userIID.NameId, systemId + ":" + getInfo.IId.SystemId}


        // (4) insert spiderIID
        // insert SecurityGroup SpiderIID to metadb
	_, err = iidRWLock.CreateIID(iidm.SGGROUP, connectionName, vpcUserID, spiderIId)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // set up SecurityGroup User IID for return info
        getInfo.IId = userIID

        // set up VPC UserIID for return info
        iidInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsVPC, cres.IID{vpcUserID, ""})
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        getInfo.VpcIID = getUserIID(iidInfo.IId)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }


        return &getInfo, nil
}

// (1) check exist(NameID)
// (2) generate SP-XID and create reqIID, driverIID
// (3) create Resource
// (4) create spiderIID: {reqNameID, "driverNameID:driverSystemID"}
// (5) insert spiderIID
// (6) create userIID
func CreateSecurity(connectionName string, rsType string, reqInfo cres.SecurityReqInfo) (*cres.SecurityInfo, error) {
	cblog.Info("call CreateSecurity()")

	// check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
		cblog.Error(err)
                return nil, err
        }

	//+++++++++++++++++++++++++++++++++++++++++++
	// set VPC's SystemId
	vpcIIDInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsVPC, reqInfo.VpcIID)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	reqInfo.VpcIID.SystemId = getDriverSystemId(vpcIIDInfo.IId)
	//+++++++++++++++++++++++++++++++++++++++++++

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	handler, err := cldConn.CreateSecurityHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	sgRWLock.Lock()
	defer sgRWLock.Unlock()
	// (1) check exist(NameID)
	iidInfoList, err := getAllSGIIDInfoList(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	var isExist bool=false
	for _, OneIIdInfo := range iidInfoList {
		if OneIIdInfo.IId.NameId == reqInfo.IId.NameId {
			isExist = true
		}
	}

	if isExist == true {
		err :=  fmt.Errorf(rsType + "-" + reqInfo.IId.NameId + " already exists!")
		cblog.Error(err)
		return nil, err
	}

	// if no CIDR in input rules, set default ("0.0.0.0/0")
	for n, _ := range *reqInfo.SecurityRules {
		if (*reqInfo.SecurityRules)[n].CIDR == "" {
			(*reqInfo.SecurityRules)[n].CIDR = "0.0.0.0/0"
		}
	}

	// (2) generate SP-XID and create reqIID, driverIID
	//     ex) SP-XID {"vm-01-9m4e2mr0ui3e8a215n4g"}
	//
	//     create reqIID: {reqNameID, reqSystemID}   # reqSystemID=SP-XID
	//         ex) reqIID {"seoul-service", "vm-01-9m4e2mr0ui3e8a215n4g"} 
	//
	//     create driverIID: {driverNameID, driverSystemID}   # driverNameID=SP-XID, driverSystemID=csp's ID
	//         ex) driverIID {"vm-01-9m4e2mr0ui3e8a215n4g", "i-0bc7123b7e5cbf79d"}
	spUUID, err := iidm.New(connectionName, reqInfo.IId.NameId)
	if err != nil {
                cblog.Error(err)
                return nil, err
        }

	// reqIID
	reqIId := cres.IID{reqInfo.IId.NameId, spUUID}
	// driverIID
	driverIId := cres.IID{spUUID, ""}
	reqInfo.IId = driverIId

	// (3) create Resource
	info, err := handler.CreateSecurity(reqInfo)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// set VPC NameId
	info.VpcIID.NameId = reqInfo.VpcIID.NameId

	// (4) create spiderIID: {reqNameID, "driverNameID:driverSystemID"}
	//     ex) spiderIID {"seoul-service", "vm-01-9m4e2mr0ui3e8a215n4g:i-0bc7123b7e5cbf79d"}
	spiderIId := cres.IID{reqIId.NameId, info.IId.NameId + ":" + info.IId.SystemId}

	// (5) insert spiderIID
	iidInfo, err := iidRWLock.CreateIID(iidm.SGGROUP, connectionName, reqInfo.VpcIID.NameId, spiderIId)  // reqIId.NameId => rsType
	if err != nil {
		cblog.Error(err)
		// rollback
		_, err2 := handler.DeleteSecurity(info.IId)
		if err2 != nil {
			cblog.Error(err2)
			return nil, fmt.Errorf(err.Error() + ", " + err2.Error())
		}
		return nil, err
	}

	// (6) create userIID: {reqNameID, driverSystemID}
	//     ex) userIID {"seoul-service", "i-0bc7123b7e5cbf79d"}
	info.IId = getUserIID(iidInfo.IId)

	// set VPC SystemId
	info.VpcIID.SystemId = getDriverSystemId(vpcIIDInfo.IId)

	return &info, nil
}

// (1) get IID:list
// (2) get CSP:list
// (3) filtering CSP-list by IID-list
// (4) set userIID
func ListSecurity(connectionName string, rsType string) ([]*cres.SecurityInfo, error) {
	cblog.Info("call ListSecurity()")

	// check empty and trim user inputs
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

	handler, err := cldConn.CreateSecurityHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	sgRWLock.RLock()
	defer sgRWLock.RUnlock()

	// (1) get IID:list
	iidInfoList, err := getAllSGIIDInfoList(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	var infoList []*cres.SecurityInfo
	if iidInfoList == nil || len(iidInfoList) <= 0 {
		infoList = []*cres.SecurityInfo{}
		return infoList, nil
	}

	// (2) get CSP:list
	infoList, err = handler.ListSecurity()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	if infoList == nil { // if iidInfoList not null, then infoList has any list.
		return nil, fmt.Errorf("<IID-CSP mismatch> " + rsType + " IID List has " + strconv.Itoa(len(iidInfoList)) + ", but " + connectionName + " Resource list has nothing!")
	}

	// (3) filtering CSP-list by IID-list
	infoList2 := []*cres.SecurityInfo{}
	for _, iidInfo := range iidInfoList {
		exist := false
		for _, info := range infoList {
			if getDriverSystemId(iidInfo.IId) == info.IId.SystemId {

				// (4) set userIID
				info.IId = getUserIID(iidInfo.IId)

				// set VPC SystemId
				vpcIIDInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsVPC, cres.IID{iidInfo.ResourceType /*vpcName*/, ""})
				if err != nil {
					cblog.Error(err)
					return nil, err
				}
				// set userIID
				info.VpcIID = getUserIID(vpcIIDInfo.IId)

				infoList2 = append(infoList2, info)
				exist = true
			}
		}
		if exist == false {
			cblog.Info("<IID-CSP mismatch> " + rsType + "-" + iidInfo.IId.NameId + ":" + iidInfo.IId.SystemId + " exsits. but " + connectionName + " does not have!")
			//return nil, fmt.Errorf("<IID-CSP mismatch> " + rsType + "-" + iidInfo.IId.NameId + ":" + iidInfo.IId.SystemId + " exsits. but " + connectionName + " does not have!")
		}
	}

	return infoList2, nil
}

// Get All IID:list of SecurityGroup
// (1) Get VPC's Name List
// (2) Create All SG's IIDInfo List
func getAllSGIIDInfoList(connectionName string) ([]*iidm.IIDInfo, error) {

        // (1) Get VPC's Name List
        // format) /resource-info-spaces/{iidGroup}/{connectionName}/{resourceType}/{resourceName} [{resourceID}]
        vpcNameList, err := iidRWLock.ListResourceType(iidm.SGGROUP, connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
	vpcNameList = uniqueNameList(vpcNameList)
        // (2) Create All SG's IIDInfo List
        iidInfoList := []*iidm.IIDInfo{}
        for _, vpcName := range vpcNameList {
                iidInfoListForOneVPC, err := iidRWLock.ListIID(iidm.SGGROUP, connectionName, vpcName)
                if err != nil {
                        cblog.Error(err)
                        return nil, err
                }
                iidInfoList = append(iidInfoList, iidInfoListForOneVPC...)
        }
        return iidInfoList, nil
}

func uniqueNameList(vpcNameList []string) []string {
    keys := make(map[string]bool)
    list := []string{}	
    for _, entry := range vpcNameList {
        if _, value := keys[entry]; !value {
            keys[entry] = true
            list = append(list, entry)
        }
    }    
    return list
}

// (1) get IID(NameId)
// (2) get resource(SystemId)
// (3) set ResourceInfo(IID.NameId)
func GetSecurity(connectionName string, rsType string, nameID string) (*cres.SecurityInfo, error) {
	cblog.Info("call GetSecurity()")

	// check empty and trim user inputs
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

	handler, err := cldConn.CreateSecurityHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	sgRWLock.RLock()
	defer sgRWLock.RUnlock()
	// (1) get IID(NameId)
	iidInfoList, err := getAllSGIIDInfoList(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	var iidInfo *iidm.IIDInfo
	var bool_ret = false
	for _, OneIIdInfo := range iidInfoList {
		if OneIIdInfo.IId.NameId == nameID {
			iidInfo = OneIIdInfo
			bool_ret = true
			break;
		}
	}
	if bool_ret == false {
                return nil, fmt.Errorf(rsType + "-" + nameID + " does not exist!")
        }

	// (2) get resource(SystemId)
	info, err := handler.GetSecurity(getDriverIID(iidInfo.IId))
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (3) set ResourceInfo(IID.NameId)
	// set ResourceInfo
	info.IId = getUserIID(iidInfo.IId)

	// set VPC SystemId
	vpcIIDInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsVPC, cres.IID{iidInfo.ResourceType/*vpcName*/, ""})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	info.VpcIID = getUserIID(vpcIIDInfo.IId)

	return &info, nil
}

//================ KeyPair Handler

// UserIID{UserID, CSP-ID} => SpiderIID{UserID, SP-XID:CSP-ID}
// (1) check existence(UserID)
// (2) get resource info(CSP-ID)
// (3) create spiderIID: {UserID, SP-XID:CSP-ID}
// (4) insert spiderIID
func RegisterKey(connectionName string, userIID cres.IID) (*cres.KeyPairInfo, error) {
        cblog.Info("call RegisterKey()")

	// check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
		cblog.Error(err)
                return nil, err
        }

        rsType := rsKey

        cldConn, err := ccm.GetCloudConnection(connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        handler, err := cldConn.CreateKeyPairHandler()
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        keyRWLock.Lock()
        defer keyRWLock.Unlock()

        // (1) check existence(UserID)
	bool_ret, err := iidRWLock.IsExistIID(iidm.IIDSGROUP, connectionName, rsType, userIID)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        if bool_ret == true {
                return nil, fmt.Errorf(rsType + "-" + userIID.NameId + " already exists!")
        }

        // (2) get resource info(CSP-ID)
        // check existence and get info of this resouce in the CSP
	// Do not user NamieId, because Azure driver use it like SystemId
        getInfo, err := handler.GetKey( cres.IID{userIID.SystemId, userIID.SystemId} )
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // (3) create spiderIID: {UserID, SP-XID:CSP-ID}
        //     ex) spiderIID {"vpc-01", "vpc-01-9m4e2mr0ui3e8a215n4g:i-0bc7123b7e5cbf79d"}
	// Do not user NamieId, because Azure driver use it like SystemId
	systemId := getMSShortID(getInfo.IId.SystemId)
        spiderIId := cres.IID{userIID.NameId, systemId + ":" + getInfo.IId.SystemId}

        // (4) insert spiderIID
        // insert KeyPair SpiderIID to metadb
        _, err = iidRWLock.CreateIID(iidm.IIDSGROUP, connectionName, rsType, spiderIId)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // set up KeyPair User IID for return info
        getInfo.IId = userIID

        return &getInfo, nil
}


// (1) check exist(NameID)
// (2) generate SP-XID and create reqIID, driverIID
// (3) create Resource
// (4) create spiderIID: {reqNameID, "driverNameID:driverSystemID"}
// (5) insert spiderIID
// (6) create userIID
func CreateKey(connectionName string, rsType string, reqInfo cres.KeyPairReqInfo) (*cres.KeyPairInfo, error) {
	cblog.Info("call CreateKey()")

	// check empty and trim user inputs
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

	handler, err := cldConn.CreateKeyPairHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	keyRWLock.Lock()
	defer keyRWLock.Unlock()
	// (1) check exist(NameID)
	bool_ret, err := iidRWLock.IsExistIID(iidm.IIDSGROUP, connectionName, rsType, reqInfo.IId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	if bool_ret == true {
		return nil, fmt.Errorf(reqInfo.IId.NameId + " already exists!")
	}

	// (2) generate SP-XID and create reqIID, driverIID
	//     ex) SP-XID {"vm-01-9m4e2mr0ui3e8a215n4g"}
	//
	//     create reqIID: {reqNameID, reqSystemID}   # reqSystemID=SP-XID
	//         ex) reqIID {"seoul-service", "vm-01-9m4e2mr0ui3e8a215n4g"} 
	//
	//     create driverIID: {driverNameID, driverSystemID}   # driverNameID=SP-XID, driverSystemID=csp's ID
	//         ex) driverIID {"vm-01-9m4e2mr0ui3e8a215n4g", "i-0bc7123b7e5cbf79d"}
	spUUID, err := iidm.New(connectionName, reqInfo.IId.NameId)
	if err != nil {
                cblog.Error(err)
                return nil, err
        }

	// reqIID
	reqIId := cres.IID{reqInfo.IId.NameId, spUUID}
	// driverIID
	driverIId := cres.IID{spUUID, ""}
	reqInfo.IId = driverIId

	// (3) create Resource
	info, err := handler.CreateKey(reqInfo)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (4) create spiderIID: {reqNameID, "driverNameID:driverSystemID"}
	//     ex) spiderIID {"seoul-service", "vm-01-9m4e2mr0ui3e8a215n4g:i-0bc7123b7e5cbf79d"}
	spiderIId := cres.IID{reqIId.NameId, info.IId.NameId + ":" + info.IId.SystemId}

	// (5) insert spiderIID
	iidInfo, err := iidRWLock.CreateIID(iidm.IIDSGROUP, connectionName, rsType, spiderIId)
	if err != nil {
		cblog.Error(err)
		// rollback
		_, err2 := handler.DeleteKey(info.IId)
		if err2 != nil {
			cblog.Error(err2)
			return nil, fmt.Errorf(err.Error() + ", " + err2.Error())
		}
		return nil, err
	}

	// (6) create userIID: {reqNameID, driverSystemID}
	//     ex) userIID {"seoul-service", "i-0bc7123b7e5cbf79d"}
	info.IId = getUserIID(iidInfo.IId)

	return &info, nil
}

// (1) get IID:list
// (2) get CSP:list
// (3) filtering CSP-list by IID-list
// (4) set userIID
func ListKey(connectionName string, rsType string) ([]*cres.KeyPairInfo, error) {
	cblog.Info("call ListKey()")

	// check empty and trim user inputs
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

	handler, err := cldConn.CreateKeyPairHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	keyRWLock.RLock()
	defer keyRWLock.RUnlock()
	// (1) get IID:list
	iidInfoList, err := iidRWLock.ListIID(iidm.IIDSGROUP, connectionName, rsType)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	var infoList []*cres.KeyPairInfo
	if iidInfoList == nil || len(iidInfoList) <= 0 {
		infoList = []*cres.KeyPairInfo{}
		return infoList, nil
	}

	// (2) get CSP:list
	infoList, err = handler.ListKey()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	if infoList == nil { // if iidInfoList not null, then infoList has any list.
		return nil, fmt.Errorf("<IID-CSP mismatch> " + rsType + " IID List has " + strconv.Itoa(len(iidInfoList)) + ", but " + connectionName + " Resource list has nothing!")
	}

	// (3) filtering CSP-list by IID-list
	infoList2 := []*cres.KeyPairInfo{}
	for _, iidInfo := range iidInfoList {
		exist := false
		for _, info := range infoList {
			if getDriverSystemId(iidInfo.IId) == info.IId.SystemId {
				// (4) set userIID
				info.IId = getUserIID(iidInfo.IId)
				infoList2 = append(infoList2, info)
				exist = true
			}
		}
		if exist == false {
			cblog.Info("<IID-CSP mismatch> " + rsType + "-" + iidInfo.IId.NameId + ":" + iidInfo.IId.SystemId + " exsits. but " + connectionName + " does not have!")
			//return nil, fmt.Errorf("<IID-CSP mismatch> " + rsType + "-" + iidInfo.IId.NameId + ":" + iidInfo.IId.SystemId + " exsits. but " + connectionName + " does not have!")
		}
	}

	return infoList2, nil
}

// (1) get IID(NameId)
// (2) get resource(SystemId)
// (3) set ResourceInfo(IID.NameId)
func GetKey(connectionName string, rsType string, nameID string) (*cres.KeyPairInfo, error) {
	cblog.Info("call GetKey()")

	// check empty and trim user inputs
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

	handler, err := cldConn.CreateKeyPairHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	keyRWLock.RLock()
	defer keyRWLock.RUnlock()
	// (1) get IID(NameId)
	iidInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsType, cres.IID{nameID, ""})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (2) get resource(SystemId)
	info, err := handler.GetKey(getDriverIID(iidInfo.IId))
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (3) set ResourceInfo(IID.NameId)
	info.IId.NameId = iidInfo.IId.NameId

	return &info, nil
}

func cloneReqInfoWithDriverIID(ConnectionName string, reqInfo cres.VMReqInfo) (cres.VMReqInfo, error) {

	newReqInfo := cres.VMReqInfo {
		IId:       cres.IID{reqInfo.IId.NameId, reqInfo.IId.SystemId},

		// set Image SystemId
		ImageIID:         cres.IID{reqInfo.ImageIID.NameId, reqInfo.ImageIID.NameId},
		//VpcIID:           cres.IID{reqInfo.VpcIID.NameId, reqInfo.VpcIID.SystemId},
		//SubnetIID:        cres.IID{reqInfo.SubnetIID.NameId, reqInfo.SubnetIID.SystemId},
		//SecurityGroupIIDs: getSecurityGroupIIDs(),

		VMSpecName:       reqInfo.VMSpecName,
		//KeyPairIID:       cres.IID{reqInfo.KeyPairIID.NameId, reqInfo.KeyPairIID.SystemId},

		VMUserId:         reqInfo.VMUserId,
		VMUserPasswd:	  reqInfo.VMUserPasswd,
	}

	// set VPC SystemId
	if reqInfo.VpcIID.NameId != "" {
		// get spiderIID
		IIdInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, ConnectionName, rsVPC, reqInfo.VpcIID)
		if err != nil {
			cblog.Error(err)
			return cres.VMReqInfo{}, err
		}
		// set driverIID
		newReqInfo.VpcIID = getDriverIID(IIdInfo.IId)
	}

	// set Subnet SystemId
	if reqInfo.SubnetIID.NameId != "" {		
		IIdInfo, err := iidRWLock.GetIID(iidm.SUBNETGROUP, ConnectionName, reqInfo.VpcIID.NameId, reqInfo.SubnetIID) // reqInfo.VpcIID.NameId => rsType
		if err != nil {
			cblog.Error(err)
			return cres.VMReqInfo{}, err
		}
		// set driverIID
		newReqInfo.SubnetIID = getDriverIID(IIdInfo.IId)
	}

	// set SecurityGroups SystemId
	for _, sgIID := range reqInfo.SecurityGroupIIDs {
		IIdInfo, err := iidRWLock.GetIID(iidm.SGGROUP, ConnectionName, reqInfo.VpcIID.NameId, sgIID)  // reqInfo.VpcIID.NameId => rsType
		if err != nil {
			cblog.Error(err)
			return cres.VMReqInfo{}, err
		}
		// set driverIID
		newReqInfo.SecurityGroupIIDs = append(newReqInfo.SecurityGroupIIDs, getDriverIID(IIdInfo.IId))
	}

	// set KeyPair SystemId
	if reqInfo.KeyPairIID.NameId != "" {
		IIdInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, ConnectionName, rsKey, reqInfo.KeyPairIID)
		if err != nil {
			cblog.Error(err)
			return cres.VMReqInfo{}, err
		}
		newReqInfo.KeyPairIID = getDriverIID(IIdInfo.IId)
	}

	return newReqInfo, nil
}

//================ VM Handler

// UserIID{UserID, CSP-ID} => SpiderIID{UserID, SP-XID:CSP-ID}
// (1) check existence(UserID)
// (2) get resource info(CSP-ID)
// (3) create spiderIID: {UserID, SP-XID:CSP-ID}
// (4) insert spiderIID
func RegisterVM(connectionName string, userIID cres.IID) (*cres.VMInfo, error) {
        cblog.Info("call RegisterVM()")

	// check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
		cblog.Error(err)
                return nil, err
        }

        rsType := rsVM

        cldConn, err := ccm.GetCloudConnection(connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        handler, err := cldConn.CreateVMHandler()
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        //vmRWLock.Lock()
        //defer vmRWLock.Unlock()

        // (1) check existence(UserID)
	bool_ret, err := iidRWLock.IsExistIID(iidm.IIDSGROUP, connectionName, rsType, userIID)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        if bool_ret == true {
                return nil, fmt.Errorf(rsType + "-" + userIID.NameId + " already exists!")
        }

        // (2) get resource info(CSP-ID)
        // check existence and get info of this resouce in the CSP
	// Do not user NamieId, because Azure driver use it like SystemId
        getInfo, err := handler.GetVM( cres.IID{userIID.SystemId, userIID.SystemId} )
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

	// check and set ID
	err = getSetNameId(connectionName, &getInfo)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // current: Assume 22 port, except Cloud-Twin, by powerkim, 2021.03.24.
        if getInfo.SSHAccessPoint == "" {
                getInfo.SSHAccessPoint = getInfo.PublicIP + ":22"
        }


        // (3) create spiderIID: {UserID, SP-XID:CSP-ID}
        //     ex) spiderIID {"vpc-01", "vpc-01-9m4e2mr0ui3e8a215n4g:i-0bc7123b7e5cbf79d"}
	// Do not user NamieId, because Azure driver use it like SystemId
	systemId := getMSShortID(getInfo.IId.SystemId)
        spiderIId := cres.IID{userIID.NameId, systemId + ":" + getInfo.IId.SystemId}

        // (4) insert spiderIID
        // insert VM SpiderIID to metadb
        _, err = iidRWLock.CreateIID(iidm.IIDSGROUP, connectionName, rsType, spiderIId)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // set up VM User IID for return info
        getInfo.IId = userIID


        return &getInfo, nil
}

// (1) check exist(NameID)
// (2) generate SP-XID and create reqIID, driverIID
// (3) clone the reqInfo with DriverIID
// (4) create Resource
// (5) create spiderIID: {reqNameID, "driverNameID:driverSystemID"}
// (6) insert spiderIID
// (7) create userIID
func StartVM(connectionName string, rsType string, reqInfo cres.VMReqInfo) (*cres.VMInfo, error) {
	cblog.Info("call StartVM()")

	// check empty and trim user inputs
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

	handler, err := cldConn.CreateVMHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// vmRWLock.Lock() @todo undo this until supporting async call. by powerkim, 2020.05.10
	// defer vmRWLock.Unlock() @todo undo this until supporting async call. by powerkim, 2020.05.10
	// (1) check exist(NameID)
	bool_ret, err := iidRWLock.IsExistIID(iidm.IIDSGROUP, connectionName, rsType, reqInfo.IId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	if bool_ret == true {
		return nil, fmt.Errorf(rsType + "-" + reqInfo.IId.NameId + " already exists!")
	}

	providerName, err := ccm.GetProviderNameByConnectionName(connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        regionName, zoneName, err := ccm.GetRegionNameByConnectionName(connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

	// (2) generate SP-XID and create reqIID, driverIID
	//     ex) SP-XID {"vm-01-9m4e2mr0ui3e8a215n4g"}
	//
	//     create reqIID: {reqNameID, reqSystemID}   # reqSystemID=SP-XID
	//         ex) reqIID {"seoul-service", "vm-01-9m4e2mr0ui3e8a215n4g"} 
	//
	//     create driverIID: {driverNameID, driverSystemID}   # driverNameID=SP-XID, driverSystemID=csp's ID
	//         ex) driverIID {"vm-01-9m4e2mr0ui3e8a215n4g", "i-0bc7123b7e5cbf79d"}
	spUUID, err := iidm.New(connectionName, reqInfo.IId.NameId)
	if err != nil {
                cblog.Error(err)
                return nil, err
        }

	// reqIID
	reqIId := cres.IID{reqInfo.IId.NameId, spUUID}

	// driverIID
	driverIId := cres.IID{spUUID, ""}
	reqInfo.IId = driverIId

	// (3) clone the reqInfo with DriverIID
	reqInfoForDriver, err := cloneReqInfoWithDriverIID(connectionName, reqInfo)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	callInfo := call.CLOUDLOGSCHEMA {
                CloudOS: call.CLOUD_OS(providerName),
                RegionZone: regionName + "/" + zoneName,
                ResourceType: call.VM,
                ResourceName: reqInfo.IId.NameId,
		CloudOSAPI: "CB-Spider:StartVM()",
                ElapsedTime: "",
                ErrorMSG: "",
        }
        start := call.Start()

	// (4) create Resource
	info, err := handler.StartVM(reqInfoForDriver)
	callInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblog.Error(err)
		callInfo.ErrorMSG = err.Error()
		return nil, err
	}
	callogger.Info(call.String(callInfo))


	// (5) create spiderIID: {reqNameID, "driverNameID:driverSystemID"}
	//     ex) spiderIID {"seoul-service", "vm-01-9m4e2mr0ui3e8a215n4g:i-0bc7123b7e5cbf79d"}
	spiderIId := cres.IID{reqIId.NameId, info.IId.NameId + ":" + info.IId.SystemId}



	// (6) insert spiderIID
	iidInfo, err := iidRWLock.CreateIID(iidm.IIDSGROUP, connectionName, rsType, spiderIId)
	if err != nil {
		cblog.Error(err)
		// rollback
		_, err2 := handler.TerminateVM(iidInfo.IId) // @todo check validation
		if err2 != nil {
			cblog.Error(err2)
			return nil, fmt.Errorf(err.Error() + ", " + err2.Error())
		}
		return nil, err
	}

/*
	// set sg NameId from VPCNameId-SecurityGroupNameId
	// IID.NameID format => {VPC NameID} + SG_DELIMITER + {SG NameID}
	for i, sgIID := range info.SecurityGroupIIds {
		vpc_sg_nameid := strings.Split(sgIID.NameId, SG_DELIMITER)
		info.SecurityGroupIIds[i].NameId = vpc_sg_nameid[1]
	}
*/
	// (7) create userIID: {reqNameID, driverSystemID}
	//     ex) userIID {"seoul-service", "i-0bc7123b7e5cbf79d"}
	info.IId = getUserIID(iidInfo.IId)

	// set NameId for info by reqInfo
	setNameId(connectionName, &info, &reqInfo)

	// current: Assume 22 port, except Cloud-Twin, by powerkim, 2021.03.24.
	if info.SSHAccessPoint == "" {
		info.SSHAccessPoint = info.PublicIP + ":22"
	}

	return &info, nil
}

func setNameId(ConnectionName string, vmInfo *cres.VMInfo, reqInfo *cres.VMReqInfo) error {

	// set Image SystemId
	// @todo before Image Handling by powerkim
	if reqInfo.ImageIID.NameId != "" {
		vmInfo.ImageIId.NameId = reqInfo.ImageIID.NameId
	}

	// set VPC NameId
	if reqInfo.VpcIID.NameId != "" {
		vmInfo.VpcIID.NameId = reqInfo.VpcIID.NameId
	}

	// set Subnet NameId
	if reqInfo.SubnetIID.NameId != "" {
		vmInfo.SubnetIID.NameId = reqInfo.SubnetIID.NameId
	}

	// set SecurityGroups NameId
	for i, sgIID := range vmInfo.SecurityGroupIIds {
		IIdInfo, err := iidRWLock.GetIIDbySystemID(iidm.SGGROUP, ConnectionName, reqInfo.VpcIID.NameId, sgIID)
		if err != nil {
			cblog.Error(err)
			return err
		}
		vmInfo.SecurityGroupIIds[i].NameId = IIdInfo.IId.NameId
	}

	if reqInfo.KeyPairIID.NameId != "" {
		// set KeyPair SystemId
		vmInfo.KeyPairIId.NameId = reqInfo.KeyPairIID.NameId
	}

	return nil
}

// (1) get IID:list
// (2) get CSP:list
// (3) filtering CSP-list by IID-list
func ListVM(connectionName string, rsType string) ([]*cres.VMInfo, error) {
	cblog.Info("call ListVM()")

	// check empty and trim user inputs
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

	handler, err := cldConn.CreateVMHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	//vmRWLock.RLock()
	//defer vmRWLock.RUnlock()
	// (1) get IID:list
	iidInfoList, err := iidRWLock.ListIID(iidm.IIDSGROUP, connectionName, rsType)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	var infoList []*cres.VMInfo
	if iidInfoList == nil || len(iidInfoList) <= 0 {
		infoList = []*cres.VMInfo{}
		return infoList, nil
	}

	// (2) get CSP:list
	infoList, err = handler.ListVM()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	if infoList == nil { // if iidInfoList not null, then infoList has any list.
		return nil, fmt.Errorf("<IID-CSP mismatch> " + rsType + " IID List has " + strconv.Itoa(len(iidInfoList)) + ", but " + connectionName + " Resource list has nothing!")
	}

	// (3) filtering CSP-list by IID-list
	infoList2 := []*cres.VMInfo{}
	for _, iidInfo := range iidInfoList {
		exist := false
		for _, info := range infoList {
			if getDriverSystemId(iidInfo.IId) == info.IId.SystemId {

				// set ResourceInfo
				info.IId = getUserIID(iidInfo.IId)

				err = getSetNameId(connectionName, info)
				if err != nil {
					cblog.Error(err)
					return nil, err
				}
/*
				// set sg NameId from VPCNameId-SecurityGroupNameId
				// IID.NameID format => {VPC NameID} + SG_DELIMITER + {SG NameID}
				for i, sgIID := range info.SecurityGroupIIds {
					vpc_sg_nameid := strings.Split(sgIID.NameId, SG_DELIMITER)
					info.SecurityGroupIIds[i].NameId = vpc_sg_nameid[1]
				}
*/
				// current: Assume 22 port, except Cloud-Twin, by powerkim, 2021.03.24.
				if info.SSHAccessPoint == "" {
					info.SSHAccessPoint = info.PublicIP + ":22"
				}

				infoList2 = append(infoList2, info)
				exist = true
			}
		}
		if exist == false {
			cblog.Info("<IID-CSP mismatch> " + rsType + "-" + iidInfo.IId.NameId + ":" + iidInfo.IId.SystemId + " exsits. but " + connectionName + " does not have!")
			//return nil, fmt.Errorf("<IID-CSP mismatch> " + rsType + "-" + iidInfo.IId.NameId + ":" + iidInfo.IId.SystemId + " exsits. but " + connectionName + " does not have!")
		}
	}

	return infoList2, nil
}

func getSetNameId(ConnectionName string, vmInfo *cres.VMInfo) error {

	// set Image NameId
	// @todo before Image Handling by powerkim
	//vmInfo.ImageIId.NameId = vmInfo.ImageIId.SystemId

	if vmInfo.VpcIID.SystemId != "" {
		// set VPC NameId
		IIdInfo, err := iidRWLock.GetIIDbySystemID(iidm.IIDSGROUP, ConnectionName, rsVPC, vmInfo.VpcIID)
		if err != nil {
			cblog.Error(err)
			return err
		}
		vmInfo.VpcIID.NameId = IIdInfo.IId.NameId
	}

	if vmInfo.SubnetIID.SystemId != "" {
		// set Subnet NameId
		IIdInfo, err := iidRWLock.GetIIDbySystemID(iidm.SUBNETGROUP, ConnectionName, vmInfo.VpcIID.NameId, vmInfo.SubnetIID)  // reqInfo.VpcIID.NameId => rsType
		if err != nil {
			cblog.Error(err)
			return err
		}
		vmInfo.SubnetIID.NameId = IIdInfo.IId.NameId
	}

	// set SecurityGroups NameId
	for i, sgIID := range vmInfo.SecurityGroupIIds {
		IIdInfo, err := iidRWLock.GetIIDbySystemID(iidm.SGGROUP, ConnectionName, vmInfo.VpcIID.NameId, sgIID)  // reqInfo.VpcIID.NameId => rsType
		if err != nil {
			cblog.Error(err)
			return err
		}
		vmInfo.SecurityGroupIIds[i].NameId = IIdInfo.IId.NameId
	}

	if vmInfo.KeyPairIId.SystemId != "" {
		// set KeyPair NameId
		IIdInfo, err := iidRWLock.GetIIDbySystemID(iidm.IIDSGROUP, ConnectionName, rsKey, vmInfo.KeyPairIId)
		if err != nil {
			cblog.Error(err)
			return err
		}
		vmInfo.KeyPairIId.NameId = IIdInfo.IId.NameId
	}

	return nil
}

// (1) get IID(NameId)
// (2) get resource(SystemId)
// (3) set ResourceInfo(IID.NameId)
func GetVM(connectionName string, rsType string, nameID string) (*cres.VMInfo, error) {
	cblog.Info("call GetVM()")

	// check empty and trim user inputs
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

	handler, err := cldConn.CreateVMHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	//vmRWLock.RLock()
	//defer vmRWLock.RUnlock()
	// (1) get IID(NameId)
	iidInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsType, cres.IID{nameID, ""})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (2) get resource(SystemId)
	info, err := handler.GetVM(getDriverIID(iidInfo.IId))
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (3) set ResourceInfo(IID.NameId)
	// set ResourceInfo
	info.IId = getUserIID(iidInfo.IId)

	err = getSetNameId(connectionName, &info)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
/*
	// set sg NameId from VPCNameId-SecurityGroupNameId
	// IID.NameID format => {VPC NameID} + SG_DELIMITER + {SG NameID}
	for i, sgIID := range info.SecurityGroupIIds {
		vpc_sg_nameid := strings.Split(sgIID.NameId, SG_DELIMITER)
		info.SecurityGroupIIds[i].NameId = vpc_sg_nameid[1]
	}
*/
	// current: Assume 22 port, except Cloud-Twin, by powerkim, 2021.03.24.
        if info.SSHAccessPoint == "" {
                info.SSHAccessPoint = info.PublicIP + ":22"
        }

	return &info, nil
}

// (1) get IID:list
// (2) get CSP:VMStatuslist
// (3) filtering CSP-VMStatuslist by IID-list
func ListVMStatus(connectionName string, rsType string) ([]*cres.VMStatusInfo, error) {
	cblog.Info("call ListVMStatus()")

	// check empty and trim user inputs
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

	handler, err := cldConn.CreateVMHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	//vmRWLock.RLock()
	//defer vmRWLock.RUnlock()
	// (1) get IID:list
	iidInfoList, err := iidRWLock.ListIID(iidm.IIDSGROUP, connectionName, rsType)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	var infoList []*cres.VMStatusInfo
	if iidInfoList == nil || len(iidInfoList) <= 0 {
		infoList = []*cres.VMStatusInfo{}
		return infoList, nil
	}

	// (2) get CSP:VMStatuslist
	infoList, err = handler.ListVMStatus()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	if infoList == nil { // if iidInfoList not null, then infoList has any list.
		return nil, fmt.Errorf("<IID-CSP mismatch> " + rsType + " IID List has " + strconv.Itoa(len(iidInfoList)) + ", but " + connectionName + " Resource list has nothing!")
	}

	// (3) filtering CSP-VMStatuslist by IID-list
	infoList2 := []*cres.VMStatusInfo{}
	for _, iidInfo := range iidInfoList {
		exist := false
		for _, info := range infoList {
			if getDriverSystemId(iidInfo.IId) == info.IId.SystemId {
				// set ResourceInfo
				info.IId = getUserIID(iidInfo.IId)
				infoList2 = append(infoList2, info)
				exist = true
			}
		}
		if exist == false {
			cblog.Info("<IID-CSP mismatch> " + rsType + "-" + iidInfo.IId.NameId + ":" + iidInfo.IId.SystemId + " exsits. but " + connectionName + " does not have!")
			//return nil, fmt.Errorf("<IID-CSP mismatch> " + rsType + "-" + iidInfo.IId.NameId + ":" + iidInfo.IId.SystemId + " exsits. but " + connectionName + " does not have!")
		}
	}

	return infoList2, nil
}

// (1) get IID(NameId)
// (2) get CSP:VMStatus(SystemId)
func GetVMStatus(connectionName string, rsType string, nameID string) (cres.VMStatus, error) {
	cblog.Info("call GetVMStatus()")

	// check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
		cblog.Error(err)
                return "", err
        }

        nameID, err = EmptyCheckAndTrim("nameID", nameID)
        if err != nil {
		cblog.Error(err)
                return "", err
        }

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	handler, err := cldConn.CreateVMHandler()
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	//vmRWLock.RLock()
	//defer vmRWLock.RUnlock()
	// (1) get IID(NameId)
	iidInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsType, cres.IID{nameID, ""})
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	// (2) get CSP:VMStatus(SystemId)
	info, err := handler.GetVMStatus(getDriverIID(iidInfo.IId)) // type of info => string
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	return info, nil
}

// (1) get IID(NameId)
// (2) control CSP:VM(SystemId)
func ControlVM(connectionName string, rsType string, nameID string, action string) (cres.VMStatus, error) {
	cblog.Info("call ControlVM()")

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	handler, err := cldConn.CreateVMHandler()
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	//vmRWLock.RLock()
	//defer vmRWLock.RUnlock()
	// (1) get IID(NameId)
	iidInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsType, cres.IID{nameID, ""})
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	// (2) control CSP:VM(SystemId)
	vmIID := getDriverIID(iidInfo.IId)

	var info cres.VMStatus

	switch strings.ToLower(action) {
	case "suspend":
		info, err = handler.SuspendVM(vmIID)
	case "resume":
		info, err = handler.ResumeVM(vmIID)
	case "reboot":
		info, err = handler.RebootVM(vmIID)
	default:
		return "", fmt.Errorf(action + " is not a valid action!!")

	}
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	return info, nil
}

// list all Resources for management
// (1) get IID:list
// (2) get CSP:list
// (3) filtering CSP-list by IID-list
// (4) make MappedList, OnlySpiderList, OnlyCSPList
func ListAllResource(connectionName string, rsType string) (AllResourceList, error) {
	cblog.Info("call ListAllResource()")

	// check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
                return AllResourceList{}, err
		cblog.Error(err)
        }

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		return AllResourceList{}, err
	}

	var handler interface{}

	switch rsType {
	case rsVPC:
		handler, err = cldConn.CreateVPCHandler()
	case rsSG:
		handler, err = cldConn.CreateSecurityHandler()
	case rsKey:
		handler, err = cldConn.CreateKeyPairHandler()
	case rsVM:
		handler, err = cldConn.CreateVMHandler()
	default:
		return AllResourceList{}, fmt.Errorf(rsType + " is not supported Resource!!")
	}
	if err != nil {
		return AllResourceList{}, err
	}

	var allResList AllResourceList

	switch rsType {
	case rsVPC:
		vpcRWLock.RLock()
		defer vpcRWLock.RUnlock()
	case rsSG:
		sgRWLock.RLock()
		defer sgRWLock.RUnlock()
	case rsKey:
		keyRWLock.RLock()
		defer keyRWLock.RUnlock()
	case rsVM:
	//	vmRWLock.RLock()
	//	defer vmRWLock.RUnlock()
	default:
		return AllResourceList{}, fmt.Errorf(rsType + " is not supported Resource!!")
	}

	// (1) get IID:list
	iidInfoList := []*iidm.IIDInfo{}
	switch rsType {
	case rsSG:
		iidInfoList, err = getAllSGIIDInfoList(connectionName)
		if err != nil {
			cblog.Error(err)
			return AllResourceList{}, err
		}

	default:
		iidInfoList, err = iidRWLock.ListIID(iidm.IIDSGROUP, connectionName, rsType)
		if err != nil {
			cblog.Error(err)
			return AllResourceList{}, err
		}
	}

	// if iidInfoList is empty, OnlySpiderList is empty.
	if iidInfoList == nil || len(iidInfoList) <= 0 {
		emptyIIDInfoList := []*cres.IID{}
		allResList.AllList.MappedList = emptyIIDInfoList
		allResList.AllList.OnlySpiderList = emptyIIDInfoList
	}

	// (2) get CSP:list
	iidCSPList := []*cres.IID{}
	switch rsType {
	case rsVPC:
		infoList, err := handler.(cres.VPCHandler).ListVPC()
		if err != nil {
			cblog.Error(err)
			return AllResourceList{}, err
		}
		if infoList != nil {
			for _, info := range infoList {
				iidCSPList = append(iidCSPList, &info.IId)
			}
		}
	case rsSG:
		infoList, err := handler.(cres.SecurityHandler).ListSecurity()
		if err != nil {
			cblog.Error(err)
			return AllResourceList{}, err
		}
		if infoList != nil {
			for _, info := range infoList {
				iidCSPList = append(iidCSPList, &info.IId)
			}
		}
	case rsKey:
		infoList, err := handler.(cres.KeyPairHandler).ListKey()
		if err != nil {
			cblog.Error(err)
			return AllResourceList{}, err
		}
		if infoList != nil {
			for _, info := range infoList {
				iidCSPList = append(iidCSPList, &info.IId)
			}
		}
	case rsVM:
		infoList, err := handler.(cres.VMHandler).ListVM()
		if err != nil {
			cblog.Error(err)
			return AllResourceList{}, err
		}
		if infoList != nil {
			for _, info := range infoList {
				iidCSPList = append(iidCSPList, &info.IId)
			}
		}
	default:
		return AllResourceList{}, fmt.Errorf(rsType + " is not supported Resource!!")
	}

	if iidCSPList == nil || len(iidCSPList) <= 0 {
		// if iidCSPList is empty, iidInfoList is empty => all list is empty <-------------- (1)
		if iidInfoList == nil || len(iidInfoList) <= 0 {
			emptyIIDInfoList := []*cres.IID{}
			allResList.AllList.OnlyCSPList = emptyIIDInfoList

			return allResList, nil
		} else { // iidCSPList is empty and iidInfoList has value => only OnlySpiderList <--(2)
			emptyIIDInfoList := []*cres.IID{}
			allResList.AllList.MappedList = emptyIIDInfoList
			allResList.AllList.OnlyCSPList = emptyIIDInfoList
			allResList.AllList.OnlySpiderList = getUserIIDList(iidInfoList)

			return allResList, nil
		}
	}

	// iidInfoList is empty, iidCSPList has values => only OnlyCSPList <--------------------------(3)
	if iidInfoList == nil || len(iidInfoList) <= 0 {
		OnlyCSPList := []*cres.IID{}
		for _, iid := range iidCSPList {
			OnlyCSPList = append(OnlyCSPList, iid)
		}
		allResList.AllList.OnlyCSPList = OnlyCSPList

		return allResList, nil
	}

	////// iidInfoList has values, iidCSPList has values  <----------------------------------(4)
	// (3) filtering CSP-list by IID-list
	MappedList := []*cres.IID{}
	OnlySpiderList := []*cres.IID{}
	for _, iidInfo := range iidInfoList {
		exist := false
		for _, iid := range iidCSPList {
			userIId := getUserIID(iidInfo.IId)
			if userIId.SystemId == iid.SystemId {
				MappedList = append(MappedList, &userIId)
				exist = true
			}
		}
		if exist == false {
			userIId := getUserIID(iidInfo.IId)
			OnlySpiderList = append(OnlySpiderList, &userIId)
		}
	}

	allResList.AllList.MappedList = MappedList
	allResList.AllList.OnlySpiderList = OnlySpiderList

	OnlyCSPList := []*cres.IID{}
	for _, iid := range iidCSPList {
		if MappedList == nil || len(MappedList) <= 0 {
			//userIId := getUserIID(*iid)
			OnlyCSPList = append(OnlyCSPList, iid)
		} else {
			isMapped := false
			for _, mappedInfo := range MappedList {
				if iid.SystemId == mappedInfo.SystemId {
					isMapped = true
				}
			}
			if isMapped == false {
				// userIId := getUserIID(*iid)
				OnlyCSPList = append(OnlyCSPList, iid)
			}
		}
	}
	allResList.AllList.OnlyCSPList = OnlyCSPList

	return allResList, nil
}

func getUserIIDList(iidInfoList []*iidm.IIDInfo) []*cres.IID {
	iidList := []*cres.IID{}
	for _, iidInfo := range iidInfoList {
		userIId := getUserIID(iidInfo.IId)
		iidList = append(iidList, &userIId)
	}
	return iidList
}

// (1) get spiderIID
// (2) delete Resource(SystemId)
// (3) delete IID
func DeleteResource(connectionName string, rsType string, nameID string, force string) (bool, cres.VMStatus, error) {
	cblog.Info("call DeleteResource()")

	// check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
                return false, "", err
		cblog.Error(err)
        }

        nameID, err = EmptyCheckAndTrim("nameID", nameID)
        if err != nil {
                return false, "", err
		cblog.Error(err)
        }

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return false, "", err
	}

	var handler interface{}

	switch rsType {
	case rsVPC:
		handler, err = cldConn.CreateVPCHandler()
	case rsSG:
		handler, err = cldConn.CreateSecurityHandler()
	case rsKey:
		handler, err = cldConn.CreateKeyPairHandler()
	case rsVM:
		handler, err = cldConn.CreateVMHandler()
	default:
		return false, "", fmt.Errorf(rsType + " is not supported Resource!!")
	}
	if err != nil {
		cblog.Error(err)
		return false, "", err
	}

	switch rsType {
	case rsVPC:
		vpcRWLock.Lock()
		defer vpcRWLock.Unlock()
	case rsSG:
		sgRWLock.Lock()
		defer sgRWLock.Unlock()
	case rsKey:
		keyRWLock.Lock()
		defer keyRWLock.Unlock()
	case rsVM:
	//	vmRWLock.Lock()
	//	defer vmRWLock.Unlock()
	default:
		return false, "", fmt.Errorf(rsType + " is not supported Resource!!")
	}

	// (1) get spiderIID for creating driverIID
	var iidInfo *iidm.IIDInfo
	switch rsType {
	case rsSG:
		iidInfoList, err := getAllSGIIDInfoList(connectionName)
		if err != nil {
			cblog.Error(err)
			return false, "", err
		}
		for _, OneIIdInfo := range iidInfoList {
			if OneIIdInfo.IId.NameId == nameID {
				iidInfo = OneIIdInfo
			}
		}

	default:
		iidInfo, err = iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsType, cres.IID{nameID, ""})
		if err != nil {
			cblog.Error(err)
			return false, "", err
		}
	}

	// (2) delete Resource(SystemId)
	driverIId := getDriverIID(iidInfo.IId)
	result := false
	var vmStatus cres.VMStatus
	switch rsType {
	case rsVPC:
		result, err = handler.(cres.VPCHandler).DeleteVPC(driverIId)
		if err != nil {
			cblog.Error(err)
			if force != "true" {
				return false, "", err
			}
		}
	case rsSG:		
		result, err = handler.(cres.SecurityHandler).DeleteSecurity(driverIId)
		if err != nil {
			cblog.Error(err)
			if force != "true" {
				return false, "", err
			}
		}
	case rsKey:
		result, err = handler.(cres.KeyPairHandler).DeleteKey(driverIId)
		if err != nil {
			cblog.Error(err)
			if force != "true" {
				return false, "", err
			}
		}
	case rsVM:
		providerName, err := ccm.GetProviderNameByConnectionName(connectionName)
		if err != nil {
			cblog.Error(err)
			return false, "", err
		}

		regionName, zoneName, err := ccm.GetRegionNameByConnectionName(connectionName)
		if err != nil {
			cblog.Error(err)
			return false, "", err
		}

		callInfo := call.CLOUDLOGSCHEMA {
			CloudOS: call.CLOUD_OS(providerName),
			RegionZone: regionName + "/" + zoneName,
			ResourceType: call.VM,
			ResourceName: iidInfo.IId.NameId,
			CloudOSAPI: "CB-Spider:TerminateVM()",
			ElapsedTime: "",
			ErrorMSG: "",
		}
		start := call.Start()
		vmStatus, err = handler.(cres.VMHandler).TerminateVM(driverIId)
		callInfo.ElapsedTime = call.Elapsed(start)
		if err != nil {
			cblog.Error(err)
			callInfo.ErrorMSG = err.Error()
			if force != "true" {
				return false, vmStatus, err
			}
		}
		callogger.Info(call.String(callInfo))


	default:
		return false, "", fmt.Errorf(rsType + " is not supported Resource!!")
	}

	if force != "true" {
		if rsType != rsVM {
			if result == false {
				return result, "", nil
			}
		}
	}

	// (3) delete IID
        switch rsType {
        case rsVPC:
		// for vPC
		_, err = iidRWLock.DeleteIID(iidm.IIDSGROUP, connectionName, rsType, iidInfo.IId)
		if err != nil {
			cblog.Error(err)
			if force != "true" {
				return false, "", err
			}
		}
                // for Subnet list
                // key-value structure: ~/{SUBNETGROUP}/{ConnectionName}/{VPC-NameId}/{Subnet-reqNameId} [subnet-driverNameId:subnet-driverSystemId]  # VPC NameId => rsType
                subnetIIdInfoList, err2 := iidRWLock.ListIID(iidm.SUBNETGROUP, connectionName, iidInfo.IId.NameId/*vpcName*/)
                if err2 != nil {
                        cblog.Error(err)
                        if force != "true" {
                                return false, "", err
                        }
                }
                for _, subnetIIdInfo := range subnetIIdInfoList {
                        // key-value structure: ~/{SUBNETGROUP}/{ConnectionName}/{VPC-NameId}/{Subnet-reqNameId} [subnet-driverNameId:subnet-driverSystemId]  # VPC NameId => rsType
                        _, err := iidRWLock.DeleteIID(iidm.SUBNETGROUP, connectionName, iidInfo.IId.NameId/*vpcName*/, subnetIIdInfo.IId)
                        if err != nil {
                                cblog.Error(err)
                                if force != "true" {
                                        return false, "", err
                                }
                        }
                }

                // @todo Should we also delete the SG list of this VPC ?

        case rsSG:
                _, err = iidRWLock.DeleteIID(iidm.SGGROUP, connectionName, iidInfo.ResourceType/*vpcName*/, cres.IID{nameID, ""})
                if err != nil {
                        cblog.Error(err)
                        if force != "true" {
                                return false, "", err
                        }
                }
        case rsVM:
                _, err = iidRWLock.DeleteIID(iidm.IIDSGROUP, connectionName, rsType, iidInfo.IId)
                if err != nil {
                        cblog.Error(err)
                        if force != "true" {
                                return false, "", err
                        }
                }
		return result, vmStatus, nil
        default: // ex) KeyPair
		_, err = iidRWLock.DeleteIID(iidm.IIDSGROUP, connectionName, rsType, iidInfo.IId)
		if err != nil {
			cblog.Error(err)
			if force != "true" {
				return false, "", err
			}
		}
        }


	// except rsVM
	return result, "", nil
}

// (1) get spiderIID
// (2) delete Resource(SystemId)
// (3) delete IID
func RemoveSubnet(connectionName string, vpcName string, nameID string, force string) (bool, error) {
	cblog.Info("call RemoveSubnet()")

	// check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
                return false, err
		cblog.Error(err)
        }

        vpcName, err = EmptyCheckAndTrim("vpcName", vpcName)
        if err != nil {
                return false, err
		cblog.Error(err)
        }

        nameID, err = EmptyCheckAndTrim("nameID", nameID)
        if err != nil {
                return false, err
		cblog.Error(err)
        }

        force, err = EmptyCheckAndTrim("force", force)
        if err != nil {
                return false, err
		cblog.Error(err)
        }

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	handler, err := cldConn.CreateVPCHandler()
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	vpcRWLock.Lock()
	defer vpcRWLock.Unlock()

	// (1) get spiderIID for creating driverIID
	iidInfo, err := iidRWLock.GetIID(iidm.SUBNETGROUP, connectionName, vpcName, cres.IID{nameID, ""})
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	// (2) delete Resource(SystemId)
	driverIId := getDriverIID(iidInfo.IId)
	result := false


	iidVPCInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsVPC, cres.IID{vpcName, ""})
	if err != nil {
		cblog.Error(err)
		return false, err
	}
	result, err = handler.(cres.VPCHandler).RemoveSubnet(getDriverIID(iidVPCInfo.IId), driverIId)
	if err != nil {
		cblog.Error(err)
		if force != "true" {
			return false, err
		}
	}

	if force != "true" {
		if result == false {
			return result, nil
		}
	}

	// (3) delete IID
	_, err = iidRWLock.DeleteIID(iidm.SUBNETGROUP, connectionName, vpcName, cres.IID{nameID, ""})
	if err != nil {
		cblog.Error(err)
		if force != "true" {
			return false, err
		}
	}


	return result, nil
}

// delete CSP's Resource(SystemId)
func DeleteCSPResource(connectionName string, rsType string, systemID string) (bool, cres.VMStatus, error) {
	cblog.Info("call DeleteCSPResource()")

	// check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
                return false, "", err
		cblog.Error(err)
        }

        systemID, err = EmptyCheckAndTrim("systemID", systemID)
        if err != nil {
                return false, "", err
		cblog.Error(err)
        }

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return false, "", err
	}

	var handler interface{}

	switch rsType {
	case rsVPC:
		handler, err = cldConn.CreateVPCHandler()
	case rsSG:
		handler, err = cldConn.CreateSecurityHandler()
	case rsKey:
		handler, err = cldConn.CreateKeyPairHandler()
	case rsVM:
		handler, err = cldConn.CreateVMHandler()
	default:
		return false, "", fmt.Errorf(rsType + " is not supported Resource!!")
	}
	if err != nil {
		cblog.Error(err)
		return false, "", err
	}

	iid := cres.IID{"", systemID}

	// delete CSP's Resource(SystemId)	
	result := false
	var vmStatus cres.VMStatus
	switch rsType {
	case rsVPC:
		result, err = handler.(cres.VPCHandler).DeleteVPC(iid)
		if err != nil {
			cblog.Error(err)
			return false, "", err
		}
	case rsSG:
		result, err = handler.(cres.SecurityHandler).DeleteSecurity(iid)
		if err != nil {
			cblog.Error(err)
			return false, "", err
		}
	case rsKey:
		result, err = handler.(cres.KeyPairHandler).DeleteKey(iid)
		if err != nil {
			cblog.Error(err)
			return false, "", err
		}
	case rsVM:
		vmStatus, err = handler.(cres.VMHandler).TerminateVM(iid)
		if err != nil {
			cblog.Error(err)
			return false, vmStatus, err
		}
	default:
		return false, "", fmt.Errorf(rsType + " is not supported Resource!!")
	}

	if rsType != rsVM {
		if result == false {
			return result, "", nil
		}
	}

	if rsType == rsVM {
		return result, vmStatus, nil
	} else {
		return result, "", nil
	}
}


// remove CSP's Subnet(SystemId)
func RemoveCSPSubnet(connectionName string, vpcName string, systemID string) (bool, error) {
        cblog.Info("call DeleteCSPSubnet()")

	// check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
                return false, err
		cblog.Error(err)
        }

        vpcName, err = EmptyCheckAndTrim("vpcName", vpcName)
        if err != nil {
                return false, err
		cblog.Error(err)
        }

        systemID, err = EmptyCheckAndTrim("systemID", systemID)
        if err != nil {
                return false, err
		cblog.Error(err)
        }

        cldConn, err := ccm.GetCloudConnection(connectionName)
        if err != nil {
                cblog.Error(err)
                return false, err
        }

	handler, err := cldConn.CreateVPCHandler()
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        iid := cres.IID{"", systemID}

        // delete Resource(SystemId)
        result := false
	// get owner vpc IIDInfo
	iidVPCInfo, err := iidRWLock.GetIID(iidm.IIDSGROUP, connectionName, rsVPC, cres.IID{vpcName, ""})
	if err != nil {
		cblog.Error(err)
		return false, err
	}
	result, err = handler.(cres.VPCHandler).RemoveSubnet(getDriverIID(iidVPCInfo.IId), iid)
	if err != nil {
		cblog.Error(err)
		return false, err
	}


	return result, nil
}

