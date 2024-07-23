// Cloud Info Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// Updated: 2024.07.11.
// by CB-Spider Team, 2020.06.

package adminweb

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
)

type RegionMetaInfo struct {
	Region []string `json:"Region"`
}

func fetchRegionInfos() (map[string][]RegionInfo, error) {
	resp, err := http.Get("http://localhost:1024/spider/region")
	if err != nil {
		return nil, fmt.Errorf("error fetching regions: %v", err)
	}
	defer resp.Body.Close()

	var regions Regions
	if err := json.NewDecoder(resp.Body).Decode(&regions); err != nil {
		return nil, fmt.Errorf("error decoding regions: %v", err)
	}

	regionMap := make(map[string][]RegionInfo)
	for _, region := range regions.Regions {
		regionMap[region.ProviderName] = append(regionMap[region.ProviderName], region)
	}

	return regionMap, nil
}

func fetchRegionMetaInfo(provider string) ([]string, error) {
	resp, err := http.Get(fmt.Sprintf("http://localhost:1024/spider/cloudos/metainfo/%s", provider))
	if err != nil {
		return nil, fmt.Errorf("error fetching region meta info for provider %s: %v", provider, err)
	}
	defer resp.Body.Close()

	var metaInfo RegionMetaInfo
	if err := json.NewDecoder(resp.Body).Decode(&metaInfo); err != nil {
		return nil, fmt.Errorf("error decoding region meta info: %v", err)
	}

	return metaInfo.Region, nil
}

func RegionManagement(c echo.Context) error {
	regions, err := fetchRegionInfos()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	providers, err := fetchProviders()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	data := struct {
		Regions   map[string][]RegionInfo
		Providers []string
	}{
		Regions:   regions,
		Providers: providers,
	}

	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/region.html")
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error loading template: " + err.Error()})
	}

	return tmpl.Execute(c.Response().Writer, data)
}
