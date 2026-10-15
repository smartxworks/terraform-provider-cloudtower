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

func TestConfigureServerSelection(t *testing.T) {
	p := New("test")()
	configure := p.ConfigureContextFunc

	t.Run("rejects both server attributes", func(t *testing.T) {
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
