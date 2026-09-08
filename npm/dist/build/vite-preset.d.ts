/**
 * Vite build preset for Construct spaces.
 *
 * Usage in a space's vite.config.ts:
 *   import { createSpaceBuildConfig } from 'construct/vite'
 *   export default createSpaceBuildConfig('code')
 */
import { type UserConfig } from 'vite';
export interface SpaceBuildOptions {
    /** Space ID (e.g. 'code', 'design') — read from manifest if omitted */
    id?: string;
    /** Root directory of the space (defaults to process.cwd()) */
    root?: string;
    /** Additional Vite plugins */
    plugins?: UserConfig['plugins'];
}
/**
 * Reads space.manifest.json and generates a Vite config that:
 * - Outputs a single IIFE bundle: space-{id}.iife.js
 * - Externalizes all host-provided packages
 * - Bundles space-specific dependencies
 * - Extracts CSS to space-{id}.css
 * - Generates entry from manifest pages
 */
export declare function createSpaceBuildConfig(options?: SpaceBuildOptions | string): UserConfig;
//# sourceMappingURL=vite-preset.d.ts.map