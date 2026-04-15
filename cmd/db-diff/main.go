package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/spf13/cobra"

	"github.com/tidylogic/db-diff/internal/config"
	"github.com/tidylogic/db-diff/internal/connector"
	"github.com/tidylogic/db-diff/internal/diff"
	"github.com/tidylogic/db-diff/internal/migrate"
	"github.com/tidylogic/db-diff/internal/output"
	"github.com/tidylogic/db-diff/internal/schema"
	"github.com/tidylogic/db-diff/web"
)

func main() {
	if err := rootCmd().Execute(); err != nil {
		os.Exit(2)
	}
}

func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "db-diff",
		Short: "Cross-platform RDBMS schema diff tool",
		Long:  "Compare database schemas across MySQL and PostgreSQL environments.",
	}
	root.AddCommand(compareCmd())
	root.AddCommand(versionCmd())
	root.AddCommand(webCmd())
	return root
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the db-diff version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("db-diff v0.1.0")
		},
	}
}

func webCmd() *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "web",
		Short: "Start the web GUI server",
		Long:  "Launch the db-diff web GUI. Open the printed URL in your browser, then load a JSON diff file.",
		RunE: func(cmd *cobra.Command, args []string) error {
			srv := &web.Server{Port: port}
			return srv.ListenAndServe()
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 8080, "port to listen on")
	return cmd
}

func compareCmd() *cobra.Command {
	var flags config.CLIFlags

	cmd := &cobra.Command{
		Use:   "compare",
		Short: "Compare schemas between two database endpoints",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCompare(flags)
		},
	}

	// Config file
	cmd.Flags().StringVar(&flags.ConfigPath, "config", "", "path to YAML config file (default: auto-discover db-diff.yaml)")

	// Source
	cmd.Flags().StringVar(&flags.Source, "source", "", `source DSN, e.g. "mysql://user:pass@host:3306/db"`)
	cmd.Flags().StringVar(&flags.SourceName, "source-name", "", "human-readable label for source (e.g. DEV)")

	// Target
	cmd.Flags().StringVar(&flags.Target, "target", "", `target DSN, e.g. "mysql://user:pass@host:3307/db"`)
	cmd.Flags().StringVar(&flags.TargetName, "target-name", "", "human-readable label for target (e.g. QA)")

	// Output
	cmd.Flags().StringVar(&flags.Output, "output", "", `output format: "table" or "json" (default: table)`)

	// Schema / DB name override
	cmd.Flags().StringVar(&flags.Schema, "schema", "", "database/schema name to extract (overrides name from DSN path)")

	// Filters
	cmd.Flags().StringVar(&flags.IgnoreTables, "ignore-tables", "", "comma-separated table names to skip")
	cmd.Flags().StringVar(&flags.IgnoreFields, "ignore-fields", "", "comma-separated column names to skip in every table")

	// Migrate
	cmd.Flags().BoolVar(&flags.Migrate, "migrate", false, "generate migration SQL file")
	cmd.Flags().StringVar(&flags.MigrateDirection, "migrate-direction", "", `"apply_to_target" or "apply_to_source" (default: apply_to_target)`)
	cmd.Flags().StringVar(&flags.MigrateOutput, "migrate-output", "", "output file for migration SQL (default: migrate.sql)")

	return cmd
}

