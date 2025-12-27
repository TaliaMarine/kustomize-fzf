package cmd

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/TaliaMarine/kustomize-fzf/pkg/fzf"
	"github.com/TaliaMarine/kustomize-fzf/pkg/parser"
)

var (
	flagMulti         = flag.Bool("multi", false, "allow selecting multiple manifests")
	flagVersion       = flag.Bool("version", false, "print version and exit")
	flagNoAlign       = flag.Bool("no-align", false, "(legacy no-op) previously disabled column alignment")
	flagNoYQ          = flag.Bool("no-yq", false, "disable yq formatting (preview & output)")
	flagShowAPI       = flag.Bool("show-apiversion", false, "show apiVersion prefix before Kind")
	flagShowNamespace = flag.Bool("show-namespace", false, "show namespace prefix before Name")
)

// version is injected at build-time via -ldflags "-X main.version=..."
var version = "dev"

func Entry() {
	flag.Parse()
	if *flagVersion {
		fmt.Println(version)
		return
	}

	info, err := os.Stdin.Stat()
	if err != nil {
		log.Fatalf("stat stdin: %v", err)
	}
	if info.Mode()&os.ModeCharDevice != 0 {
		usage()
		os.Exit(1)
	}

	objects, err := parser.Parse(os.Stdin)
	if err != nil {
		log.Fatalf("parse error: %v", err)
	}
	if len(objects) == 0 {
		log.Fatalf("no Kubernetes objects found in input")
	}

	// Manage yq disable via environment so both preview and WriteSelected honor it.
	restoreYQ := ensureYQDisable(*flagNoYQ)
	defer restoreYQ()

	selected, err := fzf.Select(objects, fzf.Options{Multi: *flagMulti, DisableAlign: *flagNoAlign, DisableYQ: *flagNoYQ, ShowAPIVersion: *flagShowAPI, ShowNamespace: *flagShowNamespace})
	if err != nil {
		log.Fatalf("selection: %v", err)
	}
	if err := fzf.WriteSelected(os.Stdout, selected); err != nil {
		log.Fatalf("write: %v", err)
	}
}

func usage() {
	w := io.Discard
	if flag.CommandLine.Output() != nil {
		w = flag.CommandLine.Output()
	}
	fmt.Fprintf(w, `kustomize-fzf - interactively filter Kubernetes manifests via fzf\n\n`)
	fmt.Fprintf(w, `Usage: cat manifests.yaml | kustomize-fzf [--multi] [--show-apiversion] [--show-namespace] [--no-yq]\n`)
	fmt.Fprintf(w, `Options:\n`)
	flag.PrintDefaults()
}

func ensureYQDisable(disable bool) func() {
	if !disable {
		return func() {}
	}
	prev, had := os.LookupEnv("kustomize-fzf_NO_YQ")
	_ = os.Setenv("kustomize-fzf_NO_YQ", "1")
	return func() {
		if had {
			_ = os.Setenv("kustomize-fzf_NO_YQ", prev)
		} else {
			_ = os.Unsetenv("kustomize-fzf_NO_YQ")
		}
	}
}
