package resources

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/cloud-barista/cb-spider/test/permissionTest/connection"
	"github.com/cloud-barista/cb-spider/test/permissionTest/request"
)

type VMResource struct {}

type VMResponse struct {
	AccessPoint string `json:"AccessPoint"`
	IId         struct {
		NameId   string `json:"NameId"`
		SystemId string `json:"SystemId"`
	} `json:"IId"`
	ImageIId struct {
		NameId   string `json:"NameId"`
		SystemId string `json:"SystemId"`
	} `json:"ImageIId"`
	ImageType    string `json:"ImageType"`
	KeyPairIId   struct {
		NameId   string `json:"NameId"`
		SystemId string `json:"SystemId"`
	} `json:"KeyPairIId"`
	PublicIP    string `json:"PublicIP"`
	VMSpecName  string `json:"VMSpecName"`
	VMUserId    string `json:"VMUserId"`
	VMUserPasswd string `json:"VMUserPasswd"`
	VpcIID      struct {
		NameId   string `json:"NameId"`
		SystemId string `json:"SystemId"`
	} `json:"VpcIID"`
}

type VMListResponse struct {
	VMList []VMResponse `json:"vm"`
}

func (v VMResource) GetName() string {
	return "VM"
}

func (v VMResource) CreateResource (conn string, errChan chan<-error) error{
	connectionName := connection.Connections[conn].Name
	canWrite := connection.Connections[conn].CanWrite

	// ✅ VM을 생성할 VPC, Subnet, SecurityGroup, Keypair 매핑
	vmDependencies := map[string]map[string]string{
		"rw-conn": {
			"VPC":            "VPC-02",
			"Subnet":         "second-Subnet-01",
			"SecurityGroup": "security-rw-1",
			"KeyPair":        "keypair-rw-1",
		},
		"rw-conn2": {
			"VPC":            "VPC-01",
			"Subnet":         "second-Subnet-02",
			"SecurityGroup": "security-rw2-1",
			"KeyPair":        "keypair-rw2-1",
		},
		"readonly-conn": {
			"VPC":            "VPC-01",
			"Subnet":         "second-Subnet-02",
			"SecurityGroup": "security-rw2-1",
			"KeyPair":        "keypair-rw2-1",
		},
		"non-permission-conn": {
			"VPC":            "VPC-02",
			"Subnet":         "second-Subnet-01",
			"SecurityGroup": "security-rw-1",
			"KeyPair":        "keypair-rw-1",
		},
	}

	if _, exists := vmDependencies[conn]; !exists {
		return fmt.Errorf("[ERROR] %s has no valid VM dependency configuration", connectionName)
	}

	vmConfig := vmDependencies[conn]

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		data := map[string]interface{}{
			"ConnectionName":  connectionName,
			"IDTransformMode": "ON",
			"ReqInfo": map[string]interface{}{
				"ImageName":          os.Getenv("VM_IMAGE"),
				"ImageType":          "PublicImage",
				"KeyPairName":        vmConfig["KeyPair"],
				"Name":               fmt.Sprintf("%s-vm", conn),
				"SecurityGroupNames": []string{vmConfig["SecurityGroup"]},
				"SubnetName":         vmConfig["Subnet"],
				"VMSpecName":         os.Getenv("VM_SPEC"),
				"VPCName":            vmConfig["VPC"],
			},
		}

		url := fmt.Sprintf("%s/vm", baseURL)

		statusCode, response, err := request.SendRequest("POST", url, data)
		if err != nil {
			errChan <- fmt.Errorf("[ERROR] %s failed to create VM: %v", connectionName, err)
			return
		}
	
		if statusCode == 500 {
			if !canWrite {
				fmt.Printf("[PERMISSION-SUCCESS] %s was denied VM creation as expected ✅\n", connectionName)
				return
			}
			errChan <- fmt.Errorf("[ERROR] %s received 500 error while creating VM", connectionName)
			return
		}

		if statusCode == 200 && canWrite {
			// ✅ Response 파싱
			var vmResponse struct {
				PublicIP string `json:"PublicIP"`
				IId      struct {
					NameId   string `json:"NameId"`
					SystemId string `json:"SystemId"`
				} `json:"IId"`
			}

			err = json.Unmarshal(response, &vmResponse)
			if err != nil {
				errChan <- fmt.Errorf("[ERROR] %s failed to parse VM response: %v", connectionName, err)
				return
			}

			// ✅ 성공 메시지 출력
			fmt.Printf("[SUCCESS] %s created VM %s ✅\n", connectionName, vmResponse.IId.NameId)
			fmt.Printf("🌐 Public IP: %s\n", vmResponse.PublicIP)
		} else {
			errChan <- fmt.Errorf("[ERROR] %s received unexpected status code %d while creating VM", connectionName, statusCode)
		}
	}()

	wg.Wait()
	return nil
} 

