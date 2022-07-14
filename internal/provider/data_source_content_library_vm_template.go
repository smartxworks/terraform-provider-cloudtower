package provider

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/terraform-provider-cloudtower/internal/cloudtower"
	"github.com/hashicorp/terraform-provider-cloudtower/internal/helper"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/content_library_vm_template"
	"github.com/smartxworks/cloudtower-go-sdk/v2/client/vm_template"
	"github.com/smartxworks/cloudtower-go-sdk/v2/models"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceContentLibraryVmTemplate() *schema.Resource {
	return &schema.Resource{
		Description: "CloudTower content library vm template data source.",

		ReadContext: dataSourceContentLibraryVmTemplateRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name_in"},
				Description:   "content library vm template's name",
			},
			"name_in": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name"},
				Description:   "content library vm template's name as an array",
				Elem:          &schema.Schema{Type: schema.TypeString},
			},
			"name_contains": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "filter content_library vm template by its name contains characters",
			},
			"cluster_in": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "the cluster id which template has already distributed to.",
			},
			"content_library_vm_templates": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "list of queried vm templates",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "template's id",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "template's name",
						},
						"create_time": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "template's create_time",
						},
						"vm_templates": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "content library vm template's origin vm_templates' id",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"id": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "vm_template's id",
									},
									"cluster": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "the cluster template store at",
									},
									"disks": {
										Type: schema.TypeList,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"boot": {
													Type:        schema.TypeInt,
													Computed:    true,
													Description: "disk's boot order",
												},
												"bus": {
													Type:        schema.TypeString,
													Computed:    true,
													Description: "disk's bus",
												},
												"storage_policy": {
													Type:        schema.TypeString,
													Computed:    true,
													Description: "vm volume's storage policy",
												},
												"name": {
													Type:        schema.TypeString,
													Computed:    true,
													Description: "vm volume's name",
												},
												"size": {
													Type:        schema.TypeFloat,
													Computed:    true,
													Description: "vm volume's size, in the unit of byte",
												},
												"path": {
													Type:        schema.TypeString,
													Computed:    true,
													Description: "vm volume's iscsi LUN path",
												},
											},
										},
										Computed:    true,
										Description: "template's disks",
									},
									"nics": {
										Type: schema.TypeList,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"vlan_id": {
													Type:        schema.TypeString,
													Required:    true,
													Description: "specific the vlan's id the template nic is using",
												},
												"enabled": {
													Type:        schema.TypeBool,
													Description: "whether the template nic is enabled",
													Computed:    true,
												},
												"mirror": {
													Type:        schema.TypeBool,
													Description: "whether the template nic use mirror mode",
													Computed:    true,
												},
												"model": {
													Type:        schema.TypeString,
													Description: "template nic's model",
													Computed:    true,
												},
												"idx": {
													Type:        schema.TypeInt,
													Description: "template nic's index",
													Computed:    true,
												},
											},
										},
										Computed:    true,
										Description: "template's nics",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceContentLibraryVmTemplateRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	ct := meta.(*cloudtower.Client)

	gp := content_library_vm_template.NewGetContentLibraryVMTemplatesParams()
	gp.RequestBody = &models.GetContentLibraryVMTemplatesRequestBody{
		Where:   &models.ContentLibraryVMTemplateWhereInput{},
		OrderBy: models.ContentLibraryVMTemplateOrderByInputCreatedAtDESC.Pointer(),
	}
	if name := d.Get("name").(string); name != "" {
		gp.RequestBody.Where.Name = &name
	} else {
		nameIn, err := helper.SliceInterfacesToTypeSlice[string](d.Get("name_in").([]interface{}))
		if err != nil {
			return diag.FromErr(err)
		} else if len(nameIn) > 0 {
			gp.RequestBody.Where.NameIn = nameIn
		}
	}

	if nameContains := d.Get("name_contains").(string); nameContains != "" {
		gp.RequestBody.Where.NameContains = &nameContains
	}

	raw_cluster_id, ok := d.GetOk("cluster_id")
	if ok {
		bytes, err := json.Marshal(raw_cluster_id)
		if err != nil {
			return diag.FromErr(err)
		}
		cluster_ids := make([]string, 0)
		err = json.Unmarshal(bytes, &cluster_ids)
		if err != nil {
			return diag.FromErr(err)
		}
		gp.RequestBody.Where.VMTemplatesSome = &models.VMTemplateWhereInput{
			Cluster: &models.ClusterWhereInput{
				IDIn: cluster_ids,
			},
		}
	}
	vm_templates, err := ct.Api.ContentLibraryVMTemplate.GetContentLibraryVMTemplates(gp)
	if err != nil {
		return diag.FromErr(err)
	}
	output := make([]map[string]interface{}, 0)
	var err_channel chan error = make(chan error, len(vm_templates.Payload))
	for _, d := range vm_templates.Payload {
		var vm_templates []map[string]interface{} = make([]map[string]interface{}, 0)
		var wg sync.WaitGroup
		for _, template := range d.VMTemplates {
			wg.Add(1)
			go func(t *models.NestedVMTemplate) {
				defer wg.Done()
				var disks []map[string]interface{} = make([]map[string]interface{}, 0)
				var nics []map[string]interface{} = make([]map[string]interface{}, 0)
				gvtp := vm_template.NewGetVMTemplatesParams()
				gvtp.RequestBody = &models.GetVMTemplatesRequestBody{
					Where: &models.VMTemplateWhereInput{
						ID: t.ID,
					},
				}
				resp, err := ct.Api.VMTemplate.GetVMTemplates(gvtp)
				if err != nil {
					err_channel <- err
				}
				rawTemplate := resp.Payload[0]
				for _, disk := range rawTemplate.VMDisks {
					if *disk.Type == models.VMDiskTypeDISK {
						storagePolicy, err := helper.GetElfStoragePolicyByLocalId(ct.Api, *disk.StoragePolicyUUID)
						if err != nil {
							err_channel <- err
							continue
						}
						disks = append(disks, map[string]interface{}{
							"boot":           disk.Boot,
							"bus":            disk.Bus,
							"storage_policy": storagePolicy,
							"name":           disk.DiskName,
							"size":           disk.Size,
							"path":           disk.Path,
						})
					}
					// ignore cd-rom in content library vm template
				}
				for _, nic := range rawTemplate.VMNics {
					nicVlan, err := helper.GetVlanFromLocalId(ct.Api, *nic.Vlan.VlanLocalID)
					if err != nil {
						err_channel <- err
						continue
					}
					nics = append(nics, map[string]interface{}{
						"vlan_id": nicVlan.ID,
						"enabled": nic.Enabled,
						"mirror":  nic.Mirror,
						"model":   nic.Model,
						"idx":     nic.Index,
					})
				}
				vm_templates = append(vm_templates, map[string]interface{}{
					"disks":   disks,
					"nics":    nics,
					"id":      rawTemplate.ID,
					"cluster": rawTemplate.Cluster.ID,
				})
			}(template)
		}
		wg.Wait()
		close(err_channel)
		for err := range err_channel {
			if err != nil {
				diags = append(diags, diag.FromErr(err)...)
			}
		}
		output = append(output, map[string]interface{}{
			"id":           d.ID,
			"name":         d.Name,
			"create_time":  d.CreatedAt,
			"vm_templates": vm_templates,
		})
	}
	if diags.HasError() {
		return diags
	}
	err = d.Set("content_library_vm_templates", output)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return diags
}
