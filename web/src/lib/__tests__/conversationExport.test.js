/*-------------------------------------------------------------------------
 *
 * pgEdge MCP Client - Conversation Export Tests
 *
 * Copyright (c) 2025 - 2026, pgEdge, Inc.
 * This software is released under The PostgreSQL License
 *
 *-------------------------------------------------------------------------
 */

import { describe, it, expect } from 'vitest';
import { conversationToMarkdown } from '../conversationExport';

// All data below is synthetic and made up for testing only.
const messages = [
    { role: 'user', content: 'Hello from Test User' },
    {
        role: 'assistant',
        content: 'Hello back from the example assistant',
        provider: 'example-provider',
        model: 'example-model',
        activity: [
            { type: 'tool', name: 'run_query', tokens: 42 },
        ],
    },
    { role: 'system', content: 'example system note' },
];

describe('conversationToMarkdown', () => {
    it('includes a top-level Chat History heading', () => {
        const md = conversationToMarkdown(messages);
        expect(md).toContain('# Chat History');
    });

    it('produces headings for user and assistant messages', () => {
        const md = conversationToMarkdown(messages);
        expect(md).toContain('## User');
        expect(md).toContain('Hello from Test User');
        expect(md).toContain('## Assistant');
        expect(md).toContain('Hello back from the example assistant');
        expect(md).toContain('*example-provider: example-model*');
    });

    it('omits system messages when debug is false', () => {
        const md = conversationToMarkdown(messages, { debug: false });
        expect(md).not.toContain('## System');
        expect(md).not.toContain('example system note');
    });

    it('includes system messages when debug is true', () => {
        const md = conversationToMarkdown(messages, { debug: true });
        expect(md).toContain('## System');
        expect(md).toContain('> example system note');
    });

    it('omits activity detail when showActivity is false', () => {
        const md = conversationToMarkdown(messages, { showActivity: false });
        expect(md).not.toContain('### Activity');
        expect(md).not.toContain('run_query');
    });

    it('includes activity detail when showActivity is true', () => {
        const md = conversationToMarkdown(messages, { showActivity: true });
        expect(md).toContain('### Activity');
        expect(md).toContain('**run_query**');
        expect(md).toContain('(~42 tokens)');
    });

    it('returns header-only output for an empty conversation', () => {
        const md = conversationToMarkdown([]);
        expect(md).toContain('# Chat History');
        expect(md).not.toContain('## User');
    });
});
