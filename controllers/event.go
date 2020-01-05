package controllers

const (
	// EventPodRestart is the `reason` used by EventRecorder to represent
	// "pod restart" events
	EventPodRestart = "PodRestart"

	// EventConfigMapUpdated is the `reason` used by EventRecorder to represent
	// "configmap updated" events
	EventConfigMapUpdated = "ConfigMapUpdated"

	// EventLinkerdConfigUpdated is the `reason` used by EventRecorder to represent
	// "Linkerd config updated" events
	EventLinkerdConfigUpdated = "LinkerdConfigUpdated"
)
