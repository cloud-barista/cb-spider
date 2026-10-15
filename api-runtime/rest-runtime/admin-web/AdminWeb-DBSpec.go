package adminweb

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/labstack/echo/v4"
)

//====================================== DBSpec

// driverCapabilityRDBMSEngines is the subset of DriverCapabilityInfo needed to gray out
// DB Engine options this connection's CSP driver doesn't support, and to tell whether
// DBSpecHandler (DBSpec itself) is implemented at all (e.g. KT is not).
type driverCapabilityRDBMSEngines struct {
	DBSpecHandler          bool `json:"DBSpecHandler"`
	RDBMSMySQLHandler      bool `json:"RDBMSMySQLHandler"`
	RDBMSMariaDBHandler    bool `json:"RDBMSMariaDBHandler"`
	RDBMSPostgreSQLHandler bool `json:"RDBMSPostgreSQLHandler"`
}

// fetchDBSpecCapability looks up connConfig's DBSpecHandler/per-engine support in one
// call: dbSpecSupported gates whether DBSpec is available at all for this CSP (e.g.
// KT), and engineSupport grays out unsupported DB Engine options in the dropdown (mirrors the
// same GetDriverCapability-based graying already done for the Add New RDBMS modal). Fails open
// (everything reported as supported) if the capability lookup itself fails, so a hiccup there
// doesn't block the whole DBSpec page from rendering.
func fetchDBSpecCapability(connConfig string) (dbSpecSupported bool, engineSupport map[string]bool) {
	engineSupport = map[string]bool{"mysql": true, "mariadb": true, "postgresql": true}
	resBody, err := getResourceList_JsonByte("driver/capability?ConnectionName=" + url.QueryEscape(connConfig))
	if err != nil {
		cblog.Warnf("DBSpec: failed to load driver capability for %s: %v", connConfig, err)
		return true, engineSupport
	}
	var capability driverCapabilityRDBMSEngines
	if err := json.Unmarshal(resBody, &capability); err != nil {
		cblog.Warnf("DBSpec: failed to parse driver capability for %s: %v", connConfig, err)
		return true, engineSupport
	}
	engineSupport["mysql"] = capability.RDBMSMySQLHandler
	engineSupport["mariadb"] = capability.RDBMSMariaDBHandler
	engineSupport["postgresql"] = capability.RDBMSPostgreSQLHandler
	return capability.DBSpecHandler, engineSupport
}

func DBSpec(c echo.Context) error {
	cblog.Info("call DBSpec()")

	connConfig := c.Param("ConnectConfig")
	if connConfig == "region not set" {
		htmlStr := `
            <html>
            <head>
                <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
				<style>
				th {
				  border: 1px solid lightgray;
				}
				td {
				  border: 1px solid lightgray;
				  border-radius: 4px;
				}
				</style>
            </head>
            <body>
                <br>
                <br>
                <label style="font-size:24px;color:#606262;">&nbsp;&nbsp;&nbsp;Please select a Connection Configuration! (MENU: 2.CONNECTION)</label>
            </body>
        `

		return c.HTML(http.StatusOK, htmlStr)
	}

	dbEngine := c.QueryParam("DBEngine")
	if dbEngine == "" {
		dbEngine = "mysql"
	}

	dbSpecSupported, engineSupport := fetchDBSpecCapability(connConfig)
	if !dbSpecSupported {
		providerName, _ := getProviderName(connConfig)
		data := struct {
			ConnConfig    string
			DBEngine      string
			ErrMessage    string
			DBSpecs       []*cres.DBSpecInfo
			EngineSupport map[string]bool
		}{
			ConnConfig:    connConfig,
			DBEngine:      dbEngine,
			ErrMessage:    "This CSP (" + providerName + ") does not support DBSpec.",
			EngineSupport: engineSupport,
		}
		return renderDBSpecTemplate(c, data)
	}

	resourceName := "dbspec?ConnectionName=" + url.QueryEscape(connConfig) + "&DBEngine=" + url.QueryEscape(dbEngine)
	resBody, err := getResourceList_JsonByte(resourceName)
	if err != nil {
		cblog.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var info struct {
		ResultList []*cres.DBSpecInfo `json:"dbspec"`
	}
	if err := json.Unmarshal(resBody, &info); err != nil {
		// e.g. the CSP driver does not support DBSpecHandler, or the engine is
		// not supported: the /dbspec API returns an error body instead of {"dbspec": [...]}.
		var errBody struct {
			Message string `json:"message"`
		}
		json.Unmarshal(resBody, &errBody)
		if errBody.Message == "" {
			errBody.Message = string(resBody)
		}
		data := struct {
			ConnConfig    string
			DBEngine      string
			ErrMessage    string
			DBSpecs       []*cres.DBSpecInfo
			EngineSupport map[string]bool
		}{
			ConnConfig:    connConfig,
			DBEngine:      dbEngine,
			ErrMessage:    errBody.Message,
			EngineSupport: engineSupport,
		}
		return renderDBSpecTemplate(c, data)
	}

	data := struct {
		ConnConfig    string
		DBEngine      string
		ErrMessage    string
		DBSpecs       []*cres.DBSpecInfo
		EngineSupport map[string]bool
	}{
		ConnConfig:    connConfig,
		DBEngine:      dbEngine,
		DBSpecs:       info.ResultList,
		EngineSupport: engineSupport,
	}
	return renderDBSpecTemplate(c, data)
}

func renderDBSpecTemplate(c echo.Context, data interface{}) error {
	tmplPath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/dbspec.html")
	tmpl, err := template.New("dbspec.html").Funcs(template.FuncMap{
		"inc": func(i int) int {
			return i + 1
		},
	}).ParseFiles(tmplPath)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, data); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.HTML(http.StatusOK, result.String())
}
