package resources

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/cloud-barista/cb-spider/permissionTest/connection"
	"github.com/cloud-barista/cb-spider/permissionTest/request"
)

type SecurityGroupResource struct {}

type SecurityGroupResponse struct {
	IId struct {
		NameId   string `json:"NameId"`
		SystemId string `json:"SystemId"`
	} `json:"IId"`
	SecurityRules []struct {
		CIDR       string `json:"CIDR"`
		Direction  string `json:"Direction"`
		FromPort   string `json:"FromPort"`
		IPProtocol string `json:"IPProtocol"`
		ToPort     string `json:"ToPort"`
	} `json:"SecurityRules"`
	TagList []struct {
		Key   string `json:"Key"`
		Value string `json:"Value"`
	} `json:"TagList"`
	VpcIID struct {
		NameId   string `json:"NameId"`
		SystemId string `json:"SystemId"`
	} `json:"VpcIID"`
}

// 전체 SecurityGroup 목록 응답 구조체
type SecurityGroupListResponse struct {
	SecurityGroups []SecurityGroupResponse `json:"securitygroup"`
}

func (s SecurityGroupResource) GetName() string{
	return "SecurityGroup"
}


// Create SecurityGroup
func (s SecurityGroupResource) CreateResource(conn string, errChan chan<- error) error {
	connectionName := connection.Connections[conn].Name
	canWrite := connection.Connections[conn].CanWrite

	// ✅ SecurityGroup을 생성할 VPC 매핑
	securityGroupToVpc := map[string]string{
		"rw-conn":  "VPC-02", // ✅ rw-conn → VPC-02
		"rw-conn2": "VPC-01", // ✅ rw-conn2 → VPC-01
	}

	// ✅ 각 Connection에 대한 SecurityGroup 목록
	securityGroupNames := map[string][]string{
		"rw-conn":  {"security-rw-1", "security-rw-2"},
		"rw-conn2": {"security-rw2-1", "security-rw2-2"},
	}

	// ✅ VPC 및 SecurityGroup 가져오기
	vpcName, exists := securityGroupToVpc[conn]
	if !exists {
		vpcName = "default-vpc"
	}
	securityGroups, exists := securityGroupNames[conn]
	if !exists {
		securityGroups = []string{
			fmt.Sprintf("%s-security-1", conn),
			fmt.Sprintf("%s-security-2", conn),
		}
	}

	var wg sync.WaitGroup

	for _, securityGroupName := range securityGroups {
		wg.Add(1)
		go func(securityGroupName string, vpcName string) {
			defer wg.Done()

			data := map[string]interface{}{
				"ConnectionName":  connectionName,
				"IDTransformMode": "ON",
				"ReqInfo": map[string]interface{}{
					"Name": securityGroupName,
					"SecurityRules": []map[string]interface{}{
						{
							"CIDR":       "0.0.0.0/0",
							"Direction":  "inbound",
							"FromPort":   "22",
							"IPProtocol": "TCP",
							"ToPort":     "22",
						},
					},
					"VPCName": vpcName,
				},
			}

			url := fmt.Sprintf("%s/securitygroup", baseURL)
			statusCode, _, err := request.SendRequest("POST", url, data)
			if err != nil {
				errChan <- fmt.Errorf("[ERROR] %s failed to create SecurityGroup %s in %s: %v", connectionName, securityGroupName, vpcName, err)
				return
			}

			// ✅ 권한 없는 경우 처리
			if statusCode == 500 {
				if !canWrite {
					fmt.Printf("[PERMISSION-SUCCESS] %s was denied SecurityGroup creation in %s as expected ✅\n", connectionName, vpcName)
					return
				}
				errChan <- fmt.Errorf("[ERROR] %s received 500 error while creating SecurityGroup %s in %s", connectionName, securityGroupName, vpcName)
				return
			}

			if statusCode == 200 && canWrite {
				fmt.Printf("[SUCCESS] %s created SecurityGroup %s in %s ✅\n", connectionName, securityGroupName, vpcName)
			} else {
				errChan <- fmt.Errorf("[ERROR] %s received unexpected status code %d while creating SecurityGroup %s in %s", connectionName, statusCode, securityGroupName, vpcName)
			}
		}(securityGroupName, vpcName)
	}

	wg.Wait()
	return nil
}


