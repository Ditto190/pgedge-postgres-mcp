/*-------------------------------------------------------------------------
 *
 * pgEdge MCP Client - Conversation Export Utilities
 *
 * Copyright (c) 2025 - 2026, pgEdge, Inc.
 * This software is released under The PostgreSQL License
 *
 * Pure helpers for exporting a conversation to Markdown and triggering
 * a client-side download. Extracted from MessageInput so the export can
 * be driven from the StatusBanner header menu.
 *
 *-------------------------------------------------------------------------
 */

// Format a message timestamp as a locale string, or '' when absent.
const formatTimestamp = (msg) =>
    msg.timestamp ? new Date(msg.timestamp).toLocaleString() : '';

// Render a single activity entry as a Markdown bullet, or null when the
// activity type is not one we export.
const formatActivityItem = (act) => {
    if (act.type === 'tool') {
        const tokenInfo = act.tokens ? ` (~${act.tokens} tokens)` : '';
        const errorInfo = act.isError ? ' [ERROR]' : '';
        if (act.name === 'read_resource' && act.uri) {
            return `- **${act.name}**: \`${act.uri}\`${tokenInfo}${errorInfo}`;
        }
        return `- **${act.name}**${tokenInfo}${errorInfo}`;
    }
    if (act.type === 'compaction') {
        return `- *Compacted: ${act.originalCount} → ${act.compactedCount} messages*`;
    }
    if (act.type === 'rate_limit_pause') {
        return `- *Rate limit pause: ${act.message}*`;
    }
    return null;
};

// Render the "### Activity" block for an assistant message; returns an
// empty array when there is nothing to show.
const formatActivityBlock = (activity) => {
    const items = activity.map(formatActivityItem).filter((line) => line !== null);
    if (items.length === 0) return [];
    return ['### Activity', '', ...items, ''];
};

const formatUserMessage = (msg, timestamp) => {
    const lines = ['## User'];
    if (timestamp) lines.push(`*${timestamp}*`);
    lines.push('', msg.content, '');
    return lines;
};

const formatAssistantMessage = (msg, timestamp, showActivity) => {
    const lines = ['## Assistant'];
    if (timestamp) lines.push(`*${timestamp}*`);
    if (msg.provider && msg.model) lines.push(`*${msg.provider}: ${msg.model}*`);
    lines.push('');
    if (showActivity && msg.activity && msg.activity.length > 0) {
        lines.push(...formatActivityBlock(msg.activity));
    }
    lines.push(msg.content, '');
    return lines;
};

const formatSystemMessage = (msg, timestamp) => {
    const lines = ['## System'];
    if (timestamp) lines.push(`*${timestamp}*`);
    lines.push('', `> ${msg.content}`, '');
    return lines;
};

// Render one message to its Markdown lines, or null when it should be
// skipped (e.g. a system message while debug is disabled).
const formatMessage = (msg, { showActivity, debug }) => {
    const timestamp = formatTimestamp(msg);
    if (msg.role === 'user') return formatUserMessage(msg, timestamp);
    if (msg.role === 'assistant') return formatAssistantMessage(msg, timestamp, showActivity);
    if (msg.role === 'system' && debug) return formatSystemMessage(msg, timestamp);
    return null;
};

/**
 * Convert an array of chat messages to a Markdown document.
 *
 * @param {Array<Object>} messages - Chat messages to export.
 * @param {Object} [options] - Export options.
 * @param {boolean} [options.showActivity=false] - Include tool/activity
 *     detail for assistant messages when true.
 * @param {boolean} [options.debug=false] - Include system messages when
 *     true; otherwise they are skipped.
 * @returns {string} The Markdown document.
 */
export const conversationToMarkdown = (messages = [], { showActivity = false, debug = false } = {}) => {
    const lines = [
        '# Chat History',
        '',
        `*Exported: ${new Date().toLocaleString()}*`,
        '',
        '---',
        '',
    ];

    for (const msg of messages) {
        const messageLines = formatMessage(msg, { showActivity, debug });
        if (messageLines === null) continue;
        lines.push(...messageLines, '---', '');
    }

    return lines.join('\n');
};

/**
 * Trigger a client-side download of a Markdown string as a file.
 *
 * @param {string} markdown - The Markdown content to download.
 * @param {string} filename - The download filename.
 */
export const downloadMarkdown = (markdown, filename) => {
    const blob = new Blob([markdown], { type: 'text/markdown' });
    const url = URL.createObjectURL(blob);

    // Create a temporary link and trigger download
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);

    // Clean up the URL object
    URL.revokeObjectURL(url);
};
