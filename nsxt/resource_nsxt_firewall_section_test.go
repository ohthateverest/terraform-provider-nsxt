/* Copyright © 2018 VMware, Inc. All Rights Reserved.
   SPDX-License-Identifier: MPL-2.0 */

package nsxt

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/vmware/go-vmware-nsxt"
	"net/http"
	"testing"
)

func TestAccResourceNsxtFirewallSection_basic(t *testing.T) {
	sectionName := fmt.Sprintf("test-nsx-firewall-section-basic")
	updatesectionName := fmt.Sprintf("%s-update", sectionName)
	testResourceName := "nsxt_firewall_section.test"
	tags := singleTag
	updatedTags := doubleTags
	tos := string("[]")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: func(state *terraform.State) error {
			return testAccNSXFirewallSectionCheckDestroy(state, sectionName)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccNSXFirewallSectionCreateEmptyTemplate(sectionName, tags, tos),
				Check: resource.ComposeTestCheckFunc(
					testAccNSXFirewallSectionExists(sectionName, testResourceName),
					resource.TestCheckResourceAttr(testResourceName, "display_name", sectionName),
					resource.TestCheckResourceAttr(testResourceName, "description", "Acceptance Test"),
					resource.TestCheckResourceAttr(testResourceName, "tag.#", "1"),
					resource.TestCheckResourceAttr(testResourceName, "section_type", "LAYER3"),
					resource.TestCheckResourceAttr(testResourceName, "stateful", "true"),
					resource.TestCheckResourceAttr(testResourceName, "rule.#", "0"),
					resource.TestCheckResourceAttr(testResourceName, "applied_to.#", "0"),
				),
			},
			{
				Config: testAccNSXFirewallSectionUpdateEmptyTemplate(updatesectionName, updatedTags, tos),
				Check: resource.ComposeTestCheckFunc(
					testAccNSXFirewallSectionExists(updatesectionName, testResourceName),
					resource.TestCheckResourceAttr(testResourceName, "display_name", updatesectionName),
					resource.TestCheckResourceAttr(testResourceName, "description", "Acceptance Test Update"),
					resource.TestCheckResourceAttr(testResourceName, "tag.#", "2"),
					resource.TestCheckResourceAttr(testResourceName, "section_type", "LAYER3"),
					resource.TestCheckResourceAttr(testResourceName, "stateful", "true"),
					resource.TestCheckResourceAttr(testResourceName, "rule.#", "0"),
					resource.TestCheckResourceAttr(testResourceName, "applied_to.#", "0"),
				),
			},
		},
	})
}

func TestAccResourceNsxtFirewallSection_withTos(t *testing.T) {
	sectionName := fmt.Sprintf("test-nsx-firewall-section-tos")
	updatesectionName := fmt.Sprintf("%s-update", sectionName)
	testResourceName := "nsxt_firewall_section.test"
	tags := singleTag
	tos := string("[{target_type = \"NSGroup\", target_id = \"${nsxt_ns_group.grp1.id}\"}]")
	updatedTos := string("[{target_type = \"NSGroup\", target_id = \"${nsxt_ns_group.grp1.id}\"}, {target_type = \"NSGroup\", target_id = \"${nsxt_ns_group.grp2.id}\"}]")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: func(state *terraform.State) error {
			return testAccNSXFirewallSectionCheckDestroy(state, sectionName)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccNSXFirewallSectionCreateEmptyTemplate(sectionName, tags, tos),
				Check: resource.ComposeTestCheckFunc(
					testAccNSXFirewallSectionExists(sectionName, testResourceName),
					resource.TestCheckResourceAttr(testResourceName, "display_name", sectionName),
					resource.TestCheckResourceAttr(testResourceName, "description", "Acceptance Test"),
					resource.TestCheckResourceAttr(testResourceName, "tag.#", "1"),
					resource.TestCheckResourceAttr(testResourceName, "section_type", "LAYER3"),
					resource.TestCheckResourceAttr(testResourceName, "stateful", "true"),
					resource.TestCheckResourceAttr(testResourceName, "rule.#", "0"),
					resource.TestCheckResourceAttr(testResourceName, "applied_to.#", "1"),
				),
			},
			{
				Config: testAccNSXFirewallSectionUpdateEmptyTemplate(updatesectionName, tags, updatedTos),
				Check: resource.ComposeTestCheckFunc(
					testAccNSXFirewallSectionExists(updatesectionName, testResourceName),
					resource.TestCheckResourceAttr(testResourceName, "display_name", updatesectionName),
					resource.TestCheckResourceAttr(testResourceName, "description", "Acceptance Test Update"),
					resource.TestCheckResourceAttr(testResourceName, "tag.#", "1"),
					resource.TestCheckResourceAttr(testResourceName, "section_type", "LAYER3"),
					resource.TestCheckResourceAttr(testResourceName, "stateful", "true"),
					resource.TestCheckResourceAttr(testResourceName, "rule.#", "0"),
					resource.TestCheckResourceAttr(testResourceName, "applied_to.#", "2"),
				),
			},
		},
	})
}

