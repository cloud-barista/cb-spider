// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2022.09.

package commonruntime

import (
	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

//================ AnyCall Handler
func AnyCall(connectionName string, reqInfo cres.AnyCallInfo) (*cres.AnyCallInfo, error) {
        cblog.Info("call AnyCall()")

        // check empty and trim user inputs
        connectionName, err := EmptyCheckAndTrim("connectionName", connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        cldConn, err := ccm.GetCloudConnection(connectionName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        handler, err := cldConn.CreateAnyCallHandler()
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        //  Call AnyCall
        info, err := handler.AnyCall(reqInfo)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        return &info, nil
}
