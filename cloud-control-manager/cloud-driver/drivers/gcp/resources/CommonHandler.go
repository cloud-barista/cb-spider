package resources

import (
	"fmt"

	irs "../../../interfaces/resources"
)

func GetKeyValueList(i map[string]interface{}) []irs.KeyValue {
	var keyValueList []irs.KeyValue
	for k, v := range i {
		keyValueList = append(keyValueList, irs.KeyValue{k, v.(string)})
		fmt.Println("getKeyValueList : ", keyValueList)
	}

	return keyValueList
}
