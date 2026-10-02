package provider

import (
	"bytes"
	"encoding/json"
	"io"
)

// claudeStreamWriter converts the stream-json output of `claude -p` into the
// plain-text transcript that evaluation assertions read. It keeps only what
// the agent produced: the text of assistant messages and the name and input
// of every tool call. System events, tool results, and a successful final
// result are dropped, because the agent did not write them or they repeat the
// last message; the system init event alone lists every installed skill name
// and would satisfy any handoff assertion. A line that is not a stream event,
// such as a CLI error, passes through unchanged.
type claudeStreamWriter struct {
	out     io.Writer
	pending []byte
}

func newClaudeStreamWriter(out io.Writer) *claudeStreamWriter {
	return &claudeStreamWriter{out: out}
}

// Write buffers partial lines and converts every complete line.
func (w *claudeStreamWriter) Write(p []byte) (int, error) {
	w.pending = append(w.pending, p...)
	for {
		index := bytes.IndexByte(w.pending, '\n')
		if index < 0 {
			return len(p), nil
		}
		line := w.pending[:index]
		w.pending = w.pending[index+1:]
		if err := w.convert(line); err != nil {
			return len(p), err
		}
	}
}

// Close converts a final line that has no trailing newline.
func (w *claudeStreamWriter) Close() error {
	if len(w.pending) == 0 {
		return nil
	}
	line := w.pending
	w.pending = nil
	return w.convert(line)
}

type claudeStreamEvent struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype"`
	Result  string `json:"result"`
	Message *struct {
		Content []struct {
			Type  string          `json:"type"`
			Text  string          `json:"text"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
	} `json:"message"`
}

func (w *claudeStreamWriter) convert(line []byte) error {
	trimmed := bytes.TrimSpace(line)
	if len(trimmed) == 0 {
		return nil
	}
	var event claudeStreamEvent
	if trimmed[0] != '{' || json.Unmarshal(trimmed, &event) != nil || event.Type == "" {
		_, err := w.out.Write(append(append([]byte{}, line...), '\n'))
		return err
	}
	var text bytes.Buffer
	switch event.Type {
	case "assistant":
		if event.Message == nil {
			return nil
		}
		for _, block := range event.Message.Content {
			switch block.Type {
			case "text":
				text.WriteString(block.Text)
				text.WriteByte('\n')
			case "tool_use":
				text.WriteString("[tool " + block.Name + "] ")
				text.Write(block.Input)
				text.WriteByte('\n')
			}
		}
	case "result":
		if event.Subtype != "success" {
			text.WriteString(event.Result)
			text.WriteByte('\n')
		}
	}
	if text.Len() == 0 {
		return nil
	}
	_, err := w.out.Write(text.Bytes())
	return err
}
