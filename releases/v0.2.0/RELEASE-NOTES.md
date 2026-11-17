# OpsRelay v0.2.0

## Changes

- Compact Windows client (620 × 410) with a consistent dark blue and mint palette, connection indicator, inline errors, and an updated icon.
- Saved client identity now waits for the device owner to choose **Подключить сохранённое** before connecting.
- Create a one-time client enrollment token with `pnpm operator token <device-name> --json`.
- Run the administrator CLI on the server host, including from the deployed Docker image, while keeping authentication, policy checks, and the audit trail on the server API.

The protocol format remains at version 0.1.0. Replace the Windows executable with this release to update an existing client; its saved device identity remains in the same configuration location. Keep the client window open while support is connected.

The server deployment steps are in [the deployment runbook](https://github.com/DemaNFox/OpsRelay-/blob/v0.2.0/docs/18-deployment-runbook.md). Verify `/health` after restarting the server service.
