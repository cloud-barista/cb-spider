# Fetch VPC List
curl -sX GET http://localhost:1024/spider/vpc?ConnectionName=ncp-korea1-config > curl.spider.vpc.ncp-korea1-config.json

# Fetch VM List
curl -sX GET http://localhost:1024/spider/vm?ConnectionName=ncp-korea1-config > curl.spider.vm.ncp-korea1-config.json
curl -sX GET http://localhost:1024/spider/vm?ConnectionName=nhn-korea-pangyo1-config > curl.spider.vm.nhn-korea-pangyo1-config.json