//go:build windows

package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/opsrelay/agent/internal/config"
	"github.com/opsrelay/agent/internal/core"
	"github.com/opsrelay/agent/internal/executor"
	"github.com/opsrelay/agent/internal/platform"
	"github.com/opsrelay/agent/internal/transport"
)

type guiState struct {
	window        *walk.MainWindow
	serverEdit    *walk.LineEdit
	tokenEdit     *walk.LineEdit
	nameEdit      *walk.LineEdit
	deviceEdit    *walk.LineEdit
	statusLabel   *walk.Label
	statusDot     *walk.Label
	activityLabel *walk.Label
	serverLabel   *walk.Label
	connectBtn    *walk.PushButton
	savedBtn      *walk.PushButton
	stopBtn       *walk.PushButton
	cancel        context.CancelFunc
	runID         uint64
	logger        *slog.Logger
}

func runWindowsGUI(logger *slog.Logger) error {
	state := &guiState{logger: logger}
	cfgPath := config.DefaultPath()
	prefsPath := config.DefaultPrefsPath()
	existing, hasConfig := loadExistingConfig(cfgPath)
	prefs, hasPrefs := loadPrefs(prefsPath)

	serverURL := "https://opsrelay.example.com"
	deviceID := "not enrolled"
	deviceName := os.Getenv("COMPUTERNAME")
	if hasConfig {
		serverURL = existing.ServerURL
		deviceID = existing.DeviceID
		deviceName = existing.Name
	} else if hasPrefs {
		if prefs.ServerURL != "" {
			serverURL = prefs.ServerURL
		}
		if prefs.Name != "" {
			deviceName = prefs.Name
		}
	}
	if deviceName == "" {
		deviceName = "windows-client"
	}

	go func() {
		if hasConfig {
			state.waitAndConnectSaved(cfgPath)
		}
	}()

	_, err := MainWindow{
		AssignTo: &state.window,
		Title:    "OpsRelay Agent",
		Size:     Size{Width: 600, Height: 400},
		MinSize:  Size{Width: 600, Height: 400},
		Layout:   VBox{MarginsZero: false},
		Children: []Widget{
			Label{Text: "OpsRelay  ·  Device connection", Font: Font{PointSize: 15, Bold: true}},
			Label{Text: "Your administrator can connect only while this window is open."},
			Composite{Layout: HBox{}, Children: []Widget{
				Label{AssignTo: &state.statusDot, Text: "●", Font: Font{PointSize: 20}, TextColor: statusColor(initialStatus(hasConfig))},
				Label{AssignTo: &state.statusLabel, Text: friendlyStatus(initialStatus(hasConfig)), Font: Font{PointSize: 12, Bold: true}},
			}},
			Label{AssignTo: &state.activityLabel, Text: statusActivity(initialStatus(hasConfig))},
			Composite{
				Layout: Grid{Columns: 2},
				Children: []Widget{
					Label{Text: "Server URL"},
					LineEdit{
						AssignTo: &state.serverEdit,
						Text:     serverURL,
						OnTextChanged: func() {
							state.savePrefs(prefsPath)
						},
					},
					Label{Text: "Enrollment token"},
					LineEdit{AssignTo: &state.tokenEdit, PasswordMode: true},
					Label{Text: "Device name"},
					LineEdit{
						AssignTo: &state.nameEdit,
						Text:     deviceName,
						OnTextChanged: func() {
							state.savePrefs(prefsPath)
						},
					},
					Label{Text: "Device ID"},
					LineEdit{AssignTo: &state.deviceEdit, Text: deviceID, ReadOnly: true},
					Label{Text: "Server check"},
					Label{AssignTo: &state.serverLabel, Text: "not checked"},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					PushButton{
						Text:     "Enroll and connect",
						AssignTo: &state.connectBtn,
						OnClicked: func() {
							state.enrollAndConnect(cfgPath)
						},
					},
					PushButton{
						Text:     "Connect with saved config",
						AssignTo: &state.savedBtn,
						Enabled:  hasConfig,
						OnClicked: func() {
							state.connectSaved(cfgPath)
						},
					},
					PushButton{
						Text: "Check server",
						OnClicked: func() {
							state.checkServer()
						},
					},
					PushButton{
						Text:     "Stop",
						AssignTo: &state.stopBtn,
						Enabled:  false,
						OnClicked: func() {
							state.stop()
						},
					},
				},
			},
			Label{Text: "You can stop the connection at any time. Closing this window also disconnects."},
		},
	}.Run()
	if state.cancel != nil {
		state.cancel()
	}
	return err
}

