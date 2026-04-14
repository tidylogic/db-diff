package schema

// Schema represents the full extracted metadata of a single database.
type Schema struct {
	Name   string
	Tables map[string]Table
	Views  map[string]View
}

// Table represents a database table with all its metadata.
type Table struct {
	Name        string
	Comment     string
	Columns     []Column
	Indexes     []Index
	Constraints []Constraint
}

// Column represents a single column in a table.
type Column struct {
	Name         string
	OrdinalPos   int
	DataType     string // normalized type, e.g. "varchar", "int"
	RawType      string // driver-native full type, e.g. "varchar(255)", "int(11)"
	Nullable     bool
	Default      *string // nil = no default; pointer to "" = explicit empty string default
	Comment      string
	CharMaxLen   *int64
	NumPrecision *int64
	NumScale     *int64
}

// Index represents a table index.
type Index struct {
	Name      string
	Columns   []string
	Unique    bool
	IsPrimary bool
}

// Constraint represents a table constraint (PK, FK, UNIQUE, CHECK).
type Constraint struct {
	Name       string
	Type       string // "PRIMARY KEY" | "FOREIGN KEY" | "UNIQUE" | "CHECK"
	Columns    []string
	RefTable   string
	RefColumns []string
}

// View represents a database view.
type View struct {
	Name       string
	Definition string
}
