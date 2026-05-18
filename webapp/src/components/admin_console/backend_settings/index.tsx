// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useEffect, useRef, useMemo} from 'react';
import styled from 'styled-components';

import BackendList from './BackendList';
import {createDefaultBackend} from './default_backend';
import NoBackendsPage from './NoBackendsPage';
import {normalizeBackendsValue} from './normalize_backends_value';
import type {BackendConfig} from './types';
import {useBackendStatus} from './useBackendStatus';
import {collectBackendValidationErrors, type ValidationErrors} from './validation';

import {newBackendUUID} from '../../../utils/random_id';

type Props = {
    id: string;
    label: string;
    helpText: React.ReactNode;
    value?: unknown;
    disabled: boolean;
    config?: any;
    currentState?: any;
    license?: any;
    setByEnv: boolean;
    onChange: (id: string, value: any) => void;
    setSaveNeeded: () => void;
    registerSaveAction?: (action: () => Promise<{error?: {message?: string}}>) => void;
    unRegisterSaveAction?: (action: () => Promise<{error?: {message?: string}}>) => void;
};

const BackendSettings = (props: Props) => {
    const backends = useMemo(() => normalizeBackendsValue(props.value), [props.value]);
    const {statusMap} = useBackendStatus();

    // Validation errors shown after save attempt or on component mount
    const [validationErrors, setValidationErrors] = useState<Record<string, ValidationErrors>>({});

    // Track if user has made changes to avoid re-validating during edits
    const userHasMadeChanges = useRef(false);

    const handleBackendsChange = (newBackends: BackendConfig[]) => {
        userHasMadeChanges.current = true;
        props.onChange(props.id, newBackends);
        props.setSaveNeeded();

        // Clear validation errors when user makes changes
        setValidationErrors({});
    };

    // Validate backends on mount or when loaded from config (e.g., page refresh)
    useEffect(() => {
        // Skip validation if user has made changes (validation errors were already cleared)
        if (userHasMadeChanges.current) {
            return;
        }

        const errors = collectBackendValidationErrors(backends);
        if (Object.keys(errors).length > 0) {
            setValidationErrors(errors);
        }
    }, [backends]);

    // Register save action for validation
    useEffect(() => {
        if (!props.registerSaveAction || !props.unRegisterSaveAction) {
            return undefined;
        }

        const saveAction = async () => {
            const errors = collectBackendValidationErrors(backends);
            if (Object.keys(errors).length > 0) {
                setValidationErrors(errors);
                return {error: {message: 'Please fix validation errors before saving'}};
            }

            // All validations passed
            return {};
        };

        props.registerSaveAction(saveAction);
        return () => props.unRegisterSaveAction!(saveAction);
    }, [backends, props.registerSaveAction, props.unRegisterSaveAction]);

    const handleAddBackend = () => {
        handleBackendsChange([...backends, createDefaultBackend(newBackendUUID())]);
    };

    // Empty state: no backends configured
    if (backends.length === 0) {
        return (
            <Container>
                <NoBackendsPage onAddBackendPressed={handleAddBackend}/>
            </Container>
        );
    }

    // Render backends list
    return (
        <Container>
            <BackendList
                backends={backends}
                statusMap={statusMap}
                onChange={handleBackendsChange}
                validationErrors={validationErrors}
            />
        </Container>
    );
};

const Container = styled.div`
    box-sizing: border-box;
    display: flex;
    flex-direction: column;
    gap: 20px;
    width: 100%;
    min-height: 200px;
    align-self: stretch;
    flex: 1 1 auto;
`;

export default BackendSettings;
