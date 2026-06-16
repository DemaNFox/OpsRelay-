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
	window      *walk.MainWindow
	serverEdit  *walk.LineEdit
	tokenEdit   *walk.LineEdit
	nameEdit    *walk.LineEdit
	deviceEdit  *walk.LineEdit
	statusLabel *walk.Label
	serverLabel *walk.Label
	connectBtn  *walk.PushButton
	stopBtn     *walk.PushButton
	cancel      context.CancelFunc
	logger      *slog.Logger
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
		Size:     Size{Width: 560, Height: 345},
		MinSize:  Size{Width: 560, Height: 345},
		MaxSize:  Size{Width: 560, Height: 345},
		Layout:   VBox{MarginsZero: false},
		Children: []Widget{
			Label{Text: "OpsRelay Agent"},
			Label{Text: "This device is visible to the administrator only after enrollment and connection."},
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
					Label{Text: "Status"},
					Label{AssignTo: &state.statusLabel, Text: initialStatus(hasConfig)},
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
						Text: "Connect with saved config",
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
			Label{Text: "Keep this window open while the administrator is connected. Closing it stops the MVP agent."},
		},
	}.Run()
	return err
}

func (state *guiState) enrollAndConnect(cfgPath string) {
	serverURL := state.serverEdit.Text()
	token := state.tokenEdit.Text()
	name := state.nameEdit.Text()
	if serverURL == "" || token == "" || name == "" {
		state.showError("Server URL, enrollment token and device name are required.")
		return
	}
	state.savePrefs(config.DefaultPrefsPath())

	state.setStatus("enrolling")
	state.setServerStatus("checking")
	client := transport.NewClient(serverURL)
	cfg, err := client.Enroll(context.Background(), transport.EnrollRequest{
		Token:           token,
		Name:            name,
		OS:              platform.OSInfo(),
		AgentVersion:    core.AgentVersion,
		ProtocolVersion: core.ProtocolVersion,
		Capabilities:    platform.Capabilities(),
	})
	if err != nil {
		state.setStatus("enrollment failed")
		state.setServerStatus("reachable, enrollment failed")
		state.showError(err.Error())
		return
	}
	cfg.ServerURL = serverURL
	cfg.Name = name
	if err := config.Save(cfgPath, cfg); err != nil {
		state.setStatus("config save failed")
		state.showError(err.Error())
		return
	}
	state.deviceEdit.SetText(cfg.DeviceID)
	state.tokenEdit.SetText("")
	state.savePrefs(config.DefaultPrefsPath())
	state.setServerStatus("reachable")
	state.startAgent(cfg)
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
		state.connectSaved(cfgPath)
	})
}

func (state *guiState) startAgent(cfg config.Config) {
	state.stop()
	ctx, cancel := context.WithCancel(context.Background())
	state.cancel = cancel
	state.connectBtn.SetEnabled(false)
	state.stopBtn.SetEnabled(true)

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
		state.setStatus(status)
		if status == "connected" {
			state.setServerStatus("reachable")
		}
		if status == "reconnecting" {
			state.setServerStatus("connection lost")
		}
	})

	go func() {
		err := runner.Run(ctx)
		if err != nil && ctx.Err() == nil {
			state.setStatus("stopped with error")
			state.showError(err.Error())
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
	if state.cancel != nil {
		state.cancel()
		state.cancel = nil
	}
	if state.connectBtn != nil {
		state.connectBtn.SetEnabled(true)
	}
	if state.stopBtn != nil {
		state.stopBtn.SetEnabled(false)
	}
	state.setStatus("stopped")
}

func (state *guiState) setStatus(status string) {
	if state.window == nil || state.statusLabel == nil {
		return
	}
	state.window.Synchronize(func() {
		state.statusLabel.SetText(status)
	})
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
