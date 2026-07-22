-- pgEdge Natural Language Agent
-- Copyright (c) 2025 - 2026, pgEdge, Inc.
-- This software is released under The PostgreSQL License
--
-- Metadata loader query. Produces one row per (schema, table, column),
-- left-joined so that tables without columns still appear (with all
-- column_info fields NULL). The Go side groups rows by (schema, table)
-- and treats NULL column fields as "table has no columns".
WITH table_comments AS (
    SELECT
        n.nspname AS schema_name,
        c.relname AS table_name,
        CASE c.relkind
            WHEN 'r' THEN 'TABLE'
            WHEN 'p' THEN 'PARTITIONED TABLE'
            WHEN 'v' THEN 'VIEW'
            WHEN 'm' THEN 'MATERIALIZED VIEW'
        END AS table_type,
        obj_description(c.oid) AS table_description,
        c.relkind = 'p' AS is_partitioned,
        EXISTS (SELECT 1 FROM pg_inherits WHERE inhrelid = c.oid) AS is_partition
    FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE c.relkind IN ('r', 'p', 'v', 'm')
        AND n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
    ORDER BY n.nspname, c.relname
),
column_info AS (
    SELECT
        n.nspname AS schema_name,
        c.relname AS table_name,
        a.attname AS column_name,
        pg_catalog.format_type(a.atttypid, a.atttypmod) AS data_type,
        CASE WHEN a.attnotnull THEN 'NO' ELSE 'YES' END AS is_nullable,
        col_description(c.oid, a.attnum) AS column_description,
        t.typname AS type_name,
        a.atttypmod AS type_modifier,
        a.attnum AS column_num,
        c.oid AS table_oid,
        a.attidentity::text AS identity_type
    FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
    JOIN pg_attribute a ON a.attrelid = c.oid
    JOIN pg_type t ON t.oid = a.atttypid
    WHERE c.relkind IN ('r', 'p', 'v', 'm')
        AND n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
        AND a.attnum > 0
        AND NOT a.attisdropped
    ORDER BY n.nspname, c.relname, a.attnum
),
pk_columns AS (
    SELECT
        n.nspname AS schema_name,
        c.relname AS table_name,
        a.attname AS column_name
    FROM pg_constraint con
    JOIN pg_class c ON c.oid = con.conrelid
    JOIN pg_namespace n ON n.oid = c.relnamespace
    JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = ANY(con.conkey)
    WHERE con.contype = 'p'
),
unique_columns AS (
    SELECT DISTINCT
        n.nspname AS schema_name,
        c.relname AS table_name,
        a.attname AS column_name
    FROM pg_constraint con
    JOIN pg_class c ON c.oid = con.conrelid
    JOIN pg_namespace n ON n.oid = c.relnamespace
    JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = ANY(con.conkey)
    WHERE con.contype = 'u'
),
-- A single column can participate in more than one foreign-key
-- constraint (e.g. the same column referenced by two different parent
-- tables), so aggregate every reference for a column into one ordered,
-- de-duplicated array. This yields exactly one row per
-- (schema, table, column); without the GROUP BY the downstream LEFT
-- JOIN would multiply the per-column rows and emit duplicate columns
-- (issue #171).
fk_columns AS (
    SELECT
        n.nspname AS schema_name,
        c.relname AS table_name,
        a.attname AS column_name,
        array_agg(DISTINCT fn.nspname || '.' || fc.relname || '.' || fa.attname
                  ORDER BY fn.nspname || '.' || fc.relname || '.' || fa.attname) AS fk_references
    FROM pg_constraint con
    JOIN pg_class c ON c.oid = con.conrelid
    JOIN pg_namespace n ON n.oid = c.relnamespace
    JOIN pg_class fc ON fc.oid = con.confrelid
    JOIN pg_namespace fn ON fn.oid = fc.relnamespace
    JOIN LATERAL unnest(con.conkey, con.confkey) WITH ORDINALITY AS cols(col_num, ref_num, ord) ON true
    JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = cols.col_num
    JOIN pg_attribute fa ON fa.attrelid = fc.oid AND fa.attnum = cols.ref_num
    WHERE con.contype = 'f'
    GROUP BY n.nspname, c.relname, a.attname
),
indexed_columns AS (
    SELECT DISTINCT
        n.nspname AS schema_name,
        c.relname AS table_name,
        a.attname AS column_name
    FROM pg_index i
    JOIN pg_class c ON c.oid = i.indrelid
    JOIN pg_namespace n ON n.oid = c.relnamespace
    JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = ANY(i.indkey)
    WHERE n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
),
column_defaults AS (
    SELECT
        n.nspname AS schema_name,
        c.relname AS table_name,
        a.attname AS column_name,
        pg_get_expr(d.adbin, d.adrelid) AS default_value
    FROM pg_attrdef d
    JOIN pg_class c ON c.oid = d.adrelid
    JOIN pg_namespace n ON n.oid = c.relnamespace
    JOIN pg_attribute a ON a.attrelid = d.adrelid AND a.attnum = d.adnum
    WHERE n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
        AND NOT a.attisdropped
)
SELECT
    tc.schema_name,
    tc.table_name,
    tc.table_type,
    COALESCE(tc.table_description, '') AS table_description,
    tc.is_partitioned,
    tc.is_partition,
    ci.column_name,
    ci.data_type,
    ci.is_nullable,
    COALESCE(ci.column_description, '') AS column_description,
    ci.type_name,
    ci.type_modifier,
    CASE WHEN pk.column_name IS NOT NULL THEN true ELSE false END AS is_primary_key,
    CASE WHEN uq.column_name IS NOT NULL THEN true ELSE false END AS is_unique,
    COALESCE(fk.fk_references, ARRAY[]::text[]) AS fk_references,
    CASE WHEN ix.column_name IS NOT NULL THEN true ELSE false END AS is_indexed,
    COALESCE(ci.identity_type, '') AS identity_type,
    COALESCE(cd.default_value, '') AS default_value
FROM table_comments tc
LEFT JOIN column_info ci ON tc.schema_name = ci.schema_name AND tc.table_name = ci.table_name
LEFT JOIN pk_columns pk ON ci.schema_name = pk.schema_name AND ci.table_name = pk.table_name AND ci.column_name = pk.column_name
LEFT JOIN unique_columns uq ON ci.schema_name = uq.schema_name AND ci.table_name = uq.table_name AND ci.column_name = uq.column_name
LEFT JOIN fk_columns fk ON ci.schema_name = fk.schema_name AND ci.table_name = fk.table_name AND ci.column_name = fk.column_name
LEFT JOIN indexed_columns ix ON ci.schema_name = ix.schema_name AND ci.table_name = ix.table_name AND ci.column_name = ix.column_name
LEFT JOIN column_defaults cd ON ci.schema_name = cd.schema_name AND ci.table_name = cd.table_name AND ci.column_name = cd.column_name
ORDER BY tc.schema_name, tc.table_name, ci.column_name
