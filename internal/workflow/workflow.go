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
		return nil, errors.Wrap(err, "Unable to parse workflow.")
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
					// TODO: How many of the switches can be converted to type assertions and do we need to handle errors?
					switch typedEventConfiguration := eventConfiguration.(type) {
					case map[interface{}]interface{}:
						if inputs, ok := typedEventConfiguration["inputs"]; ok {
							switch typedInputs := inputs.(type) {
							case map[interface{}]interface{}:
								for inputName, inputConfiguration := range typedInputs {
									input := Input{
										Name: inputName.(string),
									}
									if inputDescription, ok := inputConfiguration.(map[interface{}]interface{})["description"]; ok {
										input.Description = inputDescription.(string)
									}
									workflow.Inputs = append(workflow.Inputs, input)
								}
							}
						}
					}
				}
			}
		default:
			return nil, errors.Errorf("Unable to parse workflow \"on\" clause. Unexpected type %T.", on) // TODO: Should we error here?
		}
	}
	return &workflow, nil
}
