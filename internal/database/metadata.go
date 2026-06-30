/*-------------------------------------------------------------------------
 *
 * pgEdge Natural Language Agent
 *
 * Copyright (c) 2025 - 2026, pgEdge, Inc.
 * This software is released under The PostgreSQL License
 *
 *-------------------------------------------------------------------------
 */

// Helpers for loading database schema metadata. The query lives in
// load_metadata.sql so it can be diffed and reviewed independently; the
// per-row scan and the (rows -> TableInfo map) transformation are split
// into their own functions so the latter is testable without a database.

package database

import (
	"database/sql"
	_ "embed"
	"regexp"
	"strconv"

	"github.com/jackc/pgx/v5"
)

//go:embed load_metadata.sql
var loadMetadataSQL string

// vectorDimsRe extracts the dimension count from a pgvector type
// descriptor like "vector(1536)".
var vectorDimsRe = regexp.MustCompile(`vector\((\d+)\)`)

// metadataRow is one scanned row of loadMetadataSQL. Columns originating
// from the LEFT JOIN against column_info are nullable because a table
// with zero columns still produces a row with all column_info fields
// NULL (see issue #126).
type metadataRow struct {
	SchemaName    string
	TableName     string
	TableType     string
	TableDesc     string
	IsPartitioned bool
	IsPartition   bool
	ColumnName    sql.NullString
	DataType      sql.NullString
	IsNullable    sql.NullString
	ColumnDesc    string
	TypeName      sql.NullString
	TypeModifier  sql.NullInt32
	IsPrimaryKey  bool
	IsUnique      bool
	FKReference   string
	IsIndexed     bool
	IdentityType  string
	DefaultValue  string
}

// scanMetadataRow scans one row of loadMetadataSQL into a metadataRow.
// The order of the Scan arguments must match the SELECT list of the
// query exactly.
func scanMetadataRow(rows pgx.Rows) (metadataRow, error) {
	var r metadataRow
	err := rows.Scan(
		&r.SchemaName,
		&r.TableName,
		&r.TableType,
		&r.TableDesc,
		&r.IsPartitioned,
		&r.IsPartition,
		&r.ColumnName,
		&r.DataType,
		&r.IsNullable,
		&r.ColumnDesc,
		&r.TypeName,
		&r.TypeModifier,
		&r.IsPrimaryKey,
		&r.IsUnique,
		&r.FKReference,
		&r.IsIndexed,
		&r.IdentityType,
		&r.DefaultValue,
	)
	return r, err
}

// vectorColumnInfo returns whether the row describes a pgvector column
// and, if so, the parsed dimension count. A typename of "vector" but a
// data_type without a parenthesised dimension count yields (true, 0).
func vectorColumnInfo(r metadataRow) (isVector bool, dimensions int) {
	if !r.TypeName.Valid || r.TypeName.String != "vector" {
		return false, 0
	}
	matches := vectorDimsRe.FindStringSubmatch(r.DataType.String)
	if len(matches) < 2 {
		return true, 0
	}
	dim, err := strconv.Atoi(matches[1])
	if err != nil {
		return true, 0
	}
	return true, dim
}

// columnInfoFromRow builds a ColumnInfo from a single metadataRow.
// Callers must check that the row actually carries column data
// (r.ColumnName.Valid and non-empty) before calling.
func columnInfoFromRow(r metadataRow) ColumnInfo {
	isVector, dimensions := vectorColumnInfo(r)
	return ColumnInfo{
		ColumnName:       r.ColumnName.String,
		DataType:         r.DataType.String,
		IsNullable:       r.IsNullable.String,
		Description:      r.ColumnDesc,
		IsPrimaryKey:     r.IsPrimaryKey,
		IsUnique:         r.IsUnique,
		ForeignKeyRef:    r.FKReference,
		IsIndexed:        r.IsIndexed,
		IsIdentity:       r.IdentityType,
		DefaultValue:     r.DefaultValue,
		IsVectorColumn:   isVector,
		VectorDimensions: dimensions,
	}
}

// tableInfoFromRow constructs a fresh TableInfo with table-level
// attributes from r and an empty Columns slice.
func tableInfoFromRow(r metadataRow) TableInfo {
	return TableInfo{
		SchemaName:    r.SchemaName,
		TableName:     r.TableName,
		TableType:     r.TableType,
		Description:   r.TableDesc,
		IsPartitioned: r.IsPartitioned,
		IsPartition:   r.IsPartition,
		Columns:       []ColumnInfo{},
	}
}

// metadataAccumulator groups metadata rows by (schema, table) into the
// metadata map as they arrive. Rows can be fed one at a time via add, so
// callers may stream straight from a result set without first buffering
// every row in a slice. The grouping logic lives here so the streaming
// path and the slice-based buildTableInfo helper stay in sync.
type metadataAccumulator struct {
	metadata    map[string]TableInfo
	schemas     map[string]bool
	columnCount int
}

// newMetadataAccumulator returns an accumulator with initialised maps.
func newMetadataAccumulator() *metadataAccumulator {
	return &metadataAccumulator{
		metadata: make(map[string]TableInfo),
		schemas:  make(map[string]bool),
	}
}

// add folds a single scanned row into the accumulated metadata. Rows with
// a NULL or empty column name (tables with no columns) contribute a
// TableInfo with Columns == [] but do not increment columnCount.
func (a *metadataAccumulator) add(r metadataRow) {
	key := r.SchemaName + "." + r.TableName
	a.schemas[r.SchemaName] = true

	table, exists := a.metadata[key]
	if !exists {
		table = tableInfoFromRow(r)
	}

	if r.ColumnName.Valid && r.ColumnName.String != "" {
		table.Columns = append(table.Columns, columnInfoFromRow(r))
		a.columnCount++
	}

	a.metadata[key] = table
}

// buildTableInfo groups scanned rows by (schema, table) and constructs
// the metadata map. The returned map is keyed "schema.table". schemas
// collects the distinct schema names seen. columnCount is the total
// number of column entries appended across all tables — rows with a
// NULL or empty column name (tables with no columns) contribute a
// TableInfo with Columns == [] but do not increment columnCount.
//
// This function is pure: no I/O, no time, no shared state. It is retained
// for the unit tests; the production load path streams rows through a
// metadataAccumulator instead of buffering them in a slice.
func buildTableInfo(rows []metadataRow) (metadata map[string]TableInfo, schemas map[string]bool, columnCount int) {
	acc := newMetadataAccumulator()
	for _, r := range rows {
		acc.add(r)
	}
	return acc.metadata, acc.schemas, acc.columnCount
}
