// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by CB-Spider Team, 2019.10.

package resources

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type KeyValue struct {
	Key   string `json:"Key" validate:"required" example:"key1"`
	Value string `json:"Value,omitempty" validate:"omitempty"  example:"value1"`
}

func StructToKeyValueList(obj interface{}) []KeyValue {
	var keyValueList []KeyValue

	// Get the reflect.Value of the object and its type
	val := reflect.ValueOf(obj)
	typ := val.Type()

	// If the obj is a pointer, dereference it
	if typ.Kind() == reflect.Ptr {
		val = val.Elem()
		typ = val.Type()
	}

	// If the obj is not a struct, return an empty list
	if typ.Kind() != reflect.Struct {
		return keyValueList
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		value := val.Field(i)

		if !field.IsExported() {
			continue
		}

		if isNilOrEmptyValue(value) {
			continue
		}

		// Use field name as key
		keyName := field.Name

		// Skip if json tag is "-"
		if jsonTag := field.Tag.Get("json"); jsonTag == "-" {
			continue
		}

		// Get the field value as a string
		var fieldValue string
		switch value.Kind() {
		case reflect.String:
			fieldValue = value.String()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			fieldValue = strconv.FormatInt(value.Int(), 10)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			fieldValue = strconv.FormatUint(value.Uint(), 10)
		case reflect.Float32, reflect.Float64:
			fieldValue = fmt.Sprintf("%.2f", value.Float())
		case reflect.Bool:
			fieldValue = strconv.FormatBool(value.Bool())
		case reflect.Slice:
			if value.Len() > 0 {
				sliceValues := make([]string, value.Len())
				for j := 0; j < value.Len(); j++ {
					jsonBytes, _ := json.Marshal(value.Index(j).Interface())
					sliceValues[j] = strings.ReplaceAll(string(jsonBytes), `"`, "")
					// sliceValues[j] = string(jsonBytes) ------------------------- // json format for 'Value' field
				}
				fieldValue = strings.Join(sliceValues, "; ")
			}
		default:
			// For all other types, try to marshal the value to JSON
			jsonBytes, err := json.Marshal(value.Interface())
			if err == nil {
				fieldValue = strings.ReplaceAll(string(jsonBytes), `"`, "")
				// fieldValue = string(jsonBytes) ------------------------- // json format for 'Value' field
			}
		}

		if fieldValue != "" {
			keyValueList = append(keyValueList, KeyValue{
				Key:   keyName,
				Value: fieldValue,
			})
		}
	}

	return keyValueList
}

func isNilOrEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}
