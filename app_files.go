package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
	"unicode/utf16"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"puffsql/internal/model"
)

func (a *App) LoadCustomTemplates() (model.CustomTemplatesResponse, error) {
	path, err := a.store.EnsureTemplateFile()
	if err != nil {
		return model.CustomTemplatesResponse{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return model.CustomTemplatesResponse{}, fmt.Errorf("read custom templates: %w", err)
	}
	templates := []model.CustomTemplateItem{}
	if err := json.Unmarshal(data, &templates); err != nil {
		// Keep the editable source intact and let the UI show no custom templates.
		templates = []model.CustomTemplateItem{}
	}
	absolute, _ := filepath.Abs(path)
	return model.CustomTemplatesResponse{FilePath: absolute, Templates: templates}, nil
}

func (a *App) SaveCustomTemplates(templates []model.CustomTemplateItem) (string, error) {
	path, err := a.store.EnsureTemplateFile()
	if err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(templates, "", "  ")
	if err != nil {
		return "", err
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", fmt.Errorf("write custom templates: %w", err)
	}
	absolute, _ := filepath.Abs(path)
	return absolute, nil
}

func (a *App) OpenCustomTemplatesFile() (string, error) {
	path, err := a.store.EnsureTemplateFile()
	if err != nil {
		return "", err
	}
	absolute, _ := filepath.Abs(path)
	if err := revealPath(absolute); err != nil {
		return absolute, err
	}
	return absolute, nil
}

func (a *App) PickSQLFolder() (*string, error) {
	if a.ctx == nil {
		return nil, nil
	}
	path, err := wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title:                "選取 SQL 監控資料夾",
		CanCreateDirectories: true,
	})
	if err != nil || path == "" {
		return nil, err
	}
	return &path, nil
}

func (a *App) ScanSQLFolder(folderPath string) (model.SQLFileNode, error) {
	root, err := filepath.Abs(folderPath)
	if err != nil {
		return model.SQLFileNode{}, err
	}
	info, err := os.Stat(root)
	if err != nil {
		return model.SQLFileNode{}, fmt.Errorf("資料夾不存在: %s", folderPath)
	}
	if !info.IsDir() {
		return model.SQLFileNode{}, fmt.Errorf("指定路徑非資料夾: %s", folderPath)
	}
	children, err := scanSQLChildren(root)
	if err != nil {
		return model.SQLFileNode{}, err
	}
	return model.SQLFileNode{
		Name: filepath.Base(root), Path: root, IsDir: true, Children: children,
	}, nil
}

func (a *App) ReadSQLFile(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("無法讀取檔案 [%s]: %w", filePath, err)
	}
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return string(data[3:]), nil
	}
	if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xFE {
		return decodeUTF16(data[2:], true), nil
	}
	if len(data) >= 2 && data[0] == 0xFE && data[1] == 0xFF {
		return decodeUTF16(data[2:], false), nil
	}
	return string(data), nil
}

func (a *App) WriteSQLFile(filePath string, content string) error {
	if err := os.WriteFile(filePath, []byte(content), 0o600); err != nil {
		return fmt.Errorf("無法寫入檔案 [%s]: %w", filePath, err)
	}
	return nil
}

func (a *App) RenameSQLPath(oldPath string, newPath string) error {
	if _, err := os.Stat(oldPath); err != nil {
		return fmt.Errorf("原檔案或資料夾不存在: %s", oldPath)
	}
	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("目標路徑已存在同名檔案或資料夾: %s", newPath)
	}
	if _, err := os.Stat(filepath.Dir(newPath)); err != nil {
		return fmt.Errorf("目標上層資料夾不存在: %s", filepath.Dir(newPath))
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("重新命名失敗: %w", err)
	}
	return nil
}

func scanSQLChildren(dir string) ([]model.SQLFileNode, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []model.SQLFileNode{}, err
	}
	sort.SliceStable(entries, func(i, j int) bool {
		leftDir, rightDir := entries[i].IsDir(), entries[j].IsDir()
		if leftDir != rightDir {
			return leftDir
		}
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})
	result := []model.SQLFileNode{}
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			children, err := scanSQLChildren(path)
			if err != nil || len(children) == 0 {
				continue
			}
			result = append(result, model.SQLFileNode{
				Name: entry.Name(), Path: path, IsDir: true, Children: children,
			})
			continue
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".sql") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		size := info.Size()
		modified := info.ModTime().UTC().Format(time.RFC3339)
		result = append(result, model.SQLFileNode{
			Name: entry.Name(), Path: path, IsDir: false, Size: &size, ModifiedTime: &modified,
		})
	}
	return result, nil
}

func decodeUTF16(data []byte, littleEndian bool) string {
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}
	units := make([]uint16, len(data)/2)
	for i := range units {
		if littleEndian {
			units[i] = uint16(data[i*2]) | uint16(data[i*2+1])<<8
		} else {
			units[i] = uint16(data[i*2])<<8 | uint16(data[i*2+1])
		}
	}
	return string(utf16.Decode(units))
}

func revealPath(path string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		command = exec.Command("explorer", "/select,"+path)
	case "darwin":
		command = exec.Command("open", "-R", path)
	default:
		command = exec.Command("xdg-open", filepath.Dir(path))
	}
	err := command.Start()
	if err != nil && !errors.Is(err, exec.ErrNotFound) {
		return err
	}
	return nil
}
