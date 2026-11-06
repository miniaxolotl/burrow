package protocol

import (
	"strings"
	"testing"
)

func TestRegistrationRejectsWrongIdentityAndOversize(t *testing.T) {
	for _, data := range []string{
		`{"version":1,"tunnel_id":"other","generation":"g"}`,
		`{"version":2,"tunnel_id":"id","generation":"g"}`,
		`{"version":1,"tunnel_id":"id","generation":""}`,
		strings.Repeat(" ", MaxRegistrationSize+1),
		`{"version":1,"tunnel_id":"id","generation":"g"}{}`,
	} {
		if _, err := ReadRegistration(strings.NewReader(data), "id"); err == nil {
			t.Fatalf("accepted %q", data)
		}
	}
}
