//nolint:revive
package domains

// The following file has been autogenerated. Please avoid any changes!
import (
	"errors"

	vapiProtocolClient_ "github.com/vmware/vsphere-automation-sdk-go/runtime/protocol/client"
	client0 "github.com/vmware/vsphere-automation-sdk-go/services/nsxt/infra/domains"
	model0 "github.com/vmware/vsphere-automation-sdk-go/services/nsxt/model"
	client1 "github.com/vmware/vsphere-automation-sdk-go/services/nsxt/orgs/projects/infra/domains"

	utl "github.com/vmware/terraform-provider-nsxt/api/utl"
)

type IdsSecurityPolicyClientContext utl.ClientContext

func NewIntrusionServicePoliciesClient(sessionContext utl.SessionContext, connector vapiProtocolClient_.Connector) *IdsSecurityPolicyClientContext {
	var client interface{}

	switch sessionContext.ClientType {

	case utl.Local:
		client = client0.NewIntrusionServicePoliciesClient(connector)

	case utl.Multitenancy:
		client = client1.NewIntrusionServicePoliciesClient(connector)

	default:
		return nil
	}
	return &IdsSecurityPolicyClientContext{Client: client, ClientType: sessionContext.ClientType, ProjectID: sessionContext.ProjectID}
}

func (c IdsSecurityPolicyClientContext) Get(domainIdParam string, policyIdParam string) (model0.IdsSecurityPolicy, error) {
	var obj model0.IdsSecurityPolicy
	var err error

	switch c.ClientType {

	case utl.Local:
		client := c.Client.(client0.IntrusionServicePoliciesClient)
		obj, err = client.Get(domainIdParam, policyIdParam)
		if err != nil {
			return obj, err
		}

	case utl.Multitenancy:
		client := c.Client.(client1.IntrusionServicePoliciesClient)
		obj, err = client.Get(utl.DefaultOrgID, c.ProjectID, domainIdParam, policyIdParam)
		if err != nil {
			return obj, err
		}

	default:
		return obj, errors.New("invalid infrastructure for model")
	}
	return obj, err
}

func (c IdsSecurityPolicyClientContext) Delete(domainIdParam string, policyIdParam string) error {
	var err error

	switch c.ClientType {

	case utl.Local:
		client := c.Client.(client0.IntrusionServicePoliciesClient)
		err = client.Delete(domainIdParam, policyIdParam)

	case utl.Multitenancy:
		client := c.Client.(client1.IntrusionServicePoliciesClient)
		err = client.Delete(utl.DefaultOrgID, c.ProjectID, domainIdParam, policyIdParam)

	default:
		err = errors.New("invalid infrastructure for model")
	}
	return err
}

func (c IdsSecurityPolicyClientContext) List(domainIdParam string, cursorParam *string, includeMarkForDeleteObjectsParam *bool, includeRuleCountParam *bool, includedFieldsParam *string, pageSizeParam *int64, sortAscendingParam *bool, sortByParam *string) (model0.IdsSecurityPolicyListResult, error) {
	var err error
	var obj model0.IdsSecurityPolicyListResult

	switch c.ClientType {

	case utl.Local:
		client := c.Client.(client0.IntrusionServicePoliciesClient)
		obj, err = client.List(domainIdParam, cursorParam, includeMarkForDeleteObjectsParam, includeRuleCountParam, includedFieldsParam, pageSizeParam, sortAscendingParam, sortByParam)

	case utl.Multitenancy:
		client := c.Client.(client1.IntrusionServicePoliciesClient)
		obj, err = client.List(utl.DefaultOrgID, c.ProjectID, domainIdParam, cursorParam, includeMarkForDeleteObjectsParam, includeRuleCountParam, includedFieldsParam, pageSizeParam, sortAscendingParam, sortByParam)

	default:
		err = errors.New("invalid infrastructure for model")
	}
	return obj, err
}

func (c IdsSecurityPolicyClientContext) Patch(domainIdParam string, policyIdParam string, idsSecurityPolicyParam model0.IdsSecurityPolicy) error {
	var err error

	switch c.ClientType {

	case utl.Local:
		client := c.Client.(client0.IntrusionServicePoliciesClient)
		err = client.Patch(domainIdParam, policyIdParam, idsSecurityPolicyParam)

	case utl.Multitenancy:
		client := c.Client.(client1.IntrusionServicePoliciesClient)
		err = client.Patch(utl.DefaultOrgID, c.ProjectID, domainIdParam, policyIdParam, idsSecurityPolicyParam)

	default:
		err = errors.New("invalid infrastructure for model")
	}
	return err
}

func (c IdsSecurityPolicyClientContext) Update(domainIdParam string, policyIdParam string, idsSecurityPolicyParam model0.IdsSecurityPolicy) (model0.IdsSecurityPolicy, error) {
	var err error
	var obj model0.IdsSecurityPolicy

	switch c.ClientType {

	case utl.Local:
		client := c.Client.(client0.IntrusionServicePoliciesClient)
		obj, err = client.Update(domainIdParam, policyIdParam, idsSecurityPolicyParam)

	case utl.Multitenancy:
		client := c.Client.(client1.IntrusionServicePoliciesClient)
		obj, err = client.Update(utl.DefaultOrgID, c.ProjectID, domainIdParam, policyIdParam, idsSecurityPolicyParam)

	default:
		err = errors.New("invalid infrastructure for model")
	}
	return obj, err
}
