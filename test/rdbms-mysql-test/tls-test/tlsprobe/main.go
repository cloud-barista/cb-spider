// Command tlsprobe tests the 4-scenario client TLS matrix (see ../README.md) against a
// CB-Spider-managed RDBMS instance's real endpoint, using go-sql-driver/mysql directly —
// the exact same driver CB-Spider itself uses (see api-runtime/common-runtime/RDBMSManager.go
// and RDBMSTLSProbe.go), already proven reliable throughout this project's own TLS handling.
//
// This replaces an earlier version of this test that shelled out to the `mysql` CLI: that
// approach turned out to be unreliable across machines (a "PASS"/"FAIL" could depend on
// which SSL library a given mysql-client build happened to link against, or whether the OS's
// default CA bundle was findable), and the `mysql` CLI, `curl`, and `jq` aren't reliably
// available cross-platform. A single Go binary is: no external dependency beyond a Go
// toolchain (already required to build CB-Spider itself), byte-identical behavior on
// Windows/Linux/macOS, and reuses a driver already trusted by CB-Spider's own code.
//
// Usage:
//
//	go run . -connection aws-config01 -rdbms cb-spider-mysql-test -password 'Password123!' -csp-name AWS
//
// Or build once and reuse the binary:
//
//	go build -o tlsprobe .
//	./tlsprobe -connection aws-config01 -rdbms cb-spider-mysql-test -password 'Password123!' -csp-name AWS
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"encoding/pem"
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

type rdbmsInfo struct {
	Endpoint       string `json:"Endpoint"`
	MasterUserName string `json:"MasterUserName"`
	Message        string `json:"message"`
}

type caCertInfo struct {
	PEM          string `json:"PEM"`
	Subject      string `json:"Subject"`
	Issuer       string `json:"Issuer"`
	IsSelfSigned bool   `json:"IsSelfSigned"`
}

type secureTransportInfo struct {
	Engine                 string      `json:"Engine"`
	RequireSecureTransport string      `json:"RequireSecureTransport"`
	TLSInUse               bool        `json:"TLSInUse"`
	CACertificate          *caCertInfo `json:"CACertificate"`
	CACertificateError     string      `json:"CACertificateError"`
	Message                string      `json:"message"`
}

// scenarioResult mirrors the bash suite's per-scenario grade tokens exactly, so existing
// result-table tooling (run-all-csp-tls-tests.sh) keeps working unmodified.
type scenarioResult string

const (
	pass      scenarioResult = "PASS"
	fail      scenarioResult = "FAIL"
	skip      scenarioResult = "SKIP"
	notApplic scenarioResult = "N/A"
)

func main() {
	os.Exit(run())
}

