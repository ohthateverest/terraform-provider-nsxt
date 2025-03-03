/* Copyright © 2023 VMware, Inc. All Rights Reserved.
   SPDX-License-Identifier: MPL-2.0 */

package nsxt

import (
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/vmware/terraform-provider-nsxt/nsxt/util"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/vmware/vsphere-automation-sdk-go/runtime/bindings"
	"github.com/vmware/vsphere-automation-sdk-go/runtime/data"
	"github.com/vmware/vsphere-automation-sdk-go/runtime/protocol/client"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt-mp/nsx"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt-mp/nsx/model"
	"github.com/vmware/vsphere-automation-sdk-go/services/nsxt-mp/nsx/transport_nodes"
	"golang.org/x/exp/maps"
)

var ipAssignmentTypes = []string{
	"assigned_by_dhcp",
	"static_ip",
	"static_ip_pool",
}

const nodeTypeEdge = "EdgeNode"
const nodeTypeHost = "HostNode"

var hostSwitchModeValues = []string{
	model.StandardHostSwitch_HOST_SWITCH_MODE_STANDARD,
	model.StandardHostSwitch_HOST_SWITCH_MODE_ENS,
	model.StandardHostSwitch_HOST_SWITCH_MODE_ENS_INTERRUPT,
	model.StandardHostSwitch_HOST_SWITCH_MODE_LEGACY,
}

var edgeNodeFormFactorValues = []string{
	model.EdgeNodeDeploymentConfig_FORM_FACTOR_SMALL,
	model.EdgeNodeDeploymentConfig_FORM_FACTOR_MEDIUM,
	model.EdgeNodeDeploymentConfig_FORM_FACTOR_LARGE,
	model.EdgeNodeDeploymentConfig_FORM_FACTOR_XLARGE,
}

var cpuReservationValues = []string{
	model.CPUReservation_RESERVATION_IN_SHARES_EXTRA_HIGH_PRIORITY,
	model.CPUReservation_RESERVATION_IN_SHARES_HIGH_PRIORITY,
	model.CPUReservation_RESERVATION_IN_SHARES_NORMAL_PRIORITY,
	model.CPUReservation_RESERVATION_IN_SHARES_LOW_PRIORITY,
}

var syslogLogLevelValues = []string{
	model.SyslogConfiguration_LOG_LEVEL_EMERGENCY,
	model.SyslogConfiguration_LOG_LEVEL_ALERT,
	model.SyslogConfiguration_LOG_LEVEL_CRITICAL,
	model.SyslogConfiguration_LOG_LEVEL_ERROR,
	model.SyslogConfiguration_LOG_LEVEL_WARNING,
	model.SyslogConfiguration_LOG_LEVEL_NOTICE,
	model.SyslogConfiguration_LOG_LEVEL_INFO,
	model.SyslogConfiguration_LOG_LEVEL_DEBUG,
}

var syslogProtocolValues = []string{
	model.SyslogConfiguration_PROTOCOL_TCP,
	model.SyslogConfiguration_PROTOCOL_UDP,
	model.SyslogConfiguration_PROTOCOL_TLS,
	model.SyslogConfiguration_PROTOCOL_LI,
	model.SyslogConfiguration_PROTOCOL_LI_TLS,
}

func resourceNsxtEdgeTransportNode() *schema.Resource {
	return &schema.Resource{
		Create: resourceNsxtEdgeTransportNodeCreate,
		Read:   resourceNsxtEdgeTransportNodeRead,
		Update: resourceNsxtEdgeTransportNodeUpdate,
		Delete: resourceNsxtEdgeTransportNodeDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"revision":     getRevisionSchema(),
			"description":  getDescriptionSchema(),
			"display_name": getDisplayNameSchema(),
			"tag":          getTagsSchema(),
			"failure_domain": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Id of the failure domain",
			},
			// host_switch_spec
			"standard_host_switch": getStandardHostSwitchSchema(nodeTypeEdge),

			// node_deployment_info
			"deployment_config": getEdgeNodeDeploymentConfigSchema(),
			"node_settings":     getEdgeNodeSettingsSchema(),
			"external_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "ID of the Node",
			},
			"fqdn": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Fully qualified domain name of the fabric node",
			},
			"ip_addresses": {
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Description: "IP Addresses of the Node, version 4 or 6",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func getEdgeNodeDeploymentConfigSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		MaxItems:    1,
		Optional:    true,
		Description: "Config for automatic deployment of edge node virtual machine",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"form_factor": {
					Type:         schema.TypeString,
					Default:      model.EdgeNodeDeploymentConfig_FORM_FACTOR_MEDIUM,
					Optional:     true,
					ValidateFunc: validation.StringInSlice(edgeNodeFormFactorValues, false),
				},
				"node_user_settings": {
					Type:        schema.TypeList,
					MaxItems:    1,
					Required:    true,
					Description: "Node user settings",
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"audit_password": {
								Type:        schema.TypeString,
								Optional:    true,
								Sensitive:   true,
								Description: "Node audit user password",
							},
							"audit_username": {
								Type:        schema.TypeString,
								Optional:    true,
								Description: "CLI \"audit\" username",
							},
							"cli_password": {
								Type:        schema.TypeString,
								Required:    true,
								Sensitive:   true,
								Description: "Node cli password",
							},
							"cli_username": {
								Type:        schema.TypeString,
								Optional:    true,
								Default:     "admin",
								Description: "CLI \"admin\" username",
							},
							"root_password": {
								Type:        schema.TypeString,
								Required:    true,
								Sensitive:   true,
								Description: "Node root user password",
							},
						},
					},
				},
				"vm_deployment_config": {
					Type:        schema.TypeList,
					MaxItems:    1,
					Required:    true,
					Description: "The vSphere deployment configuration determines where to deploy the edge node",
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"compute_folder_id": {
								Type:        schema.TypeString,
								Optional:    true,
								Description: "Compute folder identifier in the specified vcenter server",
							},
							"compute_id": {
								Type:        schema.TypeString,
								Required:    true,
								Description: "Cluster identifier or resourcepool identifier for specified vcenter server",
							},
							"data_network_ids": {
								Type:        schema.TypeList,
								MinItems:    1,
								MaxItems:    4,
								Required:    true,
								Description: "List of portgroups, logical switch identifiers or segment paths for datapath connectivity",
								Elem: &schema.Schema{
									Type: schema.TypeString,
								},
							},
							"default_gateway_address": {
								Type:        schema.TypeList,
								MaxItems:    2,
								Optional:    true,
								Computed:    true,
								Description: "Default gateway for the node",
								Elem: &schema.Schema{
									Type:         schema.TypeString,
									ValidateFunc: validateSingleIP(),
								},
							},
							"host_id": {
								Type:        schema.TypeString,
								Optional:    true,
								Description: "Host identifier in the specified vcenter server",
							},
							"management_network_id": {
								Type:        schema.TypeString,
								Required:    true,
								Description: "Portgroup, logical switch identifier or segment path for management network connectivity",
							},
							"management_port_subnet": {
								Type:        schema.TypeList,
								Optional:    true,
								Description: "Port subnets for management port. IPv4, IPv6 and Dual Stack Address is supported",
								MinItems:    1,
								MaxItems:    2,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"ip_addresses": {
											Type:        schema.TypeList,
											Required:    true,
											Description: "IP Addresses",
											Elem: &schema.Schema{
												Type:         schema.TypeString,
												ValidateFunc: validateSingleIP(),
											},
										},
										"prefix_length": {
											Type:         schema.TypeInt,
											Required:     true,
											Description:  "Subnet Prefix Length",
											ValidateFunc: validation.IntBetween(1, 128),
										},
									},
								},
							},
							"reservation_info": {
								Type:        schema.TypeList,
								MaxItems:    1,
								Description: "Resource reservation settings",
								Optional:    true,
								Computed:    true,
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"cpu_reservation_in_mhz": {
											Type:        schema.TypeInt,
											Description: "CPU reservation in MHz",
											Optional:    true,
										},
										"cpu_reservation_in_shares": {
											Type:         schema.TypeString,
											Description:  "CPU reservation in shares",
											Optional:     true,
											Default:      model.CPUReservation_RESERVATION_IN_SHARES_HIGH_PRIORITY,
											ValidateFunc: validation.StringInSlice(cpuReservationValues, false),
										},
										"memory_reservation_percentage": {
											Type:         schema.TypeInt,
											Optional:     true,
											Description:  "Memory reservation percentage",
											ValidateFunc: validation.IntBetween(0, 100),
											Default:      100,
										},
									},
								},
							},
							"storage_id": {
								Type:        schema.TypeString,
								Required:    true,
								Description: "Storage/datastore identifier in the specified vcenter server",
							},
							"vc_id": {
								Type:        schema.TypeString,
								Description: "Vsphere compute identifier for identifying the vcenter server",
								Required:    true,
							},
						},
					},
				},
			},
		},
	}
}

func getEdgeNodeSettingsSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		MaxItems:    1,
		Description: "Current configuration on edge node",
		Required:    true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"advanced_configuration": getKeyValuePairListSchema(),
				"allow_ssh_root_login": {
					Type:        schema.TypeBool,
					Default:     false,
					Description: "Allow root SSH logins",
					Optional:    true,
				},
				"dns_servers": {
					Type:        schema.TypeList,
					Optional:    true,
					Description: "DNS servers",
					Elem: &schema.Schema{
						Type:         schema.TypeString,
						ValidateFunc: validateSingleIP(),
					},
				},
				"enable_ssh": {
					Type:        schema.TypeBool,
					Default:     false,
					Description: "Enable SSH",
					Optional:    true,
				},
				"enable_upt_mode": {
					Type:        schema.TypeBool,
					Default:     false,
					Description: "Enable Uniform Passthrough mode",
					Optional:    true,
				},
				"hostname": {
					Type:        schema.TypeString,
					Description: "Host name or FQDN for edge node",
					Required:    true,
				},
				"ntp_servers": {
					Type:        schema.TypeList,
					Optional:    true,
					Description: "NTP servers",
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
				"search_domains": {
					Type:        schema.TypeList,
					Optional:    true,
					Description: "Search domain names",
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
				"syslog_server": {
					Type:        schema.TypeList,
					MaxItems:    5,
					Optional:    true,
					Description: "Syslog servers",
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"log_level": {
								Type:         schema.TypeString,
								Optional:     true,
								Description:  "Log level to be redirected",
								Default:      model.SyslogConfiguration_LOG_LEVEL_INFO,
								ValidateFunc: validation.StringInSlice(syslogLogLevelValues, false),
							},
							"port": {
								Type:     schema.TypeString,
								Optional: true,
								Default:  "514",
								// Spec defines port-or-range format here, however range
								// is not accepted by NSX
								ValidateFunc: validateSinglePort(),
								Description:  "Syslog server port",
							},
							"protocol": {
								Type:         schema.TypeString,
								Optional:     true,
								Description:  "Syslog protocol",
								Default:      model.SyslogConfiguration_PROTOCOL_UDP,
								ValidateFunc: validation.StringInSlice(syslogProtocolValues, false),
							},
							"server": {
								Type:        schema.TypeString,
								Required:    true,
								Description: "Server IP or fqdn",
							},
						},
					},
				},
			},
		},
	}
}

func getStandardHostSwitchSchema(nodeType string) *schema.Schema {
	s := schema.Schema{
		Type:        schema.TypeList,
		Required:    true,
		Description: "Standard host switch specification",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"host_switch_id": {
					Type:        schema.TypeString,
					Description: "The host switch id. This ID will be used to reference a host switch",
					Optional:    true,
					Computed:    true,
				},
				"host_switch_name": {
					Type:        schema.TypeString,
					Description: "Host switch name. This name will be used to reference a host switch",
					Optional:    true,
					Computed:    true,
				},
				"host_switch_profile": getHostSwitchProfileIDsSchema(),
				"ip_assignment":       getIPAssignmentSchema(true),
				"pnic": {
					Type:        schema.TypeList,
					Optional:    true,
					Description: "Physical NICs connected to the host switch",
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"device_name": {
								Type:        schema.TypeString,
								Description: "Device name or key",
								Required:    true,
							},
							"uplink_name": {
								Type:        schema.TypeString,
								Description: "Uplink name for this Pnic",
								Required:    true,
							},
						},
					},
				},
				"transport_zone_endpoint": getTransportZoneEndpointSchema(),
			},
		},
	}
	if nodeType == nodeTypeHost {
		elemSchema := s.Elem.(*schema.Resource).Schema
		maps.Copy(elemSchema, map[string]*schema.Schema{
			"cpu_config": {
				Type:        schema.TypeList,
				Description: "Enhanced Networking Stack enabled HostSwitch CPU configuration",
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"num_lcores": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Number of Logical cpu cores (Lcores) to be placed on a specified NUMA node",
						},
						"numa_node_index": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Unique index of the Non Uniform Memory Access (NUMA) node",
						},
					},
				},
			},
			"host_switch_mode": {
				Type:         schema.TypeString,
				Description:  "Operational mode of a HostSwitch",
				Computed:     true,
				ValidateFunc: validation.StringInSlice(hostSwitchModeValues, false),
			},
			"is_migrate_pnics": {
				Type:        schema.TypeBool,
				Description: "Migrate any pnics which are in use",
				Optional:    true,
			},
			"transport_node_profile_sub_config": {
				Type:        schema.TypeList,
				MaxItems:    16,
				Optional:    true,
				Description: "Transport Node Profile sub-configuration Options",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"host_switch_config_option": {
							Type:        schema.TypeList,
							MaxItems:    1,
							Required:    true,
							Description: "Subset of the host switch configuration",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"host_switch_id": {
										Type:        schema.TypeString,
										Optional:    true,
										Description: "The host switch id. This ID will be used to reference a host switch",
									},
									"host_switch_profile": getHostSwitchProfileIDsSchema(),
									"ip_assignment":       getIPAssignmentSchema(false),
									"uplink":              getUplinksSchema(),
								},
							},
						},
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Name of the transport node profile config option",
						},
					},
				},
			},
			"uplink": getUplinksSchema(),
			"vmk_install_migration": {
				Type:        schema.TypeList,
				Description: "The vmknic and logical switch mappings",
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"destination_network": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The network id to which the ESX vmk interface will be migrated",
						},
						"device_name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "ESX vmk interface name",
						},
					},
				},
			},
		})
	}
	return &s
}

func getTransportZoneEndpointSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		Description: "Transport zone endpoints",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"transport_zone": {
					Type:         schema.TypeString,
					Required:     true,
					Description:  "Unique ID identifying the transport zone for this endpoint",
					ValidateFunc: validation.StringIsNotWhiteSpace,
				},
				"transport_zone_profiles": {
					Type:        schema.TypeList,
					Optional:    true,
					Computed:    true,
					Description: "Identifiers of the transport zone profiles associated with this transport zone endpoint on this transport node",
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
			},
		},
	}
}

func getUplinksSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		Description: "Uplink/LAG of VMware vSphere Distributed Switch connected to the HostSwitch",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"uplink_name": {
					Type:         schema.TypeString,
					Required:     true,
					Description:  "Uplink name from UplinkHostSwitch profile",
					ValidateFunc: validation.StringIsNotWhiteSpace,
				},
				"vds_lag_name": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Link Aggregation Group (LAG) name of Virtual Distributed Switch",
				},
				"vds_uplink_name": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Uplink name of VMware vSphere Distributed Switch (VDS)",
				},
			},
		},
	}
}

func getHostSwitchProfileIDsSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Description: "Identifiers of host switch profiles to be associated with this host switch",
		Optional:    true,
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
	}
}

func getIPAssignmentSchema(required bool) *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Description: "Specification for IPs to be used with host switch virtual tunnel endpoints",
		MaxItems:    1,
		Required:    required,
		Optional:    !required,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"assigned_by_dhcp": {
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     false,
					Description: "Enables DHCP assignment",
				},
				"static_ip": {
					Type:        schema.TypeList,
					MaxItems:    1,
					Optional:    true,
					Description: "IP assignment specification for Static IP List.",
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"default_gateway": {
								Type:         schema.TypeString,
								Required:     true,
								Description:  "Gateway IP",
								ValidateFunc: validateSingleIP(),
							},
							"ip_addresses": {
								Type:        schema.TypeList,
								Description: "List of IPs for transport node host switch virtual tunnel endpoints",
								MinItems:    1,
								Required:    true,
								Elem: &schema.Schema{
									Type:         schema.TypeString,
									ValidateFunc: validateSingleIP(),
								},
							},
							"subnet_mask": {
								Type:         schema.TypeString,
								Required:     true,
								Description:  "Subnet mask",
								ValidateFunc: validateSingleIP(),
							},
						},
					},
				},
				"static_ip_pool": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "IP assignment specification for Static IP Pool",
				},
			},
		},
	}
}

func getTransportNodeFromSchema(d *schema.ResourceData) (*model.TransportNode, error) {
	description := d.Get("description").(string)
	displayName := d.Get("display_name").(string)
	tags := getMPTagsFromSchema(d)
	failureDomain := d.Get("failure_domain").(string)
	hostSwitchSpec, err := getHostSwitchSpecFromSchema(d, nodeTypeEdge)
	if err != nil {
		return nil, fmt.Errorf("failed to create Transport Node: %v", err)
	}

	converter := bindings.NewTypeConverter()
	var dataValue data.DataValue
	var errs []error

	externalID := d.Get("external_id").(string)
	fqdn := d.Get("fqdn").(string)
	ipAddresses := interfaceListToStringList(d.Get("ip_addresses").([]interface{}))

	deploymentConfig, err := getEdgeNodeDeploymentConfigFromSchema(d.Get("deployment_config"))
	if err != nil {
		return nil, err
	}
	nodeSettings, err := getEdgeNodeSettingsFromSchema(d.Get("node_settings"))
	if err != nil {
		return nil, err
	}
	node := model.EdgeNode{
		ExternalId:       &externalID,
		Fqdn:             &fqdn,
		IpAddresses:      ipAddresses,
		DeploymentConfig: deploymentConfig,
		NodeSettings:     nodeSettings,
		ResourceType:     model.EdgeNode__TYPE_IDENTIFIER,
	}
	dataValue, errs = converter.ConvertToVapi(node, model.EdgeNodeBindingType())

	if errs != nil {
		log.Printf("Failed to convert node object, errors are %v", errs)
		return nil, errs[0]
	}
	nodeDeploymentInfo := dataValue.(*data.StructValue)

	obj := model.TransportNode{
		Description:        &description,
		DisplayName:        &displayName,
		Tags:               tags,
		FailureDomainId:    &failureDomain,
		HostSwitchSpec:     hostSwitchSpec,
		NodeDeploymentInfo: nodeDeploymentInfo,
	}

	return &obj, nil
}

func resourceNsxtEdgeTransportNodeCreate(d *schema.ResourceData, m interface{}) error {
	connector := getPolicyConnector(m)
	client := nsx.NewTransportNodesClient(connector)

	obj, err := getTransportNodeFromSchema(d)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Creating Transport Node with name %s", *obj.DisplayName)

	obj1, err := client.Create(*obj)
	if err != nil {
		return handleCreateError("TransportNode", *obj.DisplayName, err)
	}

	d.SetId(*obj1.Id)
	return resourceNsxtEdgeTransportNodeRead(d, m)
}

