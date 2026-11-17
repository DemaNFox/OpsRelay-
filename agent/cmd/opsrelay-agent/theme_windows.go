//go:build windows

package main

import (
	"syscall"
	"unsafe"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
)

// The native BUTTON control keeps the system's light fill on Windows 10.
// This focusable custom widget gives the application's actions the same
// palette while retaining keyboard activation and an accessibility role.
func themedButton(assign **walk.CustomWidget, label string, size Size, base, fill, foreground walk.Color, visible bool, action func()) CustomWidget {
	return CustomWidget{
		AssignTo:      assign,
		MinSize:       size,
		Visible:       visible,
		Background:    SolidColorBrush{Color: base},
		Style:         uint32(win.WS_TABSTOP),
		Font:          Font{Family: "Segoe UI", PointSize: 10, Bold: true},
		Accessibility: Accessibility{Name: label, Role: AccRolePushbutton, DefaultAction: "Нажать", State: AccStateFocusable},
		PaintMode:     PaintBuffered,
		PaintPixels: func(canvas *walk.Canvas, _ walk.Rectangle) error {
			widget := *assign
			if widget == nil {
				return nil
			}
			bounds := widget.ClientBoundsPixels()
			baseBrush, err := walk.NewSolidColorBrush(base)
			if err != nil {
				return err
			}
			defer baseBrush.Dispose()
			if err := canvas.FillRectanglePixels(baseBrush, bounds); err != nil {
				return err
			}
			color, textColor := fill, foreground
			if !widget.Enabled() {
				color, textColor = themeInput, themeMuted
			}
			brush, err := walk.NewSolidColorBrush(color)
			if err != nil {
				return err
			}
			defer brush.Dispose()
			if err := canvas.FillRoundedRectanglePixels(brush, bounds, walk.Size{Width: 10, Height: 10}); err != nil {
				return err
			}
			if widget.Focused() {
				pen, err := walk.NewCosmeticPen(walk.PenSolid, themeText)
				if err != nil {
					return err
				}
				defer pen.Dispose()
				focusBounds := walk.Rectangle{X: 2, Y: 2, Width: bounds.Width - 4, Height: bounds.Height - 4}
				if err := canvas.DrawRoundedRectanglePixels(pen, focusBounds, walk.Size{Width: 8, Height: 8}); err != nil {
					return err
				}
			}
			return canvas.DrawTextPixels(label, widget.Font(), textColor, bounds, walk.TextCenter|walk.TextVCenter|walk.TextSingleLine)
		},
		OnMouseDown: func(_, _ int, button walk.MouseButton) {
			if button == walk.LeftButton && (*assign).Enabled() {
				_ = (*assign).SetFocus()
			}
		},
		OnMouseUp: func(_, _ int, button walk.MouseButton) {
			if button == walk.LeftButton && (*assign).Enabled() {
				action()
			}
		},
		OnKeyUp: func(key walk.Key) {
			if (key == walk.KeyReturn || key == walk.KeySpace) && (*assign).Enabled() {
				action()
			}
		},
	}
}

// Keep the standard Windows title bar aligned with the application's dark palette.
// Unsupported DWM attributes are ignored so older Windows releases can still open the UI.
func setDarkTitleBar(handle uintptr) {
	proc := syscall.NewLazyDLL("dwmapi.dll").NewProc("DwmSetWindowAttribute")
	if err := proc.Find(); err != nil {
		return
	}
	setAttribute := func(attribute uintptr, value uint32) {
		proc.Call(handle, attribute, uintptr(unsafe.Pointer(&value)), unsafe.Sizeof(value))
	}
	setAttribute(20, 1)                       // DWMWA_USE_IMMERSIVE_DARK_MODE
	setAttribute(35, uint32(themeBackground)) // DWMWA_CAPTION_COLOR
	setAttribute(36, uint32(themeText))       // DWMWA_TEXT_COLOR
}
