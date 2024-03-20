package resources

import (
	"context"
	"errors"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2020-10-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/sirupsen/logrus"
	"strconv"
	"sync"
	"time"
)

const (
	resourceGroupTimeout time.Duration = 10
)

var resourceDeleteQueue map[string]map[string]string // resourceType -> resourceName Map
var resourceDeleteQueueLock sync.Mutex

func addResourceDeleteQueue(resourceType string, resourceName string) {
	resourceDeleteQueueLock.Lock()
	defer func() {
		resourceDeleteQueueLock.Unlock()
	}()

	if resourceDeleteQueue == nil {
		resourceDeleteQueue = make(map[string]map[string]string)
	}

	_, ok := resourceDeleteQueue[resourceType]
	if ok {
		resourceDeleteQueue[resourceType][resourceName] = resourceName
	} else {
		var rNames map[string]string
		rNames = make(map[string]string)
		rNames[resourceName] = resourceName
		resourceDeleteQueue[resourceType] = rNames
	}
}

func DeleteResourceDeleteQueue(resourceType string, resourceName string) {
	resourceDeleteQueueLock.Lock()
	defer func() {
		resourceDeleteQueueLock.Unlock()
	}()

	_, ok := resourceDeleteQueue[resourceType]
	if ok {
		delete(resourceDeleteQueue[resourceType], resourceName)
	}
}

func isResourceDeleteQueueEmpty() bool {
	resourceDeleteQueueLock.Lock()
	defer func() {
		resourceDeleteQueueLock.Unlock()
	}()

	for _, rType := range resourceDeleteQueue {
		if len(rType) != 0 {
			return false
		}
	}

	return true
}

func isResourceGroupEmpty() (bool, error) {
	CreateAllHandlerLock.Lock()
	defer func() {
		CreateAllHandlerLock.Unlock()
	}()

	// TODO: Not getting image list properly
	//imageList, err := Handlers.ImageHandler.ListImage()
	//if err != nil {
	//	return false, err
	//}
	//if len(imageList) != 0 {
	//	return false, nil
	//}

	securityList, err := Handlers.SecurityHandler.ListSecurity()
	if err != nil {
		return false, err
	}
	if len(securityList) != 0 {
		return false, nil
	}

	vpcList, err := Handlers.VPCHandler.ListVPC()
	if err != nil {
		return false, err
	}
	if len(vpcList) != 0 {
		return false, nil
	}

	keyList, err := Handlers.KeyPairHandler.ListKey()
	if err != nil {
		return false, err
	}
	if len(keyList) != 0 {
		return false, nil
	}

	vmList, err := Handlers.VMHandler.ListVM()
	if err != nil {
		return false, err
	}
	if len(vmList) != 0 {
		return false, nil
	}

	nlbList, err := Handlers.NLBHandler.ListNLB()
	if err != nil {
		return false, err
	}
	if len(nlbList) != 0 {
		return false, nil
	}

	diskList, err := Handlers.DiskHandler.ListDisk()
	if err != nil {
		return false, err
	}
	if len(diskList) != 0 {
		return false, nil
	}

	myImageList, err := Handlers.MyImageHandler.ListMyImage()
	if err != nil {
		return false, err
	}
	if len(myImageList) != 0 {
		return false, nil
	}

	clusterList, err := Handlers.ClusterHandler.ListCluster()
	if err != nil {
		return false, err
	}
	if len(clusterList) != 0 {
		return false, nil
	}

	return true, nil
}

var waitForDeletingResourceTimeSec = 5
var waitForDeletingResourceRetry = 6

var ResourceGroupLock sync.Mutex

var CreateAllHandlerLock sync.Mutex

var Handlers struct {
	ClusterHandler    irs.ClusterHandler
	DiskHandler       irs.DiskHandler
	ImageHandler      irs.ImageHandler
	KeyPairHandler    irs.KeyPairHandler
	MyImageHandler    irs.MyImageHandler
	NLBHandler        irs.NLBHandler
	PriceInfoHandler  irs.PriceInfoHandler
	RegionZoneHandler irs.RegionZoneHandler
	SecurityHandler   irs.SecurityHandler
	VMHandler         irs.VMHandler
	VmSpecHandler     irs.VMSpecHandler
	VPCHandler        irs.VPCHandler
}