func getEdgeNodeDeploymentConfigFromSchema(cfg interface{}) (*model.EdgeNodeDeploymentConfig, error) {
	converter := bindings.NewTypeConverter()

	if cfg == nil {
		return nil, nil
	}
	for _, ci := range cfg.([]interface{}) {
		c := ci.(map[string]interface{})
		formFactor := c["form_factor"].(string)
		var nodeUserSettings *model.NodeUserSettings
		if c["node_user_settings"] != nil {
			for _, nusi := range c["node_user_settings"].([]interface{}) {
				nus := nusi.(map[string]interface{})
				auditPassword := nus["audit_password"].(string)
				auditUsername := nus["audit_username"].(string)
				cliPassword := nus["cli_password"].(string)
				cliUsername := nus["cli_username"].(string)
				rootPassword := nus["root_password"].(string)

				nodeUserSettings = &model.NodeUserSettings{
					CliPassword:  &cliPassword,
					CliUsername:  &cliUsername,
					RootPassword: &rootPassword,
				}
				if auditUsername != "" {
					nodeUserSettings.AuditUsername = &auditUsername
				}
				if auditPassword != "" {
					nodeUserSettings.AuditPassword = &auditPassword
				}
			}
		}
		var vmDeploymentConfig *data.StructValue
		for _, vdci := range c["vm_deployment_config"].([]interface{}) {
			vdc := vdci.(map[string]interface{})
			computeFolderID := vdc["compute_folder_id"].(string)
			computeID := vdc["compute_id"].(string)
			dataNetworkIds := interface2StringList(vdc["data_network_ids"].([]interface{}))
			defaultGatewayAddresses := interface2StringList(vdc["default_gateway_address"].([]interface{}))
			hostID := vdc["host_id"].(string)
			managementNetworkID := vdc["management_network_id"].(string)
			var managemenPortSubnets []model.IPSubnet
			for _, ipsi := range vdc["management_port_subnet"].([]interface{}) {
				ips := ipsi.(map[string]interface{})
				ipAddresses := interface2StringList(ips["ip_addresses"].([]interface{}))
				prefixLength := int64(ips["prefix_length"].(int))
				subnet := model.IPSubnet{
					IpAddresses:  ipAddresses,
					PrefixLength: &prefixLength,
				}
				managemenPortSubnets = append(managemenPortSubnets, subnet)
			}
			var reservationInfo *model.ReservationInfo
			for _, ri := range vdc["reservation_info"].([]interface{}) {
				rInfo := ri.(map[string]interface{})
				cpuReservationInMhz := int64(rInfo["cpu_reservation_in_mhz"].(int))
				cpuReservationInShares := rInfo["cpu_reservation_in_shares"].(string)
				memoryReservationPercentage := int64(rInfo["memory_reservation_percentage"].(int))

				reservationInfo = &model.ReservationInfo{
					CpuReservation: &model.CPUReservation{
						ReservationInMhz:    &cpuReservationInMhz,
						ReservationInShares: &cpuReservationInShares,
					},
					MemoryReservation: &model.MemoryReservation{
						ReservationPercentage: &memoryReservationPercentage,
					},
				}
			}
			storageID := vdc["storage_id"].(string)
			vcID := vdc["vc_id"].(string)
			cfg := model.VsphereDeploymentConfig{
				ComputeId:             &computeID,
				DataNetworkIds:        dataNetworkIds,
				ManagementNetworkId:   &managementNetworkID,
				ManagementPortSubnets: managemenPortSubnets,
				ReservationInfo:       reservationInfo,
				StorageId:             &storageID,
				VcId:                  &vcID,
				PlacementType:         model.DeploymentConfig_PLACEMENT_TYPE_VSPHEREDEPLOYMENTCONFIG,
			}
			if len(defaultGatewayAddresses) > 0 {
				cfg.DefaultGatewayAddresses = defaultGatewayAddresses
			}
			if hostID != "" {
				cfg.HostId = &hostID
			}
			// Passing an empty folder here confuses vSphere while creating the Edge VM
			if computeFolderID != "" {
				cfg.ComputeFolderId = &computeFolderID
			}
			dataValue, errs := converter.ConvertToVapi(cfg, model.VsphereDeploymentConfigBindingType())
			if errs != nil {
				return nil, errs[0]
			} else if dataValue != nil {
				vmDeploymentConfig = dataValue.(*data.StructValue)
			}
		}
		return &model.EdgeNodeDeploymentConfig{
			FormFactor:         &formFactor,
			NodeUserSettings:   nodeUserSettings,
			VmDeploymentConfig: vmDeploymentConfig,
		}, nil
	}
	return nil, nil
}

func getEdgeNodeSettingsFromSchema(s interface{}) (*model.EdgeNodeSettings, error) {
	if s == nil {
		return nil, nil
	}
	settings := s.([]interface{})
	for _, settingIf := range settings {
		setting := settingIf.(map[string]interface{})
		advCfg := getKeyValuePairListFromSchema(setting["advanced_configuration"])
		allowSSHRootLogin := setting["allow_ssh_root_login"].(bool)
		dnsServers := interface2StringList(setting["dns_servers"].([]interface{}))
		enableSSH := setting["enable_ssh"].(bool)
		enableUptMode := setting["enable_upt_mode"].(bool)
		hostName := setting["hostname"].(string)
		ntpServers := interface2StringList(setting["ntp_servers"].([]interface{}))
		searchDomains := interface2StringList(setting["search_domains"].([]interface{}))
		var syslogServers []model.SyslogConfiguration
		for _, sli := range setting["syslog_server"].([]interface{}) {
			syslogServer := sli.(map[string]interface{})
			logLevel := syslogServer["log_level"].(string)
			port := syslogServer["port"].(string)
			protocol := syslogServer["protocol"].(string)
			server := syslogServer["server"].(string)
			syslogServers = append(syslogServers, model.SyslogConfiguration{
				LogLevel: &logLevel,
				Port:     &port,
				Protocol: &protocol,
				Server:   &server,
			})
		}
		obj := &model.EdgeNodeSettings{
			AdvancedConfiguration: advCfg,
			AllowSshRootLogin:     &allowSSHRootLogin,
			DnsServers:            dnsServers,
			EnableSsh:             &enableSSH,
			Hostname:              &hostName,
			NtpServers:            ntpServers,
			SearchDomains:         searchDomains,
			SyslogServers:         syslogServers,
		}
		if util.NsxVersionHigherOrEqual("4.0.0") {
			obj.EnableUptMode = &enableUptMode
		}
		return obj, nil
	}

	return nil, nil
}

func getCPUConfigFromSchema(cpuConfigList []interface{}) []model.CpuCoreConfigForEnhancedNetworkingStackSwitch {
	var cpuConfig []model.CpuCoreConfigForEnhancedNetworkingStackSwitch
	for _, cc := range cpuConfigList {
		data := cc.(map[string]interface{})
		numLCores := int64(data["num_lcores"].(int))
		numaNodeIndex := int64(data["numa_node_index"].(int))
		elem := model.CpuCoreConfigForEnhancedNetworkingStackSwitch{
			NumLcores:     &numLCores,
			NumaNodeIndex: &numaNodeIndex,
		}
		cpuConfig = append(cpuConfig, elem)
	}
	return cpuConfig
}

func getHostSwitchProfileIDsFromSchema(hswProfileList []interface{}) []model.HostSwitchProfileTypeIdEntry {
	var hswProfiles []model.HostSwitchProfileTypeIdEntry
	for _, hswp := range hswProfileList {
		key := model.BaseHostSwitchProfile_RESOURCE_TYPE_UPLINKHOSTSWITCHPROFILE
		val := hswp.(string)
		elem := model.HostSwitchProfileTypeIdEntry{
			Key:   &key,
			Value: &val,
		}
		hswProfiles = append(hswProfiles, elem)
	}
	return hswProfiles
}

