package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/chrisgavin/gh-dispatch/internal/default_ref"
	"github.com/chrisgavin/gh-dispatch/internal/dispatcher"
	"github.com/chrisgavin/gh-dispatch/internal/environment"
	"github.com/chrisgavin/gh-dispatch/internal/local_repository"
	"github.com/chrisgavin/gh-dispatch/internal/locator"
	"github.com/chrisgavin/gh-dispatch/internal/run"
	"github.com/chrisgavin/gh-dispatch/internal/version"
	"github.com/chrisgavin/gh-dispatch/internal/workflow"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/cli/safeexec"
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
	hostname         string
	repository       string
	ref              string
}

var rootFlags = rootFlagFields{}

var SilentErr = errors.New("SilentErr")

func defaultIfDefaultOption(defaultValue string, options []string) string {
	for _, option := range options {
		if option == defaultValue {
			return defaultValue
		}
	}
	if len(options) > 0 {
		return options[0]
	}
	return ""
}

var rootCmd = &cobra.Command{
	Use:           "gh dispatch <workflow>",
	Short:         "A GitHub CLI extension that makes it easy to dispatch GitHub Actions workflows.",
	Version:       fmt.Sprintf("%s (%s)", version.Version(), version.Commit()),
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		var workflows map[string]workflow.Workflow
		var currentRepository repository.Repository
		var reference string
		if rootFlags.repository == "" {
			gitRepository, err := git.PlainOpenWithOptions(".", &git.PlainOpenOptions{DetectDotGit: true})
			if err != nil {
				return errors.Wrap(err, "Unable to open git repository.")
			}
			currentRepository, err = repository.Current()
			if err != nil {
				return errors.Wrap(err, "Unable to determine current repository. Has it got a remote on GitHub?")
			}

			var remoteReferenceWarnings []string
			if rootFlags.ref == "" {
				reference, remoteReferenceWarnings, err = local_repository.GetCurrentRemoteHead(cmd.Context(), gitRepository)
				if err != nil {
					return err
				}
			} else {
				reference = rootFlags.ref
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

			locator := locator.LocalLocator{}
			workflows, err = locator.ListWorkflows()
			if err != nil {
				return errors.Wrap(err, "Failed to list workflows in repository.")
			}
		} else {
			fullRepository := rootFlags.repository
			if rootFlags.hostname != "" {
				fullRepository = fmt.Sprintf("%s/%s", rootFlags.hostname, fullRepository)
			}
			currentRepository, err = repository.Parse(fullRepository)
			if err != nil {
				return errors.Wrap(err, "Unable to parse repository.")
			}
			reference = rootFlags.ref
			if reference == "" {
				reference, err = default_ref.GetDefaultRef(currentRepository)
				if err != nil {
					return err
				}
			}
			locator := locator.RemoteLocator{
				Repository: currentRepository,
				Ref:        reference,
			}
			workflows, err = locator.ListWorkflows()
			if err != nil {
				return errors.Wrap(err, "Failed to list workflows in repository.")
			}
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
			sort.Strings(workflowNames)
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

		workflowData := workflows[workflowName]

		inputArguments := map[string]string{}
		for _, input := range rootFlags.inputs {
			inputParts := strings.SplitN(input, "=", 2)
			key := inputParts[0]
			value := inputParts[1]
			inputFound := false
			for _, input := range workflowData.Inputs {
				if input.Name == key {
					inputFound = true
				}
			}
			if !inputFound {
				return errors.Errorf("Input %s not accepted by workflow.", key)
			}
			inputArguments[key] = value
		}

		var environmentCache []string
		inputQuestions := []*survey.Question{}
		inputAnswers := map[string]interface{}{}
		for _, input := range workflowData.Inputs {
			if inputValue, ok := inputArguments[input.Name]; ok {
				inputAnswers[input.Name] = inputValue
			} else if !rootFlags.noPromptInputs {
				question := survey.Question{
					Name: input.Name,
				}
				message := fmt.Sprintf("Input for %s:", input.Name)
				switch input.Type {
				case workflow.StringInput:
					question.Prompt = &survey.Input{
						Message: message,
						Help:    input.Description,
						Default: input.Default,
					}
				case workflow.BooleanInput:
					question.Prompt = &survey.Confirm{
						Message: message,
						Help:    input.Description,
						Default: input.Default == "true",
					}
				case workflow.ChoiceInput:
					options := input.OptionProvider()
					question.Prompt = &survey.Select{
						Message: message,
						Help:    input.Description,
						Options: options,
						Default: defaultIfDefaultOption(input.Default, options),
					}
				case workflow.EnvironmentInput:
					if environmentCache == nil {
						environmentCache, err = environment.ListEnvironments(currentRepository)
						if err != nil {
							return err
						}
					}
					question.Prompt = &survey.Select{
						Message: message,
						Help:    input.Description,
						Options: environmentCache,
						Default: defaultIfDefaultOption(input.Default, environmentCache),
					}
				default:
					return errors.Errorf("Unhandled input type %s. This is a bug. :(", input.Type)
				}
				inputQuestions = append(inputQuestions, &question)
			}
		}
		if err := survey.Ask(inputQuestions, &inputAnswers); err != nil {
			return errors.Wrap(err, "Unable to ask for inputs.")
		}
		workflowInputs := map[string]string{}
		for key, value := range inputAnswers {
			switch typedValue := value.(type) {
			case string:
				workflowInputs[key] = typedValue
			case survey.OptionAnswer:
				workflowInputs[key] = typedValue.Value
			case bool:
				workflowInputs[key] = strconv.FormatBool(typedValue)
			default:
				return errors.Errorf("Unhandled option answer type %T. This is a bug. :(", value)
			}
		}

		log.Info("Dispatching workflow...")
		err = dispatcher.DispatchWorkflow(currentRepository, reference, workflowName, workflowInputs)
		if err != nil {
			return err
		}

		if !rootFlags.noWatch {
			log.Info("Waiting for workflow to start...")
			workflowRun, err := run.LocateRun(currentRepository, reference)
			if err != nil {
				return err
			}

			ghPath, err := safeexec.LookPath("gh")
			if err != nil {
				return errors.Wrap(err, "Unable to find gh.")
			}

			command := exec.CommandContext(cmd.Context(), ghPath, "run", "watch", "--repo", fmt.Sprintf("%s/%s/%s", currentRepository.Host, currentRepository.Owner, currentRepository.Name), strconv.FormatInt(workflowRun.ID, 10))
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
	rootCmd.Flags().StringVar(&rootFlags.hostname, "hostname", "", "The hostname of the GitHub instance.")
	rootCmd.Flags().StringVar(&rootFlags.repository, "repository", "", "The repository to dispatch the workflow on.")
	rootCmd.Flags().StringVar(&rootFlags.ref, "ref", "", "The reference to dispatch the workflow on.")

	err := rootFlags.Init(rootCmd)
	if err != nil {
		return err
	}

	if (rootFlags.hostname != "") && (rootFlags.repository == "") {
		log.Error("If --hostname is specified then --repository must also be.")
		return SilentErr
	}

	if (rootFlags.ref != "") && !strings.HasPrefix(rootFlags.ref, "refs/") {
		rootFlags.ref = fmt.Sprintf("refs/heads/%s", rootFlags.ref)
	}

	return rootCmd.ExecuteContext(ctx)
}
