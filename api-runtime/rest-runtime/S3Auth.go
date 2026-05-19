// S3 Authentication helper for CB-Spider.
// Implements AWS4-HMAC-SHA256 signature verification for S3-compatible XML API requests.
// by CB-Spider Team

package restruntime

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// splitAccessKey splits an S3 access key in format "username@connectionName".
// Returns (username, connectionName). Username must be present (format is required).
func splitAccessKey(accessKey string) (username, connName string) {
	if idx := strings.LastIndex(accessKey, "@"); idx >= 0 {
		return accessKey[:idx], accessKey[idx+1:]
	}
	return "", accessKey
}

// aws4AuthInfo holds parsed fields from a full AWS4-HMAC-SHA256 Authorization header.
type aws4AuthInfo struct {
	AccessKey     string
	Date          string
	Region        string
	Service       string
	SignedHeaders []string
	Signature     string
}

// isFullAWS4Auth reports whether the Authorization header is a complete AWS4 signature
// (contains both SignedHeaders and Signature fields).
// Returns false for the CB-Spider shorthand "AWS4-HMAC-SHA256 Credential=connection-name".
func isFullAWS4Auth(authHeader string) bool {
	return strings.Contains(authHeader, "Signature=") && strings.Contains(authHeader, "SignedHeaders=")
}

// parseAWS4AuthInfo parses a full AWS4-HMAC-SHA256 Authorization header.
func parseAWS4AuthInfo(authHeader string) (*aws4AuthInfo, error) {
	const prefix = "AWS4-HMAC-SHA256 "
	if !strings.HasPrefix(authHeader, prefix) {
		return nil, errors.New("not an AWS4-HMAC-SHA256 Authorization header")
	}

	info := &aws4AuthInfo{}
	for _, part := range strings.Split(authHeader[len(prefix):], ",") {
		part = strings.TrimSpace(part)
		switch {
		case strings.HasPrefix(part, "Credential="):
			cred := strings.TrimPrefix(part, "Credential=")
			segs := strings.SplitN(cred, "/", 5)
			if len(segs) < 4 {
				return nil, fmt.Errorf("invalid Credential format: %q", cred)
			}
			info.AccessKey = segs[0]
			info.Date = segs[1]
			info.Region = segs[2]
			info.Service = segs[3]
		case strings.HasPrefix(part, "SignedHeaders="):
			info.SignedHeaders = strings.Split(strings.TrimPrefix(part, "SignedHeaders="), ";")
		case strings.HasPrefix(part, "Signature="):
			info.Signature = strings.TrimPrefix(part, "Signature=")
		}
	}

	if info.AccessKey == "" || info.Date == "" || len(info.SignedHeaders) == 0 || info.Signature == "" {
		return nil, errors.New("incomplete AWS4 Authorization header")
	}
	return info, nil
}

// awsURIEncode encodes a string using AWS URI encoding rules (RFC 3986 unreserved characters).
// Unreserved: A-Z a-z 0-9 - _ . ~
func awsURIEncode(s string) string {
	var buf strings.Builder
	for i := 0; i < len(s); i++ {
		b := s[i]
		if (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') ||
			(b >= '0' && b <= '9') || b == '-' || b == '_' || b == '.' || b == '~' {
			buf.WriteByte(b)
		} else {
			fmt.Fprintf(&buf, "%%%02X", b)
		}
	}
	return buf.String()
}

// buildCanonicalURI constructs the canonical URI from the request path.
// For S3, path separators (/) are preserved; each segment is AWS URI-encoded.
func buildCanonicalURI(path string) string {
	if path == "" || path == "/" {
		return "/"
	}
	segments := strings.Split(path, "/")
	for i, seg := range segments {
		if seg != "" {
			// Decode %XX sequences first, then re-encode with AWS rules
			decoded := pctDecode(seg)
			segments[i] = awsURIEncode(decoded)
		}
	}
	return strings.Join(segments, "/")
}

// pctDecode decodes %XX percent-encoded sequences in a string (but not +).
func pctDecode(s string) string {
	var buf strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '%' && i+2 < len(s) {
			hi := hexVal(s[i+1])
			lo := hexVal(s[i+2])
			if hi >= 0 && lo >= 0 {
				buf.WriteByte(byte(hi<<4 | lo))
				i += 3
				continue
			}
		}
		buf.WriteByte(s[i])
		i++
	}
	return buf.String()
}

func hexVal(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}

