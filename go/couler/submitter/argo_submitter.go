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
	// Create the workflow client
	wfClient := wfclientset.NewForConfigOrDie(config).ArgoprojV1alpha1().Workflows(submitter.namespace)

	// Submit the workflow
	createdWf, err := wfClient.Create(&wf)
	checkErr(err)
	fmt.Printf("Workflow %s submitted\n", createdWf.Name)

	// Wait for the workflow to complete
	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", createdWf.Name))
	watchIf, err := wfClient.Watch(metav1.ListOptions{FieldSelector: fieldSelector.String()})
	checkErr(err)
	defer watchIf.Stop()
	for next := range watchIf.ResultChan() {
		wf, ok := next.Object.(*wfv1.Workflow)
		if !ok {
			continue
		}
		if !wf.Status.FinishedAt.IsZero() {
			fmt.Printf("Workflow %s %s at %v\n", wf.Name, wf.Status.Phase, wf.Status.FinishedAt)
			break
		}
	}
	return wf, err
}

func checkErr(err error) {
	if err != nil {
		panic(err.Error())
	}
}
