// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {MinPollIntervalSeconds, SupportedBackendTypes} from './constants';
import type {BackendConfig} from './types';

/**
 * Validation errors for a backend configuration.
 * Each field can have an optional error message.
 */
export interface ValidationErrors {
    id?: string;
    name?: string;
    type?: string;
    url?: string;
    apiId?: string;
    apiKey?: string;
    channelId?: string;
    pollIntervalSeconds?: string;
}

/**
 * Validates if a string is a valid UUID v4.
 * UUID v4 format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
 * where x is any hexadecimal digit and y is one of 8, 9, A, or B
 */
export function isValidUUID(id: string): boolean {
    if (!id || typeof id !== 'string') {
        return false;
    }

    const uuidV4Regex = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
    return uuidV4Regex.test(id);
}

/**
 * Validates if a URL is valid and uses HTTPS protocol.
 */
export function isValidHttpsUrl(url: string): boolean {
    if (!url || typeof url !== 'string') {
        return false;
    }

    try {
        const parsed = new URL(url);
        return parsed.protocol === 'https:';
    } catch {
        return false;
    }
}

/**
 * Checks if a backend name is a duplicate.
 * Excludes the current backend being edited from the check.
 */
export function hasDuplicateName(
    name: string,
    currentId: string,
    allBackends: BackendConfig[],
): boolean {
    if (!name) {
        return false; // Empty name will be caught by required field validation
    }

    return allBackends.some(
        (backend) => backend.id !== currentId && backend.name === name,
    );
}

/**
 * Validates if a backend type is supported.
 */
export function isValidBackendType(type: string): boolean {
    return SupportedBackendTypes.includes(type as any);
}

/**
 * Validates if a poll interval is valid (>= MinPollIntervalSeconds).
 */
export function isValidPollInterval(interval: number): boolean {
    return typeof interval === 'number' && interval >= MinPollIntervalSeconds;
}

/**
 * Validates a complete backend configuration.
 * Returns an object with field-specific error messages.
 * An empty object means validation passed.
 */
export function validateBackendConfig(
    config: BackendConfig,
    allBackends: BackendConfig[],
): ValidationErrors {
    const errors: ValidationErrors = {};

    // 1. Required Fields Validation
    if (!config.id || config.id.trim() === '') {
        errors.id = 'ID is required';
    }

    if (!config.name || config.name.trim() === '') {
        errors.name = 'Name is required';
    }

    if (!config.type || config.type.trim() === '') {
        errors.type = 'Type is required';
    }

    if (!config.url || config.url.trim() === '') {
        errors.url = 'URL is required';
    }

    if (!config.apiId || config.apiId.trim() === '') {
        errors.apiId = 'API ID is required';
    }

    if (!config.apiKey || config.apiKey.trim() === '') {
        errors.apiKey = 'API Key is required';
    }

    if (!config.channelId || config.channelId.trim() === '') {
        errors.channelId = 'Channel ID is required';
    }

    if (config.pollIntervalSeconds === undefined || config.pollIntervalSeconds === null) {
        errors.pollIntervalSeconds = 'Poll interval is required';
    }

    // 2. UUID Format Validation (only if ID is not empty)
    if (config.id && !isValidUUID(config.id)) {
        errors.id = 'ID must be a valid UUID v4';
    }

    // 3. Duplicate Name Validation (only if name is not empty)
    if (config.name && hasDuplicateName(config.name, config.id, allBackends)) {
        errors.name = 'A backend with this name already exists';
    }

    // 4. Type Support Validation (only if type is not empty)
    if (config.type && !isValidBackendType(config.type)) {
        errors.type = `Type must be one of: ${SupportedBackendTypes.join(', ')}`;
    }

    // 5. URL Format Validation (only if URL is not empty)
    if (config.url && !isValidHttpsUrl(config.url)) {
        errors.url = 'URL must be a valid HTTPS URL';
    }

    // 6. Poll Interval Validation (only if poll interval is set)
    if (config.pollIntervalSeconds !== undefined &&
        config.pollIntervalSeconds !== null &&
        !isValidPollInterval(config.pollIntervalSeconds)) {
        errors.pollIntervalSeconds = `Poll interval must be at least ${MinPollIntervalSeconds} seconds`;
    }

    return errors;
}

/**
 * Checks if a ValidationErrors object has any errors.
 */
export function hasValidationErrors(errors: ValidationErrors): boolean {
    return Object.keys(errors).length > 0;
}

/**
 * Runs validateBackendConfig for each backend and returns only entries with errors.
 */
export function collectBackendValidationErrors(
    backends: BackendConfig[],
): Record<string, ValidationErrors> {
    const errors: Record<string, ValidationErrors> = {};
    for (const backend of backends) {
        const backendErrors = validateBackendConfig(backend, backends);
        if (hasValidationErrors(backendErrors)) {
            errors[backend.id] = backendErrors;
        }
    }
    return errors;
}

/**
 * Validates a single field of a backend configuration.
 * Returns an error message string, or undefined if valid.
 */
export function validateField(
    fieldName: keyof BackendConfig,
    config: BackendConfig,
    allBackends: BackendConfig[],
): string | undefined {
    const allErrors = validateBackendConfig(config, allBackends);
    return allErrors[fieldName as keyof ValidationErrors];
}
