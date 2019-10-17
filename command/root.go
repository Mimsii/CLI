package command

import (
	"os"

	"github.com/github/gh-cli/context"
	"github.com/github/gh-cli/git"
	"github.com/github/gh-cli/version"
	"github.com/spf13/cobra"
)

var (
	currentRepo   string
	currentBranch string
)

func init() {
	RootCmd.PersistentFlags().StringVarP(&currentRepo, "repo", "R", "", "current GitHub repository")
	RootCmd.PersistentFlags().StringVarP(&currentBranch, "current-branch", "B", "", "current git branch")

	ctx := context.InitDefaultContext()
	ctx.SetBranch(currentBranch)
	repo := currentRepo
	if repo == "" {
		repo = os.Getenv("GH_REPO")
	}
	ctx.SetBaseRepo(repo)

	git.InitSSHAliasMap(nil)
}

// RootCmd is the entry point of command-line execution
var RootCmd = &cobra.Command{
	Use:     "gh",
	Short:   "GitHub CLI",
	Long:    `Do things with GitHub from your terminal`,
	Version: version.Version,
}
