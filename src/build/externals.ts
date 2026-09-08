/**
 * Host-provided externals map.
 *
 * These packages are provided by the Construct host app at runtime
 * via window.__CONSTRUCT__['{pkg}']. Space builds externalize them
 * so they're NOT bundled into the space IIFE.
 */

/** Package name → window.__CONSTRUCT__['name'] global accessor */
export const HOST_EXTERNALS: Record<string, string> = {
  'vue': 'window.__CONSTRUCT__["vue"]',
  'vue-router': 'window.__CONSTRUCT__["vue-router"]',
  'pinia': 'window.__CONSTRUCT__["pinia"]',
  '@vueuse/core': 'window.__CONSTRUCT__["@vueuse/core"]',
  '@vueuse/integrations': 'window.__CONSTRUCT__["@vueuse/integrations"]',
  '@tauri-apps/api': 'window.__CONSTRUCT__["@tauri-apps/api"]',
  '@tauri-apps/api/core': 'window.__CONSTRUCT__["@tauri-apps/api/core"]',
  '@tauri-apps/api/path': 'window.__CONSTRUCT__["@tauri-apps/api/path"]',
  '@tauri-apps/api/event': 'window.__CONSTRUCT__["@tauri-apps/api/event"]',
  '@tauri-apps/plugin-fs': 'window.__CONSTRUCT__["@tauri-apps/plugin-fs"]',
  '@tauri-apps/plugin-shell': 'window.__CONSTRUCT__["@tauri-apps/plugin-shell"]',
  '@tauri-apps/plugin-dialog': 'window.__CONSTRUCT__["@tauri-apps/plugin-dialog"]',
  '@tauri-apps/plugin-process': 'window.__CONSTRUCT__["@tauri-apps/plugin-process"]',
  'reka-ui': 'window.__CONSTRUCT__["reka-ui"]',
  'lucide-vue-next': 'window.__CONSTRUCT__["lucide-vue-next"]',
  'date-fns': 'window.__CONSTRUCT__["date-fns"]',
  'dexie': 'window.__CONSTRUCT__["dexie"]',
  'zod': 'window.__CONSTRUCT__["zod"]',
  '@construct/sdk': 'window.__CONSTRUCT__["@construct/sdk"]',
  '@construct-space/sdk': 'window.__CONSTRUCT__["@construct-space/sdk"]',
}

/** Just the package names for rollup external */
export const EXTERNAL_PACKAGES = Object.keys(HOST_EXTERNALS)