func getIPAssignmentFromSchema(ipAssignmentList interface{}) (*data.StructValue, error) {
	if ipAssignmentList == nil {
		return nil, nil
	}
	converter := bindings.NewTypeConverter()

	for _, ia := range ipAssignmentList.([]interface{}) {
		iaType, iaData, err := getIPAssignmentData(ia.(map[string]interface{}))
		if err != nil {
			return nil, err
		}

		var dataValue data.DataValue
		var errs []error
		switch iaType {
		case "assigned_by_dhcp":
			dhcpEnabled := iaData.(bool)
			if dhcpEnabled {
				elem := model.AssignedByDhcp{
					ResourceType: model.IpAssignmentSpec_RESOURCE_TYPE_ASSIGNEDBYDHCP,
				}
				dataValue, errs = converter.ConvertToVapi(elem, model.AssignedByDhcpBindingType())
			} else {
				return nil, fmt.Errorf("no valid IP assignment found")
			}
		case "static_ip":
			for _, iad := range iaData.([]interface{}) {
				data := iad.(map[string]interface{})
				defaultGateway := data["default_gateway"].(string)
				ipList := interfaceListToStringList(data["ip_addresses"].([]interface{}))
				subnetMask := data["subnet_mask"].(string)
				elem := model.StaticIpListSpec{
					DefaultGateway: &defaultGateway,
					IpList:         ipList,
					SubnetMask:     &subnetMask,
					ResourceType:   model.IpAssignmentSpec_RESOURCE_TYPE_STATICIPLISTSPEC,
				}
				dataValue, errs = converter.ConvertToVapi(elem, model.StaticIpListSpecBindingType())
				break
			}

		case "static_ip_pool":
			staticIPPoolID := iaData.(string)
			elem := model.StaticIpPoolSpec{
				IpPoolId:     &staticIPPoolID,
				ResourceType: model.IpAssignmentSpec_RESOURCE_TYPE_STATICIPPOOLSPEC,
			}
			dataValue, errs = converter.ConvertToVapi(elem, model.StaticIpPoolSpecBindingType())

		default:
			return nil, fmt.Errorf("no valid IP assignment found")
		}
		if errs != nil {
			return nil, errs[0]
		}
		entryStruct := dataValue.(*data.StructValue)
		return entryStruct, nil
	}
	return nil, nil
}

func isSlice(v interface{}) bool {
	if v == nil {
		return false
	}
	return reflect.TypeOf(v).Kind() == reflect.Slice
}

func isBool(v interface{}) bool {
	if v == nil {
		return false
	}
	return reflect.TypeOf(v).Kind() == reflect.Bool
}

func isString(v interface{}) bool {
	if v == nil {
		return false
	}
	return reflect.TypeOf(v).Kind() == reflect.String
}

func getIPAssignmentData(data map[string]interface{}) (string, interface{}, error) {
	var t string
	var d interface{}

	n := 0
	for _, iaType := range ipAssignmentTypes {
		d1 := data[iaType]
		if (isString(d1) && d1 != "") || (isSlice(d1) && len(d1.([]interface{})) > 0) || (isBool(d1) && d1.(bool)) {
			t, d = iaType, data[iaType]
			n++
		}
	}
	if n > 1 {
		return "", nil, fmt.Errorf("exactly one IP assignment is allowed for ip_assignment object")
	}
	return t, d, nil
}

func getHostSwitchSpecFromSchema(d *schema.ResourceData, nodeType string) (*data.StructValue, error) {
	var dataValue data.DataValue
	var errs []error
	var hsList []model.StandardHostSwitch

	converter := bindings.NewTypeConverter()
	standardSwitchList := d.Get("standard_host_switch").([]interface{})
	for _, swEntry := range standardSwitchList {
		swData := swEntry.(map[string]interface{})

		hostSwitchID := swData["host_switch_id"].(string)
		hostSwitchName := swData["host_switch_name"].(string)
		var hostSwitchMode, hostSwitchType string
		var isMigratePNics bool
		var uplinks []model.VdsUplink
		var cpuConfig []model.CpuCoreConfigForEnhancedNetworkingStackSwitch
		if nodeType == nodeTypeHost {
			cpuConfig = getCPUConfigFromSchema(swData["cpu_config"].([]interface{}))
			hostSwitchMode = swData["host_switch_mode"].(string)
			hostSwitchType = model.StandardHostSwitch_HOST_SWITCH_TYPE_VDS
			isMigratePNics = swData["is_migrate_pnics"].(bool)
			uplinks = getUplinksFromSchema(swData["uplink"].([]interface{}))
		} else if nodeType == nodeTypeEdge {
			hostSwitchMode = model.StandardHostSwitch_HOST_SWITCH_MODE_STANDARD
			hostSwitchType = model.StandardHostSwitch_HOST_SWITCH_TYPE_NVDS
		}
		hostSwitchProfileIDs := getHostSwitchProfileIDsFromSchema(swData["host_switch_profile"].([]interface{}))
		iPAssignmentSpec, err := getIPAssignmentFromSchema(swData["ip_assignment"])
		var pNics []model.Pnic
		for _, p := range swData["pnic"].([]interface{}) {
			data := p.(map[string]interface{})
			deviceName := data["device_name"].(string)
			uplinkName := data["uplink_name"].(string)
			elem := model.Pnic{
				DeviceName: &deviceName,
				UplinkName: &uplinkName,
			}
			pNics = append(pNics, elem)
		}
		if err != nil {
			return nil, fmt.Errorf("error parsing HostSwitchSpec schema %v", err)
		}
		var transportNodeSubProfileCfg []model.TransportNodeProfileSubConfig
		if nodeType == nodeTypeHost {
			transportNodeSubProfileCfg = getTransportNodeSubProfileCfg(swData["transport_node_profile_sub_config"])
		}
		transportZoneEndpoints := getTransportZoneEndpointsFromSchema(swData["transport_zone_endpoint"].([]interface{}))

		hsw := model.StandardHostSwitch{
			HostSwitchId:           &hostSwitchID,
			HostSwitchMode:         &hostSwitchMode,
			HostSwitchProfileIds:   hostSwitchProfileIDs,
			HostSwitchType:         &hostSwitchType,
			IpAssignmentSpec:       iPAssignmentSpec,
			Pnics:                  pNics,
			TransportZoneEndpoints: transportZoneEndpoints,
		}
		if hostSwitchName != "" {
			hsw.HostSwitchName = &hostSwitchName
		}
		if nodeType == nodeTypeHost {
			hsw.CpuConfig = cpuConfig
			hsw.IsMigratePnics = &isMigratePNics
			hsw.TransportNodeProfileSubConfigs = transportNodeSubProfileCfg
			hsw.Uplinks = uplinks
		}

		hsList = append(hsList, hsw)
	}
	hostSwitchSpec := model.StandardHostSwitchSpec{
		HostSwitches: hsList,
		ResourceType: model.HostSwitchSpec_RESOURCE_TYPE_STANDARDHOSTSWITCHSPEC,
	}
	dataValue, errs = converter.ConvertToVapi(hostSwitchSpec, model.StandardHostSwitchSpecBindingType())

	if errs != nil {
		return nil, errs[0]
	} else if dataValue != nil {
		return dataValue.(*data.StructValue), nil
	}
	return nil, nil
}

