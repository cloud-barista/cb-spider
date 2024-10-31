// Cloud Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//

package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	o2 "golang.org/x/oauth2"
	goo "golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/tpu/v1"
)

type GCPAnyCallHandler struct {
	Region     idrv.RegionInfo
	Ctx        context.Context
	Client     *compute.Service
	Credential idrv.CredentialInfo
}

/*
*******************************************************

	// call example
	curl -sX POST http://localhost:1024/spider/anycall -H 'Content-Type: application/json' -d \
	'{
	        "ConnectionName" : "gcp-iowa-config",
	        "ReqInfo" : {
	                "FID" : "createTags",
	                "IKeyValueList" : [{"Key":"key1", "Value":"value1"}, {"Key":"key2", "Value":"value2"}]
	        }
	}' | json_pp

*******************************************************
*/
func (anyCallHandler *GCPAnyCallHandler) AnyCall(callInfo irs.AnyCallInfo) (irs.AnyCallInfo, error) {
	cblogger.Info("GCP Driver: called AnyCall()!")

	switch callInfo.FID {
	case "LIST_TPU_SUPPORTED_ZONE":
		return listTPUSupportedZone(anyCallHandler, callInfo)

	// control DLVM with TPU
	case "CREATE_TPU_DLVM":
		return createTPU_DLVM(anyCallHandler, callInfo)
	case "DELETE_TPU_DLVM":
		return deleteTPU_DLVM(anyCallHandler, callInfo)

	// control DLVM
	case "CREATE_DLVM":
		return createDLVM(anyCallHandler, callInfo)
	case "LIST_DLVM":
		return listDLVM(anyCallHandler, callInfo)
	case "DELETE_DLVM":
		return deleteDLVM(anyCallHandler, callInfo)

	// control TPU
	case "CREATE_TPU":
		return createTPU(anyCallHandler, callInfo)
	case "LIST_TPU":
		return listTPU(anyCallHandler, callInfo)
	case "DELETE_TPU":
		return deleteTPU(anyCallHandler, callInfo)

	default:
		return irs.AnyCallInfo{}, errors.New("GCP Driver: " + callInfo.FID + " Function is not implemented!")
	}
}

// /////////////////////////////////////////////////////////////////////
// implemented by driver developer
// /////////////////////////////////////////////////////////////////////

//----------------- TPU & DLVM Control

// listTPUSupportedZone retrieves a list of supported TPU zones for the specified project.
func listTPUSupportedZone(anyCallHandler *GCPAnyCallHandler, callInfo irs.AnyCallInfo) (irs.AnyCallInfo, error) {
	cblogger.Info("GCP Driver: called listTPUSupportedZone()!")

	// Initialize the TPU service client
	tpuService, err := getTPUClient(anyCallHandler.Credential)
	if err != nil {
		return callInfo, fmt.Errorf("failed to initialize TPU client: %v", err)
	}

	// Define the parent path at the project level (without specifying a zone)
	projectID := anyCallHandler.Credential.ProjectID
	parent := fmt.Sprintf("projects/%s", projectID)

	// Retrieve the list of supported TPU locations
	resp, err := tpuService.Projects.Locations.List(parent).Context(anyCallHandler.Ctx).Do()
	if err != nil {
		return callInfo, fmt.Errorf("failed to list supported TPU zones: %v", err)
	}

	// Process and output the supported zones
	for _, location := range resp.Locations {
		zone := location.LocationId
		locationPath := fmt.Sprintf("projects/%s/locations/%s", projectID, zone)

		cblogger.Infof("Supported TPU Zone: %s, Full Path: %s", zone, locationPath)

		// Append to OKeyValueList with the specified format
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{
			Key:   zone,         // e.g., "asia-east1-a"
			Value: locationPath, // e.g., "projects/powerkimhub/locations/asia-east1-a"
		})
	}

	return callInfo, nil
}

const (
	tpuType           = "v3-8"   // TPU type, adjust as needed (e.g., v2-8, v3-32, etc.)
	tensorflowVersion = "2.12.1" // "2.8.0"
)

