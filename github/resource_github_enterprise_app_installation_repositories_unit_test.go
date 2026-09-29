package github

import (
	"reflect"
	"testing"
)

func Test_chunkRepositoryNames(t *testing.T) {
	t.Parallel()

	for _, d := range []struct {
		testName string
		names    []string
		size     int
		expected [][]string
	}{
		{
			testName: "zero_items",
			names:    []string{},
			size:     50,
			expected: nil,
		},
		{
			testName: "one_item",
			names:    []string{"repo-0"},
			size:     50,
			expected: [][]string{{"repo-0"}},
		},
		{
			testName: "exactly_one_batch",
			names:    repoNames(50),
			size:     50,
			expected: [][]string{repoNames(50)},
		},
		{
			testName: "one_over_a_batch",
			names:    repoNames(51),
			size:     50,
			expected: [][]string{repoNames(50), repoNames(51)[50:]},
		},
		{
			testName: "multiple_full_and_partial_batches",
			names:    repoNames(120),
			size:     50,
			expected: [][]string{repoNames(120)[:50], repoNames(120)[50:100], repoNames(120)[100:]},
		},
	} {
		t.Run(d.testName, func(t *testing.T) {
			t.Parallel()

			got := chunkRepositoryNames(d.names, d.size)

			if !reflect.DeepEqual(got, d.expected) {
				t.Fatalf("expected chunks %v but got %v", d.expected, got)
			}
		})
	}
}

func Test_diffRepositoryNames(t *testing.T) {
	t.Parallel()

	for _, d := range []struct {
		testName        string
		current         []string
		desired         []string
		expectedToAdd   []string
		expectedRemoves []string
	}{
		{
			testName:        "no_change",
			current:         []string{"repo-a", "repo-b"},
			desired:         []string{"repo-a", "repo-b"},
			expectedToAdd:   nil,
			expectedRemoves: nil,
		},
		{
			testName:        "adds_only",
			current:         []string{"repo-a"},
			desired:         []string{"repo-a", "repo-b"},
			expectedToAdd:   []string{"repo-b"},
			expectedRemoves: nil,
		},
		{
			testName:        "removes_only",
			current:         []string{"repo-a", "repo-b"},
			desired:         []string{"repo-a"},
			expectedToAdd:   nil,
			expectedRemoves: []string{"repo-b"},
		},
		{
			testName:        "adds_and_removes",
			current:         []string{"repo-a", "repo-b"},
			desired:         []string{"repo-b", "repo-c"},
			expectedToAdd:   []string{"repo-c"},
			expectedRemoves: []string{"repo-a"},
		},
		{
			testName:        "empty_current_and_desired",
			current:         []string{},
			desired:         []string{},
			expectedToAdd:   nil,
			expectedRemoves: nil,
		},
	} {
		t.Run(d.testName, func(t *testing.T) {
			t.Parallel()

			toAdd, toRemove := diffRepositoryNames(d.current, d.desired)

			if !reflect.DeepEqual(toAdd, d.expectedToAdd) {
				t.Fatalf("expected toAdd %v but got %v", d.expectedToAdd, toAdd)
			}
			if !reflect.DeepEqual(toRemove, d.expectedRemoves) {
				t.Fatalf("expected toRemove %v but got %v", d.expectedRemoves, toRemove)
			}
		})
	}
}

// repoNames returns n distinct repository names for use in table-driven tests.
func repoNames(n int) []string {
	names := make([]string, n)
	for i := range names {
		names[i] = "repo-" + string(rune('a'+i%26)) + string(rune('0'+i/26))
	}
	return names
}
