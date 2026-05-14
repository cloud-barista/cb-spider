// Package web provides the Echo-based HTTP server, route setup, and request handlers.
package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cloud-barista/cb-spider/spiderwatch/internal/config"
	"github.com/cloud-barista/cb-spider/spiderwatch/internal/model"
	"github.com/cloud-barista/cb-spider/spiderwatch/internal/runner"
	"github.com/cloud-barista/cb-spider/spiderwatch/internal/store"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

// Server wraps Echo and application dependencies.
type Server struct {
	e          *echo.Echo
	store      *store.Store
	scheduler  *runner.Scheduler
	runner     *runner.Runner
	renderer   *TemplateRenderer
	configPath string
}

// New creates and configures the Echo server.
func New(s *store.Store, sched *runner.Scheduler, r *runner.Runner, renderer *TemplateRenderer, cfgPath string) *Server {
	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		log.WithError(err).Errorf("handler error: %s %s", c.Request().Method, c.Request().URL.Path)
		e.DefaultHTTPErrorHandler(err, c)
	}

	// Middleware
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
	srv := &Server{e: e, store: s, scheduler: sched, runner: r, renderer: renderer, configPath: cfgPath}

	// Static files
	e.Static("/static", "web/static")

	// Web UI routes
	e.GET("/", srv.handleIndex)
	e.GET("/runs", srv.handleRunsList)
	e.GET("/runs/:id", srv.handleRunDetail)

	// REST API routes
	api := e.Group("/api/v1")
	api.GET("/summary", srv.apiSummary)
	api.GET("/runs", srv.apiRuns)
	api.GET("/runs/latest", srv.apiLatestRun)
	api.GET("/runs/:id", srv.apiRun)
	api.POST("/runs/trigger", srv.apiTrigger)
	api.POST("/runs/cleanup", srv.apiCleanupOnly)
	api.POST("/runs/stop", srv.apiStop)
	api.DELETE("/runs/:id", srv.apiDeleteRun)
	api.GET("/status", srv.apiStatus)
	api.GET("/spider/status", srv.apiSpiderStatus)
	api.POST("/spider/start", srv.apiSpiderStart)
	api.POST("/spider/stop", srv.apiSpiderStop)
	api.GET("/config/resources", srv.apiGetResources)
	api.PUT("/config/resources", srv.apiPutResources)
	api.GET("/config/cleanup", srv.apiGetCleanup)
	api.PUT("/config/cleanup", srv.apiPutCleanup)
	api.GET("/config/csps", srv.apiGetCSPs)
	api.PUT("/config/csps", srv.apiPutCSPs)
	api.GET("/runs/:id/issue-draft", srv.apiIssueDraft)
	api.POST("/runs/:id/issue", srv.apiCreateIssue)

	// Live board fragment (polled by JS during a running test)
	e.GET("/board", srv.handleBoard)

	return srv
}

// Start begins listening on addr (e.g. ":2048").
func (s *Server) Start(addr string) error {
	return s.e.Start(addr)
}

// Echo returns the underlying Echo instance (for graceful shutdown).
func (s *Server) Echo() *echo.Echo {
	return s.e
}

// ---------------------------------------------------------------------------
// Web UI handlers
// ---------------------------------------------------------------------------

func (s *Server) handleIndex(c echo.Context) error {
	latest, err := s.store.Latest()
	if err != nil {
		log.WithError(err).Error("handleIndex: failed to load latest run")
	}
	cfg := config.Get()
	cleanupOnly := latest != nil && latest.CleanupOnly
	return c.Render(http.StatusOK, "index.html", map[string]interface{}{
		"Latest":           latest,
		"CleanupOnly":      cleanupOnly,
		"NextRunTime":      s.scheduler.NextRunTime(),
		"IsRunning":        s.runner.IsRunning(),
		"IsSpiderRunning":  s.runner.IsSpiderRunning(),
		"IsExternalSpider": cfg.Spider.ExternalURL != "",
		"IsAdmin":          true,
		"IsBoard":          false,
	})
}

func (s *Server) handleRunsList(c echo.Context) error {
	runs, err := s.store.List()
	if err != nil {
		log.WithError(err).Error("handleRunsList: failed to list runs")
		return c.Render(http.StatusInternalServerError, "error.html", map[string]interface{}{
			"Message": "Failed to load run history.",
			"IsAdmin": true,
			"IsBoard": false,
		})
	}
	return c.Render(http.StatusOK, "runs.html", map[string]interface{}{
		"Runs":    runs,
		"IsAdmin": true,
		"IsBoard": false,
	})
}

