// Spider-Mini Server Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2021.11.

package miniserver

import (
	"github.com/sirupsen/logrus"
	gc "github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/common"
	cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	"github.com/go-redis/redis"
        log "github.com/cloud-barista/cb-log"
)

var cblog *logrus.Logger

const (
	rsImage string =  "image"
)

func init() {
        cblog = log.GetLogger("CB-SPIDER")
}
func Cloner() error {
	// (1) get a region from the CacheTargetList
	// (2) cloning from that region
	// (3) insert cloned meta info into db
	// (4) delete the cloned region in the CacheTargetList

	cacheTargetInfo := GetNSet()

	strImageList, err := getImageListFromCSP(cacheTargetInfo.connectName)
	if err != err {
		cblog.Error(err)
		return err
	}

	err = insertImageInfoListToDB(cacheTargetInfo.connectName, strImageList)
	if err != err {
		cblog.Error(err)
		return err
	}

	cblog.Info(strImageList)

	return nil
}

func getImageListFromCSP(connectionName string) (string, error) {
        // Call common-runtime API
        result, err := cmrt.ListImage(connectionName, rsImage)
        if err != nil {
		cblog.Error(err)
                return "", err
        }

	var jsonResult struct {
                Result []*cres.ImageInfo `json:"image"`
        }
	jsonResult.Result = result

        return gc.ConvertToOutput("json", &jsonResult)
}

func insertImageInfoListToDB(connectionName string, strImageList string) error {
    client := redis.NewClient(&redis.Options{
                Addr: "localhost:6379",
                Password: "",
                DB: 0,
    })

    err := client.Set(connectionName, strImageList, 0).Err()
    if err != nil {
	cblog.Error(err)
	return err
    }

    return nil
}
