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
import { readFileSync, writeFileSync, mkdirSync, existsSync } from 'fs';
import { resolve, dirname } from 'path';
export function generateEntry(root) {
    const manifestPath = resolve(root, 'space.manifest.json');
    const manifest = JSON.parse(readFileSync(manifestPath, 'utf-8'));
    const pages = manifest.pages ?? [];
    // Detect whether pages live in src/pages/ or root pages/
    // Entry is written to src/entry.ts, so root pages need ../pages/ prefix
    const hasSrcPages = existsSync(resolve(root, 'src/pages'));
    const pagePrefix = hasSrcPages ? './' : '../';
    const imports = [];
    const pageMap = [];
    for (const page of pages) {
        // Derive component path from page.path
        const componentPath = page.component ?? (page.path === '' ? 'pages/index.vue' : `pages/${page.path}.vue`);
        const varName = page.path === '' ? 'IndexPage' : `${capitalize(page.path)}Page`;
        imports.push(`import ${varName} from '${pagePrefix}${componentPath}'`);
        pageMap.push(`  '${page.path}': ${varName},`);
    }
    const code = `// Auto-generated entry — do not edit manually
// Generated from space.manifest.json
${imports.join('\n')}

const spaceExport = {
  pages: {
${pageMap.join('\n')}
  },
}

export default spaceExport
`;
    return code;
}
function capitalize(s) {
    return s.charAt(0).toUpperCase() + s.slice(1).replace(/-(\w)/g, (_, c) => c.toUpperCase());
}
/**
 * Write entry.ts to src/entry.ts
 */
export function writeEntry(root) {
    const code = generateEntry(root);
    const entryPath = resolve(root, 'src/entry.ts');
    mkdirSync(dirname(entryPath), { recursive: true });
    writeFileSync(entryPath, code, 'utf-8');
}
