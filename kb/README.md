# Knowledgebase Directory

This directory is used during Docker image builds to embed a pre-built
knowledgebase database (`kb.db`) in the server container image. The
MCP server consumes the database at runtime to power the
`search_knowledgebase` tool.

The kb-builder tool that produces `kb.db` lives in its own project:
[`pgedge-ai-kb`](https://github.com/pgEdge/pgedge-ai-kb).

## Usage

You can supply `kb.db` to a Docker build in two ways.

### Option 1: Place a local file

Build or download a `kb.db` file, place it at `kb/kb.db` in this
directory, then build the image normally:

```bash
docker build -f Dockerfile.server -t mcp-server:with-kb .
```

### Option 2: Download from a URL

Pass `KB_SOURCE` as a build argument. The Dockerfile downloads the
file during the build:

```bash
docker build -f Dockerfile.server \
    --build-arg KB_SOURCE=https://example.com/kb.db \
    -t mcp-server:with-kb .
```

A common source URL is the latest `kb.db` release artifact from the
[pgEdge AI Knowledgebase Builder
releases](https://github.com/pgEdge/pgedge-ai-kb/releases) page.

## Building Your Own Knowledgebase

To build a custom `kb.db` for the MCP server, follow the
documentation in the
[pgEdge AI Knowledgebase Builder](https://github.com/pgEdge/pgedge-ai-kb)
project. The Quick Start covers installation and a first build.

## Notes

- The `kb.db` file is not committed to version control (see
  `.gitignore`).

- If no `kb.db` file is present and no `KB_SOURCE` URL is supplied,
  the server image is built without a knowledgebase.

- See the [Knowledgebase guide](../docs/advanced/knowledgebase.md)
  for details on configuring the MCP server to load the file.