func createTPU_DLVM(anyCallHandler *GCPAnyCallHandler, callInfo irs.AnyCallInfo) (irs.AnyCallInfo, error) {
	cblogger.Info("GCP Driver: called AnyCall()/createTPU_DLVM()!")

	var zoneId, tpuName, dlVMName, firewallName string

	for _, kv := range callInfo.IKeyValueList {
		switch kv.Key {
		case "ZoneId":
			if kv.Value == "" {
				return callInfo, fmt.Errorf("ZoneId is not provided in IKeyValueList")
			}
			zoneId = kv.Value
		case "Name":
			if kv.Value == "" {
				return callInfo, fmt.Errorf("Name is not provided in IKeyValueList")
			}
			tpuName = kv.Value
			firewallName = kv.Value
			dlVMName = kv.Value
		default:
			return callInfo, fmt.Errorf("Invalid Key in IKeyValueList: %s", kv.Key)
		}
	}

	// Initialize OKeyValueList if it's nil
	if callInfo.OKeyValueList == nil {
		callInfo.OKeyValueList = []irs.KeyValue{}
	}

	// Initialize the TPU service client
	tpuService, err := getTPUClient(anyCallHandler.Credential)
	if err != nil {
		return callInfo, fmt.Errorf("failed to initialize TPU client: %v", err)
	}

	// Create the TPU instance
	if err := createTPUInstance(anyCallHandler.Ctx, tpuService, anyCallHandler.Credential.ProjectID, zoneId, tpuName); err != nil {
		cblogger.Errorf("Failed to create TPU: %v", err)
		return callInfo, fmt.Errorf("Failed to create TPU: %v", err)
	}

	vmClient := anyCallHandler.Client

	// Check and create the firewall rule if it does not exist
	if err := createFirewallRule(anyCallHandler, vmClient, firewallName); err != nil {
		return callInfo, fmt.Errorf("failed to create or verify firewall rule: %v", err)
	}
	// Append firewall information to OKeyValueList
	firewallInfo, err := getFirewallInfo(anyCallHandler, firewallName)
	if err == nil {
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{
			Key:   "Firewall",
			Value: firewallInfo,
		})
	}

	// Check if the VM already exists before creating it
	if !checkInstanceExists(anyCallHandler.Ctx, vmClient, anyCallHandler.Credential.ProjectID, zoneId, dlVMName) {
		// Create the VM instance with Deep Learning VM image
		if err := createVMInstance(anyCallHandler.Ctx, vmClient, anyCallHandler.Credential.ProjectID, zoneId, dlVMName); err != nil {
			cblogger.Errorf("Failed to create VM: %v", err)
			return callInfo, fmt.Errorf("Failed to create VM: %v", err)
		}

		// Retrieve Jupyter Lab token from Serial Port 1
		jupyterToken, err := getJupyterToken(anyCallHandler.Ctx, vmClient, anyCallHandler.Credential.ProjectID, zoneId, dlVMName)
		if err != nil {
			return callInfo, fmt.Errorf("failed to retrieve Jupyter Lab token: %v", err)
		}

		// Append VM information, including Jupyter Lab token, to OKeyValueList
		vmInfo, err := getInstanceInfo(anyCallHandler, zoneId, dlVMName)
		if err == nil {
			vmInfoWithToken := fmt.Sprintf("%s, Jupyter Lab token: %s", vmInfo, jupyterToken)
			callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{
				Key:   "VM",
				Value: vmInfoWithToken,
			})
		}

		// Set Jupyter Lab token as a label on the VM
		err = setVMLabel(anyCallHandler, zoneId, dlVMName, "cb_spider_tpu_vm_jupyter_token", jupyterToken)
		if err != nil {
			return callInfo, fmt.Errorf("failed to set Jupyter Lab token label: %v", err)
		}
	} else {
		cblogger.Errorf("%s already exists", dlVMName)
		return callInfo, fmt.Errorf("%s already exists", dlVMName)
	}

	// Append TPU information to OKeyValueList
	tpuInfo, err := getTPUInfo(anyCallHandler, tpuService, zoneId, tpuName)
	if err == nil {
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{
			Key:   "TPU",
			Value: tpuInfo,
		})
	}

	return callInfo, nil
}

// setVMLabel sets a label with a specified key and value on the given VM instance
func setVMLabel(anyCallHandler *GCPAnyCallHandler, zone, instanceName, labelKey, labelValue string) error {
	vmClient := anyCallHandler.Client

	// Retrieve the current labels and fingerprint for the VM instance
	instance, err := vmClient.Instances.Get(anyCallHandler.Credential.ProjectID, zone, instanceName).Context(anyCallHandler.Ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to get instance for label update: %v", err)
	}

	// Update labels map with the new key-value pair
	if instance.Labels == nil {
		instance.Labels = make(map[string]string)
	}
	instance.Labels[labelKey] = labelValue

	// Prepare the set labels request
	labelRequest := &compute.InstancesSetLabelsRequest{
		Labels:           instance.Labels,
		LabelFingerprint: instance.LabelFingerprint,
	}

	// Apply the label update
	_, err = vmClient.Instances.SetLabels(anyCallHandler.Credential.ProjectID, zone, instanceName, labelRequest).Context(anyCallHandler.Ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to set label on instance: %v", err)
	}

	cblogger.Infof("Label '%s: %s' set on instance %s", labelKey, labelValue, instanceName)
	return nil
}

// getFirewallInfo retrieves the detailed information of the created firewall
func getFirewallInfo(anyCallHandler *GCPAnyCallHandler, firewallName string) (string, error) {
	vmClient := anyCallHandler.Client
	firewall, err := vmClient.Firewalls.Get(anyCallHandler.Credential.ProjectID, firewallName).Context(anyCallHandler.Ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to get firewall info: %v", err)
	}

	// Collect allowed protocols and ports
	var allowedInfo []string
	for _, allowed := range firewall.Allowed {
		allowedInfo = append(allowedInfo, fmt.Sprintf("Protocol: %s, Ports: %v", allowed.IPProtocol, allowed.Ports))
	}
	allowedPortsInfo := strings.Join(allowedInfo, "; ")

	// Format the Firewall information string
	info := fmt.Sprintf("Firewall Name: %s, Network: %s, Allowed: [%s], Source Ranges: %v",
		firewall.Name, firewall.Network, allowedPortsInfo, firewall.SourceRanges)
	return info, nil
}

