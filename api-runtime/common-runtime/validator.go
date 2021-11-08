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
	//icbs "github.com/cloud-barista/cb-store/interfaces"
	//ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
)

func EmptyCheckAndTrim(inputName string, inputValue string) (string, error) {

	if inputValue == "" {
                return "", fmt.Errorf(inputName + " is empty!")
        }
        // trim user inputs
        return strings.TrimSpace(inputValue), nil
}

func ValidateStruct(is interface{}, nilPermissionList []string) error {
	inValue := reflect.ValueOf(is)
	if inValue.Kind() == reflect.Ptr {
		inValue = inValue.Elem() // When Input is ptr of Struct, Get the element.
	}

	var retErr error
	for i := 0; i < inValue.NumField(); i++ {
		fv := inValue.Field(i)
		switch fv.Kind() {
		case reflect.Struct, reflect.Ptr:
			err := ValidateStruct(fv.Interface(), nilPermissionList)
			if err != nil {
				if retErr != nil {
					retErr = fmt.Errorf("%v\n%v", retErr, err)
				}else {
					retErr = err
				}
			}
		case reflect.Slice:
			for j := 0; j < fv.Len(); j++ {
				err := ValidateStruct(fv.Index(j).Interface(), nilPermissionList)
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
				err := checkNilPermission(argNameType, nilPermissionList)
				if err != nil {
					if retErr != nil {
						retErr = fmt.Errorf("%v\n%v", retErr, err)
					}else {
						retErr = err
					}
				}
			}

		default:
			fmt.Println("=========== other type: ", inValue.Field(i).Kind())
		}
	}
	return retErr
}

// Check the arguments that can be used as empty.
func checkNilPermission(argTypeName string, nilPermissionList []string) error {

	for _, permittedArgTypeName := range(nilPermissionList) {
		if permittedArgTypeName == argTypeName {
			return nil
		}
	}
	return fmt.Errorf("%v's input value is empty!", argTypeName)
}

/*
// (1) Extract and sort the list of keys from inKeyValue,
// (2) Get the list of keys from driver, and sort it.
// (3) compare them.
func ValidateKeyValue(inKeyValueList []icbs.KeyValue, keyList []string, notNullNameList []string) error {

	ccm.GetCloudDriver(
	// (1) Extract the list of keys from inKeyValue and sort it.
	for  _, kv := range inKeyValueList {
		for _, key := range keyList {
			if kv.Key == key {
				keyList = keyList.remove(key)
			}
		}
	}
	// (2) Get the list of keys from driver, and sort it.

	// (3) compare them.
	return nil
}


func getCloudDriver(providerName string) , error) {
        cblog.Info("CloudDriverHandler: called getStaticCloudDriver() - " + cldDrvInfo.DriverName)

        var cloudDriver idrv.CloudDriver

        // select driver
        switch cldDrvInfo.ProviderName {
        case "AWS":
                cloudDriver = new(awsdrv.AwsDriver)
        case "AZURE":
                cloudDriver = new(azuredrv.AzureDriver)
        case "GCP":
                cloudDriver = new(gcpdrv.GCPDriver)
        case "ALIBABA":
                cloudDriver = new(alibabadrv.AlibabaDriver)
        case "OPENSTACK":
                cloudDriver = new(openstackdrv.OpenStackDriver)
        case "CLOUDIT":
                cloudDriver = new(clouditdrv.ClouditDriver)
        case "DOCKER":
                cloudDriver = new(dockerdrv.DockerDriver)
        case "TENCENT":
                cloudDriver = new(tencentdrv.TencentDriver)
        // case "NCP": // NCP
        //  cloudDriver = new(ncpdrv.NcpDriver) // NCP
        // case "NCPVPC": // NCP-VPC
        //  cloudDriver = new(ncpvpcdrv.NcpVpcDriver) // NCP-VPC
        case "MOCK":
                cloudDriver = new(mockdrv.MockDriver)

        default:
                errmsg := cldDrvInfo.ProviderName + " is not supported static Cloud Driver!!"
                return cloudDriver, fmt.Errorf(errmsg)
        }

        return cloudDriver, nil
}

*/


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
