package mssql

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"sync"
	"time"

	_ "github.com/microsoft/go-mssqldb"
	"puffsql/internal/model"
)

// Server owns one SQL Server pool. A database/sql connection is deliberately not pinned here;
// queries use Conn and a trailing USE statement so they remain on the requested database.
type Server struct {
	DB       *sql.DB
	Profile  model.ConnectionProfile
	Password string
	mu       sync.Mutex
	database string
}

func Open(profile model.ConnectionProfile, password string) (*Server, error) {
	dsn, err := DSN(profile, password)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		return nil, fmt.Errorf("open SQL Server: %w", err)
	}
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(30 * time.Minute)
	server := &Server{DB: db, Profile: profile, Password: password, database: profile.Database}
	return server, nil
}

func (s *Server) Ping(ctx context.Context) error {
	var one int
	if err := s.DB.QueryRowContext(ctx, "SELECT 1").Scan(&one); err != nil {
		return fmt.Errorf("SQL Server handshake/authentication failed: %w", err)
	}
	if one != 1 {
		return fmt.Errorf("SQL Server returned an unexpected health-check result")
	}
	return nil
}

func (s *Server) Close() error {
	return s.DB.Close()
}

func (s *Server) Database() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.database
}

func (s *Server) SetDatabase(database string) {
	s.mu.Lock()
	s.database = database
	s.mu.Unlock()
}

func (s *Server) PingLive(ctx context.Context) error {
	if err := s.Ping(ctx); err != nil {
		_ = s.Close()
		return err
	}
	return nil
}

func DSN(profile model.ConnectionProfile, password string) (string, error) {
	if profile.Host == "" {
		return "", fmt.Errorf("connection host cannot be empty")
	}
	port := profile.Port
	if port <= 0 || port > 65535 {
		port = 1433
	}
	query := url.Values{}
	if profile.Database != "" {
		query.Set("database", profile.Database)
	}
	query.Set("encrypt", strconv.FormatBool(profile.Encrypt))
	query.Set("TrustServerCertificate", strconv.FormatBool(profile.TrustServerCertificate))
	query.Set("app name", "PuffSQL")
	query.Set("connection timeout", "20")

	u := &url.URL{
		Scheme:   "sqlserver",
		Host:     net.JoinHostPort(profile.Host, strconv.Itoa(port)),
		RawQuery: query.Encode(),
	}
	if profile.Username != "" {
		u.User = url.UserPassword(profile.Username, password)
	}
	return u.String(), nil
}

func QuoteIdentifier(value string) string {
	result := make([]rune, 0, len(value)+2)
	result = append(result, '[')
	for _, r := range value {
		if r == ']' {
			result = append(result, ']')
		}
		result = append(result, r)
	}
	result = append(result, ']')
	return string(result)
}

func QualifiedTable(schema, table string) string {
	return QuoteIdentifier(schema) + "." + QuoteIdentifier(table)
}
