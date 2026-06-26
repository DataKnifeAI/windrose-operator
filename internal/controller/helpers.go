package controller

import (
	"encoding/json"
	"strings"

	windrosev1alpha1 "github.com/DataKnifeAI/windrose-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"

	egv1a1 "github.com/envoyproxy/gateway/api/v1alpha1"
)

type serverDescriptionFile struct {
	Version                     int                         `json:"Version"`
	ServerDescriptionPersistent serverDescriptionPersistent `json:"ServerDescription_Persistent"`
}

type serverDescriptionPersistent struct {
	InviteCode                      string `json:"InviteCode,omitempty"`
	IsPasswordProtected             bool   `json:"IsPasswordProtected"`
	Password                        string `json:"Password,omitempty"`
	ServerName                      string `json:"ServerName,omitempty"`
	MaxPlayerCount                  int32  `json:"MaxPlayerCount"`
	UserSelectedRegion              string `json:"UserSelectedRegion,omitempty"`
	P2pProxyAddress                 string `json:"P2pProxyAddress"`
	UseDirectConnection             bool   `json:"UseDirectConnection"`
	DirectConnectionServerPort      int32  `json:"DirectConnectionServerPort"`
	DirectConnectionProxyAddress    string `json:"DirectConnectionProxyAddress"`
	AutoLoadLatestBackupIfHasBroken bool   `json:"AutoLoadLatestBackupIfHasBroken"`
}

type derivedNames struct {
	pvcName        string
	configMapName  string
	deploymentName string
	serviceName    string
	envoyService   string
	gatewayName    string
	envoyProxyName string
	tcpRouteName   string
	udpRouteName   string
}

func boolValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func gatewayBaseName(name string) string {
	if strings.HasSuffix(name, "-server") {
		return strings.TrimSuffix(name, "-server")
	}
	return name
}

func deriveNames(server *windrosev1alpha1.WindroseServer) derivedNames {
	base := gatewayBaseName(server.Name)
	names := derivedNames{
		pvcName:        server.Name + "-files",
		configMapName:  server.Name + "-config",
		deploymentName: server.Name,
		serviceName:    server.Name,
		envoyService:   server.Name + "-envoy",
		gatewayName:    base + "-gateway",
		envoyProxyName: "game-" + base + "-kubevip",
		tcpRouteName:   base + "-game-tcp",
		udpRouteName:   base + "-game-udp",
	}
	if server.Spec.Gateway.GatewayName != "" {
		names.gatewayName = server.Spec.Gateway.GatewayName
	}
	if server.Spec.Gateway.EnvoyProxyName != "" {
		names.envoyProxyName = server.Spec.Gateway.EnvoyProxyName
	}
	return names
}

func serverImage(spec windrosev1alpha1.WindroseServerSpec) string {
	if spec.ServerImage != "" {
		return spec.ServerImage
	}
	return defaultServerImage
}

func imagePullPolicy(spec windrosev1alpha1.WindroseServerSpec) corev1.PullPolicy {
	if spec.ImagePullPolicy != "" {
		return spec.ImagePullPolicy
	}
	return corev1.PullIfNotPresent
}

func gatewayClassName(spec windrosev1alpha1.WindroseServerSpec) string {
	if spec.Gateway.ClassName != "" {
		return spec.Gateway.ClassName
	}
	return defaultGatewayClassName
}

func externalTrafficPolicy(spec windrosev1alpha1.WindroseServerSpec) corev1.ServiceExternalTrafficPolicy {
	if spec.Gateway.ExternalTrafficPolicy != "" {
		return spec.Gateway.ExternalTrafficPolicy
	}
	return corev1.ServiceExternalTrafficPolicyCluster
}

func envoyExternalTrafficPolicy(spec windrosev1alpha1.WindroseServerSpec) egv1a1.ServiceExternalTrafficPolicy {
	if externalTrafficPolicy(spec) == corev1.ServiceExternalTrafficPolicyLocal {
		return egv1a1.ServiceExternalTrafficPolicyLocal
	}
	return egv1a1.ServiceExternalTrafficPolicyCluster
}

