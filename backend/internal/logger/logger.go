package logger

import (
	"io"
	"log"
	"log/slog"
	"os"
	"strings"
)

const (
	FormatJSON = "json"
	FormatText = "text"
)

type Config struct {
	Environment string
	Service     string
	Level       string
	Format      string
	Output      io.Writer
}

func Setup(cfg *Config) *slog.Logger {
	if cfg == nil {
		cfg = &Config{}
	}

	output := cfg.Output
	if output == nil {
		output = os.Stdout
	}

	level := parseLevel(cfg.Level)
	options := &slog.HandlerOptions{
		Level: level,
	}

	attrs := baseAttrs(cfg)
	handler := slog.Handler(slog.NewJSONHandler(output, options))
	if normalizeFormat(cfg.Format, cfg.Environment) == FormatText {
		handler = slog.NewTextHandler(output, options)
	}
	if len(attrs) > 0 {
		handler = handler.WithAttrs(attrs)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	log.SetOutput(slog.NewLogLogger(handler, level).Writer())

	return logger
}

func baseAttrs(cfg *Config) []slog.Attr {
	attrs := make([]slog.Attr, 0, 2)
	if service := strings.TrimSpace(cfg.Service); service != "" {
		attrs = append(attrs, slog.String("service", service))
	}
	if environment := strings.TrimSpace(cfg.Environment); environment != "" {
		attrs = append(attrs, slog.String("environment", environment))
	}
	return attrs
}

func normalizeFormat(formatValue, environment string) string {
	formatValue = strings.ToLower(strings.TrimSpace(formatValue))
	switch formatValue {
	case FormatJSON, FormatText:
		return formatValue
	}

	if strings.EqualFold(strings.TrimSpace(environment), "development") {
		return FormatText
	}
	return FormatJSON
}

func parseLevel(value string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
