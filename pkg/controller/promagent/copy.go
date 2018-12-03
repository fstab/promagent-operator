package promagent

import (
	"fmt"
	"io"
	corev1 "k8s.io/api/core/v1"
	"os"
)

// 'kubectl copy' is actually implemented executing 'tar cf -' on the sender side, and 'tar xf -' on the receiver side.
// There is no copy function in the client-go library, so we need to implement this ourselves.
// As we just need to copy single files, we don't need to call 'tar', we can simply use 'cat'.
// See https://stackoverflow.com/questions/49421365/kubernetes-java-api-for-kubectl-copy-command/49421557#49421557
func copyToPod(srcFilePath, destFilePath string, pod *corev1.Pod, container *corev1.Container, execClient *ExecClient) error {
	var (
		execClientErr, readFileErr error
		reader, writer             = io.Pipe()
		errors                     = make(chan error)
	)
	go readFile(srcFilePath, writer, errors)
	_, execClientErr = execClient.Exec(pod, container, reader, fmt.Sprintf("cat > '%s'", destFilePath))
	reader.Close()
	readFileErr = <-errors
	if readFileErr != nil {
		return readFileErr
	}
	if execClientErr != nil {
		return execClientErr
	}
	return nil
}

func readFile(srcFilePath string, dest io.WriteCloser, errors chan error) {
	var (
		srcFile *os.File
		err     error
	)
	defer close(errors)
	srcFile, err = os.Open(srcFilePath)
	if err != nil {
		dest.Close()
		errors <- fmt.Errorf("failed to open file %v: %v", srcFilePath, err)
		return
	}
	defer srcFile.Close()
	_, err = io.Copy(dest, srcFile)
	if err != nil {
		dest.Close()
		errors <- fmt.Errorf("failed to read file %v: %v", srcFilePath, err)
		return
	}
	dest.Close()
}