// getInstanceInfo retrieves the detailed information of the created VM
func getInstanceInfo(anyCallHandler *GCPAnyCallHandler, zone, instanceName string) (string, error) {
	vmClient := anyCallHandler.Client
	instance, err := vmClient.Instances.Get(anyCallHandler.Credential.ProjectID, zone, instanceName).Context(anyCallHandler.Ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to get instance info: %v", err)
	}

	// Collect internal and external IP addresses
	var internalIPs, externalIPs []string
	for _, networkInterface := range instance.NetworkInterfaces {
		internalIPs = append(internalIPs, networkInterface.NetworkIP)
		for _, accessConfig := range networkInterface.AccessConfigs {
			externalIPs = append(externalIPs, accessConfig.NatIP)
		}
	}
	internalIPInfo := strings.Join(internalIPs, ", ")
	externalIPInfo := strings.Join(externalIPs, ", ")

	// Collect disk information
	var disks []string
	for _, disk := range instance.Disks {
		disks = append(disks, fmt.Sprintf("DeviceName: %s, Type: %s, SizeGb: %d", disk.DeviceName, disk.Type, disk.DiskSizeGb))
	}
	diskInfo := strings.Join(disks, "; ")

	// Extract and format machine type (e.g., "n1-standard-8")
	machineTypeParts := strings.Split(instance.MachineType, "/")
	machineTypeShort := machineTypeParts[len(machineTypeParts)-1]

	// Format the VM information string
	info := fmt.Sprintf("VM Name: %s, Status: %s, MachineType: %s, Internal IPs: [%s], External IPs: [%s], Disks: [%s], Created: %s",
		instance.Name, instance.Status, machineTypeShort, internalIPInfo, externalIPInfo, diskInfo, instance.CreationTimestamp)
	return info, nil
}

// getTPUInfo retrieves the detailed information of the created TPU
func getTPUInfo(anyCallHandler *GCPAnyCallHandler, tpuService *tpu.Service, zone, tpuName string) (string, error) {
	tpuNode, err := tpuService.Projects.Locations.Nodes.Get(fmt.Sprintf("projects/%s/locations/%s/nodes/%s", anyCallHandler.Credential.ProjectID, zone, tpuName)).Context(anyCallHandler.Ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to get TPU info: %v", err)
	}

	// Collect IP addresses from NetworkEndpoints
	var ipAddresses []string
	for _, endpoint := range tpuNode.NetworkEndpoints {
		ipAddresses = append(ipAddresses, endpoint.IpAddress)
	}
	ipInfo := strings.Join(ipAddresses, ", ")

	// Format the TPU information string
	info := fmt.Sprintf("TPU Name: %s, Status: %s, IP: [%s], Type: %s, TPU SW Version: %s, Created: %s",
		tpuNode.Name, tpuNode.State, ipInfo, tpuNode.AcceleratorType, tpuNode.TensorflowVersion, tpuNode.CreateTime)
	return info, nil
}

func getTPUClient(credential idrv.CredentialInfo) (*tpu.Service, error) {
	gcpType := "service_account"
	data := make(map[string]string)

	data["type"] = gcpType
	data["private_key"] = credential.PrivateKey
	data["client_email"] = credential.ClientEmail

	res, _ := json.Marshal(data)
	authURL := "https://www.googleapis.com/auth/cloud-platform"

	conf, err := goo.JWTConfigFromJSON(res, authURL)
	if err != nil {
		return nil, err
	}

	var client *http.Client
	// Use the default client if CALL_COUNT is not set.
	client = conf.Client(o2.NoContext)

	tpuService, err := tpu.New(client)
	if err != nil {
		return nil, err
	}

	return tpuService, nil
}

func checkInstanceExists(ctx context.Context, computeService *compute.Service, projectID string, zone string, instanceName string) bool {
	_, err := computeService.Instances.Get(projectID, zone, instanceName).Context(ctx).Do()
	return err == nil
}

func createVMInstance(ctx context.Context, computeService *compute.Service, projectID, zone, instanceName string) error {
	instance := &compute.Instance{
		Name:        instanceName,
		MachineType: fmt.Sprintf("zones/%s/machineTypes/n1-standard-8", zone),
		Disks: []*compute.AttachedDisk{
			{
				Boot:       true,
				AutoDelete: true,
				InitializeParams: &compute.AttachedDiskInitializeParams{
					SourceImage: "projects/deeplearning-platform-release/global/images/family/tf-2-17-cpu",
				},
			},
		},
		NetworkInterfaces: []*compute.NetworkInterface{
			{
				AccessConfigs: []*compute.AccessConfig{
					{
						Name: "External NAT",
						Type: "ONE_TO_ONE_NAT",
					},
				},
			},
		},
		Metadata: &compute.Metadata{
			Items: []*compute.MetadataItems{
				{
					Key:   "startup-script",
					Value: setupScript(),
				},
			},
		},

		// Service account configuration for GCS access
		ServiceAccounts: []*compute.ServiceAccount{
			{
				Email: "powerkimhub@powerkimhub.iam.gserviceaccount.com",
				Scopes: []string{
					"https://www.googleapis.com/auth/devstorage.read_only", // GCS read access
					"https://www.googleapis.com/auth/cloud-platform",       // Full access to other GCP resources
				},
			},
		},
	}

	cblogger.Infof("Creating instance %s in zone %s...\n", instanceName, zone)
	op, err := computeService.Instances.Insert(projectID, zone, instance).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("could not create instance: %v", err)
	}

	cblogger.Infof("Waiting for VM creation to complete...")
	err = waitForOperation(ctx, computeService, projectID, zone, op.Name)
	if err != nil {
		return fmt.Errorf("could not complete VM creation: %v", err)
	}

	return nil
}

