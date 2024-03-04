package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testRepo   = "github/example"
	testOwner  = "github"
	testDigest = "sha256:12313213"
)

func NewClientWithMockGHClient() Client {
	fetcher := mockDataGenerator{
		NumAttestations: 5,
	}

	return &LiveClient{
		api: mockAPIClient{
			OnREST: fetcher.OnRESTSuccess,
			OnRESTWithNext: fetcher.OnRESTSuccessWithNextPage,
		},
	}
}

func NewClientWithMockGHClientWithNextPage() Client {
	fetcher := mockDataGenerator{
		NumAttestations: 5,
	}

	return &LiveClient{
		api: mockAPIClient{
			OnREST: fetcher.OnRESTSuccess,
			OnRESTWithNext: fetcher.OnRESTSuccessWithNextPage,
		},
	}
}

func TestGetURL(t *testing.T) {
	c := LiveClient{}

	testData := []struct {
		repo     string
		digest   string
		expected string
	}{
		{repo: "/github/example/", digest: "sha256:12313213", expected: "repos/github/example/attestations/sha256:12313213"},
		{repo: "/github/example", digest: "sha256:12313213", expected: "repos/github/example/attestations/sha256:12313213"},
	}

	for _, data := range testData {
		s := c.BuildRepoAndDigestURL(data.repo, data.digest)
		assert.Equal(t, data.expected, s)
	}
}

func TestGetByDigest(t *testing.T) {
	c := NewClientWithMockGHClient()
	attestations, err := c.GetByRepoAndDigest(testRepo, testDigest, DefaultLimit)
	require.NoError(t, err)

	assert.Equal(t, len(attestations), 5)
	bundle := (attestations)[0].Bundle
	assert.Equal(t, bundle.GetMediaType(), "application/vnd.dev.sigstore.bundle+json;version=0.1")

	attestations, err = c.GetByOwnerAndDigest(testOwner, testDigest, DefaultLimit)
	require.NoError(t, err)

	assert.Equal(t, len(attestations), 5)
	bundle = (attestations)[0].Bundle
	assert.Equal(t, bundle.GetMediaType(), "application/vnd.dev.sigstore.bundle+json;version=0.1")
}

func TestGetByDigestGreaterThanLimit(t *testing.T) {
	c := NewClientWithMockGHClient()

	limit := 3
	// The method should return five results when the limit is not set
	attestations, err := c.GetByRepoAndDigest(testRepo, testDigest, limit)
	require.NoError(t, err)

	assert.Equal(t, len(attestations), 3)
	bundle := (attestations)[0].Bundle
	assert.Equal(t, bundle.GetMediaType(), "application/vnd.dev.sigstore.bundle+json;version=0.1")

	attestations, err = c.GetByOwnerAndDigest(testOwner, testDigest, limit)
	require.NoError(t, err)

	assert.Equal(t, len(attestations), limit)
	bundle = (attestations)[0].Bundle
	assert.Equal(t, bundle.GetMediaType(), "application/vnd.dev.sigstore.bundle+json;version=0.1")
}

func TestGetByDigestWithNextPage(t *testing.T) {
	c := NewClientWithMockGHClientWithNextPage()
	attestations, err := c.GetByRepoAndDigest(testRepo, testDigest, DefaultLimit)
	require.NoError(t, err)

	assert.Equal(t, len(attestations), 10)
	bundle := (attestations)[0].Bundle
	assert.Equal(t, bundle.GetMediaType(), "application/vnd.dev.sigstore.bundle+json;version=0.1")

	attestations, err = c.GetByOwnerAndDigest(testOwner, testDigest, DefaultLimit)
	require.NoError(t, err)

	assert.Equal(t, len(attestations), 10)
	bundle = (attestations)[0].Bundle
	assert.Equal(t, bundle.GetMediaType(), "application/vnd.dev.sigstore.bundle+json;version=0.1")
}

func TestGetByDigestGreaterThanLimitWithNextPage(t *testing.T) {
	c := NewClientWithMockGHClientWithNextPage()

	limit := 7
	// The method should return five results when the limit is not set
	attestations, err := c.GetByRepoAndDigest(testRepo, testDigest, limit)
	require.NoError(t, err)

	assert.Equal(t, len(attestations), limit)
	bundle := (attestations)[0].Bundle
	assert.Equal(t, bundle.GetMediaType(), "application/vnd.dev.sigstore.bundle+json;version=0.1")

	attestations, err = c.GetByOwnerAndDigest(testOwner, testDigest, limit)
	require.NoError(t, err)

	assert.Equal(t, len(attestations), limit)
	bundle = (attestations)[0].Bundle
	assert.Equal(t, bundle.GetMediaType(), "application/vnd.dev.sigstore.bundle+json;version=0.1")
}

func TestGetByDigest_NoAttestationsFound(t *testing.T) {
	fetcher := mockDataGenerator{
		NumAttestations: 5,
	}

	c := LiveClient{
		api: mockAPIClient{
			OnREST: fetcher.OnRESTNoAttestations,
			OnRESTWithNext: fetcher.OnRESTSuccessWithNextPage,
		},
	}

	attestations, err := c.GetByRepoAndDigest(testRepo, testDigest, DefaultLimit)
	assert.Error(t, err)
	assert.IsType(t, ErrNoAttestations{}, err)
	assert.Nil(t, attestations)

	attestations, err = c.GetByOwnerAndDigest(testOwner, testDigest, DefaultLimit)
	assert.Error(t, err)
	assert.IsType(t, ErrNoAttestations{}, err)
	assert.Nil(t, attestations)
}

func TestGetByDigest_Error(t *testing.T) {
	fetcher := mockDataGenerator{
		NumAttestations: 5,
	}

	c := LiveClient{
		api: mockAPIClient{
			OnREST: fetcher.OnRESTError,
			OnRESTWithNext: fetcher.OnRESTSuccessWithNextPage,
		},
	}

	attestations, err := c.GetByRepoAndDigest(testRepo, testDigest, DefaultLimit)
	assert.Error(t, err)
	assert.Nil(t, attestations)

	attestations, err = c.GetByOwnerAndDigest(testOwner, testDigest, DefaultLimit)
	assert.Error(t, err)
	assert.Nil(t, attestations)
}