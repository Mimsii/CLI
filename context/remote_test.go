package context

import (
	"errors"
	"net/url"
	"testing"

	"github.com/cli/cli/api"
	"github.com/cli/cli/git"
	"github.com/cli/cli/internal"
	"github.com/cli/cli/internal/ghrepo"
)

func Test_Remotes_FindByName(t *testing.T) {
	list := Remotes{
		&Remote{Remote: &git.Remote{Name: "mona"}, Owner: "monalisa", Repo: "myfork"},
		&Remote{Remote: &git.Remote{Name: "origin"}, Owner: "monalisa", Repo: "octo-cat"},
		&Remote{Remote: &git.Remote{Name: "upstream"}, Owner: "hubot", Repo: "tools"},
	}

	r, err := list.FindByName("upstream", "origin")
	eq(t, err, nil)
	eq(t, r.Name, "upstream")

	r, err = list.FindByName("nonexist", "*")
	eq(t, err, nil)
	eq(t, r.Name, "mona")

	_, err = list.FindByName("nonexist")
	eq(t, err, errors.New(`no GitHub remotes found`))
}

func Test_translateRemotes(t *testing.T) {
	publicURL, _ := url.Parse("https://" + internal.Host + "/monalisa/hello")
	originURL, _ := url.Parse("http://example.com/repo")

	gitRemotes := git.RemoteSet{
		&git.Remote{
			Name:     "origin",
			FetchURL: originURL,
		},
		&git.Remote{
			Name:     "public",
			FetchURL: publicURL,
		},
	}

	identityURL := func(u *url.URL) *url.URL {
		return u
	}
	result := translateRemotes(gitRemotes, identityURL)

	if len(result) != 1 {
		t.Errorf("got %d results", len(result))
	}
	if result[0].Name != "public" {
		t.Errorf("got %q", result[0].Name)
	}
	if result[0].RepoName() != "hello" {
		t.Errorf("got %q", result[0].RepoName())
	}
}

func Test_resolvedRemotes_triangularSetup(t *testing.T) {
	resolved := ResolvedRemotes{
		BaseOverride: nil,
		Remotes: Remotes{
			&Remote{
				Remote: &git.Remote{Name: "origin"},
				Owner:  "OWNER",
				Repo:   "REPO",
			},
			&Remote{
				Remote: &git.Remote{Name: "fork"},
				Owner:  "MYSELF",
				Repo:   "REPO",
			},
		},
		Network: api.RepoNetworkResult{
			Repositories: []*api.Repository{
				&api.Repository{
					Name:             "NEWNAME",
					Owner:            api.RepositoryOwner{Login: "NEWOWNER"},
					ViewerPermission: "READ",
				},
				&api.Repository{
					Name:             "REPO",
					Owner:            api.RepositoryOwner{Login: "MYSELF"},
					ViewerPermission: "ADMIN",
				},
			},
		},
	}

	baseRepo, err := resolved.BaseRepo()
	if err != nil {
		t.Fatalf("got %v", err)
	}
	eq(t, ghrepo.FullName(baseRepo), "NEWOWNER/NEWNAME")
	baseRemote, err := resolved.RemoteForRepo(baseRepo)
	if err != nil {
		t.Fatalf("got %v", err)
	}
	if baseRemote.Name != "origin" {
		t.Errorf("got remote %q", baseRemote.Name)
	}

	headRepo, err := resolved.HeadRepo()
	if err != nil {
		t.Fatalf("got %v", err)
	}
	eq(t, ghrepo.FullName(headRepo), "MYSELF/REPO")
	headRemote, err := resolved.RemoteForRepo(headRepo)
	if err != nil {
		t.Fatalf("got %v", err)
	}
	if headRemote.Name != "fork" {
		t.Errorf("got remote %q", headRemote.Name)
	}
}

func Test_resolvedRemotes_clonedFork(t *testing.T) {
	resolved := ResolvedRemotes{
		BaseOverride: nil,
		Remotes: Remotes{
			&Remote{
				Remote: &git.Remote{Name: "origin"},
				Owner:  "OWNER",
				Repo:   "REPO",
			},
		},
		Network: api.RepoNetworkResult{
			Repositories: []*api.Repository{
				&api.Repository{
					Name:             "REPO",
					Owner:            api.RepositoryOwner{Login: "OWNER"},
					ViewerPermission: "ADMIN",
					Parent: &api.Repository{
						Name:             "REPO",
						Owner:            api.RepositoryOwner{Login: "PARENTOWNER"},
						ViewerPermission: "READ",
					},
				},
			},
		},
	}

	baseRepo, err := resolved.BaseRepo()
	if err != nil {
		t.Fatalf("got %v", err)
	}
	eq(t, ghrepo.FullName(baseRepo), "PARENTOWNER/REPO")
	baseRemote, err := resolved.RemoteForRepo(baseRepo)
	if baseRemote != nil || err == nil {
		t.Error("did not expect any remote for base")
	}

	headRepo, err := resolved.HeadRepo()
	if err != nil {
		t.Fatalf("got %v", err)
	}
	eq(t, ghrepo.FullName(headRepo), "OWNER/REPO")
	headRemote, err := resolved.RemoteForRepo(headRepo)
	if err != nil {
		t.Fatalf("got %v", err)
	}
	if headRemote.Name != "origin" {
		t.Errorf("got remote %q", headRemote.Name)
	}
}
