import type { JobForAgent, ServerJobMessage } from "@opsrelay/protocol";
import { WebSocket } from "ws";

export class AgentConnections {
  private readonly sockets = new Map<string, WebSocket>();

  public register(deviceId: string, socket: WebSocket): void {
    this.sockets.set(deviceId, socket);
  }

  public unregister(deviceId: string, socket: WebSocket): void {
    if (this.sockets.get(deviceId) === socket) {
      this.sockets.delete(deviceId);
    }
  }

  public dispatch(job: JobForAgent): boolean {
    const socket = this.sockets.get(job.device_id);
    if (socket === undefined || socket.readyState !== WebSocket.OPEN) {
      return false;
    }
    const message: ServerJobMessage = { type: "job", job };
    socket.send(JSON.stringify(message));
    return true;
  }
}