func getTransportZoneEndpointsFromSchema(endpointList []interface{}) []model.TransportZoneEndPoint {
	var tzEPList []model.TransportZoneEndPoint
	for _, endpoint := range endpointList {
		data := endpoint.(map[string]interface{})
		transportZoneID := data["transport_zone"].(string)
		var transportZoneProfileIDs []model.TransportZoneProfileTypeIdEntry
		if data["transport_zone_profiles"] != nil {
			for _, tzpID := range data["transport_zone_profiles"].([]interface{}) {
				profileID := tzpID.(string)
				resourceType := model.TransportZoneProfileTypeIdEntry_RESOURCE_TYPE_BFDHEALTHMONITORINGPROFILE
				elem := model.TransportZoneProfileTypeIdEntry{
					ProfileId:    &profileID,
					ResourceType: &resourceType,
				}
				transportZoneProfileIDs = append(transportZoneProfileIDs, elem)
			}
		}

		elem := model.TransportZoneEndPoint{
			TransportZoneId:         &transportZoneID,
			TransportZoneProfileIds: transportZoneProfileIDs,
		}
		tzEPList = append(tzEPList, elem)
	}
	return tzEPList
}

func getUplinksFromSchema(uplinksList []interface{}) []model.VdsUplink {
	var uplinks []model.VdsUplink
	for _, ul := range uplinksList {
		data := ul.(map[string]interface{})
		uplinkName := data["uplink_name"].(string)
		vdsLagName := data["vds_lag_name"].(string)
		vdsUplinkName := data["vds_uplink_name"].(string)
		elem := model.VdsUplink{
			UplinkName:    &uplinkName,
			VdsLagName:    &vdsLagName,
			VdsUplinkName: &vdsUplinkName,
		}
		uplinks = append(uplinks, elem)
	}
	return uplinks
}

func getTransportNodeSubProfileCfg(iface interface{}) []model.TransportNodeProfileSubConfig {
	var cfgList []model.TransportNodeProfileSubConfig
	if iface == nil {
		return cfgList
	}
	profileCfgList := iface.([]interface{})
	for _, pCfg := range profileCfgList {
		data := pCfg.(map[string]interface{})
		name := data["name"].(string)
		var swCfgOpt *model.HostSwitchConfigOption
		for _, cfgOpt := range data["host_switch_config_option"].([]interface{}) {
			opt := cfgOpt.(map[string]interface{})
			swID := opt["host_switch_id"].(string)
			profileIDs := getHostSwitchProfileIDsFromSchema(opt["host_switch_profile"].([]interface{}))
			iPAssignmentSpec, _ := getIPAssignmentFromSchema(opt["ip_assignment"].([]interface{}))
			uplinks := getUplinksFromSchema(opt["uplink"].([]interface{}))
			swCfgOpt = &model.HostSwitchConfigOption{
				HostSwitchId:         &swID,
				HostSwitchProfileIds: profileIDs,
				IpAssignmentSpec:     iPAssignmentSpec,
				Uplinks:              uplinks,
			}
		}

		elem := model.TransportNodeProfileSubConfig{
			Name:                   &name,
			HostSwitchConfigOption: swCfgOpt,
		}
		cfgList = append(cfgList, elem)
	}
	return cfgList
}

func resourceNsxtEdgeTransportNodeRead(d *schema.ResourceData, m interface{}) error {
	connector := getPolicyConnector(m)
	id := d.Id()
	if id == "" {
		return fmt.Errorf("error obtaining logical object id")
	}

	client := nsx.NewTransportNodesClient(connector)
	obj, err := client.Get(id)
	if err != nil {
		return handleReadError(d, "TransportNode", id, err)
	}

	d.Set("revision", obj.Revision)
	d.Set("description", obj.Description)
	d.Set("display_name", obj.DisplayName)
	setMPTagsInSchema(d, obj.Tags)
	d.Set("failure_domain", obj.FailureDomainId)

	if obj.HostSwitchSpec != nil {
		err = setHostSwitchSpecInSchema(d, obj.HostSwitchSpec, nodeTypeEdge)
		if err != nil {
			return handleReadError(d, "TransportNode", id, err)
		}
	}

	converter := bindings.NewTypeConverter()
	base, errs := converter.ConvertToGolang(obj.NodeDeploymentInfo, model.EdgeNodeBindingType())
	if errs != nil {
		return handleReadError(d, "TransportNode", id, errs[0])
	}
	node := base.(model.EdgeNode)

	if node.DeploymentConfig != nil {
		err = setEdgeDeploymentConfigInSchema(d, node.DeploymentConfig)
		if err != nil {
			return handleReadError(d, "TransportNode", id, err)
		}
	}

	if node.NodeSettings != nil {
		err = setEdgeNodeSettingsInSchema(d, node.NodeSettings)
		if err != nil {
			return handleReadError(d, "TransportNode", id, err)
		}
	}

	// Set base object attributes
	d.Set("external_id", node.ExternalId)
	d.Set("fqdn", node.Fqdn)
	d.Set("ip_addresses", node.IpAddresses)

	if err != nil {
		return handleReadError(d, "TransportNode", id, err)
	}

	return nil
}

func setEdgeNodeSettingsInSchema(d *schema.ResourceData, nodeSettings *model.EdgeNodeSettings) error {
	elem := getElemOrEmptyMapFromSchema(d, "node_settings")
	elem["advanced_configuration"] = setKeyValueListForSchema(nodeSettings.AdvancedConfiguration)
	elem["allow_ssh_root_login"] = nodeSettings.AllowSshRootLogin
	elem["dns_servers"] = nodeSettings.DnsServers
	elem["enable_ssh"] = nodeSettings.EnableSsh
	elem["enable_upt_mode"] = nodeSettings.EnableUptMode
	elem["hostname"] = nodeSettings.Hostname
	elem["ntp_servers"] = nodeSettings.NtpServers
	elem["search_domains"] = nodeSettings.SearchDomains
	var syslogServers []map[string]interface{}
	for _, syslogServer := range nodeSettings.SyslogServers {
		e := make(map[string]interface{})
		e["log_level"] = syslogServer.LogLevel
		e["port"] = syslogServer.Port
		e["protocol"] = syslogServer.Protocol
		e["server"] = syslogServer.Server
		syslogServers = append(syslogServers, e)
	}
	elem["syslog_server"] = syslogServers
	d.Set("node_settings", []map[string]interface{}{elem})
	return nil
}

