package resources

import (
	"encoding/json"
	"fmt"

	"github.com/cloud-barista/cb-spider/permissionTest/connection"
	"github.com/cloud-barista/cb-spider/permissionTest/request"
)

type VPCResource struct{}

// VPC 단일 응답 구조체
type VPCResponse struct {
    IId struct {
        NameId   string `json:"NameId"`
        SystemId string `json:"SystemId"`
    } `json:"IId"`
    IPv4_CIDR      string `json:"IPv4_CIDR"`
    SubnetInfoList []struct {
        IId struct {
            NameId   string `json:"NameId"`
            SystemId string `json:"SystemId"`
        } `json:"IId"`
        IPv4_CIDR string `json:"IPv4_CIDR"`
    } `json:"SubnetInfoList"`
}

// 전체 VPC 목록 응답 구조체
type VPCListResponse struct {
    VPCList []VPCResponse `json:"vpc"`
}

func (v VPCResource) GetName() string {
	return "VPC"
}

var baseURL = "http://localhost:1024/spider"
var vpcNames = map[string]string{
	"rw-conn": "VPC-01",
	"rw-conn2": "VPC-02",
	"readonly-conn": "VPC-03",
	"non-permission-conn": "VPC-04",
}

// create VPC
func (v VPCResource) CreateResource(conn string, errChan chan<- error) error {
    connectionName := connection.Connections[conn].Name
    canWrite := connection.Connections[conn].CanWrite
    vpcName := vpcNames[conn]

    url := baseURL + "/vpc"
    data := map[string]interface{}{
        "ConnectionName": connectionName,
        "ReqInfo": map[string]interface{}{
            "Name":      vpcName,
            "IPv4_CIDR": "10.0.0.0/16",
            "SubnetInfoList": []map[string]interface{}{
                {"Name": "Subnet-01", "IPv4_CIDR": "10.0.1.0/24"},
            },
        },
    }

    statusCode, _, err := request.SendRequest("POST", url, data)
    if err != nil {
        return fmt.Errorf("[ERROR] %s failed to create VPC: %v", connectionName, err)
    }

    if statusCode == 200 && canWrite {
        fmt.Printf("[SUCCESS] %s created VPC\n", connectionName)
    } else if statusCode == 500 && !canWrite {
        fmt.Printf("[PERMISSION-SUCCESS] %s was denied VPC creation as expected\n", connectionName)
    } else {
        return fmt.Errorf("[ERROR] %s received unexpected status code %d while creating VPC", connectionName, statusCode)
    }

    return nil
}

// read VPC
func (v VPCResource) ReadResource(conn string, errChan chan<- error) error {
    connectionName := connection.Connections[conn].Name
    canRead := connection.Connections[conn].CanRead

    url := fmt.Sprintf("%s/vpc", baseURL)
    data := map[string]interface{}{
        "ConnectionName": connectionName,
    }

    statusCode, response, err := request.SendRequest("GET", url, data)
    if err != nil {
        return fmt.Errorf("[ERROR] %s failed to read VPCs: %v", connectionName, err)
    }

    if statusCode == 500 {
        if canRead {
            return fmt.Errorf("[ERROR] %s received 500 error while reading VPCs", connectionName)
        } else {
            fmt.Printf("[PERMISSION-SUCCESS] %s was denied VPC read as expected\n", connectionName)
            return nil
        }
    }

    var vpcListResponse VPCListResponse
    err = json.Unmarshal(response, &vpcListResponse)
    if err != nil {
        return fmt.Errorf("[ERROR] %s failed to parse VPC response: %v", connectionName, err)
    }

    expectedVPCs := map[string]bool{
        "VPC-01": false,
        "VPC-02": false,
    }

    for _, vpc := range vpcListResponse.VPCList {
        if _, exists := expectedVPCs[vpc.IId.NameId]; exists {
            expectedVPCs[vpc.IId.NameId] = true
        }
    }

    allFound := true
    for vpc, found := range expectedVPCs {
        if !found {
            errChan <- fmt.Errorf("[ERROR] %s could not find expected VPC: %s", connectionName, vpc)
            allFound = false
        }
    }

    if allFound {
        fmt.Printf("[SUCCESS] %s retrieved all expected VPCs\n", connectionName)
    } else {
        return fmt.Errorf("[ERROR] %s did not retrieve all expected VPCs", connectionName)
    }

    return nil
}

// Delete VPC
func (v VPCResource) DeleteResource(conn string, errChan chan<- error) error {
    connectionName := connection.Connections[conn].Name
    canWrite := connection.Connections[conn].CanWrite

    expectedVPCs := map[string]bool{}

    switch conn {
    case "rw-conn":
        expectedVPCs["VPC-02"] = false
    case "rw-conn2":
        expectedVPCs["VPC-01"] = false
    default:
        expectedVPCs["VPC-01"] = false
        expectedVPCs["VPC-02"] = false
    }

    for vpcName := range expectedVPCs {
        url := fmt.Sprintf("%s/vpc/%s", baseURL, vpcName)
        data := map[string]interface{}{
            "ConnectionName": connectionName,
        }

        statusCode, _, err := request.SendRequest("DELETE", url, data)
        if err != nil {
            errChan <- fmt.Errorf("[ERROR] %s failed to delete VPC %s: %v", connectionName, vpcName, err)
            continue
        }

        if canWrite && statusCode == 200 {
            expectedVPCs[vpcName] = true
        } else if canWrite && statusCode == 500 {
            errChan <- fmt.Errorf("[ERROR] %s received 500 error while deleting VPC %s", connectionName, vpcName)
        } else if !canWrite && statusCode == 200 {
            errChan <- fmt.Errorf("[PERMISSION ERROR] %s was able to delete VPC %s despite lacking permission", connectionName, vpcName)
        } else {
            fmt.Printf("[PERMISSION-SUCCESS] %s was denied VPC deletion as expected\n", connectionName)
            expectedVPCs[vpcName] = true
        }
    }

    allPassed := true
    for vpcName, pass := range expectedVPCs {
        if !pass {
            errChan <- fmt.Errorf("[ERROR] %s failed to delete VPC %s", connectionName, vpcName)
            allPassed = false
        }
    }

    if allPassed {
        fmt.Printf("[SUCCESS] %s successfully deleted all expected VPCs\n", connectionName)
    } else {
        return fmt.Errorf("[FINAL ERROR] %s encountered issues during VPC deletions", connectionName)
    }

    return nil
}
