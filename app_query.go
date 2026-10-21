package main

import (
	"strings"
	"time"

	"puffsql/internal/model"
)

func (a *App) ExecuteQuery(
	connectionID string,
	database *string,
	sql string,
	maxRows *int,
	requestID *string,
	timeoutSeconds *int,
) (model.QueryResult, error) {
	if strings.TrimSpace(sql) == "" {
		return model.QueryResult{
			ResultSets: []model.ResultSet{},
			Messages:   []model.QueryMessage{},
		}, nil
	}
	result, err := a.manager.ExecuteQuery(
		a.context(), connectionID, database, sql, maxRows, requestID, timeoutSeconds, nil,
	)
	if err != nil {
		a.logger.LogQuery(connectionID, databaseValue(database), sql, []model.QueryMessage{{
			Level: "error", Message: err.Error(), Timestamp: time.Now().Format(time.RFC3339),
		}}, result.ExecutionTime, "ERROR")
		return result, err
	}
	status := "SUCCESS"
	for _, message := range result.Messages {
		if message.Level == "error" {
			status = "ERROR"
		} else if message.Level == "warning" && strings.Contains(message.Message, "cancelled") {
			status = "CANCELLED"
		}
	}
	a.logger.LogQuery(connectionID, databaseValue(database), sql, result.Messages, result.ExecutionTime, status)
	return result, nil
}

func (a *App) CancelQuery(connectionID string, requestID *string) error {
	return a.manager.CancelQuery(connectionID, requestID)
}

func (a *App) GetConnectionSPID(connectionID string) *int {
	return a.manager.SPID(connectionID)
}

func (a *App) OpenQueryLogFile() (string, error) {
	return a.logger.Open(a.logger.QueryPath())
}

func (a *App) GetQueryLogPath() string {
	return a.logger.QueryPath()
}

func databaseValue(database *string) string {
	if database == nil {
		return ""
	}
	return *database
}
