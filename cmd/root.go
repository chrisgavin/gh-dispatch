package cmd

import (
	"context"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/chrisgavin/gh-dispatch/internal/dispatcher"
	"github.com/chrisgavin/gh-dispatch/internal/locator"
	"github.com/chrisgavin/gh-dispatch/internal/version"
	"github.com/cli/go-gh"
	"github.com/go-git/go-git/v5"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type rootFlagFields struct {
}

var rootFlags = rootFlagFields{}

var SilentErr = errors.New("SilentErr")

var rootCmd = &cobra.Command{
	Short:         "A GitHub CLI extension that makes it easy to dispatch GitHub Actions workflows.",
	Version:       fmt.Sprintf("%s (%s)", version.Version(), version.Commit()),
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		workflows, err := locator.ListWorkflowsInRepository()
		if err != nil {
			return errors.Wrap(err, "Failed to list workflows in repository.")
		}
		if len(workflows) == 0 {
			log.Error("No dispatchable workflows found in repository.")
			return SilentErr
		}
		workflowNames := []string{}
		for workflowName := range workflows {
			workflowNames = append(workflowNames, workflowName)
		}
		workflowQuestion := &survey.Select{
			Message: "What workflow do you want to dispatch?",
			Options: workflowNames,
		}

		var workflowName string
		if err := survey.AskOne(workflowQuestion, &workflowName); err != nil {
			return errors.Wrap(err, "Unable to ask for workflow.")
		}

		workflow := workflows[workflowName]

		inputQuestions := []*survey.Question{}
		for _, input := range workflow.Inputs {
			inputQuestions = append(inputQuestions, &survey.Question{
				Name: input.Name,
				Prompt: &survey.Input{
					Message: fmt.Sprintf("Input for %s:", input.Name),
					Help:    input.Description,
				},
			})
		}
		inputAnswers := map[string]interface{}{}
		if err := survey.Ask(inputQuestions, &inputAnswers); err != nil {
			return errors.Wrap(err, "Unable to ask for inputs.")
		}

		currentRepository, err := gh.CurrentRepository()
		if err != nil {
			return errors.Wrap(err, "Unable to determine current repository. Has it got a remote on GitHub?")
		}

		gitRepository, err := git.PlainOpen(".")
		if err != nil {
			return errors.Wrap(err, "Unable to open git repository.")
		}
		head, err := gitRepository.Head()
		if err != nil {
			return errors.Wrap(err, "Unable to get repository HEAD.")
		}
		reference := head.Name().String()

		err = dispatcher.DispatchWorkflow(currentRepository, reference, workflowName, inputAnswers)
		if err != nil {
			return err
		}
		return nil
	},
}

func (f *rootFlagFields) Init(cmd *cobra.Command) error {
	cmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		cmd.PrintErrln(err)
		cmd.PrintErrln()
		cmd.PrintErr(cmd.UsageString())
		return SilentErr
	})

	return nil
}

func Execute(ctx context.Context) error {
	err := rootFlags.Init(rootCmd)
	if err != nil {
		return err
	}

	return rootCmd.ExecuteContext(ctx)
}
