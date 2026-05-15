package parser

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
)

func trimBytes(data []byte) []byte {
	return bytes.TrimSpace(data)
}

func normalizeRecord(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))

	for key, value := range in {
		out[normalizeKey(key)] = strings.TrimSpace(value)
	}

	return out
}

func normalizeKey(in string) string {
	in = strings.ToLower(strings.TrimSpace(in))

	replacer := strings.NewReplacer(
		"_", "",
		"-", "",
		" ", "",
		".", "",
		"/", "",
		"\\", "",
	)

	return replacer.Replace(in)
}

func get(rec map[string]string, keys ...string) string {
	for _, key := range keys {
		if value, ok := rec[normalizeKey(key)]; ok {
			return value
		}
	}

	return ""
}

func hasAny(rec map[string]string, keys ...string) bool {
	for _, key := range keys {
		if _, ok := rec[normalizeKey(key)]; ok {
			return true
		}
	}

	return false
}

func rawJSON(rec map[string]string) string {
	data, err := json.Marshal(rec)
	if err != nil {
		return ""
	}

	return string(data)
}

func firstNonEmptyLine(data []byte) string {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			return line
		}
	}

	return ""
}

func deriveNodeKind(kind string, desc string, nodeType int32) string {
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind != "" {
		return kind
	}

	desc = strings.ToLower(desc)

	switch {
	case strings.Contains(desc, "switch"), strings.Contains(desc, "sw"):
		return "switch"
	case strings.Contains(desc, "host"), strings.Contains(desc, "hca"), strings.Contains(desc, "ca"):
		return "host"
	case nodeType == 2:
		return "switch"
	case nodeType == 1:
		return "host"
	default:
		return "unknown"
	}
}
