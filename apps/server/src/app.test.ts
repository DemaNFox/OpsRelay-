import { describe, expect, it } from "vitest";
import { PROTOCOL_VERSION } from "@opsrelay/protocol";
import { createApp } from "./app.js";

const config = {
  host: "127.0.0.1",
  port: 0,
  adminToken: "test-admin-token-123",
  logLevel: "silent"
};

function authHeaders() {
  return { authorization: `Bearer ${config.adminToken}` };
}

describe("server API", () => {
  it("rejects admin routes without a token", async () => {
    const { app } = await createApp({ config });
    const response = await app.inject({ method: "GET", url: "/v1/devices" });
    expect(response.statusCode).toBe(401);
    await app.close();
  });

  it("creates an enrollment token and enrolls a device once", async () => {
    const { app } = await createApp({ config });
    const tokenResponse = await app.inject({
      method: "POST",
      url: "/v1/enrollment-tokens",
      headers: authHeaders(),
      payload: { name: "office-pc-01" }
    });
    expect(tokenResponse.statusCode).toBe(200);
    const tokenBody = tokenResponse.json<{ ok: true; data: { token: string } }>();

    const enrollResponse = await app.inject({
      method: "POST",
      url: "/v1/agent/enroll",
      payload: {
        token: tokenBody.data.token,
        name: "office-pc-01",
        os: { type: "windows", version: "11" },
        agent_version: "0.1.0",
        protocol_version: PROTOCOL_VERSION,
        capabilities: ["shell.powershell", "shell.cmd", "reboot"]
      }
    });
    expect(enrollResponse.statusCode).toBe(200);

    const reuseResponse = await app.inject({
      method: "POST",
      url: "/v1/agent/enroll",
      payload: {
        token: tokenBody.data.token,
        name: "copy",
        os: { type: "windows", version: "11" },
        agent_version: "0.1.0",
        protocol_version: PROTOCOL_VERSION,
        capabilities: ["shell.powershell"]
      }
    });
    expect(reuseResponse.statusCode).toBe(403);
    await app.close();
  });

  it("requires explicit confirmation for reboot", async () => {
    const { app, store } = await createApp({ config });
    const token = (await store.createEnrollmentToken("linux", 3600, "admin_mvp")).token;
    const enrolled = await store.enrollDevice({
      token,
      name: "linux",
      os: { type: "linux", version: "6.8", distro: "ubuntu" },
      agent_version: "0.1.0",
      protocol_version: PROTOCOL_VERSION,
      capabilities: ["shell.sh", "reboot"]
    });
    await store.markDeviceOnline(enrolled.device.id);

    const response = await app.inject({
      method: "POST",
      url: `/v1/devices/${enrolled.device.id}/reboot`,
      headers: authHeaders(),
      payload: { confirm: false }
    });

    expect(response.statusCode).toBe(403);
    await app.close();
  });

  it("rejects unsupported shell capability", async () => {
    const { app, store } = await createApp({ config });
    const token = (await store.createEnrollmentToken("linux", 3600, "admin_mvp")).token;
    const enrolled = await store.enrollDevice({
      token,
      name: "linux",
      os: { type: "linux", version: "6.8", distro: "ubuntu" },
      agent_version: "0.1.0",
      protocol_version: PROTOCOL_VERSION,
      capabilities: ["shell.sh"]
    });
    await store.markDeviceOnline(enrolled.device.id);

    const response = await app.inject({
      method: "POST",
      url: "/v1/jobs",
      headers: authHeaders(),
      payload: {
        device_id: enrolled.device.id,
        kind: "shell.exec",
        shell: "powershell",
        command: "Get-Service"
      }
    });

    expect(response.statusCode).toBe(403);
    await app.close();
  });
});
