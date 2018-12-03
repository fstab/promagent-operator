package promagent

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

type ExecClient struct {
	config *rest.Config
}

// execute the given command inside the specified container remotely
func (c *ExecClient) Exec(pod *corev1.Pod, container *corev1.Container, stdin io.Reader, command string) (string, error) {

	restClient := kubernetes.NewForConfigOrDie(c.config).CoreV1().RESTClient()
	request := restClient.Post()
	request.Resource("pods").Name(pod.Name).Namespace(pod.Namespace).SubResource("exec")
	request.VersionedParams(&corev1.PodExecOptions{
		Container: container.Name,
		Command:   []string{"sh", "-c", command},
		Stdout:    true,
		Stderr:    true,
		Stdin:     stdin != nil,
	}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(c.config, "POST", request.URL())

	if err != nil {
		return "", err
	}

	stdOut := bytes.Buffer{}
	stdErr := bytes.Buffer{}

	err = executor.Stream(remotecommand.StreamOptions{
		Stdout: bufio.NewWriter(&stdOut),
		Stderr: bufio.NewWriter(&stdErr),
		Stdin:  stdin,
		Tty:    false,
	})

	if err != nil {
		return "", err
	}

	if stdErr.Len() > 0 {
		return "", errors.New(stdErr.String())
	}

	return stdOut.String(), nil
}