func setEdgeDeploymentConfigInSchema(d *schema.ResourceData, deploymentConfig *model.EdgeNodeDeploymentConfig) error {
	var err error
	elem := getElemOrEmptyMapFromSchema(d, "deployment_config")

	elem["form_factor"] = deploymentConfig.FormFactor
	var nodeUserSettings map[string]interface{}
	if elem["node_user_settings"] != nil {
		nodeUserSettings = elem["node_user_settings"].([]interface{})[0].(map[string]interface{})
	} else {
		nodeUserSettings = make(map[string]interface{})
	}
	// Note: password attributes is sensitive and is not returned by NSX
	if deploymentConfig.NodeUserSettings.AuditUsername != nil {
		nodeUserSettings["audit_username"] = deploymentConfig.NodeUserSettings.AuditUsername
	}
	nodeUserSettings["cli_username"] = deploymentConfig.NodeUserSettings.CliUsername
	elem["node_user_settings"] = []interface{}{nodeUserSettings}

	elem["vm_deployment_config"], err = setVMDeploymentConfigInSchema(deploymentConfig.VmDeploymentConfig)
	if err != nil {
		return err
	}
	d.Set("deployment_config", []map[string]interface{}{elem})

	return nil
}

func setVMDeploymentConfigInSchema(config *data.StructValue) (interface{}, error) {
	converter := bindings.NewTypeConverter()
	base, errs := converter.ConvertToGolang(config, model.DeploymentConfigBindingType())
	if errs != nil {
		return nil, errs[0]
	}
	cfgType := base.(model.DeploymentConfig).PlacementType

	// Only VsphereDeploymentConfig is supported
	if cfgType != model.DeploymentConfig_PLACEMENT_TYPE_VSPHEREDEPLOYMENTCONFIG {
		return nil, fmt.Errorf("unsupported PlacementType %s", cfgType)
	}

	vCfg, errs := converter.ConvertToGolang(config, model.VsphereDeploymentConfigBindingType())
	if errs != nil {
		return nil, errs[0]
	}
	vSphereCfg := vCfg.(model.VsphereDeploymentConfig)
	elem := make(map[string]interface{})
	elem["compute_folder_id"] = vSphereCfg.ComputeFolderId
	elem["compute_id"] = vSphereCfg.ComputeId
	elem["data_network_ids"] = vSphereCfg.DataNetworkIds
	elem["default_gateway_address"] = vSphereCfg.DefaultGatewayAddresses
	elem["host_id"] = vSphereCfg.HostId
	elem["management_network_id"] = vSphereCfg.ManagementNetworkId

	var mpSubnets []map[string]interface{}
	for _, mps := range vSphereCfg.ManagementPortSubnets {
		e := make(map[string]interface{})
		e["ip_addresses"] = mps.IpAddresses
		e["prefix_length"] = mps.PrefixLength
		mpSubnets = append(mpSubnets, e)
	}
	elem["management_port_subnet"] = mpSubnets

	reservationInfo := make(map[string]interface{})
	if vSphereCfg.ReservationInfo.CpuReservation.ReservationInMhz != nil {
		reservationInfo["cpu_reservation_in_mhz"] = vSphereCfg.ReservationInfo.CpuReservation.ReservationInMhz
	}
	if vSphereCfg.ReservationInfo.CpuReservation.ReservationInShares != nil {
		reservationInfo["cpu_reservation_in_shares"] = vSphereCfg.ReservationInfo.CpuReservation.ReservationInShares
	}
	if vSphereCfg.ReservationInfo.MemoryReservation.ReservationPercentage != nil {
		reservationInfo["memory_reservation_percentage"] = vSphereCfg.ReservationInfo.MemoryReservation.ReservationPercentage
	}
	elem["reservation_info"] = []interface{}{reservationInfo}

	elem["storage_id"] = vSphereCfg.StorageId
	elem["vc_id"] = vSphereCfg.VcId

	return []interface{}{elem}, nil
}

func setHostSwitchSpecInSchema(d *schema.ResourceData, spec *data.StructValue, nodeType string) error {
	converter := bindings.NewTypeConverter()

	base, errs := converter.ConvertToGolang(spec, model.HostSwitchSpecBindingType())
	if errs != nil {
		return errs[0]
	}
	swType := base.(model.HostSwitchSpec).ResourceType

	switch swType {
	case model.HostSwitchSpec_RESOURCE_TYPE_STANDARDHOSTSWITCHSPEC:
		var swList []map[string]interface{}
		entry, errs := converter.ConvertToGolang(spec, model.StandardHostSwitchSpecBindingType())
		if errs != nil {
			return errs[0]
		}
		swEntry := entry.(model.StandardHostSwitchSpec)
		for _, sw := range swEntry.HostSwitches {

			// TODO - remove this when SDK is updated with NSX 4.1.2
			if sw.IpAssignmentSpec == nil {
				return fmt.Errorf("Ipv6 assignments are not supported yet")
			}

			elem := make(map[string]interface{})
			elem["host_switch_id"] = sw.HostSwitchId
			elem["host_switch_name"] = sw.HostSwitchName
			profiles := setHostSwitchProfileIDsInSchema(sw.HostSwitchProfileIds)
			if len(profiles) > 0 {
				elem["host_switch_profile"] = profiles
			}
			var err error
			elem["ip_assignment"], err = setIPAssignmentInSchema(sw.IpAssignmentSpec)
			if err != nil {
				return err
			}
			var pnics []map[string]interface{}
			for _, pnic := range sw.Pnics {
				e := make(map[string]interface{})
				e["device_name"] = pnic.DeviceName
				e["uplink_name"] = pnic.UplinkName
				pnics = append(pnics, e)
			}
			elem["pnic"] = pnics
			if nodeType == nodeTypeHost {
				var cpuConfig []map[string]interface{}
				for _, c := range sw.CpuConfig {
					e := make(map[string]interface{})
					e["num_lcores"] = c.NumLcores
					e["numa_node_index"] = c.NumaNodeIndex
					cpuConfig = append(cpuConfig, e)
				}
				elem["cpu_config"] = cpuConfig
				elem["is_migrate_pnics"] = sw.IsMigratePnics
				elem["host_switch_mode"] = sw.HostSwitchMode
				elem["uplink"] = setUplinksFromSchema(sw.Uplinks)
				var tnpSubConfig []map[string]interface{}
				for _, tnpsc := range sw.TransportNodeProfileSubConfigs {
					e := make(map[string]interface{})
					hsCfgOpt := make(map[string]interface{})
					hsCfgOpt["host_switch_id"] = tnpsc.HostSwitchConfigOption.HostSwitchId
					profiles := setHostSwitchProfileIDsInSchema(tnpsc.HostSwitchConfigOption.HostSwitchProfileIds)
					if len(profiles) > 0 {
						hsCfgOpt["host_switch_profile"] = profiles
					}
					if tnpsc.HostSwitchConfigOption.IpAssignmentSpec != nil {
						hsCfgOpt["ip_assignment"], err = setIPAssignmentInSchema(tnpsc.HostSwitchConfigOption.IpAssignmentSpec)
						if err != nil {
							return err
						}
					}
					hsCfgOpt["uplink"] = setUplinksFromSchema(tnpsc.HostSwitchConfigOption.Uplinks)
					e["host_switch_config_option"] = []interface{}{hsCfgOpt}
					e["name"] = tnpsc.Name
					tnpSubConfig = append(tnpSubConfig, e)
				}
				elem["transport_node_profile_sub_config"] = tnpSubConfig

				var vmkIMList []map[string]interface{}
				for _, vmkIM := range sw.VmkInstallMigration {
					e := make(map[string]interface{})
					e["destination_network"] = vmkIM.DestinationNetwork
					e["device_name"] = vmkIM.DeviceName
					vmkIMList = append(vmkIMList, e)
				}
				elem["vmk_install_migration"] = vmkIMList
			}
			elem["transport_zone_endpoint"] = setTransportZoneEndpointInSchema(sw.TransportZoneEndpoints)

			swList = append(swList, elem)
		}
		d.Set("standard_host_switch", swList)

	case model.HostSwitchSpec_RESOURCE_TYPE_PRECONFIGUREDHOSTSWITCHSPEC:
		return fmt.Errorf("preconfigured host switch is not supported")
	}
	return nil
}

