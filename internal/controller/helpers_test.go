package controller

import (
	"testing"

	windrosev1alpha1 "github.com/DataKnifeAI/windrose-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildServerDescriptionDirectConnection(t *testing.T) {
	useDirect := true
	spec := windrosev1alpha1.WindroseServerSpec{
		Gateway: windrosev1alpha1.GatewayConfig{Address: "192.168.14.186"},
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
