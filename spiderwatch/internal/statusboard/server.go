// Package statusboard provides the public Spider Status Board HTTP server.
// It is a read-only web service that receives run results from SpiderWatch
// and exposes them to the public without any admin controls.
package statusboard

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/cloud-barista/cb-spider/spiderwatch/internal/model"
	"github.com/cloud-barista/cb-spider/spiderwatch/internal/store"
	"github.com/cloud-barista/cb-spider/spiderwatch/internal/web"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

// Server is the Status Board HTTP server.
type Server struct {
	e        *echo.Echo
	store    *store.Store
	renderer *web.TemplateRenderer
}

// New creates and configures the Status Board Echo server.
func New(s *store.Store, renderer *web.TemplateRenderer) *Server {
	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		log.WithError(err).Errorf("handler error: %s %s", c.Request().Method, c.Request().URL.Path)
		e.DefaultHTTPErrorHandler(err, c)
	}

	e.Use(middleware.Recover())
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: `{"time":"${time_rfc3339}","method":"${method}","uri":"${uri}","status":${status},"latency":"${latency_human}"}` + "\n",
	}))
	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "SAMEORIGIN",
		HSTSMaxAge:            3600,
		ContentSecurityPolicy: "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:;",
	}))
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{Level: 5}))

	e.Renderer = renderer
	srv := &Server{e: e, store: s, renderer: renderer}

	// Static files (shared with SpiderWatch)
	e.Static("/static", "web/static")

	// Web UI routes (read-only)
	e.GET("/", srv.handleIndex)
	e.GET("/runs", srv.handleRunsList)
	e.GET("/runs/:id", srv.handleRunDetail)

	// Live board fragment (polled by JS during status refresh)
	e.GET("/board", srv.handleBoard)

	// REST API routes (read-only)
	api := e.Group("/api/v1")
	api.GET("/summary", srv.apiSummary)
	api.GET("/runs", srv.apiRuns)
	api.GET("/runs/latest", srv.apiLatestRun)
	api.GET("/runs/:id", srv.apiRun)
	api.GET("/status", srv.apiStatus)

	// Push endpoint: receives run results from SpiderWatch (authenticated)
	api.POST("/push", srv.apiPush)

	return srv
}

// SetLogger replaces the package-level logger (called from main after config load).
func SetLogger(l *logrus.Logger) { log = l }

// Start begins listening on addr (e.g. ":80").
func (s *Server) Start(addr string) error {
	return s.e.Start(addr)
}

// Echo returns the underlying Echo instance (for graceful shutdown).
func (s *Server) Echo() *echo.Echo {
	return s.e
}

// ---------------------------------------------------------------------------
// Template context helpers
// ---------------------------------------------------------------------------

// baseCtx returns the common read-only template context.
func baseCtx(extra map[string]interface{}) map[string]interface{} {
	ctx := map[string]interface{}{
		"IsAdmin": false,
		"IsBoard": true,
	}
	for k, v := range extra {
		ctx[k] = v
	}
	return ctx
}

// ---------------------------------------------------------------------------
// Web UI handlers
// ---------------------------------------------------------------------------

func (s *Server) handleIndex(c echo.Context) error {
	latest, err := s.store.Latest()
	if err != nil {
		log.WithError(err).Error("handleIndex: failed to load latest run")
	}
	cleanupOnly := latest != nil && latest.CleanupOnly
	return c.Render(http.StatusOK, "index.html", baseCtx(map[string]interface{}{
		"Latest":      latest,
		"CleanupOnly": cleanupOnly,
		"NextRunTime": nil,
		"IsRunning":   false,
	}))
}

func (s *Server) handleRunsList(c echo.Context) error {
	runs, err := s.store.List()
	if err != nil {
		log.WithError(err).Error("handleRunsList: failed to list runs")
		return c.Render(http.StatusInternalServerError, "error.html", baseCtx(map[string]interface{}{
			"Message": "Failed to load run history.",
		}))
	}
	return c.Render(http.StatusOK, "runs.html", baseCtx(map[string]interface{}{
		"Runs": runs,
	}))
}

