package github

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func Test_hashRepositoryName(t *testing.T) {
	t.Parallel()

	if hashRepositoryName("Hello-World") != hashRepositoryName("hello-world") {
		t.Fatalf("expected %q and %q to hash equal", "Hello-World", "hello-world")
	}

	if hashRepositoryName("repo-a") == hashRepositoryName("repo-b") {
		t.Fatalf("expected %q and %q to hash differently", "repo-a", "repo-b")
	}
}

func Test_selectedRepositoriesSetIgnoresCase(t *testing.T) {
	t.Parallel()

	a := schema.NewSet(hashRepositoryName, []any{"Repo-A", "repo-b"})
	b := schema.NewSet(hashRepositoryName, []any{"repo-a", "REPO-B"})

	if got := a.Difference(b).Len(); got != 0 {
		t.Fatalf("expected a.Difference(b) to be empty, got %d elements", got)
	}
	if got := b.Difference(a).Len(); got != 0 {
		t.Fatalf("expected b.Difference(a) to be empty, got %d elements", got)
	}

	c := schema.NewSet(hashRepositoryName, []any{"repo-a", "repo-c"})

	diff := c.Difference(a).List()
	if len(diff) != 1 || diff[0] != "repo-c" {
		t.Fatalf("expected c.Difference(a) to be [\"repo-c\"], got %v", diff)
	}
	if got := a.Difference(c).Len(); got != 1 {
		t.Fatalf("expected a.Difference(c) to have 1 element, got %d", got)
	}
}
