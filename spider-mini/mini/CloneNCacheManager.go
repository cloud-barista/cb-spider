// Spider-Mini Server Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2021.11.

package miniserver

import (
	"sync"
	"os"
	"strconv"
	"time"
	"strings"
	"github.com/sirupsen/logrus"
	cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager/connection-config-info-manager"

	"github.com/go-redis/redis"
	"encoding/json"

        log "github.com/cloud-barista/cb-log"
)

var cblog *logrus.Logger

const (
	rsImage string =  "image"
)

var startClone chan bool

func init() {
        cblog = log.GetLogger("CB-SPIDER")
	startClone = make(chan bool)
}

func RunServer() {
	// (1) create the CacheTargetInfoMap
	// (2) fork Cloners
	// (3) wait until next turn
	// (4) manage cloning(reclone or kill clone, ...) @TBD

	strCloneInterval := os.Getenv("EXPERIMENTAL_MINI_CLONE_INTERVAL")
	if strCloneInterval == "" {
		cblog.Info("$EXPERIMENTAL_MINI_CLONE_INTERVAL is not set!!")
		strCloneInterval = "36000" // default: 10H
	}
	intCloneInterval, err := strconv.Atoi(strCloneInterval)
	if err != nil {
		cblog.Error(err)
	}
	cloneInterval := time.Duration(intCloneInterval)

	strMAX_CLONER := os.Getenv("EXPERIMENTAL_MINI_MAX_CLONER")
	if strMAX_CLONER == "" {
                cblog.Info("$EXPERIMENTAL_MINI_MAX_CLONER is not set!!")
                strMAX_CLONER = "10" // default: 10
        }
	MAX_CLONER, err := strconv.Atoi(strMAX_CLONER)
	if err != nil {
		cblog.Error(err)
	}

	go func() { // timer

		for {
			time.Sleep(time.Second * cloneInterval)
			startClone <- true  // wake up
		}
	}()

	for {
		makeCacheTargetinfoMap()

		clonerNum := Count() // CacheTargetInfo number
		if clonerNum > MAX_CLONER {
			clonerNum = MAX_CLONER
		}

		wg := new(sync.WaitGroup)
		for i:=0 ; i<clonerNum ; i++ {
			wg.Add(1)
			go Cloner(wg)
		}
		wg.Wait() // wait until all cloned

		<-startClone // wait calling
	}

}

func makeCacheTargetinfoMap() {
	// (1) Get registerd ConnectionName List for Clone // format) mini:imageinfo:aws:ap-northeast-2:ap-northeast-2a
	// (2) create the CacheTargetInfoMap with the 'mini:' prefix connection
	ccList, err := cim.ListConnectionConfig()
        if err != nil {
                cblog.Error(err)
        }

        for _, v := range ccList {
		AddCacheTargetInfo(v.ConfigName)
        }
}

// when user add new targetRegion
func InsertNewCloneNCache(connectName string) {
	AddCacheTargetInfo(connectName)
	startClone <- true
}

func AddCacheTargetInfo(connectName string) {
	if strings.HasPrefix(connectName, "mini:") {
		cloneName := connectName + "-1"
		metaInfoType :=  strings.Split(connectName, ":")[1]
		Add(cloneName, connectName, METAINFOTYPE(metaInfoType))
	}
}

func Cloner(wg *sync.WaitGroup) error {
	// (1) get a CacheTargetInfo(about Region info) from the CacheTargetList
	// (2) cloning from that region
	// (3) insert cloned meta info into db
	// (4) delete the cloned CacheTargetInfo in the CacheTargetList

	defer wg.Done()

	for {
		cacheTargetInfo := GetNSet()
		if cacheTargetInfo == nil {
			return nil
		}

		cblog.Info("\n====================== Cloning: ", cacheTargetInfo)

		byteImageList, err := getImageListFromCSP(cacheTargetInfo.connectName)
		if err != err {
			cblog.Error(err)
			cblog.Error("\n====================== Can not cache: ", cacheTargetInfo)
			Del(cacheTargetInfo.cloneName)
			continue
		}

		err = insertImageInfoListToDB(cacheTargetInfo.connectName, byteImageList)
		if err != err {
			cblog.Error(err)
			cblog.Error("\n====================== Can not cache: ", cacheTargetInfo)
			Del(cacheTargetInfo.cloneName)
			continue
		}

		cblog.Info("\n====================== Cached: ", cacheTargetInfo)

		Del(cacheTargetInfo.cloneName)
	}

	return nil
}

func getImageListFromCSP(connectionName string) ([]byte, error) {
        // Call common-runtime API
        result, err := cmrt.ListImage(connectionName, rsImage)
        if err != nil {
		cblog.Error(err)
                return nil, err
        }

	var jsonResult struct {
                Result []*cres.ImageInfo `json:"image"`
        }
	jsonResult.Result = result

        return json.Marshal(jsonResult)
}

func insertImageInfoListToDB(connectionName string, byteImageList []byte) error {
    client := redis.NewClient(&redis.Options{
                Addr: "localhost:6379",
                Password: "",
                DB: 0,
    })

    err := client.Set(connectionName, byteImageList, 0).Err()
    if err != nil {
	cblog.Error(err)
	return err
    }

    return nil
}
