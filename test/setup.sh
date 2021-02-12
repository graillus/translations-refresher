#!/usr/bin/env bash

kind create cluster --name test --config cluster.yaml

docker build -t translations-watcher:test -f ../Dockerfile ../

kind --name test load docker-image translations-watcher:test

helm upgrade --install translations-refresher ../deploy/helm --set tag=test

kubectl apply -f deployment.yaml
