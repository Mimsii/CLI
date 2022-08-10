package download

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/MakeNowJust/heredoc"
	"github.com/cli/cli/v2/pkg/cmd/run/shared"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/cli/cli/v2/pkg/prompt"
	"github.com/cli/cli/v2/pkg/set"
	"github.com/spf13/cobra"
)

type DownloadOptions struct {
	IO       *iostreams.IOStreams
	Platform platform
	Prompter prompter

	DoPrompt          bool
	OverwriteExisting bool
	SkipExisting      bool

	RunID          string
	DestinationDir string
	Names          []string
	FilePatterns   []string
}

type platform interface {
	List(runID string) ([]shared.Artifact, error)
	Download(url string, dir string, force bool, skip bool) error
}
type prompter interface {
	Prompt(message string, options []string, result interface{}) error
}

func NewCmdDownload(f *cmdutil.Factory, runF func(*DownloadOptions) error) *cobra.Command {
	opts := &DownloadOptions{
		IO: f.IOStreams,
	}

	cmd := &cobra.Command{
		Use:   "download [<run-id>]",
		Short: "Download artifacts generated by a workflow run",
		Long: heredoc.Doc(`
			Download artifacts generated by a GitHub Actions workflow run.

			The contents of each artifact will be extracted under separate directories based on
			the artifact name. If only a single artifact is specified, it will be extracted into
			the current directory.
		`),
		Args: cobra.MaximumNArgs(1),
		Example: heredoc.Doc(`
		  # Download all artifacts generated by a workflow run
		  $ gh run download <run-id>

		  # Download a specific artifact within a run
		  $ gh run download <run-id> -n <name>

		  # Download specific artifacts across all runs in a repository
		  $ gh run download -n <name1> -n <name2>

		  # Select artifacts to download interactively
		  $ gh run download
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.RunID = args[0]
			} else if len(opts.Names) == 0 &&
				len(opts.FilePatterns) == 0 &&
				opts.IO.CanPrompt() {
				opts.DoPrompt = true
			}
			if opts.OverwriteExisting && opts.SkipExisting {
				return errors.New("specify either --clobber or --skip-existing")
			}
			// support `-R, --repo` override
			baseRepo, err := f.BaseRepo()
			if err != nil {
				return err
			}
			httpClient, err := f.HttpClient()
			if err != nil {
				return err
			}
			opts.Platform = &apiPlatform{
				client: httpClient,
				repo:   baseRepo,
			}
			opts.Prompter = &surveyPrompter{}

			if runF != nil {
				return runF(opts)
			}
			return runDownload(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.DestinationDir, "dir", "D", ".", "The directory to download artifacts into")
	cmd.Flags().StringArrayVarP(&opts.Names, "name", "n", nil, "Download artifacts that match any of the given names")
	cmd.Flags().StringArrayVarP(&opts.FilePatterns, "pattern", "p", nil, "Download artifacts that match a glob pattern")
	cmd.Flags().BoolVar(&opts.OverwriteExisting, "clobber", false, "Overwrite existing assets of the same name")
	cmd.Flags().BoolVar(&opts.SkipExisting, "skip-existing", false, "Skip existing assets of the same name")

	return cmd
}

func runDownload(opts *DownloadOptions) error {
	opts.IO.StartProgressIndicator()
	artifacts, err := opts.Platform.List(opts.RunID)
	opts.IO.StopProgressIndicator()
	if err != nil {
		return fmt.Errorf("error fetching artifacts: %w", err)
	}

	numValidArtifacts := 0
	for _, a := range artifacts {
		if a.Expired {
			continue
		}
		numValidArtifacts++
	}
	if numValidArtifacts == 0 {
		return errors.New("no valid artifacts found to download")
	}

	wantPatterns := opts.FilePatterns
	wantNames := opts.Names
	if opts.DoPrompt {
		artifactNames := set.NewStringSet()
		for _, a := range artifacts {
			if !a.Expired {
				artifactNames.Add(a.Name)
			}
		}
		options := artifactNames.ToSlice()
		if len(options) > 10 {
			options = options[:10]
		}
		err := opts.Prompter.Prompt("Select artifacts to download:", options, &wantNames)
		if err != nil {
			return err
		}
		if len(wantNames) == 0 {
			return errors.New("no artifacts selected")
		}
	}

	opts.IO.StartProgressIndicator()
	defer opts.IO.StopProgressIndicator()

	// track downloaded artifacts and avoid re-downloading any of the same name
	downloaded := set.NewStringSet()
	for _, a := range artifacts {
		if a.Expired {
			continue
		}
		if downloaded.Contains(a.Name) {
			continue
		}
		if len(wantNames) > 0 || len(wantPatterns) > 0 {
			if !matchAnyName(wantNames, a.Name) && !matchAnyPattern(wantPatterns, a.Name) {
				continue
			}
		}
		destDir := opts.DestinationDir
		if len(wantPatterns) != 0 || len(wantNames) != 1 {
			destDir = filepath.Join(destDir, a.Name)
		}
		err := opts.Platform.Download(a.DownloadURL, destDir, opts.OverwriteExisting, opts.SkipExisting)
		if err != nil {
			return fmt.Errorf("error downloading %s: %w", a.Name, err)
		}
		downloaded.Add(a.Name)
	}

	if downloaded.Len() == 0 {
		return errors.New("no artifact matches any of the names or patterns provided")
	}

	return nil
}

func matchAnyName(names []string, name string) bool {
	for _, n := range names {
		if name == n {
			return true
		}
	}
	return false
}

func matchAnyPattern(patterns []string, name string) bool {
	for _, p := range patterns {
		if isMatch, err := filepath.Match(p, name); err == nil && isMatch {
			return true
		}
	}
	return false
}

type surveyPrompter struct{}

func (sp *surveyPrompter) Prompt(message string, options []string, result interface{}) error {
	return prompt.SurveyAskOne(&survey.MultiSelect{
		Message: message,
		Options: options,
	}, result)
}
