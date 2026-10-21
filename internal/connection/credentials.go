package connection

import (
	"errors"

	"github.com/zalando/go-keyring"
)

const credentialService = "PuffSQLDesktopClient"
const legacyCredentialService = "SQLightDesktopClient"

type CredentialStore interface {
	Get(id string) (string, bool, error)
	Set(id, password string) error
	Delete(id string) error
}

type OSKeyring struct{}

func (OSKeyring) Get(id string) (string, bool, error) {
	password, err := keyring.Get(credentialService, id)
	if err == nil {
		return password, true, nil
	}
	if !errors.Is(err, keyring.ErrNotFound) {
		return "", false, err
	}
	password, err = keyring.Get(legacyCredentialService, id)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	// Promote legacy credentials once so the old service name is no longer needed.
	_ = keyring.Set(credentialService, id, password)
	return password, true, nil
}

func (OSKeyring) Set(id, password string) error {
	return keyring.Set(credentialService, id, password)
}

func (OSKeyring) Delete(id string) error {
	err := keyring.Delete(credentialService, id)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return err
	}
	legacyErr := keyring.Delete(legacyCredentialService, id)
	if legacyErr != nil && !errors.Is(legacyErr, keyring.ErrNotFound) {
		return legacyErr
	}
	return nil
}