func setTransportZoneEndpointInSchema(endpoints []model.TransportZoneEndPoint) interface{} {
	var endpointList []map[string]interface{}
	for _, endpoint := range endpoints {
		e := make(map[string]interface{})
		e["transport_zone"] = endpoint.TransportZoneId
		var tzpIDs []string
		for _, tzpID := range endpoint.TransportZoneProfileIds {
			tzpIDs = append(tzpIDs, *tzpID.ProfileId)
		}
		e["transport_zone_profiles"] = tzpIDs
		endpointList = append(endpointList, e)
	}
	return endpointList
}

func setUplinksFromSchema(uplinks []model.VdsUplink) interface{} {
	var uplinkList []map[string]interface{}
	for _, uplink := range uplinks {
		e := make(map[string]interface{})
		e["uplink_name"] = uplink.UplinkName
		e["vds_lag_name"] = uplink.VdsLagName
		e["vds_uplink_name"] = uplink.VdsUplinkName
		uplinkList = append(uplinkList, e)
	}
	return uplinkList
}

func setIPAssignmentInSchema(spec *data.StructValue) (interface{}, error) {
	elem := make(map[string]interface{})

	converter := bindings.NewTypeConverter()
	base, errs := converter.ConvertToGolang(spec, model.IpAssignmentSpecBindingType())
	if errs != nil {
		return nil, errs[0]
	}
	assignmentType := base.(model.IpAssignmentSpec).ResourceType

	switch assignmentType {
	case model.IpAssignmentSpec_RESOURCE_TYPE_ASSIGNEDBYDHCP:
		elem["assigned_by_dhcp"] = true

	case model.IpAssignmentSpec_RESOURCE_TYPE_STATICIPLISTSPEC:
		e := make(map[string]interface{})
		entry, errs := converter.ConvertToGolang(spec, model.StaticIpListSpecBindingType())
		if errs != nil {
			return nil, errs[0]
		}
		ipAsEntry := entry.(model.StaticIpListSpec)
		e["default_gateway"] = ipAsEntry.DefaultGateway
		e["ip_addresses"] = ipAsEntry.IpList
		e["subnet_mask"] = ipAsEntry.SubnetMask
		elem["static_ip"] = []map[string]interface{}{e}

	case model.IpAssignmentSpec_RESOURCE_TYPE_STATICIPPOOLSPEC:
		entry, errs := converter.ConvertToGolang(spec, model.StaticIpPoolSpecBindingType())
		if errs != nil {
			return nil, errs[0]
		}
		ipAsEntry := entry.(model.StaticIpPoolSpec)
		elem["static_ip_pool"] = ipAsEntry.IpPoolId
	}
	return []interface{}{elem}, nil
}

func setHostSwitchProfileIDsInSchema(hspIDs []model.HostSwitchProfileTypeIdEntry) []interface{} {
	var hostSwitchProfileIDs []interface{}
	for _, hspID := range hspIDs {
		hostSwitchProfileIDs = append(hostSwitchProfileIDs, hspID.Value)
	}
	return hostSwitchProfileIDs
}

func resourceNsxtEdgeTransportNodeUpdate(d *schema.ResourceData, m interface{}) error {
	connector := getPolicyConnector(m)
	id := d.Id()
	if id == "" {
		return fmt.Errorf("error obtaining logical object id")
	}

	client := nsx.NewTransportNodesClient(connector)

	obj, err := getTransportNodeFromSchema(d)
	if err != nil {
		return handleUpdateError("TransportNode", id, err)
	}
	revision := int64(d.Get("revision").(int))
	obj.Revision = &revision

	_, err = client.Update(id, *obj, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		return handleUpdateError("TransportNode", id, err)
	}

	return resourceNsxtEdgeTransportNodeRead(d, m)
}

func getTransportNodeStateConf(connector client.Connector, id string) *resource.StateChangeConf {
	return &resource.StateChangeConf{
		Pending: []string{"notyet"},
		Target:  []string{"success", "failed"},
		Refresh: func() (interface{}, string, error) {
			client := transport_nodes.NewStateClient(connector)

			_, err := client.Get(id)

			if isNotFoundError(err) {
				return "success", "success", nil
			}

			if err != nil {
				log.Printf("[DEBUG]: NSX Failed to retrieve TransportNode state: %v", err)
				return nil, "failed", err
			}

			return "notyet", "notyet", nil
		},
		Delay:        time.Duration(5) * time.Second,
		Timeout:      time.Duration(1200) * time.Second,
		PollInterval: time.Duration(5) * time.Second,
	}
}

func resourceNsxtEdgeTransportNodeDelete(d *schema.ResourceData, m interface{}) error {
	connector := getPolicyConnector(m)

	id := d.Id()
	if id == "" {
		return fmt.Errorf("error obtaining logical object id")
	}

	client := nsx.NewTransportNodesClient(connector)

	err := client.Delete(id, nil, nil)
	if err != nil {
		return handleDeleteError("TransportNode", id, err)
	}

	stateConf := getTransportNodeStateConf(connector, id)
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("failed to get deletion status for %s: %v", id, err)
	}

	return nil
}
