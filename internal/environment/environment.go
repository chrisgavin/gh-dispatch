package environment

import (
	"fmt"

	"github.com/chrisgavin/gh-dispatch/internal/client"
	"github.com/cli/go-gh/pkg/api"
	"github.com/cli/go-gh/pkg/repository"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type Environment struct {
	Name string `json:"name"`
}

type Environments struct {
	Environments []Environment `json:"environments"`
}

func ListEnvironments(repository repository.Repository) ([]string, error) {
	client, err := client.NewClient(repository.Host())
	if err != nil {
		return nil, err
	}

	environments := Environments{}
	if err := client.Get(fmt.Sprintf("repos/%s/%s/environments", repository.Owner(), repository.Name()), &environments); err != nil {
		if httpError, ok := err.(api.HTTPError); ok {
			if httpError.StatusCode == 404 {
				log.Warn("Got a 404 when listing environments for the repository. Unfortunately the environments API is a limited to paid organization plans.")
				return nil, nil
			}
		}
		return nil, errors.Wrap(err, "Unable to get list of environments.")
	}

	names := []string{}
	for _, environment := range environments.Environments {
		names = append(names, environment.Name)
	}

	return names, nil
}
