package controllers

const (
	proxyReconcileLabel    = "config.linkerd.io/reconcile"
	proxyReconcileModeAuto = "auto"

	linkerdOptOutLabel        = "config.linkerd.io/admission-webhooks"
	linkerdOptOutModeDisabled = "disabled"
)

func annotations(createdBy string) map[string]string {
	return map[string]string{
		"linkerd.io/created-by": createdBy,
	}
}

func labels(namespace string) map[string]string {
	return map[string]string{
		"linkerd.io/control-plane-component": "controller",
		"linkerd.io/control-plane-ns":        namespace,
	}
}
