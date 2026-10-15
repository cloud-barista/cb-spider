// Command collationprobe verifies that CB-Spider's Collation parameter on the RDBMS
// "create database" API (see api-runtime/common-runtime/RDBMSManager.go's CreateRDBMSDatabase
// and rdbmsDatabaseOptionsManager) actually takes effect on the database engine, not just that
// the API call itself returns "created". This is the collation counterpart of
// ../../charset-test/charsetprobe -- see that file for the full rationale (CB-Spider does not
// validate Collation against a fixed list, and at least one CSP has been observed to silently
// accept a request without actually applying it).
//
// Charset is held constant at "utf8mb4" in both scenarios here -- only Collation varies -- so
// that any observed difference in behavior is attributable to collation alone, not a mix of
// charset and collation effects. (Testing "Collation given, Charset omitted" is a separate,
// not-yet-covered case: whether CB-Spider's CSP-native drivers correctly infer the charset from a
// collation name alone, the way plain MySQL does, hasn't been verified against every CSP's API --
// see README.)
//
// For each of two collations -- utf8mb4_general_ci (case-insensitive) and utf8mb4_bin
// (byte-exact, case-sensitive) -- chosen because both have existed, under these exact names, in
// every MySQL/MariaDB version that supports utf8mb4 at all (unlike the engine- and
// version-specific "default" collations such as utf8mb4_0900_ai_ci or utf8mb4_uca1400_ai_ci,
// which would make this test's own expectations a moving target) -- this tool:
//  1. Creates a database via CB-Spider's POST /spider/rdbms/{Name}/databases with that Collation.
//  2. Confirms information_schema.SCHEMATA.DEFAULT_COLLATION_NAME for that database matches
//     (metadata check).
//  3. Creates a table with no explicit column/table collation (so it inherits the database
//     default), inserts "TestWord", then checks whether `txt = 'testword'` (lowercase) matches --
//     a plain ASCII case comparison needs no special Unicode handling and is unambiguous under
//     any locale (functional check).
//
// Under utf8mb4_general_ci, the comparison MUST match (case-insensitive). Under utf8mb4_bin, it
// MUST NOT match (byte-exact). Either collation producing the other's expected result means the
// requested Collation was not actually honored by that CSP.
//
// Usage:
//
//	go run . -connection aws-config01 -rdbms cb-spider-mysql-test -username myadmin -password 'Password123!' -csp-name AWS
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

// fixedCharset is held constant across both scenarios so that only collation varies.
const fixedCharset = "utf8mb4"

// testWord is inserted as-is; its lowercase form is used for the comparison query. Plain ASCII
// avoids any locale-specific case-folding ambiguity (e.g. Turkish dotless i) that could make the
// expected result depend on more than just _ci vs _bin.
const testWord = "TestWord"

type scenario struct {
	label     string // display name, also used in the result file
	collation string // value sent as the Collation field to CB-Spider's create-database API
	// caseInsensitiveMatch is what a correct implementation of this collation must produce when
	// comparing testWord to its own lowercased form.
	caseInsensitiveMatch bool
}

var scenarios = []scenario{
	{label: "ci", collation: "utf8mb4_general_ci", caseInsensitiveMatch: true},
	{label: "bin", collation: "utf8mb4_bin", caseInsensitiveMatch: false},
}

type rdbmsInfo struct {
	Endpoint string `json:"Endpoint"`
	Message  string `json:"message"`
}

type simpleMsg struct {
	Message string `json:"message"`
}

// outcome is PASS/FAIL/SKIP, mirroring ../../charset-test/charsetprobe's convention so
// result-table tooling stays consistent across test directories.
type outcome string

const (
	pass outcome = "PASS"
	fail outcome = "FAIL"
	skip outcome = "SKIP"
)

