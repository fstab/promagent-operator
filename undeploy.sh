#!/bin/bash

kubectl delete -f deploy/operator.yaml
# kubectl delete -f deploy/crds/promagent_v1alpha1_promagent_cr.yaml
kubectl delete -f deploy/crds/promagent_v1alpha1_promagent_crd.yaml
kubectl delete -f deploy/role.yaml
kubectl delete -f deploy/role_binding.yaml
kubectl delete -f deploy/service_account.yaml
