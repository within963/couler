package submitter

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/clientcmd"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
)

// ArgoWorkflowSubmitter holds configurations used for workflow submission
type ArgoWorkflowSubmitter struct {
	namespace      string
	kubeConfigPath string
}

// Submit takes an Argo Workflow object and submit it to Kubernetes cluster
func (submitter *ArgoWorkflowSubmitter) Submit(wf wfv1.Workflow) (wfv1.Workflow, error) {
	// Use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", submitter.kubeConfigPath)
	if err != nil {
		return wf, fmt.Errorf("failed to get the current context in the kubeconfig file %s", submitter.kubeConfigPath)
	}

	// Create the workflow client
	wfClient := wfclientset.NewForConfigOrDie(config).ArgoprojV1alpha1().Workflows(submitter.namespace)

	// Submit the workflow
	createdWf, err := wfClient.Create(&wf)
	if err != nil {
		return wf, fmt.Errorf("failed to created the workflow %s", wf.Name)
	}
	fmt.Printf("Workflow %s submitted\n", createdWf.Name)

	// Wait for the workflow to complete
	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", createdWf.Name))
	watchIf, err := wfClient.Watch(metav1.ListOptions{FieldSelector: fieldSelector.String()})
	if err != nil {
		return wf, fmt.Errorf("failed establish a watch")
	}
	defer watchIf.Stop()
	for next := range watchIf.ResultChan() {
		wf, ok := next.Object.(*wfv1.Workflow)
		if !ok {
			continue
		}
		if wf.Status.Phase == wfv1.NodeFailed || wf.Status.Phase == wfv1.NodeError {
			return *wf, fmt.Errorf("workflow %s failed", wf.Name)
		}
		if !wf.Status.FinishedAt.IsZero() {
			fmt.Printf("Workflow %s %s at %v\n", wf.Name, wf.Status.Phase, wf.Status.FinishedAt)
			break
		}
	}
	return wf, nil
}
