package hooks

import "testing"

func TestValidatePlugin_ControlCharsInCommandRejected(t *testing.T) {
	p := PluginConfig{Command: "go\t"}
	if err := validatePlugin(p); err == nil {
		t.Fatal("expected control-character command to be rejected")
	}
}

func TestValidatePlugin_ControlCharsInArgsRejected(t *testing.T) {
	p := PluginConfig{Command: "go", Args: []string{"test\t./..."}}
	if err := validatePlugin(p); err == nil {
		t.Fatal("expected control-character arg to be rejected")
	}
}