// getJupyterToken retrieves the Jupyter token from the instance's serial port output
func getJupyterToken(ctx context.Context, computeService *compute.Service, projectID, zone, instanceName string) (string, error) {
	for i := 0; i < 40; i++ { // Retry up to 40 times with 5-second intervals
		resp, err := computeService.Instances.GetSerialPortOutput(projectID, zone, instanceName).Port(1).Context(ctx).Do()
		if err != nil {
			return "", fmt.Errorf("could not retrieve serial port output: %v", err)
		}

		// Look for the Jupyter token in the serial port output
		if token := parseJupyterToken(resp.Contents); token != "" {
			return token, nil
		}

		cblogger.Infof("Jupyter token not found yet, retrying in 5 seconds...")
		time.Sleep(5 * time.Second)
	}
	return "", fmt.Errorf("jupyter token not found after multiple attempts")
}

// parseJupyterToken searches the serial output for the token
func parseJupyterToken(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "Jupyter Lab token:") {
			return strings.TrimSpace(strings.Split(line, "Jupyter Lab token:")[1])
		}
	}
	return ""
}

// setupScript provides the startup script for the VM, which installs necessary packages and sets up TensorFlow and Jupyter.
func setupScript() *string {
	script := `#!/bin/bash

    # Create directories for Jupyter and TensorBoard logs
    mkdir -p /var/log/jupyter
    mkdir -p /var/log/tensorboard

    # Function to check GCS access availability
    check_gcs_access() {
        local RETRY_COUNT=10
        local RETRY_INTERVAL=5 # in seconds

        for ((i=1; i<=RETRY_COUNT; i++)); do
            if gsutil ls gs://powerkim_bucket/logs &>/dev/null; then
                echo "GCS access confirmed: gs://powerkim_bucket/logs"
                return 0
            else
                echo "GCS access not yet available, retrying... ($i/$RETRY_COUNT)"
                sleep $RETRY_INTERVAL
            fi
        done

        echo "Failed to access GCS: possible permission issues."
        return 1
    }

    # Start Jupyter Lab server without disabling the token
    nohup /opt/conda/bin/jupyter lab --ip=0.0.0.0 --port=8888 --no-browser --allow-root > /var/log/jupyter/jupyter.log 2>&1 &

    # Loop to check for Jupyter token, up to 10 attempts with 5-second intervals
    for i in {1..10}; do
        JUPYTER_TOKEN=$(grep -oP 'token=\K\w+' /var/log/jupyter/jupyter.log | head -1)
        if [ -n "$JUPYTER_TOKEN" ]; then
            echo "Jupyter Lab token: $JUPYTER_TOKEN" | tee /dev/ttyS1  # Print token to Serial Port 1
            break
        else
            echo "Jupyter token not found, retrying in 5 seconds... ($i/10)"
            sleep 5
        fi
    done

    # If token was not found after 10 attempts, print an error message to Serial Port
    if [ -z "$JUPYTER_TOKEN" ]; then
        echo "Failed to retrieve Jupyter Lab token after multiple attempts." | tee /dev/ttyS1
    fi

    # Check for GCS access and then start the TensorBoard server
    if check_gcs_access; then
        nohup /opt/conda/bin/tensorboard --logdir=gs://powerkim_bucket/logs --host=0.0.0.0 --port=6006 > /var/log/tensorboard/tensorboard.log 2>&1 &
    else
        echo "Cannot start TensorBoard: GCS access permission issue."
    fi
    `
	return &script
}

func createTPUInstance(ctx context.Context, tpuService *tpu.Service, projectID string, zone string, tpuName string) error {
	cblogger.Infof("Creating TPU %s in zone %s...\n", tpuName, zone)

	// Define labels for TPU, including the managed label
	labels := map[string]string{
		"cb_spider_managed_tpu": tpuName,
	}

	// Create TPU node with labels
	tpuNode := &tpu.Node{
		AcceleratorType:   tpuType,
		TensorflowVersion: tensorflowVersion,
		Network:           fmt.Sprintf("projects/%s/global/networks/default", projectID),
		Labels:            labels,
	}

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, zone)
	op, err := tpuService.Projects.Locations.Nodes.Create(parent, tpuNode).NodeId(tpuName).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("could not create TPU: %v", err)
	}

	cblogger.Infof("TPU creation operation started: %s", op.Name)

	cblogger.Infof("Waiting for TPU creation to complete...")
	for {
		result, err := tpuService.Projects.Locations.Operations.Get(op.Name).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("could not get TPU creation status: %v", err)
		}

		if result.Done {
			if result.Error != nil {
				return fmt.Errorf("TPU creation completed with error: %v", result.Error)
			}
			cblogger.Infof("TPU %s created successfully.", tpuName)
			break
		}

		time.Sleep(5 * time.Second)
	}

	return nil
}

