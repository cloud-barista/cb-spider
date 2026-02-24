// Cloud Info Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// Updated: 2024.07.09.
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

type CredentialInfo struct {
	CredentialName   string `json:"CredentialName"`
	ProviderName     string `json:"ProviderName"`
	KeyValueInfoList []struct {
		Key   string `json:"Key"`
		Value string `json:"Value"`
	} `json:"KeyValueInfoList"`
}

type Credentials struct {
	Credentials []CredentialInfo `json:"credential"`
}

type CredentialMetaInfo struct {
	Credential []string `json:"Credential"`
}

func fetchCredentials() (map[string][]CredentialInfo, error) {
	resp, err := httpGetWithAuth("http://localhost:1024/spider/credential")
	if err != nil {
		return nil, fmt.Errorf("error fetching credentials: %v", err)
	}
	defer resp.Body.Close()

	var credentials Credentials
	if err := json.NewDecoder(resp.Body).Decode(&credentials); err != nil {
		return nil, fmt.Errorf("error decoding credentials: %v", err)
	}

	credentialMap := make(map[string][]CredentialInfo)
	for _, credential := range credentials.Credentials {
		credentialMap[credential.ProviderName] = append(credentialMap[credential.ProviderName], credential)
	}

	return credentialMap, nil
}

func fetchCredentialMetaInfo(provider string) ([]string, error) {
	resp, err := httpGetWithAuth(fmt.Sprintf("http://localhost:1024/spider/cloudos/metainfo/%s", provider))
	if err != nil {
		return nil, fmt.Errorf("error fetching credential meta info for provider %s: %v", provider, err)
	}
	defer resp.Body.Close()

	var metaInfo CredentialMetaInfo
	if err := json.NewDecoder(resp.Body).Decode(&metaInfo); err != nil {
		return nil, fmt.Errorf("error decoding credential meta info: %v", err)
	}

	return metaInfo.Credential, nil
}

func CredentialManagement(c echo.Context) error {
	credentials, err := fetchCredentials()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	providers, err := fetchProviders()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	data := struct {
		Credentials map[string][]CredentialInfo
		Providers   []string
		APIUsername string
		APIPassword string
	}{
		Credentials: credentials,
		Providers:   providers,
		APIUsername: os.Getenv("SPIDER_USERNAME"),
		APIPassword: os.Getenv("SPIDER_PASSWORD"),
	}

	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/credential.html")
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error loading template: " + err.Error()})
	}

	c.Response().WriteHeader(http.StatusOK)
	if err := tmpl.Execute(c.Response().Writer, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}
	return nil
}
