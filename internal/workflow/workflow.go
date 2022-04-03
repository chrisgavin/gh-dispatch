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
			for event, eventConfiguration := range typedOn {
				if event == workflowDispatch {
					workflow.Dispatchable = true
					if eventConfiguration != nil {
						typedEventConfiguration, ok := eventConfiguration.(map[interface{}]interface{})
						if !ok {
							return nil, errors.Errorf("Workflow dispatch configuration had unexpected type %T.", eventConfiguration)
						}
						if inputs, ok := typedEventConfiguration["inputs"]; ok {
							typedInputs, ok := inputs.(map[interface{}]interface{})
							if !ok {
								return nil, errors.Errorf("Workflow dispatch configuration inputs had unexpected type %T.", inputs)
							}
							for inputName, inputConfiguration := range typedInputs {
								typedInputConfiguration, ok := inputConfiguration.(map[interface{}]interface{})
								if !ok {
									return nil, errors.Errorf("Input configuration for %s had unexpected type %T.", inputName, inputConfiguration)
								}
								input := Input{
									Name: inputName.(string),
								}
								if inputDescription, ok := typedInputConfiguration["description"]; ok {
									input.Description, ok = inputDescription.(string)
									if !ok {
										return nil, errors.Errorf("Input description for %s had unexpected type %T.", inputName, inputDescription)
									}
								}
								workflow.Inputs = append(workflow.Inputs, input)
							}
						}
					}
				}
			}
		default:
			return nil, errors.Errorf("Unable to parse workflow \"on\" clause. Unexpected type %T.", on)
		}
	}
	return &workflow, nil
}
