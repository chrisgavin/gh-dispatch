package locator

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/chrisgavin/gh-dispatch/internal/workflow"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func ListWorkflowsInRepository() (map[string]workflow.Workflow, error) {
	repositoryRoot, err := getRepositoryRoot()
	if err != nil {
		return nil, err
	}
	workflowsDirectory := path.Join(repositoryRoot, workflow.WorkflowsPath)
	entries, err := ioutil.ReadDir(workflowsDirectory)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to list workflows in workflows directory.")
	}

	workflows := map[string]workflow.Workflow{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		isWorkflow := false
		for _, extension := range workflow.WorkflowExtensions() {
			if strings.HasSuffix(entry.Name(), extension) {
				isWorkflow = true
				break
			}
		}
		if !isWorkflow {
			continue
		}
		bytes, err := ioutil.ReadFile(path.Join(workflowsDirectory, entry.Name()))
		if err != nil {
			return nil, errors.Wrap(err, "Unable to read workflow file.")
		}
		loaded, err := workflow.ReadWorkflow(entry.Name(), bytes)
		if err != nil {
			log.Warnf("Workflow \"%s\" is invalid: %s", entry.Name(), err)
			continue
		}
		if !loaded.Dispatchable {
			continue
		}
		workflows[loaded.Name] = *loaded
	}

	return workflows, nil
}

func getRepositoryRoot() (string, error) {
	path, err := filepath.Abs(".")
	if err != nil {
		return "", errors.Wrap(err, "Unable to get absolute path of current working directory.")
	}
	for {
		gitDirectory := filepath.Join(path, ".git")
		if stat, err := os.Stat(gitDirectory); err != nil {
			if !os.IsNotExist(err) {
				return "", errors.Wrapf(err, "Unable to stat git directory \"%s\".", gitDirectory)
			}
			continue
		} else {
			if stat.IsDir() {
				return path, nil
			}
		}
		parent := filepath.Dir(path)
		if parent == path {
			return "", errors.New("Unable to find repository root.")
		}
		path = parent
	}
}
