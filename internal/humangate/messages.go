/*
Package humangate messages 提供人工消息落盘协议。
Author: Anner
Created on 2026/3/8
*/
package humangate

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var messagePattern = regexp.MustCompile(`## message: msg-(\d{6})`)

// Inject 追加一条人工补充消息到 planner/human_messages.md。
func (s *Service) Inject(options InjectOptions) (InjectResult, error) {
	message := strings.TrimSpace(options.Message)
	if message == "" {
		return InjectResult{}, fmt.Errorf("message is required")
	}
	content, err := readFile(s.paths.humanMessages)
	if err != nil && !strings.Contains(err.Error(), "no such file") {
		return InjectResult{}, err
	}
	if err != nil || placeholderDocument(content) {
		content = "# Human Messages\n"
	}
	messageID := nextMessageID(content)
	block := renderMessageBlock(messageID, options.RoleID, message)
	trimmed := strings.TrimRight(content, "\n")
	if err := atomicWrite(s.paths.humanMessages, []byte(trimmed+block)); err != nil {
		return InjectResult{}, err
	}
	return InjectResult{MessageID: messageID, Path: s.paths.humanMessages}, nil
}

func nextMessageID(content string) string {
	matches := messagePattern.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return "msg-000001"
	}
	last := matches[len(matches)-1][1]
	value, err := strconv.Atoi(last)
	if err != nil {
		return "msg-000001"
	}
	return fmt.Sprintf("msg-%06d", value+1)
}

func renderMessageBlock(messageID string, roleID string, message string) string {
	return strings.Join([]string{
		"",
		fmt.Sprintf("## message: %s", messageID),
		fmt.Sprintf("- message_id: %s", messageID),
		fmt.Sprintf("- source: %s", humanGateSource),
		fmt.Sprintf("- kind: %s", injectKind),
		fmt.Sprintf("- target_role_id: %s", roleID),
		fmt.Sprintf("- created_at: %s", utcNow()),
		"",
		"### content",
		strings.TrimSpace(message),
		"",
	}, "\n")
}
