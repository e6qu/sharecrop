package mcp

import (
	"bufio"
	"bytes"
	"context"
	"io"

	"github.com/e6qu/sharecrop/internal/auth"
)

const maxStdioLineBytes = 4 << 20

// ServeStdio runs the MCP server over a newline-delimited JSON-RPC stream, the
// stdio transport used by local agent clients. Each input line is one JSON-RPC
// message or batch; each response is written as one line. Notification-only
// inputs produce no output line.
func ServeStdio(ctx context.Context, server Server, subject auth.Subject, credential CallerCredential, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), maxStdioLineBytes)
	writer := bufio.NewWriter(out)

	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		result := server.HandleRaw(ctx, subject, credential, line)
		if !result.HasResponse {
			continue
		}
		if _, err := writer.Write(result.Payload); err != nil {
			return err
		}
		if err := writer.WriteByte('\n'); err != nil {
			return err
		}
		if err := writer.Flush(); err != nil {
			return err
		}
	}

	return scanner.Err()
}
