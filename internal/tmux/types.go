/*
Package tmux types 定义 tmux 模块的公开模型。
Author: Anner
Created on 2026/3/8
*/
package tmux

type Service struct {
	sessionDir string
	runner     commandRunner
}

type BootstrapOptions struct {
	SessionID        string
	RoleIDs          []string
	SocketTemplate   string
	MainPaneRatio    float64
	RoleLayout       string
	MainLaunchCommand string
	LaunchCommands   map[string]string
}

type BootstrapResult struct {
	SocketName  string
	SessionName string
	WindowName  string
	MainPaneID  string
	RolePanes   map[string]PaneBinding
}

type Layout struct {
	SessionDir    string
	SessionID     string
	SocketName    string
	SessionName   string
	WindowName    string
	RoleLayout    string
	MainPaneRatio float64
	MainPaneID    string
	UpdatedAt     string
	RolePanes     map[string]PaneBinding
}

type PaneBinding struct {
	RoleID string
	PaneID string
	Title  string
}

type ResolveTargetOptions struct {
	RoleID string
	Main   bool
}

type ResolveTargetResult struct {
	SessionDir   string
	SocketName   string
	SessionName  string
	TargetRole   string
	TargetPaneID string
	Switched     bool
}