func (state *guiState) enrollAndConnect(cfgPath string) {
	serverURL := strings.TrimSpace(state.serverEdit.Text())
	token := strings.TrimSpace(state.tokenEdit.Text())
	name := strings.TrimSpace(state.nameEdit.Text())
	if serverURL == "" || token == "" || name == "" {
		state.showError("Server URL, enrollment token and device name are required.")
		return
	}
	state.savePrefs(config.DefaultPrefsPath())

	state.setStatus("enrolling")
	state.setServerStatus("checking")
	state.connectBtn.SetEnabled(false)
	go state.finishEnrollment(cfgPath, serverURL, token, name)
}

func (state *guiState) finishEnrollment(cfgPath, serverURL, token, name string) {
	client := transport.NewClient(serverURL)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cfg, err := client.Enroll(ctx, transport.EnrollRequest{
		Token:           token,
		Name:            name,
		OS:              platform.OSInfo(),
		AgentVersion:    core.AgentVersion,
		ProtocolVersion: core.ProtocolVersion,
		Capabilities:    platform.Capabilities(),
	})
	if err != nil {
		state.window.Synchronize(func() {
			state.setStatus("enrollment failed")
			state.setServerStatus("enrollment failed")
			state.connectBtn.SetEnabled(true)
			state.showError(err.Error())
		})
		return
	}
	cfg.ServerURL = serverURL
	cfg.Name = name
	if err := config.Save(cfgPath, cfg); err != nil {
		state.window.Synchronize(func() {
			state.setStatus("config save failed")
			state.connectBtn.SetEnabled(true)
			state.showError(err.Error())
		})
		return
	}
	state.window.Synchronize(func() {
		state.deviceEdit.SetText(cfg.DeviceID)
		state.tokenEdit.SetText("")
		state.savedBtn.SetEnabled(true)
		state.setServerStatus("reachable")
		state.startAgent(cfg)
	})
}

func (state *guiState) connectSaved(cfgPath string) {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		state.showError(fmt.Sprintf("Saved config was not found or is invalid: %s", err.Error()))
		return
	}
	state.serverEdit.SetText(cfg.ServerURL)
	state.nameEdit.SetText(cfg.Name)
	state.deviceEdit.SetText(cfg.DeviceID)
	state.startAgent(cfg)
}

func (state *guiState) waitAndConnectSaved(cfgPath string) {
	for state.window == nil {
		time.Sleep(50 * time.Millisecond)
	}
	state.window.Synchronize(func() {
		if walk.MsgBox(state.window, "OpsRelay connection", "Allow this device to connect to the OpsRelay server now? You can stop it at any time.", walk.MsgBoxYesNo|walk.MsgBoxIconQuestion) == walk.DlgCmdYes {
			state.connectSaved(cfgPath)
		}
	})
}

func (state *guiState) startAgent(cfg config.Config) {
	state.stop()
	state.runID++
	runID := state.runID
	ctx, cancel := context.WithCancel(context.Background())
	state.cancel = cancel
	state.connectBtn.SetEnabled(false)
	state.savedBtn.SetEnabled(false)
	state.stopBtn.SetEnabled(true)
	state.setStatus("connecting")

	runner := core.NewAgent(
		cfg,
		transport.NewClient(cfg.ServerURL),
		executor.NewShellRunner(),
		platform.NewRebooter(cfg.AllowReboot),
		platform.OSInfo(),
		platform.Capabilities(),
		state.logger,
	)
	runner.SetStatusHandler(func(status string) {
		state.window.Synchronize(func() {
			if state.runID != runID {
				return
			}
			state.setStatus(status)
			if status == "connected" || status == "heartbeat" {
				state.setServerStatus("reachable")
			}
			if status == "reconnecting" {
				state.setServerStatus("connection lost")
			}
		})
	})

	go func() {
		err := runner.Run(ctx)
		if err != nil && ctx.Err() == nil {
			state.window.Synchronize(func() {
				if state.runID != runID {
					return
				}
				state.setStatus("stopped with error")
				state.showError(err.Error())
			})
		}
	}()
}