func (s *Server) apiDeleteRun(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "missing run id"})
	}
	if err := s.store.Delete(id); err != nil {
		log.WithError(err).Errorf("apiDeleteRun: failed to delete run %s", id)
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) handleRunDetail(c echo.Context) error {
	id := c.Param("id")
	run, err := s.store.Get(id)
	if err != nil {
		return c.Render(http.StatusNotFound, "error.html", map[string]interface{}{
			"Message": "Run not found: " + id,
			"IsAdmin": true,
			"IsBoard": false,
		})
	}
	return c.Render(http.StatusOK, "run_detail.html", map[string]interface{}{
		"Run":     run,
		"IsAdmin": true,
		"IsBoard": false,
	})
}

// handleBoard returns a rendered HTML fragment of the CSP board for the latest
// (possibly in-progress) run.  It is polled by app.js every few seconds while
// a test is running so the user can see real-time progress.
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
	data := boardData{IsAdmin: true, IsBoard: false}
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
	summary := buildSummary(latest, s.scheduler.NextRunTime())
	return c.JSON(http.StatusOK, summary)
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

func (s *Server) apiTrigger(c echo.Context) error {
	cfg := config.Get()
	if err := s.scheduler.TriggerNow(cfg); err != nil {
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	}
	return c.JSON(http.StatusAccepted, map[string]string{
		"message": "run triggered",
		"time":    time.Now().Format(time.RFC3339),
	})
}

func (s *Server) apiCleanupOnly(c echo.Context) error {
	cfg := config.Get()
	if err := s.scheduler.TriggerCleanupOnly(cfg); err != nil {
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	}
	return c.JSON(http.StatusAccepted, map[string]string{
		"message": "cleanup-only run triggered",
		"time":    time.Now().Format(time.RFC3339),
	})
}

func (s *Server) apiStop(c echo.Context) error {
	if !s.runner.Stop() {
		return echo.NewHTTPError(http.StatusConflict, "no run is currently in progress")
	}
	return c.JSON(http.StatusAccepted, map[string]string{
		"message": "stop signal sent",
		"time":    time.Now().Format(time.RFC3339),
	})
}

func (s *Server) apiStatus(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"running": s.runner.IsRunning(),
		"time":    time.Now().Format(time.RFC3339),
	})
}

func (s *Server) apiSpiderStatus(c echo.Context) error {
	cfg := config.Get()
	return c.JSON(http.StatusOK, map[string]interface{}{
		"running":  s.runner.IsSpiderRunning(),
		"external": cfg.Spider.ExternalURL != "",
	})
}

func (s *Server) apiSpiderStart(c echo.Context) error {
	if s.runner.IsRunning() {
		return echo.NewHTTPError(http.StatusConflict, "a test run is already in progress")
	}
	if s.runner.IsSpiderRunning() {
		return echo.NewHTTPError(http.StatusConflict, "spider is already running")
	}
	cfg := config.Get()
	if err := s.runner.StartSpider(cfg); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]string{
		"message": "spider started and ready",
	})
}

func (s *Server) apiSpiderStop(c echo.Context) error {
	if s.runner.IsRunning() {
		return echo.NewHTTPError(http.StatusConflict, "cannot stop spider while a test run is in progress")
	}
	cfg := config.Get()
	if err := s.runner.StopSpider(cfg); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]string{
		"message": "spider stopped",
	})
}

func (s *Server) apiGetCleanup(c echo.Context) error {
	cfg := config.Get()
	val := strings.ToLower(strings.TrimSpace(cfg.Cleanup))
	enabled := val == "true" || val == ""
	return c.JSON(http.StatusOK, map[string]bool{"cleanup": enabled})
}

func (s *Server) apiPutCleanup(c echo.Context) error {
	var body struct {
		Cleanup bool `json:"cleanup"`
	}
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	val := "false"
	if body.Cleanup {
		val = "true"
	}
	if err := config.UpdateCleanup(s.configPath, val); err != nil {
		log.WithError(err).Error("apiPutCleanup: failed to update cleanup setting")
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "cleanup updated"})
}

