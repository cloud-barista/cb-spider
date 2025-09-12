# Fetch Image Info List
curl -sX GET http://localhost:1024/spider/vmimage?ConnectionName=aws-config01 > curl.spider.vmimage.aws-config01.json
curl -sX GET http://localhost:1024/spider/vmimage?ConnectionName=azure-northeu-config > curl.spider.vmimage.azure-northeu-config.json
curl -sX GET http://localhost:1024/spider/vmimage?ConnectionName=gcp-iowa-config > curl.spider.vmimage.gcp-iowa-config.json
curl -sX GET http://localhost:1024/spider/vmimage?ConnectionName=alibaba-tokyo-config > curl.spider.vmimage.alibaba-tokyo-config.json
curl -sX GET http://localhost:1024/spider/vmimage?ConnectionName=tencent-guangzhou3-config > curl.spider.vmimage.tencent-guangzhou3-config.json
curl -sX GET http://localhost:1024/spider/vmimage?ConnectionName=ibm-us-south-1-config > curl.spider.vmimage.ibm-us-south-1-config.json
curl -sX GET http://localhost:1024/spider/vmimage?ConnectionName=ncp-korea1-config > curl.spider.vmimage.ncp-korea1-config.json
curl -sX GET http://localhost:1024/spider/vmimage?ConnectionName=nhn-korea-pangyo1-config > curl.spider.vmimage.nhn-korea-pangyo1-config.json
curl -sX GET http://localhost:1024/spider/vmimage?ConnectionName=kt-mokdong1-config > curl.spider.vmimage.kt-mokdong1-config.json
curl -sX GET http://localhost:1024/spider/vmimage?ConnectionName=ktclassic-korea-seoul1-config > curl.spider.vmimage.ktclassic-korea-seoul1-config.json

# Fetch Price Info List
curl -sX POST http://localhost:1024/spider/priceinfo/vm/ap-northeast-2?ConnectionName=aws-config01 > curl.spider.priceinfo.aws-config01.json
curl -sX POST http://localhost:1024/spider/priceinfo/vm/northeurope?ConnectionName=azure-northeu-config > curl.spider.priceinfo.azure-northeu-config.json
curl -sX POST http://localhost:1024/spider/priceinfo/vm/us-central1?ConnectionName=gcp-iowa-config > curl.spider.priceinfo.gcp-iowa-config.json
curl -sX POST http://localhost:1024/spider/priceinfo/vm/ap-northeast-1?ConnectionName=alibaba-tokyo-config > curl.spider.priceinfo.alibaba-tokyo-config.json
curl -sX POST http://localhost:1024/spider/priceinfo/vm/ap-guangzhou?ConnectionName=tencent-guangzhou3-config > curl.spider.priceinfo.tencent-guangzhou3-config.json
curl -sX POST http://localhost:1024/spider/priceinfo/vm/us-south?ConnectionName=ibm-us-south-1-config > curl.spider.priceinfo.ibm-us-south-1-config.json
curl -sX POST http://localhost:1024/spider/priceinfo/vm/KR?ConnectionName=ncp-korea1-config > curl.spider.priceinfo.ncp-korea1-config.json