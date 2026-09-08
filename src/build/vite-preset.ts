/**
 * Vite build preset for Construct spaces.
 *
 * Usage in a space's vite.config.ts:
 *   import { createSpaceBuildConfig } from 'construct/vite'
 *   export default createSpaceBuildConfig('code')
 */

import { defineConfig, type UserConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { resolve } from 'path'
import { readFileSync, writeFileSync, mkdirSync, existsSync, readdirSync, statSync, copyFileSync } from 'fs'
import { createHash } from 'crypto'
import { HOST_EXTERNALS, EXTERNAL_PACKAGES } from './externals.ts'
import { HOST_API_EXPORTS, HOST_COMPONENTS } from './host-api.ts'

export interface SpaceBuildOptions {
  /** Space ID (e.g. 'code', 'design') — read from manifest if omitted */
  id?: string
  /** Root directory of the space (defaults to process.cwd()) */
  root?: string
  /** Additional Vite plugins */
  plugins?: UserConfig['plugins']
}

/**
 * Reads space.manifest.json and generates a Vite config that:
 * - Outputs a single IIFE bundle: space-{id}.iife.js
 * - Externalizes all host-provided packages
 * - Bundles space-specific dependencies
 * - Extracts CSS to space-{id}.css
 * - Generates entry from manifest pages
 */
export function createSpaceBuildConfig(options: SpaceBuildOptions | string = {}): UserConfig {
  const opts = typeof options === 'string' ? { id: options } : options
  const root = opts.root ?? process.cwd()

  // Read manifest
  const manifestPath = resolve(root, 'space.manifest.json')
  let manifest: Record<string, any>
  try {
    manifest = JSON.parse(readFileSync(manifestPath, 'utf-8'))
  } catch {
    throw new Error(`Cannot read space.manifest.json at ${manifestPath}`)
  }

  const spaceId = opts.id ?? manifest.id
  if (!spaceId) {
    throw new Error('Space ID is required (set in space.manifest.json or pass as option)')
  }

  const safeId = spaceId.replace(/[^a-zA-Z0-9]/g, '_').toUpperCase()
  const globalName = `__CONSTRUCT_SPACE_${safeId}`

  return defineConfig({
    root,
    plugins: [
      vue(),
      tailwindcss(),
      AutoImport({
        imports: [
          'vue',
          'vue-router',
          'pinia',
          '@vueuse/core',
          {
            '@construct/sdk': HOST_API_EXPORTS,
          },
        ],
        // SDK provides its own versions of these — ignore the @vueuse/core ones
        ignore: ['useDateFormat', 'useStorage'],
        dts: false,
      }),
      Components({
        // Scan local components/ directory in the space
        dirs: [
          resolve(root, 'components'),
          resolve(root, 'src/components'),
        ],
        resolvers: [
          (componentName) => {
            if (HOST_COMPONENTS.includes(componentName)) {
              return { name: componentName, from: '@construct/sdk' }
            }
          },
        ],
        dts: false,
      }),
      ...(opts.plugins ?? []),
      // Post-build: copy manifest to dist with build metadata
      {
        name: 'construct-space-manifest',
        closeBundle() {
          const distDir = resolve(root, 'dist')
          const bundlePath = resolve(distDir, `space-${spaceId}.iife.js`)

          let checksum = ''
          let size = 0
          try {
            const bundleContent = readFileSync(bundlePath)
            checksum = createHash('sha256').update(bundleContent).digest('hex')
            size = bundleContent.length
          } catch {
            // Bundle may not exist yet in watch mode
          }

          const distManifest = {
            ...manifest,
            build: {
              checksum,
              size,
              hostApiVersion: '0.2.0',
              builtAt: new Date().toISOString(),
            },
          }

          mkdirSync(distDir, { recursive: true })
          writeFileSync(
            resolve(distDir, 'manifest.json'),
            JSON.stringify(distManifest, null, 2),
          )

          // Widget manifest generation is handled by the Go CLI (buildSpace)
          // Agent bundling is handled by the Go CLI (BundleAgentDir)
          // which produces a single config.agent at dist root.
        },
      },
    ],
    build: {
      outDir: 'dist',
      emptyOutDir: true,
      lib: {
        entry: resolve(root, 'src/entry.ts'),
        name: globalName,
        formats: ['iife'],
        fileName: () => `space-${spaceId}.iife.js`,
      },
      rollupOptions: {
        external: EXTERNAL_PACKAGES,
        output: {
          globals: HOST_EXTERNALS,
          assetFileNames: `space-${spaceId}.[ext]`,
          // Ensure single chunk
          inlineDynamicImports: true,
        },
      },
      // Minify for production
      minify: 'esbuild',
      // Generate sourcemap for debugging
      sourcemap: false,
      // CSS extraction
      cssCodeSplit: false,
    },
    resolve: {
      alias: (() => {
        // Spaces with src/pages/ use src/ as root, others use project root
        const srcRoot = existsSync(resolve(root, 'src/pages')) ? resolve(root, 'src') : root
        return { '@': srcRoot, '~': srcRoot }
      })(),
    },
  })
}

/** Recursively copy a directory */
function copyDirRecursive(src: string, dest: string): void {
  mkdirSync(dest, { recursive: true })
  for (const entry of readdirSync(src)) {
    const srcPath = resolve(src, entry)
    const destPath = resolve(dest, entry)
    if (statSync(srcPath).isDirectory()) {
      copyDirRecursive(srcPath, destPath)
    } else {
      copyFileSync(srcPath, destPath)
    }
  }
}
