// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2021.10.

package commonruntime

import (
	"fmt"
	"reflect"
	"strings"
)

func EmptyCheckAndTrim(inputName string, inputValue string) (string, error) {

	if inputValue == "" {
                return "", fmt.Errorf(inputName + " is empty!")
        }
        // trim user inputs
        return strings.TrimSpace(inputValue), nil
}

func ValidateStruct(is interface{}, emptyPermissionList []string) error {
	var retErr error

	inValue := reflect.ValueOf(is)
	if inValue.Kind() == reflect.Ptr {
		inValue = inValue.Elem() // When Input is ptr of Struct, Get the element.
		if inValue.Kind() == reflect.Slice { //  When ptr of Array, ex) SecurityRules *[]SecurityRuleInfo
			for j := 0; j < inValue.Len(); j++ {
                                err := ValidateStruct(inValue.Index(j).Interface(), emptyPermissionList)
                                if retErr != nil {
                                        retErr = fmt.Errorf("%v\n%v", retErr, err)
                                }else {
                                        retErr = err
                                }
                        }

		}
	}


	for i := 0; i < inValue.NumField(); i++ {
		fv := inValue.Field(i)
		switch fv.Kind() {
		case reflect.Struct, reflect.Ptr:
			err := ValidateStruct(fv.Interface(), emptyPermissionList)
			if err != nil {
				if retErr != nil {
					retErr = fmt.Errorf("%v\n%v", retErr, err)
				}else {
					retErr = err
				}
			}
		case reflect.Slice:
			for j := 0; j < fv.Len(); j++ {
				err := ValidateStruct(fv.Index(j).Interface(), emptyPermissionList)
				if retErr != nil {
					retErr = fmt.Errorf("%v\n%v", retErr, err)
				}else {
					retErr = err
				}
			}

		case reflect.String:
			//fmt.Printf("========== %v:%v=%v\n", inValue.Type(), inValue.Type().Field(i).Name, inValue.Field(i).Interface())
			//fmt.Println("-----------: " + argNameType)
			argNameType := fmt.Sprintf("%v:%v", inValue.Type(), inValue.Type().Field(i).Name)
			if inValue.Field(i).Interface() == "" {
				err := checkNilPermission(argNameType, emptyPermissionList)
				if err != nil {
					if retErr != nil {
						retErr = fmt.Errorf("%v\n%v", retErr, err)
					}else {
						retErr = err
					}
				}
			}

		default:
			cblog.Info("=========== Currently, Unhandling Type: ", inValue.Field(i).Kind())
			//fmt.Println("=========== other type: ", inValue.Field(i).Kind())
		}
	}
	return retErr
}

// Check the arguments that can be used as empty.
func checkNilPermission(argTypeName string, emptyPermissionList []string) error {

	for _, permittedArgTypeName := range(emptyPermissionList) {
		if permittedArgTypeName == argTypeName {
			return nil
		}
	}
	return fmt.Errorf("%v's input value is empty!", argTypeName)
}

//----------- utility

func printType(inType reflect.Type) {
        fmt.Println("\n[print Type]")
        fmt.Println("   Type.Name(): ", inType.Name())
        fmt.Println("   Type.Size(): ", inType.Size())
        fmt.Println("   Type.Kind(): ", inType.Kind())
        fmt.Println("   Type: ", inType)
        fmt.Println("---------------------------")
}

func printValue(inValue reflect.Value) {
        fmt.Println("\n[print Value]")
        fmt.Println("   Value.Type(): ", inValue.Type())
        fmt.Println("   Value.Kind(): ", inValue.Kind())
        if inValue.Kind() == reflect.Struct {
                fmt.Println("   Value.NumField(): ", inValue.NumField())
        }
        if inValue.Kind() == reflect.Ptr {
                fmt.Println("   Value.Elem(): ", inValue.Elem())
        }
        fmt.Println("   Value: ", inValue)
        fmt.Println("---------------------------")
}
