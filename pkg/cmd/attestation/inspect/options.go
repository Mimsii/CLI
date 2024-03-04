package inspect

import (
	"fmt"
	"path/filepath"

	"github.com/cli/cli/v2/pkg/cmd/attestation/artifact/digest"
	"github.com/cli/cli/v2/pkg/cmd/attestation/artifact/oci"
	"github.com/cli/cli/v2/pkg/cmd/attestation/logger"
)

// Options captures the options for the inspect command
type Options struct {
	ArtifactPath      string
	BundlePath        string
	DigestAlgorithm   string
	JsonResult        bool
	Verbose           bool
	Logger            *logger.Logger
	OCIClient         oci.Client
}

// Clean cleans the file path option values
func (opts *Options) Clean() {
	opts.BundlePath = filepath.Clean(opts.BundlePath)
}

// ConfigureLogger configures a logger using configuration provided
// through the options
func (opts *Options) ConfigureLogger() {
	opts.Logger = logger.NewLogger(false, opts.Verbose)
}

// AreFlagsValid checks that the provided flag combination is valid
// and returns an error otherwise
func (opts *Options) AreFlagsValid() error {
	// either BundlePath or Repo must be set to configure offline or online mode
	if opts.BundlePath == "" {
		return fmt.Errorf("bundle must be provided")
	}

	// DigestAlgorithm must not be empty
	if opts.DigestAlgorithm == "" {
		return fmt.Errorf("digest-alg cannot be empty")
	}

	if !digest.IsValidDigestAlgorithm(opts.DigestAlgorithm) {
		return fmt.Errorf("invalid digest algorithm '%s' provided in digest-alg", opts.DigestAlgorithm)
	}

	return nil
}