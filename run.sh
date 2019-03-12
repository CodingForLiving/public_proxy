#!/bin/bash

export GOPATH=`pwd`

go install server

go install client

nohup ./bin/server &

nohup ./bin/client &
