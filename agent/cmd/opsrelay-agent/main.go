package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/opsrelay/agent/internal/config"
	"github.com/opsrelay/agent/internal/core"
	"github.com/opsrelay/agent/internal/executor"
	"github.com/opsrelay/agent/internal/platform"
	"github.com/opsrelay/agent/internal/transport"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if len(os.Args) == 1 && runtime.GOOS == "windows" {
		if err := runWindowsGUI(logger); err != nil {
			logger.Error("agent gui failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
		return
	}
	if err := run(logger, os.Args[1:]); err != nil {
		logger.Error("agent failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run(logger *slog.Logger, args []string) error {
	if len(args) == 0 {
		args = []string{"run"}
	}

	cfgPath := config.DefaultPath()
	for index := 0; index < len(args); index++ {
		if args[index] == "--config" && index+1 < len(args) {
			cfgPath = args[index+1]
			index++
		}
	}

	switch args[0] {
	case "enroll":
		enrollArgs, err := parseEnrollArgs(args[1:], cfgPath)
		if err != nil {
			return err
		}
		client := transport.NewClient(enrollArgs.serverURL)
		cfg, err := client.Enroll(context.Background(), transport.EnrollRequest{
			Token:           enrollArgs.token,
			Name:            enrollArgs.name,
			OS:              platform.OSInfo(),
			AgentVersion:    core.AgentVersion,
			ProtocolVersion: core.ProtocolVersion,
			Capabilities:    platform.Capabilities(),
		})
		if err != nil {
			return err
		}
		cfg.ServerURL = enrollArgs.serverURL
		cfg.Name = enrollArgs.name
		if err := config.Save(enrollArgs.configPath, cfg); err != nil {
			return err
		}
		logger.Info("enrollment completed", slog.String("device_id", cfg.DeviceID), slog.String("config", enrollArgs.configPath))
		return nil
	case "run":
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return err
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		runner := core.NewAgent(
			cfg,
			transport.NewClient(cfg.ServerURL),
			executor.NewShellRunner(),
			platform.NewRebooter(cfg.AllowReboot),
			platform.OSInfo(),
			platform.Capabilities(),
			logger,
		)
		return runner.Run(ctx)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

type enrollArgs struct {
	serverURL  string
	token      string
	name       string
	configPath string
}

func parseEnrollArgs(args []string, defaultConfigPath string) (enrollArgs, error) {
	parsed := enrollArgs{configPath: defaultConfigPath}
	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--server":
			index++
			if index >= len(args) {
				return parsed, fmt.Errorf("--server requires a value")
			}
			parsed.serverURL = args[index]
		case "--token":
			index++
			if index >= len(args) {
				return parsed, fmt.Errorf("--token requires a value")
			}
			parsed.token = args[index]
		case "--name":
			index++
			if index >= len(args) {
				return parsed, fmt.Errorf("--name requires a value")
			}
			parsed.name = args[index]
		case "--config":
			index++
			if index >= len(args) {
				return parsed, fmt.Errorf("--config requires a value")
			}
			parsed.configPath = args[index]
		}
	}
	if parsed.serverURL == "" || parsed.token == "" || parsed.name == "" {
		return parsed, fmt.Errorf("enroll requires --server, --token and --name")
	}
	return parsed, nil
}
