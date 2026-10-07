// Command charsetprobe verifies that CB-Spider's Charset parameter on the RDBMS
// "create database" API (see api-runtime/common-runtime/RDBMSManager.go's CreateRDBMSDatabase
// and rdbmsDatabaseOptionsManager) actually takes effect on the database engine, not just that
// the API call itself returns "created". CB-Spider does not validate the Charset value against a
// fixed list (MySQL/MariaDB add new charsets over time) -- it only checks the string is safe to
// use as a SQL identifier and passes it straight through to the CSP API or SQL engine, so the only
// way to know whether a given CSP driver honored it (vs. silently falling back to its own default)
// is to connect directly to the created database and observe real behavior.
//
// For each of two charsets -- utf8mb4 (CB-Spider/most CSPs' default) and latin1 (a legacy,
// single-byte Western European charset -- see MySQL/MariaDB's own "Supported Character Sets"
// reference) -- this tool:
//  1. Creates a database via CB-Spider's POST /spider/rdbms/{Name}/databases with that Charset.
//  2. Confirms information_schema.SCHEMATA.DEFAULT_CHARACTER_SET_NAME for that database matches
//     (metadata check).
//  3. Creates a table with no explicit column/table charset (so it inherits the database
//     default) and inserts a string containing both Korean (3-byte UTF-8) and an emoji (4-byte
//     UTF-8) -- neither representable in latin1 -- then reads it back (functional check).
//
// Under utf8mb4, the functional check must round-trip the string byte-for-byte. Under latin1, the
// insert must either fail outright (strict sql_mode, the common case) or -- if it succeeds -- the
// round-tripped value must NOT match the original (non-strict sql_mode silently replaces
// unrepresentable characters). Either outcome proves the database is really using latin1, not
// silently defaulting to utf8mb4; an exact round-trip under the latin1 scenario would mean the
// Charset request was ignored by that CSP, which is exactly the regression this tool catches.
//
// This is the MariaDB-engine counterpart of ../../../rdbms-mysql-test/charset-test/charsetprobe
// (identical logic -- the charset/collation mechanics this tool exercises are the same on both
// engines; only the scope of CSPs that support MariaDB at all differs, see ../README.md) and
// mirrors ../../../rdbms-mysql-test/tls-test/tlsprobe's rationale for a standalone Go binary over
// shelling out to the `mysql` CLI/curl/jq: no external runtime dependency beyond the Go toolchain
// already needed to build CB-Spider, byte-identical behavior across OSes, and the same
// go-sql-driver/mysql driver CB-Spider's own SQL fallback path uses.
//
// Usage:
//
//	go run . -connection aws-config01 -rdbms cb-spider-mariadb-test -username myadmin -password 'Password123!' -csp-name AWS
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

// testString contains a 3-byte (Korean) and a 4-byte (emoji) UTF-8 character -- neither has any
// representation in latin1, so it functionally distinguishes a real utf8mb4 database from one
// that only claims to be latin1 but secretly stored the request as utf8mb4 anyway, and vice versa.
const testString = "한글테스트😀"

type scenario struct {
	label   string // display name, also used in the result file
	charset string // value sent as the Charset field to CB-Spider's create-database API
}

var scenarios = []scenario{
	{label: "utf8mb4", charset: "utf8mb4"},
	{label: "latin1", charset: "latin1"},
}

type rdbmsInfo struct {
	Endpoint string `json:"Endpoint"`
	Message  string `json:"message"`
}

type simpleMsg struct {
	Message string `json:"message"`
}

type databaseListResp struct {
	Databases []string `json:"Databases"`
	Message   string   `json:"message"`
}

// outcome is PASS/FAIL/SKIP, mirroring the bash suites' convention so result-table tooling stays
// consistent across test directories.
type outcome string

const (
	pass outcome = "PASS"
	fail outcome = "FAIL"
	skip outcome = "SKIP"
)

// scenarioResult holds the three independently-graded checks for one charset scenario.
type scenarioResult struct {
	create   outcome
	meta     outcome
	function outcome
	detail   string // last error/explanation, for logging
}

func main() {
	os.Exit(run())
}

