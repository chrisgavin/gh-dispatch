package local_repository

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/pkg/errors"
)

func GetCurrentRemoteHead(ctx context.Context, gitRepository *git.Repository) (string, []string, error) {
	warnings := []string{}

	hasUncommittedChanges, err := hasUncommittedChanges(ctx, gitRepository)
	if err != nil {
		return "", nil, err
	}
	if hasUncommittedChanges {
		warnings = append(warnings, "uncommitted changes")
	}

	repositoryConfiguration, err := gitRepository.Config()
	if err != nil {
		return "", nil, errors.Wrap(err, "Unable to get git repository configuration.")
	}
	head, err := gitRepository.Head()
	if err != nil {
		return "", nil, errors.Wrap(err, "Unable to get repository HEAD.")
	}
	remoteConfiguration, ok := repositoryConfiguration.Branches[head.Name().Short()]
	if !ok {
		return "", nil, errors.New("Unable to get remote configuration for the current branch. Has it been pushed to GitHub?")
	}

	hasUnpushedChanges, err := hasUnpushedChanges(gitRepository, remoteConfiguration, head)
	if err != nil {
		return "", nil, err
	}
	if hasUnpushedChanges {
		warnings = append(warnings, "unpushed changes")
	}

	return remoteConfiguration.Merge.String(), warnings, nil
}

func hasUncommittedChanges(ctx context.Context, gitRepository *git.Repository) (bool, error) {
	gitWorktree, err := gitRepository.Worktree()
	if err != nil {
		return false, errors.Wrap(err, "Unable to get git worktree.")
	}

	// The performance of go-git's status command is not great (https://github.com/src-d/go-git/issues/844), and it can also return incorrect results for nested Git repositories, so we just shell out to regular Git here.
	command := exec.CommandContext(ctx, "git", "status", "--porcelain=v1")
	command.Dir = gitWorktree.Filesystem.Root()
	output, err := command.Output()
	if err != nil {
		return false, errors.Wrap(err, "Unable to get Git status.")
	}

	return len(output) > 0, nil
}

func hasUnpushedChanges(gitRepository *git.Repository, remoteConfiguration *config.Branch, head *plumbing.Reference) (bool, error) {
	remoteReference, err := gitRepository.Reference(plumbing.ReferenceName(fmt.Sprintf("refs/remotes/%s/%s", remoteConfiguration.Remote, remoteConfiguration.Merge.Short())), true)
	if err != nil {
		return false, errors.Wrap(err, "Unable to get remote reference.")
	}
	return head.Hash() != remoteReference.Hash(), nil
}
