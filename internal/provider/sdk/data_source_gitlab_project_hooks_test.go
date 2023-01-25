//go:build acceptance
// +build acceptance

package sdk

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabProjectHooks_basic(t *testing.T) {
	testProject := testutil.CreateProject(t)
	testHooks := testutil.CreateProjectHooks(t, testProject.ID, 25)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_project_hooks" "this" {
						project = "%s"
					}
				`, testProject.PathWithNamespace),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_hooks.this", "hooks.#", fmt.Sprintf("%d", len(testHooks))),
					resource.TestCheckResourceAttr("data.gitlab_project_hooks.this", "hooks.0.url", testHooks[0].URL),
					resource.TestCheckResourceAttr("data.gitlab_project_hooks.this", "hooks.1.url", testHooks[1].URL),
				),
			},
		},
	})
}
