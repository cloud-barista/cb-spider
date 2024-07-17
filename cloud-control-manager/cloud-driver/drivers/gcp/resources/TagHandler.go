// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver

package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	compute "google.golang.org/api/compute/v1"
	container "google.golang.org/api/container/v1"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type GCPTagHandler struct {
	Region     idrv.RegionInfo
	Ctx        context.Context
	Credential idrv.CredentialInfo

	ComputeClient   *compute.Service
	ContainerClient *container.Service
}

var (
	supportRSType = map[irs.RSType]interface{}{
		irs.VM: nil, irs.DISK: nil, irs.CLUSTER: nil,
	}
)

func validateSupportRS(resType irs.RSType) error {
	if _, ok := supportRSType[resType]; !ok {
		return errors.New("unsupported resources type")
	}
	return nil
}

func (t *GCPTagHandler) getVm(resIID irs.IID) (*compute.Instance, error) {
	vm, err := t.ComputeClient.Instances.Get(t.Credential.ProjectID, t.Region.Zone, resIID.SystemId).Do()
	if err != nil {
		return nil, err
	}

	return vm, nil
}

func (t *GCPTagHandler) getDisk(resIID irs.IID) (*compute.Disk, error) {
	disk, err := GetDiskInfo(t.ComputeClient, t.Credential, t.Region, resIID.SystemId)
	if err != nil {
		return nil, err
	}

	return disk, nil
}

func (t *GCPTagHandler) getCluster(resIID irs.IID) (*container.Cluster, error) {
	parent := getParentClusterAtContainer(t.Credential.ProjectID, t.Region.Zone, resIID.SystemId)
	cluster, err := t.ContainerClient.Projects.Locations.Clusters.Get(parent).Do()
	if err != nil {
		return nil, err
	}

	return cluster, nil
}

func (t *GCPTagHandler) AddTag(resType irs.RSType, resIID irs.IID, tag irs.KeyValue) (irs.KeyValue, error) {
	err := validateSupportRS(resType)
	errRes := irs.KeyValue{}
	if err != nil {
		return errRes, err
	}

	projectId := t.Credential.ProjectID
	zone := t.Region.Zone
	switch resType {
	case irs.VM:
		vm, err := t.getVm(resIID)
		if err != nil {
			return errRes, err
		}

		existLabels := vm.Labels
		if existLabels == nil {
			existLabels = make(map[string]string)
		}
		existLabels[tag.Key] = tag.Value

		req := &compute.InstancesSetLabelsRequest{
			LabelFingerprint: vm.LabelFingerprint,
			Labels:           existLabels,
		}

		op, err := t.ComputeClient.Instances.SetLabels(projectId, zone, resIID.SystemId, req).Do()

		if err != nil {
			return errRes, err
		}

		if op.Error != nil {
			return errRes, fmt.Errorf("operation failed: %v", op.Error.Errors)
		}

		return tag, nil
	case irs.DISK:

		disk, err := t.getDisk(resIID)
		if err != nil {
			return errRes, err
		}

		existLabels := disk.Labels
		if existLabels == nil {
			existLabels = make(map[string]string)
		}
		existLabels[tag.Key] = tag.Value

		req := &compute.ZoneSetLabelsRequest{
			LabelFingerprint: disk.LabelFingerprint,
			Labels:           existLabels,
		}

		op, err := t.ComputeClient.Disks.SetLabels(projectId, zone, resIID.SystemId, req).Do()

		if err != nil {
			return errRes, err
		}

		if op.Error != nil {
			return errRes, fmt.Errorf("operation failed: %v", op.Error.Errors)
		}

		return tag, nil
	case irs.CLUSTER:
		cluster, err := t.getCluster(resIID)
		if err != nil {
			return errRes, err
		}

		existLabels := cluster.ResourceLabels
		if existLabels == nil {
			existLabels = make(map[string]string)
		}
		existLabels[tag.Key] = tag.Value

		name := getParentClusterAtContainer(projectId, zone, resIID.SystemId)
		req := &container.SetLabelsRequest{
			ClusterId:        resIID.SystemId,
			LabelFingerprint: cluster.LabelFingerprint,
			Name:             name,
			ProjectId:        projectId,
			Zone:             zone,
			ResourceLabels:   existLabels,
		}
		op, err := t.ContainerClient.Projects.Locations.Clusters.SetResourceLabels(name, req).Do()

		if err != nil {
			return errRes, err
		}

		if op.Error != nil {
			return errRes, fmt.Errorf("operation failed: %v", op.Error.Message)
		}

		return tag, nil
	default:
		return tag, errors.New("unsupported resource type")
	}
}

