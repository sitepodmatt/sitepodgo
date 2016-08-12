#!/bin/bash

SERVER=http://127.0.0.1:9080

~/binaries/kubernetes/kubectl -s=$SERVER create -f cluster.yaml
~/binaries/kubernetes/kubectl -s=$SERVER create -f sitepod.yaml
~/binaries/kubernetes/kubectl -s=$SERVER create -f appcomponent.yaml
~/binaries/kubernetes/kubectl -s=$SERVER create -f systemuser.yaml
~/binaries/kubernetes/kubectl -s=$SERVER create -f podtask.yaml