func createFirewallRule(anyCallHandler *GCPAnyCallHandler, computeService *compute.Service, firewallName string) error {
	existingRule, err := computeService.Firewalls.Get(anyCallHandler.Credential.ProjectID, firewallName).Context(anyCallHandler.Ctx).Do()
	if err == nil {
		cblogger.Info("Firewall rule already exists:", existingRule.Name)
		return nil
	}

	rule := &compute.Firewall{
		Name:    firewallName,
		Network: fmt.Sprintf("projects/%s/global/networks/default", anyCallHandler.Credential.ProjectID),
		Allowed: []*compute.FirewallAllowed{
			{IPProtocol: "tcp", Ports: []string{"8888", "6006", "8470", "8471", "5000"}},
		},
		SourceRanges: []string{"0.0.0.0/0"},
	}

	op, err := computeService.Firewalls.Insert(anyCallHandler.Credential.ProjectID, rule).Context(anyCallHandler.Ctx).Do()
	if err != nil {
		return fmt.Errorf("could not create firewall rule: %v", err)
	}

	cblogger.Infof("Waiting for firewall rule creation to complete...")
	for {
		result, err := computeService.GlobalOperations.Get(anyCallHandler.Credential.ProjectID, op.Name).Context(anyCallHandler.Ctx).Do()
		if err != nil {
			return fmt.Errorf("could not get firewall creation status: %v", err)
		}
		if result.Status == "DONE" {
			cblogger.Infof("Firewall rule created successfully.")
			break
		}
		time.Sleep(5 * time.Second)
	}

	return nil
}

func waitForOperation(ctx context.Context, computeService *compute.Service, projectID string, zone string, operationName string) error {
	for {
		result, err := computeService.ZoneOperations.Get(projectID, zone, operationName).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("could not get operation status: %v", err)
		}
		if result.Status == "DONE" {
			if result.Error != nil {
				return fmt.Errorf("operation completed with errors: %v", result.Error)
			}
			cblogger.Infof("Operation completed successfully.")
			break
		}
		time.Sleep(5 * time.Second)
	}
	return nil
}

// listDLVM retrieves a list of VM instances in the specified zone and project, filtering for those with a Jupyter Lab token label.
func listDLVM(anyCallHandler *GCPAnyCallHandler, callInfo irs.AnyCallInfo) (irs.AnyCallInfo, error) {
	cblogger.Info("GCP Driver: called listDLVM()!")

	// Parse input parameters
	var zoneId string
	for _, kv := range callInfo.IKeyValueList {
		if kv.Key == "ZoneId" {
			zoneId = kv.Value
			break
		}
	}
	if zoneId == "" {
		return callInfo, errors.New("ZoneId is required in IKeyValueList")
	}

	// Initialize the Compute Engine client
	vmClient := anyCallHandler.Client

	// Fetch the list of VM Instances
	resp, err := vmClient.Instances.List(anyCallHandler.Credential.ProjectID, zoneId).Context(anyCallHandler.Ctx).Do()
	if err != nil {
		return callInfo, fmt.Errorf("failed to list VM instances in zone %s: %v", zoneId, err)
	}

	// Filter and populate OKeyValueList with details of VMs having the Jupyter Lab token label
	for _, instance := range resp.Items {
		// Check if the Jupyter Lab token label exists
		jupyterToken, hasToken := instance.Labels["cb_spider_tpu_vm_jupyter_token"]
		if !hasToken {
			continue // Skip VMs without the Jupyter Lab token label
		}

		// Get VM information
		vmInfo, err := getInstanceInfo(anyCallHandler, zoneId, instance.Name)
		if err != nil {
			cblogger.Errorf("Failed to get VM info for %s: %v", instance.Name, err)
			continue
		}

		// Append VM information, including Jupyter Lab token from the label, to OKeyValueList
		vmInfoWithToken := fmt.Sprintf("%s, Jupyter Lab token: %s", vmInfo, jupyterToken)
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{
			Key:   instance.Name,
			Value: vmInfoWithToken,
		})
	}

	return callInfo, nil
}

// listTPU retrieves a list of TPU instances in the specified zone and project.
// listTPU retrieves a list of TPU instances in the specified zone and project, filtering for those with the managed label.
func listTPU(anyCallHandler *GCPAnyCallHandler, callInfo irs.AnyCallInfo) (irs.AnyCallInfo, error) {
	cblogger.Info("GCP Driver: called listTPU()!")

	// Parse input parameters
	var zoneId string
	for _, kv := range callInfo.IKeyValueList {
		if kv.Key == "ZoneId" {
			zoneId = kv.Value
			break
		}
	}
	if zoneId == "" {
		return callInfo, errors.New("ZoneId is required in IKeyValueList")
	}

	// Initialize the TPU service client
	tpuService, err := getTPUClient(anyCallHandler.Credential)
	if err != nil {
		return callInfo, fmt.Errorf("failed to initialize TPU client: %v", err)
	}

	// Define the parent location path (project and zone)
	parent := fmt.Sprintf("projects/%s/locations/%s", anyCallHandler.Credential.ProjectID, zoneId)

	// Fetch the list of TPU Nodes
	resp, err := tpuService.Projects.Locations.Nodes.List(parent).Context(anyCallHandler.Ctx).Do()
	if err != nil {
		return callInfo, fmt.Errorf("failed to list TPUs in zone %s: %v", zoneId, err)
	}

	// Populate OKeyValueList with the details of each TPU having the managed label
	for _, node := range resp.Nodes {
		// Check if the managed label exists
		if _, ok := node.Labels["cb_spider_managed_tpu"]; !ok {
			continue // Skip TPUs without the managed label
		}

		// Extract short TPU name from node.Name
		nameParts := strings.Split(node.Name, "/")
		shortTPUName := nameParts[len(nameParts)-1]

		// Collect IP addresses from NetworkEndpoints
		var ipAddresses []string
		for _, endpoint := range node.NetworkEndpoints {
			ipAddresses = append(ipAddresses, endpoint.IpAddress)
		}
		ipInfo := strings.Join(ipAddresses, ", ")

		// Get TPU software version and creation time directly from the node object
		tpuVersion := node.TensorflowVersion
		creationTime := node.CreateTime

		// Format the TPU information string with "IP" after "Status"
		info := fmt.Sprintf("TPU Name: %s, Status: %s, IP: [%s], Type: %s, TPU SW Version: %s, Created: %s",
			shortTPUName, node.State, ipInfo, node.AcceleratorType, tpuVersion, creationTime)
		cblogger.Infof(info)
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{
			Key:   shortTPUName,
			Value: info,
		})
	}

	return callInfo, nil
}

