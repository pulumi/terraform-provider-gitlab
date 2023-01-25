//go:build acceptance
// +build acceptance

package sdk

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/xanzy/go-gitlab"

	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccDataSourceGitlabCurrentUser_basic(t *testing.T) {
	//The root user has no public email by default, set the public email so it shows up properly.
	_, _, _ = testutil.TestGitlabClient.Users.ModifyUser(1, &gitlab.ModifyUserOptions{
		// The public email MUST match an email on record for the user, or it gets a bad request.
		PublicEmail: gitlab.String("admin@example.com"),
	})

	t.Cleanup(func() {
		_, _, _ = testutil.TestGitlabClient.Users.ModifyUser(1, &gitlab.ModifyUserOptions{
			//Set back to the empty state on test completion.
			PublicEmail: gitlab.String(""),
		})
	})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			{
				Config: `data "gitlab_current_user" "this" {}`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.gitlab_current_user.this", "id", "1"),
					resource.TestCheckResourceAttr("data.gitlab_current_user.this", "global_id", "gid://gitlab/User/1"),
					resource.TestCheckResourceAttr("data.gitlab_current_user.this", "name", "Administrator"),
					resource.TestCheckResourceAttr("data.gitlab_current_user.this", "username", "root"),
					resource.TestCheckResourceAttr("data.gitlab_current_user.this", "bot", "false"),
					resource.TestCheckResourceAttr("data.gitlab_current_user.this", "namespace_id", "1"),
					resource.TestCheckResourceAttr("data.gitlab_current_user.this", "global_namespace_id", "gid://gitlab/Namespaces::UserNamespace/1"),
					resource.TestCheckResourceAttr("data.gitlab_current_user.this", "public_email", "admin@example.com"),
					// Check only if this attribute is _set_, since other tests may modify it and we can't create a clean user for this test.
					resource.TestCheckResourceAttrSet("data.gitlab_current_user.this", "group_count"),
				),
			},
		},
	})
}
