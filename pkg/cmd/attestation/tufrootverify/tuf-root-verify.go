package tufrootverify

import (
	"fmt"
	"os"

	"github.com/cli/cli/v2/pkg/cmd/attestation/auth"
	"github.com/cli/cli/v2/pkg/cmd/attestation/logging"
	"github.com/cli/cli/v2/pkg/cmd/attestation/verification"
	"github.com/cli/cli/v2/pkg/cmdutil"

	"github.com/MakeNowJust/heredoc"
	"github.com/sigstore/sigstore-go/pkg/tuf"
	"github.com/spf13/cobra"
)

func NewTUFRootVerifyCmd(f *cmdutil.Factory) *cobra.Command {
	var mirror string
	var root string
	var cmd = cobra.Command{
		Use:   "tuf-root-verify --mirror <mirror-url> --root <root.json>",
		Args:  cobra.ExactArgs(0),
		Short: "Verify the TUF repository from a provided TUF root",
		Long: heredoc.Docf(`
			Verify a TUF repository with a local TUF root.

			The command requires you provide the %[1]s--mirror%[1]s flag, which should be the URL 
			of the TUF repository mirror.
			
			The command also requires you provide the %[1]s--root%[1]s flag, which should be the 
			path to the TUF root file.

			GitHub relies on TUF to securely deliver the trust root for our signing authority.
			For more information on TUF, see the official documentation: <https://theupdateframework.github.io/>.
		`, "`"),
		Example: heredoc.Doc(`
			# Verify the TUF repository from a provided TUF root
			gh attestation tuf-root-verify --mirror https://tuf-repo.github.com --root /path/to/1.root.json
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := logging.NewDefaultLogger(f.IOStreams)

			if err := auth.IsHostSupported(); err != nil {
				return err
			}

			if err := verifyTUFRoot(mirror, root); err != nil {
				return fmt.Errorf("Failed to verify the TUF repository: %w", err)
			}
			fmt.Sprintln(logger.IO.Out, logger.ColorScheme.Green("Successfully verified the TUF repository"))
			return nil
		},
	}

	cmd.Flags().StringVarP(&mirror, "mirror", "m", "", "URL to the TUF repository mirror")
	cmd.MarkFlagRequired("mirror") //nolint:errcheck
	cmd.Flags().StringVarP(&root, "root", "r", "", "Path to the TUF root file on disk")
	cmd.MarkFlagRequired("root") //nolint:errcheck

	return &cmd
}

func verifyTUFRoot(mirror, root string) error {
	rb, err := os.ReadFile(root)
	if err != nil {
		return fmt.Errorf("failed to read root file %s: %w", root, err)
	}
	opts, err := verification.GitHubTUFOptions()
	if err != nil {
		return err
	}
	opts.Root = rb
	opts.RepositoryBaseURL = mirror
	// The purpose is the verify the TUF root and repository, make
	// sure there is no caching enabled
	opts.CacheValidity = 0
	if _, err = tuf.New(opts); err != nil {
		return fmt.Errorf("failed to create TUF client: %w", err)
	}

	return nil
}