func deleteTPU_DLVM(anyCallHandler *GCPAnyCallHandler, callInfo irs.AnyCallInfo) (irs.AnyCallInfo, error) {
	cblogger.Info("GCP Driver: called AnyCall()/deleteTPU_DLVM()!")

	var zoneId, tpuName, dlVMName, firewallName string

	// Parse input parameters
	for _, kv := range callInfo.IKeyValueList {
		switch kv.Key {
		case "ZoneId":
			if kv.Value == "" {
				return callInfo, fmt.Errorf("ZoneId is not provided in IKeyValueList")
			}
			zoneId = kv.Value
		case "Name":
			if kv.Value == "" {
				return callInfo, fmt.Errorf("Name is not provided in IKeyValueList")
			}
			tpuName = kv.Value
			firewallName = kv.Value
			dlVMName = kv.Value
		default:
			return callInfo, fmt.Errorf("Invalid Key in IKeyValueList: %s", kv.Key)
		}
	}

	// Initialize OKeyValueList if it's nil
	if callInfo.OKeyValueList == nil {
		callInfo.OKeyValueList = []irs.KeyValue{}
	}

	vmClient := anyCallHandler.Client
	tpuService, err := getTPUClient(anyCallHandler.Credential)
	if err != nil {
		return callInfo, fmt.Errorf("failed to initialize TPU client: %v", err)
	}

	// Delete the VM instance
	if err := deleteVMInstance(anyCallHandler.Ctx, vmClient, anyCallHandler.Credential.ProjectID, zoneId, dlVMName); err != nil {
		if strings.Contains(err.Error(), "notFound") || strings.Contains(err.Error(), "not found") {
			cblogger.Infof("VM %s not found, skipping deletion.", dlVMName)
		} else {
			return callInfo, fmt.Errorf("failed to delete VM: %v", err)
		}
	} else {
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{
			Key:   "VM",
			Value: fmt.Sprintf("VM %s deleted successfully", dlVMName),
		})
	}

	// Delete the Firewall rule
	if err := deleteFirewallRule(anyCallHandler, firewallName); err != nil {
		if strings.Contains(err.Error(), "notFound") || strings.Contains(err.Error(), "not found") {
			cblogger.Infof("Firewall rule %s not found, skipping deletion.", firewallName)
		} else {
			return callInfo, fmt.Errorf("failed to delete Firewall rule: %v", err)
		}
	} else {
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{
			Key:   "Firewall",
			Value: fmt.Sprintf("Firewall %s deleted successfully", firewallName),
		})
	}

	// Delete the TPU instance
	if err := deleteTPUInstance(anyCallHandler.Ctx, tpuService, anyCallHandler.Credential.ProjectID, zoneId, tpuName); err != nil {
		if strings.Contains(err.Error(), "notFound") || strings.Contains(err.Error(), "not found") {
			cblogger.Infof("TPU %s not found, skipping deletion.", tpuName)
		} else {
			return callInfo, fmt.Errorf("failed to delete TPU: %v", err)
		}
	} else {
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{
			Key:   "TPU",
			Value: fmt.Sprintf("TPU %s deleted successfully", tpuName),
		})
	}

	return callInfo, nil
}

// deleteVMInstance deletes a VM instance in the specified zone and project, logging if not found.
func deleteVMInstance(ctx context.Context, computeService *compute.Service, projectID, zone, instanceName string) error {
	cblogger.Infof("Deleting VM instance %s in zone %s...\n", instanceName, zone)
	op, err := computeService.Instances.Delete(projectID, zone, instanceName).Context(ctx).Do()
	if err != nil {
		if strings.Contains(err.Error(), "notFound") || strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("VM %s not found", instanceName) // Not an error, only log info
		}
		return fmt.Errorf("could not delete VM instance: %v", err)
	}
	return waitForOperation(ctx, computeService, projectID, zone, op.Name)
}

// deleteFirewallRule deletes a specified firewall rule, logging if not found.
func deleteFirewallRule(anyCallHandler *GCPAnyCallHandler, firewallName string) error {
	cblogger.Infof("Deleting Firewall rule %s...\n", firewallName)
	op, err := anyCallHandler.Client.Firewalls.Delete(anyCallHandler.Credential.ProjectID, firewallName).Context(anyCallHandler.Ctx).Do()
	if err != nil {
		if strings.Contains(err.Error(), "notFound") || strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("Firewall rule %s not found", firewallName) // Not an error, only log info
		}
		return fmt.Errorf("could not delete firewall rule: %v", err)
	}
	return waitForGlobalOperation(anyCallHandler.Ctx, anyCallHandler.Client, anyCallHandler.Credential.ProjectID, op.Name)
}

