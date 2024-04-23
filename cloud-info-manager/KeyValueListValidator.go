// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2021.11.

package cloudos

import (
	"fmt"
	"strings"

	icdrs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

// (1) Remove list of valid keys with list of inpput keys
// (2) Check the list of remaining Keys
func ValidateKeyValueList(inKeyValueList []icdrs.KeyValue, validKeyList []string) error {

	clonedKeyList := cloneSlice(validKeyList)

	// (1) Remove list of valid keys with list of inpput keys
	inputKeyList := make([]string, len(inKeyValueList))
	for idx, kv := range inKeyValueList {
		inputKeyList[idx] = kv.Key
		for _, key := range clonedKeyList {
			if strings.EqualFold(kv.Key, key) { // ignore case
				clonedKeyList = removeSlice(clonedKeyList, key)
			}
		}
	}
	// (2) Check the list of remaining Keys
	if len(clonedKeyList) == 0 {
		return nil
	} else {
		errMSG := fmt.Sprintf("Invalid Key in input arguments.\n\t...... have %v\n\t...... want %v", inputKeyList, validKeyList)
		return fmt.Errorf(errMSG)
	}
}

func removeSlice(inSlice []string, deleteValue string) []string {
	for idx, v := range inSlice {
		if v == deleteValue {
			inSlice = append(inSlice[:idx], inSlice[idx+1:]...)
			return inSlice
		}
	}
	return inSlice
}

func cloneSlice(inSlice []string) []string {
	clonedSlice := make([]string, len(inSlice))
	copy(clonedSlice, inSlice)
	return clonedSlice
}
