// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {BackendConfig} from './types';
import {
    isValidUUID,
    isValidHttpsUrl,
    hasDuplicateName,
    isValidBackendType,
    isValidPollInterval,
    validateBackendConfig,
    hasValidationErrors,
    collectBackendValidationErrors,
    validateField,
} from './validation';

describe('validation utilities', () => {
    describe('isValidUUID', () => {
        it('should return true for valid UUID v4', () => {
            expect(isValidUUID('550e8400-e29b-41d4-a716-446655440000')).toBe(true);
            expect(isValidUUID('6ba7b810-9dad-11d1-80b4-00c04fd430c8')).toBe(false); // UUID v1
            expect(isValidUUID('f47ac10b-58cc-4372-a567-0e02b2c3d479')).toBe(true);
        });

        it('should return false for invalid UUIDs', () => {
            expect(isValidUUID('')).toBe(false);
            expect(isValidUUID('not-a-uuid')).toBe(false);
            expect(isValidUUID('550e8400-e29b-41d4-a716')).toBe(false);
            expect(isValidUUID('550e8400-e29b-41d4-a716-446655440000-extra')).toBe(false);
        });

        it('should return false for non-string inputs', () => {
            expect(isValidUUID(null as any)).toBe(false);
            expect(isValidUUID(undefined as any)).toBe(false);
            expect(isValidUUID(123 as any)).toBe(false);
        });
    });

    describe('isValidHttpsUrl', () => {
        it('should return true for valid HTTPS URLs', () => {
            expect(isValidHttpsUrl('https://example.com')).toBe(true);
            expect(isValidHttpsUrl('https://api.example.com/path')).toBe(true);
            expect(isValidHttpsUrl('https://firstalert-api.dataminr.com')).toBe(true);
        });

        it('should return false for HTTP URLs', () => {
            expect(isValidHttpsUrl('http://example.com')).toBe(false);
        });

        it('should return false for invalid URLs', () => {
            expect(isValidHttpsUrl('')).toBe(false);
            expect(isValidHttpsUrl('not-a-url')).toBe(false);
            expect(isValidHttpsUrl('ftp://example.com')).toBe(false);
        });

        it('should return false for non-string inputs', () => {
            expect(isValidHttpsUrl(null as any)).toBe(false);
            expect(isValidHttpsUrl(undefined as any)).toBe(false);
        });
    });

    describe('hasDuplicateName', () => {
        const backends: BackendConfig[] = [
            {
                id: '550e8400-e29b-41d4-a716-446655440000',
                name: 'Production',
                type: 'dataminr',
                enabled: true,
                url: 'https://api.example.com',
                apiId: 'id1',
                apiKey: 'key1',
                channelId: 'ch1',
                pollIntervalSeconds: 30,
            },
            {
                id: 'f47ac10b-58cc-4372-a567-0e02b2c3d479',
                name: 'Staging',
                type: 'dataminr',
                enabled: true,
                url: 'https://api.example.com',
                apiId: 'id2',
                apiKey: 'key2',
                channelId: 'ch2',
                pollIntervalSeconds: 30,
            },
        ];

        it('should return true if name is duplicate', () => {
            expect(hasDuplicateName('Production', 'new-id', backends)).toBe(true);
            expect(hasDuplicateName('Staging', 'new-id', backends)).toBe(true);
        });

        it('should return false if name is unique', () => {
            expect(hasDuplicateName('Development', 'new-id', backends)).toBe(false);
        });

        it('should exclude current backend from duplicate check', () => {
            expect(hasDuplicateName('Production', '550e8400-e29b-41d4-a716-446655440000', backends)).toBe(false);
        });

        it('should return false for empty name', () => {
            expect(hasDuplicateName('', 'new-id', backends)).toBe(false);
        });

        it('should return false for empty backends array', () => {
            expect(hasDuplicateName('Production', 'new-id', [])).toBe(false);
        });
    });

    describe('isValidBackendType', () => {
        it('should return true for supported types', () => {
            expect(isValidBackendType('dataminr')).toBe(true);
        });

        it('should return false for unsupported types', () => {
            expect(isValidBackendType('unsupported')).toBe(false);
            expect(isValidBackendType('')).toBe(false);
        });
    });

    describe('isValidPollInterval', () => {
        it('should return true for valid intervals', () => {
            expect(isValidPollInterval(10)).toBe(true);
            expect(isValidPollInterval(30)).toBe(true);
            expect(isValidPollInterval(100)).toBe(true);
        });

        it('should return false for intervals below minimum', () => {
            expect(isValidPollInterval(9)).toBe(false);
            expect(isValidPollInterval(0)).toBe(false);
            expect(isValidPollInterval(-1)).toBe(false);
        });

        it('should return false for non-number inputs', () => {
            expect(isValidPollInterval(null as any)).toBe(false);
            expect(isValidPollInterval(undefined as any)).toBe(false);
            expect(isValidPollInterval('30' as any)).toBe(false);
        });
    });

    describe('validateBackendConfig', () => {
        const validConfig: BackendConfig = {
            id: '550e8400-e29b-41d4-a716-446655440000',
            name: 'Test Backend',
            type: 'dataminr',
            enabled: true,
            url: 'https://api.example.com',
            apiId: 'test-id',
            apiKey: 'test-key',
            channelId: 'test-channel',
            pollIntervalSeconds: 30,
        };

        it('should return no errors for valid config', () => {
            const errors = validateBackendConfig(validConfig, []);
            expect(hasValidationErrors(errors)).toBe(false);
            expect(Object.keys(errors).length).toBe(0);
        });

        it('should return error for missing id', () => {
            const config = {...validConfig, id: ''};
            const errors = validateBackendConfig(config, []);
            expect(errors.id).toBe('ID is required');
        });

        it('should return error for invalid UUID', () => {
            const config = {...validConfig, id: 'invalid-uuid'};
            const errors = validateBackendConfig(config, []);
            expect(errors.id).toBe('ID must be a valid UUID v4');
        });

        it('should return error for missing name', () => {
            const config = {...validConfig, name: ''};
            const errors = validateBackendConfig(config, []);
            expect(errors.name).toBe('Name is required');
        });

        it('should return error for duplicate name', () => {
            const existing: BackendConfig = {
                ...validConfig,
                id: 'f47ac10b-58cc-4372-a567-0e02b2c3d479',
            };
            const errors = validateBackendConfig(validConfig, [existing]);
            expect(errors.name).toBe('A backend with this name already exists');
        });

        it('should return error for missing type', () => {
            const config = {...validConfig, type: ''};
            const errors = validateBackendConfig(config, []);
            expect(errors.type).toBe('Type is required');
        });

        it('should return error for invalid type', () => {
            const config = {...validConfig, type: 'unsupported'};
            const errors = validateBackendConfig(config, []);
            expect(errors.type).toContain('Type must be one of');
        });

        it('should return error for missing url', () => {
            const config = {...validConfig, url: ''};
            const errors = validateBackendConfig(config, []);
            expect(errors.url).toBe('URL is required');
        });

        it('should return error for non-HTTPS URL', () => {
            const config = {...validConfig, url: 'http://api.example.com'};
            const errors = validateBackendConfig(config, []);
            expect(errors.url).toBe('URL must be a valid HTTPS URL');
        });

        it('should return error for invalid URL', () => {
            const config = {...validConfig, url: 'not-a-url'};
            const errors = validateBackendConfig(config, []);
            expect(errors.url).toBe('URL must be a valid HTTPS URL');
        });

        it('should return error for missing apiId', () => {
            const config = {...validConfig, apiId: ''};
            const errors = validateBackendConfig(config, []);
            expect(errors.apiId).toBe('API ID is required');
        });

        it('should return error for missing apiKey', () => {
            const config = {...validConfig, apiKey: ''};
            const errors = validateBackendConfig(config, []);
            expect(errors.apiKey).toBe('API Key is required');
        });

        it('should return error for missing channelId', () => {
            const config = {...validConfig, channelId: ''};
            const errors = validateBackendConfig(config, []);
            expect(errors.channelId).toBe('Channel ID is required');
        });

        it('should return error for missing pollIntervalSeconds', () => {
            const config = {...validConfig, pollIntervalSeconds: undefined as any};
            const errors = validateBackendConfig(config, []);
            expect(errors.pollIntervalSeconds).toBe('Poll interval is required');
        });

        it('should return error for invalid poll interval', () => {
            const config = {...validConfig, pollIntervalSeconds: 5};
            const errors = validateBackendConfig(config, []);
            expect(errors.pollIntervalSeconds).toContain('must be at least');
        });

        it('should return multiple errors for invalid config', () => {
            const config: BackendConfig = {
                id: '',
                name: '',
                type: 'unsupported',
                enabled: true,
                url: 'http://example.com',
                apiId: '',
                apiKey: '',
                channelId: '',
                pollIntervalSeconds: 5,
            };
            const errors = validateBackendConfig(config, []);
            expect(hasValidationErrors(errors)).toBe(true);
            expect(errors.id).toBeDefined();
            expect(errors.name).toBeDefined();
            expect(errors.type).toBeDefined();
            expect(errors.url).toBeDefined();
            expect(errors.apiId).toBeDefined();
            expect(errors.apiKey).toBeDefined();
            expect(errors.channelId).toBeDefined();
            expect(errors.pollIntervalSeconds).toBeDefined();
        });
    });

    describe('hasValidationErrors', () => {
        it('should return true if errors exist', () => {
            expect(hasValidationErrors({name: 'Name is required'})).toBe(true);
            expect(hasValidationErrors({name: 'Error', url: 'Error'})).toBe(true);
        });

        it('should return false if no errors exist', () => {
            expect(hasValidationErrors({})).toBe(false);
        });
    });

    describe('collectBackendValidationErrors', () => {
        it('should return empty object when all backends are valid', () => {
            const backends: BackendConfig[] = [{
                id: '550e8400-e29b-41d4-a716-446655440000',
                name: 'A',
                type: 'dataminr',
                enabled: true,
                url: 'https://api.example.com',
                apiId: 'id',
                apiKey: 'key',
                channelId: 'ch',
                pollIntervalSeconds: 30,
            }];
            expect(collectBackendValidationErrors(backends)).toEqual({});
        });

        it('should include only backends with validation errors', () => {
            const invalid: BackendConfig = {
                id: '550e8400-e29b-41d4-a716-446655440000',
                name: '',
                type: 'dataminr',
                enabled: true,
                url: 'https://api.example.com',
                apiId: 'id',
                apiKey: 'key',
                channelId: 'ch',
                pollIntervalSeconds: 30,
            };
            const valid: BackendConfig = {
                id: 'f47ac10b-58cc-4372-a567-0e02b2c3d479',
                name: 'OK',
                type: 'dataminr',
                enabled: true,
                url: 'https://api.example.com',
                apiId: 'id2',
                apiKey: 'key2',
                channelId: 'ch2',
                pollIntervalSeconds: 30,
            };
            const collected = collectBackendValidationErrors([invalid, valid]);
            expect(Object.keys(collected)).toEqual([invalid.id]);
            expect(collected[invalid.id]?.name).toBe('Name is required');
        });
    });

    describe('validateField', () => {
        const config: BackendConfig = {
            id: '550e8400-e29b-41d4-a716-446655440000',
            name: '',
            type: 'dataminr',
            enabled: true,
            url: 'https://api.example.com',
            apiId: 'test-id',
            apiKey: 'test-key',
            channelId: 'test-channel',
            pollIntervalSeconds: 30,
        };

        it('should return error for specific field', () => {
            const error = validateField('name', config, []);
            expect(error).toBe('Name is required');
        });

        it('should return undefined for valid field', () => {
            const error = validateField('url', config, []);
            expect(error).toBeUndefined();
        });
    });
});