func run() int {
	spiderURL := flag.String("spider-url", envOr("SPIDER_URL", "http://localhost:1024"), "CB-Spider REST API URL")
	spiderAuth := flag.String("spider-auth", envOr("SPIDER_AUTH", "admin:****"), "Basic auth as user:pass")
	connectionName := flag.String("connection", "", "Spider connection config name (required)")
	rdbmsName := flag.String("rdbms", "", "RDBMS instance name (required)")
	password := flag.String("password", "", "MasterUserPassword (required)")
	cspName := flag.String("csp-name", "", "Display name for log lines (default: -connection value)")
	timeout := flag.Duration("timeout", 10*time.Second, "Per-attempt connection timeout")
	interAttemptDelay := flag.Duration("inter-attempt-delay", 3*time.Second, "Delay between scenario connection attempts (see comment at call sites -- some networks throttle bursts of new connections to the same destination with no gap at all)")
	resultFile := flag.String("result-file", "", "Optional path to write a pipe-separated result line")
	flag.Parse()

	if *connectionName == "" || *rdbmsName == "" || *password == "" {
		fmt.Fprintln(os.Stderr, "usage: tlsprobe -connection <name> -rdbms <name> -password <password> [flags]")
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
	logf("[%s] Starting TLS connection test (RDBMS='%s')...", start.Format("2006-01-02 15:04:05"), *rdbmsName)

	writeResult := func(reqSecTLS, tlsAvailable string, s1, s2, s3, s4 scenarioResult) {
		if *resultFile == "" {
			return
		}
		elapsed := formatElapsed(time.Since(start))
		line := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s\n", name, reqSecTLS, tlsAvailable, s1, s2, s3, s4, elapsed)
		if err := os.WriteFile(*resultFile, []byte(line), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "[%s] WARNING: failed to write result file %s: %v\n", name, *resultFile, err)
		}
	}

	abort := func(label, msg string) int {
		logf("ERROR on %s: %s", label, msg)
		writeResult("N/A", "N/A", skip, skip, skip, skip)
		return 1
	}

	// ── Get RDBMS Info (Endpoint, MasterUserName) ──────────────────────────────
	logf("GetRDBMS: fetching endpoint")
	var info rdbmsInfo
	infoURL := fmt.Sprintf("%s/spider/rdbms/%s?ConnectionName=%s", *spiderURL, url.PathEscape(*rdbmsName), url.QueryEscape(*connectionName))
	if err := httpGetJSON(infoURL, *spiderAuth, &info); err != nil {
		return abort("GetRDBMS", err.Error())
	}
	if info.Endpoint == "" {
		return abort("GetRDBMS", fmt.Sprintf("empty Endpoint in response (message: %q)", info.Message))
	}
	if info.MasterUserName == "" {
		return abort("GetRDBMS", fmt.Sprintf("empty MasterUserName in response (message: %q)", info.Message))
	}

	host, port := splitHostPort(info.Endpoint)
	logf("Endpoint: %s:%s (user: %s)", host, port, info.MasterUserName)

	// ── Get Secure Transport Status + CA Certificate ───────────────────────────
	logf("GetRDBMSSecureTransport: checking require_secure_transport and capturing CA cert")
	var st secureTransportInfo
	stURL := fmt.Sprintf("%s/spider/rdbms/%s/secure-transport?ConnectionName=%s&MasterUserPassword=%s",
		*spiderURL, url.PathEscape(*rdbmsName), url.QueryEscape(*connectionName), url.QueryEscape(*password))
	if err := httpGetJSON(stURL, *spiderAuth, &st); err != nil {
		return abort("GetRDBMSSecureTransport", err.Error())
	}
	if st.Message != "" {
		return abort("GetRDBMSSecureTransport", st.Message)
	}
	if st.RequireSecureTransport == "" {
		return abort("GetRDBMSSecureTransport", "response has no RequireSecureTransport field (this tool is MySQL/MariaDB-only)")
	}

	tlsAvailable := "OFF"
	if st.TLSInUse {
		tlsAvailable = "ON"
	}
	logf("require_secure_transport=%s, TLS available on this instance=%s", st.RequireSecureTransport, tlsAvailable)
	if !st.TLSInUse {
		logf("Server does not offer TLS at all -- S1 (plaintext) will still be tested below (that's the only mode this instance can use), but S2/S3/S4 (which require TLS) will be N/A.")
	}

	var caPool *x509.CertPool
	// verifyServerName is what we ask crypto/tls to check the presented chain's identity
	// against. It defaults to the connection host (the common case: a real DNS hostname, or
	// an IP with a matching IP SAN). But when Endpoint is a bare IP and the captured
	// certificate itself carries a DNS SAN (observed on GCP Cloud SQL, whose cert has no IP
	// SAN but does have a stable "N-<uuid>.<region>.sql.goog" DNS SAN), verifying against
	// that DNS name instead is both correct and necessary -- we still *dial* the IP (Endpoint
	// is unambiguous either way), only the post-handshake identity check uses the name the
	// certificate actually claims. This has no effect when the cert has no usable SAN at all
	// (e.g. MySQL's own auto-generated certs on some OpenStack/NHN instances) -- that's a
	// server-side limitation no client can work around; see README's Known Caveats.
	verifyServerName := host
	haveCA := st.CACertificate != nil && st.CACertificate.PEM != ""
	if haveCA {
		caPool = x509.NewCertPool()
		if !caPool.AppendCertsFromPEM([]byte(st.CACertificate.PEM)) {
			logf("WARNING: failed to parse captured CA certificate PEM — scenarios #3 and #4 will be SKIPped")
			haveCA = false
		} else {
			logf("CA certificate captured (%d bytes, IsSelfSigned=%v)", len(st.CACertificate.PEM), st.CACertificate.IsSelfSigned)
			if block, _ := pem.Decode([]byte(st.CACertificate.PEM)); block != nil {
				if parsed, err := x509.ParseCertificate(block.Bytes); err == nil && net.ParseIP(host) != nil && len(parsed.DNSNames) > 0 {
					verifyServerName = parsed.DNSNames[0]
					logf("Endpoint is a bare IP but the certificate has a DNS SAN (%s) -- verifying identity against that instead", verifyServerName)
				}
			}
		}
	} else if st.TLSInUse {
		logf("WARNING: no CA certificate available (%s) — scenarios #3 and #4 will be SKIPped", orDefault(st.CACertificateError, "not returned"))
	}

	expected := func(whenOn, whenOff scenarioResult) scenarioResult {
		if st.RequireSecureTransport == "ON" {
			return whenOn
		}
		return whenOff
	}

	attempt := func(label string, tlsCfg *tls.Config) (bool, error) {
		cfg := mysql.NewConfig()
		cfg.User = info.MasterUserName
		cfg.Passwd = *password
		cfg.Net = "tcp"
		cfg.Addr = net.JoinHostPort(host, port)
		cfg.Timeout = *timeout
		cfg.TLS = tlsCfg

		connector, err := mysql.NewConnector(cfg)
		if err != nil {
			return false, err
		}
		db := sql.OpenDB(connector)
		defer db.Close()
		db.SetMaxOpenConns(1)

		ctx, cancel := context.WithTimeout(context.Background(), *timeout)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			return false, err
		}
		return true, nil
	}

	logResult := func(scenario string, ok bool, err error) {
		status := "SUCCESS"
		if !ok {
			status = "FAIL"
		}
		logf("Scenario %s: actual=%s", scenario, status)
		if !ok && err != nil {
			logf("  error: %v", err)
		}
	}

	// ── Scenario 1: TLS unset (tls=false), no CA ───────────────────────────────
	// Always attempted, regardless of whether the server offers TLS at all: it doesn't
	// request TLS, so it's the only scenario that's meaningful on a TLS-less instance
	// (e.g. an Alibaba/Tencent instance with TLS turned off) -- it at least confirms plain
	// connectivity still works there, instead of that CSP's row being blank/untested.
	logf("Scenario 1: TLS unset (no TLS attempted), no CA")
	ok1, err1 := attempt("s1", nil)
	logResult("1", ok1, err1)
	exp1 := expected(fail, pass)
	s1 := gradeOutcome(ok1, exp1)
	logf("Scenario 1: expected=%s -> %s", humanWant(exp1), s1)

	// ── Scenario 2: tls=skip-verify (encrypted, but zero verification) ─────────
	// A short gap before each subsequent attempt: back-to-back new connections to the same
	// destination with no gap at all have been observed to trigger a dial-level timeout on
	// a later attempt on some networks (a burst-rate limiter somewhere in the path, not
	// anything CB-Spider or this tool controls) -- spacing them out avoids it.
	time.Sleep(*interAttemptDelay)
	var s2 scenarioResult
	if st.TLSInUse {
		logf("Scenario 2: tls=skip-verify (encrypted, no verification)")
		ok2, err2 := attempt("s2", &tls.Config{InsecureSkipVerify: true})
		logResult("2", ok2, err2)
		s2 = gradeOutcome(ok2, pass)
		logf("Scenario 2: expected=SUCCESS -> %s", s2)
	} else {
		logf("Scenario 2: N/A (server offers no TLS at all)")
		s2 = notApplic
	}

	// ── Scenario 3: MySQL VERIFY_CA equivalent -- chain trust only, no hostname check ──
	// go-sql-driver/mysql has no native DSN option for this (a PR proposing tls-verify=ca
	// was closed without merging -- see README's Known Caveats), so it's implemented with
	// the standard Go pattern: InsecureSkipVerify to disable the automatic checks, paired
	// with a VerifyPeerCertificate callback that does its own x509 chain verification
	// against the captured CA pool, deliberately leaving VerifyOptions.DNSName empty so
	// hostname/identity is never checked. This is meaningfully safer than S2
	// (tls=skip-verify): a certificate from an untrusted CA is still rejected, only the
	// hostname match is skipped -- exactly what MySQL's own --ssl-mode=VERIFY_CA does, and
	// the documented, recommended way to connect to a self-signed cert that has no SAN/CN
	// matching the connection host (e.g. OpenStack/NHN -- see Known Caveats).
	time.Sleep(*interAttemptDelay)
	var s3 scenarioResult = skip
	if haveCA {
		logf("Scenario 3: VERIFY_CA equivalent (chain trust only, no hostname check)")
		ok3, err3 := attempt("s3", verifyCAOnlyConfig(caPool))
		logResult("3", ok3, err3)
		s3 = gradeOutcome(ok3, pass)
		logf("Scenario 3: expected=SUCCESS -> %s", s3)
	} else if st.TLSInUse {
		logf("Scenario 3: SKIPPED (no CA certificate)")
	} else {
		logf("Scenario 3: N/A (server offers no TLS at all)")
		s3 = notApplic
	}

	// ── Scenario 4: tls=true, CA registered (strongest -- chain + hostname verification) ──
	time.Sleep(*interAttemptDelay)
	var s4 scenarioResult = skip
	if haveCA {
		logf("Scenario 4: tls=true, CA registered (chain + hostname verification)")
		ok4, err4 := attempt("s4", &tls.Config{RootCAs: caPool, ServerName: verifyServerName})
		logResult("4", ok4, err4)
		s4 = gradeOutcome(ok4, pass)
		logf("Scenario 4: expected=SUCCESS -> %s", s4)
	} else if st.TLSInUse {
		logf("Scenario 4: SKIPPED (no CA certificate)")
	} else {
		logf("Scenario 4: N/A (server offers no TLS at all)")
		s4 = notApplic
	}

	// ── Write Result ────────────────────────────────────────────────────────────
	elapsed := formatElapsed(time.Since(start))
	overall := pass
	for _, r := range []scenarioResult{s1, s2, s3, s4} {
		if r != pass && r != notApplic {
			overall = fail
			break
		}
	}
	logf("TLS connection test %s (elapsed: %s)", overall, elapsed)
	logf("  require_secure_transport=%s TLSAvailable=%s S1=%s S2=%s S3=%s S4=%s", st.RequireSecureTransport, tlsAvailable, s1, s2, s3, s4)

	writeResult(st.RequireSecureTransport, tlsAvailable, s1, s2, s3, s4)

	if overall != pass {
		return 1
	}
	return 0
}

