// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista

package adminweb

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"time"

	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/labstack/echo/v4"
)

// driverCapabilityCluster is the subset of DriverCapabilityInfo needed to tell whether this
// connection's CSP driver supports Cluster at all (e.g. KT does not).
type driverCapabilityCluster struct {
	ClusterHandler bool `json:"ClusterHandler"`
}

// isClusterSupported reports whether connConfig's CSP driver supports Cluster, so the Cluster
// Management page can say so up front instead of silently showing an empty list that looks
// the same as "no cluster created yet". Fails open (true) if the capability lookup itself
// fails, so a hiccup there doesn't hide the page's real content.
func isClusterSupported(connConfig string) bool {
	resBody, err := getResourceList_JsonByte("driver/capability?ConnectionName=" + url.QueryEscape(connConfig))
	if err != nil {
		cblog.Warnf("ClusterManagement: failed to load driver capability for %s: %v", connConfig, err)
		return true
	}
	var capability driverCapabilityCluster
	if err := json.Unmarshal(resBody, &capability); err != nil {
		cblog.Warnf("ClusterManagement: failed to parse driver capability for %s: %v", connConfig, err)
		return true
	}
	return capability.ClusterHandler
}

func fetchClusters(connConfig string) ([]*cres.ClusterInfo, error) {
	resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "cluster")
	if err != nil {
		return nil, fmt.Errorf("error fetching clusters: %v", err)
	}

	var info struct {
		ResultList []*cres.ClusterInfo `json:"cluster"`
	}
	if err := json.Unmarshal(resBody, &info); err != nil {
		return nil, fmt.Errorf("error decoding clusters: %v", err)
	}

	sort.Slice(info.ResultList, func(i, j int) bool {
		return info.ResultList[i].IId.NameId < info.ResultList[j].IId.NameId
	})

	return info.ResultList, nil
}

func ClusterManagement(c echo.Context) error {
	connConfig := c.Param("ConnectConfig")
	if connConfig == "region not set" {
		htmlStr := `
            <html>
            <head>
                <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
                <style>
                    th, td {border: 1px solid lightgray; border-radius: 4px;}
                </style>
            </head>
            <body>
                <br><br>
                <label style="font-size:24px;color:#606262;">Please select a Connection Configuration!</label>
            </body>
        `
		return c.HTML(http.StatusOK, htmlStr)
	}

	regionName, err := getRegionName(connConfig)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	providerName, err := getProviderName(connConfig)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	clusters, err := fetchClusters(connConfig)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	data := struct {
		ConnectionConfig string
		RegionName       string
		ProviderName     string
		Clusters         []*cres.ClusterInfo
		APIUsername      string
		APIPassword      string
		ClusterSupported bool
	}{
		ConnectionConfig: connConfig,
		RegionName:       regionName,
		ProviderName:     providerName,
		Clusters:         clusters,
		APIUsername:      os.Getenv("SPIDER_USERNAME"),
		APIPassword:      os.Getenv("SPIDER_PASSWORD"),
		ClusterSupported: isClusterSupported(connConfig),
	}

	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/cluster.html")
	tmpl, err := template.New("cluster.html").Funcs(template.FuncMap{
		"inc": func(i int) int { return i + 1 },
		"add": func(i, j int) int { return i + j },
		"formatTime": func(t time.Time) string {
			return t.Format("2006-01-02 15:04:05")
		},
	}).ParseFiles(templatePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error loading template: " + err.Error()})
	}

	c.Response().WriteHeader(http.StatusOK)
	if err := tmpl.Execute(c.Response().Writer, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}
	return nil
}
