package itemcreate

import (
	"os"
	"testing"

	"github.com/cli/cli/v2/internal/tableprinter"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"
)

func TestNewCmdCreateItem(t *testing.T) {
	tests := []struct {
		name        string
		cli         string
		wants       createItemOpts
		wantsErr    bool
		wantsErrMsg string
	}{
		{
			name:        "missing-title",
			cli:         "",
			wantsErr:    true,
			wantsErrMsg: "required flag(s) \"title\" not set",
		},

		{
			name:        "user-and-org",
			cli:         "--user monalisa --org github --title t",
			wantsErr:    true,
			wantsErrMsg: "only one of `--user` or `--org` may be used",
		},
		{
			name:        "not-a-number",
			cli:         "x --title t",
			wantsErr:    true,
			wantsErrMsg: "invalid number: x",
		},
		{
			name: "title",
			cli:  "--title t",
			wants: createItemOpts{
				title: "t",
			},
		},
		{
			name: "number",
			cli:  "123  --title t",
			wants: createItemOpts{
				number: 123,
				title:  "t",
			},
		},
		{
			name: "user",
			cli:  "--user monalisa --title t",
			wants: createItemOpts{
				userOwner: "monalisa",
				title:     "t",
			},
		},
		{
			name: "org",
			cli:  "--org github --title t",
			wants: createItemOpts{
				orgOwner: "github",
				title:    "t",
			},
		},
		{
			name: "body",
			cli:  "--body b --title t",
			wants: createItemOpts{
				body:  "b",
				title: "t",
			},
		},
		{
			name: "json",
			cli:  "--format json --title t",
			wants: createItemOpts{
				format: "json",
				title:  "t",
			},
		},
	}

	os.Setenv("GH_TOKEN", "auth-token")
	defer os.Unsetenv("GH_TOKEN")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			f := &cmdutil.Factory{
				IOStreams: ios,
			}

			argv, err := shlex.Split(tt.cli)
			assert.NoError(t, err)

			var gotOpts createItemOpts
			cmd := NewCmdCreateItem(f, func(config createItemConfig) error {
				gotOpts = config.opts
				return nil
			})

			cmd.SetArgs(argv)
			_, err = cmd.ExecuteC()
			if tt.wantsErr {
				assert.Error(t, err)
				assert.Equal(t, tt.wantsErrMsg, err.Error())
				return
			}
			assert.NoError(t, err)

			assert.Equal(t, tt.wants.number, gotOpts.number)
			assert.Equal(t, tt.wants.userOwner, gotOpts.userOwner)
			assert.Equal(t, tt.wants.orgOwner, gotOpts.orgOwner)
			assert.Equal(t, tt.wants.title, gotOpts.title)
			assert.Equal(t, tt.wants.format, gotOpts.format)
		})
	}
}

func TestRunCreateItem_Draft_User(t *testing.T) {
	defer gock.Off()
	gock.Observe(gock.DumpRequest)
	// get user ID
	gock.New("https://api.github.com").
		Post("/graphql").
		MatchType("json").
		JSON(map[string]interface{}{
			"query": "query UserLogin.*",
			"variables": map[string]interface{}{
				"login": "monalisa",
			},
		}).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"user": map[string]interface{}{
					"id": "an ID",
				},
			},
		})

	// get project ID
	gock.New("https://api.github.com").
		Post("/graphql").
		MatchType("json").
		JSON(map[string]interface{}{
			"query": "query UserProject.*",
			"variables": map[string]interface{}{
				"login":       "monalisa",
				"number":      1,
				"firstItems":  0,
				"afterItems":  nil,
				"firstFields": 0,
				"afterFields": nil,
			},
		}).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"user": map[string]interface{}{
					"projectV2": map[string]interface{}{
						"id": "an ID",
					},
				},
			},
		})

	// create item
	gock.New("https://api.github.com").
		Post("/graphql").
		BodyString(`{"query":"mutation CreateDraftItem.*","variables":{"input":{"projectId":"an ID","title":"a title","body":""}}}`).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"addProjectV2DraftIssue": map[string]interface{}{
					"projectItem": map[string]interface{}{
						"id": "item ID",
					},
				},
			},
		})

	client, err := api.NewGraphQLClient(api.ClientOptions{AuthToken: "token"})
	assert.NoError(t, err)

	ios, _, stdout, _ := iostreams.Test()
	config := createItemConfig{
		tp: tableprinter.New(ios),
		opts: createItemOpts{
			title:     "a title",
			userOwner: "monalisa",
			number:    1,
		},
		client: client,
	}

	err = runCreateItem(config)
	assert.NoError(t, err)
	assert.Equal(
		t,
		"Created item\n",
		stdout.String())
}

