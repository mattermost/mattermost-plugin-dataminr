// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {normalizeBackendsValue} from './normalize_backends_value';
import type {BackendConfig} from './types';

describe('normalizeBackendsValue', () => {
    const validBackend: BackendConfig = {
        id: '550e8400-e29b-41d4-a716-446655440000',
        name: 'Test',
        type: 'dataminr',
        enabled: true,
        url: 'https://firstalert-api.dataminr.com',
        apiId: 'a',
        apiKey: 'b',
        channelId: 'c',
        pollIntervalSeconds: 30,
    };

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
        const backends: BackendConfig[] = [validBackend];
        expect(normalizeBackendsValue(JSON.stringify(backends))).toEqual(backends);
    });

    it('returns empty array for empty array input', () => {
        expect(normalizeBackendsValue([])).toEqual([]);
    });

    it('keeps only valid backends when input array mixes malformed entries', () => {
        const mixed: unknown[] = [
            null,
            undefined,
            'not-an-object',
            42,
            {},
            {name: 123},
            {id: '550e8400-e29b-41d4-a716-446655440000'},
            {
                id: '550e8400-e29b-41d4-a716-446655440000',
                name: 'Partial',
                type: 'dataminr',
                enabled: true,
                url: 'https://example.com',
            },
            validBackend,
            {
                ...validBackend,
                pollIntervalSeconds: '30',
            },
        ];
        expect(normalizeBackendsValue(mixed)).toEqual([validBackend]);
    });

    it('filters malformed entries from JSON array string without throwing', () => {
        const mixed = [
            null,
            {foo: 'bar'},
            validBackend,
        ];
        expect(normalizeBackendsValue(JSON.stringify(mixed))).toEqual([validBackend]);
    });

    it('returns equivalent copy for an array of valid backends', () => {
        const backends: BackendConfig[] = [validBackend];
        expect(normalizeBackendsValue(backends)).toEqual(backends);
    });

    it('returns empty array for unexpected types', () => {
        expect(normalizeBackendsValue(42)).toEqual([]);
        expect(normalizeBackendsValue(true)).toEqual([]);
    });
});
