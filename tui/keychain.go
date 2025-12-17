package tui

import (
	"github.com/zalando/go-keyring"
)

const (
	keychainService = "bored-azdo-tui"
	keychainOrgKey  = "organization"
	keychainProjKey = "project"
	keychainPATKey  = "pat"
	keychainUserKey = "username"
)

// SaveCredentials saves the Azure DevOps credentials to the system keychain
func SaveCredentials(org, project, pat, username string) error {
	if err := keyring.Set(keychainService, keychainOrgKey, org); err != nil {
		return err
	}
	if err := keyring.Set(keychainService, keychainProjKey, project); err != nil {
		return err
	}
	if err := keyring.Set(keychainService, keychainPATKey, pat); err != nil {
		return err
	}
	if username != "" {
		if err := keyring.Set(keychainService, keychainUserKey, username); err != nil {
			return err
		}
	}
	return nil
}

// LoadCredentials loads the Azure DevOps credentials from the system keychain
func LoadCredentials() (org, project, pat, username string, err error) {
	org, err = keyring.Get(keychainService, keychainOrgKey)
	if err != nil {
		return "", "", "", "", err
	}
	project, err = keyring.Get(keychainService, keychainProjKey)
	if err != nil {
		return "", "", "", "", err
	}
	pat, err = keyring.Get(keychainService, keychainPATKey)
	if err != nil {
		return "", "", "", "", err
	}
	// Username is optional, don't fail if not found
	username, _ = keyring.Get(keychainService, keychainUserKey)
	return org, project, pat, username, nil
}

// ClearCredentials removes the stored credentials from the keychain
func ClearCredentials() error {
	_ = keyring.Delete(keychainService, keychainOrgKey)
	_ = keyring.Delete(keychainService, keychainProjKey)
	_ = keyring.Delete(keychainService, keychainPATKey)
	_ = keyring.Delete(keychainService, keychainUserKey)
	return nil
}

// HasStoredCredentials checks if credentials are stored in the keychain
func HasStoredCredentials() bool {
	_, err := keyring.Get(keychainService, keychainPATKey)
	return err == nil
}
