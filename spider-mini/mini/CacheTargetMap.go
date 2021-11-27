// Spider-Mini CacheTargetListHandler of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2021.11.

package miniserver

import (
	"time"
	"sync"
)

type METAINFOTYPE string
const (
        IMAGEINFO METAINFOTYPE = "imageinfo"
        SPECINFO METAINFOTYPE = "specinfo"
)

type CacheTargetInfo struct {
        cloneName         string
        connectName     string
        metaInfoType       METAINFOTYPE
	occupiedStatus    bool
	occupiedCount      int
	firstOccupiedTime  time.Time
}

var cacheTargetInfoMap map[string]*CacheTargetInfo

// for locking
var rwLock sync.RWMutex

func init() {
	cacheTargetInfoMap = make(map[string]*CacheTargetInfo)
}

func Add(cloneName string, connectName string, metaInfoType METAINFOTYPE) {
	one := CacheTargetInfo{
		cloneName:        cloneName,
		connectName:       connectName,
		metaInfoType:      metaInfoType,
		occupiedStatus:    false,
		occupiedCount:     0,
	}
	rwLock.Lock()
	cacheTargetInfoMap[cloneName] = &one
	rwLock.Unlock()
}

func GetNSet() (*CacheTargetInfo) {
	// (1) find the first target that is cache WANTED status
	// (2) set target with OCCUPIED status and return it, count up it's occupied Count

	// locking
	rwLock.Lock()
	defer rwLock.Unlock()

	for _, v := range cacheTargetInfoMap {
		if v.occupiedStatus == false {
			if v.occupiedCount == 0 {
				v.firstOccupiedTime = time.Now()
			}
			v.occupiedCount += 1
			v.occupiedStatus = true
			return v
		}
	}

	return nil
}

func Del(cloneName string) {
	rwLock.Lock()
	delete(cacheTargetInfoMap, cloneName)
	rwLock.Unlock()

}

func Count() int {
	// locking
	rwLock.Lock()
	defer rwLock.Unlock()

	return len(cacheTargetInfoMap)
}

// @TBD
func CheckNSetWantedStatus() {
	// (1) find long-term OCCUPIED targets
	// (2) reset them for more trial and count up their occupied Count
}
