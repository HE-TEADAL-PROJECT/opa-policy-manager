#!/bin/bash

docker build -t localhost:5000/opa-policy-manager:latest .
docker push localhost:5000/opa-policy-manager:latest
kubectl rollout restart deployment/opa-policy-manager
