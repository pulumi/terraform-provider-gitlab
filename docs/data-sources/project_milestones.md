---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "gitlab_project_milestones Data Source - terraform-provider-gitlab"
subcategory: ""
description: |-
  The gitlab_project_milestones data source allows get details of a project milestones.
  Upstream API: GitLab REST API docs https://docs.gitlab.com/ee/api/milestones.html
---

# gitlab_project_milestones (Data Source)

The `gitlab_project_milestones` data source allows get details of a project milestones.

**Upstream API**: [GitLab REST API docs](https://docs.gitlab.com/ee/api/milestones.html)

## Example Usage

```terraform
# By project ID
data "gitlab_project_milestones" "example" {
  project = "12345"
}

# By project full path
data "gitlab_project_milestones" "example" {
  project = "foo/bar"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `project` (String) The ID or URL-encoded path of the project owned by the authenticated user.

### Optional

- `iids` (List of Number) Return only the milestones having the given `iid` (Note: ignored if `include_parent_milestones` is set as `true`).
- `include_parent_milestones` (Boolean) Include group milestones from parent group and its ancestors. Introduced in GitLab 13.4.
- `search` (String) Return only milestones with a title or description matching the provided string.
- `state` (String) Return only `active` or `closed` milestones.
- `title` (String) Return only the milestones having the given `title`.

### Read-Only

- `id` (String) The ID of this resource.
- `milestones` (List of Object) List of milestones from a project. (see [below for nested schema](#nestedatt--milestones))

<a id="nestedatt--milestones"></a>
### Nested Schema for `milestones`

Read-Only:

- `created_at` (String)
- `description` (String)
- `due_date` (String)
- `expired` (Boolean)
- `iid` (Number)
- `milestone_id` (Number)
- `project` (String)
- `project_id` (Number)
- `start_date` (String)
- `state` (String)
- `title` (String)
- `updated_at` (String)
- `web_url` (String)

