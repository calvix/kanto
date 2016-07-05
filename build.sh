#!/bin/bash
echo "Building . . ."
#export GOPATH=/home/calvix/ext/kanto-gopath
export GOPATH=/home/calvix/kanto-gopath
go build 
RET=$?
echo "Done"

exit $RET
