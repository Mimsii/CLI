package create

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/cli/cli/v2/internal/config"
	"github.com/cli/cli/v2/internal/run"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/httpmock"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/cli/cli/v2/pkg/prompt"
	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCmdCreate(t *testing.T) {
	tests := []struct {
		name      string
		tty       bool
		cli       string
		wantsErr  bool
		errMsg    string
		wantsOpts CreateOptions
	}{
		{
			name:      "no args tty",
			tty:       true,
			cli:       "",
			wantsOpts: CreateOptions{Interactive: true},
		},
		{
			name:     "no args no-tty",
			tty:      false,
			cli:      "",
			wantsErr: true,
			errMsg:   "at least one argument required in non-interactive mode",
		},
		{
			name: "new repo from remote",
			cli:  "NEWREPO --public --clone",
			wantsOpts: CreateOptions{
				Name:   "NEWREPO",
				Public: true,
				Clone:  true},
		},
		{
			name:     "no visibility",
			tty:      true,
			cli:      "NEWREPO",
			wantsErr: true,
			errMsg:   "`--public`, `--private`, or `--internal` required when not running interactively",
		},
		{
			name:     "multiple visibility",
			tty:      true,
			cli:      "NEWREPO --public --private",
			wantsErr: true,
			errMsg:   "expected exactly one of `--public`, `--private`, or `--internal`",
		},
		{
			name: "new remote from local",
			cli:  "--source=/path/to/repo --private",
			wantsOpts: CreateOptions{
				Private: true,
				Source:  "/path/to/repo"},
		},
		{
			name: "new remote from local with remote",
			cli:  "--source=/path/to/repo --public --remote upstream",
			wantsOpts: CreateOptions{
				Public: true,
				Source: "/path/to/repo",
				Remote: "upstream",
			},
		},
		{
			name: "new remote from local with push",
			cli:  "--source=/path/to/repo --push --public",
			wantsOpts: CreateOptions{
				Public: true,
				Source: "/path/to/repo",
				Push:   true,
			},
		},
		{
			name: "new remote from local without visibility",
			cli:  "--source=/path/to/repo --push",
			wantsOpts: CreateOptions{
				Source: "/path/to/repo",
				Push:   true,
			},
			wantsErr: true,
			errMsg:   "`--public`, `--private`, or `--internal` required when not running interactively",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			io, _, _, _ := iostreams.Test()
			io.SetStdinTTY(tt.tty)
			io.SetStdoutTTY(tt.tty)

			f := &cmdutil.Factory{
				IOStreams: io,
			}

			var opts *CreateOptions
			cmd := NewCmdCreate(f, func(o *CreateOptions) error {
				opts = o
				return nil
			})

			// TODO STUPID HACK
			// cobra aggressively adds help to all commands. since we're not running through the root command
			// (which manages help when running for real) and since create has a '-h' flag (for homepage),
			// cobra blows up when it tried to add a help flag and -h is already in use. This hack adds a
			// dummy help flag with a random shorthand to get around this.
			cmd.Flags().BoolP("help", "x", false, "")

			args, err := shlex.Split(tt.cli)
			require.NoError(t, err)
			cmd.SetArgs(args)
			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})

			_, err = cmd.ExecuteC()
			if tt.wantsErr {
				assert.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
				return
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.wantsOpts.Interactive, opts.Interactive)
			assert.Equal(t, tt.wantsOpts.Source, opts.Source)
			assert.Equal(t, tt.wantsOpts.Name, opts.Name)
			assert.Equal(t, tt.wantsOpts.Public, opts.Public)
			assert.Equal(t, tt.wantsOpts.Internal, opts.Internal)
			assert.Equal(t, tt.wantsOpts.Private, opts.Private)
			assert.Equal(t, tt.wantsOpts.Clone, opts.Clone)
		})
	}
}

