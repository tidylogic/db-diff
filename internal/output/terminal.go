package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"

	"github.com/tidylogic/db-diff/internal/diff"
	"github.com/tidylogic/db-diff/internal/schema"
)

var (
	colorAdd    = color.New(color.FgGreen, color.Bold)
	colorRemove = color.New(color.FgRed, color.Bold)
	colorModify = color.New(color.FgYellow, color.Bold)
	colorHeader = color.New(color.FgCyan, color.Bold)
	colorDim    = color.New(color.Faint)
)

// WriteTerminal writes a human-friendly, ANSI-colored diff to w.
func WriteTerminal(w io.Writer, result *diff.DiffResult) error {
	separator := strings.Repeat("─", 70)

	// Header
	fmt.Fprintf(w, "\n")
	colorHeader.Fprintf(w, "Schema Diff: %s  →  %s\n", result.SourceName, result.TargetName)
	colorDim.Fprintf(w, "%s\n\n", separator)

	if result.Identical {
		color.New(color.FgGreen).Fprintf(w, "  ✓ Schemas are identical — no differences found.\n\n")
		return nil
	}

	// Tables
	for _, td := range result.Tables {
		switch td.Change {
		case diff.Added:
			colorAdd.Fprintf(w, "+ TABLE %s\n", td.Name)
		case diff.Removed:
			colorRemove.Fprintf(w, "- TABLE %s\n", td.Name)
		case diff.Modified:
			colorModify.Fprintf(w, "~ TABLE %s\n", td.Name)
			printColumnDiffs(w, td.Columns)
			printIndexDiffs(w, td.Indexes)
			printConstraintDiffs(w, td.Constraints)
		}
		fmt.Fprintln(w)
	}

	// Views
	for _, vd := range result.Views {
		switch vd.Change {
		case diff.Added:
			colorAdd.Fprintf(w, "+ VIEW %s\n", vd.Name)
		case diff.Removed:
			colorRemove.Fprintf(w, "- VIEW %s\n", vd.Name)
		case diff.Modified:
			colorModify.Fprintf(w, "~ VIEW %s\n", vd.Name)
		}
		fmt.Fprintln(w)
	}

	// Summary
	colorDim.Fprintf(w, "%s\n", separator)
	printSummary(w, result)
	fmt.Fprintln(w)

	return nil
}

func printColumnDiffs(w io.Writer, cols []diff.ColumnDiff) {
	for _, cd := range cols {
		switch cd.Change {
		case diff.Added:
			colorAdd.Fprintf(w, "  + COLUMN %-30s %s\n", cd.Name, columnSpec(cd.Target))
		case diff.Removed:
			colorRemove.Fprintf(w, "  - COLUMN %-30s %s\n", cd.Name, columnSpec(cd.Source))
		case diff.Modified:
			colorModify.Fprintf(w, "  ~ COLUMN %s\n", cd.Name)
			printFieldChange(w, "type", cd.Source.RawType, cd.Target.RawType)
			if cd.Source.Nullable != cd.Target.Nullable {
				printFieldChange(w, "nullable", boolStr(cd.Source.Nullable), boolStr(cd.Target.Nullable))
			}
			if !ptrStrEq(cd.Source.Default, cd.Target.Default) {
				printFieldChange(w, "default", ptrStrVal(cd.Source.Default), ptrStrVal(cd.Target.Default))
			}
			if cd.Source.Comment != cd.Target.Comment {
				printFieldChange(w, "comment", cd.Source.Comment, cd.Target.Comment)
			}
		}
	}
}

func printIndexDiffs(w io.Writer, idxs []diff.IndexDiff) {
	for _, id := range idxs {
		switch id.Change {
		case diff.Added:
			colorAdd.Fprintf(w, "  + INDEX %-30s (%s)\n", id.Name, strings.Join(id.Target.Columns, ", "))
		case diff.Removed:
			colorRemove.Fprintf(w, "  - INDEX %-30s (%s)\n", id.Name, strings.Join(id.Source.Columns, ", "))
		case diff.Modified:
			colorModify.Fprintf(w, "  ~ INDEX %s\n", id.Name)
			printFieldChange(w, "columns",
				strings.Join(id.Source.Columns, ", "),
				strings.Join(id.Target.Columns, ", "))
		}
	}
}

func printConstraintDiffs(w io.Writer, cons []diff.ConstraintDiff) {
	for _, cd := range cons {
		switch cd.Change {
		case diff.Added:
			colorAdd.Fprintf(w, "  + CONSTRAINT %-26s %s (%s)\n",
				cd.Name, cd.Target.Type, strings.Join(cd.Target.Columns, ", "))
		case diff.Removed:
			colorRemove.Fprintf(w, "  - CONSTRAINT %-26s %s (%s)\n",
				cd.Name, cd.Source.Type, strings.Join(cd.Source.Columns, ", "))
		case diff.Modified:
			colorModify.Fprintf(w, "  ~ CONSTRAINT %s\n", cd.Name)
		}
	}
}

func printFieldChange(w io.Writer, field, src, tgt string) {
	fmt.Fprintf(w, "      %-12s ", field+":")
	colorRemove.Fprintf(w, "%s", src)
	fmt.Fprintf(w, " → ")
	colorAdd.Fprintf(w, "%s\n", tgt)
}

func printSummary(w io.Writer, result *diff.DiffResult) {
	added, removed, modified := 0, 0, 0
	for _, td := range result.Tables {
		switch td.Change {
		case diff.Added:
			added++
		case diff.Removed:
			removed++
		case diff.Modified:
			modified++
		}
	}
	viewAdded, viewRemoved, viewModified := 0, 0, 0
	for _, vd := range result.Views {
		switch vd.Change {
		case diff.Added:
			viewAdded++
		case diff.Removed:
			viewRemoved++
		case diff.Modified:
			viewModified++
		}
	}

	parts := []string{}
	if modified > 0 {
		parts = append(parts, colorModify.Sprintf("%d modified", modified))
	}
	if added > 0 {
		parts = append(parts, colorAdd.Sprintf("%d added", added))
	}
	if removed > 0 {
		parts = append(parts, colorRemove.Sprintf("%d removed", removed))
	}
	if viewAdded+viewRemoved+viewModified > 0 {
		parts = append(parts, fmt.Sprintf("%d view change(s)", viewAdded+viewRemoved+viewModified))
	}

	fmt.Fprintf(w, "Summary (tables): %s\n", strings.Join(parts, ", "))
}

// --- helpers ---

func columnSpec(c *schema.Column) string {
	if c == nil {
		return ""
	}
	s := c.RawType
	if c.Nullable {
		s += " NULL"
	} else {
		s += " NOT NULL"
	}
	return s
}

func boolStr(b bool) string {
	if b {
		return "YES"
	}
	return "NO"
}

func ptrStrEq(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func ptrStrVal(s *string) string {
	if s == nil {
		return "(none)"
	}
	if *s == "" {
		return `""`
	}
	return *s
}
