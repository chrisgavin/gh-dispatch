package locator

import "github.com/chrisgavin/gh-dispatch/internal/workflow"

type Locator interface {
	ListWorkflows() (map[string]workflow.Workflow, error)
}
