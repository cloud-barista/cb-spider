// Package web - template renderer for Echo.
package web

import (
	"fmt"
	"html/template"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloud-barista/cb-spider/spiderwatch/internal/model"
	"github.com/labstack/echo/v4"
)

// TemplateRenderer implements echo.Renderer using html/template.
// Each page template is compiled into its own template set (base.html + page)
// so that {{define "content"}} blocks do not conflict across pages.
type TemplateRenderer struct {
	templates map[string]*template.Template
}

// NewRenderer loads HTML templates from the given glob pattern.
// It expects a base.html inside the same directory; every other .html file
// is compiled together with base.html into an isolated template set.
func NewRenderer(pattern string) (*TemplateRenderer, error) {
	funcMap := template.FuncMap{
		"fmtTime": func(t time.Time) string {
			return t.Format("01-02 15:04:05")
		},
		"fmtTimePtr": func(t *time.Time) string {
			if t == nil {
				return "-"
			}
			return t.Format("2006-01-02 15:04:05 KST")
		},
		"statusClass": func(status interface{}) string {
			switch fmt.Sprint(status) {
			case "OK":
				return "status-ok"
			case "FAIL", "FAILED":
				return "status-fail"
			case "SKIPPED":
				return "status-skipped"
			case "RUNNING":
				return "status-running"
			case "DONE":
				return "status-ok"
			case "STOPPED":
				return "status-stopped"
			default:
				return "status-unknown"
			}
		},
		"statusIcon": func(status interface{}) string {
			switch fmt.Sprint(status) {
			case "OK", "DONE":
				return "✓"
			case "FAIL", "FAILED":
				return "✗"
			case "SKIPPED":
				return "–"
			case "RUNNING":
				return "⟳"
			case "STOPPED":
				return "⏹"
			default:
				return "?"
			}
		},
		"lower": strings.ToLower,
		"base":  filepath.Base,
		"durStr": func(ms int64) string {
			if ms < 1000 {
				return fmt.Sprintf("%dms", ms)
			}
			if ms < 60_000 {
				return fmt.Sprintf("%.1fs", float64(ms)/1000)
			}
			mins := ms / 60_000
			secs := (ms % 60_000) / 1000
			if secs == 0 {
				return fmt.Sprintf("%dm", mins)
			}
			return fmt.Sprintf("%dm %ds", mins, secs)
		},
		"add": func(a, b int) int { return a + b },
		"sumDuration": func(resources []model.ResourceResult) int64 {
			var total int64
			for _, r := range resources {
				total += r.DurationMs
			}
			return total
		},
		"pct": func(ok, total int) string {
			if total == 0 {
				return "0"
			}
			return fmt.Sprintf("%.0f", float64(ok)/float64(total)*100)
		},
		"reverse": func(runs []*model.RunResult) []*model.RunResult {
			n := len(runs)
			rev := make([]*model.RunResult, n)
			for i, r := range runs {
				rev[n-1-i] = r
			}
			return rev
		},
		"runFailCount": func(run *model.RunResult) int {
			count := 0
			for _, csp := range run.CSPs {
				for _, res := range csp.Resources {
					if res.Status == model.ResourceStatusFail {
						count++
					}
				}
			}
			return count
		},
	}

	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("renderer: glob %q: %w", pattern, err)
	}

	// Locate base.html and page files
	var baseFile string
	var pageFiles []string
	for _, f := range files {
		if filepath.Base(f) == "base.html" {
			baseFile = f
		} else {
			pageFiles = append(pageFiles, f)
		}
	}
	if baseFile == "" {
		return nil, fmt.Errorf("renderer: base.html not found in %q", pattern)
	}

	// Build an isolated template set per page (base.html + page file).
	// This prevents {{define "content"}} blocks from different pages clobbering
	// each other when they all live in the same template.Template set.
	templates := make(map[string]*template.Template, len(pageFiles))
	for _, page := range pageFiles {
		name := filepath.Base(page)
		tmpl, err := template.New(name).Funcs(funcMap).ParseFiles(baseFile, page)
		if err != nil {
			return nil, fmt.Errorf("renderer: parse %q: %w", name, err)
		}
		templates[name] = tmpl
	}

	return &TemplateRenderer{templates: templates}, nil
}

// Render satisfies echo.Renderer.
// It executes the "base.html" entry point from the page-specific template set
// so that the page's {{define "content"}} properly overrides base.html's {{block}}.
func (r *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	tmpl, ok := r.templates[name]
	if !ok {
		return fmt.Errorf("renderer: template %q not found", name)
	}
	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		return fmt.Errorf("renderer: execute template %q: %w", name, err)
	}
	return nil
}

// RenderDef renders a named {{define}} block from the given template file
// without wrapping it in base.html.  Used for HTML fragment endpoints.
func (r *TemplateRenderer) RenderDef(w io.Writer, tmplKey, defName string, data interface{}) error {
	tmpl, ok := r.templates[tmplKey]
	if !ok {
		return fmt.Errorf("renderer: template %q not found", tmplKey)
	}
	if err := tmpl.ExecuteTemplate(w, defName, data); err != nil {
		return fmt.Errorf("renderer: execute define %q in %q: %w", defName, tmplKey, err)
	}
	return nil
}
