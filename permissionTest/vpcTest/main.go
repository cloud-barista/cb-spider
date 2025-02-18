package vpcTest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

const baseURL = "http://localhost:1024/spider"

type Connection struct {
	Name     string
	CanWrite bool
	CanRead  bool
}

var connections = map[string]Connection{
	"rw-conn":             {Name: os.Getenv("RW_CONN_NAME"), CanWrite: true, CanRead: true},
    "rw-conn2":            {Name: os.Getenv("RW_CONN2_NAME"), CanWrite: true, CanRead: true},
    "readonly-conn":       {Name: os.Getenv("READONLY_CONN_NAME"), CanWrite: false, CanRead: true},
    "non-permission-conn": {Name: os.Getenv("NON_PERMISSION_CONN_NAME"), CanWrite: false, CanRead: false},
}

// 테스트 시나리오에 사용할 VPC 이름
var vpcNames = map[string]string{
	"rw-conn":             "VPC-01",
	"rw-conn2":            "VPC-01",
	"readonly-conn":       "VPC-03",
	"non-permission-conn": "VPC-04",
}


var client = &http.Client{
	Timeout: time.Second * 10,
}
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

// JSON 요청을 보내고 응답을 반환하는 함수
func sendRequest(method, url string, data map[string]interface{}) (int, []byte, error) {
	jsonData, _ := json.Marshal(data)
	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, body, nil
}

func createVPCs(conn string, wg *sync.WaitGroup, errChan chan<- error) {
	defer wg.Done()
	connectionName := connections[conn].Name
	canWrite := connections[conn].CanWrite

	vpcName := vpcNames[conn]

	url := baseURL + "/vpc"
	data := map[string]interface{}{
		"ConnectionName": connectionName,
		"ReqInfo": map[string]interface{}{
			"Name":      vpcName,
			"IPv4_CIDR": "192.168.0.0/16",
			"SubnetInfoList": []map[string]interface{}{
				{"Name": "Subnet-01", "IPv4_CIDR": "192.168.1.0/24"},
			},
		},
	}
	statusCode, _, err := sendRequest("POST", url, data)
	if err != nil {
		errChan <- fmt.Errorf("[ERROR] %s failed to create VPC: %v\n", connectionName, err)
		return
	}

	if statusCode == 200 && canWrite {
		fmt.Printf("[SUCCESS] %s created VPC\n", connectionName)
	} else if statusCode == 500 && !canWrite {
		fmt.Printf("[PERMISSON-SUCCESS] %s was denied VPC creation as expected.\n", connectionName) // 권한 없는 데 생성 시도했을 경우 error 발생 -> Test 성공
	} else {
		errChan <- fmt.Errorf("[ERROR] %s received unexpected status code %d while creating VPC.\n", connectionName, statusCode)
	}
}

func readVPCs(conn string, wg *sync.WaitGroup, errChan chan<- error) {
	defer wg.Done()
	connectionName := connections[conn].Name
	canRead := connections[conn].CanRead

	url := fmt.Sprintf("%s/vpc", baseURL)
	data := map[string]interface{}{
		"ConnectionName": connectionName,
	}

	statusCode, response, err := sendRequest("GET", url, data)
	if err != nil {
		errChan <- fmt.Errorf("[ERROR] %s failed to read VPCs: %v\n", connectionName, err)
		return
	}

	// 권한이 없어서 500이면 정상적으로 차단된 것
	if statusCode == 500 {
		if !canRead {
			fmt.Printf("[PERMISSION-SUCCESS] %s was denied VPC read as expected.\n", connectionName)
			return
		} else {
			errChan <- fmt.Errorf("[ERROR] %s received 500 error while reading VPCs. Server issue?\n", connectionName)
			return
		}
	}

	// JSON 파싱
	var vpcListResponse VPCListResponse
	err = json.Unmarshal(response, &vpcListResponse)
	if err != nil {
		errChan <- fmt.Errorf("[ERROR] %s failed to parse VPC response: %v\n", connectionName, err)
		return
	}

	// 생성한 VPC가 모두 조회되는지 확인
	expectedVPCs := map[string]bool{
		"VPC-01": false,
		"VPC-02": false,
	}

	for _, vpc := range vpcListResponse.VPCList {
		if _, exists := expectedVPCs[vpc.IId.NameId]; exists {
			expectedVPCs[vpc.IId.NameId] = true
		}
	}

	// 모든 VPC가 조회되었는지 확인
	allFound := true
	for vpc, found := range expectedVPCs {
		if !found {
			errChan <- fmt.Errorf("[ERROR] %s could not find expected VPC: %s\n", connectionName, vpc)
			allFound = false
		}
	}

	if allFound {
		fmt.Printf("[SUCCESS] %s retrieved all expected VPCs.\n", connectionName)
	} else {
		errChan <- fmt.Errorf("[ERROR] %s did not retrieve all expected VPCs.\n", connectionName)
	}
}

