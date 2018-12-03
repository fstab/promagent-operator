package promagent

import (
	"context"
	"fmt"
	promagentv1alpha1 "github.com/fstab/promagent-operator/pkg/apis/promagent/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	promagentState                      = "promagent-state"
	promagentStateNoJavaProcess         = "no-java-process"
	promagentStateInstrumetned          = "instrumented"
	promagentStateInstrumentationFailed = "failed"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Promagent Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcilePromagent{
		client:     mgr.GetClient(),
		execClient: &ExecClient{config: mgr.GetConfig()},
		scheme:     mgr.GetScheme(),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("promagent-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Promagent
	err = c.Watch(&source.Kind{Type: &promagentv1alpha1.Promagent{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Promagent
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &promagentv1alpha1.Promagent{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestsFromMapFunc{
		ToRequests: &MyMapper{},
	})
	return nil
}

type MyMapper struct{}

func (m *MyMapper) Map(o handler.MapObject) []reconcile.Request {
	result := make([]reconcile.Request, 0, 1)
	if pod, ok := o.Object.(*corev1.Pod); ok {
		if pod.Status.Phase == corev1.PodRunning {
			result = append(result, reconcile.Request{NamespacedName: types.NamespacedName{
				Namespace: pod.Namespace,
				Name:      pod.Name,
			}})
		}
	}
	return result
}

var _ reconcile.Reconciler = &ReconcilePromagent{}

// ReconcilePromagent reconciles a Promagent object
type ReconcilePromagent struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client     client.Client
	execClient *ExecClient
	scheme     *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Promagent object and makes changes based on the state read
// and what is in the Promagent.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcilePromagent) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := logf.Log.WithName("promagent").WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Promagent")

	pod := &corev1.Pod{}
	err := r.client.Get(context.TODO(), request.NamespacedName, pod)
	if err != nil {
		reqLogger.Error(err, fmt.Sprintf("pod %v not found", request.NamespacedName))
		return reconcile.Result{}, nil
	}
	return r.reconcilePod(pod, reqLogger.WithValues("pod", request.NamespacedName))
}

func (r *ReconcilePromagent) reconcilePod(pod *corev1.Pod, log logr.Logger) (reconcile.Result, error) {
	var (
		err           error
		state         = promagentStateInstrumentationFailed
		javaProcesses javaProcesses
	)
	if label, ok := pod.Labels[promagentState]; ok {
		log.Info(fmt.Sprintf("ignoring pod, because label %v=%v is already present", promagentState, label))
		return reconcile.Result{}, nil
	}

	// Update the promagent_state label of the pod.
	// The deferred function captures the state variable, i.e. state is evaluated when we return.
	defer func() {
		pod.Labels[promagentState] = state
		err = r.client.Update(context.TODO(), pod)
		if err != nil {
			log.Error(err, fmt.Sprintf("failed to update label %v=%v", promagentState, state))
		}
	}()

	if len(pod.Spec.Containers) == 0 {
		log.Error(nil, "no containers found for pod")
		return reconcile.Result{}, nil
	}

	container := pod.Spec.Containers[0]
	if len(pod.Spec.Containers) > 1 {
		// prometheus doesn't support scraping multiple containers running on the same pod
		log.Info(fmt.Sprintf("%v containers found for pod. instrumenting the first container, which is %v.", len(pod.Spec.Containers), container.Name))
	}

	javaProcesses, err = r.queryJavaProcesses(pod, &container, log)
	if err != nil {
		log.Error(err, "calling jps failed")
		return reconcile.Result{}, nil
	}

	if len(javaProcesses) == 0 {
		log.Info("skipping pod, because no Java process found")
		state = promagentStateNoJavaProcess
		return reconcile.Result{}, nil
	}

	if len(javaProcesses) > 1 {
		// agent can run only once per container, because we cannot expose port 9300 multiple times
		log.Info(fmt.Sprintf("more than one Java processes found, instrumenting %v, ignoring %v", javaProcesses[0], javaProcesses[1:]))
	}

	err = r.copyAgentToPod(pod, &container, log)
	if err != nil {
		log.Error(err, "failed to copy agent to prod")
		return reconcile.Result{}, nil
	}

	err = r.exposeContainerPort(9300, pod, &container)
	if err != nil {
		log.Error(err, "failed to expose container port 9300")
		return reconcile.Result{}, nil
	}

	log.Info(fmt.Sprintf("attaching agent to java process %v", javaProcesses[0]))
	err = r.attachAgent(javaProcesses[0].pid, pod, &container, log)
	if err != nil {
		log.Error(err, fmt.Sprintf("failed to attach agent to java process %v", javaProcesses[0]))
		return reconcile.Result{}, nil
	}
	state = promagentStateInstrumetned
	return reconcile.Result{}, nil
}

func (r *ReconcilePromagent) copyAgentToPod(pod *corev1.Pod, container *corev1.Container, log logr.InfoLogger) error {
	var err error
	log.Info("copying promagent.jar to pod")
	err = copyToPod("/promagent.jar", "/tmp/promagent.jar", pod, container, r.execClient)
	if err != nil {
		return fmt.Errorf("failed to copy promagent.jar to pod: %v", err)
	}
	log.Info("copying promagent-loader-1.0-SNAPSHOT.jar to pod")
	err = copyToPod("/promagent-loader-1.0-SNAPSHOT.jar", "/tmp/promagent-loader-1.0-SNAPSHOT.jar", pod, container, r.execClient)
	if err != nil {
		return fmt.Errorf("failed to copy promagent-loader-1.0-SNAPSHOT.jar to pod: %v", err)
	}
	return nil
}

func (r *ReconcilePromagent) exposeContainerPort(portNumber int32, pod *corev1.Pod, container *corev1.Container) error {
	for _, port := range container.Ports {
		if port.ContainerPort == portNumber {
			return fmt.Errorf("port number %d is already in use", portNumber)
		}
	}
	container.Ports = append(container.Ports, corev1.ContainerPort{
		ContainerPort: portNumber,
		Name:          "prometheusJmxMetrics",
		Protocol:      corev1.ProtocolTCP,
	})
	return r.client.Update(context.TODO(), pod)
}

func parseJpsOutput(line string) (javaProcess, error) {
	var result javaProcess
	_, err := fmt.Sscanf(line, "%d %s", &result.pid, &result.name)
	return result, err
}

func (r *ReconcilePromagent) attachAgent(pid int, pod *corev1.Pod, container *corev1.Container, log logr.InfoLogger) error {
	var (
		command = fmt.Sprintf("java -cp \"$JAVA_HOME/lib/tools.jar:/tmp/promagent-loader-1.0-SNAPSHOT.jar\" io.promagent.loader.PromagentLoader -agent /tmp/promagent.jar -port 9300 -pid %d", pid)
		err     error
	)
	log.Info(fmt.Sprintf("executing command %q", command))
	_, err = r.execClient.Exec(pod, container, nil, command)
	return err
}
