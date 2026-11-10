package main

import (
	"fmt"
	"strings"

	"puffsql/internal/applog"
	"puffsql/internal/errlog"
)

func (a *App) LogAIRequest(endpoint string, method *string, content string) error {
	value := "POST"
	if method != nil && *method != "" {
		value = *method
	}
	a.logger.LogAIRequest(endpoint, value, content)
	return nil
}

func (a *App) LogAIResponse(
	endpoint string,
	content string,
	status *int,
	durationMS *uint64,
	isError *bool,
) error {
	failed := isError != nil && *isError
	a.logger.LogAIResponse(endpoint, content, status, durationMS, failed)
	return nil
}

func (a *App) OpenAILogFile() (string, error) {
	return a.logger.Open(a.logger.AIPath())
}

func (a *App) GetAILogPath() string {
	return a.logger.AIPath()
}

func (a *App) ClearAILog() (string, error) {
	return applog.Clear(a.logger.AIPath(), "PuffSQL AI Request Log")
}

func (a *App) GetErrorLogPath() string {
	return a.errLogger.Path()
}

func (a *App) OpenErrorLog() (string, error) {
	path := a.errLogger.Path()
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("error log is not available")
	}
	return a.logger.Open(path)
}

// LogClientError lets the Vue layer persist unhandled exceptions and rejected promises in the
// same diagnostics file as backend failures.
func (a *App) LogClientError(level string, message string, stack string) error {
	normalized := strings.ToUpper(strings.TrimSpace(level))
	if normalized == "" {
		normalized = "ERROR"
	}
	if len(normalized) > 16 {
		normalized = normalized[:16]
	}
	a.errLogger.Write(errlog.Entry{
		Level:   normalized,
		Source:  "client",
		Message: message,
		Stack:   stack,
	})
	return nil
}
