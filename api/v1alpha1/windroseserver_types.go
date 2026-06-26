package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	PhasePending = "Pending"
	PhaseRunning = "Running"
	PhaseFailed  = "Failed"
)

// WindroseServerSpec defines the desired state of a Windrose dedicated game server.
// Settings map to ServerDescription.json fields documented at
// https://playwindrose.com/dedicated-server-guide and the official Docker image
// https://hub.docker.com/r/windroseserver/windroseserver
type WindroseServerSpec struct {
	// ServerImage is the official Windrose Linux server container image.
	// +kubebuilder:default="windroseserver/windroseserver:latest"
	// +optional
	ServerImage string `json:"serverImage,omitempty"`

	// InviteCode is used when UseDirectConnection is false. Minimum 6 characters.
	// +kubebuilder:validation:MinLength=6
	// +kubebuilder:validation:MaxLength=32
	// +optional
	InviteCode string `json:"inviteCode,omitempty"`

	// UseDirectConnection enables direct IP connections (recommended for Kubernetes).
	// +kubebuilder:default=true
	// +optional
	UseDirectConnection *bool `json:"useDirectConnection,omitempty"`

	// DirectConnectionServerPort is exposed for TCP and UDP game traffic.
	// +kubebuilder:default=7777
	// +kubebuilder:validation:Minimum=1024
	// +kubebuilder:validation:Maximum=65535
	// +optional
	DirectConnectionServerPort int32 `json:"directConnectionServerPort,omitempty"`

	// DirectConnectionProxyAddress selects the bind address inside the container.
	// +kubebuilder:default="0.0.0.0"
	// +optional
	DirectConnectionProxyAddress string `json:"directConnectionProxyAddress,omitempty"`

	// ServerName is shown in the server browser when invite codes look similar.
	// +optional
	ServerName string `json:"serverName,omitempty"`

	// Password protects the server when set. Leave empty for a public server.
	// +optional
	Password string `json:"password,omitempty"`

	// MaxPlayerCount is the maximum number of simultaneous players.
	// +kubebuilder:default=4
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=32
	// +optional
	MaxPlayerCount int32 `json:"maxPlayerCount,omitempty"`

	// UserSelectedRegion selects the connection service region: SEA, CIS, or EU.
	// Leave empty to auto-detect based on latency.
	// +kubebuilder:validation:Enum=SEA;CIS;EU;""
	// +optional
	UserSelectedRegion string `json:"userSelectedRegion,omitempty"`

	// P2pProxyAddress is the IP address for listening sockets when using invite codes.
	// +kubebuilder:default="127.0.0.1"
	// +optional
	P2pProxyAddress string `json:"p2pProxyAddress,omitempty"`

	// AutoLoadLatestBackupIfHasBroken restores broken saves from backups on launch.
	// +kubebuilder:default=true
	// +optional
	AutoLoadLatestBackupIfHasBroken *bool `json:"autoLoadLatestBackupIfHasBroken,omitempty"`

	// StorageSize is the requested PVC capacity for world saves.
	// +kubebuilder:default="35Gi"
	// +optional
	StorageSize string `json:"storageSize,omitempty"`

	// StorageClassName selects the StorageClass for persistent saves.
	// +optional
	StorageClassName string `json:"storageClassName,omitempty"`

	// Resources for the game server container. Defaults match Windrose docs for 4 players.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// ServiceType exposes the game port outside the cluster.
	// +kubebuilder:validation:Enum=LoadBalancer;NodePort;ClusterIP
	// +kubebuilder:default=LoadBalancer
	// +optional
	ServiceType corev1.ServiceType `json:"serviceType,omitempty"`

	// NodePort is used when ServiceType is NodePort. Must be in range 30000-32767 when set.
	// +kubebuilder:validation:Minimum=30000
	// +kubebuilder:validation:Maximum=32767
	// +optional
	NodePort int32 `json:"nodePort,omitempty"`
}

// WindroseServerStatus defines the observed state of WindroseServer.
type WindroseServerStatus struct {
	// Phase summarizes high-level lifecycle state.
	// +optional
	Phase string `json:"phase,omitempty"`

	// Ready indicates the game server pod is ready.
	// +optional
	Ready bool `json:"ready,omitempty"`

	// InviteCode reflects the active invite code when not using direct connection.
	// +optional
	InviteCode string `json:"inviteCode,omitempty"`

	// ConnectionAddress is the host/IP clients should use for direct connections.
	// +optional
	ConnectionAddress string `json:"connectionAddress,omitempty"`

	// ConnectionPort is the port clients should use for direct connections.
	// +optional
	ConnectionPort int32 `json:"connectionPort,omitempty"`

	// Message provides human-readable status details.
	// +optional
	Message string `json:"message,omitempty"`

	// Conditions represent the latest available observations.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=ws
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`
// +kubebuilder:printcolumn:name="Port",type=integer,JSONPath=`.status.connectionPort`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// WindroseServer is the Schema for the windroseservers API.
type WindroseServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WindroseServerSpec   `json:"spec,omitempty"`
	Status WindroseServerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WindroseServerList contains a list of WindroseServer.
type WindroseServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WindroseServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WindroseServer{}, &WindroseServerList{})
}
