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

var waitForDeletingResourceTimeSec = 5
var waitForDeletingResourceRetry = 6

var ResourceGroupLock sync.Mutex

func getResourceClient(connectionInfo idrv.ConnectionInfo) (*resources.GroupsClient, *context.Context, error) {
	config := auth.NewClientCredentialsConfig(connectionInfo.CredentialInfo.ClientId, connectionInfo.CredentialInfo.ClientSecret, connectionInfo.CredentialInfo.TenantId)
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil, nil, err
	}

	resourceClient := resources.NewGroupsClient(connectionInfo.CredentialInfo.SubscriptionId)
	resourceClient.Authorizer = authorizer
	ctx, cancel := context.WithTimeout(context.Background(), resourceGroupTimeout*time.Second)
	defer func() {
		cancel()
	}()

	return &resourceClient, &ctx, nil
}

func HasResourceGroup(connectionInfo idrv.ConnectionInfo) (bool, error) {
	ResourceGroupLock.Lock()
	defer func() {
		ResourceGroupLock.Unlock()
	}()

	resourceClient, ctx, err := getResourceClient(connectionInfo)
	if err != nil {
		return false, err
	}

	_, err = (*resourceClient).Get(*ctx, connectionInfo.RegionInfo.ResourceGroup)
	if err != nil {
		de, ok := err.(autorest.DetailedError)
		if ok && de.Original != nil {
			re, ok := de.Original.(*azure.RequestError)
			if ok && re.ServiceError != nil && re.ServiceError.Code == "ResourceGroupNotFound" {
				return false, nil
			}
		}
	}

	return true, nil
}

func CreateResourceGroup(connectionInfo idrv.ConnectionInfo) error {
	ResourceGroupLock.Lock()
	defer func() {
		ResourceGroupLock.Unlock()
	}()

	resourceClient, ctx, err := getResourceClient(connectionInfo)
	if err != nil {
		return err
	}

	_, err = resourceClient.CreateOrUpdate(*ctx, connectionInfo.RegionInfo.ResourceGroup,
		resources.Group{
			Name:     to.StringPtr(connectionInfo.RegionInfo.ResourceGroup),
			Location: to.StringPtr(connectionInfo.RegionInfo.Region),
		})
	if err != nil {
		return err
	}

	return nil
}

func DeleteResourceGroup(logger *logrus.Logger, connectionInfo idrv.ConnectionInfo) error {
	ResourceGroupLock.Lock()
	defer func() {
		ResourceGroupLock.Unlock()
	}()

	resourceClient, ctx, err := getResourceClient(connectionInfo)
	if err != nil {
		return err
	}

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
			"(ResourceGroup: " + connectionInfo.RegionInfo.ResourceGroup + ")")
	}

	logger.Info("Removing resource group. (" + connectionInfo.RegionInfo.ResourceGroup + ")")
	_, err = resourceClient.Delete(*ctx, connectionInfo.RegionInfo.ResourceGroup)
	if err != nil {
		return errors.New("error occurred while deleting the resource group. " +
			"(ResourceGroup: " + connectionInfo.RegionInfo.ResourceGroup + ", " +
			"Error: " + err.Error() + ")")
	}
	logger.Info("Resource group successfully removed. (" + connectionInfo.RegionInfo.ResourceGroup + ")")

	return nil
}
