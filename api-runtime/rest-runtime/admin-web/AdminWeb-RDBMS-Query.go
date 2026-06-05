// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, June 2025.

package adminweb

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	cr "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/go-sql-driver/mysql"
	"github.com/labstack/echo/v4"

	_ "github.com/lib/pq"
)

// validIdentifier checks that a SQL identifier (table/column name) contains only safe characters.
var validIdentifier = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func isValidIdentifier(name string) bool {
	return validIdentifier.MatchString(name) && len(name) <= 128
}

// fetchRDBMSInfo retrieves RDBMS info via the internal REST API.
func fetchRDBMSInfo(connConfig, rdbmsName string) (*cres.RDBMSInfo, error) {
	url := "http://localhost" + cr.ServerPort + "/spider/rdbms/" + rdbmsName

	var reqBody struct {
		ConnectionName string `json:"ConnectionName"`
	}
	reqBody.ConnectionName = connConfig

	jsonValue, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	request, err := http.NewRequest("GET", url, strings.NewReader(string(jsonValue)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	setBasicAuthIfConfigured(request)

	client := http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch RDBMS info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch RDBMS info: HTTP %d", resp.StatusCode)
	}

	var info cres.RDBMSInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode RDBMS info: %w", err)
	}
	return &info, nil
}

// openDBConnection creates a database/sql connection to the RDBMS instance.
// dbNameOverride specifies the database name to connect to.
func openDBConnection(info *cres.RDBMSInfo, password, dbNameOverride string) (*sql.DB, string, error) {
	engine := strings.ToLower(info.DBEngine)
	endpoint := info.Endpoint
	user := info.MasterUserName
	dbName := dbNameOverride
	// Treat "NA" as no database specified (some drivers return "NA" as placeholder)
	if strings.EqualFold(dbName, "NA") {
		dbName = ""
	}

	if endpoint == "" {
		status := string(info.Status)
		if status != "" && status != "Available" {
			return nil, "", fmt.Errorf("RDBMS is not available yet (Status: %s). Please wait until it becomes Available", status)
		}
		return nil, "", fmt.Errorf("RDBMS endpoint is empty. The instance may still be provisioning")
	}

	// Strip port from endpoint if it already contains one (e.g., "host.rds.amazonaws.com:3306")
	host := endpoint
	port := ""
	if idx := strings.LastIndex(endpoint, ":"); idx > 0 {
		hostPart := endpoint[:idx]
		portPart := endpoint[idx+1:]
		// Check if the part after ':' looks like a port number
		if _, err := fmt.Sscanf(portPart, "%d", new(int)); err == nil {
			host = hostPart
			port = portPart
		}
	}

	var driverName, dsn string

	switch {
	case engine == "mysql" || engine == "mariadb":
		driverName = "mysql"
		if port == "" {
			port = "3306"
		}
		mysqlConfig := mysql.NewConfig()
		mysqlConfig.User = user
		mysqlConfig.Passwd = password
		mysqlConfig.Net = "tcp"
		mysqlConfig.Addr = net.JoinHostPort(host, port)
		mysqlConfig.DBName = dbName
		mysqlConfig.ParseTime = true
		mysqlConfig.Timeout = 30 * time.Second
		if mysqlHostRequiresTLS(host) {
			mysqlConfig.TLSConfig = "skip-verify"
		}
		dsn = mysqlConfig.FormatDSN()

	case engine == "postgresql" || engine == "postgres":
		driverName = "postgres"
		if port == "" {
			port = "5432"
		}
		pgDB := dbName
		if pgDB == "" {
			pgDB = "postgres" // default database for PostgreSQL
		}
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=require connect_timeout=30",
			host, port, user, password, pgDB)

	default:
		return nil, "", fmt.Errorf("unsupported DB engine: %s", engine)
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open DB connection: %w", err)
	}

	db.SetConnMaxLifetime(30 * time.Second)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(0)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, "", fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, driverName, nil
}

