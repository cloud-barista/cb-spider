package mocktest

import (
	"encoding/json"
	"testing"

	cblog "github.com/cloud-barista/cb-log"
	mockres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/mock/resources"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/itchyny/gojq"
)

func init() {
	// make the log level lower to print clearly
	cblog.GetLogger("CB-SPIDER")
	cblog.SetLevel("error")
}

func TestListProductFamily(t *testing.T) {
	handler := &mockres.MockPriceInfoHandler{}

	results, err := handler.ListProductFamily()
	if err != nil {
		t.Errorf("ListProductFamily returned an error: %v", err)
	}
	if len(results) == 0 {
		t.Error("ListProductFamily returned no results")
	}

	for _, result := range results {
		t.Logf("\t%#v\n", result)
	}
}

func TestGetComputeInstancePriceInfo(t *testing.T) {
	handler := &mockres.MockPriceInfoHandler{}

	productFamily := mockres.COMPUTE_INSTANCE

	// test cases
	testCases := []struct {
		regionName    string
		filterList    []irs.KeyValue
		expectedMatch int // expected product count of result
	}{
		{"mercury", nil, 2},
		{"mercury", []irs.KeyValue{}, 2},
		{"mercury", []irs.KeyValue{{Key: "productId", Value: "mock.enhnace1.mercury"}}, 1},
		{"mercury", []irs.KeyValue{{Key: "noField", Value: "mock.enhnace1.mercury"}}, 0},
		{"mercury", []irs.KeyValue{{Key: "vcpu", Value: "8"}}, 2},
		{"mercury", []irs.KeyValue{{Key: "pricingPolicy", Value: "OnDemand"}}, 2},
		{"mercury", []irs.KeyValue{{Key: "LeaseContractLength", Value: "1 Year"}}, 2},
	}

	for _, tc := range testCases {
		jsonPriceInfo, err := handler.GetPriceInfo(productFamily, tc.regionName, tc.filterList)
		if err != nil {
			t.Errorf("GetPriceInfo returned an error: %v", err)
		}
		if jsonPriceInfo == "" {
			t.Error("GetPriceInfo returned empty jsonPriceInfo")
		}

		// validate result
		pidNum := countProductInJson(jsonPriceInfo, t)
		if pidNum != tc.expectedMatch {
			t.Errorf("GetPriceInfo returned %d products, expected %d", pidNum, tc.expectedMatch)
		}
	} // end of for statement with testCases
}

func TestGetStoragePriceInfo(t *testing.T) {
	handler := &mockres.MockPriceInfoHandler{}

	productFamily := mockres.STORAGE

	// test cases
	testCases := []struct {
		regionName    string
		filterList    []irs.KeyValue
		expectedMatch int // expected product count of result
	}{
		{"mercury", nil, 1},
		{"mercury", []irs.KeyValue{}, 1},
		{"mercury", []irs.KeyValue{{Key: "productId", Value: "mock.storage1.mercury"}}, 1},
		{"mercury", []irs.KeyValue{{Key: "noField", Value: "mock.storage1.mercury"}}, 0},
		{"mercury", []irs.KeyValue{{Key: "volumeType", Value: "SSD"}}, 1},
		{"mercury", []irs.KeyValue{{Key: "pricingPolicy", Value: "OnDemand"}}, 1},
		{"mercury", []irs.KeyValue{{Key: "LeaseContractLength", Value: "1 Year"}}, 1},
	}

	for _, tc := range testCases {
		jsonPriceInfo, err := handler.GetPriceInfo(productFamily, tc.regionName, tc.filterList)
		if err != nil {
			t.Errorf("GetPriceInfo returned an error: %v", err)
		}
		if jsonPriceInfo == "" {
			t.Error("GetPriceInfo returned empty jsonPriceInfo")
		}

		// validate result
		pidNum := countProductInJson(jsonPriceInfo, t)
		if pidNum != tc.expectedMatch {
			t.Errorf("GetPriceInfo returned %d products, expected %d", pidNum, tc.expectedMatch)
		}
	} // end of for statement with testCases
}

func TestGetNLBPriceInfo(t *testing.T) {
	handler := &mockres.MockPriceInfoHandler{}

	productFamily := mockres.NETWORK_LOAD_BALANCER

	// test cases
	testCases := []struct {
		regionName    string
		filterList    []irs.KeyValue
		expectedMatch int // expected product count of result
	}{
		{"mercury", nil, 1},
		{"mercury", []irs.KeyValue{}, 1},
		{"mercury", []irs.KeyValue{{Key: "productId", Value: "mock.nlb.mercury"}}, 1},
		{"mercury", []irs.KeyValue{{Key: "noField", Value: "mock.nlb.mercury"}}, 0},
		{"mercury", []irs.KeyValue{{Key: "pricingPolicy", Value: "OnDemand"}}, 1},
		{"mercury", []irs.KeyValue{{Key: "LeaseContractLength", Value: "1 Year"}}, 1},
	}

	for _, tc := range testCases {
		jsonPriceInfo, err := handler.GetPriceInfo(productFamily, tc.regionName, tc.filterList)
		if err != nil {
			t.Errorf("GetPriceInfo returned an error: %v", err)
		}
		if jsonPriceInfo == "" {
			t.Error("GetPriceInfo returned empty jsonPriceInfo")
		}

		// validate result
		pidNum := countProductInJson(jsonPriceInfo, t)
		if pidNum != tc.expectedMatch {
			t.Errorf("GetPriceInfo returned %d products, expected %d", pidNum, tc.expectedMatch)
		}
	} // end of for statement with testCases
}

func countProductInJson(jsonPriceInfo string, t *testing.T) int {

	var jsonData interface{}
	err := json.Unmarshal([]byte(jsonPriceInfo), &jsonData)
	if err != nil {
		t.Fatalf("Error unmarshaling JSON: %v", err)
	}

	// gojq query
	query, err := gojq.Parse(".cloudPriceList[].priceList[].productInfo.productId")
	if err != nil {
		t.Fatalf("Error parsing query: %v", err)
	}

	// run gojq query
	var productIds []string
	iter := query.Run(jsonData)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			t.Fatalf("Error executing query: %v", err)
		}
		productIds = append(productIds, v.(string))
	}

	// print result
	for _, id := range productIds {
		t.Log("\tProductId:", id)
	}
	t.Logf("\tTotal ProductIds: %d\n", len(productIds))

	return len(productIds)
}
