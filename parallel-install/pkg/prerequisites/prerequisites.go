package prerequisites

import (
	"context"
	"fmt"
	"github.com/kyma-incubator/hydroform/parallel-install/pkg/components"
	"github.com/kyma-incubator/hydroform/parallel-install/pkg/config"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const logPrefix = "[prerequisites/prerequisites.go]"

func InstallPrerequisites(ctx context.Context, prerequisites []components.Component, kubeClient kubernetes.Interface) <-chan error {

	statusChan := make(chan error)

	go func() {
		defer close(statusChan)

		config.Log("Creating kyma-installer namespace")
		_, err := kubeClient.CoreV1().Namespaces().Create(context.Background(), &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "kyma-installer",
				Labels: map[string]string{"istio-injection": "disabled", "kyma-project.io/installation": ""},
			},
		}, metav1.CreateOptions{})
		if err != nil {
			statusChan <- fmt.Errorf("Unable to create kyma-installer namespace. Error: %v", err)
			return
		}

		for _, prerequisite := range prerequisites {
			//TODO: Is there a better way to find out if Context is canceled?
			if ctx.Err() != nil {
				//Context is canceled or timed-out. Skip processing
				config.Log("%s Finishing work: %v", logPrefix, ctx.Err())
				return
			}

			config.Log("%s Installing component %s ", logPrefix, prerequisite.Name)
			err := prerequisite.InstallComponent(ctx)
			if err != nil {
				statusChan <- err
				return
			}
			statusChan <- nil
		}
	}()

	return statusChan
}

func UninstallPrerequisites(ctx context.Context, kubeClient kubernetes.Interface, prerequisites []components.Component) <-chan error {

	statusChan := make(chan error)

	go func() {
		defer close(statusChan)

		for i := len(prerequisites) - 1; i >= 0; i-- {
			prereq := prerequisites[i]
			//TODO: Is there a better way to find out if Context is canceled?
			if ctx.Err() != nil {
				//Context is canceled or timed-out. Skip processing
				config.Log("%s Finishing work: %v", logPrefix, ctx.Err())

				return
			}
			config.Log("%s Uninstalling component %s ", logPrefix, prereq.Name)

			err := prereq.UninstallComponent(ctx)
			if err != nil {
				statusChan <- err
				return
			}
			statusChan <- nil
		}

		// TODO: Delete namespace deletion once xip-patch is gone.
		config.Log("Deleting kyma-installer namespace")
		err := kubeClient.CoreV1().Namespaces().Delete(context.Background(), "kyma-installer", metav1.DeleteOptions{})
		if err != nil {
			statusChan <- fmt.Errorf("Unable to delete kyma-installer namespace. Error: %v", err)
			return
		}
	}()

	return statusChan
}
