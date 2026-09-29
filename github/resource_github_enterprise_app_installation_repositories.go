package github

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/go-github/v92/github"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// maxAppInstallationRepositoriesPerRequest is the GitHub API's limit on the
// number of repositories that can be added to, or removed from, an
// enterprise app installation in a single request.
const maxAppInstallationRepositoriesPerRequest = 50

func resourceGithubEnterpriseAppInstallationRepositories() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages the repositories accessible to a GitHub App installed on an enterprise-owned organization.",
		CreateContext: resourceGithubEnterpriseAppInstallationRepositoriesCreateOrUpdate,
		ReadContext:   resourceGithubEnterpriseAppInstallationRepositoriesRead,
		UpdateContext: resourceGithubEnterpriseAppInstallationRepositoriesCreateOrUpdate,
		DeleteContext: resourceGithubEnterpriseAppInstallationRepositoriesDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		CustomizeDiff: resourceGithubEnterpriseAppInstallationRepositoriesDiff,

		Schema: map[string]*schema.Schema{
			"enterprise_slug": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The slug of the enterprise.",
			},
			"organization": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The login of the enterprise-owned organization the app is installed on.",
			},
			"installation_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The GitHub app installation id.",
			},
			"repository_selection": {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          "selected",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"all", "selected"}, false)),
				Description:      "The repository access granted to the installation. Can be `all` or `selected`. Defaults to `selected`.",
			},
			"selected_repositories": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Set:         schema.HashString,
				Optional:    true,
				Description: "Names of the repositories the installation can access. Must be empty when `repository_selection` is `all`, and must contain at least one repository when `repository_selection` is `selected`.",
			},
		},
	}
}

func resourceGithubEnterpriseAppInstallationRepositoriesDiff(_ context.Context, d *schema.ResourceDiff, _ any) error {
	repositorySelection, _ := d.Get("repository_selection").(string)
	selectedRepositories, _ := d.Get("selected_repositories").(*schema.Set)

	switch repositorySelection {
	case "all":
		if selectedRepositories.Len() > 0 {
			return fmt.Errorf("selected_repositories must be empty when repository_selection is \"all\"")
		}
	case "selected":
		if selectedRepositories.Len() == 0 {
			return fmt.Errorf("selected_repositories must contain at least one repository when repository_selection is \"selected\"")
		}
	}

	return nil
}

