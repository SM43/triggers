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
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	v1beta12 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	"github.com/tektoncd/triggers/pkg/client/injection/reconciler/triggers/v1alpha1/syncrepo"
	"github.com/tektoncd/triggers/pkg/resources"
	"github.com/tektoncd/triggers/pkg/template"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	discoveryclient "k8s.io/client-go/discovery"
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
	DiscoveryClient   discoveryclient.ServerResourcesInterface
	DynamicClientSet  dynamic.Interface
	KubeClientSet     kubernetes.Interface
	TriggersClientSet triggersclientset.Interface
	Requeue           func(obj interface{}, after time.Duration)
}

func (r *Reconciler) ReconcileKind(ctx context.Context, sr *v1alpha1.SyncRepo) pkgreconciler.Event {

	logger := logging.FromContext(ctx)

	fmt.Println("************************************************************")
	fmt.Println("************************************************************")
	fmt.Println("What's the time :P ", time.Now())
	fmt.Println("************************************************************")
	fmt.Println("************************************************************")

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
		logger.Error("Error occurred while json chi unmarshalling: ", err)
		return err
	}

	abyte, err := json.Marshal(gc[0])
	if err != nil {
		logger.Error("Error occurred while interface chi marshalling: ", err)
		return err
	}

	logger.Info("Sha: ", gc[0]["sha"].(string))

	// https://api.github.com/repos/{owner}/{repo}
	// https://api.github.com/repos/tektoncd/hub/commits

	tb, err := r.TriggersClientSet.TriggersV1beta1().TriggerBindings("default").Get(ctx, "pipeline-binding", metav1.GetOptions{})
	if err != nil {
		logger.Error("Error occurred while getting tb: ", err)
		return err
	}
	logger.Info("----found tb ", tb.Spec.Params)

	p, err := template.ApplyEventValuesToParams(tb.Spec.Params, abyte, http.Header{}, map[string]interface{}{}, []v1beta12.ParamSpec{})
	if err != nil {
		logger.Error("templating failed ", err)
		return err
	}

	tt, err := r.TriggersClientSet.TriggersV1beta1().TriggerTemplates("default").Get(ctx, "pipeline-template", metav1.GetOptions{})
	if err != nil {
		logger.Error("Error occurred while getting tb: ", err)
		return err
	}

	logger.Infof("ResolvedParams : %+v", p)
	resolvedRes := template.ResolveResources(tt, p)

	for _, re := range resolvedRes {
		data := new(unstructured.Unstructured)
		if err := data.UnmarshalJSON(re); err != nil {
			return fmt.Errorf("couldn't unmarshal json from the TriggerTemplate: %v", err)
		}

		logger.Info("trigger template data : ", data)

		apiResource, err := resources.FindAPIResource(data.GetAPIVersion(), data.GetKind(), r.DiscoveryClient)
		if err != nil {
			logger.Error("Error occurred while api resource: ", err)
			return err
		}

		name := data.GetName()
		if name == "" {
			name = data.GetGenerateName()
		}
		logger.Infof("Generating resource: kind: %s, name: %s", apiResource, name)

		gvr := schema.GroupVersionResource{
			Group:    apiResource.Group,
			Version:  apiResource.Version,
			Resource: apiResource.Name,
		}

		if _, err := r.DynamicClientSet.Resource(gvr).Namespace("default").Create(context.Background(), data, metav1.CreateOptions{}); err != nil {
			if kerrors.IsUnauthorized(err) || kerrors.IsForbidden(err) {
				return err
			}
			return fmt.Errorf("couldn't create resource with group version kind %q: %v", gvr, err)
		}
	}

	logger.Info("Now I am queuing .... yay !!!")
	r.Requeue(sr, time.Minute)
	logger.Info("See you after a minute .... !!!")

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
