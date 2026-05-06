// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import styled from 'styled-components';

import {PlusIcon} from '@mattermost/compass-icons/components';

import BackendCard from './BackendCard';
import {TertiaryButton} from './buttons';
import type {BackendConfig, BackendStatus} from './types';
import {mergeBackendStatus} from './types';
import type {ValidationErrors} from './validation';

import {newBackendUUID} from '../../../utils/random_id';

const defaultNewBackend: Omit<BackendConfig, 'id'> = {
    name: '',
    type: 'dataminr',
    enabled: true,
    url: 'https://firstalert-api.dataminr.com',
    apiId: '',
    apiKey: '',
    channelId: '',
    pollIntervalSeconds: 30,
};

type Props = {
    backends: BackendConfig[];
    statusMap: Record<string, BackendStatus>;
    onChange: (backends: BackendConfig[]) => void;
    validationErrors?: Record<string, ValidationErrors>;
};

const BackendList = (props: Props) => {
    const addNewBackend = (e: React.MouseEvent<HTMLButtonElement>) => {
        e.preventDefault();
        const id = newBackendUUID();
        props.onChange([
            ...props.backends,
            {
                ...defaultNewBackend,
                id,
            },
        ]);
    };

    const onChange = (newBackend: BackendConfig) => {
        props.onChange(props.backends.map((b) => (b.id === newBackend.id ? newBackend : b)));
    };

    const onDelete = (id: string) => {
        props.onChange(props.backends.filter((b) => b.id !== id));
    };

    // Merge backend configs with status data
    const backendsWithStatus = props.backends.map((backend) =>
        mergeBackendStatus(backend, props.statusMap),
    );

    return (
        <>
            <BackendsListContainer>
                {backendsWithStatus.map((backend) => (
                    <BackendCard
                        key={backend.id}
                        backend={backend}
                        allBackends={props.backends}
                        onChange={onChange}
                        onDelete={() => onDelete(backend.id)}
                        validationErrors={props.validationErrors?.[backend.id]}
                    />
                ))}
            </BackendsListContainer>
            <TertiaryButton onClick={addNewBackend}>
                <PlusIcon/>
                {'Add Backend'}
            </TertiaryButton>
        </>
    );
};

const BackendsListContainer = styled.div`
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding-bottom: 24px;
`;

export default BackendList;
