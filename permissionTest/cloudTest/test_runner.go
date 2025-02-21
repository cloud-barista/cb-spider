package cloudTest

import (
	"fmt"
	"sync"

	"github.com/cloud-barista/cb-spider/permissionTest/connection"
	"github.com/cloud-barista/cb-spider/permissionTest/resources"
)

var resourceList = []resources.ResourceTester{
	// resources.VPCResource{},
	// resources.SubnetResource{},
	// resources.SecurityGroupResource{},
	// resources.KeypairResource{},
	resources.VMResource{},
}

func RunAllTests(){
    fmt.Println("-----------Running WRITE(CREATE) Tests -----------")
    writeError := RunWriteTests(resourceList)

    fmt.Println("\n-----------# Running READ Tests -----------")
    readError := RunReadTests(resourceList)


    fmt.Println("\n-----------# Running DELETE Tests -----------")
    deleteError := RunDeleteTests(resourceList)

    if writeError || readError || deleteError {
        fmt.Println("\n❌ Test failed")
    } else {
        fmt.Println("\n✅ ALL Test passed")
    }
}

// RunWritesTests runs create tests for the given resources
func RunWriteTests(resources []resources.ResourceTester) bool {
    errChan := make(chan error, len(connection.Connections)*len(resources))

    for _, resource := range resources{
        fmt.Printf("\n### Creating %s ###\n", resource.GetName())

        var wg sync.WaitGroup

        for connName := range connection.Connections{
            wg.Add(1)
            go createResource(connName, resource, &wg, errChan)
        }
        wg.Wait()
    }

    isError := handleErrors(errChan)
    return isError
}

// createResource creates a resource for the given connection
func createResource(conn string, resource resources.ResourceTester, wg *sync.WaitGroup, errChan chan <- error){
    defer wg.Done()
    err := resource.CreateResource(conn, errChan)
    if err != nil{
        errChan <- fmt.Errorf("[ERROR] %s: creation failed for %s: %v", resource.GetName(), conn, err)
    }
}

// RunReadTests runs read tests for the given resources
func RunReadTests(resources []resources.ResourceTester) bool {
    
    errChan := make(chan error, len(connection.Connections)*len(resources))

    for _, resource := range resources{
        var wg sync.WaitGroup
        fmt.Printf("\n### Reading %s ###\n", resource.GetName())

        for connName := range connection.Connections{
            wg.Add(1)
            go readResource(connName, resource, &wg, errChan)
        }
        wg.Wait()
    }
    
    isError := handleErrors(errChan)
    return isError
}

// readResource reads a resource for the given connection
func readResource(conn string, res resources.ResourceTester, wg *sync.WaitGroup, errChan chan <- error){
    defer wg.Done()
    
    err := res.ReadResource(conn, errChan)
    if err != nil{
        errChan <- fmt.Errorf("[ERROR] %s: read failed for %s: %v", res.GetName(), conn, err)
    }
}

// RunDeleteTests runs delete tests for the given resources
func RunDeleteTests(resources []resources.ResourceTester) bool {
    errChan := make(chan error, len(connection.Connections)*len(resources))

    for i:= len(resources) - 1; i>=0; i--{
        resource := resources[i]
        fmt.Printf("\n### Deleting %s ###\n", resource.GetName())

        var wg sync.WaitGroup

        for connName := range connection.Connections{
            wg.Add(1)
            go deleteResource(connName, resource, &wg, errChan)
        }

        wg.Wait()
    }

    isError := handleErrors(errChan)
    return isError
}

// deleteResource deletes a resource for the given connection
func deleteResource(conn string, resource resources.ResourceTester, wg *sync.WaitGroup, errChan chan <- error){
    defer wg.Done()
    err := resource.DeleteResource(conn, errChan)
    if err != nil{
        errChan <- fmt.Errorf("[ERROR] %s: delete failed for %s: %v", resource.GetName(), conn, err)
    }
}

func handleErrors(errChan chan error) bool {
    close(errChan)
    isError := false
    for err := range errChan{
        fmt.Println(err)
        isError = true
    }
    return isError
}

