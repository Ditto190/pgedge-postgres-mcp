# Try It Now -- One-Command Installer

These installer scripts let you get the pgEdge Postgres MCP
Server running with Claude Code or Claude Desktop in a single
command. The scripts are designed for quick evaluation and
demos; for production deployment, see the
[main project documentation](../../docs/index.md).

Two installer scripts are available:

- `install.sh` supports macOS and Linux systems.
- `install.ps1` supports Windows via PowerShell.

## Quick Start

Run one of the following commands to start the interactive
installer.

On macOS or Linux, run the following `curl` command to
download and execute the installer:

```bash
curl -fsSL https://raw.githubusercontent.com/pgEdge/pgedge-postgres-mcp/main/examples/claude-try-it/install.sh | bash
```

On Windows, run the following PowerShell command to download
and execute the installer:

```powershell
irm https://raw.githubusercontent.com/pgEdge/pgedge-postgres-mcp/main/examples/claude-try-it/install.ps1 | iex
```

The interactive installer guides you through database
selection and client configuration.

## What the Installer Does

Each installer performs the following steps in order:

1. Download the pgEdge MCP Server binary for your platform.
2. Connect to a database; you can use your own PostgreSQL
   instance or start a demo database with Docker and the
   Northwind sample dataset.
3. Configure Claude Code and Claude Desktop to use the
   MCP Server.

## Existing Install Detection

The installer detects previous installations automatically.
When the installer finds an existing installation, the
installer offers to update the binary to the latest version.
The installer also offers to reconfigure the database
connection on an existing installation.

## Command-Line Flags

You can pass flags to run the installer non-interactively.
Non-interactive mode works well with Claude Code, which can
run the installer on your behalf.

### Bash Flags (macOS and Linux)

The following table lists the flags for `install.sh`:

| Flag | Description |
|------|-------------|
| `--demo` | Start a Docker demo database with Northwind sample data. |
| `--detect` | Auto-connect to a detected PostgreSQL instance. |
| `--own-db` | Connect to your own database using the connection flags below. |
| `--install-docker` | Install Docker before starting the demo database. |
| `--db-host=HOST` | Set the database hostname. |
| `--db-port=PORT` | Set the database port. |
| `--db-name=DB` | Set the database name. |
| `--db-user=USER` | Set the database username. |
| `--db-pass=PASS` | Set the database password. |
| `--version=VERSION` | Install a specific release tag (for example, `v1.0.0`) instead of resolving the latest. See [Pinning a Version](#pinning-a-version). |

In the following example, the `--demo` flag starts a demo
database without prompting:

```bash
curl -fsSL https://raw.githubusercontent.com/pgEdge/pgedge-postgres-mcp/main/examples/claude-try-it/install.sh | bash -s -- --demo
```

In the following example, the `--detect` flag auto-connects
to a running PostgreSQL instance:

```bash
curl -fsSL https://raw.githubusercontent.com/pgEdge/pgedge-postgres-mcp/main/examples/claude-try-it/install.sh | bash -s -- --detect
```

### PowerShell Flags (Windows)

The following table lists the flags for `install.ps1`:

| Flag | Description |
|------|-------------|
| `-Demo` | Start a Docker demo database with Northwind sample data. |
| `-Detect` | Auto-connect to a detected PostgreSQL instance. |
| `-OwnDb` | Connect to your own database using the connection flags below. |
| `-InstallDocker` | Install Docker before starting the demo database. |
| `-DbHost` | Set the database hostname. |
| `-DbPort` | Set the database port. |
| `-DbName` | Set the database name. |
| `-DbUser` | Set the database username. |
| `-DbPass` | Set the database password. |
| `-Version` | Install a specific release tag (for example, `v1.0.0`) instead of resolving the latest. See [Pinning a Version](#pinning-a-version). |

In the following example, the `-Demo` flag starts a demo
database without prompting:

```powershell
.\install.ps1 -Demo
```

In the following example, the `-Detect` flag auto-connects
to an instance on a non-default port:

```powershell
.\install.ps1 -Detect -DbPort 5433
```

## Pinning a Version

By default, the installer resolves the latest stable release
(a `v*` tag that ships a binary for your platform). To install
a specific version instead, pin it with the `--version` flag
(`-Version` on Windows) or the `PGEDGE_MCP_VERSION` environment
variable. The flag takes precedence over the environment variable.

Pinning skips the latest-release lookup entirely, so it is also a
reliable fallback if release resolution ever fails.

On macOS and Linux:

```bash
# Flag — note it prefixes `bash`, after `-s --`:
curl -fsSL https://raw.githubusercontent.com/pgEdge/pgedge-postgres-mcp/main/examples/claude-try-it/install.sh | bash -s -- --version=v1.0.0

# Environment variable — must prefix `bash`, not `curl`:
curl -fsSL https://raw.githubusercontent.com/pgEdge/pgedge-postgres-mcp/main/examples/claude-try-it/install.sh | PGEDGE_MCP_VERSION=v1.0.0 bash
```

On Windows (PowerShell):

```powershell
# Flag (script saved locally):
.\install.ps1 -Version v1.0.0

# Environment variable (piped install):
$env:PGEDGE_MCP_VERSION = "v1.0.0"
irm https://raw.githubusercontent.com/pgEdge/pgedge-postgres-mcp/main/examples/claude-try-it/install.ps1 | iex
```

The value matches the GitHub release tag (for example, `v1.0.0`).
A leading `v` is optional. Browse available tags on the
[releases page](https://github.com/pgEdge/pgedge-postgres-mcp/releases).

## PostgreSQL Detection

The installer scans for running PostgreSQL instances when
you choose the auto-detect option or run interactively.

The detection process works as follows:

1. Scan TCP ports 5432 through 5436 for listening
   PostgreSQL instances.
2. Attempt passwordless authentication on each instance;
   the installer tries the `postgres` user first, then
   the current OS user.
3. List the available databases on each authenticated
   instance.
4. Present an interactive menu so you can pick a database.

When you use the `--detect` flag in non-interactive mode,
the installer connects to the first available instance
automatically.

## Install Location

Both installers place the MCP Server binary at the
following path:

```text
~/.pgedge/bin/pgedge-postgres-mcp
```

The demo database files and configuration live under
`~/.pgedge/demo/` when you use the demo option.

Each demo run replaces any previous demo container
(`pgedge-demo-*`); previous demo data in those containers
is removed.

## Production Deployment

These scripts are for quick evaluation and demos only. For
production deployment, see the
[Installation Guide](../../docs/guide/quickstart.md) in
the main project documentation.
