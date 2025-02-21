package resources

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/cloud-barista/cb-spider/permissionTest/connection"
	"github.com/cloud-barista/cb-spider/permissionTest/request"
)

type KeypairResource struct{}

type KeypairResponse struct{
	Fingerprint string `json:"Fingerprint"`
	IId struct {
		NameId   string `json:"NameId"`
		SystemId string `json:"SystemId"`
	} `json:"IId"`
	PublicKey string `json:"PublicKey"`
	VMUserID  string `json:"VMUserID"`
}

// Keypair 목록 응답 구조
type KeypairListResponse struct {
	KeypairList []KeypairResponse `json:"keypair"`
}

func (k KeypairResource) GetName() string {
	return "Keypair"
}

func (k KeypairResource) CreateResource(conn string, errChan chan<-error) error {
	connectionName := connection.Connections[conn].Name
	canWrite := connection.Connections[conn].CanWrite

	keypairNames := map[string][]string{
		"rw-conn":  {"keypair-rw-1", "keypair-rw-2"},
		"rw-conn2": {"keypair-rw2-1", "keypair-rw2-2"},
	}

	keypairs := keypairNames[conn]

	var wg sync.WaitGroup

	for _, keypairName := range keypairs{
		wg.Add(1)
		go func(keypairName string){
			defer wg.Done()

			data := map[string]interface{}{
				"ConnectionName":  connectionName,
				"IDTransformMode": "ON",
				"ReqInfo": map[string]interface{}{
					"Name": keypairName,
					"TagList": []map[string]interface{}{
						{
							"Key":   "key1",
							"Value": "value1",
						},
					},
				},
			}

			url := fmt.Sprintf("%s/keypair", baseURL)
			statusCode, response, err := request.SendRequest("POST", url, data)
			if err != nil{
				errChan <-fmt.Errorf("[ERROR] %s failed to create Keypair %s: %v", connectionName, keypairName, err)
				return
			}

			if statusCode == 500{
				if !canWrite{
					fmt.Printf("[PERMISSION-SUCCESS] %s was denied Keypair creation as expected ✅\n", connectionName)
					return
				}
			}

			if statusCode == 200 && canWrite {

				var keypairResponse struct {
					Fingerprint string `json:"Fingerprint"`
					PublicKey   string `json:"PublicKey"`
					PrivateKey  string `json:"PrivateKey"`
				}

				err = json.Unmarshal(response, &keypairResponse)
				if err != nil {
					errChan <- fmt.Errorf("[ERROR] %s failed to parse Keypair response for %s: %v", connectionName, keypairName, err)
					return
				}

				fmt.Printf("[SUCCESS] %s created Keypair %s ✅\n", connectionName, keypairName)
				fmt.Printf("🔑 Fingerprint: %s\n", keypairResponse.Fingerprint)
				fmt.Printf("🔑 Public Key: %s\n", keypairResponse.PublicKey)
			} else {
				errChan <- fmt.Errorf("[ERROR] %s received unexpected status code %d while creating Keypair %s", connectionName, statusCode, keypairName)
			}
		}(keypairName)
	}
	wg.Wait()
	return nil
}

// Read Keypair
func (k KeypairResource) ReadResource(conn string, errChan chan<- error) error {
	connectionName := connection.Connections[conn].Name
	canRead := connection.Connections[conn].CanRead

	url := fmt.Sprintf("%s/keypair", baseURL)
	data := map[string]interface{}{
		"ConnectionName": connectionName,
	}

	statusCode, response, err := request.SendRequest("GET", url, data)
	if err != nil {
		return fmt.Errorf("[ERROR] %s failed to read Keypairs: %v", connectionName, err)
	}

	if statusCode == 500 {
		if !canRead {
			fmt.Printf("[PERMISSION-SUCCESS] %s was denied Keypair read as expected✅\n", connectionName)
			return nil
		}
		return fmt.Errorf("[ERROR] %s received 500 error while reading Keypairs", connectionName)
	}
	var keypairListResponse KeypairListResponse
	err = json.Unmarshal(response, &keypairListResponse)
	if err != nil {
		return fmt.Errorf("[ERROR] %s failed to parse Keypair response: %v", connectionName, err)
	}

	// ✅ 생성한 Keypair가 모두 조회되는지 확인
	expectedKeypairs := map[string]bool{
		"keypair-rw-1":  false,
		"keypair-rw-2":  false,
		"keypair-rw2-1": false,
		"keypair-rw2-2": false,
	}

	for _, keypair := range keypairListResponse.KeypairList {
		if _, exists := expectedKeypairs[keypair.IId.NameId]; exists {
			expectedKeypairs[keypair.IId.NameId] = true
		}
	}

	// ✅ 모든 Keypair가 조회되었는지 확인
	allFound := true
	for keypair, found := range expectedKeypairs {
		if !found {
			errChan <- fmt.Errorf("[ERROR] %s could not find expected Keypair: %s", connectionName, keypair)
			allFound = false
		}
	}

	if allFound {
		fmt.Printf("[SUCCESS] %s retrieved all expected Keypairs ✅\n", connectionName)
	} else {
		return fmt.Errorf("[ERROR] %s did not retrieve all expected Keypairs", connectionName)
	}

	return nil
}


// Delete Keypair
func (k KeypairResource) DeleteResource(conn string, errChan chan<- error) error {
	connectionName := connection.Connections[conn].Name
	canWrite := connection.Connections[conn].CanWrite

	var keypairNames = map[string][]string{
		"rw-conn":  {"keypair-rw-1", "keypair-rw-2"},
		"rw-conn2": {"keypair-rw2-1", "keypair-rw2-2"},
	}

	keypairs, exists := keypairNames[conn]
	if !exists {
		keypairs = []string{
			fmt.Sprintf("%s-keypair-1", conn),
			fmt.Sprintf("%s-keypair-2", conn),
		} // Default Keypair 이름 설정
	}

	var wg sync.WaitGroup

	for _, keypairName := range keypairs {
		wg.Add(1)
		go func(keypairName string) {
			defer wg.Done()
			url := fmt.Sprintf("%s/keypair/%s", baseURL, keypairName)
			data := map[string]interface{}{
				"ConnectionName": connectionName,
			}

			statusCode, _, err := request.SendRequest("DELETE", url, data)
			if err != nil {
				errChan <- fmt.Errorf("[ERROR] %s failed to delete Keypair %s: %v", connectionName, keypairName, err)
				return
			}

			if statusCode == 500 {
				if !canWrite {
					fmt.Printf("[PERMISSION-SUCCESS] %s was denied Keypair deletion as expected ✅\n", connectionName)
					return
				}
				errChan <- fmt.Errorf("[ERROR] %s received 500 error while deleting Keypair %s", connectionName, keypairName)
				return
			}

			if statusCode == 200 && canWrite {
				fmt.Printf("[SUCCESS] %s deleted Keypair %s ✅\n", connectionName, keypairName)
			} else {
				errChan <- fmt.Errorf("[ERROR] %s received unexpected status code %d while deleting Keypair %s", connectionName, statusCode, keypairName)
			}
		}(keypairName)
	}

	wg.Wait()
	return nil
}
