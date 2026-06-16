OpsRelay Agent v0.1.0

This package contains the Windows client GUI agent.

File:
  opsrelay-agent-windows-amd64.exe

Client flow:
  1. Double-click opsrelay-agent-windows-amd64.exe.
  2. Enter the server URL provided by the administrator.
  3. Paste the one-time enrollment token provided by the administrator.
  4. Confirm or edit the device name.
  5. Click "Enroll and connect".
  6. Keep the window open while support is connected.

After enrollment:
  - Device identity is stored in C:\ProgramData\OpsRelayAgent\config.json.
  - Later launches reconnect automatically.
  - The enrollment token is not saved and cannot be reused.

Status fields:
  - Status: local agent connection state.
  - Server check: server /health reachability.

Security notes:
  - Use only on devices where the owner has explicitly approved support access.
  - The agent is visible and foreground-only in this MVP.
  - Closing the window stops the client agent.