func (state *guiState) checkServer() {
	serverURL := state.serverEdit.Text()
	if serverURL == "" {
		state.setServerStatus("missing server URL")
		return
	}
	state.savePrefs(config.DefaultPrefsPath())
	state.setServerStatus("checking")
	go func() {
		client := &http.Client{Timeout: 5 * time.Second}
		response, err := client.Get(strings.TrimRight(serverURL, "/") + "/health")
		if err != nil {
			state.setServerStatus("not reachable")
			return
		}
		defer response.Body.Close()
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			state.setServerStatus("reachable")
			return
		}
		state.setServerStatus(fmt.Sprintf("HTTP %d", response.StatusCode))
	}()
}

func (state *guiState) savePrefs(path string) {
	if state.serverEdit == nil || state.nameEdit == nil {
		return
	}
	if err := config.SavePrefs(path, config.Prefs{
		ServerURL: state.serverEdit.Text(),
		Name:      state.nameEdit.Text(),
	}); err != nil {
		state.logger.Warn("failed to save gui prefs", slog.String("error", err.Error()))
	}
}

func (state *guiState) stop() {
	state.runID++
	if state.cancel != nil {
		state.cancel()
		state.cancel = nil
	}
	if state.connectBtn != nil {
		state.connectBtn.SetEnabled(true)
	}
	if state.savedBtn != nil {
		state.savedBtn.SetEnabled(state.deviceEdit != nil && state.deviceEdit.Text() != "not enrolled")
	}
	if state.stopBtn != nil {
		state.stopBtn.SetEnabled(false)
	}
	state.setStatus("stopped")
}

func (state *guiState) setStatus(status string) {
	if state.window == nil || state.statusLabel == nil || state.statusDot == nil {
		return
	}
	state.window.Synchronize(func() {
		state.statusLabel.SetText(friendlyStatus(status))
		state.statusDot.SetTextColor(statusColor(status))
		if state.activityLabel != nil {
			state.activityLabel.SetText(statusActivity(status))
		}
	})
}

func friendlyStatus(status string) string {
	switch status {
	case "connected", "heartbeat":
		return "Connected"
	case "connecting":
		return "Connecting…"
	case "reconnecting":
		return "Connection lost — retrying…"
	case "enrolling":
		return "Enrolling device…"
	case "enrollment failed":
		return "Enrollment failed"
	case "config save failed":
		return "Could not save device settings"
	case "saved config found":
		return "Ready to connect"
	case "not enrolled":
		return "Enrollment needed"
	case "stopped with error":
		return "Connection stopped with error"
	default:
		return "Disconnected"
	}
}

func statusColor(status string) walk.Color {
	switch status {
	case "connected", "heartbeat":
		return walk.RGB(38, 165, 81)
	case "connecting", "enrolling", "reconnecting":
		return walk.RGB(229, 151, 30)
	case "enrollment failed", "config save failed", "stopped with error":
		return walk.RGB(200, 58, 58)
	default:
		return walk.RGB(140, 150, 160)
	}
}

func statusActivity(status string) string {
	switch status {
	case "connected":
		return "Connection established. Waiting for administrator requests."
	case "heartbeat":
		return "Connection active · last heartbeat sent " + time.Now().Format("15:04:05")
	case "connecting":
		return "Contacting server…"
	case "reconnecting":
		return "Trying again automatically. Check your network or server URL."
	case "enrolling":
		return "Registering this device with the server…"
	case "enrollment failed":
		return "Check the server URL and one-time token, then retry."
	case "saved config found":
		return "Saved device found. Choose Connect to allow a session."
	case "not enrolled":
		return "Enter the server URL and one-time token to get started."
	default:
		return "The administrator cannot reach this device while disconnected."
	}
}

func (state *guiState) setServerStatus(status string) {
	if state.window == nil || state.serverLabel == nil {
		return
	}
	state.window.Synchronize(func() {
		state.serverLabel.SetText(status)
	})
}

func (state *guiState) showError(message string) {
	if state.window == nil {
		return
	}
	state.window.Synchronize(func() {
		walk.MsgBox(state.window, "OpsRelay Agent", message, walk.MsgBoxIconError)
	})
}

func loadExistingConfig(path string) (config.Config, bool) {
	cfg, err := config.Load(path)
	return cfg, err == nil
}

func loadPrefs(path string) (config.Prefs, bool) {
	prefs, err := config.LoadPrefs(path)
	return prefs, err == nil
}

func initialStatus(hasConfig bool) string {
	if hasConfig {
		return "saved config found"
	}
	return "not enrolled"
}
