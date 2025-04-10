package workflow

import (
	"fmt"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type InputType string

const (
	StringInput      InputType = "string"
	BooleanInput     InputType = "boolean"
	ChoiceInput      InputType = "choice"
	EnvironmentInput InputType = "environment"
)

var inputTypesMap = map[string]InputType{
	string(StringInput):      StringInput,
	string(BooleanInput):     BooleanInput,
	string(ChoiceInput):      ChoiceInput,
	string(EnvironmentInput): EnvironmentInput,
}

type Input struct {
	Name           string
	Description    string
	Type           InputType
	OptionProvider func() []string
	Default        string
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
			if workflowDispatchTrigger, ok := typedOn[workflowDispatch]; ok && workflowDispatchTrigger == nil {
				workflow.Dispatchable = true
			} else if typedParsedWorkflow.On.WorkflowDispatch != nil {
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
						input.Type = StringInput
						if inputType, ok := mapInputConfiguration["type"]; ok {
							typedInputType, ok := inputType.(string)
							if !ok {
								return nil, errors.Errorf("Input type for %s had unexpected type %T.", inputName, inputType)
							}
							if input.Type, ok = inputTypesMap[typedInputType]; !ok {
								log.Warnf("Input %s has unknown type %s.", input.Name, inputType)
							} else {
								if input.Type == ChoiceInput {
									if inputOptions, ok := mapInputConfiguration["options"]; ok {
										if typedInputOptions, ok := inputOptions.([]interface{}); ok {
											input.OptionProvider = func() []string {
												choices := []string{}
												for _, inputOption := range typedInputOptions {
													choices = append(choices, fmt.Sprintf("%v", inputOption))
												}
												return choices
											}
										} else {
											return nil, errors.Errorf("Input options for %s had unexpected type %T.", input.Name, inputOptions)
										}
									} else {
										return nil, errors.Errorf("Input %s is a choice input but has no options property.", input.Name)
									}
								}
							}
						}
						if inputDefault, ok := mapInputConfiguration["default"]; ok {
							input.Default = fmt.Sprintf("%v", inputDefault)
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
