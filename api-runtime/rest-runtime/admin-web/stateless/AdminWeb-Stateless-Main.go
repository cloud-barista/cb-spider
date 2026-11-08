// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// AdminWeb pages specific to Stateless API mode. The shared chrome (main.html, login/
// session, static assets) comes from the sibling admin-web package and is reused as-is;
// only the left menu and the body_frame landing page differ, because the managed-mode
// versions hard-code a "select a ConnectionName first" flow that does not apply here.
// Every future Stateless-mode resource page (SecurityGroup, VM, ...) belongs in this
// package, alongside its own html/stateless/<resource>.html template.
//
// by CB-Spider Team, 2026.11.

package statelessweb

import (
	"html/template"
	"os"
	"path/filepath"

	cblogger "github.com/cloud-barista/cb-log"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

var cblog *logrus.Logger

func init() {
	cblog = cblogger.GetLogger("CLOUD-BARISTA")
}

// LeftMenu renders the Stateless Mode left menu. Unlike the managed-mode menu, entries
// here never require a pre-selected ConnectionName: each Stateless resource page gathers
// Provider/Credential/Region on the page itself.
func LeftMenu(c echo.Context) error {
	cblog.Info("call statelessweb.LeftMenu()")
	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/stateless/left_menu.html")
	return c.File(templatePath)
}

// BodyFrame renders the Stateless Mode landing page shown in the body_frame iframe.
func BodyFrame(c echo.Context) error {
	cblog.Info("call statelessweb.BodyFrame()")
	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/stateless/body_frame.html")
	return c.File(templatePath)
}

// comingSoonData is the template data for a not-yet-implemented Stateless resource page.
type comingSoonData struct {
	ResourceName string // e.g. "VPC/Subnet"
}

// ComingSoon returns a handler for a Stateless resource that doesn't have a dedicated
// AdminWeb page yet, pointing the user at the Swagger UI instead. Once a resource gets
// its own page (e.g. "vpc"), replace its route in CBSpiderRuntime.go with the real handler.
func ComingSoon(resourceName string) echo.HandlerFunc {
	return func(c echo.Context) error {
		cblog.Info("call statelessweb.ComingSoon(" + resourceName + ")")
		templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/stateless/coming-soon.html")
		tmpl, err := template.ParseFiles(templatePath)
		if err != nil {
			return err
		}
		return tmpl.Execute(c.Response(), comingSoonData{ResourceName: resourceName})
	}
}
