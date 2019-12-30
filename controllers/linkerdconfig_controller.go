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
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ref "k8s.io/client-go/tools/reference"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

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
	log.V(1).Info("found the custom resource")

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
				configmap.ObjectMeta.Namespace = namespacedName.Namespace
				configmap.ObjectMeta.Name = namespacedName.Name
				return nil
			}

			return err
		}

		return nil
	}

	if err := fetchConfigMap(); err != nil {
		log.Error(err, "fail to create configmap")
		return ctrl.Result{}, err
	}

	log.V(1).Info("reconciling the configmap")
	if err := r.reconcileConfigMap(&config, &configmap); err != nil {
		return ctrl.Result{}, err
	}

	log.V(1).Info("reconciling the custom resource's status")
	if err := r.reconcileStatus(ctx, &config); err != nil {
		return ctrl.Result{}, err
	}

	if createConfigCM {
		log.Info("creating the missing configmap", "name", configmap.Name)
		if err := r.Create(ctx, &configmap); err != nil {
			log.Error(err, "fail to create configmap")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	log.V(1).Info("updating the configmap")
	if err := r.Update(ctx, &configmap); err != nil {
		return ctrl.Result{}, err
	}

	log.V(1).Info("updating the custom resource's status")
	if err := r.Status().Update(ctx, &config); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *LinkerdConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&configv1alpha1.LinkerdConfig{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

func ignoreNotFound(err error) error {
	if apierrs.IsNotFound(err) {
		return nil
	}

	return err
}

func (r *LinkerdConfigReconciler) reconcileConfigMap(config *v1alpha1.LinkerdConfig, configmap *corev1.ConfigMap) error {
	if err := ctrl.SetControllerReference(config, configmap, r.Scheme); err != nil {
		cast, ok := err.(*controllerutil.AlreadyOwnedError)
		if !ok { // return original error as-is
			return err
		}

		if !reflect.DeepEqual(cast.Object, config) {
			return errors.New("configmap has unexpected ownerRef")
		}
	}

	return nil
}

func (r *LinkerdConfigReconciler) reconcileStatus(ctx context.Context, config *v1alpha1.LinkerdConfig) error {
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