func mysqlHostRequiresTLS(host string) bool {
	lowerHost := strings.ToLower(host)
	return strings.Contains(lowerHost, ".azure.") || strings.Contains(lowerHost, ".databases.appdomain.cloud")
}

// --- Request/Response structures ---

type rdbmsQueryRequest struct {
	ConnectionName string `json:"ConnectionName"`
	Password       string `json:"Password"`
	DatabaseName   string `json:"DatabaseName,omitempty"`
}

type createTableRequest struct {
	ConnectionName string       `json:"ConnectionName"`
	Password       string       `json:"Password"`
	DatabaseName   string       `json:"DatabaseName,omitempty"`
	TableName      string       `json:"TableName"`
	Columns        []columnInfo `json:"Columns"`
}

type columnInfo struct {
	Name       string `json:"Name"`
	Type       string `json:"Type"`
	PrimaryKey bool   `json:"PrimaryKey,omitempty"`
	NotNull    bool   `json:"NotNull,omitempty"`
}

type insertRowRequest struct {
	ConnectionName string            `json:"ConnectionName"`
	Password       string            `json:"Password"`
	DatabaseName   string            `json:"DatabaseName,omitempty"`
	Values         map[string]string `json:"Values"`
}

type deleteRowRequest struct {
	ConnectionName string            `json:"ConnectionName"`
	Password       string            `json:"Password"`
	DatabaseName   string            `json:"DatabaseName,omitempty"`
	Where          map[string]string `json:"Where"`
}

// --- Handlers ---

// RDBMSTestConnection tests connectivity to an RDBMS instance.
func RDBMSTestConnection(c echo.Context) error {
	rdbmsName := c.Param("Name")

	var req rdbmsQueryRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	if req.ConnectionName == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ConnectionName and Password are required"})
	}

	info, err := fetchRDBMSInfo(req.ConnectionName, rdbmsName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	db, _, err := openDBConnection(info, req.Password, req.DatabaseName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer db.Close()

	return c.JSON(http.StatusOK, map[string]string{"status": "connected"})
}

// ============================================================
// NHN Cloud RDS v3.0 helper functions
// NHN RDS does not support CREATE DATABASE SQL or stored procedures.
// DB schema creation/deletion must go through the NHN Cloud RDS v3.0 REST API.
// ============================================================

// getNHNRDSCredentials fetches the NHN RDS API credentials (appKey, userAccessKey, secretAccessKey)
// from the CB-Spider connection config and its associated credential info.
func getNHNRDSCredentials(connName string) (appKey, userAccessKey, secretAccessKey string, err error) {
	ccBody, err := getResource_JsonByte("connectionconfig", connName)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get connection config for %q: %w", connName, err)
	}
	var cc struct {
		CredentialName string `json:"CredentialName"`
	}
	if err := json.Unmarshal(ccBody, &cc); err != nil {
		return "", "", "", fmt.Errorf("failed to parse connection config: %w", err)
	}
	if cc.CredentialName == "" {
		return "", "", "", fmt.Errorf("CredentialName is empty in connection config %q", connName)
	}

	credBody, err := getResource_JsonByte("credential", cc.CredentialName)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get credential %q: %w", cc.CredentialName, err)
	}
	var cred struct {
		KeyValueInfoList []struct {
			Key   string `json:"Key"`
			Value string `json:"Value"`
		} `json:"KeyValueInfoList"`
	}
	if err := json.Unmarshal(credBody, &cred); err != nil {
		return "", "", "", fmt.Errorf("failed to parse credential: %w", err)
	}

	for _, kv := range cred.KeyValueInfoList {
		switch kv.Key {
		case "appKey":
			appKey = kv.Value
		case "User Access Key":
			userAccessKey = kv.Value
		case "Secret Access Key":
			secretAccessKey = kv.Value
		}
	}
	if appKey == "" || userAccessKey == "" || secretAccessKey == "" {
		return "", "", "", fmt.Errorf("missing NHN RDS credentials in %q (appKey=%v, userAccessKey=%v, secretAccessKey=%v)",
			cc.CredentialName, appKey != "", userAccessKey != "", secretAccessKey != "")
	}
	return appKey, userAccessKey, secretAccessKey, nil
}

