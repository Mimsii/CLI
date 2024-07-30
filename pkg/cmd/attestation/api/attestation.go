package api

import (
	"fmt"

	"github.com/sigstore/sigstore-go/pkg/bundle"
)

const (
	GetAttestationByRepoAndSubjectDigestPath  = "repos/%s/attestations/%s"
	GetAttestationByOwnerAndSubjectDigestPath = "orgs/%s/attestations/%s"
	GetUsernamePath                           = "users/%s"
	GetRepoNamePath                           = "repos/%s"
)

type ErrNoAttestations struct {
	name   string
	digest string
}

func (e ErrNoAttestations) Error() string {
	return fmt.Sprintf("no attestations found for digest %s in %s", e.name, e.digest)
}

func newErrNoAttestations(name, digest string) ErrNoAttestations {
	return ErrNoAttestations{name, digest}
}

type Attestation struct {
	Bundle *bundle.ProtobufBundle `json:"bundle"`
}

type AttestationsResponse struct {
	Attestations []*Attestation `json:"attestations"`
}

type UserResponse struct {
	Login string `json:"login"`
}
type RepoResponse struct {
	FullName string `json:"full_name"`
}
