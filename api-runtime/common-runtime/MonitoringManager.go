// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.09.

package commonruntime

import (
	"fmt"
	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	infostore "github.com/cloud-barista/cb-spider/info-store"
	"strconv"
)

//================ Monitoring Handler

func GetVMMetricData(connectionName string, nameID string, metricType cres.MetricType, periodMinute string, timeBeforeHour string) (*cres.MetricData, error) {
	cblog.Info("call GetVMMetricData()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	nameID, err = EmptyCheckAndTrim("nameID", nameID)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	vmHandler, err := cldConn.CreateVMHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	vmSPLock.RLock(connectionName, nameID)
	defer vmSPLock.RUnlock(connectionName, nameID)

	// (1) get IID(NameId)
	var iidInfo VMIIDInfo
	err = infostore.GetByConditions(&iidInfo, CONNECTION_NAME_COLUMN, connectionName, NAME_ID_COLUMN, nameID)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (2) get resource(SystemId)
	vm, err := vmHandler.GetVM(getDriverIID(cres.IID{NameId: iidInfo.NameId, SystemId: iidInfo.SystemId}))
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	monitoringHandler, err := cldConn.CreateMonitoringHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (4) get monitoring info
	info, err := monitoringHandler.GetVMMetricData(cres.VMMonitoringReqInfo{
		VMIID:          vm.IId,
		MetricType:     metricType,
		IntervalMinute: periodMinute,
		TimeBeforeHour: timeBeforeHour,
	})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	return &info, nil
}

func GetClusterNodeMetricData(connectionName string, clusterNameID string, nodeGroupNameID string, nodeNumber string, metricType cres.MetricType, periodMinute string, timeBeforeHour string) (*cres.MetricData, error) {
	cblog.Info("call GetClusterNodeMetricData()")

	// check empty and trim user inputs
	connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	clusterNameID, err = EmptyCheckAndTrim("clusterNameID", clusterNameID)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	nodeGroupNameID, err = EmptyCheckAndTrim("nodeGroupNameID", nodeGroupNameID)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	nodeNumber, err = EmptyCheckAndTrim("nodeNumber", nodeNumber)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	nodeNumberInt, err := strconv.Atoi(nodeNumber)
	if err != nil || nodeNumberInt <= 0 {
		errMsg := "Invalid node number " + nodeNumber
		cblog.Error(errMsg)
		return nil, fmt.Errorf(errMsg)
	}

	cldConn, err := ccm.GetCloudConnection(connectionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	clusterSPLock.RLock(connectionName, clusterNameID)
	defer clusterSPLock.RUnlock(connectionName, clusterNameID)

	cluserDriverIID, nodeGroupDriverIID, err := getClusterDriverIIDNodeGroupDriverIID(connectionName, clusterNameID, nodeGroupNameID)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	clusterHandler, err := cldConn.CreateClusterHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	clusterInfo, err := clusterHandler.GetCluster(cluserDriverIID)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	var nodeGroupExist bool
	var nodeNameId string

	for _, nodeGroup := range clusterInfo.NodeGroupList {
		if nodeGroup.IId.NameId == nodeGroupDriverIID.NameId {
			if nodeNumberInt > len(nodeGroup.Nodes) {
				errMsg := fmt.Sprintf("Node number %s is greater than the number of nodes (%d).", nodeNumber, len(nodeGroup.Nodes))
				cblog.Error(errMsg)
				return nil, fmt.Errorf(errMsg)
			}

			nodeNameId = nodeGroup.Nodes[nodeNumberInt-1].NameId

			nodeGroupExist = true
			break
		}
	}

	if !nodeGroupExist {
		errMsg := fmt.Sprintf("node group %s not exist", nodeGroupDriverIID.NameId)
		cblog.Error(errMsg)
		return nil, fmt.Errorf(errMsg)
	}

	monitoringHandler, err := cldConn.CreateMonitoringHandler()
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// (4) get monitoring info
	info, err := monitoringHandler.GetClusterNodeMetricData(cres.ClusterNodeMonitoringReqInfo{
		ClusterIID:     cluserDriverIID,
		NodeGroupID:    nodeGroupDriverIID,
		NodeIID:        cres.IID{NameId: nodeNameId},
		MetricType:     metricType,
		IntervalMinute: periodMinute,
		TimeBeforeHour: timeBeforeHour,
	})
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	return &info, nil
}
