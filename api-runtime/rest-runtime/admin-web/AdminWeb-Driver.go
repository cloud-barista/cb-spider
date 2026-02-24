// Cloud Info Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// Updated: 2024.07.
// by CB-Spider Team, 2020.06.

package adminweb

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	cr "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	dim "github.com/cloud-barista/cb-spider/cloud-info-manager/driver-info-manager"

	"github.com/labstack/echo/v4"
)

// make the string of javascript function
func makeOnChangeDriverProviderFunc() string {
	strFunc := `
        function onChangeProvider(source) {
            var providerName = source.value;
            document.getElementById('2').value = providerName.toLowerCase() + "-driver-v1.0.so";
            document.getElementById('3').value = providerName.toLowerCase() + "-driver-01";
        }
    `
	return strFunc
}

// make the string of javascript function
func makeToggleCheckBoxFunc() string {
	strFunc := `
        function toggleCheckBox(source, tableId) {
            var table = document.getElementById(tableId);
            var checkboxes = table.querySelectorAll('input[name="deleteCheckbox"]');
            for (var i = 0; i < checkboxes.length; i++) {
                checkboxes[i].checked = source.checked;
            }
        }
    `
	return strFunc
}

// make the string of javascript function
func makePostDriverFunc() string {
	strFunc := `
        function postDriver() {
            var textboxes = document.getElementsByName('text_box');
            var sendJson = '{ "ProviderName" : "$$PROVIDER$$", "DriverLibFileName" : "$$DRVFILE$$", "DriverName" : "$$NAME$$" }';

            for (var i = 0; i < textboxes.length; i++) {
                switch (textboxes[i].id) {
                    case "1":
                        sendJson = sendJson.replace("$$PROVIDER$$", textboxes[i].value);
                        break;
                    case "2":
                        sendJson = sendJson.replace("$$DRVFILE$$", textboxes[i].value);
                        break;
                    case "3":
                        sendJson = sendJson.replace("$$NAME$$", textboxes[i].value);
                        break;
                    default:
                        break;
                }
            }
            var xhr = new XMLHttpRequest();
            xhr.open("POST", "$$SPIDER_SERVER$$/spider/driver", false);
            xhr.setRequestHeader('Content-Type', 'application/json');

            parent.frames["log_frame"].Log("curl -sX POST " + "$$SPIDER_SERVER$$/spider/driver -H 'Content-Type: application/json' -d '" + sendJson + "'");

            xhr.send(sendJson);

            parent.frames["log_frame"].Log("   => " + xhr.response);

            location.reload();
        }
    `
	strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://"+cr.ServiceIPorName+cr.ServicePort) // cr.ServicePort = ":1024"
	return strFunc
}

// make the string of javascript function
func makeDeleteDriverFunc() string {
	strFunc := `
        function deleteDriver() {
            var checkboxes = document.getElementsByName('deleteCheckbox');
            for (var i = 0; i < checkboxes.length; i++) {
                if (checkboxes[i].checked) {
                    var xhr = new XMLHttpRequest();
                    xhr.open("DELETE", "$$SPIDER_SERVER$$/spider/driver/" + checkboxes[i].value, false);
                    xhr.setRequestHeader('Content-Type', 'application/json');

                    parent.frames["log_frame"].Log("curl -sX DELETE " + "$$SPIDER_SERVER$$/spider/driver/" + checkboxes[i].value + " -H 'Content-Type: application/json'" );

                    xhr.send(null);

                    parent.frames["log_frame"].Log("   => " + xhr.response);
                }
            }
            location.reload();
        }
    `
	strFunc = strings.ReplaceAll(strFunc, "$$SPIDER_SERVER$$", "http://"+cr.ServiceIPorName+cr.ServicePort) // cr.ServicePort = ":1024"
	return strFunc
}

// make the string of javascript function
func makeDriverTRListHTML(bgcolor string, height string, fontSize string, infoList []*dim.CloudDriverInfo) string {
	if bgcolor == "" {
		bgcolor = "#FFFFFF"
	}
	if height == "" {
		height = "30"
	}
	if fontSize == "" {
		fontSize = "2"
	}

	// make base TR frame for info list
	strTR := fmt.Sprintf(`
        <tr bgcolor="%s" align="center" height="%s">
            <td>
                <font size=%s>$$NUM$$</font>
            </td>
            <td>
                <font size=%s>$$S1$$</font>
            </td>
            <td>
                <font size=%s>$$S2$$</font>
            </td>
            <td>
                <input type="checkbox" name="deleteCheckbox" value=$$S3$$>
            </td>
        </tr>
    `, bgcolor, height, fontSize, fontSize, fontSize)

	strData := ""
	// set data and make TR list
	for i, one := range infoList {
		str := strings.ReplaceAll(strTR, "$$NUM$$", strconv.Itoa(i+1))
		str = strings.ReplaceAll(str, "$$S1$$", one.DriverName)
		str = strings.ReplaceAll(str, "$$S2$$", one.DriverLibFileName)
		str = strings.ReplaceAll(str, "$$S3$$", one.DriverName)
		strData += str
	}

	return strData
}

func fetchDriverInfos() ([]*dim.CloudDriverInfo, error) {
	resp, err := httpGetWithAuth("http://localhost:1024/spider/driver")
	if err != nil {
		return nil, fmt.Errorf("error fetching drivers: %v", err)
	}
	defer resp.Body.Close()

	var drivers struct {
		ResultList []*dim.CloudDriverInfo `json:"driver"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&drivers); err != nil {
		return nil, fmt.Errorf("error decoding drivers: %v", err)
	}

	return drivers.ResultList, nil
}

// DriverManagement - Handles the management of drivers
func DriverManagement(c echo.Context) error {
	// Fetch driver information
	drivers, err := fetchDriverInfos()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Fetch provider information
	providers, err := fetchProviders()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	driverMap := make(map[string][]*dim.CloudDriverInfo)
	for _, driver := range drivers {
		driverMap[driver.ProviderName] = append(driverMap[driver.ProviderName], driver)
	}

	data := struct {
		Drivers     map[string][]*dim.CloudDriverInfo
		Providers   []string
		APIUsername string
		APIPassword string
	}{
		Drivers:     driverMap,
		Providers:   providers,
		APIUsername: os.Getenv("SPIDER_USERNAME"),
		APIPassword: os.Getenv("SPIDER_PASSWORD"),
	}

	// Define template path
	templatePath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/driver.html")
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error loading template: " + err.Error()})
	}

	// Execute template
	c.Response().WriteHeader(http.StatusOK)
	if err := tmpl.Execute(c.Response().Writer, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}
	return nil
}
