// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {normalizeBackendsValue} from './normalize_backends_value';
import type {BackendConfig} from './types';

describe('normalizeBackendsValue', () => {
    it('returns empty array for null and undefined', () => {
        expect(normalizeBackendsValue(null)).toEqual([]);
        expect(normalizeBackendsValue(undefined)).toEqual([]);
    });

    it('returns empty array for whitespace-only string', () => {
        expect(normalizeBackendsValue('   ')).toEqual([]);
    });

    it('returns empty array for non-array JSON', () => {
        expect(normalizeBackendsValue('{}')).toEqual([]);
        expect(normalizeBackendsValue('"x"')).toEqual([]);
    });

    it('returns empty array for invalid JSON string', () => {
        expect(normalizeBackendsValue('{not json')).toEqual([]);
    });

    it('parses JSON array string', () => {
        const backends: BackendConfig[] = [{
            id: '550e8400-e29b-41d4-a716-446655440000',
            name: 'Test',
            type: 'dataminr',
            enabled: true,
            url: 'https://firstalert-api.dataminr.com',
            apiId: 'a',
            apiKey: 'b',
            channelId: 'c',
            pollIntervalSeconds: 30,
        }];
        expect(normalizeBackendsValue(JSON.stringify(backends))).toEqual(backends);
    });

    it('returns array as-is', () => {
        const backends: BackendConfig[] = [];
        expect(normalizeBackendsValue(backends)).toBe(backends);
    });

    it('returns empty array for unexpected types', () => {
        expect(normalizeBackendsValue(42)).toEqual([]);
        expect(normalizeBackendsValue(true)).toEqual([]);
    });
});
