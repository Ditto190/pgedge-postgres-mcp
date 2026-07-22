/*-------------------------------------------------------------------------
 *
 * pgEdge MCP Client - Message Input Component
 *
 * Copyright (c) 2025 - 2026, pgEdge, Inc.
 * This software is released under The PostgreSQL License
 *
 * Styled to match pgEdge Cloud product aesthetics
 *
 *-------------------------------------------------------------------------
 */

import React, { useRef, useEffect } from 'react';
import PropTypes from 'prop-types';
import { Box, TextField, IconButton, Tooltip, useTheme, alpha } from '@mui/material';
import { Send as SendIcon, Stop as StopIcon, Psychology as PsychologyIcon } from '@mui/icons-material';

const MessageInput = React.memo(({
    value,
    onChange,
    onSend,
    onCancel,
    onKeyDown,
    disabled,
    isLoading = false,
    onPromptClick,
    hasPrompts = false,
}) => {
    const inputRef = useRef(null);
    const theme = useTheme();
    const isDark = theme.palette.mode === 'dark';

    // Auto-focus input when it becomes enabled
    useEffect(() => {
        if (!disabled && inputRef.current) {
            // Use setTimeout to ensure the focus happens after the disabled state update
            const timer = setTimeout(() => {
                inputRef.current?.focus();
            }, 0);
            return () => clearTimeout(timer);
        }
    }, [disabled]);

    return (
        <Box sx={{ display: 'flex', gap: 1, alignItems: 'center', mb: 2 }}>
            <TextField
                inputRef={inputRef}
                fullWidth
                multiline
                maxRows={4}
                variant="outlined"
                placeholder="Ask about your database..."
                value={value}
                onChange={onChange}
                onKeyDown={onKeyDown}
                disabled={disabled}
                autoFocus
                sx={{
                    '& .MuiOutlinedInput-root': {
                        borderRadius: 1,
                        bgcolor: isDark ? alpha('#1E293B', 0.5) : '#FFFFFF',
                        '& fieldset': {
                            borderColor: isDark ? '#334155' : '#E5E7EB',
                        },
                        '&:hover fieldset': {
                            borderColor: isDark ? '#475569' : '#9CA3AF',
                        },
                        '&.Mui-focused fieldset': {
                            borderColor: '#15AABF',
                            borderWidth: 2,
                        },
                    },
                    '& .MuiInputBase-input': {
                        color: isDark ? '#F1F5F9' : '#1F2937',
                        '&::placeholder': {
                            color: isDark ? '#64748B' : '#9CA3AF',
                            opacity: 1,
                        },
                    },
                }}
            />
            {hasPrompts && (
                <Tooltip title="Execute Prompt">
                    <IconButton
                        onClick={onPromptClick}
                        disabled={disabled}
                        size="small"
                        sx={{
                            color: isDark ? '#94A3B8' : '#6B7280',
                            '&:hover': {
                                bgcolor: isDark ? alpha('#22B8CF', 0.08) : alpha('#15AABF', 0.04),
                                color: '#15AABF',
                            },
                            '&.Mui-disabled': {
                                color: isDark ? '#475569' : '#D1D5DB',
                            },
                        }}
                    >
                        <PsychologyIcon />
                    </IconButton>
                </Tooltip>
            )}
            <Tooltip title={isLoading ? "Cancel request" : "Send message"}>
                <span>
                    <IconButton
                        onClick={isLoading ? onCancel : onSend}
                        disabled={isLoading ? false : (!value.trim() || disabled)}
                        sx={{
                            bgcolor: isLoading ? '#EF4444' : '#15AABF',
                            color: 'white',
                            width: 40,
                            height: 40,
                            '&:hover': {
                                bgcolor: isLoading ? '#DC2626' : '#0C8599',
                            },
                            '&.Mui-disabled': {
                                bgcolor: isDark ? '#334155' : '#E5E7EB',
                                color: isDark ? '#64748B' : '#9CA3AF',
                            },
                        }}
                    >
                        {isLoading ? <StopIcon /> : <SendIcon />}
                    </IconButton>
                </span>
            </Tooltip>
        </Box>
    );
});

MessageInput.displayName = 'MessageInput';

MessageInput.propTypes = {
    value: PropTypes.string.isRequired,
    onChange: PropTypes.func.isRequired,
    onSend: PropTypes.func.isRequired,
    onCancel: PropTypes.func,
    onKeyDown: PropTypes.func.isRequired,
    disabled: PropTypes.bool.isRequired,
    isLoading: PropTypes.bool,
    onPromptClick: PropTypes.func,
    hasPrompts: PropTypes.bool,
};

export default MessageInput;
