// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

function randomUUIDPolyfill(): string {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (ch) => {
        const r = Math.random() * 16 | 0;
        const v = ch === 'x' ? r : ((r & 0x3) | 0x8);
        return v.toString(16);
    });
}

/**
 * RFC 4122 v4 UUID. Uses crypto.randomUUID when available (requires a secure context in some browsers);
 * falls back for desktop/webviews or HTTP servers where randomUUID may be missing or throw.
 */
export function newBackendUUID(): string {
    const c = typeof globalThis === 'undefined' ? undefined : globalThis.crypto;
    if (c && typeof c.randomUUID === 'function') {
        try {
            return c.randomUUID();
        } catch {
            return randomUUIDPolyfill();
        }
    }
    return randomUUIDPolyfill();
}
