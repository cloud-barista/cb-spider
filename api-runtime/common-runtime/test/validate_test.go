// Mock Driver Test of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2021.10.

package validatetest

import (
	valid "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"

	"testing"
	"fmt"
)

func TestEmpty(t *testing.T) {

	inputValue := ""

	inputValue, err := valid.EmptyCheckAndTrim("inputValue", inputValue)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Printf("Trimed input value: %v.\n", inputValue)
}

func TestEmptyCheckAndTrim(t *testing.T) {

        inputValue := "inputValue "

	inputValue, err := valid.EmptyCheckAndTrim("inputValue", inputValue)
        if err != nil {
                fmt.Println(err.Error())
        }
        fmt.Printf("Trimed input value: %v.\n", inputValue)
}

func TestValidate(t *testing.T) {
        type InnerTestStruct struct {
                Depth2_a string
                Depth2_b string
        }

        type TestStruct struct {
                Depth1_a string
                Depth1_b string
                Depth1_s *InnerTestStruct
        }


        var ts = TestStruct{
                "name01",
                "name02",
                &InnerTestStruct{"name03", "name04"},
        }

        err := valid.ValidateStruct(ts, nil)
        err = valid.ValidateStruct(&ts, nil)
        if err != nil {
                fmt.Println(err.Error())
        }
}

func TestValidateNilByFail(t *testing.T) {
        type InnerTestStruct struct {
                Depth2_a string
                Depth2_b string
        }

        type TestStruct struct {
                Depth1_a string
                Depth1_b string
                Depth1_s *InnerTestStruct
        }


        var ts = TestStruct{
		Depth1_a: "name01",
		Depth1_s: &InnerTestStruct{"name03", ""},
        }

        err := valid.ValidateStruct(ts, nil)
        if err != nil {
                fmt.Println(err.Error())
        }
}

func TestValidateNilByPass(t *testing.T) {
        type InnerTestStruct struct {
                Depth2_a string
                Depth2_b string
        }

        type TestStruct struct {
                Depth1_a string
                Depth1_b string
                Depth1_s *InnerTestStruct
        }


        var ts = TestStruct{
                Depth1_a: "name01",
                Depth1_s: &InnerTestStruct{"name03", ""},
        }

	err := valid.ValidateStruct(ts, []string{"validatetest.TestStruct:Depth1_b", "validatetest.InnerTestStruct:Depth2_b"})
        if err != nil {
                fmt.Println(err.Error())
        }
}


