/*-------------------------------------------------------------------------
 *
 * pgEdge MCP Client - Conversation Actions Context
 *
 * Copyright (c) 2025 - 2026, pgEdge, Inc.
 * This software is released under The PostgreSQL License
 *
 * Lightweight context that lets ChatInterface publish its
 * conversation-level actions (save/clear) up to sibling components
 * such as StatusBanner without lifting the large messages state.
 *
 *-------------------------------------------------------------------------
 */

import React, { createContext, useContext, useState, useCallback, useMemo } from 'react';
import PropTypes from 'prop-types';

const ConversationActionsContext = createContext(null);

export const ConversationActionsProvider = ({ children }) => {
    const [actions, setActions] = useState({
        hasMessages: false,
        onSave: null,
        onClear: null,
    });

    // Stable callback (empty deps + functional setState) so consumers
    // can safely call it from an effect without triggering update loops.
    const registerActions = useCallback((next) => {
        setActions((prev) => ({
            hasMessages: next.hasMessages,
            onSave: next.onSave,
            onClear: next.onClear,
        }));
    }, []);

    const value = useMemo(() => ({
        hasMessages: actions.hasMessages,
        onSave: actions.onSave,
        onClear: actions.onClear,
        registerActions,
    }), [actions, registerActions]);

    return (
        <ConversationActionsContext.Provider value={value}>
            {children}
        </ConversationActionsContext.Provider>
    );
};

ConversationActionsProvider.propTypes = {
    children: PropTypes.node.isRequired,
};

export const useConversationActions = () => {
    const context = useContext(ConversationActionsContext);
    if (!context) {
        throw new Error(
            'useConversationActions must be used within a ConversationActionsProvider'
        );
    }
    return context;
};

export default ConversationActionsContext;
