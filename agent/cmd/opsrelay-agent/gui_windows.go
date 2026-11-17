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

const unregisteredDeviceID = "не зарегистрировано"

var (
	themeBackground = walk.RGB(13, 34, 51)
	themeSurface    = walk.RGB(24, 50, 75)
	themeInput      = walk.RGB(44, 78, 102)
	themeText       = walk.RGB(232, 245, 250)
	themeMuted      = walk.RGB(167, 195, 210)
	themeMint       = walk.RGB(78, 226, 160)
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
	connectBtn    *walk.CustomWidget
	savedBtn      *walk.CustomWidget
	stopBtn       *walk.CustomWidget
	cancel        context.CancelFunc
	runID         uint64
	logger        *slog.Logger
}

func runWindowsGUI(logger *slog.Logger) error {
	state := &guiState{logger: logger}
	var checkBtn *walk.CustomWidget
	icon, iconErr := walk.NewIconFromResourceId(1)
	if iconErr != nil {
		if exePath, err := os.Executable(); err == nil {
			icon, iconErr = walk.NewIconExtractedFromFileWithSize(exePath, 0, 32)
		}
	}
	var windowIcon interface{}
	if iconErr != nil {
		logger.Warn("window icon unavailable", slog.String("error", iconErr.Error()))
	} else {
		windowIcon = icon
		defer icon.Dispose()
	}
	cfgPath := config.DefaultPath()
	prefsPath := config.DefaultPrefsPath()
	existing, hasConfig := loadExistingConfig(cfgPath)
	prefs, hasPrefs := loadPrefs(prefsPath)

	serverURL := "https://opsrelay.example.com"
	deviceID := unregisteredDeviceID
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

	ui := MainWindow{
		AssignTo:   &state.window,
		Title:      "OpsRelay Agent",
		Icon:       windowIcon,
		Size:       Size{Width: 620, Height: 410},
		MinSize:    Size{Width: 620, Height: 410},
		MaxSize:    Size{Width: 620, Height: 410},
		Font:       Font{Family: "Segoe UI", PointSize: 9},
		Background: SolidColorBrush{Color: themeBackground},
		Layout:     VBox{Margins: Margins{Left: 12, Top: 12, Right: 12, Bottom: 12}, Spacing: 8},
		Children: []Widget{
			Composite{
				Background: SolidColorBrush{Color: themeSurface},
				Layout:     VBox{Margins: Margins{Left: 14, Top: 9, Right: 14, Bottom: 9}, Spacing: 3},
				MinSize:    Size{Width: 580, Height: 75},
				Children: []Widget{
					Label{Text: "СОСТОЯНИЕ", Font: Font{Family: "Segoe UI", PointSize: 8, Bold: true}, TextColor: themeMint},
					Composite{Background: SolidColorBrush{Color: themeSurface}, Layout: HBox{MarginsZero: true, Spacing: 5}, Children: []Widget{
						Label{AssignTo: &state.statusDot, Text: "●", Font: Font{Family: "Segoe UI", PointSize: 16}, TextColor: statusColor(initialStatus(hasConfig))},
						Label{AssignTo: &state.statusLabel, Text: friendlyStatus(initialStatus(hasConfig)), Font: Font{Family: "Segoe UI", PointSize: 11, Bold: true}, TextColor: themeText},
					}},
					Label{AssignTo: &state.activityLabel, Text: statusActivity(initialStatus(hasConfig)), TextColor: themeMuted, EllipsisMode: EllipsisEnd},
				},
			},
			Composite{
				Background: SolidColorBrush{Color: themeSurface},
				Layout:     VBox{Margins: Margins{Left: 14, Top: 10, Right: 14, Bottom: 10}, Spacing: 6},
				MinSize:    Size{Width: 580},
				Children: []Widget{
					Label{Text: "Устройство", Font: Font{Family: "Segoe UI", PointSize: 10, Bold: true}, TextColor: themeText},
					Composite{Background: SolidColorBrush{Color: themeSurface}, Layout: Grid{Columns: 2, MarginsZero: true, Spacing: 8}, Children: []Widget{
						Composite{Background: SolidColorBrush{Color: themeSurface}, Layout: VBox{MarginsZero: true, Spacing: 2}, Children: []Widget{
							Label{Text: "Адрес сервера", TextColor: themeText},
							LineEdit{AssignTo: &state.serverEdit, Text: serverURL, Background: SolidColorBrush{Color: themeInput}, TextColor: themeText, OnTextChanged: func() { state.savePrefs(prefsPath) }},
						}},
						Composite{Background: SolidColorBrush{Color: themeSurface}, Layout: VBox{MarginsZero: true, Spacing: 2}, Children: []Widget{
							Label{Text: "Имя устройства", TextColor: themeText},
							LineEdit{AssignTo: &state.nameEdit, Text: deviceName, Background: SolidColorBrush{Color: themeInput}, TextColor: themeText, OnTextChanged: func() { state.savePrefs(prefsPath) }},
						}},
						Composite{Background: SolidColorBrush{Color: themeSurface}, Layout: VBox{MarginsZero: true, Spacing: 2}, Children: []Widget{
							Label{Text: "Одноразовый токен", TextColor: themeText},
							LineEdit{AssignTo: &state.tokenEdit, PasswordMode: true, Background: SolidColorBrush{Color: themeInput}, TextColor: themeText},
						}},
						Composite{Background: SolidColorBrush{Color: themeSurface}, Layout: VBox{MarginsZero: true, Spacing: 2}, Children: []Widget{
							Label{Text: "ID устройства", TextColor: themeText},
							LineEdit{AssignTo: &state.deviceEdit, Text: deviceID, ReadOnly: true, Background: SolidColorBrush{Color: themeInput}, TextColor: themeMuted},
						}},
					}},
					Composite{Background: SolidColorBrush{Color: themeSurface}, Layout: HBox{MarginsZero: true}, Children: []Widget{
						Label{Text: "Сервер:", TextColor: themeMuted},
						Label{AssignTo: &state.serverLabel, Text: "не проверен", TextColor: themeMuted},
						themedButton(&checkBtn, "Проверить", Size{Width: 112, Height: 28}, themeSurface, themeInput, themeText, true, func() { state.checkServer() }),
					}},
				},
			},
			Composite{Background: SolidColorBrush{Color: themeBackground}, Layout: HBox{MarginsZero: true, Spacing: 8}, Children: []Widget{
				themedButton(&state.connectBtn, "Зарегистрировать и подключить", Size{Width: 236, Height: 32}, themeBackground, themeMint, themeBackground, true, func() { state.enrollAndConnect(cfgPath) }),
				themedButton(&state.savedBtn, "Подключить сохранённое", Size{Width: 184, Height: 32}, themeBackground, themeInput, themeText, hasConfig, func() { state.connectSaved(cfgPath) }),
				themedButton(&state.stopBtn, "Отключить", Size{Width: 100, Height: 32}, themeBackground, themeInput, themeText, false, func() { state.stop() }),
			}},
		},
	}
	if err := ui.Create(); err != nil {
		return err
	}
	for _, button := range []*walk.CustomWidget{checkBtn, state.connectBtn, state.savedBtn, state.stopBtn} {
		widget := button
		widget.FocusedChanged().Attach(func() { _ = widget.Invalidate() })
	}
	setDarkTitleBar(uintptr(state.window.Handle()))
	state.window.Run()
	if state.cancel != nil {
		state.cancel()
	}
	return nil
}