func TestAccResourceNsxtFirewallSection_withRules(t *testing.T) {
	sectionName := fmt.Sprintf("test-nsx-firewall-section-rules")
	updatesectionName := fmt.Sprintf("%s-update", sectionName)
	testResourceName := "nsxt_firewall_section.test"
	ruleName := "rule1.0"
	updatedRuleName := "rule1.1"
	tags := singleTag
	tos := string("[]")
	ruleTos := string("[]")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: func(state *terraform.State) error {
			return testAccNSXFirewallSectionCheckDestroy(state, sectionName)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccNSXFirewallSectionCreateTemplate(sectionName, ruleName, tags, tos, ruleTos),
				Check: resource.ComposeTestCheckFunc(
					testAccNSXFirewallSectionExists(sectionName, testResourceName),
					resource.TestCheckResourceAttr(testResourceName, "display_name", sectionName),
					resource.TestCheckResourceAttr(testResourceName, "description", "Acceptance Test"),
					resource.TestCheckResourceAttr(testResourceName, "section_type", "LAYER3"),
					resource.TestCheckResourceAttr(testResourceName, "stateful", "true"),
					resource.TestCheckResourceAttr(testResourceName, "rule.#", "1"),
					resource.TestCheckResourceAttr(testResourceName, "rule.0.display_name", ruleName),
					resource.TestCheckResourceAttr(testResourceName, "rule.0.applied_to.#", "0"),
					resource.TestCheckResourceAttr(testResourceName, "tag.#", "1"),
					resource.TestCheckResourceAttr(testResourceName, "applied_to.#", "0"),
				),
			},
			{
				Config: testAccNSXFirewallSectionUpdateTemplate(updatesectionName, updatedRuleName, tags, tos),
				Check: resource.ComposeTestCheckFunc(
					testAccNSXFirewallSectionExists(updatesectionName, testResourceName),
					resource.TestCheckResourceAttr(testResourceName, "display_name", updatesectionName),
					resource.TestCheckResourceAttr(testResourceName, "description", "Acceptance Test Update"),
					resource.TestCheckResourceAttr(testResourceName, "section_type", "LAYER3"),
					resource.TestCheckResourceAttr(testResourceName, "stateful", "true"),
					resource.TestCheckResourceAttr(testResourceName, "rule.#", "2"),
					resource.TestCheckResourceAttr(testResourceName, "rule.0.display_name", updatedRuleName),
					resource.TestCheckResourceAttr(testResourceName, "applied_to.#", "0"),
					resource.TestCheckResourceAttr(testResourceName, "tag.#", "1"),
				),
			},
		},
	})
}