func (t *GCPTagHandler) waitForOperation(o *compute.Operation) error {
	cnt := 10
	projectID := t.Credential.ProjectID
	zone := t.Region.Zone
	for cnt < 0 {
		if strings.ToUpper(o.Status) == "DONE" {
			if o.Error != nil {
				return fmt.Errorf("operation failed: %v", o.Error.Errors)
			}
			return nil
		}

		time.Sleep(2 * time.Second)
		op, err := t.ComputeClient.ZoneOperations.Get(projectID, zone, o.Name).Do()
		if err != nil {
			return fmt.Errorf("failed to get operation status: %v", err)
		}
		cnt--
		o = op
	}

	return errors.New("operation has not been finished.")
}

func (t *GCPTagHandler) ListTag(resType irs.RSType, resIID irs.IID) ([]irs.KeyValue, error) {
	err := validateSupportRS(resType)
	res := []irs.KeyValue{}
	if err != nil {
		return res, err
	}

	projectID := t.Credential.ProjectID
	zone := t.Region.Zone
	switch resType {
	case irs.VM:
		vm, err := t.ComputeClient.Instances.Get(projectID, zone, resIID.SystemId).Do()
		if err != nil {
			return res, err
		}
		for k, v := range vm.Labels {
			kv := irs.KeyValue{
				Key:   k,
				Value: v,
			}
			res = append(res, kv)
		}
		return res, nil
	case irs.DISK:
		disk, err := GetDiskInfo(t.ComputeClient, t.Credential, t.Region, resIID.SystemId)
		if err != nil {
			return res, err
		}

		for k, v := range disk.Labels {
			kv := irs.KeyValue{
				Key:   k,
				Value: v,
			}
			res = append(res, kv)
		}
		return res, nil
	case irs.CLUSTER:
		parent := getParentClusterAtContainer(projectID, zone, resIID.SystemId)
		cluster, err := t.ContainerClient.Projects.Locations.Clusters.Get(parent).Do()
		if err != nil {
			return res, err
		}

		for k, v := range cluster.ResourceLabels {
			kv := irs.KeyValue{
				Key:   k,
				Value: v,
			}
			res = append(res, kv)
		}
		return res, nil
	default:
		return res, errors.New("unsupport resources type")
	}
}
func (t *GCPTagHandler) GetTag(resType irs.RSType, resIID irs.IID, key string) (irs.KeyValue, error) {
	labels, err := t.ListTag(resType, resIID)
	res := irs.KeyValue{}
	if err != nil {
		return res, err
	}

	for _, l := range labels {
		if l.Key == key {
			res.Key = l.Key
			res.Value = l.Value
			return res, nil
		}
	}

	return res, nil
}
func (t *GCPTagHandler) RemoveTag(resType irs.RSType, resIID irs.IID, key string) (bool, error) {
	err := validateSupportRS(resType)
	if err != nil {
		return false, err
	}

	projectId := t.Credential.ProjectID
	zone := t.Region.Zone
	switch resType {
	case irs.VM:
		vm, err := t.getVm(resIID)
		if err != nil {
			return false, err
		}

		existLabels := vm.Labels
		if existLabels == nil {
			return false, errors.New("key does not exist")
		}
		if _, ok := existLabels[key]; ok {
			delete(existLabels, key)
		} else {
			return false, errors.New("key does not exist")
		}

		req := &compute.InstancesSetLabelsRequest{
			LabelFingerprint: vm.LabelFingerprint,
			Labels:           existLabels,
		}

		op, err := t.ComputeClient.Instances.SetLabels(projectId, zone, resIID.SystemId, req).Do()

		if err != nil {
			return false, err
		}

		if op.Error != nil {
			return false, fmt.Errorf("operation failed: %v", op.Error.Errors)
		}

		return true, nil
	case irs.DISK:

		disk, err := t.getDisk(resIID)
		if err != nil {
			return false, err
		}

		existLabels := disk.Labels

		if existLabels == nil {
			return false, errors.New("key does not exist")
		}

		if _, ok := existLabels[key]; ok {
			delete(existLabels, key)
		} else {
			return false, errors.New("key does not exist")
		}
		req := &compute.ZoneSetLabelsRequest{
			LabelFingerprint: disk.LabelFingerprint,
			Labels:           existLabels,
		}

		op, err := t.ComputeClient.Disks.SetLabels(projectId, zone, resIID.SystemId, req).Do()

		if err != nil {
			return false, err
		}

		if op.Error != nil {
			return false, fmt.Errorf("operation failed: %v", op.Error.Errors)
		}

		return true, nil
	case irs.CLUSTER:
		cluster, err := t.getCluster(resIID)
		if err != nil {
			return false, err
		}

		existLabels := cluster.ResourceLabels
		if existLabels == nil {
			return false, errors.New("key does not exist")
		}

		if _, ok := existLabels[key]; ok {
			delete(existLabels, key)
		} else {
			return false, errors.New("key does not exist")
		}

		name := getParentClusterAtContainer(projectId, zone, resIID.SystemId)
		req := &container.SetLabelsRequest{
			ClusterId:        resIID.SystemId,
			LabelFingerprint: cluster.LabelFingerprint,
			Name:             name,
			ProjectId:        projectId,
			Zone:             zone,
			ResourceLabels:   existLabels,
		}
		op, err := t.ContainerClient.Projects.Locations.Clusters.SetResourceLabels(name, req).Do()

		if err != nil {
			return false, err
		}

		if op.Error != nil {
			return false, fmt.Errorf("operation failed: %v", op.Error.Message)
		}

		return true, nil
	default:
		return false, errors.New("unsupported resource type")
	}
}
func (t *GCPTagHandler) getVms() ([]*compute.Instance, error) {
	vms, err := t.ComputeClient.Instances.List(t.Credential.ProjectID, t.Region.Zone).Do()
	if err != nil {
		return nil, err
	}

	return vms.Items, nil
}

