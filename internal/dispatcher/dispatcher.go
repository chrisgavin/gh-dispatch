package dispatcher

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/chrisgavin/gh-dispatch/internal/client"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/pkg/errors"
)

func DispatchWorkflow(repository repository.Repository, reference string, workflowName string, inputs map[string]string) error {
	client, err := client.NewClient(repository.Host)
	if err != nil {
		return err
	}

	body := map[string]interface{}{
		"ref":    reference,
		"inputs": inputs,
	}
	encodedBody, err := json.Marshal(body)
	if err != nil {
		return errors.Wrap(err, "Unable to marshal workflow dispatch body.")
	}

	err = client.Post(fmt.Sprintf("repos/%s/%s/actions/workflows/%s/dispatches", repository.Owner, repository.Name, workflowName), bytes.NewReader(encodedBody), nil)
	if err != nil {
		return errors.Wrap(err, "Unable to dispatch workflow.")
	}

	return nil
}
