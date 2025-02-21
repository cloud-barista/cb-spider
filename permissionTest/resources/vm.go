package resources

// "encoding/json"
// "fmt"

// "github.com/cloud-barista/cb-spider/permissionTest/connection"
// "github.com/cloud-barista/cb-spider/permissionTest/request"

type VMResource struct{}

type VMResponse struct{

}

type VMListResponse struct{
	VMList []VMResponse `json:"vm"`
}

func (v VMResource) CreateResource(conn string, errChan chan <- error) error {
	return nil
}

func (v VMResource) ReadResource(conn string, errChan chan <- error) error {
	return nil
}

func (v VMResource) DeleteResource(conn string, errCham chan <- error) error {
	return nil
}





