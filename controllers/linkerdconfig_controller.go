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
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ref "k8s.io/client-go/tools/reference"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/linkerd/linkerd-config/api/v1alpha1"
	configv1alpha1 "github.com/linkerd/linkerd-config/api/v1alpha1"
)

// LinkerdConfigReconciler reconciles a LinkerdConfig object
type LinkerdConfigReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=config.linkerd.io,resources=linkerdconfigs,verbs=get;list;watch
// +kubebuilder:rbac:groups=config.linkerd.io,resources=linkerdconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources="pods",verbs=get;list
// +kubebuilder:rbac:groups="",resources="configmaps",verbs=get;create;update

func (r *LinkerdConfigReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	var (
		ctx = context.Background()
		log = r.Log.WithValues("linkerdconfig", req.NamespacedName)
	)
	log.V(1).Info("receive request")

	var config configv1alpha1.LinkerdConfig
	if err := r.Get(ctx, req.NamespacedName, &config); err != nil {
		return ctrl.Result{}, ignoreNotFound(err)
	}

	var (
		configmap      corev1.ConfigMap
		createConfigCM bool
	)
	fetchConfigMap := func() error {
		namespacedName := types.NamespacedName{
			Namespace: config.Spec.Global.LinkerdNamespace,
			Name:      config.Spec.Global.ConfigMap,
		}
		if err := r.Get(ctx, namespacedName, &configmap); err != nil {
			if apierrs.IsNotFound(err) {
				createConfigCM = true
				configmap = corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: namespacedName.Namespace,
						Name:      namespacedName.Name,
					},
					Data: map[string]string{},
				}
				return nil
			}

			return err
		}

		return nil
	}

	if err := fetchConfigMap(); err != nil {
		log.Error(err, "fail to retrieve configmap")
		return ctrl.Result{}, err
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

		log.Info("attempting to reconcile the configmap")
		if err := r.reconcileConfigMap(&config, &configmap, log); err != nil {
			errChan <- err
			return
		}

		if createConfigCM {
			log.Info("created new configmap", "name", configmap.ObjectMeta.Name, "namespace", configmap.ObjectMeta.Namespace)
			if err := r.Create(ctx, &configmap); err != nil {
				log.Error(err, "fail to create the configmap")
				errChan <- err
				return
			}

			if err := r.Update(ctx, &config); err != nil {
				errChan <- err
			}

			return
		}

		if err := r.Update(ctx, &configmap); err != nil {
			log.Error(err, "fail to update the configmap")
			errChan <- err
		}
		log.Info("successfully reconciled configmap")
	}()

	wait.Add(1)
	go func() {
		defer wait.Done()

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

	return ctrl.Result{}, nil
}

func (r *LinkerdConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&configv1alpha1.LinkerdConfig{}).
		Owns(&corev1.ConfigMap{}).
		WithEventFilter(predicate.ResourceVersionChangedPredicate{}).
		Complete(r)
}

func ignoreNotFound(err error) error {
	if apierrs.IsNotFound(err) {
		return nil
	}

	return err
}

func (r *LinkerdConfigReconciler) reconcileConfigMap(config *v1alpha1.LinkerdConfig, configmap *corev1.ConfigMap, log logr.Logger) error {
	if err := ctrl.SetControllerReference(config, configmap, r.Scheme); err != nil {
		cast, ok := err.(*controllerutil.AlreadyOwnedError)
		if !ok { // return original error as-is
			return err
		}

		if !reflect.DeepEqual(cast.Object, config) {
			return errors.New("configmap has unexpected ownerRef")
		}
	}

	reconcileData := func(dataKey string, spec interface{}) error {
		desiredConfig, err := json.Marshal(spec)
		if err != nil {
			return err
		}

		configmap.Data[dataKey] = string(desiredConfig)
		log.V(1).Info("successfully reconciled configmap data", "key", dataKey)
		return nil
	}

	if err := reconcileData("global", config.Spec.Global); err != nil {
		return err
	}

	if err := reconcileData("proxy", config.Spec.Proxy); err != nil {
		return err
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

	isRunning := func(pod corev1.Pod) bool {
		return pod.Status.Phase == corev1.PodRunning
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
		opts = &client.ListOptions{
			Limit:    listQueryLimitPod,
			Continue: fmt.Sprintf("%s-%d", listQueryLimitContinueToken, time.Now().Second()),
		}
	)
	if err := r.List(ctx, &pods, opts); ignoreNotFound(err) != nil {
		return err
	}
	log.V(1).Info("found pods", "total", len(pods.Items))

	for _, pod := range pods.Items {
		if !isRunning(pod) {
			continue
		}

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
