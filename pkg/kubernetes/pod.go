package kubernetes

import (
	"context"
	"fmt"

	"github.com/kudobuilder/test-tools/pkg/cmd"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

//go:generate stub-gen -api CoreV1 -type Pod

// ContainerLogs returns the (current) logs of a pod's container.
func (pod Pod) ContainerLogs(container string) ([]byte, error) {
	options := corev1.PodLogOptions{
		Container: container,
	}

	result := pod.client.Kubernetes.
		CoreV1().
		Pods(pod.Namespace).
		GetLogs(pod.Name, &options).
		Do(pod.client.Ctx)

	if result.Error() != nil {
		return []byte{}, fmt.Errorf("failed to get logs of container %s: %w", container, result.Error())
	}

	return result.Raw()
}

// ContainerExec runs a command in a pod's container.
func (pod Pod) ContainerExec(container string, command cmd.Builder) error {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		60,
	)
	defer cancel()

	options := corev1.PodExecOptions{
		Container: container,
		Command:   append([]string{command.Command}, command.Arguments...),
		Stdin:     command.Stdin != nil,
		Stdout:    command.Stdout != nil,
		Stderr:    command.Stderr != nil,
		TTY:       false,
	}

	// adapted from https://github.com/kubernetes/kubernetes/blob/master/test/e2e/framework/exec_util.go
	req := pod.client.Kubernetes.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec").
		Param("container", container)
	req.VersionedParams(&options, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(&pod.client.Config, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("failed to execute \"%s\" in container %s: %w", command.Command, container, err)
	}

	return exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  command.Stdin,
		Stdout: command.Stdout,
		Stderr: command.Stderr,
		Tty:    false,
	})
}
