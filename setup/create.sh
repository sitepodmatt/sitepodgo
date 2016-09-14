#!/bin/bash

SERVER=http://127.0.0.1:9080

kubectl -s=http://localhost:9080 create -f cluster.yaml
kubectl -s=http://localhost:9080 create -f sitepod.yaml
kubectl -s=http://localhost:9080 create -f appcomponent.yaml
kubectl -s=http://localhost:9080 create -f systemuser.yaml
kubectl -s=http://localhost:9080 create -f podtask.yaml
kubectl -s=http://localhost:9080 create -f website.yaml

