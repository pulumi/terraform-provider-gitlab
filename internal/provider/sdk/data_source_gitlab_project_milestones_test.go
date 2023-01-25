//go:build acceptance
// +build acceptance

package sdk

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataGitlabProjectMilestones_basic(t *testing.T) {

	testProject := testutil.CreateProject(t)
	testMilestones := testutil.AddProjectMilestones(t, testProject, 2)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
				data "gitlab_project_milestones" "this" {
					project = "%d"
				}`, testProject.ID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_project_milestones.this", "milestones.#", fmt.Sprintf("%d", len(testMilestones))),
					resource.TestCheckResourceAttr("data.gitlab_project_milestones.this", "milestones.0.title", testMilestones[1].Title),
					resource.TestCheckResourceAttr("data.gitlab_project_milestones.this", "milestones.0.description", testMilestones[1].Description),
					resource.TestCheckResourceAttr("data.gitlab_project_milestones.this", "milestones.1.title", testMilestones[0].Title),
					resource.TestCheckResourceAttr("data.gitlab_project_milestones.this", "milestones.1.description", testMilestones[0].Description),
				),
			},
		},
	})
}
