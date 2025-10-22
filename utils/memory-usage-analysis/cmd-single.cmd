# $ ./start.sh cmd-single.cmd
#curl -sX GET http://localhost:1024/spider/vmimage?ConnectionName=ncp-korea1-config > curl.spider.vmimage.ncp-korea1-config.json
curl -sX POST http://localhost:1024/spider/priceinfo/vm/KR?ConnectionName=ncp-korea1-config > curl.spider.priceinfo.ncp-korea1-config.json
