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
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	"github.com/tektoncd/triggers/pkg/client/injection/reconciler/triggers/v1alpha1/syncrepo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/logging"
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

	logger := logging.FromContext(ctx)

	res, err := r.TriggersClientSet.TriggersV1alpha1().SyncRepos(sr.Namespace).Get(ctx, sr.Name, metav1.GetOptions{})
	if err != nil {
		logger.Error("Error occurred while fetching sync repo: ", err)
		return err
	}

	logger.Info("Repo: ", res.Spec.Repo)

	data, code, err := httpGet("https://api.github.com/repos/tektoncd/hub/commits")
	if err != nil {
		logger.Error("Error occurred while fetching repo details: ", err)
		return err
	}

	logger.Info("Status code: ", code)

	var gc []map[string]interface{}
	err = json.Unmarshal(data, &gc)
	if err != nil {
		logger.Error("Error occurred while json chi marshalling: ", err)
		return err
	}

	logger.Info("Sha: ", gc[0]["sha"].(string))

	// https://api.github.com/repos/{owner}/{repo}
	// https://api.github.com/repos/tektoncd/hub/commits

	return nil
}

// FinalizeKind cleans up associated logging config maps when an EventListener is deleted
func (r *Reconciler) FinalizeKind(ctx context.Context, sr *v1alpha1.SyncRepo) pkgreconciler.Event {
	return nil
}

// httpGet gets raw data given the url
func httpGet(url string) ([]byte, int, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}

	return data, resp.StatusCode, err
}
