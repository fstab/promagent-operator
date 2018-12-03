#!/bin/bash

kubectl create -f deploy/service_account.yaml
kubectl create -f deploy/role.yaml
kubectl create -f deploy/role_binding.yaml
kubectl create -f deploy/crds/promagent_v1alpha1_promagent_crd.yaml
kubectl create -f deploy/operator.yaml