func CheckResourceGroup(connectionInfo idrv.ConnectionInfo) error {
	ResourceGroupLock.Lock()
	defer func() {
		ResourceGroupLock.Unlock()
	}()

	config := auth.NewClientCredentialsConfig(connectionInfo.CredentialInfo.ClientId, connectionInfo.CredentialInfo.ClientSecret, connectionInfo.CredentialInfo.TenantId)
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil
	}

	resourceClient := resources.NewGroupsClient(connectionInfo.CredentialInfo.SubscriptionId)
	resourceClient.Authorizer = authorizer
	ctx, cancel := context.WithTimeout(context.Background(), resourceGroupTimeout*time.Second)
	defer func() {
		cancel()
	}()

	_, err = resourceClient.Get(ctx, connectionInfo.RegionInfo.ResourceGroup)
	if err != nil {
		de, ok := err.(autorest.DetailedError)
		if ok && de.Original != nil {
			re, ok := de.Original.(*azure.RequestError)
			if ok && re.ServiceError != nil && re.ServiceError.Code == "ResourceGroupNotFound" {
				// 해당 리소스 그룹이 없을 경우 생성
				_, err = resourceClient.CreateOrUpdate(ctx, connectionInfo.RegionInfo.ResourceGroup,
					resources.Group{
						Name:     to.StringPtr(connectionInfo.RegionInfo.ResourceGroup),
						Location: to.StringPtr(connectionInfo.RegionInfo.Region),
					})
				if err != nil {
					return err
				}

				return nil
			}
		}
	}

	return nil
}

func removeResourceGroup(logger *logrus.Logger, credentialInfo idrv.CredentialInfo, regionInfo idrv.RegionInfo) error {
	ResourceGroupLock.Lock()
	defer func() {
		ResourceGroupLock.Unlock()
	}()

	config := auth.NewClientCredentialsConfig(credentialInfo.ClientId, credentialInfo.ClientSecret, credentialInfo.TenantId)
	authorizer, err := config.Authorizer()
	if err != nil {
		return err
	}

	resourceClient := resources.NewGroupsClient(credentialInfo.SubscriptionId)
	resourceClient.Authorizer = authorizer
	ctx, cancel := context.WithTimeout(context.Background(), resourceGroupTimeout*time.Second)
	defer func() {
		cancel()
	}()

	var i = 0
	for i = 0; i < waitForDeletingResourceRetry; i++ {
		if isResourceDeleteQueueEmpty() {
			break
		} else {
			logger.Info("Wait " + strconv.Itoa(waitForDeletingResourceTimeSec) + " seconds for resources to be deleted." +
				"(Retrying " + strconv.Itoa(i+1) + "/" + strconv.Itoa(waitForDeletingResourceRetry) + ")")
			resourceDeleteQueueLock.Lock()
			logger.Info("resourceDeleteQueue :", resourceDeleteQueue)
			resourceDeleteQueueLock.Unlock()
			time.Sleep(time.Second * time.Duration(waitForDeletingResourceTimeSec))
			i++
		}
	}
	if i == waitForDeletingResourceRetry {
		return errors.New("timed out while checking the resource group. " +
			"(ResourceGroup: " + regionInfo.ResourceGroup + ")")
	}

	empty, err := isResourceGroupEmpty()
	if err != nil {
		return errors.New("error occurred while checking if the resource group is empty. " +
			"(ResourceGroup: " + regionInfo.ResourceGroup + ", " +
			"Error: " + err.Error() + ")")
	}

	if empty {
		logger.Info("Removing resource group. (" + regionInfo.ResourceGroup + ")")
		_, err = resourceClient.Delete(ctx, regionInfo.ResourceGroup)
		if err != nil {
			return errors.New("error occurred while deleting the resource group. " +
				"(ResourceGroup: " + regionInfo.ResourceGroup + ", " +
				"Error: " + err.Error() + ")")
		}
		logger.Info("Resource group successfully removed. (" + regionInfo.ResourceGroup + ")")
	}

	return nil
}