func run() int {
	spiderURL := flag.String("spider-url", envOr("SPIDER_URL", "http://localhost:1024"), "CB-Spider REST API URL")
	spiderAuth := flag.String("spider-auth", envOr("SPIDER_AUTH", "admin:****"), "Basic auth as user:pass")
	connectionName := flag.String("connection", "", "Spider connection config name (required)")
	rdbmsName := flag.String("rdbms", "", "RDBMS instance name (required)")
	username := flag.String("username", "", "MasterUserName set when the instance was created (required) -- CB-Spider's GET /spider/rdbms/{Name} no longer returns it, see RDBMSManager.go's redactRDBMSMasterCredentials")
	password := flag.String("password", "", "MasterUserPassword (required)")
	cspName := flag.String("csp-name", "", "Display name for log lines (default: -connection value)")
	timeout := flag.Duration("timeout", 15*time.Second, "Per-operation timeout for direct SQL connections (dial/ping/query/exec) -- these are always fast once the server is reachable, so a short timeout here is a meaningful failure signal")
	apiTimeout := flag.Duration("api-timeout", 120*time.Second, "Per-call timeout for CB-Spider REST API calls (GetRDBMS/CreateDatabase/DeleteDatabase). Deliberately generous: CreateDatabase on some CSPs (observed on Azure) polls a CSP-native async operation to completion before responding, which can take well over a minute -- this is normal CSP latency, not a CB-Spider problem, and ../database-test's curl-based equivalent has no client timeout at all for the same reason.")
	resultFile := flag.String("result-file", "", "Optional path to write a pipe-separated result line")
	flag.Parse()

	if *connectionName == "" || *rdbmsName == "" || *username == "" || *password == "" {
		fmt.Fprintln(os.Stderr, "usage: charsetprobe -connection <name> -rdbms <name> -username <username> -password <password> [flags]")
		flag.PrintDefaults()
		return 2
	}

	name := *cspName
	if name == "" {
		name = *connectionName
	}
	logf := func(format string, args ...interface{}) {
		fmt.Printf("[%s] %s\n", name, fmt.Sprintf(format, args...))
	}

	start := time.Now()
	logf("[%s] Starting charset test (RDBMS='%s')...", start.Format("2006-01-02 15:04:05"), *rdbmsName)

	results := make(map[string]scenarioResult, len(scenarios))

	writeResult := func() {
		if *resultFile == "" {
			return
		}
		elapsed := formatElapsed(time.Since(start))
		fields := []string{name}
		for _, s := range scenarios {
			r := results[s.label]
			if r.create == "" {
				r = scenarioResult{create: skip, meta: skip, function: skip}
			}
			fields = append(fields, string(r.create), string(r.meta), string(r.function))
		}
		fields = append(fields, elapsed)
		line := strings.Join(fields, "|") + "\n"
		if err := os.WriteFile(*resultFile, []byte(line), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "[%s] WARNING: failed to write result file %s: %v\n", name, *resultFile, err)
		}
	}

	abort := func(label, msg string) int {
		logf("ERROR on %s: %s", label, msg)
		writeResult()
		return 1
	}

	client := &spiderClient{baseURL: *spiderURL, auth: *spiderAuth, httpClient: &http.Client{Timeout: *apiTimeout}}

	// ── Get RDBMS Info (Endpoint) ────────────────────────────────────────────
	logf("GetRDBMS: fetching endpoint")
	var info rdbmsInfo
	infoURL := fmt.Sprintf("%s/spider/rdbms/%s?ConnectionName=%s", *spiderURL, url.PathEscape(*rdbmsName), url.QueryEscape(*connectionName))
	if err := client.getJSON(infoURL, &info); err != nil {
		return abort("GetRDBMS", err.Error())
	}
	if info.Endpoint == "" {
		return abort("GetRDBMS", fmt.Sprintf("empty Endpoint in response (message: %q)", info.Message))
	}
	host, port := splitHostPort(info.Endpoint)
	logf("Endpoint: %s:%s (user: %s)", host, port, *username)

	// ── Run each charset scenario ───────────────────────────────────────────
	overall := pass
	for _, s := range scenarios {
		logf("── Scenario: Charset=%s ──", s.charset)
		r := runScenario(logf, client, *connectionName, *rdbmsName, *username, *password, host, port, s, *timeout)
		results[s.label] = r
		logf("Scenario %s: Create=%s Meta=%s Function=%s", s.charset, r.create, r.meta, r.function)
		if r.create != pass || r.meta != pass || r.function != pass {
			overall = fail
		}
	}

	elapsed := formatElapsed(time.Since(start))
	logf("Charset test %s (elapsed: %s)", overall, elapsed)
	writeResult()

	if overall != pass {
		return 1
	}
	return 0
}

