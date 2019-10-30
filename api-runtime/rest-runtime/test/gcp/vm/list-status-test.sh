RESTSERVER=localhost

#정상 동작

#[참고]
# GCP 같은경우 zone을 받기 때문에 해당 지역의 vm이 조회되는게 아니라 해당 region 의  zone에 해당하는 vm만 조회가 된다. 따라서 어떻게 처리 해야 할지 고민을 해 봐야 할 문제
curl -X GET http://$RESTSERVER:1024/vmstatus?connection_name=gcp-config01 |json_pp
