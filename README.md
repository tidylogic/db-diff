# db-diff

**Database Schema Comparison Tool** - Quickly and accurately compare database schemas between MySQL and PostgreSQL environments.

🇰🇷 [한국어 문서 보기](README-ko.md)

## Overview

`db-diff` is a CLI tool that detects and visualizes database schema differences across different environments (Dev, QA, Prod). It helps developers, DBAs, and DevOps engineers maintain database consistency.

### Key Features

- **Multi-Database Support**: MySQL and PostgreSQL (extensible architecture)
- **Precision Comparison Engine**: Compare tables, columns, indexes, constraints, views, and more
- **Flexible Output Formats**: Human-readable table format and JSON output
- **Selective Filtering**: Exclude specific tables or columns
- **Automatic Migration SQL Generation**: Convert schema differences to DDL automatically
- **YAML Configuration Support**: Manage complex comparison scenarios with config files

## Installation

### Prerequisites
- Go 1.26 or later

### Option 1: Using `go install`

```bash
go install github.com/tidylogic/db-diff/cmd/db-diff@latest
```

The binary will be installed in `$GOPATH/bin/db-diff` (typically `$HOME/go/bin/db-diff`).

Make sure `$GOPATH/bin` is in your `$PATH`:
```bash
# Add to ~/.bashrc, ~/.zshrc, or your shell config
export PATH=$PATH:$HOME/go/bin
```

### Option 2: Build from Source

```bash
git clone https://github.com/tidylogic/db-diff.git
cd db-diff
go build -o db-diff ./cmd/db-diff
```

The binary will be created in the current directory.

### Docker (Optional)
```bash
docker build -t db-diff .
docker run --rm db-diff compare --help
```

## Usage

### Basic Usage

```bash
./db-diff compare \
  --source "mysql://user:pass@localhost:3306/db1" \
  --target "mysql://user:pass@localhost:3307/db2"
```

### Comparing Different Databases

```bash
# MySQL to PostgreSQL comparison is not supported (same dialect required)
# Comparison only works within the same DBMS
./db-diff compare \
  --source "postgres://user:pass@localhost:5432/db1" \
  --target "postgres://user:pass@localhost:5433/db2"
```

### Using YAML Configuration File

Create `db-diff.yaml` in your project root or specify with `--config`:

```yaml
source:
  dsn: "mysql://user:pass@dev-db:3306/myapp"
  name: "Dev Database"

target:
  dsn: "mysql://user:pass@prod-db:3306/myapp"
  name: "Prod Database"

output: "table"  # table or json

schema: "myapp"  # Override path segment from DSN (optional)

ignore:
  tables:
    - "logs"
    - "temp_*"
  fields:
    - "created_at"
    - "updated_at"

migrate:
  enabled: true
  direction: "source_to_target"  # or target_to_source
  output: "migrate.sql"
```

### Generate Migration SQL

```bash
./db-diff compare \
  --source "mysql://user:pass@host1:3306/db" \
  --target "mysql://user:pass@host2:3306/db" \
  --migrate \
  --migrate-direction source_to_target \
  --migrate-output migration.sql
```

### Filtering Options

```bash
# Exclude specific tables
./db-diff compare \
  --source "mysql://user:pass@host1:3306/db" \
  --target "mysql://user:pass@host2:3306/db" \
  --ignore-tables "logs,sessions,temp_*"

# Exclude specific columns (from all tables)
./db-diff compare \
  --source "mysql://user:pass@host1:3306/db" \
  --target "mysql://user:pass@host2:3306/db" \
  --ignore-fields "created_at,updated_at"
```

### Output Formats

#### Table Format (Default)
```
$ ./db-diff compare --source "mysql://..." --target "mysql://..."

Source: Dev Database
Target: Prod Database

MISSING TABLES (in Target):
- users_temp (18 columns)

DIFFERENT TABLES:
- users
  - Column 'email': VARCHAR(100) -> VARCHAR(255)
  - Column 'is_admin': MISSING (in Target)
  - Index 'idx_email': MISSING (in Target)

- products
  - Column 'price': DECIMAL(10,2) -> DECIMAL(12,3)
  - Constraint 'fk_category': MISSING (in Target)
```

#### JSON Format
```bash
./db-diff compare \
  --source "mysql://..." \
  --target "mysql://..." \
  --output json | jq .
```

```json
{
  "source": { "name": "Dev Database", "tables_count": 15 },
  "target": { "name": "Prod Database", "tables_count": 14 },
  "differences": {
    "missing_in_target": [
      {
        "name": "users_temp",
        "columns": 18,
        "type": "table"
      }
    ],
    "different": [
      {
        "name": "users",
        "changes": [
          {
            "type": "column_type_change",
            "field": "email",
            "from": "VARCHAR(100)",
            "to": "VARCHAR(255)"
          }
        ]
      }
    ]
  }
}
```

## Architecture

### Core Modules

```
internal/
├── connector/        # Database connections (MySQL, PostgreSQL)
├── schema/          # Schema model definitions
├── diff/            # Comparison engine
├── output/          # Result output (table, JSON)
├── migrate/         # DDL generator
└── config/          # Configuration management
```

### Processing Flow