// scenarioResult holds the three independently-graded checks for one collation scenario.
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
	apiTimeout := flag.Duration("api-timeout", 120*time.Second, "Per-call timeout for CB-Spider REST API calls (GetRDBMS/CreateDatabase/DeleteDatabase). Deliberately generous: CreateDatabase on some CSPs (observed on Azure) polls a CSP-native async operation to completion before responding, which can take well over a minute -- this is normal CSP latency, not a CB-Spider problem.")
	resultFile := flag.String("result-file", "", "Optional path to write a pipe-separated result line")
	flag.Parse()

	if *connectionName == "" || *rdbmsName == "" || *username == "" || *password == "" {
		fmt.Fprintln(os.Stderr, "usage: collationprobe -connection <name> -rdbms <name> -username <username> -password <password> [flags]")
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
	logf("[%s] Starting collation test (RDBMS='%s')...", start.Format("2006-01-02 15:04:05"), *rdbmsName)

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

	// ── Run each collation scenario ─────────────────────────────────────────
	overall := pass
	for _, s := range scenarios {
		logf("── Scenario: Collation=%s ──", s.collation)
		r := runScenario(logf, client, *connectionName, *rdbmsName, *username, *password, host, port, s, *timeout)
		results[s.label] = r
		logf("Scenario %s: Create=%s Meta=%s Function=%s", s.collation, r.create, r.meta, r.function)
		if r.create != pass || r.meta != pass || r.function != pass {
			overall = fail
		}
	}

	elapsed := formatElapsed(time.Since(start))
	logf("Collation test %s (elapsed: %s)", overall, elapsed)
	writeResult()

	if overall != pass {
		return 1
	}
	return 0
}

// runScenario creates a database with fixedCharset and the scenario's collation, verifies its
// metadata, and functionally verifies comparison behavior, then deletes the database
// (best-effort) before returning.
func runScenario(logf func(string, ...interface{}), client *spiderClient, connectionName, rdbmsName, username, password, host, port string, s scenario, timeout time.Duration) scenarioResult {
	dbName := "cbspidercol" + s.label

	// Pre-cleanup: remove a leftover database from a previous aborted run (ignore errors).
	_ = client.deleteDatabase(connectionName, rdbmsName, dbName, username, password)

	r := scenarioResult{}

	// ── CreateDatabase with fixed Charset + explicit Collation ──────────────
	if err := client.createDatabase(connectionName, rdbmsName, dbName, username, password, fixedCharset, s.collation); err != nil {
		logf("CreateDatabase(%s): FAILED: %v", s.collation, err)
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
	cfg.Collation = "utf8mb4_general_ci" // the connection's own session charset/collation; unrelated to the column collation under test
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
	var actualCollation string
	metaCtx, metaCancel := context.WithTimeout(context.Background(), timeout)
	err = db.QueryRowContext(metaCtx,
		"SELECT DEFAULT_COLLATION_NAME FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = ?", dbName,
	).Scan(&actualCollation)
	metaCancel()
	if err != nil {
		logf("Metadata check: query failed: %v", err)
		r.meta = fail
	} else if actualCollation != s.collation {
		logf("Metadata check: expected DEFAULT_COLLATION_NAME=%q, got %q", s.collation, actualCollation)
		r.meta = fail
	} else {
		logf("Metadata check: DEFAULT_COLLATION_NAME=%q (matches)", actualCollation)
		r.meta = pass
	}

	// ── Functional check: CREATE TABLE (inherits DB collation) + case comparison ──
	ddlCtx, ddlCancel := context.WithTimeout(context.Background(), timeout)
	_, err = db.ExecContext(ddlCtx, "CREATE TABLE collation_probe (txt VARCHAR(50))")
	ddlCancel()
	if err != nil {
		logf("Functional check: CREATE TABLE failed: %v", err)
		r.function = fail
		r.detail = err.Error()
		return r
	}

	insCtx, insCancel := context.WithTimeout(context.Background(), timeout)
	_, err = db.ExecContext(insCtx, "INSERT INTO collation_probe (txt) VALUES (?)", testWord)
	insCancel()
	if err != nil {
		logf("Functional check: INSERT failed: %v", err)
		r.function = fail
		r.detail = err.Error()
		return r
	}

	var matchCount int
	cmpCtx, cmpCancel := context.WithTimeout(context.Background(), timeout)
	err = db.QueryRowContext(cmpCtx,
		"SELECT COUNT(*) FROM collation_probe WHERE txt = ?", strings.ToLower(testWord),
	).Scan(&matchCount)
	cmpCancel()
	if err != nil {
		logf("Functional check: comparison query failed: %v", err)
		r.function = fail
		r.detail = err.Error()
		return r
	}

	gotMatch := matchCount > 0
	if gotMatch == s.caseInsensitiveMatch {
		logf("Functional check: %q = %q comparison returned match=%v under %s, as expected",
			testWord, strings.ToLower(testWord), gotMatch, s.collation)
		r.function = pass
	} else {
		logf("Functional check: %q = %q comparison returned match=%v under %s -- expected match=%v -- the Collation request was NOT honored",
			testWord, strings.ToLower(testWord), gotMatch, s.collation, s.caseInsensitiveMatch)
		r.function = fail
		r.detail = "case-comparison behavior did not match the requested collation"
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
