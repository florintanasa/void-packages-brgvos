package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/valentin-kaiser/go-dbase/dbase"
)

func main() {
	dbfPath := flag.String("f", "", "Path to the input DBF file (Required)")
	tableName := flag.String("t", "", "Target PostgreSQL table name (Optional, defaults to filename)")
	flag.Parse()

	if *dbfPath == "" {
		fmt.Fprintln(os.Stderr, "Error: -f (dbf file path) is required.")
		flag.Usage()
		os.Exit(1)
	}

	// Open the DBF table using the clean filesystem configuration
	db, err := dbase.OpenTable(&dbase.Config{
		Filename:          *dbfPath,
		TrimSpaces:        true,
		InterpretCodePage: true, // Automatically handles code page mapping to UTF-8
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening DBF table: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Fallback for table name based on filename if not supplied
	if *tableName == "" {
		parts := strings.Split(strings.ReplaceAll(*dbfPath, "\\", "/"), "/")
		filename := parts[len(parts)-1]
		if dot := strings.LastIndex(filename, "."); dot != -1 {
			filename = filename[:dot]
		}
		*tableName = strings.ToLower(filename)
	}

	// 1. Generate SQL Header Block
	fmt.Println("BEGIN;")
	fmt.Fprintf(os.Stdout, dropTable, *tableName)
	fmt.Fprintf(os.Stdout, createTable, *tableName)

	columns := db.Columns()
	for i, col := range columns {
		colType := mapDbaseToPostgres(col.Type())
		colName := strings.ToLower(col.Name())
		fmt.Fprintf(os.Stdout, "    %s %s", colName, colType)
		if i < len(columns)-1 {
			fmt.Println(",")
		}
	}
	fmt.Println("\n);")

	// 2. Stream Data via COPY protocol
	fmt.Fprintf(os.Stdout, "\\COPY %s FROM STDIN WITH NULL AS '\\N';\n", *tableName)

	for !db.EOF() {
		row, err := db.Next()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: skipping corrupted row: %v\n", err)
			continue
		}

		if row.Deleted {
			continue
		}

		var rowValues []string
		for _, col := range columns {
			val, err := row.ValueByName(col.Name())
			if err != nil || val == nil {
				rowValues = append(rowValues, "\\N")
				continue
			}

			// Sanitize values to safely fit text streaming standards
			switch v := val.(type) {
			case string:
				s := strings.ReplaceAll(v, "\t", " ")
				s = strings.ReplaceAll(s, "\n", " ")
				s = strings.ReplaceAll(s, "\r", " ")
				if s == "" {
					rowValues = append(rowValues, "\\N")
				} else {
					rowValues = append(rowValues, s)
				}
			case time.Time:
				rowValues = append(rowValues, v.Format("2006-01-02 15:04:05"))
			default:
				rowValues = append(rowValues, fmt.Sprintf("%v", v))
			}
		}
		fmt.Println(strings.Join(rowValues, "\t"))
	}

	fmt.Println("\\.")
	fmt.Println("COMMIT;")
}

const (
	dropTable   = "DROP TABLE IF EXISTS %s;\n"
	createTable = "CREATE TABLE %s (\n"
)

// Map engine native bytes to precise Postgres equivalents
func mapDbaseToPostgres(t string) string {
	switch t {
	case "B", "F", "N": // Numeric / Float / Double
		return "NUMERIC"
	case "I": // Integer
		return "INTEGER"
	case "L": // Logical (Boolean)
		return "BOOLEAN"
	case "D": // Date
		return "DATE"
	case "T": // DateTime / Timestamp
		return "TIMESTAMP"
	case "M": // Memo fields
		return "TEXT"
	case "Y": // Currency
		return "MONEY"
	default:
		return "VARCHAR"
	}
}
