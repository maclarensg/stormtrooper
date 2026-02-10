package tui

import (
	"testing"
)

func TestDefaultTheme(t *testing.T) {
	theme := DefaultTheme()

	// ChatBorder should have a border set
	if theme.ChatBorder.GetBorderStyle().Top == "" {
		t.Error("ChatBorder should have a border style set")
	}

	// SidebarBorder should have a border set
	if theme.SidebarBorder.GetBorderStyle().Top == "" {
		t.Error("SidebarBorder should have a border style set")
	}

	// InputBorder should have a border set
	if theme.InputBorder.GetBorderStyle().Top == "" {
		t.Error("InputBorder should have a border style set")
	}

	// StatusBar should have padding
	_, right, _, left := theme.StatusBar.GetPadding()
	if right == 0 && left == 0 {
		t.Error("StatusBar should have horizontal padding")
	}

	// UserPrefix should be bold
	if !theme.UserPrefix.GetBold() {
		t.Error("UserPrefix should be bold")
	}

	// AssistantPrefix should be bold
	if !theme.AssistantPrefix.GetBold() {
		t.Error("AssistantPrefix should be bold")
	}

	// SidebarHeading should be bold
	if !theme.SidebarHeading.GetBold() {
		t.Error("SidebarHeading should be bold")
	}

	// ToolInline should be italic
	if !theme.ToolInline.GetItalic() {
		t.Error("ToolInline should be italic")
	}

	// PermissionBorder should have a border set
	if theme.PermissionBorder.GetBorderStyle().Top == "" {
		t.Error("PermissionBorder should have a border style set")
	}
}
