# export CONN_CONFIG=ncpvpc-korea1-config
# export CONN_CONFIG=ncpvpc-korea2-config
export CONN_CONFIG=ncpvpc-singapore4-config
# export CONN_CONFIG=ncpvpc-singapore5-config
# export CONN_CONFIG=ncpvpc-japan4-config
# export CONN_CONFIG=ncpvpc-japan5-config

# Ubuntu 18.04 image(for Korea1)
export IMAGE_NAME=SW.VSVR.OS.LNX64.UBNTU.SVR1804.B050
# ncpvpc-singapore4-config도 지원
# ncpvpc-japan4-config도 지원

# vCPU: 4, Mem: 8192MB
export SPEC_NAME=SVR.VSVR.HICPU.C004.M008.NET.SSD.B050.G002
# ncpvpc-singapore4-config도 지원
# ncpvpc-japan4-config도 지원

./ncpvpc-full_test.sh
