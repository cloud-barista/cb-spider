package cloudTest

import (
	"fmt"
	"sync"

	"github.com/cloud-barista/cb-spider/permissionTest/connection"
	"github.com/cloud-barista/cb-spider/permissionTest/resources"
)
func RunTests(resource resources.ResourceTester) {
    var wg sync.WaitGroup
    errChan := make(chan error, len(connection.Connections)*3) // 3단계 실행

    // 1️⃣ CREATE 단계 (모든 conn에 대해 실행)
    fmt.Printf("----------------------------%s CREATE Test----------------------------\n", resource.GetName())
    for connName := range connection.Connections {
        wg.Add(1)
        go func(conn string) {
            defer wg.Done()
            err := resource.CreateResource(conn, errChan)
            if err != nil {
                errChan <- fmt.Errorf("[ERROR] %s: create failed for %s: %v", resource.GetName(), conn, err)
            }
        }(connName)
    }
    wg.Wait()

    // 2️⃣ READ 단계 (모든 conn에 대해 실행)
    fmt.Printf("----------------------------%s READ Test----------------------------\n", resource.GetName())
    for connName := range connection.Connections {
        wg.Add(1)
        go func(conn string) {
            defer wg.Done()
            err := resource.ReadResource(conn, errChan)
            if err != nil {
                errChan <- fmt.Errorf("[ERROR] %s: read failed for %s: %v", resource.GetName(), conn, err)
            }
        }(connName)
    }
    wg.Wait()

    // 3️⃣ DELETE 단계 (모든 conn에 대해 실행)
    fmt.Printf("----------------------------%s DELETE Test----------------------------\n", resource.GetName())
    for connName := range connection.Connections {
        wg.Add(1)
        go func(conn string) {
            defer wg.Done()
            err := resource.DeleteResource(conn, errChan)
            if err != nil {
                errChan <- fmt.Errorf("[ERROR] %s: delete failed for %s: %v", resource.GetName(), conn, err)
            }
        }(connName)
    }
    wg.Wait()

    // 에러 채널 닫기
    close(errChan)

    // 에러 출력 및 성공 여부 판단
    errOccurred := false
    for err := range errChan {
        fmt.Println(err)
        errOccurred = true
    }

    // 최종 결과 출력
    if errOccurred {
        fmt.Printf("%s Test Failed❌\n", resource.GetName())
    } else {
        fmt.Printf("%s Test Finished Successfully✅\n", resource.GetName())
    }
}
