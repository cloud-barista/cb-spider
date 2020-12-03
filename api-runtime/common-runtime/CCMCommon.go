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
	//	"strings"
	//	"strconv"
	//	"time"
)

// define string of resource types
const (
	rsImage string = "image"
	rsVPC   string = "vpc"
	// rsSubnet = SUBNET:{VPC NameID} => cook in code
	rsSG  string = "sg"
	rsKey string = "keypair"
	rsVM  string = "vm"
)

const rsSubnetPrefix string = "subnet:"
const sgDELIMITER string = "-delimiter-"

// definition of RWLock for each Resource Ops
var imgRWLock = new(sync.RWMutex)
var vpcRWLock = new(sync.RWMutex)
var sgRWLock = new(sync.RWMutex)
var keyRWLock = new(sync.RWMutex)
var vmRWLock = new(sync.RWMutex)

// definition of IIDManager RWLock
var iidRWLock = new(iidm.IIDRWLOCK)

var cblog *logrus.Logger

func init() {
	cblog = config.Cblogger
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
// (2) create Resource
// (3) insert IID
func CreateImage(connectionName string, rsType string, reqInfo cres.ImageReqInfo) (*cres.ImageInfo, error) {
	cblog.Info("call CreateImage()")

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
	bool_ret, err := iidRWLock.IsExistIID(connectionName, rsType, reqInfo.IId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	if bool_ret == true {
		return nil, fmt.Errorf(reqInfo.IId.NameId + " already exists!")
	}

	// (2) create Resource
	info, err := handler.CreateImage(reqInfo)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (3) insert IID
	iidInfo, err := iidRWLock.CreateIID(connectionName, rsType, info.IId)
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

// (1) get IID:list
// (2) get CSP:list
// (3) filtering CSP-list by IID-list
func ListRegisterImage(connectionName string, rsType string) ([]*cres.ImageInfo, error) {
        cblog.Info("call ListImage()")

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
        // (1) get IID:list
        iidInfoList, err := iidRWLock.ListIID(connectionName, rsType)
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

        // (3) filtering CSP-list by IID-list
        infoList2 := []*cres.ImageInfo{}
        for _, iidInfo := range iidInfoList {
                exist := false
                for _, info := range infoList {
                        if iidInfo.IId.SystemId == info.IId.SystemId {
                                info.IId.NameId = iidInfo.IId.NameId
                                infoList2 = append(infoList2, info)
                                exist = true
                        }
                }
                if exist == false {
                        return nil, fmt.Errorf("<IID-CSP mismatch> " + rsType + "-" + iidInfo.IId.NameId + ":" + iidInfo.IId.SystemId + " exsits. but " + connectionName + " does not have!")
                }
        }

        return infoList2, nil
}


// (1) get IID(NameId)
// (2) get resource(SystemId)
// (3) set ResourceInfo(IID.NameId)
func GetImage(connectionName string, rsType string, nameID string) (*cres.ImageInfo, error) {
	cblog.Info("call GetImage()")

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

// (1) get IID(NameId)
// (2) get resource(SystemId)
// (3) set ResourceInfo(IID.NameId)
func GetRegisterImage(connectionName string, rsType string, nameID string) (*cres.ImageInfo, error) {
        cblog.Info("call GetImage()")

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
        // (1) get IID(NameId)
        iidInfo, err := iidRWLock.GetIID(connectionName, rsType, cres.IID{nameID, ""})
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // (2) get resource(SystemId)
        start := time.Now()
        info, err := handler.GetImage(iidInfo.IId)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        elapsed := time.Since(start)
        cblog.Infof(connectionName+" : elapsed %d", elapsed.Nanoseconds()/1000000) // msec

        // (3) set ResourceInfo(IID.NameId)
        info.IId.NameId = iidInfo.IId.NameId

        return &info, nil
}

// (1) get IID(NameId)
// (2) delete Resource(SystemId)
// (3) delete IID
func DeleteImage(connectionName string, rsType string, nameID string) (bool, error) {
	cblog.Info("call DeleteImage()")

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
	// (1) get IID(NameId)
	iidInfo, err := iidRWLock.GetIID(connectionName, rsType, cres.IID{nameID, ""})
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	// keeping for rollback
	info, err := handler.GetImage(iidInfo.IId)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	// (2) delete Resource(SystemId)
	result, err := handler.DeleteImage(iidInfo.IId)
	if err != nil {
		cblog.Error(err)
		return false, err
	}
	if result == false {
		return result, nil
	}

	// (3) delete IID
	_, err = iidRWLock.DeleteIID(connectionName, rsType, iidInfo.IId)
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
// (1) check exist(NameID)
// (2) create Resource
// (3) insert IID
func CreateVPC(connectionName string, rsType string, reqInfo cres.VPCReqInfo) (*cres.VPCInfo, error) {
	cblog.Info("call CreateVPC()")

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
	bool_ret, err := iidRWLock.IsExistIID(connectionName, rsType, reqInfo.IId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	if bool_ret == true {
		return nil, fmt.Errorf(rsType + "-" + reqInfo.IId.NameId + " already exists!")
	}
	// (2) create Resource
	info, err := handler.CreateVPC(reqInfo)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	info.IId.NameId = reqInfo.IId.NameId

	// (3) insert IID
	// for VPC
	iidInfo, err := iidRWLock.CreateIID(connectionName, rsType, info.IId)
	if err != nil {
		cblog.Error(err)
		// rollback
		_, err2 := handler.DeleteVPC(iidInfo.IId)
		if err2 != nil {
			cblog.Error(err2)
			return nil, fmt.Errorf(err.Error() + ", " + err2.Error())
		}
		return nil, err
	}
	// for Subnet list
	for _, subnetInfo := range info.SubnetInfoList {
		// key-value structure: /{ConnectionName}/{VPC-NameId}/{Subnet-IId}
		_, err := iidRWLock.CreateIID(connectionName, rsSubnetPrefix+info.IId.NameId, subnetInfo.IId)
		if err != nil {
			cblog.Error(err)
			// rollback
			// (1) for resource
			cblog.Info("<<ROLLBACK:TRY:VPC-CSP>> " + info.IId.SystemId)
			_, err2 := handler.DeleteVPC(iidInfo.IId)
			if err2 != nil {
				cblog.Error(err2)
				return nil, fmt.Errorf(err.Error() + ", " + err2.Error())
			}
			// (2) for VPC IID
			cblog.Info("<<ROLLBACK:TRY:VPC-IID>> " + info.IId.NameId)
			_, err3 := iidRWLock.DeleteIID(connectionName, rsType, info.IId)
			if err3 != nil {
				cblog.Error(err3)
				return nil, fmt.Errorf(err.Error() + ", " + err3.Error())
			}
			// (3) for Subnet IID
			tmpIIdInfoList, err := iidRWLock.ListIID(connectionName, rsSubnetPrefix+info.IId.NameId)
			for _, subnetInfo := range tmpIIdInfoList {
				_, err := iidRWLock.DeleteIID(connectionName, rsSubnetPrefix+info.IId.NameId, subnetInfo.IId)
				if err != nil {
					cblog.Error(err)
					return nil, err
				}
			}
			return nil, err
		}
	}

	return &info, nil
}

// (1) get IID:list
// (2) get CSP:list
// (3) filtering CSP-list by IID-list
func ListVPC(connectionName string, rsType string) ([]*cres.VPCInfo, error) {
	cblog.Info("call ListVPC()")

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
	iidInfoList, err := iidRWLock.ListIID(connectionName, rsType)
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
	infoList2 := []*cres.VPCInfo{}
	for _, iidInfo := range iidInfoList {
		exist := false
		for _, info := range infoList {
			if iidInfo.IId.SystemId == info.IId.SystemId {

				//+++++++++++++++++++++++++++++++++++++++++++
				// set ResourceInfo(IID.NameId)
				// set VPC NameId
				IIdInfo, err := iidRWLock.GetIIDbySystemID(connectionName, rsType, info.IId)
				if err != nil {
					cblog.Error(err)
					return nil, err
				}
				info.IId.NameId = IIdInfo.IId.NameId
				//+++++++++++++++++++++++++++++++++++++++++++
				// set NameId for SubnetInfo List
				// create new SubnetInfo List
				subnetInfoList := []cres.SubnetInfo{}
				for _, subnetInfo := range info.SubnetInfoList {
					subnetIIDInfo, err := iidRWLock.GetIIDbySystemID(connectionName, rsSubnetPrefix+info.IId.NameId, subnetInfo.IId)
					if err != nil {
						cblog.Error(err)
						return nil, err
					}
					if subnetIIDInfo.IId.NameId != "" { // insert only this user created.
						subnetInfo.IId.NameId = subnetIIDInfo.IId.NameId
						subnetInfoList = append(subnetInfoList, subnetInfo)
					}

				}
				info.SubnetInfoList = subnetInfoList

				infoList2 = append(infoList2, info)
				exist = true
			}
		}
		if exist == false {
			return nil, fmt.Errorf("<IID-CSP mismatch> " + rsType + "-" + iidInfo.IId.NameId + ":" + iidInfo.IId.SystemId + " exsits. but " + connectionName + " does not have!")
		}
	}

	return infoList2, nil
}

// (1) get IID(NameId)
// (2) get resource(SystemId)
// (3) set ResourceInfo(IID.NameId)
func GetVPC(connectionName string, rsType string, nameID string) (*cres.VPCInfo, error) {
	cblog.Info("call GetVPC()")

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
	// (1) get IID(NameId)
	iidInfo, err := iidRWLock.GetIID(connectionName, rsType, cres.IID{nameID, ""})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (2) get resource(SystemId)
	info, err := handler.GetVPC(iidInfo.IId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	// (3) set ResourceInfo(IID.NameId)
	info.IId.NameId = iidInfo.IId.NameId

	// set NameId for SubnetInfo List
	// create new SubnetInfo List
	subnetInfoList := []cres.SubnetInfo{}
	for i, subnetInfo := range info.SubnetInfoList {
		subnetIIDInfo, err := iidRWLock.GetIIDbySystemID(connectionName, rsSubnetPrefix+info.IId.NameId, subnetInfo.IId)
		if err != nil {
			cblog.Error(err)
			return nil, err
		}
		if subnetIIDInfo.IId.NameId != "" { // insert only this user created.
			info.SubnetInfoList[i].IId.NameId = subnetIIDInfo.IId.NameId
			subnetInfoList = append(subnetInfoList, info.SubnetInfoList[i])
		}
	}
	info.SubnetInfoList = subnetInfoList

	return &info, nil
}

//================ SecurityGroup Handler
// (1) check exist(NameID)
// (2) create Resource
// (3) insert IID
func CreateSecurity(connectionName string, rsType string, reqInfo cres.SecurityReqInfo) (*cres.SecurityInfo, error) {
	cblog.Info("call CreateSecurity()")

	//+++++++++++++++++++++++++++++++++++++++++++
	// set VPC SystemId
	vpcIIDInfo, err := iidRWLock.GetIID(connectionName, rsVPC, reqInfo.VpcIID)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	reqInfo.VpcIID.SystemId = vpcIIDInfo.IId.SystemId
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
	bool_ret, err := iidRWLock.IsExistIID(connectionName, rsType, reqInfo.IId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	if bool_ret == true {
		return nil, fmt.Errorf(rsType + "-" + reqInfo.IId.NameId + " already exists!")
	}

	// (2) create Resource
	info, err := handler.CreateSecurity(reqInfo)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// set VPC NameId
	info.VpcIID.NameId = reqInfo.VpcIID.NameId
	info.VpcIID.SystemId = reqInfo.VpcIID.SystemId

	// (3) insert IID
	iidInfo, err := iidRWLock.CreateIID(connectionName, rsType, info.IId)
	if err != nil {
		cblog.Error(err)
		// rollback
		_, err2 := handler.DeleteSecurity(iidInfo.IId)
		if err2 != nil {
			cblog.Error(err2)
			return nil, fmt.Errorf(err.Error() + ", " + err2.Error())
		}
		return nil, err
	}

	// set ResourceInfo(IID.NameId)
	// iidInfo.IId.NameID format => {VPC NameID} + sgDELIMITER + {SG NameID}
	vpc_sg_nameid := strings.Split(info.IId.NameId, sgDELIMITER)
	info.IId.NameId = vpc_sg_nameid[1]

	return &info, nil
}

// (1) get IID:list
// (2) get CSP:list
// (3) filtering CSP-list by IID-list
func ListSecurity(connectionName string, rsType string) ([]*cres.SecurityInfo, error) {
	cblog.Info("call ListSecurity()")

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
	iidInfoList, err := iidRWLock.ListIID(connectionName, rsType)
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
			if iidInfo.IId.SystemId == info.IId.SystemId {

				// set ResourceInfo(IID.NameId)
				// iidInfo.IId.NameID format => {VPC NameID} + sgDELIMITER + {SG NameID}
				vpc_sg_nameid := strings.Split(iidInfo.IId.NameId, sgDELIMITER)
				info.VpcIID.NameId = vpc_sg_nameid[0]
				info.IId.NameId = vpc_sg_nameid[1]

				// set VPC SystemId
				vpcIIDInfo, err := iidRWLock.GetIID(connectionName, rsVPC, info.VpcIID)
				if err != nil {
					cblog.Error(err)
					return nil, err
				}
				info.VpcIID.SystemId = vpcIIDInfo.IId.SystemId

				infoList2 = append(infoList2, info)
				exist = true
			}
		}
		if exist == false {
			return nil, fmt.Errorf("<IID-CSP mismatch> " + rsType + "-" + iidInfo.IId.NameId + ":" + iidInfo.IId.SystemId + " exsits. but " + connectionName + " does not have!")
		}
	}

	return infoList2, nil
}

// (1) get IID(NameId)
// (2) get resource(SystemId)
// (3) set ResourceInfo(IID.NameId)
func GetSecurity(connectionName string, rsType string, nameID string) (*cres.SecurityInfo, error) {
	cblog.Info("call GetSecurity()")

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
	// SG NameID format => {VPC NameID} + sgDELIMITER + {SG NameID}
	iidInfo, err := iidRWLock.FindIID(connectionName, rsType, nameID)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (2) get resource(SystemId)
	info, err := handler.GetSecurity(iidInfo.IId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (3) set ResourceInfo(IID.NameId)
	// set ResourceInfo(IID.NameId)
	// iidInfo.IId.NameID format => {VPC NameID} + sgDELIMITER + {SG NameID}
	vpc_sg_nameid := strings.Split(iidInfo.IId.NameId, sgDELIMITER)
	info.VpcIID.NameId = vpc_sg_nameid[0]
	info.IId.NameId = vpc_sg_nameid[1]

	// set VPC SystemId
	vpcIIDInfo, err := iidRWLock.GetIID(connectionName, rsVPC, info.VpcIID)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	info.VpcIID.SystemId = vpcIIDInfo.IId.SystemId

	return &info, nil
}

//================ KeyPair Handler
// (1) check exist(NameID)
// (2) create Resource
// (3) insert IID
func CreateKey(connectionName string, rsType string, reqInfo cres.KeyPairReqInfo) (*cres.KeyPairInfo, error) {
	cblog.Info("call CreateKey()")

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
	bool_ret, err := iidRWLock.IsExistIID(connectionName, rsType, reqInfo.IId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	if bool_ret == true {
		return nil, fmt.Errorf(reqInfo.IId.NameId + " already exists!")
	}

	// (2) create Resource
	info, err := handler.CreateKey(reqInfo)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (3) insert IID
	iidInfo, err := iidRWLock.CreateIID(connectionName, rsType, info.IId)
	if err != nil {
		cblog.Error(err)
		// rollback
		_, err2 := handler.DeleteKey(iidInfo.IId)
		if err2 != nil {
			cblog.Error(err2)
			return nil, fmt.Errorf(err.Error() + ", " + err2.Error())
		}
		return nil, err
	}

	return &info, nil
}

// (1) get IID:list
// (2) get CSP:list
// (3) filtering CSP-list by IID-list
func ListKey(connectionName string, rsType string) ([]*cres.KeyPairInfo, error) {
	cblog.Info("call ListKey()")

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
	iidInfoList, err := iidRWLock.ListIID(connectionName, rsType)
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
			if iidInfo.IId.SystemId == info.IId.SystemId {
				infoList2 = append(infoList2, info)
				exist = true
			}
		}
		if exist == false {
			return nil, fmt.Errorf("<IID-CSP mismatch> " + rsType + "-" + iidInfo.IId.NameId + ":" + iidInfo.IId.SystemId + " exsits. but " + connectionName + " does not have!")
		}
	}

	return infoList2, nil
}

// (1) get IID(NameId)
// (2) get resource(SystemId)
// (3) set ResourceInfo(IID.NameId)
func GetKey(connectionName string, rsType string, nameID string) (*cres.KeyPairInfo, error) {
	cblog.Info("call GetKey()")

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
	iidInfo, err := iidRWLock.GetIID(connectionName, rsType, cres.IID{nameID, ""})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (2) get resource(SystemId)
	info, err := handler.GetKey(iidInfo.IId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (3) set ResourceInfo(IID.NameId)
	info.IId.NameId = iidInfo.IId.NameId

	return &info, nil
}

func getSetSystemId(ConnectionName string, reqInfo *cres.VMReqInfo) error {

	// set Image SystemId
	// @todo before Image Handling by powerkim
	reqInfo.ImageIID.SystemId = reqInfo.ImageIID.NameId

	// set VPC SystemId
	if reqInfo.VpcIID.NameId != "" {
		IIdInfo, err := iidRWLock.GetIID(ConnectionName, rsVPC, reqInfo.VpcIID)
		if err != nil {
			cblog.Error(err)
			return err
		}
		reqInfo.VpcIID.SystemId = IIdInfo.IId.SystemId
	}

	// set Subnet SystemId
	if reqInfo.SubnetIID.NameId != "" {
		IIdInfo, err := iidRWLock.GetIID(ConnectionName, rsSubnetPrefix+reqInfo.VpcIID.NameId, reqInfo.SubnetIID)
		if err != nil {
			cblog.Error(err)
			return err
		}
		reqInfo.SubnetIID.SystemId = IIdInfo.IId.SystemId
	}

	// set SecurityGroups SystemId
	for i, sgIID := range reqInfo.SecurityGroupIIDs {
		IIdInfo, err := iidRWLock.GetIID(ConnectionName, rsSG, sgIID)
		if err != nil {
			cblog.Error(err)
			return err
		}
		reqInfo.SecurityGroupIIDs[i].SystemId = IIdInfo.IId.SystemId
	}

	// set KeyPair SystemId
	if reqInfo.KeyPairIID.NameId != "" {
		IIdInfo, err := iidRWLock.GetIID(ConnectionName, rsKey, reqInfo.KeyPairIID)
		if err != nil {
			cblog.Error(err)
			return err
		}
		reqInfo.KeyPairIID.SystemId = IIdInfo.IId.SystemId
	}

	return nil
}

//================ VM Handler
// (1) check exist(NameID)
// (2) create Resource
// (3) insert IID
func StartVM(connectionName string, rsType string, reqInfo cres.VMReqInfo) (*cres.VMInfo, error) {
	cblog.Info("call StartVM()")

	// get & set SystemId
	err := getSetSystemId(connectionName, &reqInfo)
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
	bool_ret, err := iidRWLock.IsExistIID(connectionName, rsType, reqInfo.IId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	if bool_ret == true {
		return nil, fmt.Errorf(rsType + "-" + reqInfo.IId.NameId + " already exists!")
	}

	// (2) create Resource
	info, err := handler.StartVM(reqInfo)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	// set NameId for info by reqInfo
	setNameId(connectionName, &info, &reqInfo)

	// (3) insert IID
	iidInfo, err := iidRWLock.CreateIID(connectionName, rsType, info.IId)
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

	// set sg NameId from VPCNameId-SecurityGroupNameId
	// IID.NameID format => {VPC NameID} + sgDELIMITER + {SG NameID}
	for i, sgIID := range info.SecurityGroupIIds {
		vpc_sg_nameid := strings.Split(sgIID.NameId, sgDELIMITER)
		info.SecurityGroupIIds[i].NameId = vpc_sg_nameid[1]
	}

	return &info, nil
}

func setNameId(ConnectionName string, vmInfo *cres.VMInfo, reqInfo *cres.VMReqInfo) error {

	// set Image SystemId
	// @todo before Image Handling by powerkim
	if reqInfo.ImageIID.NameId != "" {
		vmInfo.ImageIId.NameId = reqInfo.ImageIID.NameId
	}

	// set VPC SystemId
	if reqInfo.VpcIID.NameId != "" {
		vmInfo.VpcIID.NameId = reqInfo.VpcIID.NameId
	}

	if reqInfo.SubnetIID.NameId != "" {
		// set Subnet SystemId
		vmInfo.SubnetIID.NameId = reqInfo.SubnetIID.NameId
	}

	vmInfo.SecurityGroupIIds = reqInfo.SecurityGroupIIDs

	// set SecurityGroups SystemId
	for i, sgIID := range reqInfo.SecurityGroupIIDs {
		IIdInfo, err := iidRWLock.GetIIDbySystemID(ConnectionName, rsSG, sgIID)
		if err != nil {
			cblog.Error(err)
			return err
		}
		reqInfo.SecurityGroupIIDs[i].NameId = IIdInfo.IId.NameId
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

	vmRWLock.RLock()
	defer vmRWLock.RUnlock()
	// (1) get IID:list
	iidInfoList, err := iidRWLock.ListIID(connectionName, rsType)
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
			if iidInfo.IId.SystemId == info.IId.SystemId {

				//+++++++++++++++++++++++++++++++++++++++++++
				// set ResourceInfo(IID.NameId)
				// set VPC NameId
				IIdInfo, err := iidRWLock.GetIIDbySystemID(connectionName, rsType, info.IId)
				if err != nil {
					cblog.Error(err)
					return nil, err
				}
				info.IId.NameId = IIdInfo.IId.NameId
				//+++++++++++++++++++++++++++++++++++++++++++
				err = getSetNameId(connectionName, info)
				if err != nil {
					cblog.Error(err)
					return nil, err
				}

				// set sg NameId from VPCNameId-SecurityGroupNameId
				// IID.NameID format => {VPC NameID} + sgDELIMITER + {SG NameID}
				for i, sgIID := range info.SecurityGroupIIds {
					vpc_sg_nameid := strings.Split(sgIID.NameId, sgDELIMITER)
					info.SecurityGroupIIds[i].NameId = vpc_sg_nameid[1]
				}

				infoList2 = append(infoList2, info)
				exist = true
			}
		}
		if exist == false {
			return nil, fmt.Errorf("<IID-CSP mismatch> " + rsType + "-" + iidInfo.IId.NameId + ":" + iidInfo.IId.SystemId + " exsits. but " + connectionName + " does not have!")
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
		IIdInfo, err := iidRWLock.GetIIDbySystemID(ConnectionName, rsVPC, vmInfo.VpcIID)
		if err != nil {
			cblog.Error(err)
			return err
		}
		vmInfo.VpcIID.NameId = IIdInfo.IId.NameId
	}

	if vmInfo.SubnetIID.SystemId != "" {
		// set Subnet NameId
		IIdInfo, err := iidRWLock.GetIIDbySystemID(ConnectionName, rsSubnetPrefix+vmInfo.VpcIID.NameId, vmInfo.SubnetIID)
		if err != nil {
			cblog.Error(err)
			return err
		}
		vmInfo.SubnetIID.NameId = IIdInfo.IId.NameId
	}

	// set SecurityGroups NameId
	for i, sgIID := range vmInfo.SecurityGroupIIds {
		IIdInfo, err := iidRWLock.GetIIDbySystemID(ConnectionName, rsSG, sgIID)
		if err != nil {
			cblog.Error(err)
			return err
		}
		vmInfo.SecurityGroupIIds[i].NameId = IIdInfo.IId.NameId
	}

	if vmInfo.KeyPairIId.SystemId != "" {
		// set KeyPair NameId
		IIdInfo, err := iidRWLock.GetIIDbySystemID(ConnectionName, rsKey, vmInfo.KeyPairIId)
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

	vmRWLock.RLock()
	defer vmRWLock.RUnlock()
	// (1) get IID(NameId)
	iidInfo, err := iidRWLock.GetIID(connectionName, rsType, cres.IID{nameID, ""})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (2) get resource(SystemId)
	info, err := handler.GetVM(iidInfo.IId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (3) set ResourceInfo(IID.NameId)
	info.IId.NameId = iidInfo.IId.NameId

	err = getSetNameId(connectionName, &info)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// set sg NameId from VPCNameId-SecurityGroupNameId
	// IID.NameID format => {VPC NameID} + sgDELIMITER + {SG NameID}
	for i, sgIID := range info.SecurityGroupIIds {
		vpc_sg_nameid := strings.Split(sgIID.NameId, sgDELIMITER)
		info.SecurityGroupIIds[i].NameId = vpc_sg_nameid[1]
	}

	return &info, nil
}

// (1) get IID:list
// (2) get CSP:VMStatuslist
// (3) filtering CSP-VMStatuslist by IID-list
func ListVMStatus(connectionName string, rsType string) ([]*cres.VMStatusInfo, error) {
	cblog.Info("call ListVMStatus()")

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

	vmRWLock.RLock()
	defer vmRWLock.RUnlock()
	// (1) get IID:list
	iidInfoList, err := iidRWLock.ListIID(connectionName, rsType)
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
			if iidInfo.IId.SystemId == info.IId.SystemId {
				info.IId.NameId = iidInfo.IId.NameId
				infoList2 = append(infoList2, info)
				exist = true
			}
		}
		if exist == false {
			return nil, fmt.Errorf("<IID-CSP mismatch> " + rsType + "-" + iidInfo.IId.NameId + ":" + iidInfo.IId.SystemId + " exsits. but " + connectionName + " does not have!")
		}
	}

	return infoList2, nil
}

// (1) get IID(NameId)
// (2) get CSP:VMStatus(SystemId)
func GetVMStatus(connectionName string, rsType string, nameID string) (cres.VMStatus, error) {
	cblog.Info("call GetVMStatus()")

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

	vmRWLock.RLock()
	defer vmRWLock.RUnlock()
	// (1) get IID(NameId)
	iidInfo, err := iidRWLock.GetIID(connectionName, rsType, cres.IID{nameID, ""})
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	// (2) get CSP:VMStatus(SystemId)
	info, err := handler.GetVMStatus(iidInfo.IId) // type of info => string
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

	vmRWLock.RLock()
	defer vmRWLock.RUnlock()
	// (1) get IID(NameId)
	iidInfo, err := iidRWLock.GetIID(connectionName, rsType, cres.IID{nameID, ""})
	if err != nil {
		cblog.Error(err)
		return "", err
	}

	// (2) control CSP:VM(SystemId)
	vmIID := iidInfo.IId

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
		vmRWLock.RLock()
		defer vmRWLock.RUnlock()
	default:
		return AllResourceList{}, fmt.Errorf(rsType + " is not supported Resource!!")
	}

	// (1) get IID:list
	iidInfoList, err := iidRWLock.ListIID(connectionName, rsType)
	if err != nil {
		cblog.Error(err)
		return AllResourceList{}, err
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
			allResList.AllList.OnlySpiderList = getIIDList(iidInfoList)

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
			if iidInfo.IId.SystemId == iid.SystemId {
				MappedList = append(MappedList, &iidInfo.IId)
				exist = true
			}
		}
		if exist == false {
			OnlySpiderList = append(OnlySpiderList, &iidInfo.IId)
		}
	}

	// if SG then MappedList, OnlySpiderList : remove delimeter and set SG name
	if rsType == rsSG { // vpc-01-delimiter-sg-01 ==> sg-01
		for i, iid := range MappedList{
			vpc_sg_nameid := strings.Split(iid.NameId, sgDELIMITER)
			MappedList[i].NameId = vpc_sg_nameid[1]
		}
		for i, iid := range OnlySpiderList{
			vpc_sg_nameid := strings.Split(iid.NameId, sgDELIMITER)
			OnlySpiderList[i].NameId = vpc_sg_nameid[1]
		}
	}


	allResList.AllList.MappedList = MappedList
	allResList.AllList.OnlySpiderList = OnlySpiderList


	OnlyCSPList := []*cres.IID{}
	for _, iid := range iidCSPList {
		if MappedList == nil || len(MappedList) <= 0 {
			OnlyCSPList = append(OnlyCSPList, iid)
		} else {
			isMapped := false
			for _, mappedInfo := range MappedList {
				if iid.SystemId == mappedInfo.SystemId {
					isMapped = true
				}
			}
			if isMapped == false {
				OnlyCSPList = append(OnlyCSPList, iid)
			}
		}
	}
	allResList.AllList.OnlyCSPList = OnlyCSPList

	return allResList, nil
}

func getIIDList(iidInfoList []*iidm.IIDInfo) []*cres.IID {
	iidList := []*cres.IID{}
	for _, iidInfo := range iidInfoList {
		iidList = append(iidList, &iidInfo.IId)
	}
	return iidList
}

// (1) get IID(NameId)
// (2) delete Resource(SystemId)
// (3) delete IID
func DeleteResource(connectionName string, rsType string, nameID string, force string) (bool, cres.VMStatus, error) {
	cblog.Info("call DeleteResource()")

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
		vmRWLock.Lock()
		defer vmRWLock.Unlock()
	default:
		return false, "", fmt.Errorf(rsType + " is not supported Resource!!")
	}

	// (1) get IID(NameId) for getting SystemId
	var iidInfo *iidm.IIDInfo
	if rsType == rsSG {
		// SG NameID format => {VPC NameID} + sgDELIMITER + {SG NameID}
		iidInfo, err = iidRWLock.FindIID(connectionName, rsType, nameID)
		if err != nil {
			cblog.Error(err)
			return false, "", err
		}
	} else {
		iidInfo, err = iidRWLock.GetIID(connectionName, rsType, cres.IID{nameID, ""})
		if err != nil {
			cblog.Error(err)
			return false, "", err
		}
	}

	// (2) delete Resource(SystemId)
	result := false
	var vmStatus cres.VMStatus
	switch rsType {
	case rsVPC:
		result, err = handler.(cres.VPCHandler).DeleteVPC(iidInfo.IId)
		if err != nil {
			cblog.Error(err)
			if force != "true" {
				return false, "", err
			}
		}
	case rsSG:
		result, err = handler.(cres.SecurityHandler).DeleteSecurity(iidInfo.IId)
		if err != nil {
			cblog.Error(err)
			if force != "true" {
				return false, "", err
			}
		}
	case rsKey:
		result, err = handler.(cres.KeyPairHandler).DeleteKey(iidInfo.IId)
		if err != nil {
			cblog.Error(err)
			if force != "true" {
				return false, "", err
			}
		}
	case rsVM:
		vmStatus, err = handler.(cres.VMHandler).TerminateVM(iidInfo.IId)
		if err != nil {
			cblog.Error(err)
			if force != "true" {
				return false, vmStatus, err
			}
		}
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
	_, err = iidRWLock.DeleteIID(connectionName, rsType, iidInfo.IId)
	if err != nil {
		cblog.Error(err)
		if force != "true" {
			return false, "", err
		}
	}

	// if VPC
	if rsType == rsVPC {
		// for Subnet list
		// key-value structure: /{ConnectionName}/rsSubnetPrefix+{VPC-NameId}/{Subnet-IId}
		subnetInfoList, err2 := iidRWLock.ListIID(connectionName, rsSubnetPrefix+iidInfo.IId.NameId)
		if err2 != nil {
			cblog.Error(err)
			if force != "true" {
				return false, "", err
			}
		}
		for _, subnetInfo := range subnetInfoList {
			// key-value structure: /{ConnectionName}/rsSubnetPrefix+{VPC-NameId}/{Subnet-IId}
			_, err := iidRWLock.DeleteIID(connectionName, rsSubnetPrefix+iidInfo.IId.NameId, subnetInfo.IId)
			if err != nil {
				cblog.Error(err)
				if force != "true" {
					return false, "", err
				}
			}
		}
	}

	if rsType == rsVM {
		return result, vmStatus, nil
	} else {
		return result, "", nil
	}
}

// (1) delete Resource(SystemId)
func DeleteCSPResource(connectionName string, rsType string, systemID string) (bool, cres.VMStatus, error) {
	cblog.Info("call DeleteResource()")

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

	// (1) delete Resource(SystemId)
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
