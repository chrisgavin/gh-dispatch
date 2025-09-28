package locator

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

	"github.com/chrisgavin/gh-dispatch/internal/client"
	"github.com/chrisgavin/gh-dispatch/internal/workflow"
	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type RemoteLocator struct {
	Repository repository.Repository
	Ref        string
}

type apiWorkflow struct {
	Path string `json:"path"`
}

type apiWorkflows struct {
	Workflows []apiWorkflow `json:"workflows"`
}

type apiFile struct {
	Content string `json:"content"`
}

type concurrentWorkflowResult struct {
	workflow *workflow.Workflow
	err      error
}

func (locator RemoteLocator) ListWorkflows() (map[string]workflow.Workflow, error) {
	client, err := client.NewClient(locator.Repository.Host)
	if err != nil {
		return nil, err
	}

	apiWorkflows := apiWorkflows{}
	if err := client.Get(fmt.Sprintf("repos/%s/%s/actions/workflows?per_page=100", locator.Repository.Owner, locator.Repository.Name), &apiWorkflows); err != nil {
		return nil, errors.Wrap(err, "Unable to list workflows.")
	}

	channel := make(chan concurrentWorkflowResult)
	for _, apiWorkflowValue := range apiWorkflows.Workflows {
		go func(apiWorkflowValue apiWorkflow) {
			if apiWorkflowValue.Path == "" {
				// It's not totally clear why this happens. It could happen for dynamic workflows, but I've also seen it in other cases.
				channel <- concurrentWorkflowResult{}
				return
			}
			apiFile := apiFile{}
			urlParameters := url.Values{}
			if locator.Ref != "" {
				urlParameters.Add("ref", locator.Ref)
			}
			if err := client.Get(fmt.Sprintf("repos/%s/%s/contents/%s?%s", locator.Repository.Owner, locator.Repository.Name, apiWorkflowValue.Path, urlParameters.Encode()), &apiFile); err != nil {
				if httpError, ok := err.(*api.HTTPError); ok && httpError.StatusCode == 404 {
					// This can happen when the workflow exists on the default branch but not on the ref we're dispatching against.
					channel <- concurrentWorkflowResult{}
					return
				}
				channel <- concurrentWorkflowResult{
					err: errors.Wrapf(err, "Unable to get workflow content for workflow %s.", apiWorkflowValue.Path),
				}
				return
			}
			content, err := base64.StdEncoding.DecodeString(apiFile.Content)
			if err != nil {
				channel <- concurrentWorkflowResult{
					err: errors.Wrapf(err, "Unable to decode workflow content for workflow %s.", apiWorkflowValue.Path),
				}
				return
			}
			parts := strings.Split(apiWorkflowValue.Path, "/")
			loaded, err := workflow.ReadWorkflow(parts[len(parts)-1], content)
			if err != nil {
				log.Warnf("Workflow \"%s\" is invalid: %s", apiWorkflowValue.Path, err)
				channel <- concurrentWorkflowResult{}
				return
			}
			if !loaded.Dispatchable {
				channel <- concurrentWorkflowResult{}
				return
			}

			channel <- concurrentWorkflowResult{
				workflow: loaded,
			}
		}(apiWorkflowValue)
	}

	workflows := map[string]workflow.Workflow{}
	for i := 0; i < len(apiWorkflows.Workflows); i++ {
		result := <-channel
		if result.err != nil {
			return nil, result.err
		}
		if result.workflow != nil {
			workflows[result.workflow.Name] = *result.workflow
		}
	}

	return workflows, nil
}
