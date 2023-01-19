package pod

import (
	"bytes"
	"context"
	"github.com/Dynatrace/dynatrace-operator/test/shell"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient"

	"github.com/pkg/errors"
)

//const (
//	resourcePods = "pods"
//	resourceExec = "exec"
//)
//
//type ExecutionResult struct {
//	StdOut *bytes.Buffer
//	StdErr *bytes.Buffer
//}
//

type ExecutionQueryNg struct {
	pod       v1.Pod
	command   shell.Command
	container string
}

func NewExecutionQueryNg(pod v1.Pod, container string, command ...string) ExecutionQueryNg {
	query := ExecutionQueryNg{
		pod:       pod,
		container: container,
		command:   make([]string, 0),
	}
	query.command = append(query.command, command...)
	return query
}

func (query ExecutionQueryNg) ExecuteNg(ctx context.Context, client klient.Client) (*ExecutionResult, error) {

	result := &ExecutionResult{
		StdOut: &bytes.Buffer{},
		StdErr: &bytes.Buffer{},
	}

	err := client.Resources().ExecInPod(ctx,
		query.pod.Namespace, query.pod.Name, "dynatrace-operator", query.command, result.StdOut, result.StdErr)

	if err != nil {
		return result, errors.WithMessagef(errors.WithStack(err),
			"stdout:\n%s\nstderr:\n%s", result.StdOut.String(), result.StdErr.String())
	}

	return result, nil
}

func ExecuteNg(ctx context.Context, client klient.Client, pod v1.Pod, container string, command ...string) (*ExecutionResult, error) {
	result := &ExecutionResult{
		StdOut: &bytes.Buffer{},
		StdErr: &bytes.Buffer{},
	}

	err := client.Resources().ExecInPod(ctx,
		pod.Namespace, pod.Name, container, command, result.StdOut, result.StdErr)

	if err != nil {
		return result, errors.WithMessagef(errors.WithStack(err),
			"stdout:\n%s\nstderr:\n%s", result.StdOut.String(), result.StdErr.String())
	}

	return result, nil
}
