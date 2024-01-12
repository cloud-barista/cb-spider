// Cloud Info Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2023.09.

package adminweb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"

	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	"net/http"

	"github.com/labstack/echo/v4"
)

//====================================== RegionZone

func RegionZone(c echo.Context) error {
	cblog.Info("call RegionZone()")

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
                <script type="text/javascript">
                alert(connConfig)
                </script>
            </head>
            <body>
                <br>
                <br>
                <label style="font-size:24px;color:#606262;">&nbsp;&nbsp;&nbsp;Please select a Connection Configuration! (MENU: 2.CONNECTION)</label>   
            </body>
        `

		return c.HTML(http.StatusOK, htmlStr)
	}

	resBody, err := getResourceList_with_Connection_JsonByte(connConfig, "regionzone")
	if err != nil {
		cblog.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var info struct {
		ResultList []*cres.RegionZoneInfo `json:"regionzone"`
	}
	json.Unmarshal(resBody, &info)

	// struct for HTML template
	type ZoneInfo struct {
		ZoneName    string
		DisplayName string
		ZoneStatus  string
		IsDefault   bool
	}

	type RegionInfo struct {
		RegionName   string
		DisplayName  string
		InnerTableID string
		ZoneInfo     []ZoneInfo
	}

	type PageData struct {
		LoggingUrl    template.JS
		RegionInfo    []*RegionInfo
		LoggingResult template.JS
	}

	var regionInfos []*RegionInfo
	regionZoneInfos := info.ResultList
	for idx, rzInfo := range regionZoneInfos {
		rInfo := &RegionInfo{
			RegionName:   rzInfo.Name,
			DisplayName:  rzInfo.DisplayName,
			InnerTableID: fmt.Sprintf("%s-%d", rzInfo.Name, idx),
		}

		for i, zone := range rzInfo.ZoneList {
			isDefault := i == 0 // Only the first row is true, for the default zone
			rInfo.ZoneInfo = append(rInfo.ZoneInfo, ZoneInfo{
				ZoneName:    zone.Name,
				DisplayName: zone.DisplayName,
				ZoneStatus:  string(zone.Status),
				IsDefault:   isDefault,
			})
		}
		regionInfos = append(regionInfos, rInfo)
	}

	data := PageData{
		LoggingUrl:    template.JS(genLoggingGETURL2(connConfig, "regionzone")),
		RegionInfo:    regionInfos,
		LoggingResult: template.JS(genLoggingResult2(string(resBody[:len(resBody)-1]))),
	}

	// Parse the HTML template
	tmpl, err := template.New("index").Parse(htmlTemplate)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Execute the template with data
	var result bytes.Buffer
	err = tmpl.Execute(&result, data)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.HTML(http.StatusOK, result.String())
}

const htmlTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>Multi-Cloud Region/Zone Info</title>
<style>
    table {
        width: 100%;
        border-collapse: collapse;
    }
    th, td {
        border: 1px solid black;
        padding: 8px;
        text-align: center;
    }
    th {
        background-color: #f2f2f2;
    }
    .inner-th {
        background-color: #d9edf7;
    }
</style>
<script>
    function showAlert(regionName, innerTableId) {
        var table = document.getElementById(innerTableId);
        var rows = table.rows;

        for (var i = 1; i < rows.length; i++) {
            var cells = rows[i].cells;
            var input = cells[2].getElementsByTagName('input')[0];

            if (input.checked) {
                var zoneName = cells[0].innerText;
                alert('Region Name: ' + regionName + '\nZone Name: ' + zoneName);
                break;
            }
        }
    }

    function searchTable() {
        var input, filter, table, tr, td;
        input = document.getElementById("searchInput");
        filter = input.value.toUpperCase();
        table = document.getElementsByTagName("table")[0];
        tr = table.getElementsByTagName("tr");

        for (var i = 1; i < tr.length; i++) {
            td = tr[i].getElementsByTagName("td")[0];
            if (td) {
                var txtValue = td.textContent || td.innerText;
                if (txtValue.toUpperCase().indexOf(filter) > -1) {
                    tr[i].style.display = "";
                } else {
                    tr[i].style.display = "none";
                }
            }
        }
    }

    function filterStatus() {
        var statusFilter = document.getElementById("statusFilter").value;
        var tables = document.querySelectorAll("table table");
        tables.forEach(function(table) {
            var rows = table.rows;
            for (var i = 1; i < rows.length; i++) {
                var cells = rows[i].cells;
                var status = cells[2].innerText;
                if (statusFilter === "All" || status === statusFilter) {
                    rows[i].style.display = "";
                } else {
                    rows[i].style.display = "none";
                }
            }
        });
    }
</script>
<script>
    {{.LoggingUrl}}
    {{.LoggingResult}}
</script>
</head>
<body>
<input type="text" id="searchInput" onkeyup="searchTable()" placeholder="Search for Region Names..">
<select id="statusFilter" onchange="filterStatus()">
    <option value="All">All</option>
    <option value="Available">Available</option>
    <option value="Unavailable">Unavailable</option>
    <option value="StatusNotSupported">StatusNotSupported</option>
</select>
<table>
    <tr>
        <th>Region Name</th>
        <th>Display Name</th>
        <th>Zone List</th>
    </tr>
    {{range $region := .RegionInfo}}
    <tr>
        <td>{{$region.RegionName}}</td>
        <td>{{$region.DisplayName}}</td>
        <td>
            <table id="{{.InnerTableID}}">
                <tr>
                    <th class="inner-th">Zone Name</th>
                    <th class="inner-th">Display Name</th>
                    <th class="inner-th">Zone Status</th>
                </tr>
                {{range .ZoneInfo}}
                <tr>
                    <td>{{.ZoneName}}</td>
                    <td>{{.DisplayName}}</td>
                    <td>{{.ZoneStatus}}</td>
                </tr>
                {{end}}
            </table>
        </td>
    </tr>
    {{end}}
</table>
</body>
</html>
`
