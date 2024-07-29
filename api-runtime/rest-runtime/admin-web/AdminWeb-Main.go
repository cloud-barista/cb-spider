package adminweb

import (
	"html/template"
	"os"
	"path/filepath"

	cblogger "github.com/cloud-barista/cb-log"
	"github.com/labstack/echo/v4"
)

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