// getNHNRDSEndpoint derives the NHN Cloud RDS API endpoint from the connection's region.
func getNHNRDSEndpoint(connName string) (string, error) {
	regionName, err := getRegionName(connName)
	if err != nil {
		return "", fmt.Errorf("failed to get region name for %q: %w", connName, err)
	}
	region, _, err := getRegionZone(regionName)
	if err != nil {
		return "", fmt.Errorf("failed to get region zone for %q: %w", regionName, err)
	}
	region = strings.ToLower(region)
	switch region {
	case "kr1", "kr2", "jp1":
		return fmt.Sprintf("https://%s-rds-mysql.api.nhncloudservice.com", region), nil
	default:
		return "", fmt.Errorf("unsupported NHN Cloud RDS region: %q", region)
	}
}

// nhnRDSCallRaw makes an authenticated HTTP call to the NHN Cloud RDS v3.0 API.
func nhnRDSCallRaw(ctx context.Context, method, url, appKey, userAccessKey, secretAccessKey string, reqBody interface{}) ([]byte, int, error) {
	var bodyReader *bytes.Reader
	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	} else {
		bodyReader = bytes.NewReader(nil)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create NHN RDS request: %w", err)
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-TC-APP-KEY", appKey)
	req.Header.Set("X-TC-AUTHENTICATION-ID", userAccessKey)
	req.Header.Set("X-TC-AUTHENTICATION-SECRET", secretAccessKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("NHN RDS API call failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read NHN RDS API response: %w", err)
	}
	return body, resp.StatusCode, nil
}

// pollNHNRDSJob polls a NHN Cloud RDS job until it reaches a terminal state.
func pollNHNRDSJob(ctx context.Context, endpoint, appKey, userAccessKey, secretAccessKey, jobId string) error {
	jobURL := fmt.Sprintf("%s/v3.0/jobs/%s", endpoint, jobId)
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for NHN RDS job %s", jobId)
		default:
		}

		body, _, err := nhnRDSCallRaw(ctx, http.MethodGet, jobURL, appKey, userAccessKey, secretAccessKey, nil)
		if err == nil {
			var jobResp struct {
				JobStatus string `json:"jobStatus"`
			}
			if jsonErr := json.Unmarshal(body, &jobResp); jsonErr == nil {
				switch jobResp.JobStatus {
				case "COMPLETED":
					return nil
				case "FAILED", "ERROR", "CANCELED", "INTERRUPTED", "FAIL_TO_READY", "DELETED":
					return fmt.Errorf("NHN RDS job %s failed with status: %s", jobId, jobResp.JobStatus)
				}
			}
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for NHN RDS job %s", jobId)
		case <-time.After(3 * time.Second):
		}
	}
}

