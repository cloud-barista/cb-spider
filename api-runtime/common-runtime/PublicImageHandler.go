// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.

package commonruntime

import (
	"fmt"
	"os"
	"time"
	"strings"
	"strconv"

	"github.com/go-redis/redis"
	"encoding/json"

	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	iidm "github.com/cloud-barista/cb-spider/cloud-control-manager/iid-manager"
)

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

	emptyPermissionList := []string{
                "resources.IID:SystemId",
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

	handler, err := cldConn.CreateImageHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	imgSPLock.Lock(connectionName, reqInfo.IId.NameId)
	defer imgSPLock.Unlock(connectionName, reqInfo.IId.NameId)
	// (1) check exist(NameID)
	bool_ret, err := iidRWLock.IsExistIID(iidm.IIDSGROUP, connectionName, rsType, reqInfo.IId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	if bool_ret == true {
		err := fmt.Errorf(rsType + "-" + reqInfo.IId.NameId + " already exists!")
		cblog.Error(err)
		return nil, err
	}

	// (2) gen SP-XID and create userIID, driverIID
	//     ex) SP-XID{"vm-01-9m4e2mr0ui3e8a215n4g"}
	//     ex) userIID{"seoul-service", "vm-01-9m4e2mr0ui3e8a215n4g"}, 
	//     ex) driverIID{"vm-01-9m4e2mr0ui3e8a215n4g", ""}
	spiderUUID, err := iidm.New(connectionName, rsType, reqInfo.IId.NameId)
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
		cblog.Error(err)
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

	if os.Getenv("EXPERIMENTAL_MINI_CACHE_SERVICE") == "ON" {
		if strings.HasPrefix(connectionName, "mini:imageinfo") {
			return listImageFromCache(connectionName)
		}
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


func listImageFromCache(connectName string) ([]*cres.ImageInfo, error) {
	cblog.Info("call listImageFromCache()")

        client := redis.NewClient(&redis.Options{
                Addr: "localhost:6379",
                Password: "",
                DB: 0,
        })

        //result, err := client.Get("imageinfo:aws:ohio").Result()
        result, err := client.Get(connectName).Result()
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        var jsonResult struct {
                Result []*cres.ImageInfo `json:"image"`
        }
        json.Unmarshal([]byte(result), &jsonResult)

        return jsonResult.Result, nil
}


// (1) get spiderIID:list
// (2) get CSP:list
// (3) filtering CSP-list by spiderIID-list
// Currently this API is not used. @TODO
func ListRegisterImage(connectionName string, rsType string) ([]*cres.ImageInfo, error) {
	cblog.Info("call ListRegisterImage()")

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

	// Currently this API is not used. @TODO
	//imgSPLock.RLock(connectionName, reqInfo.IId.NameId)
	//defer imgSPLock.RUnlock(connectionName, reqInfo.IId.NameId)
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
		err := fmt.Errorf("<IID-CSP mismatch> " + rsType + " IID List has " + strconv.Itoa(len(iidInfoList)) + ", but " + connectionName + " Resource list has nothing!")
		cblog.Error(err)
		return nil, err
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
// Currently this API is not used. @TODO
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

	imgSPLock.RLock(connectionName, nameID)
	defer imgSPLock.RUnlock(connectionName, nameID)
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
// Currently this API is not used. @TODO
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

	imgSPLock.Lock(connectionName, nameID)
	defer imgSPLock.Unlock(connectionName, nameID)
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