func TestAccResourceNsxtFirewallSection_withRulesAndTags(t *testing.T) {
	sectionName := fmt.Sprintf("test-nsx-firewall-section-tags")
	updatesectionName := fmt.Sprintf("%s-update", sectionName)
	testResourceName := "nsxt_firewall_section.test"
	ruleName := "rule1.0"
	updatedRuleName := "rule1.1"
	tags := singleTag
	updatedTags := doubleTags
	tos := string("[]")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: func(state *terraform.State) error {
			return testAccNSXFirewallSectionCheckDestroy(state, sectionName)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccNSXFirewallSectionCreateTemplate(sectionName, ruleName, tags, tos, tos),
				Check: resource.ComposeTestCheckFunc(
					testAccNSXFirewallSectionExists(sectionName, testResourceName),
					resource.TestCheckResourceAttr(testResourceName, "display_name", sectionName),
					resource.TestCheckResourceAttr(testResourceName, "description", "Acceptance Test"),
					resource.TestCheckResourceAttr(testResourceName, "section_type", "LAYER3"),
					resource.TestCheckResourceAttr(testResourceName, "stateful", "true"),
					resource.TestCheckResourceAttr(testResourceName, "rule.#", "1"),
					resource.TestCheckResourceAttr(testResourceName, "rule.0.display_name", ruleName),
					resource.TestCheckResourceAttr(testResourceName, "rule.0.applied_to.#", "0"),
					resource.TestCheckResourceAttr(testResourceName, "applied_to.#", "0"),
					resource.TestCheckResourceAttr(testResourceName, "tag.#", "1"),
				),
			},
			{
				Config: testAccNSXFirewallSectionUpdateTemplate(updatesectionName, updatedRuleName, updatedTags, tos),
				Check: resource.ComposeTestCheckFunc(
					testAccNSXFirewallSectionExists(updatesectionName, testResourceName),
					resource.TestCheckResourceAttr(testResourceName, "display_name", updatesectionName),
					resource.TestCheckResourceAttr(testResourceName, "description", "Acceptance Test Update"),
					resource.TestCheckResourceAttr(testResourceName, "section_type", "LAYER3"),
					resource.TestCheckResourceAttr(testResourceName, "stateful", "true"),
					resource.TestCheckResourceAttr(testResourceName, "rule.#", "2"),
					resource.TestCheckResourceAttr(testResourceName, "rule.0.display_name", updatedRuleName),
					resource.TestCheckResourceAttr(testResourceName, "applied_to.#", "0"),
					resource.TestCheckResourceAttr(testResourceName, "tag.#", "2"),
				),
			},
		},
	})
}

func TestAccResourceNsxtFirewallSection_withRulesAndTos(t *testing.T) {
	sectionName := fmt.Sprintf("test-nsx-firewall-section-rules_and_tos")
	updatesectionName := fmt.Sprintf("%s-update", sectionName)
	testResourceName := "nsxt_firewall_section.test"
	ruleName := "rule1.0"
	updatedRuleName := "rule1.1"
	tags := singleTag
	tos := string("[{target_type = \"NSGroup\", target_id = \"${nsxt_ns_group.grp1.id}\"}]")
	ruleTos := string("[{target_type = \"NSGroup\", target_id = \"${nsxt_ns_group.grp5.id}\"}]")
	updatedTos := string("[{target_type = \"NSGroup\", target_id = \"${nsxt_ns_group.grp1.id}\"}, {target_type = \"NSGroup\", target_id = \"${nsxt_ns_group.grp2.id}\"}]")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: func(state *terraform.State) error {
			return testAccNSXFirewallSectionCheckDestroy(state, sectionName)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccNSXFirewallSectionCreateTemplate(sectionName, ruleName, tags, tos, ruleTos),
				Check: resource.ComposeTestCheckFunc(
					testAccNSXFirewallSectionExists(sectionName, testResourceName),
					resource.TestCheckResourceAttr(testResourceName, "display_name", sectionName),
					resource.TestCheckResourceAttr(testResourceName, "description", "Acceptance Test"),
					resource.TestCheckResourceAttr(testResourceName, "section_type", "LAYER3"),
					resource.TestCheckResourceAttr(testResourceName, "stateful", "true"),
					resource.TestCheckResourceAttr(testResourceName, "rule.#", "1"),
					resource.TestCheckResourceAttr(testResourceName, "rule.0.display_name", ruleName),
					resource.TestCheckResourceAttr(testResourceName, "rule.0.applied_to.#", "1"),
					resource.TestCheckResourceAttr(testResourceName, "tag.#", "1"),
					resource.TestCheckResourceAttr(testResourceName, "applied_to.#", "1"),
				),
			},
			{
				Config: testAccNSXFirewallSectionUpdateTemplate(updatesectionName, updatedRuleName, tags, updatedTos),
				Check: resource.ComposeTestCheckFunc(
					testAccNSXFirewallSectionExists(updatesectionName, testResourceName),
					resource.TestCheckResourceAttr(testResourceName, "display_name", updatesectionName),
					resource.TestCheckResourceAttr(testResourceName, "description", "Acceptance Test Update"),
					resource.TestCheckResourceAttr(testResourceName, "section_type", "LAYER3"),
					resource.TestCheckResourceAttr(testResourceName, "stateful", "true"),
					resource.TestCheckResourceAttr(testResourceName, "rule.#", "2"),
					resource.TestCheckResourceAttr(testResourceName, "rule.0.display_name", updatedRuleName),
					resource.TestCheckResourceAttr(testResourceName, "tag.#", "1"),
					resource.TestCheckResourceAttr(testResourceName, "applied_to.#", "2"),
				),
			},
		},
	})
}

