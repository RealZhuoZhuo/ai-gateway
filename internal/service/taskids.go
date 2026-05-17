package service

import "strings"

const (
	taskPrefixArk             = "ark"
	taskPrefixKlingImage      = "kling-img"
	taskPrefixKlingTextVideo  = "kling-t2v"
	taskPrefixKlingImageVideo = "kling-i2v"
	taskPrefixDashImage       = "dashscope-img"
	taskPrefixDashTextVideo   = "dashscope-t2v"
	taskPrefixDashImageVideo  = "dashscope-i2v"
)

func encodeTaskID(prefix, raw string) string {
	if raw == "" {
		return ""
	}
	return prefix + "_" + raw
}

func decodeTaskID(taskID string) (string, string, bool) {
	prefix, raw, ok := strings.Cut(strings.TrimSpace(taskID), "_")
	if !ok || prefix == "" || raw == "" {
		return "", "", false
	}
	return prefix, raw, true
}
