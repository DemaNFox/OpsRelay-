import { CliError, type CliOptions, type ParsedCli } from "./types.js";

export function parseCli(argv: string[]): ParsedCli {
  const command: string[] = [];
  const options: CliOptions = {
    json: false,
    save: false,
    sessionId: null,
    timeoutSec: null,
    confirm: false,
    name: null,
    reason: null
  };

  for (let index = 0; index < argv.length; index += 1) {
    const value = argv[index];
    if (value === undefined) {
      continue;
    }
    if (value === "--json") {
      options.json = true;
      continue;
    }
    if (value === "--save") {
      options.save = true;
      continue;
    }
    if (value === "--confirm") {
      options.confirm = true;
      continue;
    }
    if (value === "--session") {
      options.sessionId = readOptionValue(argv, index, "--session");
      index += 1;
      continue;
    }
    if (value === "--timeout") {
      const parsed = Number(readOptionValue(argv, index, "--timeout"));
      if (!Number.isInteger(parsed) || parsed < 1) {
        throw new CliError("INVALID_ARGUMENTS", "--timeout must be a positive integer", 2);
      }
      options.timeoutSec = parsed;
      index += 1;
      continue;
    }
    if (value === "--name") {
      options.name = readOptionValue(argv, index, "--name");
      index += 1;
      continue;
    }
    if (value === "--reason") {
      options.reason = readOptionValue(argv, index, "--reason");
      index += 1;
      continue;
    }
    command.push(value);
  }

  return { command, options };
}

function readOptionValue(argv: string[], index: number, option: string): string {
  const value = argv[index + 1];
  if (value === undefined || value.startsWith("--")) {
    throw new CliError("INVALID_ARGUMENTS", `${option} requires a value`, 2);
  }
  return value;
}
