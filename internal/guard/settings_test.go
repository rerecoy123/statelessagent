package guard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultGuardConfig(t *testing.T) {
	cfg := DefaultGuardConfig()
	if !cfg.Enabled {
		t.Error("expected default guard to be enabled")
	}
	if !cfg.PII.Enabled {
		t.Error("expected default PII to be enabled")
	}
	if !cfg.PII.Patterns.Email {
		t.Error("expected email pattern enabled by default")
	}
	if !cfg.PII.Patterns.LocalPath {
		t.Error("expected local_path pattern enabled by default")
	}
	if cfg.SoftMode != "block" {
		t.Errorf("expected default soft_mode 'block', got %q", cfg.SoftMode)
	}
}

func TestGuardConfig_SetKey(t *testing.T) {
	cfg := DefaultGuardConfig()

	// Disable email
	if err := cfg.SetKey("email", "off"); err != nil {
		t.Fatal(err)
	}
	if cfg.PII.Patterns.Email {
		t.Error("expected email to be off after SetKey")
	}

	// Re-enable
	if err := cfg.SetKey("email", "on"); err != nil {
		t.Fatal(err)
	}
	if !cfg.PII.Patterns.Email {
		t.Error("expected email to be on after SetKey")
	}

	// Master switch
	if err := cfg.SetKey("guard", "off"); err != nil {
		t.Fatal(err)
	}
	if cfg.Enabled {
		t.Error("expected guard disabled")
	}

	// Soft mode
	if err := cfg.SetKey("soft-mode", "warn"); err != nil {
		t.Fatal(err)
	}
	if cfg.SoftMode != "warn" {
		t.Errorf("expected soft_mode 'warn', got %q", cfg.SoftMode)
	}

	// Invalid soft mode
	if err := cfg.SetKey("soft-mode", "invalid"); err == nil {
		t.Error("expected error for invalid soft-mode value")
	}

	// Unknown key
	if err := cfg.SetKey("nonexistent", "on"); err == nil {
		t.Error("expected error for unknown key")
	}

	// Path filter variants
	if err := cfg.SetKey("path-filter", "off"); err != nil {
		t.Fatal(err)
	}
	if cfg.PathFilter.Enabled {
		t.Error("expected path-filter off")
	}
	if err := cfg.SetKey("path_filter", "on"); err != nil {
		t.Fatal(err)
	}
	if !cfg.PathFilter.Enabled {
		t.Error("expected path_filter on")
	}
}

func TestEnabledPatternNames(t *testing.T) {
	cfg := DefaultGuardConfig()
	names := cfg.EnabledPatternNames()

	// All patterns should be present
	expected := []string{"email", "us_phone", "ssn", "local_path_unix", "local_path_windows",
		"api_key_assignment", "sk_key", "aws_key", "private_key_header"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected pattern %q to be enabled", name)
		}
	}

	// Disable phone
	cfg.PII.Patterns.Phone = false
	names = cfg.EnabledPatternNames()
	if names["us_phone"] {
		t.Error("expected us_phone to be disabled")
	}

	// Disable PII entirely
	cfg.PII.Enabled = false
	names = cfg.EnabledPatternNames()
	if names != nil {
		t.Error("expected nil when PII disabled")
	}

	// Disable guard entirely
	cfg.Enabled = false
	names = cfg.EnabledPatternNames()
	if names != nil {
		t.Error("expected nil when guard disabled")
	}
}

func TestGuardConfig_SaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	cfg := DefaultGuardConfig()
	cfg.PII.Patterns.Email = false
	cfg.SoftMode = "warn"

	// Simulate save: write config file with guard field
	type fullCfg struct {
		MachineName string       `json:"machine_name,omitempty"`
		Guard       *GuardConfig `json:"guard,omitempty"`
	}
	fc := fullCfg{MachineName: "test-machine", Guard: &cfg}
	data, err := json.MarshalIndent(fc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	// Simulate load
	readData, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	var loaded fullCfg
	if err := json.Unmarshal(readData, &loaded); err != nil {
		t.Fatal(err)
	}

	if loaded.MachineName != "test-machine" {
		t.Errorf("expected machine_name preserved, got %q", loaded.MachineName)
	}
	if loaded.Guard == nil {
		t.Fatal("expected guard config to be present")
	}
	if loaded.Guard.PII.Patterns.Email {
		t.Error("expected email to be off after round-trip")
	}
	if loaded.Guard.SoftMode != "warn" {
		t.Errorf("expected soft_mode 'warn', got %q", loaded.Guard.SoftMode)
	}
	if !loaded.Guard.PII.Patterns.Phone {
		t.Error("expected phone to remain on after round-trip")
	}
}

func TestFilterByConfig_AllEnabled(t *testing.T) {
	cfg := DefaultGuardConfig()
	patterns := builtinPatterns()
	filtered := FilterByConfig(patterns, cfg.EnabledPatternNames())
	if len(filtered) != len(patterns) {
		t.Errorf("expected all %d patterns, got %d", len(patterns), len(filtered))
	}
}

func TestFilterByConfig_DisableOne(t *testing.T) {
	cfg := DefaultGuardConfig()
	cfg.PII.Patterns.Phone = false
	patterns := builtinPatterns()
	filtered := FilterByConfig(patterns, cfg.EnabledPatternNames())
	for _, p := range filtered {
		if p.Name == "us_phone" {
			t.Error("expected us_phone to be filtered out")
		}
	}
	// Should have one fewer
	if len(filtered) != len(patterns)-1 {
		t.Errorf("expected %d patterns, got %d", len(patterns)-1, len(filtered))
	}
}

func TestFilterByConfig_DisableLocalPath(t *testing.T) {
	cfg := DefaultGuardConfig()
	cfg.PII.Patterns.LocalPath = false
	patterns := builtinPatterns()
	filtered := FilterByConfig(patterns, cfg.EnabledPatternNames())
	for _, p := range filtered {
		if p.Name == "local_path_unix" || p.Name == "local_path_windows" {
			t.Errorf("expected %s to be filtered out", p.Name)
		}
	}
	// Should have two fewer (unix + windows)
	if len(filtered) != len(patterns)-2 {
		t.Errorf("expected %d patterns, got %d", len(patterns)-2, len(filtered))
	}
}

func TestFilterByConfig_NilWhenDisabled(t *testing.T) {
	filtered := FilterByConfig(builtinPatterns(), nil)
	if filtered != nil {
		t.Error("expected nil when enabled map is nil")
	}
}
