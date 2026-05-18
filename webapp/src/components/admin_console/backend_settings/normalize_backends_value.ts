// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {BackendConfig} from './types';

/**
 * True when `item` has the structural shape of BackendConfig (types only).
 * Business rules (UUID v4, HTTPS URL, etc.) are enforced by validation elsewhere.
 */
export function isBackendConfig(item: unknown): item is BackendConfig {
    if (item === null || typeof item !== 'object' || Array.isArray(item)) {
        return false;
    }
    const o = item as Record<string, unknown>;
    return (
        typeof o.id === 'string' &&
        typeof o.name === 'string' &&
        typeof o.type === 'string' &&
        typeof o.enabled === 'boolean' &&
        typeof o.url === 'string' &&
        typeof o.apiId === 'string' &&
        typeof o.apiKey === 'string' &&
        typeof o.channelId === 'string' &&
        typeof o.pollIntervalSeconds === 'number'
    );
}

function filterValidBackends(items: unknown[]): BackendConfig[] {
    return items.filter(isBackendConfig);
}

/**
 * Coerce plugin setting value from the admin console into BackendConfig[].
 * Some hosts pass JSON arrays; others may pass a serialized JSON string.
 * Entries that are not plain BackendConfig-shaped objects are dropped.
 */
export function normalizeBackendsValue(raw: unknown): BackendConfig[] {
    if (raw == null) {
        return [];
    }
    if (Array.isArray(raw)) {
        return filterValidBackends(raw);
    }
    if (typeof raw === 'string') {
        const trimmed = raw.trim();
        if (!trimmed) {
            return [];
        }
        try {
            const parsed: unknown = JSON.parse(trimmed);
            return Array.isArray(parsed) ? filterValidBackends(parsed) : [];
        } catch {
            return [];
        }
    }
    return [];
}
