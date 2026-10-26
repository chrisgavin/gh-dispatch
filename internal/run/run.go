package run

import (
	"fmt"

	"github.com/chrisgavin/gh-dispatch/internal/client"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/pkg/errors"
)

type WorkflowRun struct {
	ID         int64  `json:"id"`
	Conclusion string `json:"conclusion"`
}

func GetRun(repository repository.Repository, id int64) (*WorkflowRun, error) {
	client, err := client.NewClient(repository.Host)
	if err != nil {
		return nil, err
	}

	workflowRun := WorkflowRun{}
	if err := client.Get(fmt.Sprintf("repos/%s/%s/actions/runs/%d", repository.Owner, repository.Name, id), &workflowRun); err != nil {
		return nil, errors.Wrap(err, "Unable to get workflow run.")
	}
	return &workflowRun, nil
}
