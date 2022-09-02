package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var (
	testAccProvider   *schema.Provider
	providerFactories map[string]func() (*schema.Provider, error)
)

func init() {
	testAccProvider = New("dev")()
	providerFactories = map[string]func() (*schema.Provider, error){
		"cloudtower": func() (*schema.Provider, error) {
			return New("dev")(), nil
		},
	}
}

func TestProvider(t *testing.T) {
	if err := New("dev")().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func generateRandomResourceName() string {
	return acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)
}

func testAccPreCheck(t *testing.T) {
	// check basic config of testing
	if v := os.Getenv("CLOUDTOWER_USER"); v == "" {
		t.Fatal("CLOUDTOWER_USER must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDTOWER_PASSWORD"); v == "" {
		t.Fatal("CLOUDTOWER_PASSWORD must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDTOWER_USER_SOURCE"); v == "" {
		t.Fatal("CLOUDTOWER_USER must be set for acceptance tests")
	}
	if v := os.Getenv("CLOUDTOWER_SERVER"); v == "" {
		t.Fatal("CLOUDTOWER_SERVER must be set for acceptance tests")
	}
}
