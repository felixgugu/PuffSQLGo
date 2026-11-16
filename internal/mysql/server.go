// Package mysql implements dbengine.Server for MySQL and MariaDB.
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	driver "github.com/go-sql-driver/mysql"
	"puffsql/internal/dbengine"
	"puffsql/internal/model"
)

// Server owns one MySQL pool. MySQL has no schema level below the database, so every catalog
// call reports the database name in the model's Schema fields.
type Server struct {
	DB       *sql.DB
	Profile  model.ConnectionProfile
	mu       sync.Mutex
	database string
}

func Open(profile model.ConnectionProfile, password string) (*Server, error) {
	config, err := buildConfig(profile, password)
	if err != nil {
		return nil, err
	}
	connector, err := driver.NewConnector(config)
	if err != nil {
		return nil, fmt.Errorf("open MySQL: %w", err)
	}
	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(30 * time.Minute)
	return &Server{DB: db, Profile: profile, database: profile.Database}, nil
}

// buildConfig maps a profile to driver settings. Statements are split client-side, so
// multiStatements stays off, and date/time values stay text so zero dates survive.
func buildConfig(profile model.ConnectionProfile, password string) (*driver.Config, error) {
	if strings.TrimSpace(profile.Host) == "" {
		return nil, fmt.Errorf("connection host cannot be empty")
	}
	port := profile.Port
	if port <= 0 || port > 65535 {
		port = 3306
	}
	config := driver.NewConfig()
	config.Net = "tcp"
	config.Addr = net.JoinHostPort(profile.Host, strconv.Itoa(port))
	config.User = profile.Username
	config.Passwd = password
	config.DBName = profile.Database
	config.Timeout = 20 * time.Second
	config.TLSConfig = tlsMode(profile)
	config.ConnectionAttributes = "program_name:PuffSQL"
	return config, nil
}

// tlsMode reuses the two existing profile switches: "encrypt" turns TLS on and "trust server
// certificate" skips certificate verification.
func tlsMode(profile model.ConnectionProfile) string {
	switch {
	case !profile.Encrypt:
		return "false"
	case profile.TrustServerCertificate:
		return "skip-verify"
	default:
		return "true"
	}
}

// DSN renders the driver DSN for a profile (used by tests and diagnostics; contains the password).
func DSN(profile model.ConnectionProfile, password string) (string, error) {
	config, err := buildConfig(profile, password)
	if err != nil {
		return "", err
	}
	return config.FormatDSN(), nil
}

func (s *Server) Ping(ctx context.Context) error {
	var one int
	if err := s.DB.QueryRowContext(ctx, "SELECT 1").Scan(&one); err != nil {
		return fmt.Errorf("MySQL handshake/authentication failed: %w", err)
	}
	if one != 1 {
		return fmt.Errorf("MySQL returned an unexpected health-check result")
	}
	return nil
}

func (s *Server) Close() error {
	return s.DB.Close()
}

// KillSession interrupts the statement another connection is running without closing it.
func (s *Server) KillSession(ctx context.Context, sessionID int) error {
	_, err := s.DB.ExecContext(ctx, fmt.Sprintf("KILL QUERY %d", sessionID))
	return err
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

// resolveDatabase returns the explicit database when given, otherwise the selected one.
func (s *Server) resolveDatabase(database *string) string {
	if database != nil && strings.TrimSpace(*database) != "" {
		return strings.TrimSpace(*database)
	}
	return s.Database()
}

// QuoteIdentifier wraps a name in backticks, doubling any backtick inside it.
func QuoteIdentifier(value string) string {
	return "`" + strings.ReplaceAll(value, "`", "``") + "`"
}

// QualifiedTable is `database`.`table`.
func QualifiedTable(database, table string) string {
	return QuoteIdentifier(database) + "." + QuoteIdentifier(table)
}

var _ dbengine.Server = (*Server)(nil)
