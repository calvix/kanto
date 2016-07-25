#!/bin/bash
# kanto API
if [ "$1" -ne "" ];
then
        export kanto_url="$1"
else
        export kanto_url="127.0.0.1:80"
fi

echo "<<== working with kanto api on: $kanto_url"
echo


# test commands
echo "<<== Creating couchdb cluster \"mycluster-1\" with 4 replicas"
#create cluster
result=`curl $kanto_url/v0/create -d "username=user1&token=43ggDWgv4&cluster_tag=mycluster-1&replicas=4" 2>/dev/null`
endpoint=`echo $result | cut -d\" -f28`
echo "Cluster endpoint: $endpoint"

#add test data to cluster
echo
echo "<<= Save test data to cluster:  (test data: {\"test1\":\"replicas=4\"})"
curl --user user1:43ggDWgv4 -X PUT -d '{"test1":"replicas=4"}' "$endpoint"/test/doc1
echo
echo "<<=Check saved data (3 times)"
curl --user user1:43ggDWgv4 "$endpoint"/test/doc1
sleep 1
curl --user user1:43ggDWgv4 "$endpoint"/test/doc1
sleep 1
curl --user user1:43ggDWgv4 "$endpoint"/test/doc1

#scale cluster
echo
echo "<<== Scale cluster to 8 replicas"
curl $kanto_url/v0/scale -d "username=user1&token=43ggDWgv4&cluster_tag=mycluster-1&replicas=8"
echo
# check replicated data after replication
echo
echo "<== Check if data replicate (5 checks)"
# check data 4 times
curl --user user1:43ggDWgv4 "$endpoint"/test/doc1
sleep 1s
curl --user user1:43ggDWgv4 "$endpoint"/test/doc1
sleep 1s
curl --user user1:43ggDWgv4 "$endpoint"/test/doc1
sleep 1s
curl --user user1:43ggDWgv4 "$endpoint"/test/doc1
sleep 1s
curl --user user1:43ggDWgv4 "$endpoint"/test/doc1
sleep 1s

# save revison for couchdb update
rev=`curl --user user1:43ggDWgv4 "$endpoint"/test/doc1 2>/dev/null| cut -d\" -f8`

# update test/doc1 with new data
echo
echo "<<== Update test data to: {\"test1\":\"replicas=8\"}"
curl --user user1:43ggDWgv4 -X PUT -d '{"test1":"replicas=8"}' "$endpoint"/test/doc1?rev=$rev
echo
echo "<<== Check updated data (8 times)"
# check data 8 times
curl --user user1:43ggDWgv4 "$endpoint"/test/doc1
sleep 1s
curl --user user1:43ggDWgv4 "$endpoint"/test/doc1
sleep 1s
curl --user user1:43ggDWgv4 "$endpoint"/test/doc1
sleep 1s
curl --user user1:43ggDWgv4 "$endpoint"/test/doc1
sleep 1s
curl --user user1:43ggDWgv4 "$endpoint"/test/doc1
sleep 1s
curl --user user1:43ggDWgv4 "$endpoint"/test/doc1
sleep 1s
curl --user user1:43ggDWgv4 "$endpoint"/test/doc1
sleep 1s
curl --user user1:43ggDWgv4 "$endpoint"/test/doc1
sleep 1s
curl --user user1:43ggDWgv4 "$endpoint"/test/doc1

#scale down
echo 
echo "<<== Scale cluster down to 2 replicas"
curl $kanto_url/v0/scale -d "username=user1&token=43ggDWgv4&cluster_tag=mycluster-1&replicas=2"

echo
echo "<<== Check data"
curl --user user1:43ggDWgv4 "$endpoint"/test/doc1
sleep 1s
curl --user user1:43ggDWgv4 "$endpoint"/test/doc1

# replicate custom db
echo
echo  "<<== Replicate custom databases: \"cologne and brno\""
curl $kanto_url/v0/replicate -d "username=user1&token=43ggDWgv4&cluster_tag=mycluster-1&databases=cologne,brno"

# save data to cust database
echo
echo "<<= Save test data to cluster:  (test data: {\"test2\":\"city=cologne\"})"
curl --user user1:43ggDWgv4 -X PUT -d '{"test1":"replicas=4"}' "$endpoint"/cologne/doc1

echo
echo "<<== Check saved data (4 times)"
# check
curl --user user1:43ggDWgv4 "$endpoint"/cologne/doc1
sleep 1s
curl --user user1:43ggDWgv4 "$endpoint"/cologne/doc1
sleep 1s
curl --user user1:43ggDWgv4 "$endpoint"/cologne/doc1
sleep 1s
curl --user user1:43ggDWgv4 "$endpoint"/cologne/doc1

# delete cluster
echo
echo "<<== Deleting couchdb cluster"
curl $kanto_url/v0/delete -d "username=user1&token=43ggDWgv4&cluster_tag=mycluster-1" 
echo
echo
echo "<<== Done"
echo