func Test_createRun(t *testing.T) {
	tests := []struct {
		name       string
		tty        bool
		opts       *CreateOptions
		httpStubs  func(*httpmock.Registry)
		askStubs   func(*prompt.AskStubber)
		execStubs  func(*run.CommandStubber)
		wantStdout string
		wantErr    bool
		errMsg     string
	}{
		{
			name: "interactive create from scratch with gitignore and license",
			opts: &CreateOptions{Interactive: true},
			tty:  true,
			wantStdout: `✓ Created repository OWNER/REPO on GitHub
✓ Initialized repository in "REPO"
`,
			askStubs: func(as *prompt.AskStubber) {
				as.StubOne("Create a new repository on GitHub from scratch")
				as.Stub([]*prompt.QuestionStub{
					{Name: "repoName", Value: "REPO"},
					{Name: "repoDescription", Value: "my new repo"},
					{Name: "repoVisibility", Value: "PRIVATE"},
				})
				as.Stub([]*prompt.QuestionStub{
					{Name: "addGitIgnore", Value: true}})
				as.Stub([]*prompt.QuestionStub{
					{Name: "chooseGitIgnore", Value: "Go"}})
				as.Stub([]*prompt.QuestionStub{
					{Name: "addLicense", Value: true}})
				as.Stub([]*prompt.QuestionStub{
					{Name: "chooseLicense", Value: "GNU Lesser General Public License v3.0"}})
				as.Stub([]*prompt.QuestionStub{
					{Name: "confirmSubmit", Value: true}})
				as.StubOne(true) //clone locally?
			},
			httpStubs: func(reg *httpmock.Registry) {
				reg.Register(
					httpmock.REST("GET", "gitignore/templates"),
					httpmock.StringResponse(`["Actionscript","Android","AppceleratorTitanium","Autotools","Bancha","C","C++","Go"]`))
				reg.Register(
					httpmock.REST("GET", "licenses"),
					httpmock.StringResponse(`[{"key": "mit","name": "MIT License"},{"key": "lgpl-3.0","name": "GNU Lesser General Public License v3.0"}]`))
				reg.Register(
					httpmock.REST("POST", "user/repos"),
					httpmock.StringResponse(`{"name":"REPO", "owner":{"login": "OWNER"}, "html_url":"https://github.com/OWNER/REPO"}`))

			},
			execStubs: func(cs *run.CommandStubber) {
				cs.Register(`git clone https://github.com/OWNER/REPO.git`, 0, "")
			},
		},
		{
			name: "interactive with existing repository public",
			opts: &CreateOptions{Interactive: true},
			tty:  true,
			askStubs: func(as *prompt.AskStubber) {
				as.StubOne("Push an existing local repository to GitHub")
				as.StubOne(".")
				as.Stub([]*prompt.QuestionStub{
					{Name: "repoName", Value: "REPO"},
					{Name: "repoDescription", Value: "my new repo"},
					{Name: "repoVisibility", Value: "PRIVATE"},
				})
				as.StubOne(false)
			},
			httpStubs: func(reg *httpmock.Registry) {
				reg.Register(
					httpmock.GraphQL(`mutation RepositoryCreate\b`),
					httpmock.StringResponse(`
					{
						"data": {
							"createRepository": {
								"repository": {
									"id": "REPOID",
									"name": "REPO",
									"owner": {"login":"OWNER"},
									"url": "https://github.com/OWNER/REPO"
								}
							}
						}
					}`))
			},
			execStubs: func(cs *run.CommandStubber) {
				cs.Register(`git -C . rev-parse --git-dir`, 0, ".git")
			},
			wantStdout: "✓ Created repository OWNER/REPO on GitHub\n",
		},
		{
			name: "interactive with existing repository public add remote",
			opts: &CreateOptions{Interactive: true},
			tty:  true,
			askStubs: func(as *prompt.AskStubber) {
				as.StubOne("Push an existing local repository to GitHub")
				as.StubOne(".")
				as.Stub([]*prompt.QuestionStub{
					{Name: "repoName", Value: "REPO"},
					{Name: "repoDescription", Value: "my new repo"},
					{Name: "repoVisibility", Value: "PRIVATE"},
				})
				as.StubOne(true)     //ask for adding a remote
				as.StubOne("origin") //ask for remote name
				as.StubOne(false)    //ask to push to remote
			},
			httpStubs: func(reg *httpmock.Registry) {
				reg.Register(
					httpmock.GraphQL(`mutation RepositoryCreate\b`),
					httpmock.StringResponse(`
					{
						"data": {
							"createRepository": {
								"repository": {
									"id": "REPOID",
									"name": "REPO",
									"owner": {"login":"OWNER"},
									"url": "https://github.com/OWNER/REPO"
								}
							}
						}
					}`))
			},
			execStubs: func(cs *run.CommandStubber) {
				cs.Register(`git -C . rev-parse --git-dir`, 0, ".git")
				cs.Register(`git symbolic-ref --quiet HEAD`, 0, "HEAD")
				cs.Register(`git -C . remote add origin https://github.com/OWNER/REPO`, 0, "")
			},
			wantStdout: "✓ Created repository OWNER/REPO on GitHub\n✓ Added remote https://github.com/OWNER/REPO.git\n✓ Added branch to https://github.com/OWNER/REPO.git\n",
		},
		{
			name: "interactive with existing repository public add remote",
			opts: &CreateOptions{Interactive: true},
			tty:  true,
			askStubs: func(as *prompt.AskStubber) {
				as.StubOne("Push an existing local repository to GitHub")
				as.StubOne(".")
				as.Stub([]*prompt.QuestionStub{
					{Name: "repoName", Value: "REPO"},
					{Name: "repoDescription", Value: "my new repo"},
					{Name: "repoVisibility", Value: "PRIVATE"},
				})
				as.StubOne(true)     //ask for adding a remote
				as.StubOne("origin") //ask for remote name
				as.StubOne(true)     //ask to push to remote
			},
			httpStubs: func(reg *httpmock.Registry) {
				reg.Register(
					httpmock.GraphQL(`mutation RepositoryCreate\b`),
					httpmock.StringResponse(`
					{
						"data": {
							"createRepository": {
								"repository": {
									"id": "REPOID",
									"name": "REPO",
									"owner": {"login":"OWNER"},
									"url": "https://github.com/OWNER/REPO"
								}
							}
						}
					}`))
			},
			execStubs: func(cs *run.CommandStubber) {
				cs.Register(`git -C . rev-parse --git-dir`, 0, ".git")
				cs.Register(`git symbolic-ref --quiet HEAD`, 0, "HEAD")
				cs.Register(`git -C . remote add origin https://github.com/OWNER/REPO`, 0, "")
				cs.Register(`git -C . push -u origin HEAD`, 0, "")
			},
			wantStdout: "✓ Created repository OWNER/REPO on GitHub\n✓ Added remote https://github.com/OWNER/REPO.git\n✓ Added branch to https://github.com/OWNER/REPO.git\n✓ Pushed commits to https://github.com/OWNER/REPO.git\n",
		},
	}
	for _, tt := range tests {
		q, teardown := prompt.InitAskStubber()
		defer teardown()
		if tt.askStubs != nil {
			tt.askStubs(q)
		}

		reg := &httpmock.Registry{}
		if tt.httpStubs != nil {
			tt.httpStubs(reg)
		}
		tt.opts.HttpClient = func() (*http.Client, error) {
			return &http.Client{Transport: reg}, nil
		}
		tt.opts.Config = func() (config.Config, error) {
			return config.NewBlankConfig(), nil
		}

		cs, restoreRun := run.Stub()
		defer restoreRun(t)
		if tt.execStubs != nil {
			tt.execStubs(cs)
		}

		io, _, stdout, _ := iostreams.Test()
		io.SetStdinTTY(tt.tty)
		io.SetStdoutTTY(tt.tty)
		tt.opts.IO = io

		t.Run(tt.name, func(t *testing.T) {
			defer reg.Verify(t)
			err := createRun(tt.opts)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantStdout, stdout.String())
		})
	}
}
