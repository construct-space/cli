/**
 * Host-provided externals map.
 *
 * These packages are provided by the Construct host app at runtime
 * via window.__CONSTRUCT__['{pkg}']. Space builds externalize them
 * so they're NOT bundled into the space IIFE.
 */
/** Package name → window.__CONSTRUCT__['name'] global accessor */
export declare const HOST_EXTERNALS: Record<string, string>;
/** Just the package names for rollup external */
export declare const EXTERNAL_PACKAGES: string[];
//# sourceMappingURL=externals.d.ts.map