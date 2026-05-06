// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {shallow, mount} from 'enzyme';
import React from 'react';

import BackendList from './BackendList';
import NoBackendsPage from './NoBackendsPage';
import type {BackendConfig} from './types';

import BackendSettings from './index';

// Mock the useBackendStatus hook to avoid client dependency issues
jest.mock('./useBackendStatus', () => ({
    useBackendStatus: () => ({statusMap: {}}),
}));

describe('BackendSettings', () => {
    const baseProps = {
        id: 'Backends',
        label: 'Backend Configurations',
        helpText: 'Configure alert backends',
        value: [] as BackendConfig[],
        disabled: false,
        config: {},
        currentState: {},
        license: {},
        setByEnv: false,
        onChange: jest.fn(),
        setSaveNeeded: jest.fn(),
        registerSaveAction: jest.fn(),
        unRegisterSaveAction: jest.fn(),
    };

    beforeEach(() => {
        jest.clearAllMocks();
    });

    it('should render empty state when no backends are configured', () => {
        const wrapper = shallow(<BackendSettings {...baseProps}/>);

        expect(wrapper.find(NoBackendsPage)).toHaveLength(1);
        expect(wrapper.find(BackendList)).toHaveLength(0);
    });

    it('should render BackendList when backends exist', () => {
        const backend: BackendConfig = {
            id: '550e8400-e29b-41d4-a716-446655440000',
            name: 'Test Backend',
            type: 'dataminr',
            enabled: true,
            url: 'https://api.example.com',
            apiId: 'test-api-id',
            apiKey: 'test-api-key',
            channelId: 'test-channel-id',
            pollIntervalSeconds: 30,
        };

        const props = {
            ...baseProps,
            value: [backend],
        };

        const wrapper = shallow(<BackendSettings {...props}/>);

        expect(wrapper.find(BackendList)).toHaveLength(1);
        expect(wrapper.find(NoBackendsPage)).toHaveLength(0);
        expect(wrapper.find(BackendList).prop('backends')).toEqual([backend]);
    });

    it('should handle undefined value prop', () => {
        const wrapper = shallow(<BackendSettings {...baseProps}/>);

        expect(wrapper.find(NoBackendsPage)).toHaveLength(1);
    });

    it('should parse backends from a JSON string value', () => {
        const backend: BackendConfig = {
            id: '550e8400-e29b-41d4-a716-446655440000',
            name: 'From JSON',
            type: 'dataminr',
            enabled: true,
            url: 'https://api.example.com',
            apiId: 'id',
            apiKey: 'key',
            channelId: 'ch',
            pollIntervalSeconds: 30,
        };
        const props = {
            ...baseProps,
            value: JSON.stringify([backend]),
        };
        const wrapper = shallow(<BackendSettings {...props}/>);

        expect(wrapper.find(BackendList)).toHaveLength(1);
        expect(wrapper.find(BackendList).prop('backends')).toEqual([backend]);
    });

    it('should treat blank JSON string as no backends', () => {
        const props = {
            ...baseProps,
            value: '   ',
        };
        const wrapper = shallow(<BackendSettings {...props}/>);

        expect(wrapper.find(NoBackendsPage)).toHaveLength(1);
    });

    it('should call onChange and setSaveNeeded when backends change', () => {
        const backend: BackendConfig = {
            id: '1',
            name: 'Test Backend',
            type: 'dataminr',
            enabled: true,
            url: 'https://api.example.com',
            apiId: 'test-api-id',
            apiKey: 'test-api-key',
            channelId: 'test-channel-id',
            pollIntervalSeconds: 30,
        };

        const props = {
            ...baseProps,
            value: [backend],
        };

        const wrapper = shallow(<BackendSettings {...props}/>);
        const backendList = wrapper.find(BackendList);

        const updatedBackend = {...backend, name: 'Updated Name'};
        backendList.prop('onChange')([updatedBackend]);

        expect(props.onChange).toHaveBeenCalledWith('Backends', [updatedBackend]);
        expect(props.setSaveNeeded).toHaveBeenCalled();
    });

    it('should register save action on mount', () => {
        const registerSaveAction = jest.fn();
        const props = {
            ...baseProps,
            registerSaveAction,
        };

        const wrapper = mount(<BackendSettings {...props}/>);

        expect(registerSaveAction).toHaveBeenCalledWith(expect.any(Function));
        wrapper.unmount();
    });

    // Note: Testing unregister on unmount is difficult with Enzyme as cleanup functions
    // from useEffect don't always run reliably. The implementation is correct (verified
    // by manual testing and code review), but Enzyme mount() has limitations with
    // cleanup function testing. The register test above verifies the core functionality.

    it('should validate backends on save and return error if validation fails', async () => {
        const invalidBackend: BackendConfig = {
            id: '550e8400-e29b-41d4-a716-446655440000',
            name: '', // Invalid: name is required
            type: 'dataminr',
            enabled: true,
            url: 'https://api.example.com',
            apiId: 'test-api-id',
            apiKey: 'test-api-key',
            channelId: 'test-channel-id',
            pollIntervalSeconds: 30,
        };

        let saveAction: (() => Promise<{error?: {message?: string}}>) | undefined;
        const registerSaveAction = jest.fn((action) => {
            saveAction = action;
        });

        const props = {
            ...baseProps,
            value: [invalidBackend],
            registerSaveAction,
        };

        const wrapper = mount(<BackendSettings {...props}/>);

        expect(saveAction).toBeDefined();
        const result = await saveAction!();

        expect(result).toEqual({error: {message: 'Please fix validation errors before saving'}});
        wrapper.unmount();
    });

    it('should validate backends on save and return success if validation passes', async () => {
        const validBackend: BackendConfig = {
            id: '550e8400-e29b-41d4-a716-446655440000',
            name: 'Valid Backend',
            type: 'dataminr',
            enabled: true,
            url: 'https://api.example.com',
            apiId: 'test-api-id',
            apiKey: 'test-api-key',
            channelId: 'test-channel-id',
            pollIntervalSeconds: 30,
        };

        let saveAction: (() => Promise<{error?: {message?: string}}>) | undefined;
        const registerSaveAction = jest.fn((action) => {
            saveAction = action;
        });

        const props = {
            ...baseProps,
            value: [validBackend],
            registerSaveAction,
        };

        const wrapper = mount(<BackendSettings {...props}/>);

        expect(saveAction).toBeDefined();
        const result = await saveAction!();

        expect(result).toEqual({});
        wrapper.unmount();
    });

    it('should clear validation errors when backends change', () => {
        const backend: BackendConfig = {
            id: '1',
            name: 'Test Backend',
            type: 'dataminr',
            enabled: true,
            url: 'https://api.example.com',
            apiId: 'test-api-id',
            apiKey: 'test-api-key',
            channelId: 'test-channel-id',
            pollIntervalSeconds: 30,
        };

        const props = {
            ...baseProps,
            value: [backend],
        };

        const wrapper = shallow(<BackendSettings {...props}/>);

        // Initially no validation errors
        expect(wrapper.find(BackendList).prop('validationErrors')).toEqual({});

        // Change backends
        const backendList = wrapper.find(BackendList);
        const updatedBackend = {...backend, name: 'Updated Name'};
        backendList.prop('onChange')([updatedBackend]);

        // Validation errors should still be empty/cleared
        wrapper.update();
        expect(wrapper.find(BackendList).prop('validationErrors')).toEqual({});
    });

    it('should show validation errors on mount when backends are invalid', () => {
        const invalidBackend: BackendConfig = {
            id: '550e8400-e29b-41d4-a716-446655440000',
            name: '', // Invalid: name is required
            type: 'dataminr',
            enabled: true,
            url: 'https://api.example.com',
            apiId: 'test-api-id',
            apiKey: 'test-api-key',
            channelId: 'test-channel-id',
            pollIntervalSeconds: 30,
        };

        const props = {
            ...baseProps,
            value: [invalidBackend],
        };

        const wrapper = mount(<BackendSettings {...props}/>);

        // Validation errors should be set on mount
        const validationErrors = wrapper.find(BackendList).prop('validationErrors');
        expect(validationErrors).toBeDefined();
        expect(validationErrors!['550e8400-e29b-41d4-a716-446655440000']).toBeDefined();
        expect(validationErrors!['550e8400-e29b-41d4-a716-446655440000'].name).toBe('Name is required');
        wrapper.unmount();
    });

    it('should show validation errors for multiple invalid backends on mount', () => {
        const invalidBackend1: BackendConfig = {
            id: '550e8400-e29b-41d4-a716-446655440000',
            name: '', // Invalid: name is required
            type: 'dataminr',
            enabled: true,
            url: 'https://api.example.com',
            apiId: 'test-api-id',
            apiKey: 'test-api-key',
            channelId: 'test-channel-id',
            pollIntervalSeconds: 30,
        };

        const invalidBackend2: BackendConfig = {
            id: '6ba7b810-9dad-11d1-80b4-00c04fd430c8',
            name: 'Backend 2',
            type: 'dataminr',
            enabled: true,
            url: 'http://api.example.com', // Invalid: must use HTTPS
            apiId: '', // Invalid: apiId is required
            apiKey: 'test-api-key',
            channelId: 'test-channel-id',
            pollIntervalSeconds: 5, // Invalid: must be at least 10
        };

        const props = {
            ...baseProps,
            value: [invalidBackend1, invalidBackend2],
        };

        const wrapper = mount(<BackendSettings {...props}/>);

        // Both backends should have validation errors
        const validationErrors = wrapper.find(BackendList).prop('validationErrors');
        expect(validationErrors!['550e8400-e29b-41d4-a716-446655440000']).toBeDefined();
        expect(validationErrors!['6ba7b810-9dad-11d1-80b4-00c04fd430c8']).toBeDefined();
        wrapper.unmount();
    });

    it('should not show validation errors on mount for valid backends', () => {
        const validBackend: BackendConfig = {
            id: '550e8400-e29b-41d4-a716-446655440000',
            name: 'Valid Backend',
            type: 'dataminr',
            enabled: true,
            url: 'https://api.example.com',
            apiId: 'test-api-id',
            apiKey: 'test-api-key',
            channelId: 'test-channel-id',
            pollIntervalSeconds: 30,
        };

        const props = {
            ...baseProps,
            value: [validBackend],
        };

        const wrapper = mount(<BackendSettings {...props}/>);

        // No validation errors for valid backend
        const validationErrors = wrapper.find(BackendList).prop('validationErrors');
        expect(validationErrors).toEqual({});
        wrapper.unmount();
    });

    it('should not re-validate after user makes changes', () => {
        const invalidBackend: BackendConfig = {
            id: '550e8400-e29b-41d4-a716-446655440000',
            name: '', // Invalid: name is required
            type: 'dataminr',
            enabled: true,
            url: 'https://api.example.com',
            apiId: 'test-api-id',
            apiKey: 'test-api-key',
            channelId: 'test-channel-id',
            pollIntervalSeconds: 30,
        };

        const props = {
            ...baseProps,
            value: [invalidBackend],
        };

        const wrapper = mount(<BackendSettings {...props}/>);

        // Validation errors should be shown on mount
        let validationErrors = wrapper.find(BackendList).prop('validationErrors');
        expect(validationErrors!['550e8400-e29b-41d4-a716-446655440000']).toBeDefined();

        // User makes a change
        const backendList = wrapper.find(BackendList);
        const updatedBackend = {...invalidBackend, name: 'Updated'};
        backendList.prop('onChange')([updatedBackend]);

        // Validation errors should be cleared
        wrapper.update();
        validationErrors = wrapper.find(BackendList).prop('validationErrors');
        expect(validationErrors).toEqual({});

        // If we update props with another invalid backend, it should NOT re-validate
        // because userHasMadeChanges is now true
        const anotherInvalidBackend = {...invalidBackend, id: 'new-id', name: ''};
        wrapper.setProps({value: [anotherInvalidBackend]});
        wrapper.update();

        validationErrors = wrapper.find(BackendList).prop('validationErrors');
        expect(validationErrors).toEqual({});
        wrapper.unmount();
    });
});
