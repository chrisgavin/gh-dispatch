package workflow

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type Input struct {
	Name        string
	Description string
}

type Workflow struct {
	Name         string
	Dispatchable bool
	Inputs       []Input
}

const workflowDispatch = "workflow_dispatch"

type workflowDispatchTrigger struct {
	Inputs *yaml.MapSlice `yaml:"inputs"`
}

type workflowTriggers struct {
	WorkflowDispatch *workflowDispatchTrigger `yaml:"workflow_dispatch"`
}

type workflowInternal struct {
	On workflowTriggers `yaml:"on"`
}

func ReadWorkflow(name string, rawWorkflow []byte) (*Workflow, error) {
	workflow := Workflow{
		Name: name,
	}
	parsed := make(map[string]interface{})
	err := yaml.Unmarshal(rawWorkflow, &parsed)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to parse workflow as YAML.")
	}
	if on, ok := parsed["on"]; ok {
		switch typedOn := on.(type) {
		case string:
			workflow.Dispatchable = on == workflowDispatch
		case []interface{}:
			for _, event := range typedOn {
				if event == workflowDispatch {
					workflow.Dispatchable = true
				}
			}
		case map[interface{}]interface{}:
			// We want to preserve the order of inputs, so in this case we re-parse the workflow using the internal types specifically meant for preserving order.
			typedParsedWorkflow := workflowInternal{}
			err = yaml.Unmarshal(rawWorkflow, &typedParsedWorkflow)
			if err != nil {
				return nil, errors.Wrap(err, "Unable to parse workflow as typed YAML.")
			}
			if typedParsedWorkflow.On.WorkflowDispatch != nil {
				workflow.Dispatchable = true
				if typedParsedWorkflow.On.WorkflowDispatch.Inputs != nil {
					for _, inputData := range *typedParsedWorkflow.On.WorkflowDispatch.Inputs {
						inputName := inputData.Key
						inputConfiguration := inputData.Value
						typedInputConfiguration, ok := inputConfiguration.(yaml.MapSlice)
						if !ok {
							return nil, errors.Errorf("Input configuration for %s had unexpected type %T.", inputName, inputConfiguration)
						}
						mapInputConfiguration := map[interface{}]interface{}{}
						for _, inputConfigurationData := range typedInputConfiguration {
							mapInputConfiguration[inputConfigurationData.Key] = inputConfigurationData.Value
						}
						input := Input{
							Name: inputName.(string),
						}
						if inputDescription, ok := mapInputConfiguration["description"]; ok {
							input.Description, ok = inputDescription.(string)
							if !ok {
								return nil, errors.Errorf("Input description for %s had unexpected type %T.", inputName, inputDescription)
							}
						}
						workflow.Inputs = append(workflow.Inputs, input)
					}
				}
			}
		default:
			return nil, errors.Errorf("Unable to parse workflow \"on\" clause. Unexpected type %T.", on)
		}
	}
	return &workflow, nil
}