func (t *GCPTagHandler) getDisks() ([]*compute.Disk, error) {
	disks, err := t.ComputeClient.Disks.List(t.Credential.ProjectID, t.Region.Zone).Do()
	if err != nil {
		return nil, err
	}
	return disks.Items, nil
}

func (t *GCPTagHandler) getClusters() ([]*container.Cluster, error) {
	parent := getParentAtContainer(t.Credential.ProjectID, t.Region.Zone)
	clusters, err := t.ContainerClient.Projects.Locations.Clusters.List(parent).Do()
	if err != nil {
		return nil, err
	}

	return clusters.Clusters, nil
}

func (t *GCPTagHandler) FindTag(resType irs.RSType, keyword string) ([]*irs.TagInfo, error) {
	var res []*irs.TagInfo
	var err error

	if resType == irs.ALL {
		res1, err := getResult(
			keyword,
			irs.VM,
			t.getVms,
			func(item *compute.Instance) string {
				return item.Name
			},
			func(item *compute.Instance) map[string]string {
				return item.Labels
			},
		)

		if err != nil {
			return res, err
		}

		res = append(res, res1...)
		res2, err := getResult(
			keyword,
			irs.DISK,
			t.getDisks,
			func(item *compute.Disk) string {
				return item.Name
			},
			func(item *compute.Disk) map[string]string {
				return item.Labels
			},
		)

		if err != nil {
			return res, err
		}
		res = append(res, res2...)
		res3, err := getResult(
			keyword,
			irs.CLUSTER,
			t.getClusters,
			func(item *container.Cluster) string {
				return item.Name
			},
			func(item *container.Cluster) map[string]string {
				return item.ResourceLabels
			},
		)

		if err != nil {
			return res, err
		}
		res = append(res, res3...)
	} else {
		err = validateSupportRS(resType)
		if err != nil {
			return res, err
		}

		switch resType {
		case irs.VM:
			res, err = getResult(
				keyword,
				resType,
				t.getVms,
				func(item *compute.Instance) string {
					return item.Name
				},
				func(item *compute.Instance) map[string]string {
					return item.Labels
				},
			)

			if err != nil {
				return res, err
			}
		case irs.DISK:
			res, err = getResult(
				keyword,
				resType,
				t.getDisks,
				func(item *compute.Disk) string {
					return item.Name
				},
				func(item *compute.Disk) map[string]string {
					return item.Labels
				},
			)

			if err != nil {
				return res, err
			}
		case irs.CLUSTER:
			res, err = getResult(
				keyword,
				resType,
				t.getClusters,
				func(item *container.Cluster) string {
					return item.Name
				},
				func(item *container.Cluster) map[string]string {
					return item.ResourceLabels
				},
			)

			if err != nil {
				return res, err
			}

		default:
			return nil, errors.New("unsupport resource type")
		}
	}

	return res, nil
}

func getResult[T *compute.Instance | *compute.Disk | *container.Cluster](
	keyword string,
	resType irs.RSType,
	resultFn func() ([]T, error),
	getNameFn func(item T) string,
	getLabelFn func(item T) map[string]string,
) ([]*irs.TagInfo, error) {
	var res []*irs.TagInfo
	items, err := resultFn()
	if err != nil {
		return res, err
	}

	for _, item := range items {
		name := getNameFn(item)
		var ti *irs.TagInfo
		for k, v := range getLabelFn(item) {
			if strings.Contains(k, keyword) || strings.Contains(v, keyword) {
				if ti == nil {
					ti = &irs.TagInfo{
						ResType: resType,
						ResIId: irs.IID{
							NameId:   name,
							SystemId: name,
						},
					}
				}
				ti.TagList = append(ti.TagList, irs.KeyValue{Key: k, Value: v})
			}
		}
		if ti != nil {
			res = append(res, ti)
		}
	}

	return res, nil
}
