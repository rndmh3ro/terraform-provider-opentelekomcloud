package acceptance

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"

	acc "github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/acceptance/common"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/acceptance/env"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/common"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/common/cfg"
)

const resourceClusterName = "opentelekomcloud_css_cluster_v1.cluster"

func TestAccCssClusterV1_basic(t *testing.T) {
	name := fmt.Sprintf("css-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { acc.TestAccPreCheck(t) },
		ProviderFactories: acc.TestAccProviderFactories,
		CheckDestroy:      testAccCheckCssClusterV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCssClusterV1Basic(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCssClusterV1Exists(),
					resource.TestCheckResourceAttr(resourceClusterName, "nodes.#", "1"),
				),
			},
			{
				Config: testAccCssClusterV1Extend(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCssClusterV1Exists(),
					resource.TestCheckResourceAttr(resourceClusterName, "expect_node_num", "2"),
					resource.TestCheckResourceAttr(resourceClusterName, "nodes.#", "2"),
				),
			},
		},
	})
}

func TestAccCssClusterV1_validateDiskAndFlavor(t *testing.T) {
	name := fmt.Sprintf("css-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { acc.TestAccPreCheck(t) },
		ProviderFactories: acc.TestAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccCssClusterV1TooSmall(name),
				ExpectError: regexp.MustCompile(`invalid disk size.+`),
				PlanOnly:    true,
			},
			{
				Config:      testAccCssClusterV1TooBig(name),
				ExpectError: regexp.MustCompile(`invalid disk size.+`),
				PlanOnly:    true,
			},
			{
				Config:      testAccCssClusterV1FlavorName(name),
				ExpectError: regexp.MustCompile(`can't find flavor with name: .+`),
				PlanOnly:    true,
			},
			{
				Config:             testAccCssClusterV1Basic(name),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccCssClusterV1_encrypted(t *testing.T) {
	name := fmt.Sprintf("css-%s", acctest.RandString(10))
	if env.OS_KMS_ID == "" {
		t.Skip("KMS key ID is not set")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { acc.TestAccPreCheck(t) },
		ProviderFactories: acc.TestAccProviderFactories,
		CheckDestroy:      testAccCheckCssClusterV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCssClusterV1Encrypted(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCssClusterV1Exists(),
					resource.TestCheckResourceAttr(resourceClusterName, "nodes.#", "1"),
					resource.TestCheckResourceAttr(resourceClusterName, "node_config.0.volume.0.encryption_key", env.OS_KMS_ID),
				),
			},
		},
	})
}

func testAccCheckCssClusterV1Destroy(s *terraform.State) error {
	config := acc.TestAccProvider.Meta().(*cfg.Config)
	client, err := config.CssV1Client(env.OS_REGION_NAME)
	if err != nil {
		return fmt.Errorf("error creating CSSv1 client: %w", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "opentelekomcloud_css_cluster_v1" {
			continue
		}

		url, err := common.ReplaceVarsForTest(rs, "clusters/{id}")
		if err != nil {
			return err
		}
		url = client.ServiceURL(url)

		_, err = client.Get(url, nil, &golangsdk.RequestOpts{
			MoreHeaders: map[string]string{"Content-Type": "application/json"}})
		if err == nil {
			return fmt.Errorf("opentelekomcloud_css_cluster_v1 still exists at %s", url)
		}
	}

	return nil
}

func testAccCheckCssClusterV1Exists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		config := acc.TestAccProvider.Meta().(*cfg.Config)
		client, err := config.CssV1Client(env.OS_REGION_NAME)
		if err != nil {
			return fmt.Errorf("error creating CSSv1 client: %w", err)
		}

		rs, ok := s.RootModule().Resources["opentelekomcloud_css_cluster_v1.cluster"]
		if !ok {
			return fmt.Errorf("error checking opentelekomcloud_css_cluster_v1.cluster exist, err=not found this resource")
		}

		url, err := common.ReplaceVarsForTest(rs, "clusters/{id}")
		if err != nil {
			return fmt.Errorf("error checking opentelekomcloud_css_cluster_v1.cluster exist, err=building url failed: %s", err)
		}
		url = client.ServiceURL(url)

		_, err = client.Get(url, nil, &golangsdk.RequestOpts{
			MoreHeaders: map[string]string{"Content-Type": "application/json"}})
		if err != nil {
			if _, ok := err.(golangsdk.ErrDefault404); ok {
				return fmt.Errorf("opentelekomcloud_css_cluster_v1.cluster is not exist")
			}
			return fmt.Errorf("error checking opentelekomcloud_css_cluster_v1.cluster exist, err=send request failed: %s", err)
		}
		return nil
	}
}

func testAccCssClusterV1Basic(name string) string {
	return fmt.Sprintf(`
%s

%s

resource "opentelekomcloud_css_cluster_v1" "cluster" {
  expect_node_num = 1
  name            = "%s"
  node_config {
    flavor = "css.medium.8"
    network_info {
      security_group_id = data.opentelekomcloud_networking_secgroup_v2.default_secgroup.id
      network_id        = data.opentelekomcloud_vpc_subnet_v1.shared_subnet.network_id
      vpc_id            = data.opentelekomcloud_vpc_subnet_v1.shared_subnet.vpc_id
    }
    volume {
      volume_type = "COMMON"
      size        = 40
    }

    availability_zone = "%s"
  }
  datastore {
    version = "7.6.2"
  }
  enable_https     = true
  enable_authority = true
  admin_pass       = "QwertyUI!"
}
`, acc.DataSourceSecGroupDefault, acc.DataSourceSubnet, name, env.OS_AVAILABILITY_ZONE)
}

