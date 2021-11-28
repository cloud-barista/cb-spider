#!/bin/sh

CLIPATH=$CBSPIDER_ROOT/interface
ofile=$CBSPIDER_ROOT/utils/import-drv-region-info/export-region-list/exported-regions-list.json

$CLIPATH/spctl region list --config $CLIPATH/grpc_conf.yaml |grep RegionName | sed 's/- RegionName: //g' > $ofile