func directConnectionPort(spec windrosev1alpha1.WindroseServerSpec) int32 {
	if spec.DirectConnectionServerPort != 0 {
		return spec.DirectConnectionServerPort
	}
	return defaultDirectConnectionPort
}

func maxPlayers(spec windrosev1alpha1.WindroseServerSpec) int32 {
	if spec.MaxPlayerCount != 0 {
		return spec.MaxPlayerCount
	}
	return defaultMaxPlayers
}

func storageSize(spec windrosev1alpha1.WindroseServerSpec) string {
	if spec.StorageSize != "" {
		return spec.StorageSize
	}
	return defaultStorageSize
}

func directConnectionProxyAddress(spec windrosev1alpha1.WindroseServerSpec) string {
	if spec.DirectConnectionProxyAddress != "" {
		return spec.DirectConnectionProxyAddress
	}
	return defaultDirectConnectionProxyAddress
}

func p2pProxyAddress(spec windrosev1alpha1.WindroseServerSpec) string {
	if spec.P2pProxyAddress != "" {
		return spec.P2pProxyAddress
	}
	return defaultP2PProxyAddress
}

func defaultResources(spec windrosev1alpha1.WindroseServerSpec) corev1.ResourceRequirements {
	if spec.Resources.Requests != nil || spec.Resources.Limits != nil {
		return spec.Resources
	}
	return resourcesForPlayerCount(maxPlayers(spec))
}

// resourcesForPlayerCount returns CPU/memory based on Windrose dedicated server hardware guidance.
// See https://playwindrose.com/dedicated-server-guide
func resourcesForPlayerCount(count int32) corev1.ResourceRequirements {
	switch {
	case count <= 2:
		return podResources("2", "8Gi", "4", "10Gi")
	case count <= 4:
		return podResources("2", "12Gi", "4", "16Gi")
	default:
		return podResources("2", "16Gi", "4", "16Gi")
	}
}

func podResources(cpuRequest, memRequest, cpuLimit, memLimit string) corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resourceQuantity(cpuRequest),
			corev1.ResourceMemory: resourceQuantity(memRequest),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resourceQuantity(cpuLimit),
			corev1.ResourceMemory: resourceQuantity(memLimit),
		},
	}
}

func buildServerDescription(spec windrosev1alpha1.WindroseServerSpec) ([]byte, error) {
	password := spec.Password
	useDirect := boolValue(spec.UseDirectConnection, true)
	autoBackup := boolValue(spec.AutoLoadLatestBackupIfHasBroken, true)

	doc := serverDescriptionFile{
		Version: 1,
		ServerDescriptionPersistent: serverDescriptionPersistent{
			InviteCode:                      spec.InviteCode,
			IsPasswordProtected:             password != "",
			Password:                        password,
			ServerName:                      spec.ServerName,
			MaxPlayerCount:                  maxPlayers(spec),
			UserSelectedRegion:              spec.UserSelectedRegion,
			P2pProxyAddress:                 p2pProxyAddress(spec),
			UseDirectConnection:             useDirect,
			DirectConnectionServerPort:      directConnectionPort(spec),
			DirectConnectionProxyAddress:    directConnectionProxyAddress(spec),
			AutoLoadLatestBackupIfHasBroken: autoBackup,
		},
	}

	return json.MarshalIndent(doc, "", "  ")
}

func resourceQuantity(value string) resource.Quantity {
	return resource.MustParse(value)
}

func gameServicePorts(port int32) []corev1.ServicePort {
	return []corev1.ServicePort{
		{
			Name:       "game-tcp",
			Port:       port,
			TargetPort: intstr.FromInt32(port),
			Protocol:   corev1.ProtocolTCP,
		},
		{
			Name:       "game-udp",
			Port:       port,
			TargetPort: intstr.FromInt32(port),
			Protocol:   corev1.ProtocolUDP,
		},
	}
}
