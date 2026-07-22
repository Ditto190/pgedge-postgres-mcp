/*-------------------------------------------------------------------------
 *
 * pgEdge MCP Client - StatusBanner Conversation Actions Tests
 *
 * Copyright (c) 2025 - 2026, pgEdge, Inc.
 * This software is released under The PostgreSQL License
 *
 *-------------------------------------------------------------------------
 */

import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import StatusBanner from '../StatusBanner';
import {
    ConversationActionsProvider,
    useConversationActions,
} from '../../contexts/ConversationActionsContext';

// Mock the contexts StatusBanner depends on so we avoid all MCP network
// machinery. sessionToken is null so the mount effect does not fetch.
vi.mock('../../contexts/AuthContext', () => ({
    useAuth: () => ({ sessionToken: null, forceLogout: vi.fn() }),
}));

vi.mock('../../contexts/LLMProcessingContext', () => ({
    useLLMProcessing: () => ({ isProcessing: false }),
}));

vi.mock('../../contexts/DatabaseContext', () => ({
    useDatabaseContext: () => ({
        databases: [],
        currentDatabase: null,
        loading: false,
        error: null,
        fetchDatabases: vi.fn(),
        selectDatabase: vi.fn(),
    }),
}));

vi.mock('../../lib/mcp-client', () => ({
    MCPClient: vi.fn(),
}));

// Mock the popover to keep the tree light.
vi.mock('../DatabaseSelectorPopover', () => ({
    default: () => null,
}));

// Helper that registers conversation actions into the provider so the
// menu items are enabled, mirroring what ChatInterface does at runtime.
const ActionRegistrar = ({ onSave, onClear }) => {
    const { registerActions } = useConversationActions();
    React.useEffect(() => {
        registerActions({ hasMessages: true, onSave, onClear });
    }, [registerActions, onSave, onClear]);
    return null;
};

const renderBanner = ({ onSave = vi.fn(), onClear = vi.fn() } = {}) => {
    render(
        <ConversationActionsProvider>
            <ActionRegistrar onSave={onSave} onClear={onClear} />
            <StatusBanner />
        </ConversationActionsProvider>
    );
    return { onSave, onClear };
};

describe('StatusBanner conversation actions', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('renders the kebab actions button', () => {
        renderBanner();
        expect(
            screen.getByRole('button', { name: /conversation actions/i })
        ).toBeInTheDocument();
    });

    it('opens the menu with Save and Delete items', () => {
        renderBanner();
        fireEvent.click(
            screen.getByRole('button', { name: /conversation actions/i })
        );
        expect(screen.getByText('Save conversation')).toBeInTheDocument();
        expect(screen.getByText('Delete conversation')).toBeInTheDocument();
    });

    it('calls onSave when Save is clicked', () => {
        const { onSave } = renderBanner();
        fireEvent.click(
            screen.getByRole('button', { name: /conversation actions/i })
        );
        fireEvent.click(screen.getByText('Save conversation'));
        expect(onSave).toHaveBeenCalledTimes(1);
    });

    it('opens the confirm dialog and calls onClear on confirm', () => {
        const { onClear } = renderBanner();
        fireEvent.click(
            screen.getByRole('button', { name: /conversation actions/i })
        );
        fireEvent.click(screen.getByText('Delete conversation'));

        // Confirmation dialog appears.
        expect(
            screen.getByText(/clears the entire current conversation/i)
        ).toBeInTheDocument();
        expect(onClear).not.toHaveBeenCalled();

        // Confirm the delete.
        fireEvent.click(screen.getByRole('button', { name: 'Delete' }));
        expect(onClear).toHaveBeenCalledTimes(1);
    });
});
