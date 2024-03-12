package verify

import (
	"testing"

	"github.com/cli/cli/v2/pkg/cmd/attestation/api"
	"github.com/cli/cli/v2/pkg/cmd/attestation/artifact/oci"
	"github.com/cli/cli/v2/pkg/cmd/attestation/logging"
	"github.com/cli/cli/v2/pkg/cmd/attestation/test"
	"github.com/cli/cli/v2/pkg/cmd/attestation/verification"

	"github.com/in-toto/in-toto-golang/in_toto"
	"github.com/sigstore/sigstore-go/pkg/verify"

	"github.com/stretchr/testify/require"
)

const (
	SigstoreSanValue = "https://github.com/sigstore/sigstore-js/.github/workflows/release.yml@refs/heads/main"
	SigstoreSanRegex = "^https://github.com/sigstore/sigstore-js/"
)

func TestRunVerify(t *testing.T) {
	logger := logging.NewTestLogger()

	publicGoodOpts := Options{
		ArtifactPath:    test.NormalizeRelativePath("../test/data/sigstore-js-2.1.0.tgz"),
		BundlePath:      test.NormalizeRelativePath("../test/data/sigstore-js-2.1.0-bundle.json"),
		DigestAlgorithm: "sha512",
		APIClient:       api.NewTestClient(),
		Logger:          logger,
		OCIClient:       oci.MockClient{},
		OIDCIssuer:      GitHubOIDCIssuer,
		Owner:           "sigstore",
		SANRegex:        "^https://github.com/sigstore/",
	}

	t.Run("with valid artifact and bundle", func(t *testing.T) {
		require.Nil(t, RunVerify(&publicGoodOpts))
	})

	t.Run("with failing OCI artifact fetch", func(t *testing.T) {
		opts := publicGoodOpts
		opts.ArtifactPath = "oci://ghcr.io/github/test"
		opts.OCIClient = oci.ReferenceFailClient{}

		err := RunVerify(&opts)
		require.Error(t, err)
		require.ErrorContains(t, err, "failed to digest artifact")
	})

	t.Run("with missing artifact path", func(t *testing.T) {
		opts := publicGoodOpts
		opts.ArtifactPath = "../test/data/non-existent-artifact.zip"
		require.Error(t, RunVerify(&opts))
	})

	t.Run("with missing bundle path", func(t *testing.T) {
		opts := publicGoodOpts
		opts.BundlePath = "../test/data/non-existent-sigstoreBundle.json"
		require.Error(t, RunVerify(&opts))
	})

	t.Run("with invalid signature", func(t *testing.T) {
		opts := publicGoodOpts
		opts.BundlePath = "../test/data/sigstoreBundle-invalid-signature.json"

		err := RunVerify(&opts)
		require.Error(t, err)
		require.ErrorContains(t, err, "at least one attestation failed to verify")
		require.ErrorContains(t, err, "verifying with issuer \"sigstore.dev\"")
	})

	t.Run("with owner", func(t *testing.T) {
		opts := publicGoodOpts
		opts.BundlePath = ""
		opts.Owner = "sigstore"

		require.Nil(t, RunVerify(&opts))
	})

	t.Run("with repo", func(t *testing.T) {
		opts := publicGoodOpts
		opts.BundlePath = ""
		opts.Repo = "github/example"

		require.Nil(t, RunVerify(&opts))
	})

	t.Run("with invalid repo", func(t *testing.T) {
		opts := publicGoodOpts
		opts.BundlePath = ""
		opts.Repo = "wrong/example"
		opts.APIClient = api.NewFailTestClient()

		err := RunVerify(&opts)
		require.Error(t, err)
		require.ErrorContains(t, err, "failed to fetch attestations for subject")
	})

	t.Run("with invalid owner", func(t *testing.T) {
		opts := publicGoodOpts
		opts.BundlePath = ""
		opts.APIClient = api.NewFailTestClient()
		opts.Owner = "wrong-owner"

		err := RunVerify(&opts)
		require.Error(t, err)
		require.ErrorContains(t, err, "failed to fetch attestations for subject")
	})

	t.Run("with invalid OIDC issuer", func(t *testing.T) {
		opts := publicGoodOpts
		opts.OIDCIssuer = "not-a-real-issuer"
		require.Error(t, RunVerify(&opts))
	})

	t.Run("with SAN enforcement", func(t *testing.T) {
		opts := Options{
			ArtifactPath:    "../test/data/sigstore-js-2.1.0.tgz",
			BundlePath:      "../test/data/sigstore-js-2.1.0-bundle.json",
			APIClient:       api.NewTestClient(),
			DigestAlgorithm: "sha512",
			Logger:          logger,
			OIDCIssuer:      GitHubOIDCIssuer,
			Owner:           "sigstore",
			SAN:             SigstoreSanValue,
		}
		require.Nil(t, RunVerify(&opts))
	})

	t.Run("with invalid SAN", func(t *testing.T) {
		opts := publicGoodOpts
		opts.SAN = "fake san"
		require.Error(t, RunVerify(&opts))
	})

	t.Run("with SAN regex enforcement", func(t *testing.T) {
		opts := publicGoodOpts
		opts.SANRegex = SigstoreSanRegex
		require.Nil(t, RunVerify(&opts))
	})

	t.Run("with invalid SAN regex", func(t *testing.T) {
		opts := publicGoodOpts
		opts.SANRegex = "^https://github.com/sigstore/not-real/"
		require.Error(t, RunVerify(&opts))
	})

	t.Run("with no matching OIDC issuer", func(t *testing.T) {
		opts := publicGoodOpts
		opts.OIDCIssuer = "some-other-issuer"

		require.Error(t, RunVerify(&opts))
	})

	t.Run("with valid artifact and JSON lines file containing multiple Sigstore bundles", func(t *testing.T) {
		opts := publicGoodOpts
		opts.BundlePath = "../test/data/sigstore-js-2.1.0_with_2_bundles.jsonl"
		require.Nil(t, RunVerify(&opts))
	})

	t.Run("with missing OCI client", func(t *testing.T) {
		customOpts := publicGoodOpts
		customOpts.ArtifactPath = "oci://ghcr.io/github/test"
		customOpts.OCIClient = nil
		require.Error(t, RunVerify(&customOpts))
	})

	t.Run("with missing API client", func(t *testing.T) {
		customOpts := publicGoodOpts
		customOpts.APIClient = nil
		customOpts.BundlePath = ""
		require.Error(t, RunVerify(&customOpts))
	})
}

func TestVerifySLSAPredicateType_InvalidPredicate(t *testing.T) {
	statement := &in_toto.Statement{}
	statement.PredicateType = "some-other-predicate-type"

	apr := []*verification.AttestationProcessingResult{
		{
			VerificationResult: &verify.VerificationResult{
				Statement: statement,
			},
		},
	}

	err := verifySLSAPredicateType(logging.NewTestLogger(), apr)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNoMatchingSLSAPredicate)
}