/*-------------------------------------------------------------------------
 *
 * pgEdge MCP Client
 *
 * Copyright (c) 2025 - 2026, pgEdge, Inc.
 * This software is released under The PostgreSQL License
 *
 *-------------------------------------------------------------------------
 */

import React from 'react';
import { Box } from '@mui/material';
import { LLMProcessingProvider } from '../contexts/LLMProcessingContext';
import { DatabaseProvider } from '../contexts/DatabaseContext';
import { ConversationActionsProvider } from '../contexts/ConversationActionsContext';
import StatusBanner from './StatusBanner';
import ChatInterface from './ChatInterface';

const MainContent = ({ conversations }) => {
    return (
        <DatabaseProvider>
            <LLMProcessingProvider>
                <ConversationActionsProvider>
                    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
                        <StatusBanner />
                        <ChatInterface conversations={conversations} />
                    </Box>
                </ConversationActionsProvider>
            </LLMProcessingProvider>
        </DatabaseProvider>
    );
};

export default MainContent;