// buildCanonicalQueryString constructs the sorted, AWS-encoded canonical query string.
func buildCanonicalQueryString(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}

	type kv struct{ k, v string }
	var pairs []kv

	for _, token := range strings.Split(rawQuery, "&") {
		if token == "" {
			continue
		}
		parts := strings.SplitN(token, "=", 2)
		key := pctDecode(parts[0])
		val := ""
		if len(parts) == 2 {
			val = pctDecode(parts[1])
		}
		pairs = append(pairs, kv{key, val})
	}

	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].k != pairs[j].k {
			return pairs[i].k < pairs[j].k
		}
		return pairs[i].v < pairs[j].v
	})

	encoded := make([]string, len(pairs))
	for i, p := range pairs {
		encoded[i] = awsURIEncode(p.k) + "=" + awsURIEncode(p.v)
	}
	return strings.Join(encoded, "&")
}

// buildCanonicalHeaders constructs the canonical headers string.
// Header names are already lowercase per AWS spec (as sent by S3 clients).
func buildCanonicalHeaders(req *http.Request, signedHeaders []string) string {
	var buf strings.Builder
	for _, h := range signedHeaders {
		var val string
		if h == "host" {
			val = req.Host
		} else {
			val = req.Header.Get(h)
		}
		buf.WriteString(h)
		buf.WriteByte(':')
		buf.WriteString(strings.TrimSpace(val))
		buf.WriteByte('\n')
	}
	return buf.String()
}

// getPayloadHash returns the payload hash value for the canonical request.
// Uses x-amz-content-sha256 header if present (S3 clients should always send this).
// Falls back to SHA256 of empty string.
func getPayloadHash(req *http.Request) string {
	if h := req.Header.Get("x-amz-content-sha256"); h != "" {
		return h
	}
	return s3sha256Hex(nil)
}

// s3sha256Hex returns the lowercase hex-encoded SHA256 hash of the input.
func s3sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// s3hmacSHA256 computes HMAC-SHA256.
func s3hmacSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}

// deriveSigningKey derives the AWS4 signing key from the secret key and scope.
func deriveSigningKey(secretKey, date, region, service string) []byte {
	kDate := s3hmacSHA256([]byte("AWS4"+secretKey), []byte(date))
	kRegion := s3hmacSHA256(kDate, []byte(region))
	kService := s3hmacSHA256(kRegion, []byte(service))
	return s3hmacSHA256(kService, []byte("aws4_request"))
}

// verifyAWS4Signature verifies the AWS4-HMAC-SHA256 signature of an HTTP request.
// secretKey should be SPIDER_PASSWORD; accessKey (from Credential) is the Connection Name.
func verifyAWS4Signature(req *http.Request, authHeader, secretKey string) error {
	info, err := parseAWS4AuthInfo(authHeader)
	if err != nil {
		return fmt.Errorf("failed to parse Authorization header: %w", err)
	}

	datetime := req.Header.Get("x-amz-date")
	if datetime == "" {
		return errors.New("missing x-amz-date header")
	}

	// Step 1: Build canonical request.
	// customRemoveTrailingSlash (Pre middleware) strips trailing slashes from req.URL.Path for
	// router matching, saving the original path in X-Spider-S3-Signing-Path. Use that saved
	// path so the canonical request matches what the client (e.g. S3 Browser) actually signed.
	signingPath := req.Header.Get("X-Spider-S3-Signing-Path")
	if signingPath == "" {
		signingPath = req.URL.Path
	}
	canonicalReq := strings.Join([]string{
		req.Method,
		buildCanonicalURI(signingPath),
		buildCanonicalQueryString(req.URL.RawQuery),
		buildCanonicalHeaders(req, info.SignedHeaders),
		strings.Join(info.SignedHeaders, ";"),
		getPayloadHash(req),
	}, "\n")

	cblog.Debugf("AWS4 CanonicalRequest:\n%s", canonicalReq)

	// Step 2: Build string to sign
	scope := strings.Join([]string{info.Date, info.Region, info.Service, "aws4_request"}, "/")
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		datetime,
		scope,
		s3sha256Hex([]byte(canonicalReq)),
	}, "\n")

	cblog.Debugf("AWS4 StringToSign:\n%s", stringToSign)

	// Step 3: Derive signing key and compute signature
	signingKey := deriveSigningKey(secretKey, info.Date, info.Region, info.Service)
	computed := hex.EncodeToString(s3hmacSHA256(signingKey, []byte(stringToSign)))

	cblog.Debugf("AWS4 ComputedSig=%s ProvidedSig=%s", computed, info.Signature)

	// Use constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(computed), []byte(info.Signature)) != 1 {
		cblog.Warnf("AWS4 signature mismatch (method=%s path=%s signingPath=%s):\nCanonicalRequest:\n%s\nStringToSign:\n%s\nComputed=%s\nProvided=%s",
			req.Method, req.URL.Path, signingPath, canonicalReq, stringToSign, computed, info.Signature)
		return errors.New("signature does not match")
	}
	return nil
}