// createNHNRDSDBSchema creates a DB schema on an NHN Cloud RDS instance via the v3.0 REST API.
func createNHNRDSDBSchema(connName, dbInstanceId, dbSchemaName string) error {
	appKey, userAccessKey, secretAccessKey, err := getNHNRDSCredentials(connName)
	if err != nil {
		return fmt.Errorf("failed to get NHN RDS credentials: %w", err)
	}
	endpoint, err := getNHNRDSEndpoint(connName)
	if err != nil {
		return fmt.Errorf("failed to get NHN RDS endpoint: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	url := fmt.Sprintf("%s/v3.0/db-instances/%s/db-schemas", endpoint, dbInstanceId)
	body, statusCode, err := nhnRDSCallRaw(ctx, http.MethodPost, url, appKey, userAccessKey, secretAccessKey,
		map[string]string{"dbSchemaName": dbSchemaName})
	if err != nil {
		return fmt.Errorf("NHN RDS create DB schema API call failed: %w", err)
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("NHN RDS create DB schema API returned HTTP %d: %s", statusCode, string(body))
	}

	var createResp struct {
		JobId string `json:"jobId"`
	}
	if err := json.Unmarshal(body, &createResp); err != nil {
		return fmt.Errorf("failed to parse NHN RDS create DB schema response: %w", err)
	}
	if createResp.JobId == "" {
		return nil
	}
	return pollNHNRDSJob(ctx, endpoint, appKey, userAccessKey, secretAccessKey, createResp.JobId)
}

// dropNHNRDSDBSchema deletes a DB schema from an NHN Cloud RDS instance via the v3.0 REST API.
// It first lists schemas to find the schema ID by name, then calls the delete endpoint.
func dropNHNRDSDBSchema(connName, dbInstanceId, dbSchemaName string) error {
	appKey, userAccessKey, secretAccessKey, err := getNHNRDSCredentials(connName)
	if err != nil {
		return fmt.Errorf("failed to get NHN RDS credentials: %w", err)
	}
	endpoint, err := getNHNRDSEndpoint(connName)
	if err != nil {
		return fmt.Errorf("failed to get NHN RDS endpoint: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// List schemas to find the schema ID by name.
	listURL := fmt.Sprintf("%s/v3.0/db-instances/%s/db-schemas", endpoint, dbInstanceId)
	listBody, statusCode, err := nhnRDSCallRaw(ctx, http.MethodGet, listURL, appKey, userAccessKey, secretAccessKey, nil)
	if err != nil {
		return fmt.Errorf("NHN RDS list DB schemas API call failed: %w", err)
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("NHN RDS list DB schemas API returned HTTP %d: %s", statusCode, string(listBody))
	}

	var listResp struct {
		DBSchemas []struct {
			DBSchemaId   string `json:"dbSchemaId"`
			DBSchemaName string `json:"dbSchemaName"`
		} `json:"dbSchemas"`
	}
	if err := json.Unmarshal(listBody, &listResp); err != nil {
		return fmt.Errorf("failed to parse NHN RDS list DB schemas response: %w", err)
	}

	schemaId := ""
	for _, s := range listResp.DBSchemas {
		if strings.EqualFold(s.DBSchemaName, dbSchemaName) {
			schemaId = s.DBSchemaId
			break
		}
	}
	if schemaId == "" {
		return fmt.Errorf("NHN RDS DB schema %q not found in instance %s", dbSchemaName, dbInstanceId)
	}

	// Delete the schema by ID.
	deleteURL := fmt.Sprintf("%s/v3.0/db-instances/%s/db-schemas/%s", endpoint, dbInstanceId, schemaId)
	delBody, statusCode, err := nhnRDSCallRaw(ctx, http.MethodDelete, deleteURL, appKey, userAccessKey, secretAccessKey, nil)
	if err != nil {
		return fmt.Errorf("NHN RDS delete DB schema API call failed: %w", err)
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("NHN RDS delete DB schema API returned HTTP %d: %s", statusCode, string(delBody))
	}

	var delResp struct {
		JobId string `json:"jobId"`
	}
	if err := json.Unmarshal(delBody, &delResp); err != nil {
		return fmt.Errorf("failed to parse NHN RDS delete DB schema response: %w", err)
	}
	if delResp.JobId == "" {
		return nil
	}
	return pollNHNRDSJob(ctx, endpoint, appKey, userAccessKey, secretAccessKey, delResp.JobId)
}

// spiderRDBMSDatabaseAPIResult is returned by trySpiderDatabaseListAPI / trySpiderDatabaseCreateAPI.
type spiderRDBMSDatabaseAPIResult struct {
	NotSupported bool     // true → driver does not implement RDBMSDatabaseManager; caller must fall back to SQL
	Databases    []string // populated by list
}

// trySpiderDatabaseListAPI calls the Spider CSP-native list-databases endpoint.
// Returns (result, nil) on success; (result with NotSupported=true, nil) when HTTP 501.
func trySpiderDatabaseListAPI(connName, rdbmsName string) (spiderRDBMSDatabaseAPIResult, error) {
	url := "http://localhost" + cr.ServerPort + "/spider/rdbms/" + rdbmsName + "/databases"
	body, _ := json.Marshal(map[string]string{"ConnectionName": connName})
	req, _ := http.NewRequest("GET", url, strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	setBasicAuthIfConfigured(req)
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return spiderRDBMSDatabaseAPIResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotImplemented {
		return spiderRDBMSDatabaseAPIResult{NotSupported: true}, nil
	}
	if resp.StatusCode != http.StatusOK {
		var errBody map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return spiderRDBMSDatabaseAPIResult{}, fmt.Errorf("%s", errBody["message"])
	}
	var result struct {
		Databases []string `json:"Databases"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return spiderRDBMSDatabaseAPIResult{}, err
	}
	return spiderRDBMSDatabaseAPIResult{Databases: result.Databases}, nil
}

// trySpiderDatabaseCreateAPI calls the Spider CSP-native create-database endpoint.
// Returns (false, nil) on success; (true, nil) when HTTP 501 (caller must fall back to SQL).
func trySpiderDatabaseCreateAPI(connName, rdbmsName, dbName string) (notSupported bool, err error) {
	url := "http://localhost" + cr.ServerPort + "/spider/rdbms/" + rdbmsName + "/databases"
	body, _ := json.Marshal(map[string]string{"ConnectionName": connName, "DatabaseName": dbName})
	req, _ := http.NewRequest("POST", url, strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	setBasicAuthIfConfigured(req)
	resp, err2 := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err2 != nil {
		return false, err2
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotImplemented {
		return true, nil
	}
	if resp.StatusCode != http.StatusOK {
		var errBody map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		msg := errBody["message"]
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return false, fmt.Errorf("%s", msg)
	}
	return false, nil
}

// trySpiderDatabaseDeleteAPI calls the Spider CSP-native delete-database endpoint.
// Returns (false, nil) on success; (true, nil) when HTTP 501 (caller must fall back to SQL).
func trySpiderDatabaseDeleteAPI(connName, rdbmsName, dbName string) (notSupported bool, err error) {
	url := "http://localhost" + cr.ServerPort + "/spider/rdbms/" + rdbmsName + "/databases/" + dbName
	body, _ := json.Marshal(map[string]string{"ConnectionName": connName})
	req, _ := http.NewRequest("DELETE", url, strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	setBasicAuthIfConfigured(req)
	resp, err2 := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err2 != nil {
		return false, err2
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotImplemented {
		return true, nil
	}
	if resp.StatusCode != http.StatusOK {
		var errBody map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		msg := errBody["message"]
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return false, fmt.Errorf("%s", msg)
	}
	return false, nil
}

// RDBMSListDatabases lists all databases via the Spider REST API.
// For CSPs without CSP-native API (AWS, IBM), the caller supplies MasterUserPassword.
func RDBMSListDatabases(c echo.Context) error {
	rdbmsName := c.Param("Name")

	var req rdbmsQueryRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	if req.ConnectionName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ConnectionName is required"})
	}

	url := "http://localhost" + cr.ServerPort + "/spider/rdbms/" + rdbmsName + "/databases"
	body, _ := json.Marshal(map[string]string{"ConnectionName": req.ConnectionName, "MasterUserPassword": req.Password})
	httpReq, _ := http.NewRequest("GET", url, strings.NewReader(string(body)))
	httpReq.Header.Set("Content-Type", "application/json")
	setBasicAuthIfConfigured(httpReq)

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(httpReq)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		msg := errBody["message"]
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": msg})
	}

	var result struct {
		Databases []string `json:"Databases"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	dbs := result.Databases
	if dbs == nil {
		dbs = []string{}
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"databases": dbs})
}

// RDBMSCreateDatabase creates a new database via the Spider REST API.
// For CSPs without CSP-native API (AWS, IBM), Password is forwarded as MasterUserPassword.
func RDBMSCreateDatabase(c echo.Context) error {
	rdbmsName := c.Param("Name")

	var req struct {
		ConnectionName string `json:"ConnectionName"`
		Password       string `json:"Password"`
		DatabaseName   string `json:"DatabaseName"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	if req.ConnectionName == "" || req.DatabaseName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ConnectionName and DatabaseName are required"})
	}
	if !isValidIdentifier(req.DatabaseName) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid database name. Use only letters, numbers, and underscores."})
	}

	url := "http://localhost" + cr.ServerPort + "/spider/rdbms/" + rdbmsName + "/databases"
	body, _ := json.Marshal(map[string]string{"ConnectionName": req.ConnectionName, "DatabaseName": req.DatabaseName, "MasterUserPassword": req.Password})
	httpReq, _ := http.NewRequest("POST", url, strings.NewReader(string(body)))
	httpReq.Header.Set("Content-Type", "application/json")
	setBasicAuthIfConfigured(httpReq)

	resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(httpReq)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		msg := errBody["message"]
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": msg})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "created", "database": req.DatabaseName})
}

// RDBMSDropDatabase drops a database via the Spider REST API.
// For CSPs without CSP-native API (AWS, IBM), Password is forwarded as MasterUserPassword.
func RDBMSDropDatabase(c echo.Context) error {
	rdbmsName := c.Param("Name")
	dbName := c.Param("DBName")

	var req rdbmsQueryRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	if req.ConnectionName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ConnectionName is required"})
	}
	if !isValidIdentifier(dbName) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid database name"})
	}

	url := "http://localhost" + cr.ServerPort + "/spider/rdbms/" + rdbmsName + "/databases/" + dbName
	body, _ := json.Marshal(map[string]string{"ConnectionName": req.ConnectionName, "MasterUserPassword": req.Password})
	httpReq, _ := http.NewRequest("DELETE", url, strings.NewReader(string(body)))
	httpReq.Header.Set("Content-Type", "application/json")
	setBasicAuthIfConfigured(httpReq)

	resp, err := (&http.Client{Timeout: 60 * time.Second}).Do(httpReq)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		msg := errBody["message"]
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": msg})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "dropped", "database": dbName})
}

// RDBMSListTables lists all tables in the database.
func RDBMSListTables(c echo.Context) error {
	rdbmsName := c.Param("Name")

	var req rdbmsQueryRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	if req.ConnectionName == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ConnectionName and Password are required"})
	}

	info, err := fetchRDBMSInfo(req.ConnectionName, rdbmsName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	db, driverName, err := openDBConnection(info, req.Password, req.DatabaseName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer db.Close()

	var query string
	switch driverName {
	case "mysql":
		query = "SHOW TABLES"
	case "postgres":
		query = "SELECT tablename FROM pg_tables WHERE schemaname = 'public'"
	}

	rows, err := db.Query(query)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to list tables: %v", err)})
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to scan table name: %v", err)})
		}
		tables = append(tables, name)
	}
	if tables == nil {
		tables = []string{}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"tables": tables})
}

// RDBMSDescribeTable returns column information for a table.
func RDBMSDescribeTable(c echo.Context) error {
	rdbmsName := c.Param("Name")
	tableName := c.Param("TableName")

	if !isValidIdentifier(tableName) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid table name"})
	}

	var req rdbmsQueryRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	if req.ConnectionName == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ConnectionName and Password are required"})
	}

	info, err := fetchRDBMSInfo(req.ConnectionName, rdbmsName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	db, driverName, err := openDBConnection(info, req.Password, req.DatabaseName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer db.Close()

	type colDescription struct {
		Name     string `json:"Name"`
		Type     string `json:"Type"`
		Nullable string `json:"Nullable"`
		Key      string `json:"Key"`
		Default  string `json:"Default"`
	}

	var columns []colDescription

	switch driverName {
	case "mysql":
		rows, err := db.Query("DESCRIBE `" + tableName + "`")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to describe table: %v", err)})
		}
		defer rows.Close()

		for rows.Next() {
			var field, colType, null, key string
			var defVal, extra sql.NullString
			if err := rows.Scan(&field, &colType, &null, &key, &defVal, &extra); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to scan column info: %v", err)})
			}
			def := ""
			if defVal.Valid {
				def = defVal.String
			}
			columns = append(columns, colDescription{Name: field, Type: colType, Nullable: null, Key: key, Default: def})
		}

	case "postgres":
		rows, err := db.Query(`
			SELECT column_name, data_type, is_nullable,
				CASE WHEN pk.column_name IS NOT NULL THEN 'PRI' ELSE '' END AS key,
				COALESCE(column_default, '') AS column_default
			FROM information_schema.columns c
			LEFT JOIN (
				SELECT ku.column_name
				FROM information_schema.table_constraints tc
				JOIN information_schema.key_column_usage ku ON tc.constraint_name = ku.constraint_name
				WHERE tc.constraint_type = 'PRIMARY KEY' AND tc.table_name = $1
			) pk ON c.column_name = pk.column_name
			WHERE c.table_name = $1 AND c.table_schema = 'public'
			ORDER BY c.ordinal_position`, tableName)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to describe table: %v", err)})
		}
		defer rows.Close()

		for rows.Next() {
			var name, colType, nullable, key, def string
			if err := rows.Scan(&name, &colType, &nullable, &key, &def); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to scan column info: %v", err)})
			}
			columns = append(columns, colDescription{Name: name, Type: colType, Nullable: nullable, Key: key, Default: def})
		}
	}

	if columns == nil {
		columns = []colDescription{}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"columns": columns})
}

// RDBMSListRows returns rows from a table (with LIMIT).
func RDBMSListRows(c echo.Context) error {
	rdbmsName := c.Param("Name")
	tableName := c.Param("TableName")

	if !isValidIdentifier(tableName) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid table name"})
	}

	var req rdbmsQueryRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	if req.ConnectionName == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ConnectionName and Password are required"})
	}

	info, err := fetchRDBMSInfo(req.ConnectionName, rdbmsName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	db, driverName, err := openDBConnection(info, req.Password, req.DatabaseName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer db.Close()

	var query string
	switch driverName {
	case "mysql":
		query = "SELECT * FROM `" + tableName + "` LIMIT 200"
	case "postgres":
		query = `SELECT * FROM "` + tableName + `" LIMIT 200`
	}

	rows, err := db.Query(query)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to query rows: %v", err)})
	}
	defer rows.Close()

	colNames, err := rows.Columns()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to get column names: %v", err)})
	}

	var result []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(colNames))
		valuePtrs := make([]interface{}, len(colNames))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to scan row: %v", err)})
		}

		row := make(map[string]interface{})
		for i, col := range colNames {
			val := values[i]
			switch v := val.(type) {
			case []byte:
				row[col] = string(v)
			case nil:
				row[col] = nil
			default:
				row[col] = v
			}
		}
		result = append(result, row)
	}

	if result == nil {
		result = []map[string]interface{}{}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"columns": colNames,
		"rows":    result,
		"count":   len(result),
	})
}

// RDBMSCreateTable creates a new table.
func RDBMSCreateTable(c echo.Context) error {
	rdbmsName := c.Param("Name")

	var req createTableRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	if req.ConnectionName == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ConnectionName and Password are required"})
	}
	if !isValidIdentifier(req.TableName) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid table name (use alphanumeric and underscore only)"})
	}
	if len(req.Columns) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "At least one column is required"})
	}

	// Validate column names
	for _, col := range req.Columns {
		if !isValidIdentifier(col.Name) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Invalid column name: %s", col.Name)})
		}
	}

	info, err := fetchRDBMSInfo(req.ConnectionName, rdbmsName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	db, driverName, err := openDBConnection(info, req.Password, req.DatabaseName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer db.Close()

	// Build CREATE TABLE statement
	var colDefs []string
	var pkCols []string
	for _, col := range req.Columns {
		def := quoteIdentifier(driverName, col.Name) + " " + col.Type
		if col.NotNull {
			def += " NOT NULL"
		}
		colDefs = append(colDefs, def)
		if col.PrimaryKey {
			pkCols = append(pkCols, quoteIdentifier(driverName, col.Name))
		}
	}
	if len(pkCols) > 0 {
		colDefs = append(colDefs, "PRIMARY KEY ("+strings.Join(pkCols, ", ")+")")
	}

	ddl := "CREATE TABLE " + quoteIdentifier(driverName, req.TableName) + " (\n" + strings.Join(colDefs, ",\n") + "\n)"

	if _, err := db.Exec(ddl); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to create table: %v", err)})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "created", "table": req.TableName})
}

// RDBMSDropTable drops a table.
func RDBMSDropTable(c echo.Context) error {
	rdbmsName := c.Param("Name")
	tableName := c.Param("TableName")

	if !isValidIdentifier(tableName) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid table name"})
	}

	var req rdbmsQueryRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	if req.ConnectionName == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ConnectionName and Password are required"})
	}

	info, err := fetchRDBMSInfo(req.ConnectionName, rdbmsName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	db, driverName, err := openDBConnection(info, req.Password, req.DatabaseName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer db.Close()

	ddl := "DROP TABLE " + quoteIdentifier(driverName, tableName)
	if _, err := db.Exec(ddl); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to drop table: %v", err)})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "dropped", "table": tableName})
}

// RDBMSInsertRow inserts a row into a table.
func RDBMSInsertRow(c echo.Context) error {
	rdbmsName := c.Param("Name")
	tableName := c.Param("TableName")

	if !isValidIdentifier(tableName) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid table name"})
	}

	var req insertRowRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	if req.ConnectionName == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ConnectionName and Password are required"})
	}
	if len(req.Values) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Values are required"})
	}

	// Validate column names
	for col := range req.Values {
		if !isValidIdentifier(col) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Invalid column name: %s", col)})
		}
	}

	info, err := fetchRDBMSInfo(req.ConnectionName, rdbmsName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	db, driverName, err := openDBConnection(info, req.Password, req.DatabaseName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer db.Close()

	// Build INSERT statement with parameterized values
	var cols []string
	var placeholders []string
	var args []interface{}
	i := 1
	for col, val := range req.Values {
		cols = append(cols, quoteIdentifier(driverName, col))
		if driverName == "postgres" {
			placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		} else {
			placeholders = append(placeholders, "?")
		}
		args = append(args, val)
		i++
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		quoteIdentifier(driverName, tableName),
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "))

	if _, err := db.Exec(query, args...); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to insert row: %v", err)})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "inserted"})
}

// RDBMSDeleteRow deletes rows from a table matching WHERE conditions.
func RDBMSDeleteRow(c echo.Context) error {
	rdbmsName := c.Param("Name")
	tableName := c.Param("TableName")

	if !isValidIdentifier(tableName) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid table name"})
	}

	var req deleteRowRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	if req.ConnectionName == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ConnectionName and Password are required"})
	}
	if len(req.Where) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "WHERE conditions are required"})
	}

	// Validate column names
	for col := range req.Where {
		if !isValidIdentifier(col) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Invalid column name: %s", col)})
		}
	}

	info, err := fetchRDBMSInfo(req.ConnectionName, rdbmsName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	db, driverName, err := openDBConnection(info, req.Password, req.DatabaseName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer db.Close()

	// Build DELETE statement with parameterized WHERE
	var conditions []string
	var args []interface{}
	i := 1
	for col, val := range req.Where {
		if driverName == "postgres" {
			conditions = append(conditions, fmt.Sprintf("%s = $%d", quoteIdentifier(driverName, col), i))
		} else {
			conditions = append(conditions, quoteIdentifier(driverName, col)+" = ?")
		}
		args = append(args, val)
		i++
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s",
		quoteIdentifier(driverName, tableName),
		strings.Join(conditions, " AND "))

	result, err := db.Exec(query, args...)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to delete rows: %v", err)})
	}

	affected, _ := result.RowsAffected()
	return c.JSON(http.StatusOK, map[string]interface{}{"status": "deleted", "affected": affected})
}

// quoteIdentifier quotes a SQL identifier based on the driver.
func quoteIdentifier(driverName, name string) string {
	switch driverName {
	case "mysql":
		return "`" + name + "`"
	case "postgres":
		return `"` + name + `"`
	default:
		return `"` + name + `"`
	}
}