1. **Load Configuration**: Read from YAML file or CLI flags
2. **Database Connection**: Connect to source and target databases
3. **Extract Schemas**: Collect schema metadata from both databases in parallel
4. **Run Comparison**: Compare tables, columns, indexes, constraints
5. **Output Results**: Display results in table or JSON format
6. **Generate Migration** (optional): Convert differences to DDL and save to file

## Supported Comparison Items

| Item | MySQL | PostgreSQL | Notes |
|------|-------|-----------|-------|
| Table existence | ✓ | ✓ | Missing/Extra |
| Column definition | ✓ | ✓ | Type, NULL, defaults |
| Data types | ✓ | ✓ | Precise comparison |
| PRIMARY KEY | ✓ | ✓ | Column order included |
| UNIQUE INDEX | ✓ | ✓ | |
| FOREIGN KEY | ✓ | ✓ | Constraint names |
| Regular INDEX | ✓ | ✓ | |
| Column comments | ✓ | ✓ | DB metadata |
| Views | Planned | Planned | |
| Triggers | Planned | Planned | |

## Limitations

- **Same DBMS Only**: MySQL ↔ MySQL or PostgreSQL ↔ PostgreSQL
- **Read-Only**: Comparison only; automatic synchronization not supported
- **Procedures/Triggers**: Not currently included (planned)

## CLI Options

```bash
./db-diff compare --help

Usage:
  db-diff compare [flags]

Flags:
  --config string           Path to YAML config file (default: auto-discover db-diff.yaml)
  --source string           Source DSN (e.g., "mysql://user:pass@host:3306/db")
  --source-name string      Source display name (e.g., "DEV")
  --target string           Target DSN
  --target-name string      Target display name (e.g., "QA")
  --output string           Output format: "table" or "json" (default: table)
  --schema string           Schema name (overrides path segment from DSN)
  --ignore-tables string    Comma-separated tables to exclude
  --ignore-fields string    Comma-separated columns to exclude
  --migrate                 Enable migration SQL generation
  --migrate-direction string "source_to_target" or "target_to_source" (default: source_to_target)
  --migrate-output string   Migration file path (default: migrate.sql)
  -h, --help                Show help message
```

## Examples

### 1. Compare Dev and Production Databases

```bash
./db-diff compare \
  --source "mysql://dev_user:dev_pass@dev-db.example.com:3306/myapp" \
  --source-name "Development" \
  --target "mysql://prod_user:prod_pass@prod-db.example.com:3306/myapp" \
  --target-name "Production"
```

### 2. Compare QA Environment Against Template Database

```bash
./db-diff compare \
  --config deploy/qa-check.yaml \
  --output json > qa-report.json
```

### 3. Auto-Generate Migration Script

```bash
./db-diff compare \
  --source "mysql://staging:pass@staging-db:3306/shop" \
  --target "mysql://staging:pass@staging-db-new:3306/shop" \
  --migrate \
  --migrate-output scripts/migration-$(date +%Y%m%d).sql
```

## Contributing

Bug reports, feature requests, and pull requests are welcome!

### Development Setup

```bash
# Clone the repository
git clone https://github.com/tidylogic/db-diff.git
cd db-diff

# Run tests
go test ./...

# Build
go build -o db-diff ./cmd/db-diff
```

### Testing

Integration tests using Testcontainers spin up real database containers and verify schema extraction across multiple major versions:

| Database   | Versions tested       |
|------------|-----------------------|
| MySQL      | 5.7, 8.0, 8.4         |
| PostgreSQL | 13, 14, 15, 16, 17    |

All version subtests run in parallel. Docker must be available on the host.

```bash
# Run all compatibility tests (requires Docker)
go test -v -timeout 15m ./internal/connector/...

# Run a specific database only
go test -v -timeout 10m ./internal/connector/mysql/
go test -v -timeout 10m ./internal/connector/postgres/
```

## License

MIT License - See [LICENSE](LICENSE) for details

## Roadmap

### Phase 1 (Core) ✓
- Basic architecture and MySQL/PostgreSQL support
- Precision comparison engine
- JSON output and migration generation

### Phase 2 (Advanced)
- Complete YAML configuration
- Additional DBMS support (Oracle, SQL Server)
- Performance optimization

### Phase 3 (GUI) Planned
- Web-based visualization tool
- Advanced DDL generator
- Real-time synchronization features

### Phase 4 (Stability) Planned
- Integrated test automation
- Multi-version DBMS compatibility validation
- User documentation and tutorials

## Troubleshooting

### "cannot compare MySQL and PostgreSQL"
- Source and target must use the same DBMS
- Check DSN scheme: `mysql://` or `postgres://`

### Connection Refused Error
```bash
# 1. Verify database accessibility
mysql -h <host> -u <user> -p<password>

# 2. Check DSN format
# Correct format: mysql://user:password@host:port/database
# Note: If password contains @, URL-encode it (e.g., %40)
```

### Permission Denied Error
- Database user requires following permissions:
  - MySQL: `SELECT` (information_schema)
  - PostgreSQL: `CONNECT`, `USAGE` (schema)

## Support

- 📧 Bug Reports: [GitHub Issues](https://github.com/tidylogic/db-diff/issues)
- 📝 Documentation: See project Wiki
- 💬 Discussions: GitHub Discussions

## Changelog

### v0.1.0 (Initial Release)
- Basic MySQL and PostgreSQL support
- Schema comparison and migration generation
- YAML configuration file support
