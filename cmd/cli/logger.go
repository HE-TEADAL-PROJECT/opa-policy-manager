// Copyright 2025 Matteo Brambilla - TEADAL
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

type CliHandler struct {
	writer *os.File
}

// Enabled implements slog.Handler.
func (h *CliHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= slog.LevelInfo
}

// Handle implements slog.Handler.
func (h *CliHandler) Handle(_ context.Context, record slog.Record) error {
	var builder strings.Builder

	// Add the message to the output.
	builder.WriteString(record.Message)

	// Add context (key-value pairs) to the output.
	record.Attrs(func(attr slog.Attr) bool {
		builder.WriteString(" ")
		builder.WriteString(attr.Key)
		builder.WriteString("=")
		builder.WriteString(attr.Value.String())
		return true
	})

	// Write the formatted log to the writer.
	_, err := h.writer.WriteString(builder.String() + "\n")
	return err
}

// WithAttrs implements slog.Handler.
func (h *CliHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

// WithGroup implements slog.Handler.
func (h *CliHandler) WithGroup(name string) slog.Handler {
	return h
}

func NewCliHandler(writer *os.File) *CliHandler {
	return &CliHandler{
		writer,
	}
}

var _ slog.Handler = &CliHandler{}
