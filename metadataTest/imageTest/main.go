package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var (
	validOSArchitectures = map[string]bool{"arm64": true, "arm64_mac": true, "x86_64": true, "x86_64_mac": true, "NA": true}
	validOSPlatforms     = map[string]bool{"Linux/UNIX": true, "Windows": true, "NA": true}
	validImageStatuses   = map[string]bool{"Available": true, "Unavailable": true, "NA": true}
)

// HTTP Client (Keep-Alive 사용)
var httpClient = &http.Client{Timeout: 10 * time.Second}

// IId 구조체 정의
type IId struct {
	NameId   string `json:"NameId"`
	SystemId string `json:"SystemId"`
}

// Image 구조체 정의
type Image struct {
	IId            IId    `json:"IId"`
	GuestOS        string `json:"GuestOS"`
	Status         string `json:"Status"`
	Name           string `json:"Name"`
	OSArchitecture string `json:"OSArchitecture"`
	OSPlatform     string `json:"OSPlatform"`
	OSDistribution string `json:"OSDistribution"`
	OSDiskType     string `json:"OSDiskType"`
	OSDiskSizeInGB string `json:"OSDiskSizeInGB"`
	ImageStatus    string `json:"ImageStatus"`
	ConnectionName string `json:"ConnectionName"`
}

// Response 구조체 정의
type Response struct {
	Images []Image `json:"image"`
}

// 이미지 유효성 검사
func isValidImage(img Image) bool {
	if !validOSArchitectures[img.OSArchitecture] {
		return false
	}
	if !validOSPlatforms[img.OSPlatform] {
		return false
	}
	if _, err := strconv.Atoi(img.OSDiskSizeInGB); err != nil && img.OSDiskSizeInGB != "-1" {
		return false
	}
	if !validImageStatuses[img.ImageStatus] {
		return false
	}
	return true
}

// 이미지 데이터 가져오기
func fetchImages(connectionName string, wg *sync.WaitGroup, invalidImages *[]Image, mu *sync.Mutex) {
	defer wg.Done()
	url := fmt.Sprintf("http://172.23.49.241:1024/spider/vmimage?ConnectionName=%s", connectionName)
	resp, err := httpClient.Get(url)
	if err != nil {
		fmt.Println("Error making GET request:", err)
		return
	}
	defer resp.Body.Close()

	var responseData Response
	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}

	mu.Lock()
	for _, img := range responseData.Images {
		img.ConnectionName = connectionName
		if !isValidImage(img) {
			*invalidImages = append(*invalidImages, img)
		}
	}
	mu.Unlock()
}

func main() {
	connectionNames := []string{"aws-config01", "alibaba-tokyo-config", "gcp-iowa-config", "tencent-seoul1-config", "ncp-korea1-config", "ncpvpc-korea1-config", "ktcloud-korea-seoul1-config", "ktcloudvpc-mokdong1-config"} // 여러 개의 ConnectionName 추가
	var wg sync.WaitGroup
	var mu sync.Mutex

	invalidImages := []Image{}

	// 병렬로 이미지 데이터 가져오기
	for _, conn := range connectionNames {
		wg.Add(1)
		go fetchImages(conn, &wg, &invalidImages, &mu)
	}

	wg.Wait()

	// 결과 출력
	fmt.Println("Received Invalid Images:")
	if len(invalidImages) == 0 {
		fmt.Println("No invalid images found.")
	} else {
		for i, img := range invalidImages {
			fmt.Printf("----------------Image%d-------------\n", i+1)
			fmt.Printf("Invalid - ConnectionName: %s, NameId: %s, SystemId: %s, OSArchitecture: %s, OSPlatform: %s, OSDiskSizeInGB: %s, ImageStatus: %s\n",
				img.ConnectionName, img.IId.NameId, img.IId.SystemId, img.OSArchitecture, img.OSPlatform, img.OSDiskSizeInGB, img.ImageStatus)
		}
	}
}