// Read SecurityGroup
func (s SecurityGroupResource) ReadResource(conn string, errChan chan<- error) error {
	connectionName := connection.Connections[conn].Name
	canRead := connection.Connections[conn].CanRead

	url := fmt.Sprintf("%s/securitygroup", baseURL)
	data := map[string]interface{}{
		"ConnectionName": connectionName,
	}

	statusCode, response, err := request.SendRequest("GET", url, data)
	if err != nil {
		errChan <- fmt.Errorf("[ERROR] %s failed to read SecurityGroups : %v", connectionName, err)
		return fmt.Errorf("[ERROR] %s failed to read SecurityGroups: %v", connectionName, err)
	}

	if statusCode == 500 {
		if canRead {
			errChan <- err
			return fmt.Errorf("[ERROR] %s received 500 error while reading SecurityGroup", connectionName)
		} else {
			fmt.Printf("[PERMISSION-SUCCESS] %s was denied SecurityGroup read as expected✅\n", connectionName)
			return nil
		}
	}
	// ✅ JSON 응답을 `SecurityGroupListResponse`에 저장
	var securityGroupListResponse SecurityGroupListResponse
	err = json.Unmarshal(response, &securityGroupListResponse)
	if err != nil {
		errChan <- err
		return fmt.Errorf("[ERROR] %s failed to parse SecurityGroup response: %v", connectionName, err)
	}

	// ✅ 기대한 SecurityGroup 개수 확인
	expectedSecurityGroups := map[string]bool{
		"security-rw-1": false,
		"security-rw-2": false,
		"security-rw2-1": false,
		"security-rw2-2": false,
	}

	for _, securityGroup := range securityGroupListResponse.SecurityGroups {
		if _, exists := expectedSecurityGroups[securityGroup.IId.NameId]; exists {
			expectedSecurityGroups[securityGroup.IId.NameId] = true
		}
	}

	allFound := true
	for sg, found := range expectedSecurityGroups {
		if !found {
			errChan <- fmt.Errorf("[ERROR] %s could not find expected SecurityGroup: %s", connectionName, sg)
			allFound = false
		}
	}

	if allFound {
		fmt.Printf("[SUCCESS] %s retrieved all expected SecurityGroups ✅\n", connectionName)
		return nil
	} else {
		errChan <- fmt.Errorf("[ERROR] %s did not retrieve all expected SecurityGroups", connectionName)
		return err
	}
}


func(s SecurityGroupResource) DeleteResource(conn string, errChan chan<- error ) error {
	connectionName := connection.Connections[conn].Name
	canWrite := connection.Connections[conn].CanWrite

	securityGroupNames := map[string][]string{
		"rw-conn":  {"security-rw-1", "security-rw-2"},  // ✅ rw-conn → VPC-02의 security-rw-1, security-rw-2 삭제
		"rw-conn2": {"security-rw2-1", "security-rw2-2"}, // ✅ rw-conn2 → VPC-01의 security-rw2-1, security-rw2-2 삭제
	}

	securityGroups, exists := securityGroupNames[conn]
	if !exists {
		securityGroups = []string{
			fmt.Sprintf("%s-security-1", conn),
			fmt.Sprintf("%s-security-2", conn),
		} // Default SecurityGroup 이름 설정
	}

	var wg sync.WaitGroup

	for _, securityGroupName := range securityGroups {
		wg.Add(1)
		go func(securityGroupName string) {
			defer wg.Done()
			url := fmt.Sprintf("%s/securitygroup/%s", baseURL, securityGroupName)

			data := map[string]interface{}{
				"ConnectionName": connectionName,
			}
			statusCode, _, err := request.SendRequest("DELETE", url, data)
			if err != nil {
				errChan <- fmt.Errorf("[ERROR] %s failed to delete SecurityGroup %s: %v", connectionName, securityGroupName, err)
				return
			}

			// ✅ 권한 없는 경우 처리
			if statusCode == 500 {
				if !canWrite {
					fmt.Printf("[PERMISSION-SUCCESS] %s was denied SecurityGroup deletion as expected ✅\n", connectionName)
					return
				}
				errChan <- fmt.Errorf("[ERROR] %s received 500 error while deleting SecurityGroup %s", connectionName, securityGroupName)
				return
			}

			if statusCode == 200 && canWrite {
				fmt.Printf("[SUCCESS] %s deleted SecurityGroup %s ✅\n", connectionName, securityGroupName)
			} else {
				errChan <- fmt.Errorf("[ERROR] %s received unexpected status code %d while deleting SecurityGroup %s", connectionName, statusCode, securityGroupName)
			}
		}(securityGroupName)
	}

	wg.Wait()
	return nil
}