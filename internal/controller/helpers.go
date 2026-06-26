package controller

import (
	"encoding/json"

	windrosev1alpha1 "github.com/DataKnifeAI/windrose-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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

func boolValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func serverImage(spec windrosev1alpha1.WindroseServerSpec) string {
	if spec.ServerImage != "" {
		return spec.ServerImage
	}
	return defaultServerImage
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

func serviceType(spec windrosev1alpha1.WindroseServerSpec) corev1.ServiceType {
	if spec.ServiceType != "" {
		return spec.ServiceType
	}
	return corev1.ServiceTypeLoadBalancer
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
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resourceQuantity("2"),
			corev1.ResourceMemory: resourceQuantity("12Gi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resourceQuantity("4"),
			corev1.ResourceMemory: resourceQuantity("16Gi"),
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

func resourceNames(name string) (pvcName, configMapName, deploymentName, serviceName string) {
	return name + "-saves", name + "-config", name, name
}

func resourceQuantity(value string) resource.Quantity {
	return resource.MustParse(value)
}
