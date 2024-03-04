package download

import (
	"bufio"
	"fmt"
	"os"
	"testing"

	"github.com/cli/cli/v2/pkg/cmd/attestation/api"
	"github.com/cli/cli/v2/pkg/cmd/attestation/artifact"
	"github.com/cli/cli/v2/pkg/cmd/attestation/artifact/oci"
	"github.com/cli/cli/v2/pkg/cmd/attestation/logger"
	"github.com/cli/cli/v2/pkg/cmd/attestation/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunDownload(t *testing.T) {
	res := test.SuppressAndRestoreOutput()
	defer res()

	tempDir, err := os.MkdirTemp("", "gh-attestation-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	baseOpts := Options{
		ArtifactPath:    "../test/data/sigstore-js-2.1.0.tgz",
		APIClient:    api.NewTestClient(),
		OCIClient:       oci.NewMockClient(),
		DigestAlgorithm: "sha512",
		OutputPath:      tempDir,
		Limit:           30,
		Logger:          logger.NewDefaultLogger(),
	}

	t.Run("fetch and store attestations successfully", func(t *testing.T) {
		err = RunDownload(&baseOpts)
		assert.NoError(t, err)

		artifact, err := artifact.NewDigestedArtifact(baseOpts.OCIClient, baseOpts.ArtifactPath, baseOpts.DigestAlgorithm)
		require.NoError(t, err)

		assert.FileExists(t, fmt.Sprintf("%s/%s.jsonl", tempDir, artifact.DigestWithAlg()))

		actualLineCount, err := countLines(fmt.Sprintf("%s/%s.jsonl", tempDir, artifact.DigestWithAlg()))
		require.NoError(t, err)

		expectedLineCount := 2
		assert.Equal(t, expectedLineCount, actualLineCount)

		err = os.Remove(fmt.Sprintf("%s/%s.jsonl", tempDir, artifact.DigestWithAlg()))
		require.NoError(t, err)
	})

	t.Run("download OCI image attestations successfully", func(t *testing.T) {
		opts := baseOpts
		opts.ArtifactPath = "oci://ghcr.io/github/test"

		err = RunDownload(&opts)
		assert.NoError(t, err)

		artifact, err := artifact.NewDigestedArtifact(opts.OCIClient, opts.ArtifactPath, opts.DigestAlgorithm)
		require.NoError(t, err)

		assert.FileExists(t, fmt.Sprintf("%s/%s.jsonl", tempDir, artifact.DigestWithAlg()))

		actualLineCount, err := countLines(fmt.Sprintf("%s/%s.jsonl", tempDir, artifact.DigestWithAlg()))
		require.NoError(t, err)

		expectedLineCount := 2
		assert.Equal(t, expectedLineCount, actualLineCount)

		err = os.Remove(fmt.Sprintf("%s/%s.jsonl", tempDir, artifact.DigestWithAlg()))
		require.NoError(t, err)
	})

	t.Run("cannot find artifact", func(t *testing.T) {
		opts := baseOpts
		opts.ArtifactPath = "../test/data/not-real.zip"

		err := RunDownload(&opts)
		assert.Error(t, err)
	})

	t.Run("no attestations found", func(t *testing.T) {
		opts := baseOpts
		opts.APIClient = api.MockClient{
			OnGetByOwnerAndDigest: func(repo, digest string, limit int) ([]*api.Attestation, error) {
				return nil, nil
			},
		}

		err := RunDownload(&opts)
		assert.NoError(t, err)

		artifact, err := artifact.NewDigestedArtifact(opts.OCIClient, opts.ArtifactPath, opts.DigestAlgorithm)
		require.NoError(t, err)
		assert.NoFileExists(t, artifact.DigestWithAlg())
	})

	t.Run("cannot download OCI artifact", func(t *testing.T) {
		opts := baseOpts
		opts.ArtifactPath = "oci://ghcr.io/github/test"
		opts.OCIClient = oci.NewReferenceFailClient()

		err := RunDownload(&opts)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "failed to digest artifact")
	})
}

func TestCreateJSONLinesFilePath(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gh-attestation-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("with output path", func(t *testing.T) {
		artifact, err := artifact.NewDigestedArtifact(oci.NewMockClient(), "../test/data/sigstore-js-2.1.0.tgz", "sha512")
		require.NoError(t, err)
		path := createJSONLinesFilePath(artifact.DigestWithAlg(), tempDir)

		expectedPath := fmt.Sprintf("%s/%s.jsonl", tempDir, artifact.DigestWithAlg())
		assert.Equal(t, expectedPath, path)
	})

	t.Run("with nested output path", func(t *testing.T) {
		artifact, err := artifact.NewDigestedArtifact(oci.NewMockClient(), "../test/data/sigstore-js-2.1.0.tgz", "sha512")
		require.NoError(t, err)

		nestedPath := fmt.Sprintf("%s/subdir", tempDir)
		path := createJSONLinesFilePath(artifact.DigestWithAlg(), nestedPath)

		expectedPath := fmt.Sprintf("%s/subdir/%s.jsonl", tempDir, artifact.DigestWithAlg())
		assert.Equal(t, expectedPath, path)
	})

	t.Run("with output path with beginning slash", func(t *testing.T) {
		artifact, err := artifact.NewDigestedArtifact(oci.NewMockClient(), "../test/data/sigstore-js-2.1.0.tgz", "sha512")
		require.NoError(t, err)

		nestedPath := fmt.Sprintf("/%s/subdir", tempDir)
		path := createJSONLinesFilePath(artifact.DigestWithAlg(), nestedPath)

		expectedPath := fmt.Sprintf("/%s/subdir/%s.jsonl", tempDir, artifact.DigestWithAlg())
		assert.Equal(t, expectedPath, path)
	})

	t.Run("without output path", func(t *testing.T) {
		artifact, err := artifact.NewDigestedArtifact(oci.NewMockClient(), "../test/data/sigstore-js-2.1.0.tgz", "sha512")
		require.NoError(t, err)
		path := createJSONLinesFilePath(artifact.DigestWithAlg(), "")

		expectedPath := fmt.Sprintf("%s.jsonl", artifact.DigestWithAlg())
		assert.Equal(t, expectedPath, path)
	})
}

func countLines(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	counter := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		counter += 1
	}

	return counter, nil
}