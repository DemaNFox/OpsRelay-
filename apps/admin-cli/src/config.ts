import type { CliConfig } from "./types.js";

export function loadCliConfig(env: NodeJS.ProcessEnv): CliConfig {
  return {
    serverUrl: (env.OPSRELAY_SERVER_URL ?? "http://127.0.0.1:3030").replace(/\/$/, ""),
    adminToken: env.OPSRELAY_ADMIN_TOKEN ?? "dev-admin-token-change-me",
    sessionDir: env.OPSRELAY_SESSION_DIR ?? ".sessions"
  };
}
