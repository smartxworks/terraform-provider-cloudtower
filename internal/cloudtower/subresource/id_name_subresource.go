package subresource

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

func IdNameSubresourceSchema() map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		// VirtualDeviceFileBackingInfo
		"datastore_id": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The datastore ID the ISO is located on.",
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