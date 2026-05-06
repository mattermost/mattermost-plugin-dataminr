// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {BackendConfig} from './types';

/**
 * Coerce plugin setting value from the admin console into BackendConfig[].
 * Some hosts pass JSON arrays; others may pass a serialized JSON string.
 */
export function normalizeBackendsValue(raw: unknown): BackendConfig[] {
    if (raw == null) {
        return [];
    }
    if (Array.isArray(raw)) {
        return raw as BackendConfig[];
    }
    if (typeof raw === 'string') {
        const trimmed = raw.trim();
        if (!trimmed) {
            return [];
        }
        try {
            const parsed: unknown = JSON.parse(trimmed);
            return Array.isArray(parsed) ? (parsed as BackendConfig[]) : [];
        } catch {
            return [];
        }
    }
    return [];
}
