package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"puffsql/internal/model"
)

const appDirName = "PuffSQL"
const legacyDirName = "SQLight"

// Store owns the non-secret application files. Password material lives in the OS keychain.
type Store struct {
	dir      string
	mu       sync.Mutex
	path     string
	portable bool
}

func New() (*Store, error) {
	base, err := platformDataDir()
	if err != nil {
		return nil, err
	}
	if err := migrateLegacyDir(base); err != nil {
		return nil, err
	}
	dir := filepath.Join(base, appDirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create application data directory: %w", err)
	}
	return &Store{dir: dir, path: filepath.Join(dir, "connections.json"), portable: true}, nil
}

func NewAt(dir string) *Store {
	return &Store{dir: dir, path: filepath.Join(dir, "connections.json")}
}

func (s *Store) Dir() string {
	return s.dir
}

func (s *Store) ConnectionsPath() string {
	return s.path
}

func (s *Store) LoadProfiles() ([]model.ConnectionProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return []model.ConnectionProfile{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read connections file: %w", err)
	}

	var profiles []model.ConnectionProfile
	if err := json.Unmarshal(data, &profiles); err != nil {
		// A hand-edited or interrupted legacy file must not prevent application startup.
		return []model.ConnectionProfile{}, nil
	}
	if profiles == nil {
		profiles = []model.ConnectionProfile{}
	}
	return profiles, nil
}

func (s *Store) SaveProfiles(profiles []model.ConnectionProfile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if profiles == nil {
		profiles = []model.ConnectionProfile{}
	}
	data, err := json.MarshalIndent(profiles, "", "  ")
	if err != nil {
		return fmt.Errorf("serialize connections: %w", err)
	}
	data = append(data, '\n')
	return writeAtomic(s.path, data, 0o600)
}

func (s *Store) Path(name string) string {
	return filepath.Join(s.dir, name)
}

func (s *Store) EnsureTemplateFile() (string, error) {
	// Preserve an explicitly staged portable template file when one is present.
	if s.portable {
		for _, candidate := range portableCandidates("sql_custom_templates.json") {
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate, nil
			}
		}
	}

	path := s.Path("sql_custom_templates.json")
	_, err := os.Stat(path)
	if err == nil {
		return path, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect custom templates: %w", err)
	}

	now := time.Now().UnixMilli()
	sample := []model.CustomTemplateItem{{
		ID:            "custom-sample-1",
		Title:         "範例：自訂業務查詢 (Custom Business Query)",
		Category:      "custom",
		CategoryLabel: stringPtr("自訂範本"),
		Tags:          []string{"範例", "自訂", "Sample"},
		Description:   "這是可自由擴充的團隊 SQL 範本檔。使用任一文字編輯器修改後，回到 PuffSQL 重新載入即可生效。",
		Code:          "-- 自訂 SQL 語法範本\nSELECT TOP (50) *\nFROM dbo.YourTable\nORDER BY Id DESC;",
		IsCustom:      boolPtr(true),
		CreatedAt:     &now,
	}}
	data, err := json.MarshalIndent(sample, "", "  ")
	if err != nil {
		return "", err
	}
	data = append(data, '\n')
	if err := writeAtomic(path, data, 0o600); err != nil {
		return "", fmt.Errorf("create custom templates: %w", err)
	}
	return path, nil
}

func (s *Store) QueryLogPath() string {
	return s.Path("puffsql.log")
}

func (s *Store) AILogPath() string {
	return s.Path("ai.log")
}

func platformDataDir() (string, error) {
	if runtime.GOOS == "windows" {
		if value := os.Getenv("LOCALAPPDATA"); value != "" {
			return value, nil
		}
	}
	if runtime.GOOS != "windows" {
		if value := os.Getenv("XDG_DATA_HOME"); value != "" {
			return value, nil
		}
	}
	return os.UserConfigDir()
}

func migrateLegacyDir(base string) error {
	current := filepath.Join(base, appDirName)
	legacy := filepath.Join(base, legacyDirName)
	if _, err := os.Stat(current); err == nil {
		return nil
	}
	if _, err := os.Stat(legacy); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err := os.Rename(legacy, current); err != nil {
		return fmt.Errorf("migrate legacy SQLight data: %w", err)
	}
	return nil
}

func portableCandidates(name string) []string {
	var paths []string
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(cwd, name))
	}
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), name))
	}
	seen := map[string]bool{}
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		key := strings.ToLower(filepath.Clean(path))
		if !seen[key] {
			seen[key] = true
			result = append(result, path)
		}
	}
	return result
}

func writeAtomic(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func SortProfiles(profiles []model.ConnectionProfile) {
	sort.SliceStable(profiles, func(i, j int) bool {
		return strings.ToLower(profiles[i].Name) < strings.ToLower(profiles[j].Name)
	})
}

func boolPtr(v bool) *bool       { return &v }
func stringPtr(v string) *string { return &v }
