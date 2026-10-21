package model

// ConnectionConfig is the editable portion of a SQL Server connection.
type ConnectionConfig struct {
	Name                   string  `json:"name"`
	Alias                  *string `json:"alias,omitempty"`
	Host                   string  `json:"host"`
	Port                   int     `json:"port"`
	Database               string  `json:"database"`
	Username               string  `json:"username"`
	Encrypt                bool    `json:"encrypt"`
	TrustServerCertificate bool    `json:"trustServerCertificate"`
	Color                  *string `json:"color,omitempty"`
	ModificationPrompt     *bool   `json:"modificationPrompt,omitempty"`
}

// ConnectionProfile is returned to the UI. Passwords are never included.
type ConnectionProfile struct {
	ConnectionConfig
	ID        string `json:"id"`
	Engine    string `json:"engine"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// SaveConnectionRequest carries one connection modal submission.
type SaveConnectionRequest struct {
	ConnectionConfig
	ID               *string `json:"id,omitempty"`
	Password         *string `json:"password,omitempty"`
	CopyPasswordFrom *string `json:"copyPasswordFrom,omitempty"`
}

type ColumnDef struct {
	Name     string `json:"name"`
	DataType string `json:"dataType"`
	Nullable bool   `json:"nullable"`
	Ordinal  int    `json:"ordinal"`
}

// BinaryCell intentionally mirrors the compact binary marker used by the Vue grid.
type BinaryCell struct {
	Type   string `json:"type"`
	Length int    `json:"length"`
}

type ResultSet struct {
	Columns     []ColumnDef `json:"columns"`
	Rows        [][]any     `json:"rows"`
	RowCount    int         `json:"rowCount"`
	TotalCount  int         `json:"totalCount"`
	IsTruncated bool        `json:"isTruncated"`
}

type QueryMessage struct {
	Level      string `json:"level"`
	Message    string `json:"message"`
	Code       *int   `json:"code,omitempty"`
	LineNumber *int   `json:"lineNumber,omitempty"`
	Timestamp  string `json:"timestamp"`
}

type QueryResult struct {
	ResultSets    []ResultSet    `json:"resultSets"`
	Messages      []QueryMessage `json:"messages"`
	AffectedRows  uint64         `json:"affectedRows"`
	ExecutionTime uint64         `json:"executionTimeMs"`
}

type DatabaseItem struct {
	Name string `json:"name"`
}

type TableItem struct {
	Schema string `json:"schema"`
	Name   string `json:"name"`
	Kind   string `json:"kind"`
}

type ColumnItem struct {
	Name               string  `json:"name"`
	DataType           string  `json:"dataType"`
	MaxLength          *int    `json:"maxLength,omitempty"`
	Precision          *int    `json:"precision,omitempty"`
	Scale              *int    `json:"scale,omitempty"`
	IsNullable         bool    `json:"isNullable"`
	IsPrimaryKey       bool    `json:"isPrimaryKey"`
	IsIdentity         bool    `json:"isIdentity"`
	IsComputed         bool    `json:"isComputed"`
	IsRowVersion       bool    `json:"isRowVersion"`
	DefaultDefinition  *string `json:"defaultDefinition,omitempty"`
	Collation          *string `json:"collation,omitempty"`
	ComputedExpression *string `json:"computedExpression,omitempty"`
	IsPersisted        bool    `json:"isPersisted"`
	IdentitySeed       *int64  `json:"identitySeed,omitempty"`
	IdentityIncrement  *int64  `json:"identityIncrement,omitempty"`
	Description        *string `json:"description,omitempty"`
}

type TableSchema struct {
	Schema  string       `json:"schema"`
	Name    string       `json:"name"`
	Kind    string       `json:"kind"`
	Columns []ColumnItem `json:"columns"`
}

type ForeignKeyItem struct {
	ConstraintName string `json:"constraintName"`
	FromSchema     string `json:"fromSchema"`
	FromTable      string `json:"fromTable"`
	FromColumn     string `json:"fromColumn"`
	ToSchema       string `json:"toSchema"`
	ToTable        string `json:"toTable"`
	ToColumn       string `json:"toColumn"`
}

type KeyColumnRef struct {
	Name       string `json:"name"`
	Descending bool   `json:"descending"`
}

type TableConstraintItem struct {
	Name          string         `json:"name"`
	Kind          string         `json:"kind"`
	KeyColumns    []KeyColumnRef `json:"keyColumns"`
	Definition    *string        `json:"definition"`
	DefaultColumn *string        `json:"defaultColumn"`
	IsClustered   *bool          `json:"isClustered"`
	IsDisabled    bool           `json:"isDisabled"`
	IsNotTrusted  bool           `json:"isNotTrusted"`
	RefSchema     *string        `json:"refSchema"`
	RefTable      *string        `json:"refTable"`
	RefColumns    []string       `json:"refColumns"`
	OnDelete      *string        `json:"onDelete"`
	OnUpdate      *string        `json:"onUpdate"`
}

type IndexColumnRef struct {
	Name       string `json:"name"`
	Descending bool   `json:"descending"`
	IsIncluded bool   `json:"isIncluded"`
}

type TableIndexItem struct {
	Name               string           `json:"name"`
	TypeDesc           string           `json:"typeDesc"`
	IsUnique           bool             `json:"isUnique"`
	IsPrimaryKey       bool             `json:"isPrimaryKey"`
	IsUniqueConstraint bool             `json:"isUniqueConstraint"`
	IsDisabled         bool             `json:"isDisabled"`
	Columns            []IndexColumnRef `json:"columns"`
	FilterDefinition   *string          `json:"filterDefinition"`
}

type TableTriggerItem struct {
	Name        string   `json:"name"`
	IsInsteadOf bool     `json:"isInsteadOf"`
	IsDisabled  bool     `json:"isDisabled"`
	Events      []string `json:"events"`
	Definition  *string  `json:"definition"`
}

type TableObjects struct {
	Columns     []ColumnItem          `json:"columns"`
	Constraints []TableConstraintItem `json:"constraints"`
	Indexes     []TableIndexItem      `json:"indexes"`
	Triggers    []TableTriggerItem    `json:"triggers"`
}

type ImportCapabilities struct {
	IdentityColumn         *string `json:"identityColumn"`
	WritableColumnCount    int     `json:"writableColumnCount"`
	CanAlterTable          bool    `json:"canAlterTable"`
	EngineEdition          int     `json:"engineEdition"`
	SupportsIdentityInsert bool    `json:"supportsIdentityInsert"`
	DisabledReason         *string `json:"disabledReason"`
}

type ImportRowPayload struct {
	Line   int       `json:"line"`
	Values []*string `json:"values"`
}

type ImportRowError struct {
	Line       int     `json:"line"`
	Column     *string `json:"column"`
	Message    string  `json:"message"`
	ServerCode *int    `json:"serverCode"`
}

type ImportResult struct {
	InsertedCount uint64           `json:"insertedCount"`
	RolledBack    bool             `json:"rolledBack"`
	Errors        []ImportRowError `json:"errors"`
	ExecutionTime uint64           `json:"executionTimeMs"`
}

type ImportProgressEvent struct {
	ImportID      string `json:"importId"`
	ProcessedRows int    `json:"processedRows"`
	TotalRows     int    `json:"totalRows"`
}

type CustomTemplateItem struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Category      string   `json:"category"`
	CategoryLabel *string  `json:"categoryLabel,omitempty"`
	Tags          []string `json:"tags"`
	Description   string   `json:"description"`
	Code          string   `json:"code"`
	IsCustom      *bool    `json:"isCustom,omitempty"`
	CreatedAt     *int64   `json:"createdAt,omitempty"`
}

type CustomTemplatesResponse struct {
	FilePath  string               `json:"filePath"`
	Templates []CustomTemplateItem `json:"templates"`
}

type SQLFileNode struct {
	Name         string        `json:"name"`
	Path         string        `json:"path"`
	IsDir        bool          `json:"is_dir"`
	Size         *int64        `json:"size,omitempty"`
	ModifiedTime *string       `json:"modified_time,omitempty"`
	Children     []SQLFileNode `json:"children,omitempty"`
}

type ColumnChangeRequest struct {
	ConnectionID      string  `json:"connectionId"`
	Database          string  `json:"database"`
	Schema            string  `json:"schema"`
	Table             string  `json:"table"`
	Operation         string  `json:"operation"`
	OriginalName      string  `json:"originalName,omitempty"`
	ColumnName        string  `json:"columnName"`
	DataType          string  `json:"dataType"`
	Length            *int    `json:"length,omitempty"`
	Precision         *int    `json:"precision,omitempty"`
	Scale             *int    `json:"scale,omitempty"`
	IsNullable        bool    `json:"isNullable"`
	DefaultExpression *string `json:"defaultExpression,omitempty"`
	Description       *string `json:"description,omitempty"`
}

type ColumnDependency struct {
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

type ColumnChangeIssue struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ColumnChangePreview struct {
	SQL            string              `json:"sql"`
	Fingerprint    string              `json:"fingerprint"`
	Warnings       []ColumnChangeIssue `json:"warnings"`
	BlockingIssues []ColumnChangeIssue `json:"blockingIssues"`
	Dependencies   []ColumnDependency  `json:"dependencies"`
	AffectedRows   int64               `json:"affectedRows"`
}

type ColumnChangeResult struct {
	ExecutionTimeMS uint64 `json:"executionTimeMs"`
}

type ExcelExportProgress struct {
	Current int    `json:"current"`
	Total   int    `json:"total"`
	Table   string `json:"table"`
	Sheets  int    `json:"sheets"`
	Rows    int64  `json:"rows"`
	Columns int    `json:"columns"`
}

type ExcelExportResult struct {
	Canceled        bool   `json:"canceled"`
	FilePath        string `json:"filePath"`
	TableCount      int    `json:"tableCount"`
	SheetCount      int    `json:"sheetCount"`
	ColumnCount     int    `json:"columnCount"`
	TotalRows       int64  `json:"totalRows"`
	TruncatedCount  int    `json:"truncatedCount"`
	ExecutionTimeMS uint64 `json:"executionTimeMs"`
}

type ExcelTableRef struct {
	Schema string `json:"schema"`
	Name   string `json:"name"`
}
