package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// Spec 데이터 가져오기
func fetchSpecs(connectionName string, wg *sync.WaitGroup, validSpecs *[]map[string]interface{}, invalidSpecs *[]map[string]interface{}, mu *sync.Mutex) {
	defer wg.Done()
	url := fmt.Sprintf("http://localhost:1024/spider/vmspec?ConnectionName=%s", connectionName)
	resp, err := httpClient.Get(url)
	if err != nil {
		fmt.Println("Error making GET request:", err)
		return
	}
	defer resp.Body.Close()

	var responseData map[string][]map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}

	mu.Lock()
	if specs, exists := responseData["vmspec"]; exists {
		for _, spec := range specs {
			spec["ConnectionName"] = connectionName
			reason := isValidSpec(spec)
			if reason == "Valid" {
				*validSpecs = append(*validSpecs, spec)
			} else {
				spec["InvalidReason"] = reason
				*invalidSpecs = append(*invalidSpecs, spec)
			}
		}
	}
	mu.Unlock()
}
func isValidSpec(spec map[string]interface{}) string {
	requiredKeys := []string{"Region", "Name", "VCpu", "Mem", "Disk"}
	for _, key := range requiredKeys {
		if _, exists := spec[key]; !exists {
			return fmt.Sprintf("Missing required key: %s", key)
		}
	}

	// VCpu 내부 필드 검사
	if vcpu, ok := spec["VCpu"].(map[string]interface{}); ok {
		if _, exists := vcpu["Count"]; !exists {
			return "Missing VCpu Count"
		}
		if clock, exists := vcpu["Clock"]; !exists || clock == "0" || clock == "N/A" {
			return "Invalid VCpu Clock"
		}
	} else {
		return "Invalid VCpu structure"
	}

	// Disk 정보 확인
	if disk, ok := spec["Disk"].(string); !ok || disk == "0" || disk == "N/A" {
		return "Invalid Disk value"
	}

	// GPU 필드 검사 (GPU 정보가 제공될 경우)
	if gpuList, exists := spec["Gpu"].([]interface{}); exists {
		for _, gpu := range gpuList {
			if gpuMap, ok := gpu.(map[string]interface{}); ok {
				// 1. Count 검증 (-1이나 숫자만 가능, NA/0 불가능)
				if count, exists := gpuMap["Count"]; !exists || count == "NA" || count == "0" {
					return "Invalid GPU Count"
				}

				// 2. Mfr 검증 (문자열 또는 NA만 가능)
				if mfr, exists := gpuMap["Mfr"]; !exists || (mfr != "NA" && fmt.Sprintf("%T", mfr) != "string") {
					return "Invalid GPU Mfr"
				}

				// 3. Model 검증 (문자열 또는 NA만 가능)
				if model, exists := gpuMap["Model"]; !exists || (model != "NA" && fmt.Sprintf("%T", model) != "string") {
					return "Invalid GPU Model"
				}

				// 4. Mem 검증 (1024MB 이상이며, -1 또는 숫자여야 함. NA/0 불가능)
				if mem, exists := gpuMap["Mem"]; !exists || mem == "NA" || mem == "0" {
					return "Invalid GPU Mem"
				} else {
					// GPU Mem이 string 타입일 경우 변환
					var memValue float64
					switch v := mem.(type) {
					case float64:
						memValue = v
					case string:
						parsed, err := strconv.ParseFloat(v, 64)
						if err != nil {
							return "Invalid GPU Mem Format"
						}
						memValue = parsed
					default:
						return "Invalid GPU Mem Type"
					}

					// 1024MB 이상이어야 유효함
					if memValue != -1 && memValue < 1024 {
						return "Invalid GPU Mem Size"
					}
				}
			}
		}
	}

	return "Valid"
}

func saveToCSV(filename string, data []map[string]interface{}) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 헤더 정의
	header := []string{
		"ConnectionName", "Region", "Name", "VCpu_Count", "VCpu_Clock",
		"Mem", "Disk", "Gpu_Count", "Gpu_Mfr", "Gpu_Model", "Gpu_Mem", "InvalidReason",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	// 데이터 작성
	for _, spec := range data {
		// VCpu 정보 가져오기
		vcpu := spec["VCpu"].(map[string]interface{})
		count := fmt.Sprintf("%v", vcpu["Count"])
		clock := fmt.Sprintf("%v", vcpu["Clock"])

		// GPU 정보 가져오기 (에러가 발생하지 않은 경우에만 기록)
		gpuCount := ""
		gpuMfr := ""
		gpuModel := ""
		gpuMem := ""

		if gpuList, exists := spec["Gpu"].([]interface{}); exists && len(gpuList) > 0 {
			if gpuMap, ok := gpuList[0].(map[string]interface{}); ok {
				gpuCount = fmt.Sprintf("%v", gpuMap["Count"])
				gpuMfr = fmt.Sprintf("%v", gpuMap["Mfr"])
				gpuModel = fmt.Sprintf("%v", gpuMap["Model"])
				gpuMem = fmt.Sprintf("%v", gpuMap["Mem"])
			}
		}

		// InvalidReason 가져오기 (없으면 빈 값)
		reason := ""
		if val, exists := spec["InvalidReason"]; exists {
			reason = fmt.Sprintf("%v", val)
		}

		record := []string{
			fmt.Sprintf("%v", spec["ConnectionName"]),
			fmt.Sprintf("%v", spec["Region"]),
			fmt.Sprintf("%v", spec["Name"]),
			count,
			clock,
			fmt.Sprintf("%v", spec["Mem"]),
			fmt.Sprintf("%v", spec["Disk"]),
			gpuCount, gpuMfr, gpuModel, gpuMem,
			reason,
		}

		if err := writer.Write(record); err != nil {
			return err
		}
	}

	fmt.Printf("Saved data to %s\n", filename)
	return nil
}

func main() {
	connectionNames := []string{"aws-config01", "azure-koreacentral-config", "gcp-tokyo-config", "alibaba-beijing-config", "tencent-guangzhou3-config", "ibmvpc-us-east-1-config", "openstack-RegionOne-com3-config", "nhncloud-korea-pangyo-config", "ncp-korea1-config", "ncpvpc-korea1-config", "ktcloud-korea-seoul1-config", "ktcloudvpc-mokdong1-config"} // 여러 개의 ConnectionName 추가
	var wg sync.WaitGroup
	var mu sync.Mutex

	validSpecs := []map[string]interface{}{}
	invalidSpecs := []map[string]interface{}{}

	// 병렬로 Spec 데이터 가져오기
	for _, conn := range connectionNames {
		wg.Add(1)
		go fetchSpecs(conn, &wg, &validSpecs, &invalidSpecs, &mu)
	}

	wg.Wait()

	// CSV 파일로 저장
	if err := saveToCSV("valid_specs.csv", validSpecs); err != nil {
		fmt.Println("Error saving valid specs:", err)
	}
	if err := saveToCSV("invalid_specs.csv", invalidSpecs); err != nil {
		fmt.Println("Error saving invalid specs:", err)
	}

	fmt.Println("CSV files have been successfully created.")
}
