/**
 * Host API exports for @construct/sdk auto-import.
 *
 * These are all the stores, composables, and utilities that the Construct
 * host app provides to space IIFE bundles at runtime via:
 *   window.__CONSTRUCT__['@construct/sdk']
 *
 * unplugin-auto-import uses this list to generate:
 *   import { useProjectStore } from '@construct/sdk'
 * which Rollup externalizes to:
 *   window.__CONSTRUCT__["@construct/sdk"].useProjectStore
 */
/** UI components provided by the host via @construct/sdk */
export declare const HOST_UI_COMPONENTS: string[];
export declare const HOST_API_EXPORTS: string[];
//# sourceMappingURL=host-api.d.ts.map