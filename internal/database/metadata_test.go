/*-------------------------------------------------------------------------
 *
 * pgEdge Natural Language Agent
 *
 * Copyright (c) 2025 - 2026, pgEdge, Inc.
 * This software is released under The PostgreSQL License
 *
 *-------------------------------------------------------------------------
 */

package database

import (
	"database/sql"
	"testing"
)

// validColumn is a small helper for constructing scanned rows whose
// column_info fields are present (non-NULL).
func validColumn(schema, table, column, dataType, typeName string) metadataRow {
	return metadataRow{
		SchemaName:   schema,
		TableName:    table,
		TableType:    "TABLE",
		ColumnName:   sql.NullString{String: column, Valid: true},
		DataType:     sql.NullString{String: dataType, Valid: true},
		IsNullable:   sql.NullString{String: "YES", Valid: true},
		TypeName:     sql.NullString{String: typeName, Valid: true},
		TypeModifier: sql.NullInt32{Int32: -1, Valid: true},
	}
}

// TestBuildTableInfo_EmptyInput verifies that an empty input slice
// produces empty (but non-nil) result maps so downstream callers can
// safely range over them.
func TestBuildTableInfo_EmptyInput(t *testing.T) {
	metadata, schemas, columnCount := buildTableInfo(nil)

	if metadata == nil {
		t.Errorf("expected non-nil metadata map for empty input, got nil")
	}
	if schemas == nil {
		t.Errorf("expected non-nil schemas map for empty input, got nil")
	}
	if len(metadata) != 0 {
		t.Errorf("expected 0 metadata entries, got %d", len(metadata))
	}
	if len(schemas) != 0 {
		t.Errorf("expected 0 schemas, got %d", len(schemas))
	}
	if columnCount != 0 {
		t.Errorf("expected columnCount == 0, got %d", columnCount)
	}
}

// TestBuildTableInfo_TableWithNoColumns is the pure-function regression
// test for issue #126. A row whose column_info fields are NULL (the
// LEFT JOIN miss for a zero-column table) must still produce a
// TableInfo entry, but with zero columns and no columnCount increment.
func TestBuildTableInfo_TableWithNoColumns(t *testing.T) {
	row := metadataRow{
		SchemaName: "public",
		TableName:  "empty",
		TableType:  "TABLE",
		// ColumnName / DataType / IsNullable / TypeName intentionally
		// left as zero-value (Valid == false), matching what pgx
		// produces for NULL columns.
	}

	metadata, schemas, columnCount := buildTableInfo([]metadataRow{row})

	table, ok := metadata["public.empty"]
	if !ok {
		t.Fatalf("expected metadata to contain public.empty, got keys %v", keysOf(metadata))
	}
	if len(table.Columns) != 0 {
		t.Errorf("expected zero columns for empty table, got %d", len(table.Columns))
	}
	if columnCount != 0 {
		t.Errorf("expected columnCount == 0 for empty table, got %d", columnCount)
	}
	if !schemas["public"] {
		t.Errorf("expected schema 'public' to be recorded")
	}
}

// TestBuildTableInfo_GroupsRowsByTable verifies that multiple rows
// sharing the same (schema, table) key collapse into a single
// TableInfo with all their columns appended.
func TestBuildTableInfo_GroupsRowsByTable(t *testing.T) {
	rows := []metadataRow{
		validColumn("public", "users", "id", "integer", "int4"),
		validColumn("public", "users", "name", "text", "text"),
		validColumn("public", "users", "email", "text", "text"),
	}

	metadata, _, columnCount := buildTableInfo(rows)

	if len(metadata) != 1 {
		t.Fatalf("expected 1 table, got %d", len(metadata))
	}
	table := metadata["public.users"]
	if len(table.Columns) != 3 {
		t.Errorf("expected 3 columns, got %d", len(table.Columns))
	}
	if columnCount != 3 {
		t.Errorf("expected columnCount == 3, got %d", columnCount)
	}
}

// TestBuildTableInfo_MultipleSchemas verifies that rows from different
// schemas produce distinct entries and that the schemas set captures
// each one.
func TestBuildTableInfo_MultipleSchemas(t *testing.T) {
	rows := []metadataRow{
		validColumn("public", "users", "id", "integer", "int4"),
		validColumn("auth", "tokens", "id", "uuid", "uuid"),
		validColumn("auth", "tokens", "value", "text", "text"),
	}

	metadata, schemas, columnCount := buildTableInfo(rows)

	if len(metadata) != 2 {
		t.Errorf("expected 2 tables, got %d", len(metadata))
	}
	if !schemas["public"] || !schemas["auth"] {
		t.Errorf("expected schemas {public, auth}, got %v", keysOfBool(schemas))
	}
	if columnCount != 3 {
		t.Errorf("expected columnCount == 3, got %d", columnCount)
	}
}

// TestBuildTableInfo_VectorColumn verifies that a pgvector column is
// flagged and its dimension count is parsed out of the data_type.
func TestBuildTableInfo_VectorColumn(t *testing.T) {
	row := validColumn("public", "embeddings", "embedding", "vector(1536)", "vector")

	metadata, _, _ := buildTableInfo([]metadataRow{row})

	col := metadata["public.embeddings"].Columns[0]
	if !col.IsVectorColumn {
		t.Errorf("expected IsVectorColumn == true")
	}
	if col.VectorDimensions != 1536 {
		t.Errorf("expected VectorDimensions == 1536, got %d", col.VectorDimensions)
	}
}

