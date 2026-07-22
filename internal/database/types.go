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

// TableInfo contains information about a database table or view
type TableInfo struct {
	SchemaName    string
	TableName     string
	TableType     string // 'TABLE', 'PARTITIONED TABLE', 'VIEW', or 'MATERIALIZED VIEW'
	Description   string
	IsPartitioned bool // true for partitioned parent tables (relkind = 'p')
	IsPartition   bool // true for child partition tables
	Columns       []ColumnInfo
}

// ColumnInfo contains information about a database column
type ColumnInfo struct {
	ColumnName       string
	DataType         string
	IsNullable       string
	Description      string
	IsPrimaryKey     bool     // True if this column is part of the primary key
	IsUnique         bool     // True if this column has a unique constraint (excluding PK)
	ForeignKeyRefs   []string // References in format "schema.table.column"; one entry per FK the column participates in, empty if none
	IsIndexed        bool     // True if this column is part of any index
	IsIdentity       string   // Identity generation: "" (none), "a" (ALWAYS), "d" (BY DEFAULT)
	DefaultValue     string   // Default value expression if any, empty otherwise
	IsVectorColumn   bool     // True if this is a pgvector column
	VectorDimensions int      // Number of dimensions for vector columns (0 if not a vector)
	VectorType       string   // Underlying vector type: "vector", "halfvec", or "" if not a vector column
}
