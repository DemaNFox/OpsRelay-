import websocket from "@fastify/websocket";
import {
  PROTOCOL_VERSION,
  agentEnrollRequestSchema,
  agentMessageSchema,
  enrollmentTokenCreateRequestSchema,
  jobCreateRequestSchema,
  sessionCreateRequestSchema
} from "@opsrelay/protocol";
import Fastify, { type FastifyInstance, type FastifyReply, type FastifyRequest } from "fastify";
import type { RawData } from "ws";
import { z } from "zod";
import { AgentConnections } from "./agentConnections.js";
import { constantTimeEqual } from "./crypto.js";
import { AppError, badRequest, unauthorized } from "./errors.js";
import type { ServerConfig } from "./config.js";
import { PostgresStore } from "./pgStore.js";
import { InMemoryStore, type Store } from "./store.js";

declare module "fastify" {
  interface FastifyRequest {
    adminId?: string;
  }
}

export interface AppDeps {
  config: ServerConfig;
  store?: Store;
  connections?: AgentConnections;
}

export interface CreatedApp {
  app: FastifyInstance;
  store: Store;
}

const deviceParamsSchema = z.object({ deviceId: z.string().min(1) });
const jobParamsSchema = z.object({ jobId: z.string().min(1) });
const sessionParamsSchema = z.object({ sessionId: z.string().min(1) });
const rebootBodySchema = z.object({
  confirm: z.boolean().default(false),
  reason: z.string().max(500).optional()
});

export async function createApp(deps: AppDeps): Promise<CreatedApp> {
  const store = deps.store ?? (deps.config.databaseUrl === undefined ? new InMemoryStore() : await PostgresStore.connect(deps.config.databaseUrl));
  const connections = deps.connections ?? new AgentConnections();
  const app = Fastify({ logger: { level: deps.config.logLevel } });
  await app.register(websocket);
  if (store instanceof PostgresStore) {
    app.addHook("onClose", async () => {
      await store.close();
    });
  }

  app.setErrorHandler((error, _request, reply) => {
    if (error instanceof AppError) {
      void reply.status(error.statusCode).send({ ok: false, error: error.apiError });
      return;
    }
    if (error instanceof z.ZodError) {
      void reply.status(400).send({
        ok: false,
        error: {
          code: "VALIDATION_FAILED",
          message: "Request validation failed",
          retryable: false,
          details: { issues: error.issues }
        }
      });
      return;
    }
    app.log.error({ err: error }, "unhandled error");
    void reply.status(500).send({
      ok: false,
      error: { code: "INTERNAL_ERROR", message: "Internal server error", retryable: true }
    });
  });

  app.addHook("preHandler", async (request) => {
    if (!request.url.startsWith("/v1/") || request.url.startsWith("/v1/agent/")) {
      return;
    }
    authenticateAdmin(request, deps.config.adminToken);
  });

  app.get("/health", async () => ({ ok: true, data: { status: "ok", protocol_version: PROTOCOL_VERSION } }));

  app.post("/v1/enrollment-tokens", async (request) => {
    const body = enrollmentTokenCreateRequestSchema.parse(request.body);
    const token = await store.createEnrollmentToken(body.name, body.ttl_sec, request.adminId ?? "admin_mvp");
    return { ok: true, data: { ...token, name: body.name } };
  });

  app.post("/v1/agent/enroll", async (request) => {
    const body = agentEnrollRequestSchema.parse(request.body);
    const enrolled = await store.enrollDevice(body);
    return {
      ok: true,
      data: {
        device_id: enrolled.device.id,
        device_secret: enrolled.device_secret,
        server_time: new Date().toISOString(),
        protocol_version: PROTOCOL_VERSION
      }
    };
  });

  app.get("/v1/agent/connect", { websocket: true }, (socket, request) => {
    void (async () => {
      const query = z.object({ device_id: z.string().min(1), device_secret: z.string().min(1) }).parse(request.query);
      const device = await store.authenticateDevice(query.device_id, query.device_secret);
      connections.register(device.id, socket);
      await store.markDeviceOnline(device.id);
      for (const queuedJob of await store.queuedJobsForDevice(device.id)) {
        connections.dispatch(await store.markJobDispatched(queuedJob.id));
      }

      socket.on("message", (rawMessage: RawData) => {
        void (async () => {
      try {
        const parsed = agentMessageSchema.parse(JSON.parse(rawMessage.toString()));
        if (parsed.type === "heartbeat") {
          await store.updateHeartbeat(parsed.device_id, parsed.agent_version);
          return;
        }
        if (parsed.type === "hello") {
          await store.updateHeartbeat(parsed.device_id, parsed.agent_version);
          return;
        }
        await store.completeJob(parsed.result);
      } catch (error) {
        app.log.warn({ err: error }, "invalid agent message");
      }
        })();
    });

      socket.on("close", () => {
        void (async () => {
      connections.unregister(device.id, socket);
      await store.markDeviceOffline(device.id);
        })();
    });
    })().catch((error: unknown) => {
      app.log.warn({ err: error }, "agent connection rejected");
      socket.close();
    });
  });

  app.get("/v1/devices", async (request) => ({ ok: true, data: await store.listDevices(request.adminId ?? "admin_mvp") }));

  app.get("/v1/devices/:deviceId", async (request) => {
    const params = deviceParamsSchema.parse(request.params);
    return { ok: true, data: await store.getDevice(params.deviceId) };
  });

  app.post("/v1/sessions", async (request) => {
    const body = sessionCreateRequestSchema.parse(request.body);
    return { ok: true, data: await store.createSession(body.device_id, body.name, request.adminId ?? "admin_mvp") };
  });

  app.get("/v1/sessions/:sessionId", async (request) => {
    const params = sessionParamsSchema.parse(request.params);
    return { ok: true, data: await store.getSession(params.sessionId) };
  });

  app.post("/v1/jobs", async (request) => {
    const body = jobCreateRequestSchema.parse(request.body);
    const job = await store.createJob(body, request.adminId ?? "admin_mvp");
    const dispatched = connections.dispatch(await store.markJobDispatched(job.id));
    return { ok: true, data: { ...(await store.getJob(job.id)), dispatched } };
  });

  app.get("/v1/jobs/:jobId", async (request) => {
    const params = jobParamsSchema.parse(request.params);
    return { ok: true, data: await store.getJob(params.jobId) };
  });

  app.get("/v1/jobs/:jobId/result", async (request) => {
    const params = jobParamsSchema.parse(request.params);
    return { ok: true, data: await store.getJobResult(params.jobId) };
  });

  app.post("/v1/devices/:deviceId/reboot", async (request) => {
    const params = deviceParamsSchema.parse(request.params);
    const body = rebootBodySchema.parse(request.body);
    const job = await store.createRebootJob(params.deviceId, body.confirm, body.reason, request.adminId ?? "admin_mvp");
    const dispatched = connections.dispatch(await store.markJobDispatched(job.id));
    return { ok: true, data: { ...(await store.getJob(job.id)), dispatched } };
  });

  app.get("/v1/audit", async () => ({ ok: true, data: await store.listAudit() }));

  return { app, store };
}

function authenticateAdmin(request: FastifyRequest, expectedToken: string): void {
  const header = request.headers.authorization;
  if (header === undefined || !header.startsWith("Bearer ")) {
    throw unauthorized();
  }
  const token = header.slice("Bearer ".length);
  if (!constantTimeEqual(token, expectedToken)) {
    throw unauthorized("Admin token is invalid");
  }
  request.adminId = "admin_mvp";
}

export async function closeApp(reply: FastifyReply): Promise<void> {
  await reply;
}
