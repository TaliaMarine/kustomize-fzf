package fzf

import (
	"testing"

	"github.com/TaliaMarine/kustomize-fzf/pkg/parser"
)

func TestBuildFZFInputLines_Basic(t *testing.T) {
	// Disable color for stable assertions
	t.Setenv("kustomize-fzf_NO_COLOR", "1")
	objs := []parser.Object{
		{APIVersion: "v1", Kind: "ConfigMap", Namespace: "ns", Name: "cm1"},
		{APIVersion: "apps/v1", Kind: "Deployment", Name: "web"},
	}
	lines := buildFZFInputLines(objs, false, false)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines got %d", len(lines))
	}
	if lines[0] != "0\tConfigMap cm1" {
		t.Errorf("unexpected line0: %q", lines[0])
	}
	if lines[1] != "1\tDeployment web" {
		t.Errorf("unexpected line1: %q", lines[1])
	}
}

func TestBuildFZFInputLines_WithAPI(t *testing.T) {
	t.Setenv("kustomize-fzf_NO_COLOR", "1")
	objs := []parser.Object{{APIVersion: "apps/v1", Kind: "Deployment", Name: "web"}}
	lines := buildFZFInputLines(objs, true, false)
	if lines[0] != "0\tapps/v1.Deployment web" {
		t.Errorf("unexpected line: %q", lines[0])
	}
}

func TestBuildFZFInputLines_WithNamespace(t *testing.T) {
	t.Setenv("kustomize-fzf_NO_COLOR", "1")
	objs := []parser.Object{{APIVersion: "v1", Kind: "ConfigMap", Namespace: "ns", Name: "cm1"}}
	lines := buildFZFInputLines(objs, false, true)
	if lines[0] != "0\tConfigMap ns.cm1" {
		t.Errorf("unexpected line: %q", lines[0])
	}
}

func TestBuildFZFInputLines_WithAPIAndNamespace(t *testing.T) {
	t.Setenv("kustomize-fzf_NO_COLOR", "1")
	objs := []parser.Object{{APIVersion: "apps/v1", Kind: "Deployment", Namespace: "default", Name: "web"}}
	lines := buildFZFInputLines(objs, true, true)
	if lines[0] != "0\tapps/v1.Deployment default.web" {
		t.Errorf("unexpected line: %q", lines[0])
	}
}

func TestParseSelectionOutput(t *testing.T) {
	out := "0\tapps/v1.Deployment default.web\n1\tConfigMap ns.cm1\n"
	idxs := parseSelectionOutput(out)
	if len(idxs) != 2 || idxs[0] != 0 || idxs[1] != 1 {
		t.Fatalf("unexpected indices: %#v", idxs)
	}
}