func runCompare(flags config.CLIFlags) error {
	// 1. Load YAML config (auto-discovers db-diff.yaml in CWD if --config not set)
	cfg, err := config.Load(flags.ConfigPath)
	if err != nil {
		return err
	}

	// 2. Merge CLI flags over YAML (CLI wins)
	config.Merge(cfg, flags)

	// 3. Resolve drivers from DSN schemes
	if err := config.ResolveDriver(&cfg.Source); err != nil {
		return fmt.Errorf("source: %w", err)
	}
	if err := config.ResolveDriver(&cfg.Target); err != nil {
		return fmt.Errorf("target: %w", err)
	}

	// 4. Validate same dialect
	if cfg.Source.Driver != cfg.Target.Driver {
		return fmt.Errorf("cannot compare %s and %s: source and target must use the same database dialect",
			cfg.Source.Driver, cfg.Target.Driver)
	}
	dialect := cfg.Source.Driver

	// 5. Resolve display names (fall back to DSN when label not set)
	if cfg.Source.Name == "" {
		cfg.Source.Name = cfg.Source.DSN
	}
	if cfg.Target.Name == "" {
		cfg.Target.Name = cfg.Target.DSN
	}

	// 6. Create connectors
	srcConn, err := connector.New(dialect)
	if err != nil {
		return fmt.Errorf("creating source connector: %w", err)
	}
	defer srcConn.Close()

	tgtConn, err := connector.New(dialect)
	if err != nil {
		return fmt.Errorf("creating target connector: %w", err)
	}
	defer tgtConn.Close()

	// 7. Connect to both endpoints
	if err := srcConn.Connect(cfg.Source.DSN); err != nil {
		return fmt.Errorf("connecting to source (%s): %w", cfg.Source.Name, err)
	}
	if err := tgtConn.Connect(cfg.Target.DSN); err != nil {
		return fmt.Errorf("connecting to target (%s): %w", cfg.Target.Name, err)
	}

	// 8. Determine DB/schema name
	dbName := cfg.Schema
	if dbName == "" {
		dbName = extractDBName(cfg.Source.DSN)
	}

	// 9. Extract both schemas concurrently
	var (
		srcSchema, tgtSchema *schema.Schema
		srcErr, tgtErr       error
		wg                   sync.WaitGroup
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		srcSchema, srcErr = srcConn.ExtractSchema(dbName)
	}()
	go func() {
		defer wg.Done()
		tgtSchema, tgtErr = tgtConn.ExtractSchema(dbName)
	}()
	wg.Wait()

	if srcErr != nil {
		return fmt.Errorf("extracting source schema: %w", srcErr)
	}
	if tgtErr != nil {
		return fmt.Errorf("extracting target schema: %w", tgtErr)
	}

	// Override schema .Name with the configured display labels
	srcSchema.Name = cfg.Source.Name
	tgtSchema.Name = cfg.Target.Name

	// 10. Diff
	result := diff.Compare(srcSchema, tgtSchema, cfg.Ignore)

	// 11. Output
	switch cfg.Output {
	case "json":
		if err := output.WriteJSON(os.Stdout, result); err != nil {
			return err
		}
	default:
		if err := output.WriteTerminal(os.Stdout, result); err != nil {
			return err
		}
	}

	// 12. Generate migration SQL if requested
	if cfg.Migrate.Enabled {
		sql, err := migrate.Generate(result, cfg.Migrate.Direction, dialect)
		if err != nil {
			return fmt.Errorf("generating migration SQL: %w", err)
		}
		if err := os.WriteFile(cfg.Migrate.OutputFile, []byte(sql), 0644); err != nil {
			return fmt.Errorf("writing migration file %q: %w", cfg.Migrate.OutputFile, err)
		}
		fmt.Fprintf(os.Stderr, "Migration SQL written to: %s\n", cfg.Migrate.OutputFile)
	}

	return nil
}

// extractDBName parses the database name from the path segment of a DSN.
// Handles: mysql://user:pass@host:3306/dbname and postgres://... forms.
func extractDBName(dsn string) string {
	// Strip scheme
	for _, prefix := range []string{"mysql://", "postgres://", "postgresql://"} {
		if len(dsn) > len(prefix) && dsn[:len(prefix)] == prefix {
			dsn = dsn[len(prefix):]
			break
		}
	}
	// Find the '/' that separates host:port from path
	slash := -1
	for i, c := range dsn {
		if c == '/' {
			slash = i
			break
		}
	}
	if slash < 0 {
		return dsn
	}
	path := dsn[slash+1:]
	// Strip query string
	for i := 0; i < len(path); i++ {
		if path[i] == '?' {
			path = path[:i]
			break
		}
	}
	return path
}
