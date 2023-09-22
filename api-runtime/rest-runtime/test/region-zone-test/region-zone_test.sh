echo "#### Region-Zone Test Process - Start ###"

sleep 1 
echo -e "\n### Region/Zone : List"
curl -sX GET http://localhost:1024/spider/regionzone -H 'Content-Type: application/json' -d '{"ConnectionName": "'${CONN_CONFIG}'"}' | json_pp

sleep 2 
echo -e "\n### Region/Zone : Get"
curl -sX GET http://localhost:1024/spider/regionzone/KR -H 'Content-Type: application/json' -d '{"ConnectionName": "'${CONN_CONFIG}'"}'

sleep 2 
echo -e "\n### ORG Region : List"
curl -sX GET http://localhost:1024/spider/orgregion -H 'Content-Type: application/json' -d '{"ConnectionName": "'${CONN_CONFIG}'"}' | json_pp 

sleep 2
echo -e "\n### ORG Zone : List"
curl -sX GET http://localhost:1024/spider/orgzone -H 'Content-Type: application/json' -d '{"ConnectionName": "'${CONN_CONFIG}'"}' | json_pp 

echo "#### Region-Zone Test Process- Finished ###"