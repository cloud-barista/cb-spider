// SPLock Manager of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2022.05.

package splock

import (
        "sync"
        "bytes"
        "fmt"
)


//====================================================================
type SPLOCK struct {
        rwMutex	sync.RWMutex	// lock for handling lockMap
	lockMap	map[LockKey]*LockValue
}

type LockKey struct {
        connectionName  string  // ex) "aws-seoul-config"
        resourceId      string  // ex) "VM-01"
}

type LockValue struct {
        lock	sync.RWMutex	// for id-based locking
        count	int		// reference counter for this lock
}
//====================================================================

func New() *SPLOCK {
	var spLock = new (SPLOCK)
	spLock.lockMap = make(map[LockKey]*LockValue)
	return spLock
}

func (spLock *SPLOCK)Lock(conn string, id string) {
spLock.rwMutex.Lock()
	lockValue := spLock.lockMap[LockKey{conn, id}]
	if lockValue == nil {
		spLock.lockMap[LockKey{conn, id}] = &LockValue{}
		lockValue = spLock.lockMap[LockKey{conn, id}]
	}
	lockValue.count++
spLock.rwMutex.Unlock()

	lockValue.lock.Lock()
}

func (spLock *SPLOCK)Unlock(conn string, id string) {
spLock.rwMutex.Lock()
        lockValue := spLock.lockMap[LockKey{conn, id}]
        lockValue.count--
	if lockValue.count == 0 {
		delete(spLock.lockMap, LockKey{conn, id})
	}
spLock.rwMutex.Unlock()

        lockValue.lock.Unlock()
}

func (spLock *SPLOCK)RLock(conn string, id string) {
spLock.rwMutex.Lock()

        lockValue := spLock.lockMap[LockKey{conn, id}]
        if lockValue == nil {
                spLock.lockMap[LockKey{conn, id}] = &LockValue{}
                lockValue = spLock.lockMap[LockKey{conn, id}]
        }
        lockValue.count++
spLock.rwMutex.Unlock()

        lockValue.lock.RLock()
}

func (spLock *SPLOCK)RUnlock(conn string, id string) {
spLock.rwMutex.Lock()
        lockValue := spLock.lockMap[LockKey{conn, id}]
        lockValue.count--
        if lockValue.count == 0 {
                delete(spLock.lockMap, LockKey{conn, id})
        }
spLock.rwMutex.Unlock()

        lockValue.lock.RUnlock()
}

func (spLock *SPLOCK)GetSPLockMapStatus(lockName string) string {

	var buff bytes.Buffer
	buff.WriteString("<" + lockName + "> ")

spLock.rwMutex.RLock()
	for k, v := range spLock.lockMap {
		buff.WriteString(fmt.Sprintf("(%s:%s, %p:%d) ", k.connectionName, k.resourceId, &v.lock, v.count))
	}
spLock.rwMutex.RUnlock()
	return buff.String()
}

