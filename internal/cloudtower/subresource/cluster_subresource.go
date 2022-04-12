package subresource

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

func ClusterSubResourceSchema() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		// VirtualDeviceFileBackingInfo
		"id": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The id of vm",
		},
		"path": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The path to the ISO file on the datastore.",
		},
		// VirtualCdromRemoteAtapiBackingInfo
		"client_device": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Indicates whether the device should be mapped to a remote client device",
		},
	}
	return s
}
