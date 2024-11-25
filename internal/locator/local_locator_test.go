package locator_test

import (
	"testing"

	"github.com/chrisgavin/gh-dispatch/internal/locator"
)

func TestLocalLocatorWorkflows(t *testing.T) {
	l := locator.LocalLocator{}
	workflows, err := l.ListWorkflows()
	if err != nil {
		t.Fatalf("Error listing workflows: %s", err)
	}
	if len(workflows) == 0 {
		t.Fatalf("Expected at least one workflow, got none.")
	}
}
