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

	go func() {
		if hasConfig {
			state.waitAndConnectSaved(cfgPath)
		}
	}()

	_, err := MainWindow{
		AssignTo:   &state.window,
		Title:      "OpsRelay Agent",
		Size:       Size{Width: 760, Height: 640},
		MinSize:    Size{Width: 700, Height: 600},
		Font:       Font{Family: "Segoe UI", PointSize: 10},
		Background: SolidColorBrush{Color: walk.RGB(244, 247, 250)},
		Layout:     VBox{Margins: Margins{Left: 18, Top: 18, Right: 18, Bottom: 18}, Spacing: 12},
		Children: []Widget{
			Composite{
				Background: SolidColorBrush{Color: walk.RGB(24, 50, 75)},
				Layout:     VBox{Margins: Margins{Left: 22, Top: 17, Right: 22, Bottom: 17}, Spacing: 5},
				MinSize:    Size{Height: 94},
				Children: []Widget{
					Label{Text: "●   OPSRELAY", Font: Font{Family: "Segoe UI", PointSize: 17, Bold: true}, TextColor: walk.RGB(78, 226, 160)},
					Label{Text: "Удалённая поддержка с вашего разрешения", TextColor: walk.RGB(221, 234, 244)},
				},
			},
			Composite{
				Background: SolidColorBrush{Color: walk.RGB(255, 255, 255)},
				Border:     true,
				Layout:     VBox{Margins: Margins{Left: 20, Top: 15, Right: 20, Bottom: 15}, Spacing: 7},
				MinSize:    Size{Height: 134},
				Children: []Widget{
					Label{Text: "СОСТОЯНИЕ ПОДКЛЮЧЕНИЯ", Font: Font{Family: "Segoe UI", PointSize: 8, Bold: true}, TextColor: walk.RGB(102, 119, 135)},
					Composite{Layout: HBox{MarginsZero: true, Spacing: 8}, Children: []Widget{
						Label{AssignTo: &state.statusDot, Text: "●", Font: Font{Family: "Segoe UI", PointSize: 22}, TextColor: statusColor(initialStatus(hasConfig))},
						Label{AssignTo: &state.statusLabel, Text: friendlyStatus(initialStatus(hasConfig)), Font: Font{Family: "Segoe UI", PointSize: 14, Bold: true}, TextColor: walk.RGB(24, 50, 75)},
					}},
					Label{AssignTo: &state.activityLabel, Text: statusActivity(initialStatus(hasConfig)), TextColor: walk.RGB(84, 101, 117)},
				},
			},
			Composite{
				Background: SolidColorBrush{Color: walk.RGB(255, 255, 255)},
				Border:     true,
				Layout:     VBox{Margins: Margins{Left: 20, Top: 16, Right: 20, Bottom: 16}, Spacing: 10},
				Children: []Widget{
					Label{Text: "Настройки устройства", Font: Font{Family: "Segoe UI", PointSize: 12, Bold: true}, TextColor: walk.RGB(24, 50, 75)},
					Label{Text: "Для первого подключения введите адрес сервера и одноразовый токен.", TextColor: walk.RGB(84, 101, 117)},
					Composite{Layout: Grid{Columns: 2, MarginsZero: true, Spacing: 12}, Children: []Widget{
						Composite{Layout: VBox{MarginsZero: true, Spacing: 4}, Children: []Widget{
							Label{Text: "Адрес сервера"},
							LineEdit{AssignTo: &state.serverEdit, Text: serverURL, OnTextChanged: func() { state.savePrefs(prefsPath) }},
						}},
						Composite{Layout: VBox{MarginsZero: true, Spacing: 4}, Children: []Widget{
							Label{Text: "Имя устройства"},
							LineEdit{AssignTo: &state.nameEdit, Text: deviceName, OnTextChanged: func() { state.savePrefs(prefsPath) }},
						}},
						Composite{Layout: VBox{MarginsZero: true, Spacing: 4}, Children: []Widget{
							Label{Text: "Одноразовый токен"},
							LineEdit{AssignTo: &state.tokenEdit, PasswordMode: true},
						}},
						Composite{Layout: VBox{MarginsZero: true, Spacing: 4}, Children: []Widget{
							Label{Text: "ID устройства"},
							LineEdit{AssignTo: &state.deviceEdit, Text: deviceID, ReadOnly: true},
						}},
					}},
					Composite{Layout: HBox{MarginsZero: true}, Children: []Widget{
						Label{Text: "Сервер:", TextColor: walk.RGB(84, 101, 117)},
						Label{AssignTo: &state.serverLabel, Text: "не проверен", TextColor: walk.RGB(84, 101, 117)},
						PushButton{Text: "Проверить сервер", OnClicked: func() { state.checkServer() }},
					}},
				},
			},
			Composite{Layout: HBox{MarginsZero: true, Spacing: 10}, Children: []Widget{
				PushButton{Text: "Зарегистрировать и подключить", AssignTo: &state.connectBtn, MinSize: Size{Width: 230, Height: 38}, Font: Font{Family: "Segoe UI", PointSize: 10, Bold: true}, OnClicked: func() { state.enrollAndConnect(cfgPath) }},
				PushButton{Text: "Подключить сохранённое", AssignTo: &state.savedBtn, Enabled: hasConfig, MinSize: Size{Width: 210, Height: 38}, OnClicked: func() { state.connectSaved(cfgPath) }},
				PushButton{Text: "Отключить", AssignTo: &state.stopBtn, Enabled: false, MinSize: Size{Width: 110, Height: 38}, OnClicked: func() { state.stop() }},
			}},
			Label{Text: "Вы можете отключиться в любой момент. Закрытие окна тоже прерывает соединение.", TextColor: walk.RGB(84, 101, 117)},
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

func (state *guiState) waitAndConnectSaved(cfgPath string) {
	for state.window == nil {
		time.Sleep(50 * time.Millisecond)
	}
	state.window.Synchronize(func() {
		if walk.MsgBox(state.window, "Подключение OpsRelay", "Разрешить подключение этого устройства сейчас? Вы сможете отключить его в любой момент.", walk.MsgBoxYesNo|walk.MsgBoxIconQuestion) == walk.DlgCmdYes {
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
	}
	if state.savedBtn != nil {
		state.savedBtn.SetEnabled(state.deviceEdit != nil && state.deviceEdit.Text() != unregisteredDeviceID)
	}
	if state.stopBtn != nil {
		state.stopBtn.SetEnabled(false)
	}
	state.setInputsEnabled(true)
	state.setStatus("stopped")
}

func (state *guiState) setInputsEnabled(enabled bool) {
	for _, field := range []*walk.LineEdit{state.serverEdit, state.nameEdit, state.tokenEdit} {
		if field != nil {
			field.SetEnabled(enabled)
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
		return "Готово к подключению"
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
		return "Соединение установлено. Ожидаем запросы администратора."
	case "heartbeat":
		return "Связь активна · последний сигнал отправлен в " + time.Now().Format("15:04:05")
	case "connecting":
		return "Связываемся с сервером…"
	case "reconnecting":
		return "Повторим попытку автоматически. Проверьте сеть и адрес сервера."
	case "enrolling":
		return "Регистрируем устройство на сервере…"
	case "enrollment failed":
		return "Проверьте адрес сервера и одноразовый токен, затем повторите."
	case "saved config found":
		return "Настройки найдены. Разрешите подключение, когда будете готовы."
	case "not enrolled":
		return "Введите адрес сервера и одноразовый токен для первого подключения."
	default:
		return "Пока устройство отключено, администратор не может к нему обратиться."
	}
}

func (state *guiState) setServerStatus(status string) {
	if state.window == nil || state.serverLabel == nil {
		return
	}
	state.window.Synchronize(func() {
		state.serverLabel.SetText(friendlyServerStatus(status))
	})
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
	if state.window == nil {
		return
	}
	state.window.Synchronize(func() {
		walk.MsgBox(state.window, "OpsRelay · Ошибка", message, walk.MsgBoxIconError)
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