func (state *guiState) enrollAndConnect(cfgPath string) {
	serverURL := strings.TrimSpace(state.serverEdit.Text())
	token := strings.TrimSpace(state.tokenEdit.Text())
	name := strings.TrimSpace(state.nameEdit.Text())
	if serverURL == "" || token == "" || name == "" {
		state.showError("Укажите адрес сервера, одноразовый токен и имя устройства.")
		return
	}
	state.savePrefs(config.DefaultPrefsPath())

	state.setStatus("enrolling")
	state.setServerStatus("checking")
	state.connectBtn.SetEnabled(false)
	state.savedBtn.SetEnabled(false)
	state.setInputsEnabled(false)
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
			state.savedBtn.SetEnabled(state.deviceEdit.Text() != unregisteredDeviceID)
			state.setInputsEnabled(true)
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
			state.savedBtn.SetEnabled(state.deviceEdit.Text() != unregisteredDeviceID)
			state.setInputsEnabled(true)
			state.showError(err.Error())
		})
		return
	}
	state.window.Synchronize(func() {
		state.deviceEdit.SetText(cfg.DeviceID)
		state.tokenEdit.SetText("")
		state.savedBtn.SetEnabled(true)
		state.savedBtn.SetVisible(true)
		state.setServerStatus("reachable")
		state.startAgent(cfg)
	})
}

func (state *guiState) connectSaved(cfgPath string) {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		state.showError(fmt.Sprintf("Сохранённые настройки не найдены или повреждены: %s", err.Error()))
		return
	}
	state.serverEdit.SetText(cfg.ServerURL)
	state.nameEdit.SetText(cfg.Name)
	state.deviceEdit.SetText(cfg.DeviceID)
	state.startAgent(cfg)
}

