#!/usr/bin/env node
import { parseCli } from "./args.js";
import { executeCommand } from "./commands.js";
import { loadCliConfig } from "./config.js";
import { renderError, renderSuccess } from "./output.js";

async function main(): Promise<void> {
  const parsed = parseCli(process.argv.slice(2));
  const result = await executeCommand(parsed.command, parsed.options, loadCliConfig(process.env));
  process.stdout.write(renderSuccess(result, parsed.options.json));
}

main().catch((error: unknown) => {
  let asJson = false;
  try {
    asJson = parseCli(process.argv.slice(2)).options.json;
  } catch {
    asJson = process.argv.includes("--json");
  }
  const rendered = renderError(error, asJson);
  process.stderr.write(rendered.output);
  process.exitCode = rendered.exitCode;
});