func deleteVPCs(conn string, wg *sync.WaitGroup, errChan chan<-error) {
	defer wg.Done()
	connectionName := connections[conn].Name
	canWrite := connections[conn].CanWrite

	// 삭제해야 할 VPC 목록과 삭제 여부를 추적
	expectedVPCs := map[string]bool{};


	switch conn{
	case "rw-conn":
		expectedVPCs["VPC-02"] = false
	case "rw-conn2":
		expectedVPCs["VPC-01"] = false
	default:
		expectedVPCs["VPC-01"] = false
		expectedVPCs["VPC-02"] = false
	}

	// 모든 VPC 삭제 요청을 수행

	for vpcName := range expectedVPCs{
		url := fmt.Sprintf("%s/vpc/%s", baseURL, vpcName)
		data := map[string]interface{}{
			"ConnectionName": connectionName,
		}
		statusCode, _, err := sendRequest("DELETE", url, data)
		if err != nil {
			errChan <- fmt.Errorf("[ERROR] %s failed to delete VPC %s: %v\n", connectionName, vpcName, err)
			continue
		}

		if canWrite && statusCode == 200{
			expectedVPCs[vpcName] = true
		} else if canWrite && statusCode == 500{
			errChan <- fmt.Errorf("[ERROR] %s received 500 error while deleting VPC %s.\n", connectionName, vpcName)
		} else if !canWrite && statusCode == 200{
			errChan <- fmt.Errorf("[PERMISSION ERROR] %s was able to delete VPC %s despite lacking permission.\n", connectionName, vpcName)
		} else { // !conn.CanWrite && statusCode == 500
			fmt.Printf("[PERMISSION SUCCESS] %s was denied VPC deletion as expected.\n", connectionName)
			expectedVPCs[vpcName] = true
		}	
	}

	allPassed := true
	for vpcName, pass := range expectedVPCs{
		if !pass{
			errChan <- fmt.Errorf("[ERROR] There is an error in connection : %s, VPC : %s\n", connectionName, vpcName)
			allPassed=false
		}
	}

	if allPassed{
		fmt.Printf("[FINAL SUCCESS] %s successfully handled all VPC deletions.\n", connectionName)
	} else {
		errChan <- fmt.Errorf("[FINAL ERROR] %s encountered issues during VPC deletions.\n", connectionName)
	}
}

func CreateTest(wg *sync.WaitGroup, errChan chan<- error) {
	fmt.Println("----------------------------VPC CREATE Test----------------------------")
	for conn := range connections {
		wg.Add(1)
		go createVPCs(conn, wg, errChan)
	}
	wg.Wait()
}

func ReadTest(wg *sync.WaitGroup, errChan chan<- error) {
	fmt.Println("----------------------------VPC READ Test----------------------------")
	for conn := range connections {
		wg.Add(1)
		go readVPCs(conn, wg, errChan)
	}
	wg.Wait()
}

func DeleteTest(wg *sync.WaitGroup, errChan chan<- error) {
	fmt.Println("----------------------------VPC DELETE Test----------------------------")
	for conn := range connections {
		wg.Add(1)
		go deleteVPCs(conn, wg, errChan)
	}
	wg.Wait()
}

func main() {
	var wg sync.WaitGroup
	errChan := make(chan error, len(connections))

	CreateTest(&wg, errChan)
	ReadTest(&wg, errChan)
	DeleteTest(&wg, errChan)

	close(errChan)

	errOccured := false
	for err := range errChan {
		fmt.Println(err)
		errOccured = true
	}

	if errOccured{
		fmt.Println("Test Failed❌")
		return
	}

	fmt.Println("Test Finished Successfully✅")
	return
}
