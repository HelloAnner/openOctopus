/*
Package tmux startup 负责 pane 启动命令与默认焦点策略。
Author: Anner
Created on 2026/3/8
*/
package tmux

import "strings"

func paneStartupCommand(roleID string, sessionID string, launchCommand string) string {
	trimmed := strings.TrimSpace(launchCommand)
	if trimmed != "" {
		return trimmed
	}
	return paneBanner(roleID, sessionID)
}

func defaultFocusPaneID(mainPaneID string, rolePanes map[string]PaneBinding, roleIDs []string) string {
	for _, roleID := range roleIDs {
		binding, ok := rolePanes[roleID]
		if ok && strings.TrimSpace(binding.PaneID) != "" {
			return binding.PaneID
		}
	}
	return mainPaneID
}
