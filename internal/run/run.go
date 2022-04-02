package run

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/chrisgavin/gh-dispatch/internal/client"
	"github.com/cli/go-gh/pkg/api"
	"github.com/cli/go-gh/pkg/repository"
	"github.com/pkg/errors"
)

type User struct {
	Login string `json:"login"`
}

type WorkflowRun struct {
	ID         int64  `json:"id"`
	Conclusion string `json:"conclusion"`
}

type WorkflowRuns struct {
	WorkflowRuns []WorkflowRun `json:"workflow_runs"`
}

func findRun(client api.RESTClient, repository repository.Repository, reference string, after time.Time, before time.Time) (*WorkflowRun, error) {
	user := User{}
	if err := client.Get("user", &user); err != nil {
		return nil, errors.Wrap(err, "Unable to get the current user.")
	}

	urlParameters := url.Values{}
	urlParameters.Add("actor", user.Login)
	urlParameters.Add("branch", strings.TrimPrefix(reference, "refs/heads/"))
	urlParameters.Add("event", "workflow_dispatch")
	createdRange := fmt.Sprintf("%s..%s", after.Format(time.RFC3339), before.Format(time.RFC3339))
	urlParameters.Add("created", createdRange)

	workflowRuns := WorkflowRuns{}
	if err := client.Get(fmt.Sprintf("repos/%s/%s/actions/runs?%s", repository.Owner(), repository.Name(), urlParameters.Encode()), &workflowRuns); err != nil {
		return nil, errors.Wrap(err, "Unable to get list of recent runs.")
	}

	for _, run := range workflowRuns.WorkflowRuns {
		return &run, nil
	}
	return nil, nil
}

func LocateRun(repository repository.Repository, reference string) (*WorkflowRun, error) {
	client, err := client.NewClient(repository.Host())
	if err != nil {
		return nil, err
	}

	currentTime := time.Now()
	after := currentTime.Add(-1 * time.Minute)
	before := currentTime.Add(1 * time.Minute)

	for {
		run, err := findRun(client, repository, reference, after, before)
		if err != nil {
			return nil, err
		}
		if run != nil {
			return run, nil
		}
		if time.Now().After(currentTime.Add(1 * time.Minute)) {
			return nil, errors.New("Workflow did not start within 1 minute.")
		}
		time.Sleep(3 * time.Second)
	}
}

func GetRun(repository repository.Repository, id int64) (*WorkflowRun, error) {
	client, err := client.NewClient(repository.Host())
	if err != nil {
		return nil, err
	}

	workflowRun := WorkflowRun{}
	if err := client.Get(fmt.Sprintf("repos/%s/%s/actions/runs/%d", repository.Owner(), repository.Name(), id), &workflowRun); err != nil {
		return nil, errors.Wrap(err, "Unable to get workflow run.")
	}
	return &workflowRun, nil
}
