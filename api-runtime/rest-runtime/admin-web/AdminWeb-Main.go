// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2024.08.

package adminweb

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	cblogger "github.com/cloud-barista/cb-log"
	"github.com/labstack/echo/v4"
)

// --- Server-side session management ---

var (
	sessionMu    sync.RWMutex
	sessionStore = map[string]sessionEntry{} // token -> entry
)

type sessionEntry struct {
	Username string
	Expires  time.Time
}

const sessionCookieName = "cb_spider_session"
const sessionMaxAge = 4 * time.Hour

func generateSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func createSession(username string) (string, error) {
	token, err := generateSessionToken()
	if err != nil {
		return "", err
	}
	sessionMu.Lock()
	sessionStore[token] = sessionEntry{
		Username: username,
		Expires:  time.Now().Add(sessionMaxAge),
	}
	sessionMu.Unlock()
	return token, nil
}

func validateSession(token string) (string, bool) {
	sessionMu.RLock()
	entry, ok := sessionStore[token]
	sessionMu.RUnlock()
	if !ok || time.Now().After(entry.Expires) {
		if ok {
			sessionMu.Lock()
			delete(sessionStore, token)
			sessionMu.Unlock()
		}
		return "", false
	}
	// Extend session on each valid access (sliding expiry)
	sessionMu.Lock()
	entry.Expires = time.Now().Add(sessionMaxAge)
	sessionStore[token] = entry
	sessionMu.Unlock()
	return entry.Username, true
}

func deleteSession(token string) {
	sessionMu.Lock()
	delete(sessionStore, token)
	sessionMu.Unlock()
}

// adminweb paths that do NOT require a session
var adminWebPublicPaths = map[string]bool{
	"/spider/adminweb":            true,
	"/spider/adminweb/":           true,
	"/spider/adminweb/login":      true,
	"/spider/adminweb/logout":     true,
	"/spider/adminweb/authinfo":   true,
	"/spider/adminweb/left_menu":  true,
	"/spider/adminweb/body_frame": true,
	"/spider/adminweb1":           true,
	"/spider/adminweb1/":          true,
}

// AdminWebSessionMiddleware checks server-side session for protected adminweb pages.
func AdminWebSessionMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		reqPath := c.Request().URL.Path

		// Only apply to adminweb paths
		if !strings.HasPrefix(reqPath, "/spider/adminweb") {
			return next(c)
		}

		// Skip if auth is not enabled
		if os.Getenv("SPIDER_USERNAME") == "" || os.Getenv("SPIDER_PASSWORD") == "" {
			return next(c)
		}

		// Allow public paths
		if adminWebPublicPaths[reqPath] {
			return next(c)
		}

		// Allow static resources (images, static files, html files)
		if strings.HasPrefix(reqPath, "/spider/adminweb/images") ||
			strings.HasPrefix(reqPath, "/spider/adminweb/static") ||
			strings.HasPrefix(reqPath, "/spider/adminweb/html/") {
			return next(c)
		}

		// Check session cookie
		cookie, err := c.Cookie(sessionCookieName)
		if err != nil || cookie.Value == "" {
			return redirectToLogin(c)
		}
		if _, valid := validateSession(cookie.Value); !valid {
			return redirectToLogin(c)
		}

		return next(c)
	}
}

func redirectToLogin(c echo.Context) error {
	// If loaded in iframe, redirect the top-level window
	return c.HTML(http.StatusUnauthorized, `<!DOCTYPE html>
<html><head><title>Session Required</title></head>
<body><script>
if (window.top !== window.self) {
    window.top.location.href = '/spider/adminweb/';
} else {
    window.location.href = '/spider/adminweb/';
}
</script></body></html>`)
}

func init() {
	cblog = cblogger.GetLogger("CLOUD-BARISTA")
}

// MainPage renders the main page with iframes
func MainPage(c echo.Context) error {
	cblog.Info("call MainPage()")

	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/main.html")
	return c.File(templatePath)
}

// LeftMenu renders the left menu
func LeftMenu(c echo.Context) error {
	cblog.Info("call LeftMenu()")

	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/left_menu.html")
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return err
	}
	return tmpl.Execute(c.Response(), nil)
}

// BodyFrame renders the body frame
func BodyFrame(c echo.Context) error {
	cblog.Info("call BodyFrame()")

	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/body_frame.html")
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return err
	}
	return tmpl.Execute(c.Response(), nil)
}