func TestAccResourceNsxtFirewallSection_ordered(t *testing.T) {
	sectionNames := [4]string{"s1", "s2", "s3", "s4"}
	testResourceNames := [4]string{"nsxt_firewall_section.test1", "nsxt_firewall_section.test2", "nsxt_firewall_section.test3", "nsxt_firewall_section.test4"}

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: func(state *terraform.State) error {
			for i := 0; i <= 3; i++ {
				err := testAccNSXFirewallSectionCheckDestroy(state, sectionNames[i])
				if err != nil {
					return err
				}
			}

			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: testAccNSXFirewallSectionCreateOrderedTemplate(),
				Check: resource.ComposeTestCheckFunc(
					testAccNSXFirewallSectionExists(sectionNames[0], testResourceNames[0]),
					resource.TestCheckResourceAttr(testResourceNames[0], "display_name", sectionNames[0]),
					resource.TestCheckResourceAttr(testResourceNames[0], "section_type", "LAYER3"),
					testAccNSXFirewallSectionExists(sectionNames[1], testResourceNames[1]),
					resource.TestCheckResourceAttr(testResourceNames[1], "display_name", sectionNames[1]),
					resource.TestCheckResourceAttr(testResourceNames[1], "section_type", "LAYER3"),
					testAccNSXFirewallSectionExists(sectionNames[2], testResourceNames[2]),
					resource.TestCheckResourceAttr(testResourceNames[2], "display_name", sectionNames[2]),
					resource.TestCheckResourceAttr(testResourceNames[2], "section_type", "LAYER3"),
				),
			},
			{
				Config: testAccNSXFirewallSectionUpdateOrderedTemplate(),
				Check: resource.ComposeTestCheckFunc(
					testAccNSXFirewallSectionExists(sectionNames[0], testResourceNames[0]),
					resource.TestCheckResourceAttr(testResourceNames[0], "display_name", sectionNames[0]),
					resource.TestCheckResourceAttr(testResourceNames[0], "section_type", "LAYER3"),
					testAccNSXFirewallSectionExists(sectionNames[1], testResourceNames[1]),
					resource.TestCheckResourceAttr(testResourceNames[1], "display_name", sectionNames[1]),
					resource.TestCheckResourceAttr(testResourceNames[1], "section_type", "LAYER3"),
					testAccNSXFirewallSectionExists(sectionNames[2], testResourceNames[2]),
					resource.TestCheckResourceAttr(testResourceNames[2], "display_name", sectionNames[2]),
					resource.TestCheckResourceAttr(testResourceNames[2], "section_type", "LAYER3"),
					testAccNSXFirewallSectionExists(sectionNames[3], testResourceNames[3]),
					resource.TestCheckResourceAttr(testResourceNames[3], "display_name", sectionNames[3]),
					resource.TestCheckResourceAttr(testResourceNames[3], "section_type", "LAYER3"),
				),
			},
		},
	})
}