// deleteTPUInstance deletes a TPU instance, logging if not found.
// deleteTPUInstance deletes a TPU instance and waits for its deletion to complete.
func deleteTPUInstance(ctx context.Context, tpuService *tpu.Service, projectID, zone, tpuName string) error {
	cblogger.Infof("Deleting TPU instance %s in zone %s...\n", tpuName, zone)
	op, err := tpuService.Projects.Locations.Nodes.Delete(fmt.Sprintf("projects/%s/locations/%s/nodes/%s", projectID, zone, tpuName)).Context(ctx).Do()
	if err != nil {
		if strings.Contains(err.Error(), "notFound") || strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("TPU %s not found", tpuName) // Not an error, only log info
		}
		return fmt.Errorf("could not delete TPU instance: %v", err)
	}

	// Wait for TPU operation to complete using TPU-specific operation handling
	for {
		result, err := tpuService.Projects.Locations.Operations.Get(op.Name).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("could not get TPU deletion status: %v", err)
		}
		if result.Done {
			if result.Error != nil {
				return fmt.Errorf("TPU deletion completed with errors: %v", result.Error)
			}
			cblogger.Infof("TPU %s deleted successfully.", tpuName)
			break
		}
		time.Sleep(5 * time.Second)
	}

	return nil
}

// waitForGlobalOperation waits for a global operation to complete.
func waitForGlobalOperation(ctx context.Context, computeService *compute.Service, projectID, operationName string) error {
	for {
		result, err := computeService.GlobalOperations.Get(projectID, operationName).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("could not get global operation status: %v", err)
		}
		if result.Status == "DONE" {
			if result.Error != nil {
				return fmt.Errorf("global operation completed with errors: %v", result.Error)
			}
			cblogger.Infof("Global operation %s completed successfully.", operationName)
			break
		}
		time.Sleep(5 * time.Second)
	}
	return nil
}

func createDLVM(anyCallHandler *GCPAnyCallHandler, callInfo irs.AnyCallInfo) (irs.AnyCallInfo, error) {
	cblogger.Info("GCP Driver: called AnyCall()/createDLVM()!")

	var zoneId, dlVMName, firewallName string

	// Parse input parameters
	for _, kv := range callInfo.IKeyValueList {
		switch kv.Key {
		case "ZoneId":
			if kv.Value == "" {
				return callInfo, fmt.Errorf("ZoneId is not provided in IKeyValueList")
			}
			zoneId = kv.Value
		case "Name":
			if kv.Value == "" {
				return callInfo, fmt.Errorf("Name is not provided in IKeyValueList")
			}
			firewallName = kv.Value
			dlVMName = kv.Value
		default:
			return callInfo, fmt.Errorf("Invalid Key in IKeyValueList: %s", kv.Key)
		}
	}

	// Initialize OKeyValueList if it's nil
	if callInfo.OKeyValueList == nil {
		callInfo.OKeyValueList = []irs.KeyValue{}
	}

	vmClient := anyCallHandler.Client

	// Check and create the firewall rule if it does not exist
	if err := createFirewallRule(anyCallHandler, vmClient, firewallName); err != nil {
		return callInfo, fmt.Errorf("failed to create or verify firewall rule: %v", err)
	}

	// Append firewall information to OKeyValueList
	firewallInfo, err := getFirewallInfo(anyCallHandler, firewallName)
	if err == nil {
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{
			Key:   "Firewall",
			Value: firewallInfo,
		})
	}

	// Check if the VM already exists before creating it
	if !checkInstanceExists(anyCallHandler.Ctx, vmClient, anyCallHandler.Credential.ProjectID, zoneId, dlVMName) {
		// Create the VM instance with a DLVM image
		if err := createVMInstance(anyCallHandler.Ctx, vmClient, anyCallHandler.Credential.ProjectID, zoneId, dlVMName); err != nil {
			cblogger.Errorf("Failed to create VM: %v", err)
			return callInfo, fmt.Errorf("Failed to create VM: %v", err)
		}

		// Retrieve Jupyter Lab token from Serial Port 1
		jupyterToken, err := getJupyterToken(anyCallHandler.Ctx, vmClient, anyCallHandler.Credential.ProjectID, zoneId, dlVMName)
		if err != nil {
			return callInfo, fmt.Errorf("failed to retrieve Jupyter Lab token: %v", err)
		}

		// Append VM information, including Jupyter Lab token, to OKeyValueList
		vmInfo, err := getInstanceInfo(anyCallHandler, zoneId, dlVMName)
		if err == nil {
			vmInfoWithToken := fmt.Sprintf("%s, Jupyter Lab token: %s", vmInfo, jupyterToken)
			callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{
				Key:   "VM",
				Value: vmInfoWithToken,
			})
		}

		// Set Jupyter Lab token as a label on the VM
		err = setVMLabel(anyCallHandler, zoneId, dlVMName, "cb_spider_dlvm_jupyter_token", jupyterToken)
		if err != nil {
			return callInfo, fmt.Errorf("failed to set Jupyter Lab token label: %v", err)
		}
	} else {
		cblogger.Errorf("%s already exists", dlVMName)
		return callInfo, fmt.Errorf("%s already exists", dlVMName)
	}

	return callInfo, nil
}

