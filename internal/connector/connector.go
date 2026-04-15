package connector

import (
	"fmt"

	"github.com/tidylogic/db-diff/internal/connector/mysql"
	"github.com/tidylogic/db-diff/internal/connector/postgres"
	"github.com/tidylogic/db-diff/internal/schema"
)

// Connector abstracts a single RDBMS connection.
type Connector interface {
	// Connect opens and validates the database connection.
	Connect(dsn string) error

	// ExtractSchema retrieves the full schema metadata for the given database name.
	ExtractSchema(dbName string) (*schema.Schema, error)

	// Close releases the underlying connection.
	Close() error
}

// New returns a Connector for the given dialect.
// Supported dialects: "mysql", "postgres".
func New(dialect string) (Connector, error) {
	switch dialect {
	case "mysql":
		return mysql.New(), nil
	case "postgres":
		return postgres.New(), nil
	default:
		return nil, fmt.Errorf("unsupported dialect %q: supported values are \"mysql\" and \"postgres\"", dialect)
	}
}
