package promagent

import (
	"fmt"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"strings"
)

// If JAVA_HOME is not set, queryJavaProcesses returns an empty result and no error.
// error != nil means JAVA_HOME is set but the jps call failed.
func (r *ReconcilePromagent) queryJavaProcesses(pod *corev1.Pod, container *corev1.Container, log logr.InfoLogger) (javaProcesses, error) {
	var (
		command      = "if test -n \"$JAVA_HOME\" ; then \"$JAVA_HOME/bin/jps\" ; fi"
		result       = make([]javaProcess, 0, 1)
		proc         javaProcess
		stdout, line string
		err          error
	)
	log.Info(fmt.Sprintf("executing command %q", command))
	stdout, err = r.execClient.Exec(pod, container, nil, command)
	if err != nil {
		return nil, err
	}
	for _, line = range strings.Split(stdout, "\n") {
		proc, err = parseJpsOutput(line)
		if err == nil && !strings.EqualFold(proc.name, "jps") {
			result = append(result, proc)
		}
	}
	return result, nil
}

type javaProcess struct {
	pid  int
	name string
}

func (p javaProcess) String() string {
	return fmt.Sprintf("%q (pid=%d)", p.name, p.pid)
}

type javaProcesses []javaProcess

func (p javaProcesses) String() string {
	processes := make([]string, len(p))
	for i, proc := range p {
		processes[i] = proc.String()
	}
	return "[" + strings.Join(processes, ",") + "]"
}