func TestAccResourceNsxtFirewallSection_importBasic(t *testing.T) {
	sectionName := fmt.Sprintf("test-nsx-firewall-section-basic")
	testResourceName := "nsxt_firewall_section.test"
	tags := singleTag
	tos := string("[]")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: func(state *terraform.State) error {
			return testAccNSXFirewallSectionCheckDestroy(state, sectionName)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccNSXFirewallSectionCreateEmptyTemplate(sectionName, tags, tos),
			},
			{
				ResourceName:      testResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccResourceNsxtFirewallSection_importWithRules(t *testing.T) {
	sectionName := fmt.Sprintf("test-nsx-firewall-section-rules")
	testResourceName := "nsxt_firewall_section.test"
	ruleName := "rule1.0"
	tags := singleTag
	tos := string("[]")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: func(state *terraform.State) error {
			return testAccNSXFirewallSectionCheckDestroy(state, sectionName)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccNSXFirewallSectionCreateTemplate(sectionName, ruleName, tags, tos, tos),
			},
			{
				ResourceName:      testResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccResourceNsxtFirewallSection_importWithTos(t *testing.T) {
	sectionName := fmt.Sprintf("test-nsx-firewall-section-tos")
	testResourceName := "nsxt_firewall_section.test"
	tags := singleTag
	tos := string("[{target_type = \"NSGroup\", target_id = \"${nsxt_ns_group.grp1.id}\"}]")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: func(state *terraform.State) error {
			return testAccNSXFirewallSectionCheckDestroy(state, sectionName)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccNSXFirewallSectionCreateEmptyTemplate(sectionName, tags, tos),
			},
			{
				ResourceName:      testResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccNSXFirewallSectionExists(displayName string, resourceName string) resource.TestCheckFunc {
	return func(state *terraform.State) error {

		nsxClient := testAccProvider.Meta().(*nsxt.APIClient)

		rs, ok := state.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Firewall Section resource %s not found in resources", resourceName)
		}

		resourceID := rs.Primary.ID
		if resourceID == "" {
			return fmt.Errorf("Firewall Section resource ID not set in resources ")
		}

		profile, responseCode, err := nsxClient.ServicesApi.GetSection(nsxClient.Context, resourceID)
		if err != nil {
			return fmt.Errorf("Error while retrieving firewall section ID %s. Error: %v", resourceID, err)
		}

		if responseCode.StatusCode != http.StatusOK {
			return fmt.Errorf("Error while checking if firewall section %s exists. HTTP return code was %d", resourceID, responseCode.StatusCode)
		}

		if displayName == profile.DisplayName {
			return nil
		}
		return fmt.Errorf("Firewall Section %s wasn't found", displayName)
	}
}

func testAccNSXFirewallSectionCheckDestroy(state *terraform.State, displayName string) error {
	nsxClient := testAccProvider.Meta().(*nsxt.APIClient)

	for _, rs := range state.RootModule().Resources {

		if rs.Type != "nsxt_firewall_section" {
			continue
		}

		resourceID := rs.Primary.Attributes["id"]
		profile, responseCode, err := nsxClient.ServicesApi.GetSection(nsxClient.Context, resourceID)
		if err != nil {
			if responseCode.StatusCode != http.StatusOK {
				return nil
			}
			return fmt.Errorf("Error while retrieving firewall section ID %s. Error: %v", resourceID, err)
		}

		if displayName == profile.DisplayName {
			return fmt.Errorf("Firewall Section %s still exists", displayName)
		}
	}
	return nil
}

func testAccNSXFirewallSectionNSGroups() string {
	return fmt.Sprintf(`
resource "nsxt_ns_group" "grp1" {
  display_name = "grp1"
}

resource "nsxt_ns_group" "grp2" {
  display_name = "grp2"
}

resource "nsxt_ns_group" "grp3" {
  display_name = "grp3"
}

resource "nsxt_ns_group" "grp4" {
  display_name = "grp4"
}

resource "nsxt_ns_group" "grp5" {
  display_name = "grp5"
}

resource "nsxt_ip_protocol_ns_service" "test" {
  protocol = "6"
}`)
}

func testAccNSXFirewallSectionCreateTemplate(name string, ruleName string, tags string, tos string, ruleTos string) string {
	return testAccNSXFirewallSectionNSGroups() + fmt.Sprintf(`
resource "nsxt_firewall_section" "test" {
  display_name = "%s"
  description  = "Acceptance Test"
  section_type = "LAYER3"
  stateful     = true
  tag          = %s
  applied_to   = %s

  rule {
    display_name          = "%s",
    description           = "rule1",
    action                = "ALLOW",
    logged                = "true",
    ip_protocol           = "IPV4",
    direction             = "IN"
    destinations_excluded = "false"
    sources_excluded      = "false"
    notes                 = "test rule"
    rule_tag              = "test rule tag"
	disabled              = "false"
    applied_to            = %s

    source {
      target_id   = "${nsxt_ns_group.grp1.id}"
      target_type = "NSGroup"
    }

    source {
      target_id   = "${nsxt_ns_group.grp2.id}"
      target_type = "NSGroup"
    }

    destination {
      target_id   = "${nsxt_ns_group.grp3.id}"
      target_type = "NSGroup"
    }

    destination {
      target_id   = "${nsxt_ns_group.grp4.id}"
      target_type = "NSGroup"
    }

    service {
      target_id   = "${nsxt_ip_protocol_ns_service.test.id}"
      target_type = "NSService"
    }
  }
}`, name, tags, tos, ruleName, ruleTos)
}

func testAccNSXFirewallSectionUpdateTemplate(updatedName string, updatedRuleName string, tags string, tos string) string {
	return testAccNSXFirewallSectionNSGroups() + fmt.Sprintf(`
resource "nsxt_firewall_section" "test" {
  display_name = "%s"
  description  = "Acceptance Test Update"
  section_type = "LAYER3"
  stateful     = true
  tag          = %s
  applied_to   = %s

  rule {
    display_name = "%s",
    description  = "rule1",
    action       = "ALLOW",
    logged       = "true",
    ip_protocol  = "IPV4",
    direction    = "IN"
    disabled     = "false"
  }

  rule {
    display_name = "rule2",
    description  = "rule2",
    action       = "ALLOW",
    logged       = "true",
    ip_protocol  = "IPV6",
    direction    = "OUT"
  }
}`, updatedName, tags, tos, updatedRuleName)
}

func testAccNSXFirewallSectionCreateEmptyTemplate(name string, tags string, tos string) string {
	return testAccNSXFirewallSectionNSGroups() + fmt.Sprintf(`
resource "nsxt_firewall_section" "test" {
  display_name = "%s"
  description  = "Acceptance Test"
  section_type = "LAYER3"
  stateful     = true
  tag         = %s
  applied_to  = %s
}`, name, tags, tos)
}

func testAccNSXFirewallSectionUpdateEmptyTemplate(updatedName string, tags string, tos string) string {
	return testAccNSXFirewallSectionNSGroups() + fmt.Sprintf(`
resource "nsxt_firewall_section" "test" {
  display_name = "%s"
  description  = "Acceptance Test Update"
  section_type = "LAYER3"
  stateful     = true
  tag          = %s
  applied_to   = %s
}`, updatedName, tags, tos)
}

func testAccNSXFirewallSectionCreateOrderedTemplate() string {
	return `
resource "nsxt_firewall_section" "test1" {
  display_name = "s1"
  section_type = "LAYER3"
  stateful     = true
}

resource "nsxt_firewall_section" "test2" {
  display_name  = "s2"
  section_type  = "LAYER3"
  insert_before = "${nsxt_firewall_section.test1.id}"
  stateful      = true

  rule {
    display_name = "test"
    action       = "ALLOW",
    logged       = "true",
    ip_protocol  = "IPV4",
    direction    = "IN"
  }
}

resource "nsxt_firewall_section" "test3" {
  display_name  = "s3"
  section_type  = "LAYER3"
  insert_before = "${nsxt_firewall_section.test2.id}"
  stateful      = true
}

`
}

func testAccNSXFirewallSectionUpdateOrderedTemplate() string {
	return `
resource "nsxt_firewall_section" "test1" {
  display_name  = "s1"
  section_type  = "LAYER3"
  insert_before = "${nsxt_firewall_section.test4.id}"
  stateful      = true
}

resource "nsxt_firewall_section" "test2" {
  display_name  = "s2"
  section_type  = "LAYER3"
  insert_before = "${nsxt_firewall_section.test1.id}"
  stateful      = true
}

resource "nsxt_firewall_section" "test3" {
  display_name = "s3"
  section_type = "LAYER3"
  stateful     = true
}

resource "nsxt_firewall_section" "test4" {
  display_name = "s4"
  section_type = "LAYER3"
  stateful     = true
}

`
}
