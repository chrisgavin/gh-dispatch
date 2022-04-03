package workflow

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func parseTestWorkflow(t *testing.T, workflowContent string) *Workflow {
	const workflowName = "test.yml"
	workflow, err := ReadWorkflow(workflowName, []byte(workflowContent))
	require.NoError(t, err)
	require.Equal(t, workflowName, workflow.Name)
	return workflow
}

func TestReadNonDispatchableWorkflow(t *testing.T) {
	const workflowContent = `
on: push	
`
	workflow := parseTestWorkflow(t, workflowContent)
	require.False(t, workflow.Dispatchable)
}

func TestReadDispatchableWorkflowSingletonStyle(t *testing.T) {
	const workflowContent = `
on: workflow_dispatch	
`
	workflow := parseTestWorkflow(t, workflowContent)
	require.True(t, workflow.Dispatchable)
	require.Empty(t, workflow.Inputs)
}

func TestReadDispatchableWorkflowListStyle(t *testing.T) {
	const workflowContent = `
on:
  - push
  - pull_request
  - workflow_dispatch	
`
	workflow := parseTestWorkflow(t, workflowContent)
	require.True(t, workflow.Dispatchable)
	require.Empty(t, workflow.Inputs)
}

func TestReadDispatchableWorkflowMapStyle(t *testing.T) {
	const workflowContent = `
on:
  push: {}
  pull_request: {}
  workflow_dispatch: {}
`
	workflow := parseTestWorkflow(t, workflowContent)
	require.True(t, workflow.Dispatchable)
	require.Empty(t, workflow.Inputs)
}

func TestReadWorkflowWithInputs(t *testing.T) {
	const workflowContent = `
on:
  workflow_dispatch:
    inputs:
      some_input: {}
      some_input_with_description:
        description: "Some input description."
`
	workflow := parseTestWorkflow(t, workflowContent)
	require.True(t, workflow.Dispatchable)
	require.Equal(t, workflow.Inputs, []Input{
		{
			Name:        "some_input",
			Description: "",
		},
		{
			Name:        "some_input_with_description",
			Description: "Some input description.",
		},
	})
}