func deleteTPU(anyCallHandler *GCPAnyCallHandler, callInfo irs.AnyCallInfo) (irs.AnyCallInfo, error) {
	cblogger.Info("GCP Driver: called AnyCall()/deleteTPU()!")

	var zoneId, tpuName string

	// Parse input parameters
	for _, kv := range callInfo.IKeyValueList {
		switch kv.Key {
		case "ZoneId":
			if kv.Value == "" {
				return callInfo, fmt.Errorf("ZoneId is not provided in IKeyValueList")
			}
			zoneId = kv.Value
		case "Name":
			if kv.Value == "" {
				return callInfo, fmt.Errorf("Name is not provided in IKeyValueList")
			}
			tpuName = kv.Value
		default:
			return callInfo, fmt.Errorf("Invalid Key in IKeyValueList: %s", kv.Key)
		}
	}

	// Initialize TPU service client
	tpuService, err := getTPUClient(anyCallHandler.Credential)
	if err != nil {
		return callInfo, fmt.Errorf("failed to initialize TPU client: %v", err)
	}

	// Delete the TPU instance
	if err := deleteTPUInstance(anyCallHandler.Ctx, tpuService, anyCallHandler.Credential.ProjectID, zoneId, tpuName); err != nil {
		return callInfo, fmt.Errorf("failed to delete TPU: %v", err)
	} else {
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{
			Key:   "TPU",
			Value: fmt.Sprintf("TPU %s deleted successfully", tpuName),
		})
	}

	return callInfo, nil
}

// Function to handle the DELETE_DLVM request, which deletes a VM instance and its associated firewall
func deleteDLVM(anyCallHandler *GCPAnyCallHandler, callInfo irs.AnyCallInfo) (irs.AnyCallInfo, error) {
	cblogger.Info("GCP Driver: called AnyCall()/deleteDLVM()!")

	var zoneId, dlVMName, firewallName string

	// Parse input parameters
	for _, kv := range callInfo.IKeyValueList {
		switch kv.Key {
		case "ZoneId":
			if kv.Value == "" {
				return callInfo, fmt.Errorf("ZoneId is not provided in IKeyValueList")
			}
			zoneId = kv.Value
		case "Name": // Name used for both VM and firewall in this context
			if kv.Value == "" {
				return callInfo, fmt.Errorf("Name is not provided in IKeyValueList")
			}
			dlVMName = kv.Value
			firewallName = kv.Value
		default:
			return callInfo, fmt.Errorf("Invalid Key in IKeyValueList: %s", kv.Key)
		}
	}

	// Initialize OKeyValueList if it's nil
	if callInfo.OKeyValueList == nil {
		callInfo.OKeyValueList = []irs.KeyValue{}
	}

	vmClient := anyCallHandler.Client

	// Delete the VM instance
	if err := deleteVMInstance(anyCallHandler.Ctx, vmClient, anyCallHandler.Credential.ProjectID, zoneId, dlVMName); err != nil {
		return callInfo, fmt.Errorf("failed to delete VM: %v", err)
	} else {
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{
			Key:   "VM",
			Value: fmt.Sprintf("VM %s deleted successfully", dlVMName),
		})
	}

	// Delete the Firewall rule
	if err := deleteFirewallRule(anyCallHandler, firewallName); err != nil {
		if strings.Contains(err.Error(), "notFound") || strings.Contains(err.Error(), "not found") {
			cblogger.Infof("Firewall rule %s not found, skipping deletion.", firewallName)
		} else {
			return callInfo, fmt.Errorf("failed to delete Firewall rule: %v", err)
		}
	} else {
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{
			Key:   "Firewall",
			Value: fmt.Sprintf("Firewall %s deleted successfully", firewallName),
		})
	}

	return callInfo, nil
}

// Function to handle the CREATE_TPU request, which creates a TPU instance with specified configurations
func createTPU(anyCallHandler *GCPAnyCallHandler, callInfo irs.AnyCallInfo) (irs.AnyCallInfo, error) {
	cblogger.Info("GCP Driver: called AnyCall()/createTPU()!")

	var zoneId, tpuName string

	// Parse input parameters
	for _, kv := range callInfo.IKeyValueList {
		switch kv.Key {
		case "ZoneId":
			if kv.Value == "" {
				return callInfo, fmt.Errorf("ZoneId is not provided in IKeyValueList")
			}
			zoneId = kv.Value
		case "Name": // Name of the TPU instance
			if kv.Value == "" {
				return callInfo, fmt.Errorf("Name is not provided in IKeyValueList")
			}
			tpuName = kv.Value
		default:
			return callInfo, fmt.Errorf("Invalid Key in IKeyValueList: %s", kv.Key)
		}
	}

	// Initialize OKeyValueList if it's nil
	if callInfo.OKeyValueList == nil {
		callInfo.OKeyValueList = []irs.KeyValue{}
	}

	// Initialize the TPU service client
	tpuService, err := getTPUClient(anyCallHandler.Credential)
	if err != nil {
		return callInfo, fmt.Errorf("failed to initialize TPU client: %v", err)
	}

	// Create the TPU instance
	if err := createTPUInstance(anyCallHandler.Ctx, tpuService, anyCallHandler.Credential.ProjectID, zoneId, tpuName); err != nil {
		return callInfo, fmt.Errorf("failed to create TPU: %v", err)
	}

	// Retrieve and append TPU information to OKeyValueList
	tpuInfo, err := getTPUInfo(anyCallHandler, tpuService, zoneId, tpuName)
	if err != nil {
		return callInfo, fmt.Errorf("failed to retrieve TPU info after creation: %v", err)
	}
	callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{
		Key:   "TPU",
		Value: tpuInfo,
	})

	return callInfo, nil
}
