package get

import (
	"fmt"
	"net/http"

	"github.com/MakeNowJust/heredoc"
	"github.com/cli/cli/v2/internal/ghrepo"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/spf13/cobra"
)

type GetOptions struct {
	HttpClient func() (*http.Client, error)
	BaseRepo   func() (ghrepo.Interface, error)
	IO         *iostreams.IOStreams

	Repository string
}

func NewCmdGet(f *cmdutil.Factory, runF func(*GetOptions) error) *cobra.Command {
	opts := &GetOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		BaseRepo:   f.BaseRepo,
	}

	cmd := &cobra.Command{
		Use:   "get [<repository>]",
		Args:  cobra.MaximumNArgs(1),
		Short: "Display whether subscribed to a GitHub repository for notifications",
		Long: heredoc.Docf(`
			Display whether subscribed to a GitHub repository for notifications.

			With no argument, the subscription information for the current repository is displayed.

			The possible subscription states are:
			- %[1]sall%[1]s: you are watching the repository and notified of all notifications.
			- %[1]signore%[1]s: you are watching the repository but not notified of any notifications.
			- %[1]sunwatched%[1]s: you are not watching the repository or custom notification is set.
		`, "`"),
		Example: heredoc.Doc(`
			$ gh subscription get
			$ gh subscription get monalisa/hello-world
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Repository = args[0]
			}

			if runF != nil {
				return runF(opts)
			}
			return getRun(opts)
		},
	}

	return cmd
}

func getRun(opts *GetOptions) error {
	client, err := opts.HttpClient()
	if err != nil {
		return err
	}

	var toGet ghrepo.Interface
	if opts.Repository == "" {
		toGet, err = opts.BaseRepo()
		if err != nil {
			return err
		}
	} else {
		toGet, err = ghrepo.FromFullName(opts.Repository)
		if err != nil {
			return fmt.Errorf("argument error: %w", err)
		}
	}
	repoName := ghrepo.FullName(toGet)

	subscription, err := GetSubscription(client, toGet)
	if err != nil {
		return fmt.Errorf("Error fetching subscription information for %s: %w", repoName, err)
	}
	fmt.Fprintf(opts.IO.Out, "Your subscription to %s is %s\n", repoName, subscription)

	return nil
}