// AuthInfo returns authentication status and username
func AuthInfo(c echo.Context) error {
	apiUsername := os.Getenv("SPIDER_USERNAME")
	apiPassword := os.Getenv("SPIDER_PASSWORD")
	authEnabled := apiUsername != "" && apiPassword != ""

	loggedIn := false
	sessionUser := ""
	if authEnabled {
		if cookie, err := c.Cookie(sessionCookieName); err == nil && cookie.Value != "" {
			if user, valid := validateSession(cookie.Value); valid {
				loggedIn = true
				sessionUser = user
			}
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"username":    sessionUser,
		"authEnabled": authEnabled,
		"loggedIn":    loggedIn,
	})
}

// Login validates credentials from the login form
func Login(c echo.Context) error {
	cblog.Info("call Login()")

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "Invalid request",
		})
	}

	apiUsername := os.Getenv("SPIDER_USERNAME")
	apiPassword := os.Getenv("SPIDER_PASSWORD")

	if subtle.ConstantTimeCompare([]byte(req.Username), []byte(apiUsername)) == 1 &&
		subtle.ConstantTimeCompare([]byte(req.Password), []byte(apiPassword)) == 1 {
		token, err := createSession(req.Username)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"message": "Session creation failed",
			})
		}
		c.SetCookie(&http.Cookie{
			Name:     sessionCookieName,
			Value:    token,
			Path:     "/spider/adminweb",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int(sessionMaxAge.Seconds()),
		})
		return c.JSON(http.StatusOK, map[string]interface{}{
			"success":  true,
			"username": req.Username,
		})
	}

	return c.JSON(http.StatusUnauthorized, map[string]interface{}{
		"success": false,
		"message": "Invalid username or password",
	})
}

// Logout renders a logout page and clears session
func Logout(c echo.Context) error {
	cblog.Info("call Logout()")

	// Delete server-side session
	if cookie, err := c.Cookie(sessionCookieName); err == nil && cookie.Value != "" {
		deleteSession(cookie.Value)
	}
	// Clear cookie
	c.SetCookie(&http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/spider/adminweb",
		HttpOnly: true,
		MaxAge:   -1,
	})

	return c.HTML(http.StatusOK, `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>CB-Spider - Logged Out</title>
<link rel="icon" type="image/png" href="/spider/adminweb/images/logo.png">
<style>
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; display: flex; justify-content: flex-start; padding-top: 15vh; height: 100vh; margin: 0; background: rgba(0,0,0,0.5); box-sizing: border-box; }
.box { margin: 0 auto; text-align: center; padding: 35px 45px; background: #fff; border-radius: 12px; max-width: 420px; box-shadow: 0 10px 25px rgba(0,0,0,0.15); height: fit-content; }
.box img { height: 60px; width: auto; margin-bottom: 18px; }
.tagline { color: #0645AD; font-size: 16px; font-weight: bold; margin-bottom: 40px; min-height: 1.2em; letter-spacing: 0.3px; }
.box h2 { color: #0645AD; margin-bottom: 10px; font-size: 14px; font-weight: normal; }
.box p { color: #999; margin-bottom: 40px; font-size: 12px; }
.btn { display: inline-block; padding: 7px 30px; background: #c8a84e; color: #fff; text-decoration: none; border-radius: 5px; font-weight: 500; cursor: pointer; border: none; font-size: 12px; transition: background 0.2s; }
.btn:hover { background: #b89740; }
</style>
</head>
<body>
<div class="box">
    <img src="/spider/adminweb/images/logo.png" alt="CB-Spider">
    <div class="tagline" id="tagline"></div>
    <h2>Logged Out</h2>
    <p>You have been successfully logged out.</p>
    <a class="btn" href="/spider/adminweb/">Login Again</a>
</div>
<script>
sessionStorage.removeItem('loggedIn');
sessionStorage.removeItem('loggedInUser');
sessionStorage.removeItem('skipAbout');
(function() {
    var el = document.getElementById('tagline');
    var text = '"One-Code, Multi-Cloud"';
    var i = 0;
    el.textContent = '';
    function type() {
        if (i < text.length) {
            el.textContent += text.charAt(i);
            i++;
            setTimeout(type, 50);
        }
    }
    type();
})();
</script>
</body>
</html>`)
}
