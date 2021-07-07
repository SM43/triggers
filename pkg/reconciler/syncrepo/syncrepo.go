/*
Copyright 2021 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package syncrepo

import (
	"context"
	"fmt"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	"github.com/tektoncd/triggers/pkg/client/injection/reconciler/triggers/v1alpha1/syncrepo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	pkgreconciler "knative.dev/pkg/reconciler"
)

var (
	// Check that our Reconciler implements syncrepo.Interface
	_ syncrepo.Interface = (*Reconciler)(nil)
	// Check that our Reconciler implements syncrepo.Finalizer
	_ syncrepo.Finalizer = (*Reconciler)(nil)
)

// Reconciler implements controller.Reconciler for Configuration resources.
type Reconciler struct {
	DynamicClientSet  dynamic.Interface
	KubeClientSet     kubernetes.Interface
	TriggersClientSet triggersclientset.Interface
}

func (r *Reconciler) ReconcileKind(ctx context.Context, sr *v1alpha1.SyncRepo) pkgreconciler.Event {

	res, err := r.TriggersClientSet.TriggersV1alpha1().SyncRepos(sr.Namespace).Get(ctx, sr.Name, metav1.GetOptions{})
	if err != nil {
		fmt.Println("Err occurred: ", err)
		return err
	}
	fmt.Println("Repo: ", res.Spec.Repo)

	// https://api.github.com/repos/{owner}/{repo}
	// https://api.github.com/repos/tektoncd/hub/commits

	return nil
}

// FinalizeKind cleans up associated logging config maps when an EventListener is deleted
func (r *Reconciler) FinalizeKind(ctx context.Context, sr *v1alpha1.SyncRepo) pkgreconciler.Event {
	return nil
}