func (s *Server) handleRunDetail(c echo.Context) error {
	id := c.Param("id")
	run, err := s.store.Get(id)
	if err != nil {
		return c.Render(http.StatusNotFound, "error.html", baseCtx(map[string]interface{}{
			"Message": "Run not found: " + id,
		}))
	}
	return c.Render(http.StatusOK, "run_detail.html", baseCtx(map[string]interface{}{
		"Run": run,
	}))
}

// handleBoard returns a rendered HTML fragment of the CSP board for the latest run.
func (s *Server) handleBoard(c echo.Context) error {
	latest, err := s.store.Latest()
	if err != nil {
		log.WithError(err).Warn("handleBoard: failed to load latest run")
	}
	type boardData struct {
		RunID       string
		CSPs        []model.CSPResult
		Progresses  map[string]*model.RunProgress
		CleanupOnly bool
		IsAdmin     bool
		IsBoard     bool
	}
	data := boardData{IsAdmin: false, IsBoard: true}
	if latest != nil {
		data.RunID = latest.ID
		data.CSPs = latest.CSPs
		data.Progresses = latest.Progresses
		data.CleanupOnly = latest.CleanupOnly
	}
	var buf bytes.Buffer
	if err := s.renderer.RenderDef(&buf, "board_fragment.html", "board-fragment", data); err != nil {
		log.WithError(err).Error("handleBoard: failed to render board fragment")
		return echo.NewHTTPError(http.StatusInternalServerError, "render failed")
	}
	return c.HTML(http.StatusOK, buf.String())
}

// ---------------------------------------------------------------------------
// REST API handlers
// ---------------------------------------------------------------------------

func (s *Server) apiSummary(c echo.Context) error {
	latest, err := s.store.Latest()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	type summary struct {
		LastRunID   string       `json:"last_run_id"`
		LastRunTime *interface{} `json:"last_run_time,omitempty"`
		Status      string       `json:"status"`
	}
	if latest == nil {
		return c.JSON(http.StatusOK, map[string]interface{}{"status": "no runs"})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"last_run_id":   latest.ID,
		"last_run_time": latest.StartedAt,
		"status":        latest.Status,
		"spider_image":  latest.SpiderImage,
	})
}

func (s *Server) apiRuns(c echo.Context) error {
	runs, err := s.store.List()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, runs)
}

func (s *Server) apiLatestRun(c echo.Context) error {
	run, err := s.store.Latest()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if run == nil {
		return c.JSON(http.StatusOK, nil)
	}
	return c.JSON(http.StatusOK, run)
}

func (s *Server) apiRun(c echo.Context) error {
	id := c.Param("id")
	run, err := s.store.Get(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "run not found: "+id)
	}
	return c.JSON(http.StatusOK, run)
}

func (s *Server) apiStatus(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"service": "spider-status-board",
		"ok":      true,
	})
}

// apiPush receives a completed RunResult from SpiderWatch and stores it.
// The request must include "Authorization: Bearer <token>" matching auth.token.
func (s *Server) apiPush(c echo.Context) error {
	cfg := Get()

	// Authenticate
	authHeader := c.Request().Header.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if cfg.Auth.Token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(cfg.Auth.Token)) != 1 {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid or missing token")
	}

	var result model.RunResult
	if err := json.NewDecoder(c.Request().Body).Decode(&result); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid JSON body: "+err.Error())
	}
	if result.ID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "run id is required")
	}

	if err := s.store.Save(&result); err != nil {
		log.WithError(err).Errorf("apiPush: failed to save run %s", result.ID)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to save run result")
	}

	log.Infof("apiPush: received run %s (status=%s)", result.ID, result.Status)
	return c.JSON(http.StatusCreated, map[string]string{
		"message": "run result stored",
		"id":      result.ID,
	})
}
