package provider

import (
	"context"

	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("CLOUDTOWER_SERVER", nil),
					Description: "The CloudTower Server name.",
				},
			},
			DataSourcesMap: map[string]*schema.Resource{
				"cloudtower_datacenter":  dataSourceDatacenter(),
				"cloudtower_cluster":     dataSourceCluster(),
				"cloudtower_vlan":        dataSourceVlan(),
				"cloudtower_iso":         dataSourceIso(),
				"cloudtower_host":        dataSourceHost(),
				"cloudtower_vm":          dataSourceVm(),
				"cloudtower_vm_snapshot": dataSourceVmSnapshot(),
			},
			ResourcesMap: map[string]*schema.Resource{
				"cloudtower_datacenter":  resourceDatacenter(),
				"cloudtower_cluster":     resourceCluster(),
				"cloudtower_vm":          resourceVm(),
				"cloudtower_vm_snapshot": resourceVmSnapshot(),
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
		var usource models.UserSource
		if usersource == "LDAP" {
			usource = models.UserSourceLDAP
		} else {
			usource = models.UserSourceLOCAL
		}
		c, err := cloudtower.NewClient(server, username, password, usource)

		if err != nil {
			return nil, diag.FromErr(err)
		}

		return c, diags
	}
}
