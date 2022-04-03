package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/chrisgavin/gh-dispatch/internal/dispatcher"
	"github.com/chrisgavin/gh-dispatch/internal/local_repository"
	"github.com/chrisgavin/gh-dispatch/internal/locator"
	"github.com/chrisgavin/gh-dispatch/internal/run"
	"github.com/chrisgavin/gh-dispatch/internal/version"
	"github.com/cli/go-gh"
	"github.com/go-git/go-git/v5"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type rootFlagFields struct {
	noWatch          bool
	inputs           []string
	noPromptInputs   bool
	noPromptUnpushed bool
}

var rootFlags = rootFlagFields{}

var SilentErr = errors.New("SilentErr")

var rootCmd = &cobra.Command{
	Use:           "gh dispatch <workflow>",
	Short:         "A GitHub CLI extension that makes it easy to dispatch GitHub Actions workflows.",
	Version:       fmt.Sprintf("%s (%s)", version.Version(), version.Commit()),
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		gitRepository, err := git.PlainOpen(".")
		if err != nil {
			return errors.Wrap(err, "Unable to open git repository.")
		}
		remoteReference, remoteReferenceWarnings, err := local_repository.GetCurrentRemoteHead(cmd.Context(), gitRepository)
		if err != nil {
			return err
		}
		if len(remoteReferenceWarnings) > 0 && !rootFlags.noPromptUnpushed {
			antepenultimateIndex := len(remoteReferenceWarnings) - 2
			if antepenultimateIndex < 0 {
				antepenultimateIndex = 0
			}
			remoteReferenceWarningsString := strings.Join(append(remoteReferenceWarnings[:antepenultimateIndex], strings.Join(remoteReferenceWarnings[antepenultimateIndex:], " and ")), ", ")
			remoteReferenceWarningQuestion := &survey.Confirm{
				Message: fmt.Sprintf("You currently have %s. Would you still like to dispatch a workflow?", remoteReferenceWarningsString),
			}

			var remoteReferenceWarningAnswer bool
			if err := survey.AskOne(remoteReferenceWarningQuestion, &remoteReferenceWarningAnswer); err != nil {
				return errors.Wrap(err, "Unable to ask whether to continue despite warnings about the remote head.")
			}
			if !remoteReferenceWarningAnswer {
				log.Error("Aborting.")
				os.Exit(1)
			}
		}

		workflows, err := locator.ListWorkflowsInRepository()
		if err != nil {
			return errors.Wrap(err, "Failed to list workflows in repository.")
		}
		if len(workflows) == 0 {
			log.Error("No dispatchable workflows found in repository.")
			return SilentErr
		}

		var workflowName string
		if len(args) == 0 {
			workflowNames := []string{}
			for workflowName := range workflows {
				workflowNames = append(workflowNames, workflowName)
			}
			workflowQuestion := &survey.Select{
				Message: "What workflow do you want to dispatch?",
				Options: workflowNames,
			}

			if err := survey.AskOne(workflowQuestion, &workflowName); err != nil {
				return errors.Wrap(err, "Unable to ask for workflow.")
			}
		} else if len(args) == 1 {
			workflowPathParts := strings.Split(args[0], "/")
			workflowName = workflowPathParts[len(workflowPathParts)-1]
		} else {
			return errors.New("Too many arguments.")
		}

		workflow := workflows[workflowName]

		inputArguments := map[string]string{}
		for _, input := range rootFlags.inputs {
			inputParts := strings.SplitN(input, "=", 2)
			key := inputParts[0]
			value := inputParts[1]
			inputFound := false
			for _, input := range workflow.Inputs {
				if input.Name == key {
					inputFound = true
				}
			}
			if !inputFound {
				return errors.Errorf("Input %s not accepted by workflow.", key)
			}
			inputArguments[key] = value
		}

		inputQuestions := []*survey.Question{}
		inputAnswers := map[string]interface{}{}
		for _, input := range workflow.Inputs {
			if inputValue, ok := inputArguments[input.Name]; ok {
				inputAnswers[input.Name] = inputValue
			} else if !rootFlags.noPromptInputs {
				inputQuestions = append(inputQuestions, &survey.Question{
					Name: input.Name,
					Prompt: &survey.Input{
						Message: fmt.Sprintf("Input for %s:", input.Name),
						Help:    input.Description,
					},
				})
			}
		}
		if err := survey.Ask(inputQuestions, &inputAnswers); err != nil {
			return errors.Wrap(err, "Unable to ask for inputs.")
		}

		currentRepository, err := gh.CurrentRepository()
		if err != nil {
			return errors.Wrap(err, "Unable to determine current repository. Has it got a remote on GitHub?")
		}

		log.Info("Dispatching workflow...")
		err = dispatcher.DispatchWorkflow(currentRepository, remoteReference, workflowName, inputAnswers)
		if err != nil {
			return err
		}

		if !rootFlags.noWatch {
			log.Info("Waiting for workflow to start...")
			workflowRun, err := run.LocateRun(currentRepository, remoteReference)
			if err != nil {
				return err
			}

			command := exec.CommandContext(cmd.Context(), "gh", "run", "watch", strconv.FormatInt(workflowRun.ID, 10))
			command.Stdout = os.Stdout
			command.Stderr = os.Stderr
			err = command.Run()
			if err != nil {
				return errors.Wrap(err, "Unable to watch workflow progress.")
			}

			workflowRun, err = run.GetRun(currentRepository, workflowRun.ID)
			if err != nil {
				return err
			}
			log.Infof("Workflow completed with conclusion %s.", workflowRun.Conclusion)
			if workflowRun.Conclusion != "success" {
				os.Exit(1)
			}
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
	rootCmd.Flags().BoolVar(&rootFlags.noWatch, "no-watch", false, "Do not wait for the workflow to complete.")
	rootCmd.Flags().StringSliceVar(&rootFlags.inputs, "input", nil, "Inputs to pass to the workflow, as `key=value`.")
	rootCmd.Flags().BoolVar(&rootFlags.noPromptInputs, "no-prompt-inputs", false, "Do not prompt for any inputs to the workflow.")
	rootCmd.Flags().BoolVar(&rootFlags.noPromptUnpushed, "no-prompt-unpushed", false, "Do not warn about any uncommitted or unpushed changes.")

	err := rootFlags.Init(rootCmd)
	if err != nil {
		return err
	}

	return rootCmd.ExecuteContext(ctx)
}
