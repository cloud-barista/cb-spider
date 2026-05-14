// Package model defines the data types used across SpiderWatch.
package model

import "time"

// RunStatus represents the overall status of a test run.
type RunStatus string

const (
	RunStatusRunning RunStatus = "RUNNING"
	RunStatusDone    RunStatus = "DONE"
	RunStatusFailed  RunStatus = "FAILED"
	RunStatusStopped RunStatus = "STOPPED"
)

// ResourceStatus represents the result status for a single resource kind.
type ResourceStatus string

const (
	ResourceStatusOK      ResourceStatus = "OK"
	ResourceStatusFail    ResourceStatus = "FAIL"
	ResourceStatusSkipped ResourceStatus = "SKIPPED"
)

// RunProgress tracks the currently executing resource within a running test.
type RunProgress struct {
	CSP          string            `json:"csp"`
	Resource     string            `json:"resource"`                // empty when between resources
	Operation    string            `json:"operation"`               // current op within a resource, e.g. "create", "wait-active"
	CompletedOps []OperationResult `json:"completed_ops,omitempty"` // ops finished before current one
}

// RunResult is the top-level record for a single SpiderWatch test run.
type RunResult struct {
	ID          string     `json:"id"`
	StartedAt   time.Time  `json:"started_at"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	Status      RunStatus  `json:"status"`
	Error       string     `json:"error,omitempty"`        // top-level run error (e.g. Docker unavailable)
	SpiderImage string     `json:"spider_image,omitempty"` // e.g. "cloudbaristaorg/cb-spider:edge@sha256:abc123... (2026-05-06T10:11:12Z)"
	// CleanupOnly is true when the run was triggered via the Cleanup Only button
	// (cfg.Cleanup == "only"). Used by templates to suppress the stats bar.
	CleanupOnly bool `json:"cleanup_only,omitempty"`
	// Progresses holds live per-CSP progress while a run is active.
	// Keyed by CSP name; entry is deleted when that CSP finishes.
	Progresses map[string]*RunProgress `json:"progresses,omitempty"`
	CSPs       []CSPResult             `json:"csps"`
}

// CSPResult holds test results for one CSP within a run.
type CSPResult struct {
	Name          string           `json:"name"`
	Connection    string           `json:"connection"`
	ExpectedTotal int              `json:"expected_total"`
	Resources     []ResourceResult `json:"resources"`
}

// OperationResult holds the outcome of a single CRUD operation (create/list/get/delete).
type OperationResult struct {
	Op         string         `json:"op"`
	Status     ResourceStatus `json:"status"`
	Error      string         `json:"error,omitempty"`
	Message    string         `json:"message,omitempty"`
	DurationMs int64          `json:"duration_ms"`
}

// ResourceResult holds the outcome for a single resource kind test.
type ResourceResult struct {
	Kind        string            `json:"kind"`
	Status      ResourceStatus    `json:"status"`
	Count       int               `json:"count"`
	Error       string            `json:"error,omitempty"`
	DurationMs  int64             `json:"duration_ms"`
	TestedAt    time.Time         `json:"tested_at"`
	Operations  []OperationResult `json:"operations,omitempty"`
	IssueNumber int               `json:"issue_number,omitempty"` // GitHub issue number after filing
	IssueURL    string            `json:"issue_url,omitempty"`    // GitHub issue HTML URL
}

// Summary is the aggregated view shown on the dashboard homepage.
type Summary struct {
	LastRunID    string       `json:"last_run_id"`
	LastRunTime  *time.Time   `json:"last_run_time,omitempty"`
	NextRunTime  *time.Time   `json:"next_run_time,omitempty"`
	Status       RunStatus    `json:"status"`
	CSPSummaries []CSPSummary `json:"csp_summaries"`
}

// CSPSummary is a condensed per-CSP status.
type CSPSummary struct {
	Name    string         `json:"name"`
	Total   int            `json:"total"`
	OK      int            `json:"ok"`
	Failed  int            `json:"failed"`
	Skipped int            `json:"skipped"`
	Status  ResourceStatus `json:"status"`
}
