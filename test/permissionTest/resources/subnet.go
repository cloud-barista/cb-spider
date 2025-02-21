package resources

import (
	"encoding/json"
	"fmt"

	"github.com/cloud-barista/cb-spider/test/permissionTest/connection"
	"github.com/cloud-barista/cb-spider/test/permissionTest/request"
)

type SubnetResource struct{}

type SubnetResponse struct {
    IId struct {
        NameId   string `json:"NameId"`
        SystemId string `json:"SystemId"`
    } `json:"IId"`
    IPv4_CIDR  string `json:"IPv4_CIDR"`
    Zone       string `json:"Zone"`
    TagList    []struct {
        Key   string `json:"Key"`
        Value string `json:"Value"`
    } `json:"TagList"`
}

type SubnetListResponse struct {
    SubnetList []SubnetResponse `json:"subnet"`
}

// Subnet Creation name
var subnetNames = map[string]string{
	"rw-conn": "second-Subnet-01",
	"rw-conn2": "second-Subnet-02",
	"readonly-conn": "second-Subnet-03",
	"non-permission-conn": "second-Subnet-04",
}

func (s SubnetResource) GetName() string {
	return "Subnet"
}

// mapping subnet to target vpc
var subnetToVpc = map[string][]string {
    "rw-conn": {"VPC-02"},
    "rw-conn2": {"VPC-01"},
    "readonly-conn": {"VPC-01", "VPC-02"},
    "non-permission-conn": {"VPC-01", "VPC-02"},
}


// Create Subnet 
func (s SubnetResource) CreateResource(conn string, errChan chan<- error) error {
    connectionName := connection.Connections[conn].Name
    subnetName := subnetNames[conn]
	canWrite := connection.Connections[conn].CanWrite

    vpcList := subnetToVpc[conn]

    for _, vpcName := range vpcList{

        url := fmt.Sprintf("%s/vpc/%s/subnet", baseURL, vpcName)
        data := map[string]interface{}{
            "ConnectionName": connectionName,
            "IDTransformMode": "ON",
            "ReqInfo": map[string]interface{}{
                "Name":      subnetName,
                "IPv4_CIDR": "10.0.2.0/24",
            },
        }
        
        statusCode, _, err := request.SendRequest("POST", url, data)
        if err != nil {
            return fmt.Errorf("[ERROR] %s failed to create Subnet: %v", connectionName, err)
        }

        if statusCode == 200 && canWrite {
            fmt.Printf("[SUCCESS] %s created Subnet %s in %s✅\n", connectionName, subnetName, vpcName)
        } else if statusCode == 500 && !canWrite{
            fmt.Printf("[PERMISSION-SUCCESS] %s was denied Subnet creation in %s as expected✅\n", connectionName, vpcName)
        } else {
            return fmt.Errorf("[ERROR] %s received unexpected status code %d while creating Subnet in %s", connectionName, statusCode, vpcName)
        }
    }
    return nil
}

//  read Subnet
func (s SubnetResource) ReadResource(conn string, errChan chan<- error) error {
    connectionName := connection.Connections[conn].Name
    canRead := connection.Connections[conn].CanRead

    vpcList := []string{"VPC-01", "VPC-02"}

    for _, vpcName := range vpcList{
        url := fmt.Sprintf("%s/vpc/%s", baseURL, vpcName)
        data := map[string]interface{}{
            "ConnectionName":connectionName,
        }

        statusCode, response, err := request.SendRequest("GET", url, data)
        if err != nil{
            return fmt.Errorf("[ERROR] %s failed to read VPC in Read Subnet: %v", connectionName, err)
        }

        if statusCode == 500{
            if canRead{
                return fmt.Errorf("[ERROR] %s received 500 error while reading VPC in Read Subnet", connectionName)
            } else {
                fmt.Printf("[PERMISSION-SUCCESS] %s was denied VPC read in Read Subnet as expected✅n", connectionName)
                return nil
            }
        }

        var vpcResponse VPCResponse
        err = json.Unmarshal(response, &vpcResponse)
        if err != nil{
            return fmt.Errorf("[ERROR] %s failed to parse VPC response: %v", connectionName, err)
        }

        subnetCount := len(vpcResponse.SubnetInfoList)
        if subnetCount != 2{
            return fmt.Errorf("[ERROR] %s received unexpected count of Subnet in VPC(expected = 2): %d", connectionName, subnetCount)
        }
        fmt.Printf("[SUCCESS] %s read Subnet in %s✅\n", connectionName, vpcName)
    }

    return nil
}

// Delete Subnet 
func (s SubnetResource) DeleteResource(conn string, errChan chan<- error) error {
    connectionName := connection.Connections[conn].Name
    canWrite := connection.Connections[conn].CanWrite

    toRemove := map[string]string{
        "rw-conn": "second-Subnet-02",
        "rw-conn2": "second-Subnet-01",
        "readonly-conn": "Subnet-01",
        "non-permission-conn": "Subnet-02",
    }

    subnetName := toRemove[conn]
    vpcName := vpcNames[conn]

    url := fmt.Sprintf("%s/vpc/%s/subnet/%s", baseURL, vpcName, subnetName)
    data := map[string]interface{}{
        "ConnectionName": connectionName,
    }

    statusCode, _, err := request.SendRequest("DELETE", url, data)
    if err != nil {
        fmt.Printf("[ERROR] %s failed to send a Request in Delete Subnet: %v\n", connectionName, err)
        return err
    }
    if statusCode == 500{
        if !canWrite{
            fmt.Printf("[PERMISSION-SUCCESS] %s was denied Subnet deletion as expected✅\n", connectionName)
            return nil
        } else {
            return fmt.Errorf("[ERROR] %s failed to deelte Subnet: %v", connectionName, err)
        }
    }

    if statusCode == 200 {
        fmt.Printf("[SUCCESS] %s deleted Subnet %s✅\n", connectionName, subnetName)
    } else {
        return fmt.Errorf("[ERROR] %s received unexpected status code %d while deleting Subnet", connectionName, statusCode)
    }

    return nil
}