func resourceGithubEnterpriseAppInstallationRepositoriesCreateOrUpdate(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	meta, _ := m.(*Owner)
	client := meta.v3client

	enterpriseSlug, _ := d.Get("enterprise_slug").(string)
	organization, _ := d.Get("organization").(string)
	installationIDString, _ := d.Get("installation_id").(string)
	repositorySelection, _ := d.Get("repository_selection").(string)

	installationID, err := strconv.ParseInt(installationIDString, 10, 64)
	if err != nil {
		return diag.FromErr(unconvertibleIdErr(installationIDString, err))
	}

	installation, err := findEnterpriseAppInstallation(ctx, client, enterpriseSlug, organization, installationID, meta.maxPerPage)
	if err != nil {
		return diag.FromErr(err)
	}
	if installation == nil {
		return diag.Errorf("no app installation %d found for organization %q in enterprise %q", installationID, organization, enterpriseSlug)
	}

	currentSelection := installation.GetRepositorySelection()

	switch {
	case repositorySelection == "all":
		if currentSelection != "all" {
			selection := "all"
			if _, _, err := client.Enterprise.UpdateAppInstallationRepositories(ctx, enterpriseSlug, organization, installationID, github.UpdateAppInstallationRepositoriesRequest{
				RepositorySelection: &selection,
			}); err != nil {
				return diag.FromErr(err)
			}
		}
	case currentSelection == "all":
		desiredRepositories := stringSetValues(d, "selected_repositories")
		batches := chunkRepositoryNames(desiredRepositories, maxAppInstallationRepositoriesPerRequest)

		selection := "selected"
		toggle := github.UpdateAppInstallationRepositoriesRequest{RepositorySelection: &selection}
		if len(batches) > 0 {
			toggle.Repositories = batches[0]
			batches = batches[1:]
		}

		if _, _, err := client.Enterprise.UpdateAppInstallationRepositories(ctx, enterpriseSlug, organization, installationID, toggle); err != nil {
			return diag.FromErr(err)
		}

		for _, batch := range batches {
			if _, _, err := client.Enterprise.AddRepositoriesToAppInstallation(ctx, enterpriseSlug, organization, installationID, github.AppInstallationRepositoriesRequest{Repositories: batch}); err != nil {
				return diag.FromErr(err)
			}
		}
	default:
		desiredRepositories := stringSetValues(d, "selected_repositories")

		currentRepositories, err := listEnterpriseAppInstallationRepositories(ctx, client, enterpriseSlug, organization, installationID, meta.maxPerPage)
		if err != nil {
			return diag.FromErr(err)
		}

		toAdd, toRemove := diffRepositoryNames(currentRepositories, desiredRepositories)

		for _, batch := range chunkRepositoryNames(toAdd, maxAppInstallationRepositoriesPerRequest) {
			if _, _, err := client.Enterprise.AddRepositoriesToAppInstallation(ctx, enterpriseSlug, organization, installationID, github.AppInstallationRepositoriesRequest{Repositories: batch}); err != nil {
				return diag.FromErr(err)
			}
		}

		for _, batch := range chunkRepositoryNames(toRemove, maxAppInstallationRepositoriesPerRequest) {
			if _, _, err := client.Enterprise.RemoveRepositoriesFromAppInstallation(ctx, enterpriseSlug, organization, installationID, github.AppInstallationRepositoriesRequest{Repositories: batch}); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	id, err := buildID(enterpriseSlug, organization, installationIDString)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(id)

	return resourceGithubEnterpriseAppInstallationRepositoriesRead(ctx, d, m)
}

func resourceGithubEnterpriseAppInstallationRepositoriesRead(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	meta, _ := m.(*Owner)
	client := meta.v3client

	enterpriseSlug, organization, installationIDString, err := parseID3(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	installationID, err := strconv.ParseInt(installationIDString, 10, 64)
	if err != nil {
		return diag.FromErr(unconvertibleIdErr(installationIDString, err))
	}

	installation, err := findEnterpriseAppInstallation(ctx, client, enterpriseSlug, organization, installationID, meta.maxPerPage)
	if err != nil {
		return diag.FromErr(err)
	}
	if installation == nil {
		tflog.Info(ctx, "Removing enterprise app installation repositories from state because it no longer exists in GitHub", map[string]any{"id": d.Id()})
		d.SetId("")
		return nil
	}

	if err := d.Set("enterprise_slug", enterpriseSlug); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("organization", organization); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("installation_id", installationIDString); err != nil {
		return diag.FromErr(err)
	}

	repositorySelection := installation.GetRepositorySelection()
	if err := d.Set("repository_selection", repositorySelection); err != nil {
		return diag.FromErr(err)
	}

	selectedRepositories := []string{}
	if repositorySelection == "selected" {
		selectedRepositories, err = listEnterpriseAppInstallationRepositories(ctx, client, enterpriseSlug, organization, installationID, meta.maxPerPage)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set("selected_repositories", selectedRepositories); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceGithubEnterpriseAppInstallationRepositoriesDelete(ctx context.Context, d *schema.ResourceData, m any) diag.Diagnostics {
	meta, _ := m.(*Owner)
	client := meta.v3client

	enterpriseSlug, organization, installationIDString, err := parseID3(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	installationID, err := strconv.ParseInt(installationIDString, 10, 64)
	if err != nil {
		return diag.FromErr(unconvertibleIdErr(installationIDString, err))
	}

	repositorySelection, _ := d.Get("repository_selection").(string)
	if repositorySelection == "all" {
		tflog.Info(ctx, "Skipping delete for enterprise app installation repositories: repository_selection is \"all\"; the installation is left untouched", map[string]any{"id": d.Id()})
		return nil
	}

	managedRepositories := stringSetValues(d, "selected_repositories")

	for _, batch := range chunkRepositoryNames(managedRepositories, maxAppInstallationRepositoriesPerRequest) {
		if _, _, err := client.Enterprise.RemoveRepositoriesFromAppInstallation(ctx, enterpriseSlug, organization, installationID, github.AppInstallationRepositoriesRequest{Repositories: batch}); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

// findEnterpriseAppInstallation pages through the app installations for
// organization until it finds the one matching installationID, returning nil
// if no such installation exists.
func findEnterpriseAppInstallation(ctx context.Context, client *github.Client, enterpriseSlug, organization string, installationID int64, perPage int) (*github.Installation, error) {
	opts := &github.ListOptions{PerPage: perPage}

	for {
		installations, resp, err := client.Enterprise.ListAppInstallations(ctx, enterpriseSlug, organization, opts)
		if err != nil {
			return nil, err
		}

		for _, installation := range installations {
			if installation.GetID() == installationID {
				return installation, nil
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return nil, nil
}

// listEnterpriseAppInstallationRepositories pages through the repositories
// accessible to installationID, returning their names.
func listEnterpriseAppInstallationRepositories(ctx context.Context, client *github.Client, enterpriseSlug, organization string, installationID int64, perPage int) ([]string, error) {
	opts := &github.ListOptions{PerPage: perPage}
	names := []string{}

	for {
		repos, resp, err := client.Enterprise.ListRepositoriesForOrgAppInstallation(ctx, enterpriseSlug, organization, installationID, opts)
		if err != nil {
			return nil, err
		}

		for _, repo := range repos {
			names = append(names, repo.GetName())
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return names, nil
}

// stringSetValues returns the string values of a *schema.TypeSet attribute.
func stringSetValues(d *schema.ResourceData, key string) []string {
	set, _ := d.Get(key).(*schema.Set)
	if set == nil {
		return nil
	}

	return expandStringList(set.List())
}

// chunkRepositoryNames splits names into batches of at most size, matching
// the GitHub API's limit on repositories per add/remove request.
func chunkRepositoryNames(names []string, size int) [][]string {
	if len(names) == 0 {
		return nil
	}

	chunks := make([][]string, 0, (len(names)+size-1)/size)
	for i := 0; i < len(names); i += size {
		end := min(i+size, len(names))
		chunks = append(chunks, names[i:end])
	}

	return chunks
}

// diffRepositoryNames compares the currently granted repository names against
// the desired set, returning the names to add and the names to remove.
func diffRepositoryNames(current, desired []string) (toAdd, toRemove []string) {
	currentSet := make(map[string]struct{}, len(current))
	for _, name := range current {
		currentSet[name] = struct{}{}
	}

	desiredSet := make(map[string]struct{}, len(desired))
	for _, name := range desired {
		desiredSet[name] = struct{}{}
	}

	for _, name := range desired {
		if _, ok := currentSet[name]; !ok {
			toAdd = append(toAdd, name)
		}
	}

	for _, name := range current {
		if _, ok := desiredSet[name]; !ok {
			toRemove = append(toRemove, name)
		}
	}

	return toAdd, toRemove
}