// Read VM
func (v VMResource) ReadResource(conn string, errChan chan<- error) error {
	connectionName := connection.Connections[conn].Name
	canRead := connection.Connections[conn].CanRead

	url := fmt.Sprintf("%s/vm", baseURL)
	data := map[string]interface{}{
		"ConnectionName": connectionName,
	}


	statusCode, response, err := request.SendRequest("GET", url, data)
	if err != nil {
		return fmt.Errorf("[ERROR] %s failed to read VM list: %v", connectionName, err)
	}

	if statusCode == 500 {
		if !canRead {
			fmt.Printf("[PERMISSION-SUCCESS] %s was denied VM read as expected ✅\n", connectionName)
			return nil
		} else {
			return fmt.Errorf("[ERROR] %s received 500 error while reading VM list", connectionName)
		}
	}

	var vmListResponse struct {
		VMList []struct {
			IId struct {
				NameId   string `json:"NameId"`
				SystemId string `json:"SystemId"`
			} `json:"IId"`
		} `json:"vm"`
	}

	err = json.Unmarshal(response, &vmListResponse)
	if err != nil {
		return fmt.Errorf("[ERROR] %s failed to parse VM response: %v", connectionName, err)
	}

	expectedVMs := map[string]bool{
		"rw-conn-vm":  false,
		"rw-conn2-vm": false,
	}

	for _, vm := range vmListResponse.VMList {
		if _, exists := expectedVMs[vm.IId.NameId]; exists {
			expectedVMs[vm.IId.NameId] = true
		}
	}

	allFound := true
	for vm, found := range expectedVMs {
		if !found {
			errChan <- fmt.Errorf("[ERROR] %s could not find expected VM: %s", connectionName, vm)
			allFound = false
		}
	}

	if allFound {
		fmt.Printf("[SUCCESS] %s retrieved all expected VMs ✅\n", connectionName)
	} else {
		return fmt.Errorf("[ERROR] %s did not retrieve all expected VMs", connectionName)
	}

	return nil
}

// Delete VM
func (v VMResource) DeleteResource(conn string, errChan chan<- error) error {
	connectionName := connection.Connections[conn].Name
	canWrite := connection.Connections[conn].CanWrite

	vmNames := map[string]string{
		"rw-conn":  "rw-conn2-vm",
		"rw-conn2": "rw-conn-vm",
		"readonly-conn": "rw-conn-vm", 
		"non-permission-conn": "rw-conn2-vm",  
	}

	vmName, exists := vmNames[conn]
	if !exists {
		fmt.Printf("[ERROR] %s has no VM assigned for deletion \n", connectionName)
		return nil
	}

	url := fmt.Sprintf("%s/vm/%s", baseURL, vmName)
	data := map[string]interface{}{
		"ConnectionName": connectionName,
	}

	statusCode, _, err := request.SendRequest("DELETE", url, data)
	if err != nil {
		return fmt.Errorf("[ERROR] %s failed to delete VM %s: %v", connectionName, vmName, err)
	}

	if statusCode == 500 {
		if !canWrite {
			fmt.Printf("[PERMISSION-SUCCESS] %s was denied VM deletion as expected ✅\n", connectionName)
			return nil
		} else {
			return fmt.Errorf("[ERROR] %s received 500 error while deleting VM %s", connectionName, vmName)
		}
	}
	if statusCode == 200 && canWrite {
		fmt.Printf("[SUCCESS] %s deleted VM %s ✅\n", connectionName, vmName)
	} else {
		return fmt.Errorf("[ERROR] %s received unexpected status code %d while deleting VM %s", connectionName, statusCode, vmName)
	}

	return nil
}




