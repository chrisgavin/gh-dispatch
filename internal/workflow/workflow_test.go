package workflow

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func parseTestWorkflow(t *testing.T, workflowContent string) *Workflow {
	const workflowName = "test.yml"
	workflowData, err := ReadWorkflow(workflowName, []byte(workflowContent))
	require.NoError(t, err)
	require.Equal(t, workflowName, workflowData.Name)
	return workflowData
}

func TestReadNonDispatchableWorkflow(t *testing.T) {
	const workflowContent = `
on: push	
`
	workflowData := parseTestWorkflow(t, workflowContent)
	require.False(t, workflowData.Dispatchable)
}

func TestReadDispatchableWorkflowSingletonStyle(t *testing.T) {
	const workflowContent = `
on: workflow_dispatch	
`
	workflowData := parseTestWorkflow(t, workflowContent)
	require.True(t, workflowData.Dispatchable)
	require.Empty(t, workflowData.Inputs)
}

func TestReadDispatchableWorkflowListStyle(t *testing.T) {
	const workflowContent = `
on:
  - push
  - pull_request
  - workflow_dispatch	
`
	workflowData := parseTestWorkflow(t, workflowContent)
	require.True(t, workflowData.Dispatchable)
	require.Empty(t, workflowData.Inputs)
}

func TestReadDispatchableWorkflowMapStyle(t *testing.T) {
	const workflowContent = `
on:
  push: {}
  pull_request: {}
  workflow_dispatch: {}
`
	workflowData := parseTestWorkflow(t, workflowContent)
	require.True(t, workflowData.Dispatchable)
	require.Empty(t, workflowData.Inputs)
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
	workflowData := parseTestWorkflow(t, workflowContent)
	require.True(t, workflowData.Dispatchable)
	require.Equal(t, []Input{
		{
			Name:        "some_input",
			Description: "",
			Type:        StringInput,
		},
		{
			Name:        "some_input_with_description",
			Description: "Some input description.",
			Type:        StringInput,
		},
	}, workflowData.Inputs)
}

func TestReadWorkflowWithBooleanInputs(t *testing.T) {
	const workflowContent = `
on:
  workflow_dispatch:
    inputs:
      some_input:
        type: boolean
`
	workflowData := parseTestWorkflow(t, workflowContent)
	require.True(t, workflowData.Dispatchable)
	require.Equal(t, 1, len(workflowData.Inputs))
	require.Equal(t, BooleanInput, workflowData.Inputs[0].Type)
}

func TestReadWorkflowWithChoiceInputs(t *testing.T) {
	const workflowContent = `
on:
  workflow_dispatch:
    inputs:
      some_input:
        type: choice
        options: [foo, bar]
`
	workflowData := parseTestWorkflow(t, workflowContent)
	require.True(t, workflowData.Dispatchable)
	require.Equal(t, 1, len(workflowData.Inputs))
	require.Equal(t, ChoiceInput, workflowData.Inputs[0].Type)
	require.Equal(t, []string{"foo", "bar"}, workflowData.Inputs[0].OptionProvider())
}

func TestReadWorkflowWithEnvironmentInputs(t *testing.T) {
	const workflowContent = `
on:
  workflow_dispatch:
    inputs:
      some_input:
        type: environment
`
	workflowData, err := ReadWorkflow("test.yml", []byte(workflowContent))
	require.NoError(t, err)
	require.True(t, workflowData.Dispatchable)
	require.Equal(t, 1, len(workflowData.Inputs))
	require.Equal(t, EnvironmentInput, workflowData.Inputs[0].Type)
}

func TestReadWorkflowWithDefaultValueInputs(t *testing.T) {
	const workflowContent = `
on:
  workflow_dispatch:
    inputs:
      some_input:
        default: foo
`
	workflowData, err := ReadWorkflow("test.yml", []byte(workflowContent))
	require.NoError(t, err)
	require.True(t, workflowData.Dispatchable)
	require.Equal(t, 1, len(workflowData.Inputs))
	require.Equal(t, "foo", workflowData.Inputs[0].Default)
}
