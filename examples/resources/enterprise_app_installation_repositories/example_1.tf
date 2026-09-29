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
