// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {BackendConfig} from './types';

const defaultNewBackendFields: Omit<BackendConfig, 'id'> = {
    name: '',
    type: 'dataminr',
    enabled: true,
    url: 'https://firstalert-api.dataminr.com',
    apiId: '',
    apiKey: '',
    channelId: '',
    pollIntervalSeconds: 30,
};

export function createDefaultBackend(id: string): BackendConfig {
    return {...defaultNewBackendFields, id};
}
