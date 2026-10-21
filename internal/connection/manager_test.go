package connection

import (
	"errors"
	"testing"

	"puffsql/internal/model"
	"puffsql/internal/storage"
)

type fakeCredentials struct {
	values map[string]string
}

func (f *fakeCredentials) Get(id string) (string, bool, error) {
	value, ok := f.values[id]
	return value, ok, nil
}

func (f *fakeCredentials) Set(id, password string) error {
	f.values[id] = password
	return nil
}

func (f *fakeCredentials) Delete(id string) error {
	delete(f.values, id)
	return nil
}

func TestSaveUpdateCopyAndDeleteProfile(t *testing.T) {
	credentials := &fakeCredentials{values: map[string]string{}}
	manager := NewManager(storage.NewAt(t.TempDir()))
	manager.credentials = credentials

	password := "s3cret"
	saved, err := manager.SaveProfile(model.SaveConnectionRequest{
		ConnectionConfig: model.ConnectionConfig{
			Name: " Local ", Host: "localhost", Port: 1433, Database: "master", Username: "sa",
		},
		Password: &password,
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.Name != "Local" || credentials.values[saved.ID] != password {
		t.Fatalf("profile or keychain value not saved: %#v", saved)
	}

	renamed := "Renamed"
	updated, err := manager.SaveProfile(model.SaveConnectionRequest{
		ConnectionConfig: model.ConnectionConfig{
			Name: "Renamed", Host: "db.example", Port: 1444, Database: "Sales", Username: "app",
		},
		ID:               &saved.ID,
		CopyPasswordFrom: func() *string { value := saved.ID; return &value }(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.CreatedAt != saved.CreatedAt || updated.Name != renamed || credentials.values[updated.ID] != password {
		t.Fatalf("update did not preserve identity and password: %#v", updated)
	}
	if _, err := manager.SaveProfile(model.SaveConnectionRequest{
		ConnectionConfig: model.ConnectionConfig{Name: "renamed", Host: "other"},
	}); err == nil {
		t.Fatal("duplicate connection names must be rejected case-insensitively")
	}
	if err := manager.DeleteProfile(saved.ID); err != nil {
		t.Fatal(err)
	}
	if _, exists := credentials.values[saved.ID]; exists {
		t.Fatal("deleting a profile must remove its keychain secret")
	}
	profiles, err := manager.Profiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 0 {
		t.Fatalf("profile list was not updated: %#v", profiles)
	}
}

func TestProfileValidationAndCredentialFailures(t *testing.T) {
	manager := NewManager(storage.NewAt(t.TempDir()))
	manager.credentials = &fakeCredentials{values: map[string]string{}}
	if _, err := manager.SaveProfile(model.SaveConnectionRequest{}); err == nil {
		t.Fatal("empty names must be rejected")
	}
	manager.credentials = failingCredentials{}
	password := "secret"
	_, err := manager.SaveProfile(model.SaveConnectionRequest{
		ConnectionConfig: model.ConnectionConfig{Name: "Valid", Host: "localhost"},
		Password:         &password,
	})
	if err == nil {
		t.Fatal("keychain failures must not silently lose a password")
	}
}

type failingCredentials struct{}

func (failingCredentials) Get(string) (string, bool, error) {
	return "", false, errors.New("keychain unavailable")
}
func (failingCredentials) Set(string, string) error { return errors.New("keychain unavailable") }
func (failingCredentials) Delete(string) error      { return errors.New("keychain unavailable") }