func (s *Server) apiGetCSPs(c echo.Context) error {
	cfg := config.Get()
	type cspInfo struct {
		Name       string `json:"name"`
		Connection string `json:"connection"`
		Enabled    bool   `json:"enabled"`
	}
	csps := make([]cspInfo, 0, len(cfg.CSPs))
	for _, c := range cfg.CSPs {
		csps = append(csps, cspInfo{Name: c.Name, Connection: c.Connection, Enabled: c.Enabled})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"csps": csps})
}

func (s *Server) apiPutCSPs(c echo.Context) error {
	var body struct {
		CSPs []string `json:"csps"`
	}
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	// Validate: only allow names that exist in the config.
	cfg := config.Get()
	known := make(map[string]bool, len(cfg.CSPs))
	for _, csp := range cfg.CSPs {
		known[csp.Name] = true
	}
	for _, name := range body.CSPs {
		if !known[name] {
			return echo.NewHTTPError(http.StatusBadRequest, "unknown CSP name: "+name)
		}
	}
	if err := config.UpdateCSPsEnabled(s.configPath, body.CSPs); err != nil {
		log.WithError(err).Error("apiPutCSPs: failed to update CSP enabled flags")
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "csps updated"})
}

func (s *Server) apiGetResources(c echo.Context) error {
	cfg := config.Get()
	return c.JSON(http.StatusOK, map[string]interface{}{
		"all":     config.AllKnownResources,
		"enabled": cfg.Resources,
	})
}

func (s *Server) apiPutResources(c echo.Context) error {
	var body struct {
		Resources []string `json:"resources"`
	}
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	// Validate: only allow known resource kinds.
	known := make(map[string]bool, len(config.AllKnownResources))
	for _, k := range config.AllKnownResources {
		known[k] = true
	}
	for _, r := range body.Resources {
		if !known[r] {
			return echo.NewHTTPError(http.StatusBadRequest, "unknown resource kind: "+r)
		}
	}
	if err := config.UpdateResources(s.configPath, body.Resources); err != nil {
		log.WithError(err).Error("apiPutResources: failed to update resources")
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "resources updated"})
}

// ---------------------------------------------------------------------------
// GitHub issue handlers
// ---------------------------------------------------------------------------

// apiIssueDraft returns a pre-filled issue title and body for a FAIL resource.
// Query params: csp=<name>&resource=<kind>
func (s *Server) apiIssueDraft(c echo.Context) error {
	id := c.Param("id")
	cspName := c.QueryParam("csp")
	resKind := c.QueryParam("resource")
	if id == "" || cspName == "" || resKind == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing run id, csp, or resource")
	}
	run, err := s.store.Get(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "run not found")
	}
	rr, ok := findResource(run, cspName, resKind)
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound, "resource not found in run")
	}
	cfg := config.Get()

	title := fmt.Sprintf("[SpiderWatch] %s / %s FAIL (%s)", cspName, resKind, id)
	body := buildIssueBody(run, cspName, rr, cfg)

	return c.JSON(http.StatusOK, map[string]string{
		"title": title,
		"body":  body,
	})
}

