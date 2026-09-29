package github

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubEnterpriseAppInstallationRepositories(t *testing.T) {
	t.Parallel()

	t.Run("manages selected repositories on an enterprise app installation", func(t *testing.T) {
		t.Parallel()

		if testAccConf.testOrgAppInstallationId == 0 {
			t.Skip("No org app installation id provided")
		}

		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
		repoName := fmt.Sprintf("%srepo-enterprise-app-install-%s", testResourcePrefix, randomID)

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "%s"
				auto_init = true
			}

			resource "github_enterprise_app_installation_repositories" "test" {
				enterprise_slug       = "%s"
				organization          = "%s"
				installation_id       = "%d"
				repository_selection  = "selected"
				selected_repositories = [github_repository.test.name]
			}
		`, repoName, testAccConf.enterpriseSlug, testAccConf.owner, testAccConf.testOrgAppInstallationId)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"github_enterprise_app_installation_repositories.test", "repository_selection", "selected",
			),
			resource.TestCheckResourceAttr(
				"github_enterprise_app_installation_repositories.test", "selected_repositories.#", "1",
			),
		)

		resource.Test(t, resource.TestCase{
			PreCheck:          func() { skipUnlessEnterprise(t) },
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
					Check:  check,
				},
			},
		})
	})

	t.Run("imports an enterprise app installation with selected repositories", func(t *testing.T) {
		t.Parallel()

		if testAccConf.testOrgAppInstallationId == 0 {
			t.Skip("No org app installation id provided")
		}

		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
		repoName := fmt.Sprintf("%srepo-enterprise-app-install-import-%s", testResourcePrefix, randomID)

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "%s"
				auto_init = true
			}

			resource "github_enterprise_app_installation_repositories" "test" {
				enterprise_slug       = "%s"
				organization          = "%s"
				installation_id       = "%d"
				repository_selection  = "selected"
				selected_repositories = [github_repository.test.name]
			}
		`, repoName, testAccConf.enterpriseSlug, testAccConf.owner, testAccConf.testOrgAppInstallationId)

		resource.Test(t, resource.TestCase{
			PreCheck:          func() { skipUnlessEnterprise(t) },
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
				},
				{
					ResourceName:      "github_enterprise_app_installation_repositories.test",
					ImportState:       true,
					ImportStateVerify: true,
				},
			},
		})
	})
}
