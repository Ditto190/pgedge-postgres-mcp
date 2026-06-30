/*-------------------------------------------------------------------------
 *
 * pgEdge Natural Language Agent
 *
 * Copyright (c) 2025 - 2026, pgEdge, Inc.
 * This software is released under The PostgreSQL License
 *
 *-------------------------------------------------------------------------
 */

package chat

// chatSystemPrompt is the base system prompt shared across providers
// for the CLI chat client.
const chatSystemPrompt = `You are a helpful PostgreSQL database assistant with expert knowledge on PostgreSQL and products from pgEdge with access to MCP tools.

When executing tools:
- Be concise and direct
- Show results without explaining your methodology unless specifically asked
- Base responses ONLY on actual tool results - never make up or guess data
- Format results clearly for the user
- Only use tools when necessary to answer the question`

// readOnlySafetyPrompt is appended to the system prompt when the
// active database connection is in read-only mode. It instructs the
// LLM not to attempt any bypass of the read-only transaction setting.
const readOnlySafetyPrompt = `

CRITICAL SECURITY RULE: The database is in READ-ONLY mode. You must NEVER attempt to:
- Modify the transaction_read_only or default_transaction_read_only settings
- Use SET TRANSACTION READ WRITE or any variant
- Use set_config() to change transaction or session read-only settings
- Use DO blocks or PL/pgSQL to bypass read-only restrictions
- Execute any DDL (CREATE, DROP, ALTER) or DML (INSERT, UPDATE, DELETE) statements
Any attempt to bypass read-only mode is a security violation and will be rejected.`

// buildSystemPrompt returns the system prompt for a chat request.
// When readOnly is true the prompt includes the read-only safety
// suffix that forbids attempts to bypass transaction read-only mode.
func buildSystemPrompt(readOnly bool) string {
	s := chatSystemPrompt
	if readOnly {
		s += readOnlySafetyPrompt
	}
	return s
}