// runScenario creates a database with the given charset, verifies its metadata, and functionally
// verifies storage behavior, then deletes the database (best-effort) before returning.
func runScenario(logf func(string, ...interface{}), client *spiderClient, connectionName, rdbmsName, username, password, host, port string, s scenario, timeout time.Duration) scenarioResult {
	dbName := "cbspidercs" + s.charset

	// Pre-cleanup: remove a leftover database from a previous aborted run (ignore errors).
	_ = client.deleteDatabase(connectionName, rdbmsName, dbName, username, password)

	r := scenarioResult{}

	// ── CreateDatabase with explicit Charset ───────────────────────────────
	if err := client.createDatabase(connectionName, rdbmsName, dbName, username, password, s.charset, ""); err != nil {
		logf("CreateDatabase(%s): FAILED: %v", s.charset, err)
		r.create, r.meta, r.function = fail, skip, skip
		r.detail = err.Error()
		return r
	}
	r.create = pass
	defer func() {
		if err := client.deleteDatabase(connectionName, rdbmsName, dbName, username, password); err != nil {
			logf("WARNING: cleanup DeleteDatabase(%s) failed: %v", dbName, err)
		}
	}()

	// ── Connect directly to the new database ───────────────────────────────
	cfg := mysql.NewConfig()
	cfg.User = username
	cfg.Passwd = password
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(host, port)
	cfg.DBName = dbName
	// The connection's own session charset is always utf8mb4, regardless of which charset is
	// under test: this is what lets a real client send proper UTF-8 bytes over the wire in the
	// first place. What we're testing is what the SERVER does when storing those bytes into a
	// column that has no explicit charset of its own and so inherits dbName's default -- not
	// what charset the client-server wire protocol itself uses.
	cfg.Collation = "utf8mb4_general_ci"
	cfg.Timeout = timeout
	cfg.TLSConfig = "preferred" // matches RDBMSManager.go's own SQL fallback connections

	connector, err := mysql.NewConnector(cfg)
	if err != nil {
		r.meta, r.function = fail, skip
		r.detail = fmt.Sprintf("mysql.NewConnector: %v", err)
		return r
	}
	db := sql.OpenDB(connector)
	defer db.Close()
	db.SetMaxOpenConns(1)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		logf("Connect to database %q: FAILED: %v", dbName, err)
		r.meta, r.function = fail, skip
		r.detail = err.Error()
		return r
	}

	// ── Metadata check: information_schema.SCHEMATA ─────────────────────────
	var actualCharset string
	metaCtx, metaCancel := context.WithTimeout(context.Background(), timeout)
	err = db.QueryRowContext(metaCtx,
		"SELECT DEFAULT_CHARACTER_SET_NAME FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = ?", dbName,
	).Scan(&actualCharset)
	metaCancel()
	if err != nil {
		logf("Metadata check: query failed: %v", err)
		r.meta = fail
	} else if actualCharset != s.charset {
		logf("Metadata check: expected DEFAULT_CHARACTER_SET_NAME=%q, got %q", s.charset, actualCharset)
		r.meta = fail
	} else {
		logf("Metadata check: DEFAULT_CHARACTER_SET_NAME=%q (matches)", actualCharset)
		r.meta = pass
	}

	// ── Functional check: CREATE TABLE (inherits DB charset) + round-trip INSERT/SELECT ──
	ddlCtx, ddlCancel := context.WithTimeout(context.Background(), timeout)
	_, err = db.ExecContext(ddlCtx, "CREATE TABLE charset_probe (txt VARCHAR(50))")
	ddlCancel()
	if err != nil {
		logf("Functional check: CREATE TABLE failed: %v", err)
		r.function = fail
		r.detail = err.Error()
		return r
	}

	insCtx, insCancel := context.WithTimeout(context.Background(), timeout)
	_, insErr := db.ExecContext(insCtx, "INSERT INTO charset_probe (txt) VALUES (?)", testString)
	insCancel()

	switch {
	case s.charset == "utf8mb4":
		// Must succeed, and must round-trip exactly.
		if insErr != nil {
			logf("Functional check: expected INSERT to succeed under utf8mb4, but it failed: %v", insErr)
			r.function = fail
			r.detail = insErr.Error()
			return r
		}
		var got string
		selCtx, selCancel := context.WithTimeout(context.Background(), timeout)
		err = db.QueryRowContext(selCtx, "SELECT txt FROM charset_probe LIMIT 1").Scan(&got)
		selCancel()
		if err != nil {
			logf("Functional check: SELECT back failed: %v", err)
			r.function = fail
			r.detail = err.Error()
			return r
		}
		if got != testString {
			logf("Functional check: round-trip mismatch under utf8mb4: sent %q, got %q", testString, got)
			r.function = fail
			r.detail = "round-trip mismatch"
			return r
		}
		logf("Functional check: INSERT/SELECT round-tripped exactly under utf8mb4, as expected")
		r.function = pass

	default: // latin1 (or any future non-Unicode scenario)
		if insErr != nil {
			// Expected under strict sql_mode: latin1 cannot represent the test string at all.
			logf("Functional check: INSERT correctly rejected under %s (%v)", s.charset, insErr)
			r.function = pass
			return r
		}
		// Non-strict sql_mode: the insert "succeeded" but may have silently mangled the value.
		// That's still proof the column is really latin1, as long as it did NOT round-trip
		// exactly -- an exact match here would mean the charset request was silently ignored.
		var got string
		selCtx, selCancel := context.WithTimeout(context.Background(), timeout)
		err = db.QueryRowContext(selCtx, "SELECT txt FROM charset_probe LIMIT 1").Scan(&got)
		selCancel()
		if err != nil {
			logf("Functional check: SELECT back failed: %v", err)
			r.function = fail
			r.detail = err.Error()
			return r
		}
		if got == testString {
			logf("Functional check: INSERT round-tripped EXACTLY under %s -- the Charset request was NOT honored (data that only utf8mb4 can hold survived a supposedly-latin1 column)", s.charset)
			r.function = fail
			r.detail = "charset request appears to have been ignored"
			return r
		}
		logf("Functional check: INSERT succeeded but value was mangled under %s (sent %q, got %q) -- consistent with a real latin1 column under non-strict sql_mode", s.charset, testString, got)
		r.function = pass
	}

	return r
}

