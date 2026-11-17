# Windows Service Packaging

MVP GUI binary name: `opsrelay-agent-windows-amd64.exe`.

The GUI exe is intended for a normal user:

```text
1. User double-clicks the exe.
2. User enters server URL and one-time enrollment token.
3. User clicks "Зарегистрировать и подключить".
4. Window displays Device ID, a colored connection indicator and the time of the last sent heartbeat.
5. Administrator sees the device as online and selects it from the CLI.
```

For technical support/debugging, build a console binary:

```powershell
.\scripts\build-windows.ps1 -Console -Output dist\opsrelay-agent-windows-amd64-console.exe
```

Expected production paths:

```text
Binary: C:\Program Files\OpsRelayAgent\opsrelay-agent.exe
Config: C:\ProgramData\OpsRelayAgent\config.json
Logs:   C:\ProgramData\OpsRelayAgent\logs\
Service name: OpsRelayAgent
```

The initial code keeps service packaging separate from runtime logic. Do not add silent or hidden install behavior.

`opsrelay.ico` is embedded in the GUI exe by `scripts/build-windows.ps1`; `opsrelay-icon.png` is its source preview.
