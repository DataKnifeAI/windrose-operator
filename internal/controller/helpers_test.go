package controller

import (
	"testing"

	windrosev1alpha1 "github.com/DataKnifeAI/windrose-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildServerDescriptionDirectConnection(t *testing.T) {
	useDirect := true
	spec := windrosev1alpha1.WindroseServerSpec{
		Gateway:                    windrosev1alpha1.GatewayConfig{Address: "192.168.14.186"},
		ServerName:                 "Test Server",
		MaxPlayerCount:             6,
		UseDirectConnection:        &useDirect,
		DirectConnectionServerPort: 17777,
		Password:                   "secret",
	}

	content, err := buildServerDescription(spec)
	if err != nil {
		t.Fatalf("buildServerDescription() error = %v", err)
	}

	body := string(content)
	for _, want := range []string{
		`"UseDirectConnection": true`,
		`"DirectConnectionServerPort": 17777`,
		`"IsPasswordProtected": true`,
		`"ServerName": "Test Server"`,
	} {
		if !contains(body, want) {
			t.Fatalf("expected %q in %s", want, body)
		}
	}
}

func TestResourcesForPlayerCount(t *testing.T) {
	tests := []struct {
		players  int32
		memReq   string
		memLimit string
	}{
		{players: 2, memReq: "8Gi", memLimit: "10Gi"},
		{players: 4, memReq: "12Gi", memLimit: "16Gi"},
		{players: 10, memReq: "16Gi", memLimit: "16Gi"},
	}

	for _, tt := range tests {
		resources := resourcesForPlayerCount(tt.players)
		if got := resources.Requests.Memory().String(); got != tt.memReq {
			t.Fatalf("players=%d memory request = %s, want %s", tt.players, got, tt.memReq)
		}
		if got := resources.Limits.Memory().String(); got != tt.memLimit {
			t.Fatalf("players=%d memory limit = %s, want %s", tt.players, got, tt.memLimit)
		}
	}
}

func TestDefaultResourcesAutoSelectAndOverride(t *testing.T) {
	auto := defaultResources(windrosev1alpha1.WindroseServerSpec{
		Gateway:        windrosev1alpha1.GatewayConfig{Address: "192.168.14.186"},
		MaxPlayerCount: 2,
	})
	if got := auto.Requests.Memory().String(); got != "8Gi" {
		t.Fatalf("auto-selected memory = %s, want 8Gi", got)
	}

	override := defaultResources(windrosev1alpha1.WindroseServerSpec{
		Gateway:        windrosev1alpha1.GatewayConfig{Address: "192.168.14.186"},
		MaxPlayerCount: 2,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resourceQuantity("1"),
				corev1.ResourceMemory: resourceQuantity("4Gi"),
			},
		},
	})
	if got := override.Requests.Memory().String(); got != "4Gi" {
		t.Fatalf("override memory = %s, want 4Gi", got)
	}
}

func TestDeriveNamesWindroseServer(t *testing.T) {
	server := &windrosev1alpha1.WindroseServer{
		ObjectMeta: metav1.ObjectMeta{Name: "windrose-server"},
		Spec: windrosev1alpha1.WindroseServerSpec{
			Gateway: windrosev1alpha1.GatewayConfig{Address: "192.168.14.186"},
		},
	}

	names := deriveNames(server)
	checks := map[string]string{
		names.pvcName:        "windrose-server-files",
		names.envoyService:   "windrose-server-envoy",
		names.gatewayName:    "windrose-gateway",
		names.envoyProxyName: "game-windrose-kubevip",
		names.tcpRouteName:   "windrose-game-tcp",
		names.udpRouteName:   "windrose-game-udp",
	}
	for got, want := range checks {
		if got != want {
			t.Fatalf("deriveNames() = %q, want %q", got, want)
		}
	}
}

func contains(haystack, needle string) bool {
	return len(needle) == 0 || (len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
