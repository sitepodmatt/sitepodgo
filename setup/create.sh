#!/bin/bash

SERVER=http://127.0.0.1:8080

~/binaries/kubernetes/kubectl -s=$SERVER create -f counter.yaml
~/binaries/kubernetes/kubectl -s=$SERVER create -f cluster.yaml
~/binaries/kubernetes/kubectl -s=$SERVER create -f service.yaml
~/binaries/kubernetes/kubectl -s=$SERVER create -f sitepod.yaml
~/binaries/kubernetes/kubectl -s=$SERVER create -f systemuser.yaml

