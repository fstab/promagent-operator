Promagent Operator
==================

This is a demo [operator](https://coreos.com/operators/) showing how to instrument [Java](http://java.sun.com) applications in [Kubernetes](https://kubernetes.io) with [Prometheus](https://prometheus.io) metrics.

What does it do?
----------------

The `promagent-operator` detects when a pod is started in a Kubernetes cluster and reacts as follows:

1. Check if the pod runs a Java process (trying to execute `jps` in the pod). If there is no Java process, the pod is ignored.
2. Copy `promagent.jar` and `promagent-loader-1.0-SNAPSHOT.jar` into the `/tmp/` directory of the pod. See demo code on [promagent.io](http://promagent.io).
3. Expose port `9300` on the pod.
4. Attach `promagent.jar` to the running Java process, as described on [github.com/fstab/promagent/tree/master/promagent-framework/promagent-loader](https://github.com/fstab/promagent/tree/master/promagent-framework/promagent-loader).
5. Add the label `promagent-state: instrumented` to the pod, which will make the pod part of the `promagent` endpoint.


How to install
--------------

There is a pre-built image on [hub.docker.com/r/fstab/promagent-operator](https://hub.docker.com/r/fstab/promagent-operator/), which can be installed as follows:

```
export DEPLOY_DIR=https://raw.githubusercontent.com/fstab/promagent-operator/master/deploy/

kubectl create -f $DEPLOY_DIR/service_account.yaml
kubectl create -f $DEPLOY_DIR/role.yaml
kubectl create -f $DEPLOY_DIR/role_binding.yaml
kubectl create -f $DEPLOY_DIR/crds/promagent_v1alpha1_promagent_crd.yaml
kubectl create -f $DEPLOY_DIR/operator.yaml
kubectl create -f $DEPLOY_DIR/service.yaml
```

You can build the Docker image from source with the [operator-sdk](https://github.com/operator-framework/operator-sdk) (tested with [v0.2.1](https://github.com/operator-framework/operator-sdk/release://github.com/operator-framework/operator-sdk/releases)):

```
mkdir -p $GOROOT/src/github.com/fstab
cd github.com/fstab
git clone https://github.com/fstab/promagent-operator
cd promagent-operator
operator-sdk build fstab/promagent-operator
```

How to test
-----------

The [java-demo](https://github.com/fstab/java-demo) is a simple HTTP service that dumps the pod's enviornment variables as an ASCII table. We use the java-demo for testing the promagent-operator.

Install the Java demo from [github.com/fstab/java-demo](https://github.com/fstab/java-demo)

```
curl -LO https://raw.githubusercontent.com/fstab/java-demo/master/java-demo.yaml
kubectl create -f java-demo.yaml
```

Test if the java-demo works:

```
export DEMO_SERVICE_IP=$(kubectl get service java-demo -o=jsonpath='{.spec.clusterIP}')
curl $DEMO_SERVICE_IP
```

View the endpoints. If the java demo was instrumented successfully, you should not only see the java-demo endpoints with port 8080, but also promagent endpoints with port 9300.

```
kubectl get endpoints
```

Metrics are exposed on path `/metrics`:

```
curl <pod-ip>:9300/metrics
```

Prometheus service discovery uses the Kubernetes endpoints to find the pods to be scraped. Depending on how Prometheus was installed, there are different options how to configure this. If Prometheus was installed using CoreOS's [prometheus-operator](https://github.com/coreos/prometheus-operator/), a `ServiceMonitor` custom resource is used to configure the new endpoints:

```
export DEPLOY_DIR=https://raw.githubusercontent.com/fstab/promagent-operator/master/deploy/

kubectl create -f $DEPLOY_DIR/service_monitor.yaml
```

Troubleshooting
---------------

The operator runs as a pod `promagent-operator-*`. View the pod's logs for error messages with `kubectl logs <pod-name>`.

Status
------

This is experimental demo code. It was tested with [github.com/fstab/java-demo](https://github.com/fstab/java-demo), but it should not be run in production.

The code is based on a scaffold generated with the [operator-sdk](https://github.com/operator-framework/operator-sdk). Some of the generated code (watch for custom resource) is not used yet and should be refactored. The main implementation is in the `Reconcile()` function in `pkg/controller/promagent/promagent_controller.go`.


Resources
---------

* Similar operator and great blog post: [https://banzaicloud.com/blog/prometheus-jmx-exporter-operator/](https://banzaicloud.com/blog/prometheus-jmx-exporter-operator/)
* List of awesome operators: [github.com/operator-framework/awesome-operators](https://github.com/operator-framework/awesome-operators)