// apiCreateIssue creates a GitHub issue and saves the issue number/URL back to the run.
func (s *Server) apiCreateIssue(c echo.Context) error {
	id := c.Param("id")
	var req struct {
		CSP      string `json:"csp"`
		Resource string `json:"resource"`
		Title    string `json:"title"`
		Body     string `json:"body"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.CSP == "" || req.Resource == "" || req.Title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "csp, resource, and title are required")
	}

	cfg := config.Get()
	if cfg.GitHub.Token == "" {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "GitHub token is not configured")
	}

	run, err := s.store.Get(id)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "run not found")
	}

	issueNum, issueURL, err := createGitHubIssue(cfg, req.Title, req.Body)
	if err != nil {
		log.WithError(err).Errorf("apiCreateIssue: GitHub API call failed")
		return echo.NewHTTPError(http.StatusBadGateway, "GitHub API error: "+err.Error())
	}

	// Persist the issue reference back into the stored run result.
	if err := updateResourceIssue(run, req.CSP, req.Resource, issueNum, issueURL); err != nil {
		log.WithError(err).Warn("apiCreateIssue: resource not found for issue update")
	}
	if saveErr := s.store.Save(run); saveErr != nil {
		log.WithError(saveErr).Warn("apiCreateIssue: failed to persist issue reference to run")
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"issue_number": issueNum,
		"issue_url":    issueURL,
	})
}

// ---------------------------------------------------------------------------
// GitHub helpers
// ---------------------------------------------------------------------------

func findResource(run *model.RunResult, cspName, resKind string) (model.ResourceResult, bool) {
	for _, csp := range run.CSPs {
		if csp.Name != cspName {
			continue
		}
		for _, rr := range csp.Resources {
			if rr.Kind == resKind {
				return rr, true
			}
		}
	}
	return model.ResourceResult{}, false
}

func updateResourceIssue(run *model.RunResult, cspName, resKind string, num int, url string) error {
	for ci := range run.CSPs {
		if run.CSPs[ci].Name != cspName {
			continue
		}
		for ri := range run.CSPs[ci].Resources {
			if run.CSPs[ci].Resources[ri].Kind == resKind {
				run.CSPs[ci].Resources[ri].IssueNumber = num
				run.CSPs[ci].Resources[ri].IssueURL = url
				return nil
			}
		}
	}
	return fmt.Errorf("resource %s/%s not found in run %s", cspName, resKind, run.ID)
}

func buildIssueBody(run *model.RunResult, cspName string, rr model.ResourceResult, cfg *config.Config) string {
	var b strings.Builder
	b.WriteString("## SpiderWatch Failure Report\n\n")
	b.WriteString("| Field | Value |\n|---|---|\n")
	b.WriteString(fmt.Sprintf("| Run ID | `%s` |\n", run.ID))
	b.WriteString(fmt.Sprintf("| CSP | `%s` |\n", cspName))
	b.WriteString(fmt.Sprintf("| Resource | `%s` |\n", rr.Kind))
	b.WriteString(fmt.Sprintf("| Status | `%s` |\n", rr.Status))
	if run.SpiderImage != "" {
		b.WriteString(fmt.Sprintf("| Spider Image | `%s` |\n", run.SpiderImage))
	}
	b.WriteString(fmt.Sprintf("| Tested At | `%s` |\n", rr.TestedAt.Format(time.RFC3339)))
	b.WriteString("\n")

	if len(rr.Operations) > 0 {
		b.WriteString("### Operation Results\n\n")
		for _, op := range rr.Operations {
			icon := "✅"
			if op.Status == model.ResourceStatusFail {
				icon = "❌"
			} else if op.Status == model.ResourceStatusSkipped {
				icon = "⏭"
			}
			b.WriteString(fmt.Sprintf("**%s `%s`** (%dms)\n", icon, op.Op, op.DurationMs))
			if op.Error != "" {
				b.WriteString("```\n")
				b.WriteString(op.Error)
				b.WriteString("\n```\n")
			}
			b.WriteString("\n")
		}
	} else if rr.Error != "" {
		b.WriteString("### Error\n\n```\n")
		b.WriteString(rr.Error)
		b.WriteString("\n```\n\n")
	}

	return b.String()
}

func createGitHubIssue(cfg *config.Config, title, body string) (int, string, error) {
	gh := cfg.GitHub
	payload := map[string]interface{}{
		"title":  title,
		"body":   body,
		"labels": gh.Labels,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return 0, "", fmt.Errorf("marshal payload: %w", err)
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues",
		gh.Owner, gh.Repo)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return 0, "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+gh.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Number  int    `json:"number"`
		HTMLURL string `json:"html_url"`
		Message string `json:"message"` // GitHub error message
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, "", fmt.Errorf("decode response: %w", err)
	}
	if resp.StatusCode != http.StatusCreated {
		return 0, "", fmt.Errorf("GitHub returned HTTP %d: %s", resp.StatusCode, result.Message)
	}
	return result.Number, result.HTMLURL, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func buildSummary(latest *model.RunResult, nextRun *time.Time) model.Summary {
	s := model.Summary{
		NextRunTime: nextRun,
		Status:      model.RunStatusDone,
	}
	if latest == nil {
		return s
	}
	s.LastRunID = latest.ID
	s.LastRunTime = &latest.StartedAt
	s.Status = latest.Status
	for _, csp := range latest.CSPs {
		cs := model.CSPSummary{Name: csp.Name, Total: len(csp.Resources)}
		for _, r := range csp.Resources {
			switch r.Status {
			case model.ResourceStatusOK:
				cs.OK++
			case model.ResourceStatusFail:
				cs.Failed++
			case model.ResourceStatusSkipped:
				cs.Skipped++
			}
		}
		if cs.Failed > 0 {
			cs.Status = model.ResourceStatusFail
		} else {
			cs.Status = model.ResourceStatusOK
		}
		s.CSPSummaries = append(s.CSPSummaries, cs)
	}
	return s
}
