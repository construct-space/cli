/**
 * Space manifest JSON schema and validation.
 *
 * Every space must have a space.manifest.json that conforms to this shape.
 */
/**
 * Validate a manifest object. Returns array of error messages (empty = valid).
 */
export function validateManifest(manifest) {
    const errors = [];
    if (!manifest || typeof manifest !== 'object') {
        return ['Manifest must be a JSON object'];
    }
    const m = manifest;
    if (typeof m.id !== 'string' || !/^[a-z][a-z0-9-]*$/.test(m.id)) {
        errors.push('id must be a lowercase alphanumeric string starting with a letter');
    }
    if (typeof m.name !== 'string' || m.name.length === 0) {
        errors.push('name is required');
    }
    if (typeof m.version !== 'string' || !/^\d+\.\d+\.\d+/.test(m.version)) {
        errors.push('version must be semver (e.g. 1.0.0)');
    }
    if (typeof m.description !== 'string') {
        errors.push('description is required');
    }
    if (!m.author || typeof m.author !== 'object') {
        errors.push('author is required');
    }
    if (typeof m.icon !== 'string') {
        errors.push('icon is required');
    }
    if (!['company', 'project', 'both'].includes(m.scope)) {
        errors.push('scope must be "company", "project", or "both"');
    }
    if (!Array.isArray(m.pages) || m.pages.length === 0) {
        errors.push('pages must be a non-empty array');
    }
    if (!m.navigation || typeof m.navigation !== 'object') {
        errors.push('navigation is required');
    }
    return errors;
}
