package applog

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"puffsql/internal/model"
)

type Logger struct {
	queryPath string
	aiPath    string
	mu        sync.Mutex
}

func New(queryPath, aiPath string) *Logger {
	return &Logger{queryPath: queryPath, aiPath: aiPath}
}

func (l *Logger) Init() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now().Format("2006-01-02 15:04:05")
	queryHeader := fmt.Sprintf(
		"================================================================================\n"+
			"PuffSQL Query Log - Session Started at %s\n"+
			"Log File: %s\n"+
			"================================================================================\n\n",
		now, l.queryPath,
	)
	aiHeader := fmt.Sprintf(
		"================================================================================\n"+
			"PuffSQL AI Request Log - Session Started at %s\n"+
			"Log File: %s\n"+
			"================================================================================\n\n",
		now, l.aiPath,
	)
	if err := writeFile(l.queryPath, queryHeader); err != nil {
		return err
	}
	return writeFile(l.aiPath, aiHeader)
}

func (l *Logger) QueryPath() string { return l.queryPath }
func (l *Logger) AIPath() string    { return l.aiPath }

func (l *Logger) LogQuery(
	connectionID, database, sql string,
	messages []model.QueryMessage,
	durationMS uint64,
	status string,
) {
	l.mu.Lock()
	defer l.mu.Unlock()
	var body strings.Builder
	fmt.Fprintf(&body, "[%s] [%s] connection=%s database=%s duration=%dms\n",
		time.Now().Format("2006-01-02 15:04:05"), status, connectionID, database, durationMS)
	body.WriteString("--- SQL ---\n")
	body.WriteString(strings.TrimSpace(sql))
	body.WriteString("\n")
	if len(messages) > 0 {
		body.WriteString("--- MESSAGES ---\n")
		for _, message := range messages {
			fmt.Fprintf(&body, "[%s] %s\n", strings.ToUpper(message.Level), message.Message)
		}
	}
	body.WriteString("--------------------------------------------------------------------------------\n\n")
	_ = appendFile(l.queryPath, body.String())
}

func (l *Logger) LogAIRequest(endpoint, method, content string) {
	if method == "" {
		method = "POST"
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	entry := fmt.Sprintf(
		"[%s] [REQUEST] [%s] %s\n--- REQUEST CONTENT ---\n%s\n"+
			"--------------------------------------------------------------------------------\n\n",
		time.Now().Format("2006-01-02 15:04:05"), method, endpoint, prettyJSON(content),
	)
	_ = appendFile(l.aiPath, entry)
}

func (l *Logger) LogAIResponse(
	endpoint, content string,
	status *int,
	durationMS *uint64,
	isError bool,
) {
	l.mu.Lock()
	defer l.mu.Unlock()
	statusText := "OK"
	if status != nil {
		statusText = fmt.Sprintf("HTTP %d", *status)
	} else if isError {
		statusText = "ERROR"
	}
	duration := ""
	if durationMS != nil {
		duration = fmt.Sprintf(" [Duration: %dms]", *durationMS)
	}
	tag := "RESPONSE"
	if isError {
		tag = "RESPONSE ERROR"
	}
	entry := fmt.Sprintf(
		"[%s] [%s] [%s] %s%s\n--- RESPONSE CONTENT ---\n%s\n"+
			"--------------------------------------------------------------------------------\n\n",
		time.Now().Format("2006-01-02 15:04:05"), tag, statusText, endpoint, duration,
		prettyJSON(content),
	)
	_ = appendFile(l.aiPath, entry)
}

func (l *Logger) Open(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		absolute = path
	}
	if _, err := os.Stat(absolute); err != nil {
		if os.IsNotExist(err) {
			if err := writeFile(absolute, ""); err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	}
	var command *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		command = exec.Command("explorer", absolute)
	case "darwin":
		command = exec.Command("open", absolute)
	default:
		command = exec.Command("xdg-open", absolute)
	}
	if err := command.Start(); err != nil {
		return absolute, fmt.Errorf("open log file: %w", err)
	}
	return absolute, nil
}

func Clear(path, title string) (string, error) {
	header := fmt.Sprintf(
		"================================================================================\n"+
			"%s - Session Started at %s\n"+
			"Log File: %s\n"+
			"================================================================================\n\n",
		title, time.Now().Format("2006-01-02 15:04:05"), path,
	)
	if err := writeFile(path, header); err != nil {
		return "", err
	}
	return path, nil
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o600)
}

func appendFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(content)
	return err
}

func prettyJSON(content string) string {
	var value any
	if err := json.Unmarshal([]byte(content), &value); err != nil {
		return strings.TrimSpace(content)
	}
	pretty, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return strings.TrimSpace(content)
	}
	return string(pretty)
}
