package sdk

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var _ = registerDataSource("gitlab_project_protected_branch", func() *schema.Resource {
	return &schema.Resource{
		Description: `The ` + "`gitlab_protected_branch`" + ` data source allows details of a protected branch to be retrieved by its name and the project it belongs to.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/protected_branches.html#get-a-single-protected-branch-or-wildcard-protected-branch)`,

		ReadContext: dataSourceGitlabProjectProtectedBranchRead,
		Schema: map[string]*schema.Schema{
			"project_id": {
				Description:  "The integer or path with namespace that uniquely identifies the project.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"name": {
				Description:  "The name of the protected branch.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"id": {
				Description: "The ID of this resource.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"push_access_levels":  dataSourceGitlabProjectProtectedBranchSchemaAccessLevels(),
			"merge_access_levels": dataSourceGitlabProjectProtectedBranchSchemaAccessLevels(),
			"allow_force_push": {
				Description: "Whether force push is allowed.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"code_owner_approval_required": {
				Description: "Reject code pushes that change files listed in the CODEOWNERS file.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
		},
	}
})

func dataSourceGitlabProjectProtectedBranchSchemaAccessLevels() *schema.Schema {
	return &schema.Schema{
		Description: "Describes which access levels, users, or groups are allowed to perform the action.",
		Type:        schema.TypeList,
		Computed:    true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"access_level": {
					Description: fmt.Sprintf("The access level allowed to perform the respective action (shows as 40 - \"maintainer\" if `user_id` or `group_id` are present). Valid values are: %s", utils.RenderValueListForDocs(api.ValidProtectedBranchTagAccessLevelNames)),
					Type:        schema.TypeString,
					Computed:    true,
				},
				"access_level_description": {
					Description: "A description of the allowed access level(s), or the name of the user or group if `user_id` or `group_id` are present.",
					Type:        schema.TypeString,
					Computed:    true,
				},
				"user_id": {
					Description: "If present, indicates that the user is allowed to perform the respective action. (only GitLab Premium or higher)",
					Type:        schema.TypeInt,
					Computed:    true,
					Optional:    true,
				},
				"group_id": {
					Description: "If present, indicates that the group is allowed to perform the respective action. (only GitLab Premium or higher)",
					Type:        schema.TypeInt,
					Computed:    true,
					Optional:    true,
				},
			},
		},
	}
}

func dataSourceGitlabProjectProtectedBranchRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	log.Printf("[INFO] Reading Gitlab protected branch")

	project := d.Get("project_id")
	name := d.Get("name").(string)

	// Get protected branch by project ID/path and branch name
	pb, _, err := client.ProtectedBranches.GetProtectedBranch(project, name, gitlab.WithContext(ctx))
	if err != nil {
		return diag.Errorf("error getting protected branch (Project: %v / Name %v): %v", project, name, err)
	}

	if err := d.Set("push_access_levels", flattenBranchAccessDescriptions(pb.PushAccessLevels)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("merge_access_levels", flattenBranchAccessDescriptions(pb.MergeAccessLevels)); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("allow_force_push", pb.AllowForcePush); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("code_owner_approval_required", pb.CodeOwnerApprovalRequired); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%d", pb.ID))

	return nil
}

func flattenBranchAccessDescriptions(descriptions []*gitlab.BranchAccessDescription) (values []map[string]interface{}) {
	for _, description := range descriptions {
		v := map[string]interface{}{
			"access_level":             api.AccessLevelValueToName[description.AccessLevel],
			"access_level_description": description.AccessLevelDescription,
		}
		if description.UserID != 0 {
			v["user_id"] = description.UserID
		}
		if description.GroupID != 0 {
			v["group_id"] = description.GroupID
		}
		values = append(values, v)
	}
	return values
}