// TestBuildTableInfo_VectorWithoutDimensions verifies that a vector
// column whose data_type does not include a parenthesised dimension
// count (older pgvector or unsized columns) is still flagged as a
// vector but has dimensions == 0.
func TestBuildTableInfo_VectorWithoutDimensions(t *testing.T) {
	row := validColumn("public", "embeddings", "embedding", "vector", "vector")

	metadata, _, _ := buildTableInfo([]metadataRow{row})

	col := metadata["public.embeddings"].Columns[0]
	if !col.IsVectorColumn {
		t.Errorf("expected IsVectorColumn == true for typename 'vector' without dim")
	}
	if col.VectorDimensions != 0 {
		t.Errorf("expected VectorDimensions == 0, got %d", col.VectorDimensions)
	}
}

// TestBuildTableInfo_NonVectorColumnNotFlagged verifies that ordinary
// types are not mis-flagged as vector columns.
func TestBuildTableInfo_NonVectorColumnNotFlagged(t *testing.T) {
	row := validColumn("public", "users", "name", "text", "text")

	metadata, _, _ := buildTableInfo([]metadataRow{row})

	col := metadata["public.users"].Columns[0]
	if col.IsVectorColumn {
		t.Errorf("expected IsVectorColumn == false for text column")
	}
	if col.VectorDimensions != 0 {
		t.Errorf("expected VectorDimensions == 0 for text column, got %d", col.VectorDimensions)
	}
}

// TestBuildTableInfo_TableLevelAttributesFromFirstRow verifies that
// table-level attributes (table type, description, partitioning flags)
// are taken from the first row encountered for a given table.
func TestBuildTableInfo_TableLevelAttributesFromFirstRow(t *testing.T) {
	rows := []metadataRow{
		{
			SchemaName:    "public",
			TableName:     "orders",
			TableType:     "PARTITIONED TABLE",
			TableDesc:     "Orders parent partition",
			IsPartitioned: true,
			IsPartition:   false,
			ColumnName:    sql.NullString{String: "id", Valid: true},
			DataType:      sql.NullString{String: "integer", Valid: true},
			IsNullable:    sql.NullString{String: "NO", Valid: true},
			TypeName:      sql.NullString{String: "int4", Valid: true},
		},
		{
			SchemaName:    "public",
			TableName:     "orders",
			TableType:     "PARTITIONED TABLE",
			TableDesc:     "Orders parent partition",
			IsPartitioned: true,
			IsPartition:   false,
			ColumnName:    sql.NullString{String: "amount", Valid: true},
			DataType:      sql.NullString{String: "numeric", Valid: true},
			IsNullable:    sql.NullString{String: "YES", Valid: true},
			TypeName:      sql.NullString{String: "numeric", Valid: true},
		},
	}

	metadata, _, _ := buildTableInfo(rows)
	table := metadata["public.orders"]

	if table.TableType != "PARTITIONED TABLE" {
		t.Errorf("expected TableType 'PARTITIONED TABLE', got %q", table.TableType)
	}
	if table.Description != "Orders parent partition" {
		t.Errorf("expected description preserved, got %q", table.Description)
	}
	if !table.IsPartitioned {
		t.Errorf("expected IsPartitioned == true")
	}
	if len(table.Columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(table.Columns))
	}
}

// TestBuildTableInfo_PreservesAllColumnAttributes verifies that
// constraint flags, identity info, default expressions, descriptions
// and FK references are all carried over from the scanned row to
// ColumnInfo.
func TestBuildTableInfo_PreservesAllColumnAttributes(t *testing.T) {
	row := metadataRow{
		SchemaName:   "public",
		TableName:    "users",
		TableType:    "TABLE",
		ColumnName:   sql.NullString{String: "id", Valid: true},
		DataType:     sql.NullString{String: "integer", Valid: true},
		IsNullable:   sql.NullString{String: "NO", Valid: true},
		ColumnDesc:   "primary identifier",
		TypeName:     sql.NullString{String: "int4", Valid: true},
		TypeModifier: sql.NullInt32{Int32: -1, Valid: true},
		IsPrimaryKey: true,
		IsUnique:     true,
		FKReference:  "auth.tokens.user_id",
		IsIndexed:    true,
		IdentityType: "a",
		DefaultValue: "nextval('users_id_seq'::regclass)",
	}
	want := ColumnInfo{
		ColumnName:       "id",
		DataType:         "integer",
		IsNullable:       "NO",
		Description:      "primary identifier",
		IsPrimaryKey:     true,
		IsUnique:         true,
		ForeignKeyRef:    "auth.tokens.user_id",
		IsIndexed:        true,
		IsIdentity:       "a",
		DefaultValue:     "nextval('users_id_seq'::regclass)",
		IsVectorColumn:   false,
		VectorDimensions: 0,
	}

	metadata, _, _ := buildTableInfo([]metadataRow{row})
	got := metadata["public.users"].Columns[0]

	if got != want {
		t.Errorf("column attributes not preserved:\n got:  %+v\n want: %+v", got, want)
	}
}

// TestBuildTableInfo_EmptyColumnNameSkipped covers a row whose
// ColumnName is Valid but empty (e.g., a hypothetical edge case where
// the LEFT JOIN produces ""). It should be treated like NULL: the
// table appears but the column is not added.
func TestBuildTableInfo_EmptyColumnNameSkipped(t *testing.T) {
	row := metadataRow{
		SchemaName: "public",
		TableName:  "weird",
		TableType:  "TABLE",
		ColumnName: sql.NullString{String: "", Valid: true},
	}

	metadata, _, columnCount := buildTableInfo([]metadataRow{row})
	table, ok := metadata["public.weird"]
	if !ok {
		t.Fatalf("expected public.weird in metadata")
	}
	if len(table.Columns) != 0 {
		t.Errorf("expected 0 columns when ColumnName is empty, got %d", len(table.Columns))
	}
	if columnCount != 0 {
		t.Errorf("expected columnCount == 0, got %d", columnCount)
	}
}

func keysOf(m map[string]TableInfo) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func keysOfBool(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