func testAccCssClusterV1TooSmall(name string) string {
	return fmt.Sprintf(`
%s

%s

resource "opentelekomcloud_css_cluster_v1" "cluster" {
  expect_node_num = 1
  name            = "%s"
  node_config {
    flavor = "css.medium.8"
    network_info {
      security_group_id = data.opentelekomcloud_networking_secgroup_v2.default_secgroup.id
      network_id        = data.opentelekomcloud_vpc_subnet_v1.shared_subnet.network_id
      vpc_id            = data.opentelekomcloud_vpc_subnet_v1.shared_subnet.vpc_id
    }
    volume {
      volume_type = "COMMON"
      size        = 1
    }

    availability_zone = "%s"
  }
  datastore {
    version = "7.6.2"
  }
  enable_https     = true
  enable_authority = true
  admin_pass       = "QwertyUI!"
}
`, acc.DataSourceSecGroupDefault, acc.DataSourceSubnet, name, env.OS_AVAILABILITY_ZONE)
}

func testAccCssClusterV1TooBig(name string) string {
	return fmt.Sprintf(`
%s

%s

resource "opentelekomcloud_css_cluster_v1" "cluster" {
  expect_node_num = 1
  name            = "%s"
  node_config {
    flavor = "css.medium.8"
    network_info {
      security_group_id = data.opentelekomcloud_networking_secgroup_v2.default_secgroup.id
      network_id        = data.opentelekomcloud_vpc_subnet_v1.shared_subnet.network_id
      vpc_id            = data.opentelekomcloud_vpc_subnet_v1.shared_subnet.vpc_id
    }
    volume {
      volume_type = "COMMON"
      size        = 10000000
    }

    availability_zone = "%s"
  }
  datastore {
    version = "7.6.2"
  }
  enable_https     = true
  enable_authority = true
  admin_pass       = "QwertyUI!"
}
`, acc.DataSourceSecGroupDefault, acc.DataSourceSubnet, name, env.OS_AVAILABILITY_ZONE)
}

func testAccCssClusterV1FlavorName(name string) string {
	return fmt.Sprintf(`
%s

%s

resource "opentelekomcloud_css_cluster_v1" "cluster" {
  expect_node_num = 1
  name            = "%s"
  node_config {
    flavor = "css.large.8"
    network_info {
      security_group_id = data.opentelekomcloud_networking_secgroup_v2.default_secgroup.id
      network_id        = data.opentelekomcloud_vpc_subnet_v1.shared_subnet.network_id
      vpc_id            = data.opentelekomcloud_vpc_subnet_v1.shared_subnet.vpc_id
    }
    volume {
      volume_type = "COMMON"
      size        = 20
    }

    availability_zone = "%s"
  }
  datastore {
    version = "7.6.2"
  }
  enable_https     = true
  enable_authority = true
  admin_pass       = "QwertyUI!"
}
`, acc.DataSourceSecGroupDefault, acc.DataSourceSubnet, name, env.OS_AVAILABILITY_ZONE)
}

func testAccCssClusterV1Extend(name string) string {
	return fmt.Sprintf(`
%s

%s

resource "opentelekomcloud_css_cluster_v1" "cluster" {
  expect_node_num = 2
  name            = "%s"
  node_config {
    flavor = "css.medium.8"
    network_info {
      security_group_id = data.opentelekomcloud_networking_secgroup_v2.default_secgroup.id
      network_id        = data.opentelekomcloud_vpc_subnet_v1.shared_subnet.network_id
      vpc_id            = data.opentelekomcloud_vpc_subnet_v1.shared_subnet.vpc_id
    }
    volume {
      volume_type = "COMMON"
      size        = 40
    }

    availability_zone = "%s"
  }
  datastore {
    version = "7.6.2"
  }
  enable_https     = true
  enable_authority = true
  admin_pass       = "QwertyUI!"
}
`, acc.DataSourceSecGroupDefault, acc.DataSourceSubnet, name, env.OS_AVAILABILITY_ZONE)
}

func testAccCssClusterV1Encrypted(name string) string {
	return fmt.Sprintf(`
%s

%s

resource "opentelekomcloud_css_cluster_v1" "cluster" {
  expect_node_num = 1
  name            = "%s"
  node_config {
    flavor = "css.medium.8"
    network_info {
      security_group_id = data.opentelekomcloud_networking_secgroup_v2.default_secgroup.id
      network_id        = data.opentelekomcloud_vpc_subnet_v1.shared_subnet.network_id
      vpc_id            = data.opentelekomcloud_vpc_subnet_v1.shared_subnet.vpc_id
    }
    volume {
      volume_type    = "COMMON"
      size           = 40
      encryption_key = "%s"
    }

    availability_zone = "%s"
  }
  datastore {
    version = "7.6.2"
  }
  enable_https     = true
  enable_authority = true
  admin_pass       = "QwertyUI!"
}
`, acc.DataSourceSecGroupDefault, acc.DataSourceSubnet, name, env.OS_KMS_ID, env.OS_AVAILABILITY_ZONE)
}
