# base test scenarios

test1:
 * create couchdb cluster (tag=mycluster-1, username=user1, token=43ggDWgv4, replicas=2 )
 * get data from couchdb endpoint
 * delete couchdb cluster

```bash
export kanto_url="127.0.0.1:80"
# create cluster
result=`curl $kanto_url/v0/create -d "username=user1&token=43ggDWgv4&cluster_tag=mycluster-1&replicas=2"`
endpoint=`echo $result | cut -d\" -f28`
curl $endpoint
curl $kanto_url/v0/delete -d "username=user1&token=43ggDWgv4&cluster_tag=mycluster-1"

```




test5:
 * create couchdb cluster (tag=mycluster-1, username=user1, token=43ggDWgv4, replicas=4 )
 * put data to database "test" doc "doc1" value {"test1":"replicas=4"} 
 * scale to 8 replicas
 * get data from test/doc1
 * update data in test/doc1 to {"test1":"replicas=8"}
 * check data
 * delete cluster


```
echo ""
result=`curl $kanto_url/v0/create -d "username=user1&token=43ggDWgv4&cluster_tag=mycluster-1&replicas=4"`
endpoint=`echo $result | cut -d\" -f28`
curl --user user1:43ggDWgv4 -X PUT -d '{"test1":"replicas=4"}' "$endpoint"/test/doc1
curl $kanto_url/v0/scale -d "username=user1&token=43ggDWgv4&cluster_tag=mycluster-1&replicas=8"
echo "wait until replication start work (3s wait)"
sleep 3s
# check data 4 times
curl --user user1:43ggDWgv4 "$endpoint"/test/doc1
sleep 1s
curl --user user1:43ggDWgv4 "$endpoint"/test/doc1
sleep 1s
curl --user user1:43ggDWgv4 "$endpoint"/test/doc1
sleep 1s
curl --user user1:43ggDWgv4 "$endpoint"/test/doc1
sleep 1s
# save revison for couchdb update
rev=`curl --user user1:43ggDWgv4 "$endpoint"/test/doc1 | cut -d\" -f8`
# update test/doc1 with new data
curl --user user1:43ggDWgv4 -X PUT -d '{"test1":"replicas=8"}' "$endpoint"/test/doc1?rev=$rev
# check data 4 times
curl --user user1:43ggDWgv4 "$endpoint"/test/doc1
sleep 1s
curl --user user1:43ggDWgv4 "$endpoint"/test/doc1
sleep 1s
curl --user user1:43ggDWgv4 "$endpoint"/test/doc1
sleep 1s
curl --user user1:43ggDWgv4 "$endpoint"/test/doc1
# delete cluster
curl $kanto_url/v0/delete -d "username=user1&token=43ggDWgv4&cluster_tag=mycluster-1"

```

