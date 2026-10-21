package main

import "puffsql/internal/applog"

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
