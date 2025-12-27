package fzf

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/TaliaMarine/kustomize-fzf/pkg/parser"
)

const (
	// Update to green coloring for (apiVersion.)Kind
	kindColorPrefix = "\x1b[32m" // green
	colorReset      = "\x1b[0m"
)

// Options encapsulates configuration for running fzf selection.
type Options struct {
	BinaryPath     string // path to fzf (optional, will search PATH if empty)
	Multi          bool   // allow selecting multiple objects
	DisableYQ      bool   // if true, do not attempt to use yq for formatting
	DisableAlign   bool   // legacy; retained for compatibility (no-op in new format)
	ShowAPIVersion bool   // show apiVersion prefix before Kind
	ShowNamespace  bool   // show namespace prefix before Name
}

var (
	yqOnce sync.Once
	yqPath string
)

func detectYQ() string {
	yqOnce.Do(func() {
		if p, err := exec.LookPath("yq"); err == nil {
			yqPath = p
		}
	})
	return yqPath
}

// Select launches fzf to let the user choose one or more Kubernetes objects.
// Returns the selected objects (in original order of appearance) or error.
func Select(objs []parser.Object, opts Options) ([]parser.Object, error) {
	if len(objs) == 0 {
		return nil, errors.New("no objects to select")
	}
	fzfBin := opts.BinaryPath
	if fzfBin == "" {
		if env := os.Getenv("kustomize-fzf_FZF_BIN"); env != "" {
			fzfBin = env
		} else {
			var err error
			fzfBin, err = exec.LookPath("fzf")
			if err != nil {
				return nil, errors.New("fzf executable not found in PATH; install fzf or set kustomize-fzf_FZF_BIN")
			}
		}
	}

	yqAvailable := false
	if !opts.DisableYQ && os.Getenv("kustomize-fzf_NO_YQ") == "" {
		if detectYQ() != "" {
			yqAvailable = true
		}
	}

	// Prepare temp directory with per-object files for preview.
	tmpDir, err := os.MkdirTemp("", "kustomize-fzf-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	for i, o := range objs {
		fname := filepath.Join(tmpDir, fmt.Sprintf("kustomize-fzf-%d.yaml", i))
		if err := os.WriteFile(fname, []byte(o.Raw), 0o600); err != nil {
			return nil, fmt.Errorf("write temp file: %w", err)
		}
	}

	lines := buildFZFInputLines(objs, opts.ShowAPIVersion, opts.ShowNamespace)

	var cmdArgs []string
	if opts.Multi {
		cmdArgs = append(cmdArgs, "--multi")
	}
	cmdArgs = append(cmdArgs,
		"--ansi",
		"--delimiter", "\t",
		"--with-nth=2..", // hide internal index (first tab-delimited field is internal id)
	)

	var previewCmd string
	if yqAvailable {
		colorFlag := ""
		if os.Getenv("kustomize-fzf_NO_COLOR") == "" { // allow disabling color explicitly
			colorFlag = "-C "
		}
		previewCmd = fmt.Sprintf("yq %s'.' '%s'/kustomize-fzf-{1}.yaml 2>/dev/null || cat '%s'/kustomize-fzf-{1}.yaml", colorFlag, tmpDir, tmpDir)
	} else {
		previewCmd = fmt.Sprintf("cat '%s'/kustomize-fzf-{1}.yaml", tmpDir)
	}
	cmdArgs = append(cmdArgs,
		"--preview", previewCmd,
		"--preview-window", "right:60%:wrap",
	)

	cmd := exec.Command(fzfBin, cmdArgs...)
	cmd.Env = append(os.Environ(), "kustomize-fzf_TMP="+tmpDir)
	cmd.Stdin = bytes.NewReader([]byte(strings.Join(lines, "\n") + "\n"))
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// If user aborts (Ctrl-C / ESC) fzf exits non-zero; treat as no selection.
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() != 0 {
			return nil, errors.New("selection aborted")
		}
		return nil, err
	}

	selections := parseSelectionOutput(out.String())
	if len(selections) == 0 {
		return nil, errors.New("no selection")
	}
	// Map indices back to objects; preserve original order of indices appearance.
	chosen := make([]parser.Object, 0, len(selections))
	for _, idx := range selections {
		if idx < 0 || idx >= len(objs) {
			continue
		}
		chosen = append(chosen, objs[idx])
	}
	return chosen, nil
}

// buildFZFInputLines constructs tab-delimited lines for fzf where the first (hidden) column
// is the internal index followed by a single display field of the form:
//
//	(apiVersion.)Kind (Namespace.)Name
//
// apiVersion and Namespace segments are included only when their respective flags are true
// and the value exists.
func buildFZFInputLines(objs []parser.Object, showAPI, showNS bool) []string {
	lines := make([]string, 0, len(objs))
	noColor := os.Getenv("kustomize-fzf_NO_COLOR") != ""
	for i, o := range objs {
		kindPart := o.Kind
		if kindPart == "" {
			kindPart = "-"
		}
		if showAPI && o.APIVersion != "" {
			kindPart = o.APIVersion + "." + kindPart
		}
		if !noColor {
			kindPart = kindColorPrefix + kindPart + colorReset
		}
		namePart := o.Name
		if namePart == "" {
			namePart = "-"
		}
		if showNS && o.Namespace != "" {
			namePart = o.Namespace + "." + namePart
		}
		display := kindPart + " " + namePart
		lines = append(lines, fmt.Sprintf("%d\t%s", i, display))
	}
	return lines
}

func stringOrDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func parseSelectionOutput(out string) []int {
	var idxs []int
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		fields := strings.Split(l, "\t")
		if len(fields) == 0 {
			continue
		}
		var idx int
		_, err := fmt.Sscanf(fields[0], "%d", &idx)
		if err == nil {
			idxs = append(idxs, idx)
		}
	}
	return idxs
}

// WriteSelected concatenates selected objects' raw manifests to writer, separated by `---`.
// If yq is available, each document is passed through `yq -r '.'` for normalized formatting.
func WriteSelected(w io.Writer, objs []parser.Object) error {
	yq := detectYQ()
	useColor := os.Getenv("kustomize-fzf_NO_COLOR") == ""
	for i, o := range objs {
		if i > 0 {
			if _, err := io.WriteString(w, "---\n"); err != nil {
				return err
			}
		}
		content := o.Raw
		if yq != "" && os.Getenv("kustomize-fzf_NO_YQ") == "" {
			formatted, err := runYQ(yq, content, useColor)
			if err == nil && strings.TrimSpace(formatted) != "" {
				// Ensure trailing newline
				if !strings.HasSuffix(formatted, "\n") {
					formatted += "\n"
				}
				content = formatted
			}
		}
		if _, err := io.WriteString(w, content); err != nil {
			return err
		}
	}
	return nil
}

// runYQ executes yq with either color (-C) or raw (-r) formatting.
func runYQ(yqPath, input string, color bool) (string, error) {
	modeFlag := "-r"
	if color {
		modeFlag = "-C"
	}
	cmd := exec.Command(yqPath, modeFlag, ".")
	cmd.Stdin = strings.NewReader(input)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = new(bytes.Buffer)
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return out.String(), nil
}
