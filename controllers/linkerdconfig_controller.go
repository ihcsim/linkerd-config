/*

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

package controllers

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apilabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ref "k8s.io/client-go/tools/reference"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/ihcsim/linkerd-config/api/v1alpha1"
	configv1alpha1 "github.com/ihcsim/linkerd-config/api/v1alpha1"
)

// LinkerdConfigReconciler reconciles a LinkerdConfig object
type LinkerdConfigReconciler struct {
	client.Client
	record.EventRecorder
	Log                logr.Logger
	Scheme             *runtime.Scheme
	indexFieldPodPhase string
}

// +kubebuilder:rbac:groups=config.linkerd.io,resources=linkerdconfigs,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=config.linkerd.io,resources=linkerdconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources="pods",verbs=get;list;watch;delete;deletecollection
// +kubebuilder:rbac:groups="",resources="namespaces",verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources="configmaps",verbs=get;create;update;list;watch
// +kubebuilder:rbac:groups="",resources="events",verbs=create;patch

func (r *LinkerdConfigReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	var (
		ctx = context.Background()
		log = r.Log.WithValues("linkerdconfig", req.NamespacedName)
	)
	log.V(1).Info("receive request")

	var config configv1alpha1.LinkerdConfig
	if err := r.Get(ctx, req.NamespacedName, &config); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	configmap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   config.Spec.Global.LinkerdNamespace,
			Name:        config.Spec.Global.ConfigMap,
			Annotations: annotations(fmt.Sprintf("linkerd/reconciler %s", config.Spec.Global.Version)),
			Labels:      labels(config.Spec.Global.LinkerdNamespace),
		},
		Data: map[string]string{},
	}

	var (
		wait    = sync.WaitGroup{}
		errChan = make(chan error)
		errs    []error
	)

	// gather all errors returned by the goroutines
	go func() {
		for err := range errChan {
			errs = append(errs, err)
		}
	}()

	wait.Add(1)
	go func() {
		defer wait.Done()

		// attempt to reconcile the configmap.
		// create it if it's missing. otherwise, update it.
		// then restart all opt-in pods.

		log.Info("attempting to reconcile the configmap")
		reconcileConfigMap := func() error {
			return r.reconcileConfigMap(ctx, &config, &configmap, log)
		}
		result, err := controllerutil.CreateOrUpdate(ctx, r.Client, &configmap, reconcileConfigMap)
		if err != nil {
			errChan <- err
			return
		}
		log.Info("successfully reconciled configmap", "result", result)

		go func() {
			restarted, err := r.restartPods(ctx, log)
			if err != nil {
				log.Error(err, "fail to restart some pods")
				errChan <- err
			}

			for _, namespace := range restarted.Items {
				r.Eventf(&config, corev1.EventTypeNormal, EventPodRestart, "restarted opt-in pods in %s", namespace.Name)
			}
		}()
	}()

	wait.Add(1)
	go func() {
		defer wait.Done()

		// attempt to reconcile the LinkerdConfig resource.
		// update its status to reflect the state of the world.

		log.Info("attempting to reconcile custom resource status")
		if err := r.reconcileStatus(ctx, &config, log); err != nil {
			errChan <- err
			return
		}

		if err := r.Status().Update(ctx, &config); err != nil {
			errChan <- err
			return
		}
		log.Info("successfullly reconciled custom resource status")
	}()

	wait.Wait()

	//  return any errors returned by the goroutines to the caller
	for _, err := range errs {
		return ctrl.Result{}, err
	}

	r.Eventf(&config, corev1.EventTypeNormal, EventLinkerdConfigUpdated, "successfully sync LinkerdConfig '%s' with configmap '%s'", req.NamespacedName.Name, configmap.Name)
	return ctrl.Result{}, nil
}

func (r *LinkerdConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// set up an index to allow us to list pods by their pod phase field.
	// the client will use the r.indexFieldPodPhase as the index key.
	// see https://godoc.org/sigs.k8s.io/controller-runtime/pkg/client#FieldIndexer
	// and https://github.com/kubernetes-sigs/kubebuilder/blob/master/docs/book/src/cronjob-tutorial/testdata/project/controllers/cronjob_controller.go
	r.indexFieldPodPhase = ".status.phase"
	indexerFunc := func(obj runtime.Object) []string {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			return nil
		}

		return []string{string(pod.Status.Phase)}
	}
	if err := mgr.GetFieldIndexer().IndexField(&corev1.Pod{}, r.indexFieldPodPhase, indexerFunc); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&configv1alpha1.LinkerdConfig{}).
		Owns(&corev1.ConfigMap{}).
		WithEventFilter(predicate.ResourceVersionChangedPredicate{}).
		Complete(r)
}

func (r *LinkerdConfigReconciler) reconcileConfigMap(ctx context.Context, config *v1alpha1.LinkerdConfig, configmap *corev1.ConfigMap, log logr.Logger) error {
	// update the configmap's ownerRef to point to the custom resource, making
	// the custom resource its owner.
	if err := ctrl.SetControllerReference(config, configmap, r.Scheme); err != nil {
		cast, ok := err.(*controllerutil.AlreadyOwnedError)
		if !ok { // return original error as-is
			return err
		}

		if !reflect.DeepEqual(cast.Object, config) {
			return errors.New("configmap has unexpected ownerRef")
		}
	}

	// reconcile the 'global' and 'proxy' data.
	// also, convert them to the Linkerd proto format.
	reconcileData := func(dataKey string, toProtoJSON func() (string, error)) error {
		p, err := toProtoJSON()
		if err != nil {
			return err
		}

		configmap.Data[dataKey] = p
		log.V(1).Info("successfully reconciled configmap data", "key", dataKey)
		return nil
	}

	const (
		keyGlobal = "global"
		keyProxy  = "proxy"
	)

	if err := reconcileData(keyGlobal, config.Spec.Global.ToProtoJSON); err != nil {
		return err
	}

	if err := reconcileData(keyProxy, config.Spec.Proxy.ToProtoJSON); err != nil {
		return err
	}

	// remove all unexpected keys
	for key := range configmap.Data {
		if key != keyGlobal && key != keyProxy {
			delete(configmap.Data, key)
		}
	}

	return nil
}

func (r *LinkerdConfigReconciler) reconcileStatus(ctx context.Context, config *v1alpha1.LinkerdConfig, log logr.Logger) error {
	const (
		listQueryLimitPod           = 100
		listQueryLimitContinueToken = "l5d-list-pods-continue"
	)

	addIfMissing := func(objRefs []corev1.ObjectReference, podRef corev1.ObjectReference) []corev1.ObjectReference {
		var exists bool
		for _, objRef := range objRefs {
			if objRef.Name == podRef.Name {
				exists = true
				break
			}
		}

		if !exists {
			return append(objRefs, podRef)
		}

		return objRefs
	}

	hasProxy := func(pod corev1.Pod) bool {
		const proxyName = "linkerd-proxy"

		for _, container := range pod.Spec.Containers {
			if container.Name == proxyName {
				return true
			}
		}

		return false
	}

	// find all the pods in the cluster and determine if they are injected.
	// if a pod is injected, add it to the LinkerdConfig.Status.Injected list.
	// otherwise, add it to the LinkerdConfig.Status.Uninjected list.

	var (
		pods = corev1.PodList{}
		opts = []client.ListOption{
			client.Limit(listQueryLimitPod),
			client.Continue(fmt.Sprintf("%s-%d", listQueryLimitContinueToken, time.Now().Second())),
			client.MatchingFields{r.indexFieldPodPhase: string(corev1.PodRunning)},
		}
	)
	if err := r.List(ctx, &pods, opts...); client.IgnoreNotFound(err) != nil {
		return err
	}
	log.V(1).Info("found running pods", "total", len(pods.Items))

	for _, pod := range pods.Items {
		podRef, err := ref.GetReference(r.Scheme, &pod)
		if err != nil {
			continue
		}

		if hasProxy(pod) {
			config.Status.Injected = addIfMissing(config.Status.Injected, *podRef)
			continue
		}

		config.Status.Uninjected = addIfMissing(config.Status.Uninjected, *podRef)
	}

	return nil
}

func (r *LinkerdConfigReconciler) restartPods(ctx context.Context, log logr.Logger) (corev1.NamespaceList, error) {
	// find all namespaces without the config.linkerd.io/admission-webooks=disabled label
	var (
		namespaces   corev1.NamespaceList
		nsListOption client.ListOptions
	)
	nsListSelector, err := apilabels.Parse(fmt.Sprintf("%s != %s", linkerdOptOutLabel, linkerdOptOutModeDisabled))
	if err != nil {
		return namespaces, err
	}
	nsListOption.LabelSelector = nsListSelector
	if err := r.List(ctx, &namespaces, &nsListOption); err != nil {
		return namespaces, err
	}

	var errs []error
	selector, err := apilabels.Parse(fmt.Sprintf("%s = %s", proxyReconcileLabel, proxyReconcileModeAuto))
	if err != nil {
		return namespaces, err
	}
	for _, namespace := range namespaces.Items {
		log.V(1).Info("looking for opt-in pods to be restarted", "namespace", namespace.Name)
		option := client.DeleteAllOfOptions{
			ListOptions: client.ListOptions{
				Namespace:     namespace.Name,
				LabelSelector: selector,
			},
		}

		if err := r.DeleteAllOf(ctx, &corev1.Pod{}, &option); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		var combined error
		for _, err := range errs {
			combined = fmt.Errorf("%s,%s", combined, err)
		}

		return namespaces, combined
	}

	return namespaces, nil
}
