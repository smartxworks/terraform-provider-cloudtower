package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestProvider(t *testing.T) {
	if err := New("test")().InternalValidate(); err != nil {
		t.Fatalf("provider schema is invalid: %v", err)
	}
}

func TestEnvListDefaultFunc(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  []string // nil when the default should stay unset
	}{
		{name: "empty value", value: "", want: nil},
		{name: "single item", value: "https://a.example.com", want: []string{"https://a.example.com"}},
		{name: "comma separated", value: "https://a.example.com,https://b.example.com", want: []string{"https://a.example.com", "https://b.example.com"}},
		{name: "whitespace trimmed", value: " https://a.example.com , https://b.example.com ", want: []string{"https://a.example.com", "https://b.example.com"}},
		{name: "blank items skipped", value: "https://a.example.com,, ,https://b.example.com", want: []string{"https://a.example.com", "https://b.example.com"}},
		{name: "only separators", value: " , ,", want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CLOUDTOWER_SERVERS", tt.value)
			got, err := envListDefaultFunc("CLOUDTOWER_SERVERS")()
			if err != nil {
				t.Fatalf("envListDefaultFunc returned error: %v", err)
			}
			if tt.want == nil {
				if got != nil {
					t.Fatalf("envListDefaultFunc() = %v, want nil", got)
				}
				return
			}
			list, ok := got.([]interface{})
			if !ok {
				t.Fatalf("envListDefaultFunc() = %T, want []interface{}", got)
			}
			if len(list) != len(tt.want) {
				t.Fatalf("envListDefaultFunc() = %v, want %v", list, tt.want)
			}
			for i := range tt.want {
				if list[i] != tt.want[i] {
					t.Fatalf("envListDefaultFunc() = %v, want %v", list, tt.want)
				}
			}
		})
	}
}

func TestConfigureServerSelection(t *testing.T) {
	// keep the environment out of the way so the tests only see raw config
	t.Setenv("CLOUDTOWER_SERVER", "")
	t.Setenv("CLOUDTOWER_SERVERS", "")

	p := New("test")()
	configure := p.ConfigureContextFunc

	t.Run("rejects both server attributes", func(t *testing.T) {
		// even when one of the two values only comes from an environment
		// variable, configuring both is ambiguous
		d := schema.TestResourceDataRaw(t, p.Schema, map[string]interface{}{
			"username":           "user",
			"password":           "pass",
			"user_source":        "LOCAL",
			"cloudtower_server":  "https://a.example.com",
			"cloudtower_servers": []interface{}{"https://b.example.com", "https://c.example.com"},
		})
		_, diags := configure(context.Background(), d)
		if !diags.HasError() {
			t.Fatal("configure() should reject setting both cloudtower_server and cloudtower_servers")
		}
		if msg := diags[0].Summary; !strings.Contains(msg, "only one of") {
			t.Fatalf("error = %q, want it to explain that only one attribute may be set", msg)
		}
	})

	t.Run("rejects missing server attributes", func(t *testing.T) {
		d := schema.TestResourceDataRaw(t, p.Schema, map[string]interface{}{
			"username":    "user",
			"password":    "pass",
			"user_source": "LOCAL",
		})
		_, diags := configure(context.Background(), d)
		if !diags.HasError() {
			t.Fatal("configure() should reject a missing server configuration")
		}
		if msg := diags[0].Summary; !strings.Contains(msg, "must be set") {
			t.Fatalf("error = %q, want it to explain that a server must be set", msg)
		}
	})
}
