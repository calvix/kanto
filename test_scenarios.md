# base test scenarios

test1:
 * create couchdb cluster (tag=mycluster-1, username=user1, token=43ggDWgv4, replicas=2 )
 * get data from couchdb endpoint
 * delete couchdb cluster


test5:
 * create couchdb cluster (tag=mycluster-1, username=user1, token=43ggDWgv4, replicas=4 )
 * put data to database "test" doc "doc1" value {"test1":"replicas=4"} 
 * scale to 8 replicas
 * get data from test/doc1
 * update data in test/doc1 to {"test1":"replicas=8"}
 * scale to 2 replicas
 * set replication