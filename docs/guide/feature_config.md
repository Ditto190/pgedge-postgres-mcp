# Enabling or Disabling Built-in Features

You can selectively enable or disable built-in tools, resources, and prompts; all features are enabled by default. When a feature is disabled:

    - It is not advertised to the LLM in list operations
    - Attempts to use it return an error message

Within the `builtins` section of the configuration file, you can indicate if you would like the feature to be enabled (`true`) or disabled (`false`):

```yaml
builtins:
  tools:
    query_database: true        # Execute SQL queries
    get_schema_info: true       # Get schema information
    similarity_search: false    # Disable vector similarity search
    execute_explain: true       # Execute EXPLAIN queries
    generate_embedding: false   # Disable embedding generation
    search_knowledgebase: true  # Search documentation knowledgebase
    count_rows: true            # Count table rows
    llm_connection_selection: false  # LLM database switching (disabled by default)
  resources:
    system_info: true           # pg://system_info
  prompts:
    explore_database: true      # explore-database prompt
    setup_semantic_search: true # setup-semantic-search prompt
    diagnose_query_issue: true  # diagnose-query-issue prompt
    design_schema: true         # design-schema prompt
```

!!! Notes

    - The `read_resource` tool is always enabled as it is required for listing resources.
    - Features can also be disabled by other configuration settings (e.g., `search_knowledgebase` requires `knowledgebase.enabled: true`).
    - The `llm_connection_selection` option is disabled by default for security.
      When enabled, it provides `list_database_connections` and
      `select_database_connection` tools that allow the LLM to switch between
      configured databases. Use `allow_llm_switching: false` on individual
      database connections to exclude them from LLM switching.

## Using Environment Variables

Each built-in feature can also be toggled via an environment variable;
this is convenient for containerized deployments where editing the
configuration file is awkward. Environment variables take precedence
over the configuration file.

Accepted truthy values are `true`, `1`, and `yes` (case-insensitive);
any other non-empty value is treated as false. If the variable is unset,
the configuration file value (or the built-in default) is used.

| Feature | Environment Variable |
|---------|---------------------|
| `query_database` tool | `PGEDGE_BUILTIN_TOOL_QUERY_DATABASE` |
| `get_schema_info` tool | `PGEDGE_BUILTIN_TOOL_GET_SCHEMA_INFO` |
| `similarity_search` tool | `PGEDGE_BUILTIN_TOOL_SIMILARITY_SEARCH` |
| `execute_explain` tool | `PGEDGE_BUILTIN_TOOL_EXECUTE_EXPLAIN` |
| `generate_embedding` tool | `PGEDGE_BUILTIN_TOOL_GENERATE_EMBEDDING` |
| `search_knowledgebase` tool | `PGEDGE_BUILTIN_TOOL_SEARCH_KNOWLEDGEBASE` |
| `count_rows` tool | `PGEDGE_BUILTIN_TOOL_COUNT_ROWS` |
| `llm_connection_selection` tools | `PGEDGE_BUILTIN_TOOL_LLM_CONNECTION_SELECTION` |
| `pg://system_info` resource | `PGEDGE_BUILTIN_RESOURCE_SYSTEM_INFO` |
| `explore-database` prompt | `PGEDGE_BUILTIN_PROMPT_EXPLORE_DATABASE` |
| `setup-semantic-search` prompt | `PGEDGE_BUILTIN_PROMPT_SETUP_SEMANTIC_SEARCH` |
| `diagnose-query-issue` prompt | `PGEDGE_BUILTIN_PROMPT_DIAGNOSE_QUERY_ISSUE` |
| `design-schema` prompt | `PGEDGE_BUILTIN_PROMPT_DESIGN_SCHEMA` |

In the following example, the `pg://system_info` resource is disabled
for a single container run:

```bash
PGEDGE_BUILTIN_RESOURCE_SYSTEM_INFO=false ./bin/pgedge-postgres-mcp
```
