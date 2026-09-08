#!/usr/bin/env node
/**
 * construct CLI
 *
 * Commands:
 *   construct space scaffold <name>   Create a new space from template
 *   construct space build             Build the current space into an IIFE bundle
 *   construct space dev               Build in watch mode
 *   construct space validate           Validate space.manifest.json
 *   construct space publish            Tag + push to trigger CI release
 *   construct version                  Show CLI version
 */
import { scaffold } from './commands/scaffold.js';
import { build } from './commands/build.js';
import { validate } from './commands/validate.js';
import { dev } from './commands/dev.js';
import { run } from './commands/run.js';
import { publish } from './commands/publish.js';
const args = process.argv.slice(2);
const command = args[0];
const subcommand = args[1];
async function main() {
    if (command === 'version' || command === '--version' || command === '-v') {
        console.log('construct v0.1.0');
        return;
    }
    if (command === 'space') {
        switch (subcommand) {
            case 'scaffold':
            case 'new':
            case 'create':
                await scaffold(args.slice(2));
                break;
            case 'build':
                await build(args.slice(2));
                break;
            case 'dev':
                await dev(args.slice(2));
                break;
            case 'run':
                await run(args.slice(2));
                break;
            case 'validate':
                await validate(args.slice(2));
                break;
            case 'publish':
                await publish(args.slice(2));
                break;
            default:
                printSpaceHelp();
        }
        return;
    }
    if (command === 'help' || command === '--help' || !command) {
        printHelp();
        return;
    }
    console.error(`Unknown command: ${command}`);
    console.error('Run "construct help" for usage.');
    process.exit(1);
}
function printHelp() {
    console.log(`
construct — Construct Space CLI

Usage:
  construct space scaffold <name>   Create a new space from template
  construct space build             Build space into IIFE bundle
  construct space dev               Watch build + auto-install to ~/.construct/spaces/
  construct space run               Install built space for testing
  construct space validate          Validate space.manifest.json
  construct space publish           Tag and push for CI release
  construct version                 Show CLI version
  construct help                    Show this help
`);
}
function printSpaceHelp() {
    console.log(`
construct space — Space management commands

Usage:
  construct space scaffold <name>   Create a new space from template
  construct space build             Build space into IIFE bundle
  construct space dev               Watch build + auto-install to ~/.construct/spaces/
  construct space run               Install built space for testing
  construct space validate          Validate space.manifest.json
  construct space publish           Tag and push for CI release
`);
}
main().catch((err) => {
    console.error(err);
    process.exit(1);
});
