package provider

import (
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testAccProtoV6ProviderFactories(serverURL string) map[string]func() (tfprotov6.ProviderServer, error) {
	// Set the API URL for tests
	os.Setenv("TOWEROPS_TEST_API_URL", serverURL)

	return map[string]func() (tfprotov6.ProviderServer, error){
		"towerops": providerserver.NewProtocol6WithError(New("test")()),
	}
}

func TestProvider_Schema(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(server.URL),
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig(server.URL),
			},
		},
	})
}

func TestProvider_MissingToken(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(""),
		Steps: []resource.TestStep{
			{
				Config: `
provider "towerops" {
  api_url = "http://localhost"
}

resource "towerops_site" "test" {
  name = "Test"
}
`,
				ExpectError: regexp.MustCompile(`(Missing TowerOps API Token|token.*is required)`),
			},
		},
	})
}

func TestProvider_EmptyToken(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(""),
		Steps: []resource.TestStep{
			{
				Config: `
provider "towerops" {
  token   = ""
  api_url = "http://localhost"
}

resource "towerops_site" "test" {
  name = "Test"
}
`,
				ExpectError: regexp.MustCompile(`Missing TowerOps API Token`),
			},
		},
	})
}

func testAccProviderConfig(apiURL string) string {
	return `
provider "towerops" {
  token   = "test-token"
  api_url = "` + apiURL + `"
}
`
}