func (state *guiState) startAgent(cfg config.Config) {
	state.stop()
	state.runID++
	runID := state.runID
	ctx, cancel := context.WithCancel(context.Background())
	state.cancel = cancel
	state.connectBtn.SetEnabled(false)
	state.connectBtn.SetVisible(false)
	state.savedBtn.SetEnabled(false)
	state.savedBtn.SetVisible(false)
	state.stopBtn.SetEnabled(true)
	state.stopBtn.SetVisible(true)
	state.setInputsEnabled(false)
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
		state.connectBtn.SetVisible(true)
	}
	if state.savedBtn != nil {
		hasSaved := state.deviceEdit != nil && state.deviceEdit.Text() != unregisteredDeviceID
		state.savedBtn.SetEnabled(hasSaved)
		state.savedBtn.SetVisible(hasSaved)
	}
	if state.stopBtn != nil {
		state.stopBtn.SetEnabled(false)
		state.stopBtn.SetVisible(false)
	}
	state.setInputsEnabled(true)
	state.setStatus("stopped")
}

func (state *guiState) setInputsEnabled(enabled bool) {
	for _, field := range []*walk.LineEdit{state.serverEdit, state.nameEdit, state.tokenEdit} {
		if field != nil {
			field.SetReadOnly(!enabled)
		}
	}
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
			state.activityLabel.SetTextColor(themeMuted)
			state.activityLabel.SetToolTipText("")
		}
	})
}

func friendlyStatus(status string) string {
	switch status {
	case "connected", "heartbeat":
		return "Подключено"
	case "connecting":
		return "Подключаемся…"
	case "reconnecting":
		return "Связь потеряна — повторяем…"
	case "enrolling":
		return "Регистрируем устройство…"
	case "enrollment failed":
		return "Ошибка регистрации"
	case "config save failed":
		return "Не удалось сохранить настройки"
	case "saved config found":
		return "Подключиться сейчас?"
	case "not enrolled":
		return "Нужна регистрация"
	case "stopped with error":
		return "Соединение остановлено из-за ошибки"
	default:
		return "Отключено"
	}
}

func statusColor(status string) walk.Color {
	switch status {
	case "connected", "heartbeat":
		return themeMint
	case "connecting", "enrolling", "reconnecting":
		return walk.RGB(229, 151, 30)
	case "enrollment failed", "config save failed", "stopped with error":
		return walk.RGB(200, 58, 58)
	default:
		return themeMuted
	}
}

func statusActivity(status string) string {
	switch status {
	case "connected":
		return "Ожидаем запросы администратора."
	case "heartbeat":
		return "Связь активна · последний сигнал отправлен в " + time.Now().Format("15:04:05")
	case "connecting":
		return "Связываемся с сервером…"
	case "reconnecting":
		return "Повторяем попытку автоматически."
	case "enrolling":
		return "Регистрируем устройство на сервере…"
	case "enrollment failed":
		return "Проверьте адрес и токен."
	case "saved config found":
		return "Настройки сохранены."
	case "not enrolled":
		return "Введите адрес и одноразовый токен."
	default:
		return "Соединение закрыто."
	}
}

func (state *guiState) setServerStatus(status string) {
	if state.window == nil || state.serverLabel == nil {
		return
	}
	state.window.Synchronize(func() {
		state.serverLabel.SetText(friendlyServerStatus(status))
		state.serverLabel.SetTextColor(serverStatusColor(status))
	})
}

func serverStatusColor(status string) walk.Color {
	switch status {
	case "reachable":
		return themeMint
	case "not reachable", "connection lost", "enrollment failed":
		return walk.RGB(255, 139, 139)
	default:
		return themeMuted
	}
}

func friendlyServerStatus(status string) string {
	switch status {
	case "checking":
		return "проверяем…"
	case "reachable":
		return "доступен"
	case "connection lost":
		return "связь потеряна"
	case "not reachable":
		return "не отвечает"
	case "missing server URL":
		return "укажите адрес"
	case "enrollment failed":
		return "регистрация не удалась"
	default:
		return status
	}
}

func (state *guiState) showError(message string) {
	if state.window == nil || state.activityLabel == nil {
		return
	}
	state.window.Synchronize(func() {
		state.activityLabel.SetText(message)
		state.activityLabel.SetTextColor(walk.RGB(255, 139, 139))
		state.activityLabel.SetToolTipText(message)
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
