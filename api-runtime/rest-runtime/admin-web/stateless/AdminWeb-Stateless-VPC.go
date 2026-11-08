// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// Stateless Mode AdminWeb page for VPC/Subnet. Unlike VPC-SubnetRest.go's managed-mode
// counterpart, this page never references a ConnectionName: Provider/CredentialInfo/
// RegionInfo are entered on the page itself and kept only in the browser's sessionStorage,
// then sent inline on every call to /spider/stateless/vpc.
//
// by CB-Spider Team, 2026.11.

package statelessweb

import (
	"html/template"
	"os"
	"path/filepath"

	"github.com/labstack/echo/v4"
)

// vpcPageData carries the Basic Auth credentials the page's JS needs to call the
// Spider REST API directly (same pattern as the managed-mode resource pages).
type vpcPageData struct {
	APIUsername string
	APIPassword string
}

// VPCManagement renders the Stateless Mode VPC/Subnet management page.
func VPCManagement(c echo.Context) error {
	cblog.Info("call statelessweb.VPCManagement()")
	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/stateless/vpc.html")
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return err
	}
	data := vpcPageData{
		APIUsername: os.Getenv("SPIDER_USERNAME"),
		APIPassword: os.Getenv("SPIDER_PASSWORD"),
	}
	return tmpl.Execute(c.Response(), data)
}
