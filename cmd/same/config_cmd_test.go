package main

import "testing"

func TestRunEditor_LookPathValidation(t *testing.T) {
	err := runEditor("definitely-not-a-real-editor-binary", "/tmp/same-config-test.toml")
	if err == nil {
		t.Fatal("expected runEditor to fail for missing editor binary")
	}
}
