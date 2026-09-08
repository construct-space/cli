/**
 * Generates entry.ts for a space from its manifest.
 *
 * The entry file exports all pages as a map so the host can load them.
 * Run this as a pre-build step or use the Vite plugin.
 *
 * Output example for space-code:
 *   window.__CONSTRUCT_SPACE_code = {
 *     pages: { '': IndexPage, 'editor': EditorPage },
 *     manifest: { ... }
 *   }
 */
export interface ManifestPage {
    path: string;
    label: string;
    icon?: string;
    default?: boolean;
    component?: string;
}
export declare function generateEntry(root: string): string;
/**
 * Write entry.ts to src/entry.ts
 */
export declare function writeEntry(root: string): void;
//# sourceMappingURL=entry-generator.d.ts.map