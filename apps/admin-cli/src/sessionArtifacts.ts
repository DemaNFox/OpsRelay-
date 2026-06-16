import { mkdir, readdir, writeFile, appendFile } from "node:fs/promises";
import { join } from "node:path";
import type { Job, JobResult, Session } from "@opsrelay/protocol";
import type { CliConfig } from "./types.js";

export async function saveSession(session: Session, config: CliConfig): Promise<void> {
  const root = join(config.sessionDir, session.id);
  await mkdir(root, { recursive: true });
  await mkdir(join(root, "commands"), { recursive: true });
  await mkdir(join(root, "results"), { recursive: true });
  await mkdir(join(root, "reports"), { recursive: true });
  await writeFile(join(root, "session.json"), `${JSON.stringify(session, null, 2)}\n`, "utf8");
  await appendFile(join(root, "transcript.md"), `# ${session.name}\n\nStarted: ${session.created_at}\n\n`, "utf8");
}

export async function saveJobArtifact(job: Job, result: JobResult, config: CliConfig): Promise<string> {
  const sessionId = job.session_id ?? job.id;
  const root = join(config.sessionDir, sessionId);
  await mkdir(join(root, "commands"), { recursive: true });
  await mkdir(join(root, "results"), { recursive: true });
  const sequence = await nextSequence(join(root, "results"));
  const shell = job.shell ?? "job";
  await writeFile(join(root, "commands", `${sequence}.command.${extensionForShell(shell)}`), `${job.command ?? job.kind}\n`, "utf8");
  await writeFile(join(root, "results", `${sequence}.stdout.txt`), result.stdout, "utf8");
  await writeFile(join(root, "results", `${sequence}.stderr.txt`), result.stderr, "utf8");
  await writeFile(join(root, "results", `${sequence}.meta.json`), `${JSON.stringify({ job, result }, null, 2)}\n`, "utf8");
  await appendFile(
    join(root, "transcript.md"),
    `## ${sequence} ${job.kind}\n\n- job: ${job.id}\n- status: ${result.status}\n- exit_code: ${result.exit_code ?? "null"}\n\n`,
    "utf8"
  );
  return root;
}

async function nextSequence(resultsDir: string): Promise<string> {
  const entries = await readdir(resultsDir).catch(() => []);
  const count = entries.filter((entry) => entry.endsWith(".meta.json")).length + 1;
  return String(count).padStart(4, "0");
}

function extensionForShell(shell: string): string {
  if (shell === "powershell") {
    return "ps1";
  }
  if (shell === "cmd") {
    return "cmd";
  }
  return "sh";
}
