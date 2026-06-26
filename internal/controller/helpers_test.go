package controller

import (
	"testing"

	windrosev1alpha1 "github.com/DataKnifeAI/windrose-operator/api/v1alpha1"
)

func TestBuildServerDescriptionDirectConnection(t *testing.T) {
	useDirect := true
	spec := windrosev1alpha1.WindroseServerSpec{
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
