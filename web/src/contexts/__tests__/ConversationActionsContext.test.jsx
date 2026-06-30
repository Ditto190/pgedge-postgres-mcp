/*-------------------------------------------------------------------------
 *
 * pgEdge MCP Client - Conversation Actions Context Tests
 *
 * Copyright (c) 2025 - 2026, pgEdge, Inc.
 * This software is released under The PostgreSQL License
 *
 *-------------------------------------------------------------------------
 */

import { describe, it, expect } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import {
    ConversationActionsProvider,
    useConversationActions,
} from '../ConversationActionsContext';

const wrapper = ({ children }) => (
    <ConversationActionsProvider>{children}</ConversationActionsProvider>
);

describe('ConversationActionsContext', () => {
    it('provides default values', () => {
        const { result } = renderHook(() => useConversationActions(), { wrapper });

        expect(result.current.hasMessages).toBe(false);
        expect(result.current.onSave).toBeNull();
        expect(result.current.onClear).toBeNull();
        expect(typeof result.current.registerActions).toBe('function');
    });

    it('updates consumed values when registerActions is called', () => {
        const { result } = renderHook(() => useConversationActions(), { wrapper });

        const onSave = () => {};
        const onClear = () => {};

        act(() => {
            result.current.registerActions({ hasMessages: true, onSave, onClear });
        });

        expect(result.current.hasMessages).toBe(true);
        expect(result.current.onSave).toBe(onSave);
        expect(result.current.onClear).toBe(onClear);
    });

    it('keeps registerActions identity stable across renders', () => {
        const { result, rerender } = renderHook(() => useConversationActions(), { wrapper });

        const firstRegister = result.current.registerActions;

        act(() => {
            result.current.registerActions({ hasMessages: true, onSave: null, onClear: null });
        });
        rerender();

        expect(result.current.registerActions).toBe(firstRegister);
    });

    it('throws when used outside the provider', () => {
        expect(() => renderHook(() => useConversationActions())).toThrow(
            /must be used within a ConversationActionsProvider/
        );
    });
});
