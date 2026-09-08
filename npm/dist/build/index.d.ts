/**
 * Build tooling for Construct spaces.
 *
 * Provides Vite preset, entry generator, manifest validation, and host externals.
 */
export { createSpaceBuildConfig } from './vite-preset.js';
export type { SpaceBuildOptions } from './vite-preset.js';
export { HOST_EXTERNALS, EXTERNAL_PACKAGES } from './externals.js';
export { generateEntry, writeEntry } from './entry-generator.js';
export { validateManifest } from './manifest-schema.js';
export type { SpaceManifest } from './manifest-schema.js';
//# sourceMappingURL=index.d.ts.map