// verifyCAOnlyConfig builds a tls.Config equivalent to MySQL's --ssl-mode=VERIFY_CA: the
// presented certificate chain must trace back to a certificate in caPool (so an untrusted CA
// is still rejected), but the hostname/identity of the leaf certificate is never checked. Go's
// crypto/tls has no built-in mode for this, so InsecureSkipVerify disables its automatic
// verification and VerifyPeerCertificate does the chain check by hand -- the standard
// documented pattern for this exact case (see crypto/tls.Config's own InsecureSkipVerify docs).
func verifyCAOnlyConfig(caPool *x509.CertPool) *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return fmt.Errorf("no certificate presented")
			}
			certs := make([]*x509.Certificate, len(rawCerts))
			for i, raw := range rawCerts {
				cert, err := x509.ParseCertificate(raw)
				if err != nil {
					return err
				}
				certs[i] = cert
			}
			intermediates := x509.NewCertPool()
			for _, cert := range certs[1:] {
				intermediates.AddCert(cert)
			}
			// DNSName deliberately left empty: chain trust only, no hostname check.
			_, err := certs[0].Verify(x509.VerifyOptions{
				Roots:         caPool,
				Intermediates: intermediates,
			})
			return err
		},
	}
}

// gradeOutcome compares an actual SUCCESS/FAIL outcome against the single expected outcome
// (used for all 4 scenarios, which always have exactly one correct answer).
func gradeOutcome(actualSuccess bool, want scenarioResult) scenarioResult {
	got := fail
	if actualSuccess {
		got = pass
	}
	if want == pass && got == pass {
		return pass
	}
	if want == fail && got == fail {
		return pass
	}
	return fail
}

func humanWant(want scenarioResult) string {
	if want == pass {
		return "SUCCESS"
	}
	return "FAIL"
}

func httpGetJSON(rawURL, basicAuth string, out interface{}) error {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	if user, pass, ok := strings.Cut(basicAuth, ":"); ok {
		req.SetBasicAuth(user, pass)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
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

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
