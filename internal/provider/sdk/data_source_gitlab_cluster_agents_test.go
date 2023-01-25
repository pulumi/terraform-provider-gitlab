//go:build acceptance
// +build acceptance

package sdk

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabClusterAgents_basic(t *testing.T) {
	testutil.RunIfAtLeast(t, "14.10")

	testProject := testutil.CreateProject(t)
	testClusterAgents := testutil.CreateClusterAgents(t, testProject.ID, 25)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					data "gitlab_cluster_agents" "this" {
						project = "%d"
					}
				`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_cluster_agents.this", "cluster_agents.#", fmt.Sprintf("%d", len(testClusterAgents))),
					resource.TestCheckResourceAttrSet("data.gitlab_cluster_agents.this", "cluster_agents.0.name"),
					resource.TestCheckResourceAttrSet("data.gitlab_cluster_agents.this", "cluster_agents.0.created_at"),
					resource.TestCheckResourceAttrSet("data.gitlab_cluster_agents.this", "cluster_agents.0.created_by_user_id"),
					resource.TestCheckResourceAttrSet("data.gitlab_cluster_agents.this", "cluster_agents.1.name"),
					resource.TestCheckResourceAttrSet("data.gitlab_cluster_agents.this", "cluster_agents.1.created_at"),
					resource.TestCheckResourceAttrSet("data.gitlab_cluster_agents.this", "cluster_agents.1.created_by_user_id"),
				),
			},
		},
	})
}
