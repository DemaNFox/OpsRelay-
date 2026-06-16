import { z } from "zod";

const configSchema = z.object({
  host: z.string().min(1).default("127.0.0.1"),
  port: z.coerce.number().int().min(1).max(65_535).default(3030),
  adminToken: z.string().min(16),
  logLevel: z.string().min(1).default("info"),
  databaseUrl: z.string().min(1).optional()
});

export type ServerConfig = z.infer<typeof configSchema>;

export function loadConfig(env: NodeJS.ProcessEnv): ServerConfig {
  return configSchema.parse({
    host: env.OPSRELAY_SERVER_HOST,
    port: env.OPSRELAY_SERVER_PORT,
    adminToken: env.OPSRELAY_ADMIN_TOKEN ?? "dev-admin-token-change-me",
    logLevel: env.OPSRELAY_LOG_LEVEL,
    databaseUrl: env.OPSRELAY_DATABASE_URL
  });
}
