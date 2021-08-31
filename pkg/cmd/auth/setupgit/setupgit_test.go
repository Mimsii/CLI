package setupgit

import (
	"bytes"
	"fmt"
	"regexp"
	"testing"

	"github.com/cli/cli/internal/config"
	"github.com/cli/cli/pkg/cmdutil"
	"github.com/cli/cli/pkg/iostreams"
	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockGitConfigurer struct {
	setupErr error
}

func (gf *mockGitConfigurer) Setup(hostname, username, authToken string) error {
	return gf.setupErr
}

func Test_NewCmdSetupGit(t *testing.T) {
	tests := []struct {
		name     string
		cli      string
		wants    SetupGitOptions
		wantsErr bool
	}{
		{
			name:  "no arguments",
			cli:   "",
			wants: SetupGitOptions{Hostname: ""},
		},
		{
			name:  "hostname argument",
			cli:   "--hostname whatever",
			wants: SetupGitOptions{Hostname: "whatever"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &cmdutil.Factory{}

			argv, err := shlex.Split(tt.cli)
			assert.NoError(t, err)

			var gotOpts *SetupGitOptions
			cmd := NewCmdSetupGit(f, func(opts *SetupGitOptions) error {
				gotOpts = opts
				return nil
			})

			// TODO cobra hack-around
			cmd.Flags().BoolP("help", "x", false, "")

			cmd.SetArgs(argv)
			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})

			_, err = cmd.ExecuteC()
			assert.NoError(t, err)

			assert.Equal(t, tt.wants.Hostname, gotOpts.Hostname)
		})
	}
}

func Test_setupGitRun(t *testing.T) {
	tests := []struct {
		name           string
		opts           *SetupGitOptions
		expectedErr    string
		expectedErrOut *regexp.Regexp
	}{
		{
			name: "opts.Config returns an error",
			opts: &SetupGitOptions{
				Config: func() (config.Config, error) {
					return nil, fmt.Errorf("oops")
				},
			},
			expectedErr: "oops",
		},
		{
			name:           "no authenticated hostnames",
			opts:           &SetupGitOptions{},
			expectedErr:    "SilentError",
			expectedErrOut: regexp.MustCompile("You are not logged into any GitHub hosts."),
		},
		{
			name: "not authenticated with the hostname given as flag",
			opts: &SetupGitOptions{
				Hostname: "foo",
				Config: func() (config.Config, error) {
					cfg := config.NewBlankConfig()
					require.NoError(t, cfg.Set("bar", "", ""))
					return cfg, nil
				},
			},
			expectedErr:    "SilentError",
			expectedErrOut: regexp.MustCompile("You are not logged into any Github host with the hostname foo"),
		},
		{
			name: "error setting up git for hostname",
			opts: &SetupGitOptions{
				gitConfigure: &mockGitConfigurer{
					setupErr: fmt.Errorf("broken"),
				},
				Config: func() (config.Config, error) {
					cfg := config.NewBlankConfig()
					require.NoError(t, cfg.Set("bar", "", ""))
					return cfg, nil
				},
			},
			expectedErr:    "SilentError",
			expectedErrOut: regexp.MustCompile("failed to setup git credential helper"),
		},
		{
			name: "no hostname option given. Setup git for each hostname in config",
			opts: &SetupGitOptions{
				gitConfigure: &mockGitConfigurer{},
				Config: func() (config.Config, error) {
					cfg := config.NewBlankConfig()
					require.NoError(t, cfg.Set("bar", "", ""))
					return cfg, nil
				},
			},
		},
		{
			name: "setup git for the hostname given via options",
			opts: &SetupGitOptions{
				Hostname:     "yes",
				gitConfigure: &mockGitConfigurer{},
				Config: func() (config.Config, error) {
					cfg := config.NewBlankConfig()
					require.NoError(t, cfg.Set("bar", "", ""))
					require.NoError(t, cfg.Set("yes", "", ""))
					return cfg, nil
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.opts.Config == nil {
				tt.opts.Config = func() (config.Config, error) {
					return config.NewBlankConfig(), nil
				}
			}

			io, _, _, stderr := iostreams.Test()

			io.SetStdinTTY(true)
			io.SetStderrTTY(true)
			io.SetStdoutTTY(true)
			tt.opts.IO = io

			err := setupGitRun(tt.opts)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}

			if tt.expectedErrOut == nil {
				assert.Equal(t, "", stderr.String())
			} else {
				assert.True(t, tt.expectedErrOut.MatchString(stderr.String()))
			}
		})
	}
}
