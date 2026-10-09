package provider

import (
	"context"

	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func init() {
	// Set descriptions to support markdown syntax, this will be used in document generation
	// and the language server.
	schema.DescriptionKind = schema.StringMarkdown

	// Customize the content of descriptions when output. For example you can add defaults on
	// to the exported descriptions if present.
	// schema.SchemaDescriptionBuilder = func(s *schema.Schema) string {
	// 	desc := s.Description
	// 	if s.Default != nil {
	// 		desc += fmt.Sprintf(" Defaults to `%v`.", s.Default)
	// 	}
	// 	return strings.TrimSpace(desc)
	// }
}

func New(version string) func() *schema.Provider {
	return func() *schema.Provider {
		p := &schema.Provider{
			Schema: map[string]*schema.Schema{
				"username": {
					Type:        schema.TypeString,
					Required:    true,
					DefaultFunc: schema.EnvDefaultFunc("CLOUDTOWER_USER", nil),
					Description: "The username for CloudTower API operations.",
				},
				"password": {
					Type:        schema.TypeString,
					Required:    true,
					DefaultFunc: schema.EnvDefaultFunc("CLOUDTOWER_PASSWORD", nil),
					Description: "The user password for CloudTower API operations.",
				},
				"user_source": {
					Type:        schema.TypeString,
					Required:    true,
					DefaultFunc: schema.EnvDefaultFunc("CLOUDTOWER_USER_SOURCE", nil),
					Description: "The source type of user",
				},
				"cloudtower_server": {
					Type:          schema.TypeString,
					Optional:      true,
					DefaultFunc:   schema.EnvDefaultFunc("CLOUDTOWER_SERVER", nil),
					ConflictsWith: []string{"cloudtower_servers"},
					Description:   "The CloudTower server address as a bare `host[:port]`, e.g. `tower.example.com`. For an HA CloudTower use `cloudtower_servers` instead.",
				},
				"cloudtower_servers": {
					Type:          schema.TypeList,
					Optional:      true,
					Elem:          &schema.Schema{Type: schema.TypeString},
					ConflictsWith: []string{"cloudtower_server"},
					Description: "The management endpoints of an HA CloudTower (4.9.0+, multiple management IPs in an active-passive deployment) as bare `host[:port]` addresses, " +
						"e.g. `[\"tower-a.example.com\", \"tower-b.example.com\"]`. " +
						"The provider probes the endpoints to find the active node and fails over automatically when the active node switches. Conflicts with `cloudtower_server`.",
				},
				"scheme": {
					Type:         schema.TypeString,
					Optional:     true,
					DefaultFunc:  schema.EnvDefaultFunc("CLOUDTOWER_SCHEME", "http"),
					ValidateFunc: validation.StringInSlice([]string{"http", "https"}, false),
					Description: "The URL scheme used to connect to CloudTower, either `http` or `https`. " +
						"Defaults to `http`. Set to `https` (optionally together with `skip_tls_verify`) " +
						"when the CloudTower endpoint serves TLS.",
				},
				"skip_tls_verify": {
					Type:        schema.TypeBool,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("CLOUDTOWER_SKIP_TLS_VERIFY", nil),
					Description: "Skip TLS certificate verification when connecting to CloudTower over HTTPS. " +
						"Useful for endpoints whose self-signed certificates cannot be verified, e.g. bare IP addresses. " +
						"Use with caution: the connection is no longer protected against man-in-the-middle attacks.",
				},
			},
			DataSourcesMap: map[string]*schema.Resource{
				"cloudtower_datacenter":                  dataSourceDatacenter(),
				"cloudtower_cluster":                     dataSourceCluster(),
				"cloudtower_vlan":                        dataSourceVlan(),
				"cloudtower_iso":                         dataSourceIso(),
				"cloudtower_svt_iso":                     dataSourceSvtImage(),
				"cloudtower_host":                        dataSourceHost(),
				"cloudtower_vm":                          dataSourceVm(),
				"cloudtower_vm_snapshot":                 dataSourceVmSnapshot(),
				"cloudtower_vm_template":                 dataSourceVmTemplate(),
				"cloudtower_content_library_vm_template": dataSourceContentLibraryVmTemplate(),
				"cloudtower_vds":                         dataSourceVds(),
			},
			ResourcesMap: map[string]*schema.Resource{
				"cloudtower_datacenter":                  resourceDatacenter(),
				"cloudtower_cluster":                     resourceCluster(),
				"cloudtower_vm_vlan":                     resourceVmVlan(),
				"cloudtower_vm":                          resourceVm(),
				"cloudtower_vm_snapshot":                 resourceVmSnapshot(),
				"cloudtower_vm_template":                 resourceVmTemplate(),
				"cloudtower_content_library_vm_template": resourceContentLibraryVmTemplate(),
			},
		}

		p.ConfigureContextFunc = configure(version, p)

		return p
	}
}

func configure(version string, p *schema.Provider) func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		var diags diag.Diagnostics

		username := d.Get("username").(string)
		password := d.Get("password").(string)
		usersource := d.Get("user_source").(string)
		server := d.Get("cloudtower_server").(string)
		servers := d.Get("cloudtower_servers").([]interface{})
		scheme := d.Get("scheme").(string)
		skipTLSVerify := d.Get("skip_tls_verify").(bool)
		var usource models.UserSource
		if usersource == "LDAP" {
			usource = models.UserSourceLDAP
		} else {
			usource = models.UserSourceLOCAL
		}

		var c *cloudtower.Client
		var err error
		if len(servers) > 0 {
			if server != "" {
				return nil, diag.Errorf("only one of cloudtower_server and cloudtower_servers can be set; cloudtower_server may also be set via the CLOUDTOWER_SERVER environment variable")
			}
			serverList := make([]string, 0, len(servers))
			for _, s := range servers {
				serverList = append(serverList, s.(string))
			}
			c, err = cloudtower.NewHAClient(ctx, serverList, username, password, usource, scheme, skipTLSVerify)
		} else {
			if server == "" {
				return nil, diag.Errorf("one of cloudtower_server and cloudtower_servers must be set; cloudtower_server may also be set via the CLOUDTOWER_SERVER environment variable")
			}
			c, err = cloudtower.NewClient(ctx, server, username, password, usource, scheme, skipTLSVerify)
		}

		if err != nil {
			return nil, diag.FromErr(err)
		}

		return c, diags
	}
}
