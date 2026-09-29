---
page_title: "github_enterprise_app_installation_repositories (Resource) - GitHub"
description: |-
  Manages the repositories accessible to a GitHub App installed on an enterprise-owned organization.
---

# github_enterprise_app_installation_repositories (Resource)

This resource manages the repositories a GitHub App can access on an enterprise-owned organization, using the [enterprise organization installations API](https://docs.github.com/enterprise-cloud@latest/rest/enterprise-admin/organization-installations).

Unlike `github_app_installation_repositories`, which authenticates through `/user/installations/...` and cannot both read and write repository access with any single token type ([#2103](https://github.com/integrations/terraform-provider-github/issues/2103)), this resource requires an enterprise-owned GitHub App with the "Enterprise organization installations: write" permission, installed on the enterprise, and authenticated as that App's installation.

## Example Usage

```terraform
data "github_enterprise" "enterprise" {
  slug = "my-enterprise"
}

resource "github_repository" "some_repo" {
  name = "some-repo"
}

resource "github_repository" "another_repo" {
  name = "another-repo"
}

resource "github_enterprise_app_installation_repositories" "some_app_repos" {
  enterprise_slug = data.github_enterprise.enterprise.slug
  organization    = "my-organization"

  # The installation id of the app on the organization.
  installation_id       = "1234567"
  selected_repositories = [github_repository.some_repo.name, github_repository.another_repo.name]
}
```

## Argument Reference

The following arguments are supported:

- `enterprise_slug` - (Required) The slug of the enterprise.
- `organization` - (Required) The login of the enterprise-owned organization the app is installed on.
- `installation_id` - (Required) The GitHub app installation id.
- `repository_selection` - (Optional) The repository access granted to the installation. Can be `all` or `selected`. Defaults to `selected`.
- `selected_repositories` - (Optional) Names of the repositories the installation can access. Must be empty when `repository_selection` is `all`, and must contain at least one repository when `repository_selection` is `selected`.

## Import

This resource can be imported using an ID made up of `enterprise_slug`, `organization` and `installation_id`, e.g.

```shell
terraform import github_enterprise_app_installation_repositories.some_app_repos my-enterprise:my-organization:1234567
```

~> **Note**: Deleting this resource when `repository_selection` is `selected` removes the managed `selected_repositories` from the installation. Deleting it when `repository_selection` is `all` does nothing: the app installation and its access to all repositories are left untouched. In neither case is the app uninstalled from the organization.
