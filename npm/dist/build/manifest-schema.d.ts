/**
 * Space manifest JSON schema and validation.
 *
 * Every space must have a space.manifest.json that conforms to this shape.
 */
export interface SpaceManifest {
    /** Unique space identifier (lowercase, alphanumeric + hyphens) */
    id: string;
    /** Display name */
    name: string;
    /** Semver version */
    version: string;
    /** Short description */
    description: string;
    /** Author info */
    author: {
        name: string;
        email?: string;
        url?: string;
    };
    /** Lucide icon identifier */
    icon: string;
    /** Scope: company-only, project-only, or both */
    scope: 'company' | 'project' | 'both';
    /** Minimum Construct app version required */
    minConstructVersion: string;
    /** Navigation entry */
    navigation: {
        label: string;
        icon: string;
        to: string;
        order: number;
    };
    /** Pages this space provides */
    pages: Array<{
        path: string;
        label: string;
        icon?: string;
        default?: boolean;
        requiresContext?: boolean;
        component?: string;
        toolbar?: Array<{
            id: string;
            icon: string;
            label: string;
            action?: string;
            to?: string;
        }>;
    }>;
    /** Space-level toolbar items */
    toolbar?: Array<{
        id: string;
        icon: string;
        label: string;
        action?: string;
        to?: string;
    }>;
    /** Path to agent markdown (relative to space root) */
    agent?: string;
    /** Paths to skill markdowns */
    skills?: string[];
    /** Dependencies on other spaces or skills */
    dependencies?: {
        spaces?: string[];
        skills?: string[];
    };
    /** Permission declarations */
    permissions?: {
        canAccessNetwork?: boolean;
        canAccessFileSystem?: boolean;
        canRunCommands?: boolean;
    };
    /** Screenshot URLs for marketplace */
    screenshots?: string[];
    /** Search keywords */
    keywords?: string[];
    /** Whether this space is recommended for first-run install */
    recommended?: boolean;
    /** Tailwind color classes for UI identity */
    theme?: {
        color: string;
        bg: string;
    };
}
/**
 * Validate a manifest object. Returns array of error messages (empty = valid).
 */
export declare function validateManifest(manifest: unknown): string[];
//# sourceMappingURL=manifest-schema.d.ts.map