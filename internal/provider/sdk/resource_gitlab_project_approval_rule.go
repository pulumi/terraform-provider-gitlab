package sdk

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/utils"
)

var _ = registerResource("gitlab_project_approval_rule", func() *schema.Resource {
	var validRuleTypeValues = []string{
		"regular",
		"any_approver",
	}
	return &schema.Resource{
		Description: `The ` + "`" + `gitlab_project_approval_rule` + "`" + ` resource allows to manage the lifecycle of a project-level approval rule.

-> This resource requires a GitLab Enterprise instance.

~> A project is limited to one "any_approver" rule at a time, any attempt to create a second rule of type "any_approver" will fail. As a result, if 
   an "any_approver" rule is already present on a project at creation time, and that rule requires 0 approvers, the rule will be automatically imported
   to prevent a common error with this resource.

~> Since a project is limited to one "any_approver" rule, attempting to add two "any_approver" rules to the same project in terraform will result in 
   terraform identifying changes with every "plan" operation, and may result in an error during the "apply" operation.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/merge_request_approvals.html#project-level-mr-approvals)`,

		CreateContext: resourceGitlabProjectApprovalRuleCreate,
		ReadContext:   resourceGitlabProjectApprovalRuleRead,
		UpdateContext: resourceGitlabProjectApprovalRuleUpdate,
		DeleteContext: resourceGitlabProjectApprovalRuleDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"project": {
				Description: "The name or id of the project to add the approval rules.",
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
			},
			"name": {
				Description: "The name of the approval rule.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"approvals_required": {
				Description: "The number of approvals required for this rule.",
				Type:        schema.TypeInt,
				Required:    true,
			},
			"rule_type": {
				Description:      fmt.Sprintf("String, defaults to 'regular'. The type of rule. `any_approver` is a pre-configured default rule with `approvals_required` at `0`. Valid values are %s.", utils.RenderValueListForDocs(validRuleTypeValues)),
				Type:             schema.TypeString,
				ForceNew:         true,
				Optional:         true,
				Computed:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice(validRuleTypeValues, false)),
			},
			"user_ids": {
				Description: "A list of specific User IDs to add to the list of approvers.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeInt},
				Set:         schema.HashInt,
			},
			"group_ids": {
				Description: "A list of group IDs whose members can approve of the merge request.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeInt},
				Set:         schema.HashInt,
			},
			"protected_branch_ids": {
				Description: "A list of protected branch IDs (not branch names) for which the rule applies.",
				Type:        schema.TypeSet,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeInt},
				Set:         schema.HashInt,
			},
			"disable_importing_default_any_approver_rule_on_create": {
				Description: "When this flag is set, the default `any_approver` rule will not be imported if present.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
			},
		},
	}
})

func resourceGitlabProjectApprovalRuleCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gitlab.Client)

	project := d.Get("project").(string)

	//Retrieve the rule_type, which is needed to determine if the rule is "any_approver"
	ruleType := ""
	if v, ok := d.GetOk("rule_type"); ok {
		ruleType = v.(string)
	}

	importBehavior := d.Get("disable_importing_default_any_approver_rule_on_create").(bool)

	// If the rule_type is "any_approver", then we need to check if the rule already exists, and update it instead of
	// create it.
	anyApproverRuleId := 0
	if ruleType == "any_approver" && !importBehavior {
		ruleId, err := getAnyApproverRuleId(ctx, client, project)
		if err != nil {
			return diag.FromErr(err)
		}
		anyApproverRuleId = ruleId
	}

	//If our ruleID is not 0, we need to update instead of create.
	//ruleID will be 0 if the rule is not found, or if the import is disabled
	ruleIDString := ""
	if anyApproverRuleId == 0 {

		options := gitlab.CreateProjectLevelRuleOptions{
			Name:               gitlab.String(d.Get("name").(string)),
			ApprovalsRequired:  gitlab.Int(d.Get("approvals_required").(int)),
			UserIDs:            expandApproverIds(d.Get("user_ids")),
			GroupIDs:           expandApproverIds(d.Get("group_ids")),
			ProtectedBranchIDs: expandProtectedBranchIDs(d.Get("protected_branch_ids")),
		}

		if v, ok := d.GetOk("rule_type"); ok {
			options.RuleType = gitlab.String(v.(string))
		}

		tflog.Debug(ctx, `Creating gitlab project-level rule`, map[string]interface{}{
			"Project": project, "Options": options,
		})

		rule, _, err := client.Projects.CreateProjectApprovalRule(project, &options, gitlab.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}

		ruleIDString = strconv.Itoa(rule.ID)
	} else {

		// We don't need to set "rule_type" because it's already implied in updating the "any_approver" rule.
		options := gitlab.UpdateProjectLevelRuleOptions{
			Name:               gitlab.String(d.Get("name").(string)),
			ApprovalsRequired:  gitlab.Int(d.Get("approvals_required").(int)),
			UserIDs:            expandApproverIds(d.Get("user_ids")),
			GroupIDs:           expandApproverIds(d.Get("group_ids")),
			ProtectedBranchIDs: expandProtectedBranchIDs(d.Get("protected_branch_ids")),
		}
		tflog.Debug(ctx, `Updating project level approval rule for "any_approver"`, map[string]interface{}{
			"Project": project, "RuleID": anyApproverRuleId, "Options": options,
		})

		rule, _, err := client.Projects.UpdateProjectApprovalRule(project, anyApproverRuleId, &options, gitlab.WithContext(ctx))
		if err != nil {
			return diag.FromErr(err)
		}
		ruleIDString = strconv.Itoa(rule.ID)
	}

	d.SetId(utils.BuildTwoPartID(&project, &ruleIDString))

	return resourceGitlabProjectApprovalRuleRead(ctx, d, meta)
}

func resourceGitlabProjectApprovalRuleRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, `Reading gitlab project-level rule`, map[string]interface{}{"ruleId": d.Id()})

	projectID, parsedRuleID, err := utils.ParseTwoPartID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	ruleID, err := strconv.Atoi(parsedRuleID)
	if err != nil {
		return diag.FromErr(err)
	}

	client := meta.(*gitlab.Client)

	rule, _, err := client.Projects.GetProjectApprovalRule(projectID, ruleID, gitlab.WithContext(ctx))
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, `No gitlab project-level rule found, removing from state`, map[string]interface{}{"ruleId": d.Id()})
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.Set("project", projectID)
	d.Set("name", rule.Name)
	d.Set("approvals_required", rule.ApprovalsRequired)
	d.Set("rule_type", rule.RuleType)

	if err := d.Set("group_ids", flattenApprovalRuleGroupIDs(rule.Groups)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("user_ids", flattenApprovalRuleUserIDs(rule.Users)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("protected_branch_ids", flattenProtectedBranchIDs(rule.ProtectedBranches)); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceGitlabProjectApprovalRuleUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	projectID, ruleID, err := utils.ParseTwoPartID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	ruleIDInt, err := strconv.Atoi(ruleID)
	if err != nil {
		return diag.FromErr(err)
	}

	options := gitlab.UpdateProjectLevelRuleOptions{
		Name:               gitlab.String(d.Get("name").(string)),
		ApprovalsRequired:  gitlab.Int(d.Get("approvals_required").(int)),
		UserIDs:            expandApproverIds(d.Get("user_ids")),
		GroupIDs:           expandApproverIds(d.Get("group_ids")),
		ProtectedBranchIDs: expandProtectedBranchIDs(d.Get("protected_branch_ids")),
	}

	tflog.Debug(ctx, `Updating gitlab project-level rule`, map[string]interface{}{"project": projectID, "options": options})

	client := meta.(*gitlab.Client)

	_, _, err = client.Projects.UpdateProjectApprovalRule(projectID, ruleIDInt, &options, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceGitlabProjectApprovalRuleRead(ctx, d, meta)
}

func resourceGitlabProjectApprovalRuleDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	project, ruleID, err := utils.ParseTwoPartID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	ruleIDInt, err := strconv.Atoi(ruleID)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, `Deleting gitlab project-level rule`, map[string]interface{}{"ruleId": ruleIDInt, "project": project})

	client := meta.(*gitlab.Client)

	_, err = client.Projects.DeleteProjectApprovalRule(project, ruleIDInt, gitlab.WithContext(ctx))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// flattenApprovalRuleUserIDs flattens a list of approval user ids into a list
// of ints for storage in state.
func flattenApprovalRuleUserIDs(users []*gitlab.BasicUser) []int {
	var userIDs []int

	for _, user := range users {
		userIDs = append(userIDs, user.ID)
	}

	return userIDs
}

// flattenApprovalRuleGroupIDs flattens a list of approval group ids into a list
// of ints for storage in state.
func flattenApprovalRuleGroupIDs(groups []*gitlab.Group) []int {
	var groupIDs []int

	for _, group := range groups {
		groupIDs = append(groupIDs, group.ID)
	}

	return groupIDs
}

func flattenProtectedBranchIDs(protectedBranches []*gitlab.ProtectedBranch) []int {
	var protectedBranchIDs []int

	for _, protectedBranch := range protectedBranches {
		protectedBranchIDs = append(protectedBranchIDs, protectedBranch.ID)
	}

	return protectedBranchIDs
}

// expandApproverIds Expands an interface into a list of ints to read from state.
func expandApproverIds(ids interface{}) *[]int {
	var approverIDs []int

	for _, id := range ids.(*schema.Set).List() {
		approverIDs = append(approverIDs, id.(int))
	}

	return &approverIDs
}

func expandProtectedBranchIDs(ids interface{}) *[]int {
	var protectedBranchIDs []int

	for _, id := range ids.(*schema.Set).List() {
		protectedBranchIDs = append(protectedBranchIDs, id.(int))
	}

	return &protectedBranchIDs
}

func getAnyApproverRuleId(ctx context.Context, client *gitlab.Client, project string) (int, error) {
	rules, _, err := client.Projects.GetProjectApprovalRules(project)
	if err != nil {
		if api.Is404(err) {
			tflog.Debug(ctx, `Project approval rules not found, skipping update for "any_approver" and creating instead.`, map[string]interface{}{
				"project": project,
			})
		} else {
			tflog.Error(ctx, `Error calling GitLab APi when retrieving approval rules for the "any_approver" rule check.`, map[string]interface{}{
				"project": project,
			})
			return 0, err
		}
	}

	for _, v := range rules {
		if v.RuleType == "any_approver" && v.ApprovalsRequired == 0 {
			tflog.Debug(ctx, `"any_approver" rule with 0 approvers already exists, updating instead of creating.`, map[string]interface{}{
				"project": project, "rule_id": v.ID,
			})
			return v.ID, nil
		}

		if v.RuleType == "any_approver" && v.ApprovalsRequired > 0 {
			tflog.Debug(ctx, `"any_approver" rule with more than 0 approvers exists, not eligible for auto-import.`, map[string]interface{}{
				"project": project, "rule_id": v.ID, "approvals_required": v.ApprovalsRequired,
			})
			return 0, nil
		}
	}

	// There was no error, and no rule identified
	return 0, nil
}