func TestRunCreateItem_Draft_Org(t *testing.T) {
	defer gock.Off()
	gock.Observe(gock.DumpRequest)
	// get org ID
	gock.New("https://api.github.com").
		Post("/graphql").
		MatchType("json").
		JSON(map[string]interface{}{
			"query": "query OrgLogin.*",
			"variables": map[string]interface{}{
				"login": "github",
			},
		}).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"organization": map[string]interface{}{
					"id": "an ID",
				},
			},
		})

	// get project ID
	gock.New("https://api.github.com").
		Post("/graphql").
		MatchType("json").
		JSON(map[string]interface{}{
			"query": "query OrgProject.*",
			"variables": map[string]interface{}{
				"login":       "github",
				"number":      1,
				"firstItems":  0,
				"afterItems":  nil,
				"firstFields": 0,
				"afterFields": nil,
			},
		}).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"organization": map[string]interface{}{
					"projectV2": map[string]interface{}{
						"id": "an ID",
					},
				},
			},
		})

	// create item
	gock.New("https://api.github.com").
		Post("/graphql").
		BodyString(`{"query":"mutation CreateDraftItem.*","variables":{"input":{"projectId":"an ID","title":"a title","body":""}}}`).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"addProjectV2DraftIssue": map[string]interface{}{
					"projectItem": map[string]interface{}{
						"id": "item ID",
					},
				},
			},
		})

	client, err := api.NewGraphQLClient(api.ClientOptions{AuthToken: "token"})
	assert.NoError(t, err)

	ios, _, stdout, _ := iostreams.Test()
	config := createItemConfig{
		tp: tableprinter.New(ios),
		opts: createItemOpts{
			title:    "a title",
			orgOwner: "github",
			number:   1,
		},
		client: client,
	}

	err = runCreateItem(config)
	assert.NoError(t, err)
	assert.Equal(
		t,
		"Created item\n",
		stdout.String())
}

func TestRunCreateItem_Draft_Me(t *testing.T) {
	defer gock.Off()
	gock.Observe(gock.DumpRequest)
	// get viewer ID
	gock.New("https://api.github.com").
		Post("/graphql").
		MatchType("json").
		JSON(map[string]interface{}{
			"query": "query ViewerLogin.*",
		}).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"viewer": map[string]interface{}{
					"id": "an ID",
				},
			},
		})

	// get project ID
	gock.New("https://api.github.com").
		Post("/graphql").
		MatchType("json").
		JSON(map[string]interface{}{
			"query": "query ViewerProject.*",
			"variables": map[string]interface{}{
				"number":      1,
				"firstItems":  0,
				"afterItems":  nil,
				"firstFields": 0,
				"afterFields": nil,
			},
		}).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"viewer": map[string]interface{}{
					"projectV2": map[string]interface{}{
						"id": "an ID",
					},
				},
			},
		})

	// create item
	gock.New("https://api.github.com").
		Post("/graphql").
		BodyString(`{"query":"mutation CreateDraftItem.*","variables":{"input":{"projectId":"an ID","title":"a title","body":"a body"}}}`).
		Reply(200).
		JSON(map[string]interface{}{
			"data": map[string]interface{}{
				"addProjectV2DraftIssue": map[string]interface{}{
					"projectItem": map[string]interface{}{
						"id": "item ID",
					},
				},
			},
		})

	client, err := api.NewGraphQLClient(api.ClientOptions{AuthToken: "token"})
	assert.NoError(t, err)

	ios, _, stdout, _ := iostreams.Test()
	config := createItemConfig{
		tp: tableprinter.New(ios),
		opts: createItemOpts{
			title:     "a title",
			userOwner: "@me",
			number:    1,
			body:      "a body",
		},
		client: client,
	}

	err = runCreateItem(config)
	assert.NoError(t, err)
	assert.Equal(
		t,
		"Created item\n",
		stdout.String())
}