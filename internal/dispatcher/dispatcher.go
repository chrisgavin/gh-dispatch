package dispatcher

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/chrisgavin/gh-dispatch/internal/client"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/pkg/errors"
)

type DispatchedRun struct {
	ID      int64  `json:"workflow_run_id"`
	HTMLURL string `json:"html_url"`
}

func DispatchWorkflow(repository repository.Repository, reference string, workflowName string, inputs map[string]string) (*DispatchedRun, error) {
	client, err := client.NewClient(repository.Host)
	if err != nil {
		return nil, err
	}

	body := map[string]interface{}{
		"ref":                reference,
		"inputs":             inputs,
		"return_run_details": true,
	}
	encodedBody, err := json.Marshal(body)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to marshal workflow dispatch body.")
	}

	dispatchedRun := DispatchedRun{}
	err = client.Post(fmt.Sprintf("repos/%s/%s/actions/workflows/%s/dispatches", repository.Owner, repository.Name, workflowName), bytes.NewReader(encodedBody), &dispatchedRun)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to dispatch workflow.")
	}
	if dispatchedRun.ID == 0 {
		return nil, errors.New("Workflow dispatch did not return a run ID.")
	}

	return &dispatchedRun, nil
}
