// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This covers only the Public Authoritative DNS baseline (Zone + Record CRUD).
//
// by CB-Spider Team, 2026.09.

package restruntime

import (
	"net/http"
	"strconv"

	cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	"github.com/labstack/echo/v4"
)

//================ DNS Zone Handler

// DNSZoneRegisterRequest represents the request body for registering a DNS Zone.
type DNSZoneRegisterRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		Name  string `json:"Name" validate:"required" example:"my-zone"`
		CSPId string `json:"CSPId" validate:"required" example:"/hostedzone/Z1234567890ABC"`
	} `json:"ReqInfo" validate:"required"`
}

// registerDNSZone godoc
// @ID register-dnszone
// @Summary Register DNS Zone
// @Description Register an existing Public DNS Zone with the specified name and CSP ID.
// @Tags [DNS Management]
// @Accept  json
// @Produce  json
// @Param DNSZoneRegisterRequest body restruntime.DNSZoneRegisterRequest true "Request body for registering a DNS Zone"
// @Success 200 {object} cres.DNSZoneInfo "Details of the registered DNS Zone"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /regdnszone [post]
func RegisterDNSZone(c echo.Context) error {
	cblog.Info("call RegisterDNSZone()")

	req := DNSZoneRegisterRequest{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	userIId := cres.IID{NameId: req.ReqInfo.Name, SystemId: req.ReqInfo.CSPId}

	result, err := cmrt.RegisterDNSZone(req.ConnectionName, userIId)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// unregisterDNSZone godoc
// @ID unregister-dnszone
// @Summary Unregister DNS Zone
// @Description Unregister a DNS Zone with the specified name, without deleting it from the CSP.
// @Tags [DNS Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for unregistering a DNS Zone"
// @Param Name path string true "The name of the DNS Zone to unregister"
// @Success 200 {object} BooleanInfo "Result of the unregister operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /regdnszone/{Name} [delete]
func UnregisterDNSZone(c echo.Context) error {
	cblog.Info("call UnregisterDNSZone()")

	var req ConnectionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	result, err := cmrt.UnregisterResource(req.ConnectionName, DNS, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &BooleanInfo{Result: strconv.FormatBool(result)})
}

// DNSZoneCreateRequest represents the request body for creating a DNS Zone.
type DNSZoneCreateRequest struct {
	ConnectionName  string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	IDTransformMode string `json:"IDTransformMode,omitempty" validate:"omitempty" example:"ON"` // ON: transform CSP ID, OFF: no-transform CSP ID
	ReqInfo         struct {
		Name       string          `json:"Name" validate:"required" example:"my-zone"`
		DomainName string          `json:"DomainName" validate:"required" example:"example.com"`
		TagList    []cres.KeyValue `json:"TagList,omitempty" validate:"omitempty"`
	} `json:"ReqInfo" validate:"required"`
}

// createDNSZone godoc
// @ID create-dnszone
// @Summary Create DNS Zone
// @Description Create a new Public Authoritative DNS Zone(Hosted Zone) for the given domain name.
// @Tags [DNS Management]
// @Accept  json
// @Produce  json
// @Param DNSZoneCreateRequest body restruntime.DNSZoneCreateRequest true "Request body for creating a DNS Zone"
// @Success 200 {object} cres.DNSZoneInfo "Details of the created DNS Zone"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /dnszone [post]
func CreateDNSZone(c echo.Context) error {
	cblog.Info("call CreateDNSZone()")

	req := DNSZoneCreateRequest{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	reqInfo := cres.DNSZoneReqInfo{
		IId:        cres.IID{NameId: req.ReqInfo.Name, SystemId: ""},
		DomainName: req.ReqInfo.DomainName,
		TagList:    req.ReqInfo.TagList,
	}

	result, err := cmrt.CreateDNSZone(req.ConnectionName, DNS, reqInfo, req.IDTransformMode)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

type DNSZoneListResponse struct {
	Result []*cres.DNSZoneInfo `json:"dnszone" validate:"required" description:"A list of DNS Zone information"`
}

// listDNSZone godoc
// @ID list-dnszone
// @Summary List DNS Zones
// @Description Retrieve a list of Public DNS Zones associated with a specific connection.
// @Tags [DNS Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to list DNS Zones for"
// @Success 200 {object} DNSZoneListResponse "List of DNS Zones"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /dnszone [get]
func ListDNSZone(c echo.Context) error {
	cblog.Info("call ListDNSZone()")

	var req ConnectionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	result, err := cmrt.ListDNSZone(req.ConnectionName, DNS)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &DNSZoneListResponse{Result: result})
}

// getDNSZone godoc
// @ID get-dnszone
// @Summary Get DNS Zone
// @Description Retrieve a specific Public DNS Zone, including the records CB-Spider manages in it.
// @Tags [DNS Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection"
// @Param Name path string true "The name of the DNS Zone to retrieve"
// @Success 200 {object} cres.DNSZoneInfo "Details of the requested DNS Zone"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid parameters"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /dnszone/{Name} [get]
func GetDNSZone(c echo.Context) error {
	cblog.Info("call GetDNSZone()")

	var req ConnectionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	result, err := cmrt.GetDNSZone(req.ConnectionName, DNS, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// deleteDNSZone godoc
// @ID delete-dnszone
// @Summary Delete DNS Zone
// @Description Delete a Public DNS Zone, including every record CB-Spider or the CSP has in it.
// @Tags [DNS Management]
// @Accept  json
// @Produce  json
// @Param Name path string true "The name of the DNS Zone to delete"
// @Param force query string false "Force delete the DNS Zone. ex) true or false(default: false)"
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for deleting a DNS Zone"
// @Success 200 {object} BooleanInfo "Result of the delete operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /dnszone/{Name} [delete]
func DeleteDNSZone(c echo.Context) error {
	cblog.Info("call DeleteDNSZone()")

	var req ConnectionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	result, err := cmrt.DeleteDNSZone(req.ConnectionName, DNS, c.Param("Name"), c.QueryParam("force"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &BooleanInfo{Result: strconv.FormatBool(result)})
}

// countAllDNSZones godoc
// @ID count-all-dnszone
// @Summary Count All DNS Zones
// @Description Get the total number of DNS Zones across all connections.
// @Tags [DNS Management]
// @Produce  json
// @Success 200 {object} CountResponse "Total count of DNS Zones"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /countdnszone [get]
func CountAllDNSZones(c echo.Context) error {
	count, err := cmrt.CountAllDNSZones()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, CountResponse{Count: int(count)})
}

// countDNSZonesByConnection godoc
// @ID count-dnszone-by-connection
// @Summary Count DNS Zones by Connection
// @Description Get the total number of DNS Zones for a specific connection.
// @Tags [DNS Management]
// @Produce  json
// @Param ConnectionName path string true "The name of the Connection"
// @Success 200 {object} CountResponse "Total count of DNS Zones for the connection"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /countdnszone/{ConnectionName} [get]
func CountDNSZonesByConnection(c echo.Context) error {
	count, err := cmrt.CountDNSZonesByConnection(c.Param("ConnectionName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, CountResponse{Count: int(count)})
}

//================ DNS Record Handler

// DNSRecordAddRequest represents the request body for adding a record to a DNS Zone.
type DNSRecordAddRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		Name  string   `json:"Name" validate:"required" example:"my-a-record"`
		Type  string   `json:"Type" validate:"required" example:"A"`
		Host  string   `json:"Host" validate:"required" example:"www.example.com"`
		Value []string `json:"Value" validate:"required" example:"203.0.113.10"`
		TTL   int      `json:"TTL,omitempty" validate:"omitempty" example:"300"`
	} `json:"ReqInfo" validate:"required"`
}

// addDNSRecord godoc
// @ID add-dnsrecord
// @Summary Add DNS Record
// @Description Add a new record(ex: A, CNAME) to an existing Public DNS Zone.
// @Tags [DNS Management]
// @Accept  json
// @Produce  json
// @Param DNSZoneName path string true "The name of the DNS Zone to add the record to"
// @Param DNSRecordAddRequest body restruntime.DNSRecordAddRequest true "Request body for adding a DNS Record"
// @Success 200 {object} cres.DNSZoneInfo "Details of the DNS Zone including the added record"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /dnszone/{DNSZoneName}/record [post]
func AddDNSRecord(c echo.Context) error {
	cblog.Info("call AddDNSRecord()")

	req := DNSRecordAddRequest{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	reqInfo := cres.DNSRecordReqInfo{
		IId:   cres.IID{NameId: req.ReqInfo.Name, SystemId: ""},
		Type:  cres.DNSRecordType(req.ReqInfo.Type),
		Name:  req.ReqInfo.Host,
		Value: req.ReqInfo.Value,
		TTL:   req.ReqInfo.TTL,
	}

	result, err := cmrt.AddDNSRecord(req.ConnectionName, DNSRECORD, c.Param("DNSZoneName"), reqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// DNSRecordChangeRequest represents the request body for changing a record in a DNS Zone.
type DNSRecordChangeRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		Type  string   `json:"Type" validate:"required" example:"A"`
		Host  string   `json:"Host" validate:"required" example:"www.example.com"`
		Value []string `json:"Value" validate:"required" example:"203.0.113.20"`
		TTL   int      `json:"TTL,omitempty" validate:"omitempty" example:"300"`
	} `json:"ReqInfo" validate:"required"`
}

// changeDNSRecord godoc
// @ID change-dnsrecord
// @Summary Change DNS Record
// @Description Change the value/TTL of an existing record(by Name) in a Public DNS Zone.
// @Tags [DNS Management]
// @Accept  json
// @Produce  json
// @Param DNSZoneName path string true "The name of the DNS Zone"
// @Param RecordName path string true "The CB-Spider name of the DNS Record to change"
// @Param DNSRecordChangeRequest body restruntime.DNSRecordChangeRequest true "Request body for changing a DNS Record"
// @Success 200 {object} cres.DNSZoneInfo "Details of the DNS Zone including the changed record"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /dnszone/{DNSZoneName}/record/{RecordName} [put]
func ChangeDNSRecord(c echo.Context) error {
	cblog.Info("call ChangeDNSRecord()")

	req := DNSRecordChangeRequest{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	reqInfo := cres.DNSRecordReqInfo{
		IId:   cres.IID{NameId: c.Param("RecordName"), SystemId: ""},
		Type:  cres.DNSRecordType(req.ReqInfo.Type),
		Name:  req.ReqInfo.Host,
		Value: req.ReqInfo.Value,
		TTL:   req.ReqInfo.TTL,
	}

	result, err := cmrt.ChangeDNSRecord(req.ConnectionName, c.Param("DNSZoneName"), reqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// removeDNSRecord godoc
// @ID remove-dnsrecord
// @Summary Remove DNS Record
// @Description Remove an existing record from a Public DNS Zone.
// @Tags [DNS Management]
// @Accept  json
// @Produce  json
// @Param DNSZoneName path string true "The name of the DNS Zone"
// @Param RecordName path string true "The CB-Spider name of the DNS Record to remove"
// @Param force query string false "Force delete the record. ex) true or false(default: false)"
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for removing a DNS Record"
// @Success 200 {object} BooleanInfo "Result of the remove operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /dnszone/{DNSZoneName}/record/{RecordName} [delete]
func RemoveDNSRecord(c echo.Context) error {
	cblog.Info("call RemoveDNSRecord()")

	var req ConnectionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	result, err := cmrt.RemoveDNSRecord(req.ConnectionName, c.Param("DNSZoneName"), c.Param("RecordName"), c.QueryParam("force"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &BooleanInfo{Result: strconv.FormatBool(result)})
}
