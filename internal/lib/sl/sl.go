package sl

import "log/slog"

// Err wraps an error as a log attribute.
func Err(err error) slog.Attr {
	return slog.Attr{Key: "error", Value: slog.StringValue(err.Error())}
}

// Module tags log records with the emitting component.
func Module(mod string) slog.Attr {
	return slog.Attr{Key: "mod", Value: slog.StringValue(mod)}
}
