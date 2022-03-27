package dispatcher

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/cli/go-gh"
	"github.com/cli/go-gh/pkg/api"
	"github.com/cli/go-gh/pkg/repository"
	"github.com/pkg/errors"
)

func DispatchWorkflow(repository repository.Repository, reference string, workflowName string, inputs map[string]interface{}) error {
	client, err := gh.RESTClient(&api.ClientOptions{Host: repository.Host()})
	if err != nil {
		return errors.Wrap(err, "Unable to create GitHub client.")
	}

	body := map[string]interface{}{
		"ref":    reference,
		"inputs": inputs,
	}
	encodedBody, err := json.Marshal(body)
	if err != nil {
		return errors.Wrap(err, "Unable to marshal workflow dispatch body.")
	}

	err = client.Post(fmt.Sprintf("repos/%s/%s/actions/workflows/%s/dispatches", repository.Name(), repository.Owner(), workflowName), bytes.NewReader(encodedBody), nil)
	if err != nil {
		return errors.Wrap(err, "Unable to dispatch workflow.")
	}

	return nil
}
