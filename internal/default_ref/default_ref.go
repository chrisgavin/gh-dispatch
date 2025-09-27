package default_ref

import (
	"fmt"

	"github.com/chrisgavin/gh-dispatch/internal/client"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/pkg/errors"
)

type gitHubRepository struct {
	DefaultBranch string `json:"default_branch"`
}

func GetDefaultRef(currentRepository repository.Repository) (string, error) {
	client, err := client.NewClient(currentRepository.Host())
	if err != nil {
		return "", err
	}
	repository := gitHubRepository{}
	err = client.Get(fmt.Sprintf("repos/%s/%s", currentRepository.Owner(), currentRepository.Name()), &repository)
	if err != nil {
		return "", errors.Wrap(err, "Unable to get default ref.")
	}
	return fmt.Sprintf("refs/heads/%s", repository.DefaultBranch), nil
}