// ── Minimal CB-Spider REST client ──────────────────────────────────────────────

type spiderClient struct {
	baseURL    string
	auth       string
	httpClient *http.Client
}

func (c *spiderClient) getJSON(rawURL string, out interface{}) error {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	c.setAuth(req)
	return c.doJSON(req, out)
}

func (c *spiderClient) createDatabase(connectionName, rdbmsName, dbName, username, password, charset, collation string) error {
	body := map[string]string{
		"ConnectionName":     connectionName,
		"DatabaseName":       dbName,
		"MasterUserName":     username,
		"MasterUserPassword": password,
	}
	if charset != "" {
		body["Charset"] = charset
	}
	if collation != "" {
		body["Collation"] = collation
	}
	payload, _ := json.Marshal(body)
	reqURL := fmt.Sprintf("%s/spider/rdbms/%s/databases", c.baseURL, url.PathEscape(rdbmsName))
	req, err := http.NewRequest(http.MethodPost, reqURL, strings.NewReader(string(payload)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.setAuth(req)

	var resp simpleMsg
	if err := c.doJSON(req, &resp); err != nil {
		return err
	}
	if resp.Message != "created" {
		return fmt.Errorf("unexpected response: %q", resp.Message)
	}
	return nil
}

func (c *spiderClient) deleteDatabase(connectionName, rdbmsName, dbName, username, password string) error {
	body := map[string]string{
		"ConnectionName":     connectionName,
		"MasterUserName":     username,
		"MasterUserPassword": password,
	}
	payload, _ := json.Marshal(body)
	reqURL := fmt.Sprintf("%s/spider/rdbms/%s/databases/%s", c.baseURL, url.PathEscape(rdbmsName), url.PathEscape(dbName))
	req, err := http.NewRequest(http.MethodDelete, reqURL, strings.NewReader(string(payload)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.setAuth(req)

	var resp simpleMsg
	if err := c.doJSON(req, &resp); err != nil {
		return err
	}
	if resp.Message != "deleted" {
		return fmt.Errorf("unexpected response: %q", resp.Message)
	}
	return nil
}

func (c *spiderClient) setAuth(req *http.Request) {
	if user, pass, ok := strings.Cut(c.auth, ":"); ok {
		req.SetBasicAuth(user, pass)
	}
}

func (c *spiderClient) doJSON(req *http.Request, out interface{}) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("failed to parse JSON response: %w (body: %s)", err, truncate(string(body), 300))
	}
	return nil
}

// ── Small helpers ───────────────────────────────────────────────────────────────

func splitHostPort(endpoint string) (host, port string) {
	h, p, err := net.SplitHostPort(endpoint)
	if err != nil {
		return endpoint, "3306"
	}
	return h, p
}

func formatElapsed(d time.Duration) string {
	sec := int(d.Seconds())
	if sec < 60 {
		return fmt.Sprintf("%ds", sec)
	}
	return fmt.Sprintf("%dm%ds", sec/60, sec%60)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
