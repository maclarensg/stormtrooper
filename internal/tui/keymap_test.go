package tui

import (
	"testing"
)

func TestDefaultKeyMap(t *testing.T) {
	km := DefaultKeyMap()

	tests := []struct {
		name    string
		keys    []string
		binding func() []string
	}{
		{"Send", []string{"enter"}, func() []string { return km.Send.Keys() }},
		{"NewLine", []string{"ctrl+j"}, func() []string { return km.NewLine.Keys() }},
		{"ScrollUp", []string{"up", "k"}, func() []string { return km.ScrollUp.Keys() }},
		{"ScrollDown", []string{"down", "j"}, func() []string { return km.ScrollDown.Keys() }},
		{"FocusChat", []string{"esc"}, func() []string { return km.FocusChat.Keys() }},
		{"FocusInput", []string{"i"}, func() []string { return km.FocusInput.Keys() }},
		{"Quit", []string{"ctrl+c"}, func() []string { return km.Quit.Keys() }},
		{"PermAllow", []string{"y"}, func() []string { return km.PermAllow.Keys() }},
		{"PermDeny", []string{"n"}, func() []string { return km.PermDeny.Keys() }},
		{"Tab", []string{"tab"}, func() []string { return km.Tab.Keys() }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.binding()
			if len(got) == 0 {
				t.Fatalf("%s binding has no keys", tt.name)
			}
			// Verify at least the first expected key is present
			found := false
			for _, g := range got {
				if g == tt.keys[0] {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("%s: expected key %q in %v", tt.name, tt.keys[0], got)
			}
		})
	}
}
