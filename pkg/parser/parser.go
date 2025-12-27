package parser

import (
	"bytes"
	"errors"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

// Object represents a minimal Kubernetes object with associated raw manifest text.
type Object struct {
	APIVersion string
	Kind       string
	Namespace  string
	Name       string
	Raw        string
}

// Parse parses Kubernetes YAML manifests from the given reader. It returns a slice
// of Objects preserving raw document text. Empty or comment-only documents are ignored.
func Parse(r io.Reader) ([]Object, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	segments := splitYAMLDocuments(data)
	objects := make([]Object, 0, len(segments))
	for _, seg := range segments {
		trimmed := strings.TrimSpace(string(seg))
		if trimmed == "" { // skip empty
			continue
		}
		if isCommentOnly(trimmed) {
			continue
		}
		obj, err := decodeObject([]byte(trimmed))
		if err != nil {
			// Skip documents that are clearly not Kubernetes style objects rather than failing everything.
			continue
		}
		obj.Raw = trimmed + "\n" // ensure trailing newline for consistency
		objects = append(objects, obj)
	}
	return objects, nil
}

// splitYAMLDocuments splits a multi-document YAML payload into individual document byte slices.
func splitYAMLDocuments(data []byte) [][]byte {
	var docs [][]byte
	var current bytes.Buffer
	lines := bytes.Split(data, []byte("\n"))
	for i, line := range lines {
		trim := strings.TrimSpace(string(line))
		if trim == "---" { // document boundary
			if current.Len() > 0 {
				b := bytes.TrimRight(current.Bytes(), "\n")
				// copy to avoid aliasing with future buffer writes
				copyBuf := append([]byte(nil), b...)
				docs = append(docs, copyBuf)
				current.Reset()
			}
			continue
		}
		current.Write(line)
		// Re-add newline except possibly last line
		if i < len(lines)-1 {
			current.WriteByte('\n')
		}
	}
	if current.Len() > 0 {
		b := bytes.TrimRight(current.Bytes(), "\n")
		copyBuf := append([]byte(nil), b...)
		docs = append(docs, copyBuf)
	}
	return docs
}

type k8sMeta struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

type k8sMinimal struct {
	APIVersion string  `yaml:"apiVersion"`
	Kind       string  `yaml:"kind"`
	Metadata   k8sMeta `yaml:"metadata"`
}

func decodeObject(data []byte) (Object, error) {
	var m k8sMinimal
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(false)
	if err := dec.Decode(&m); err != nil {
		return Object{}, err
	}
	if m.APIVersion == "" && m.Kind == "" { // not a k8s object; require at least one
		return Object{}, errors.New("not a k8s style object")
	}
	return Object{
		APIVersion: m.APIVersion,
		Kind:       m.Kind,
		Namespace:  m.Metadata.Namespace,
		Name:       m.Metadata.Name,
	}, nil
}

func isCommentOnly(doc string) bool {
	lines := strings.Split(doc, "\n")
	for _, l := range lines {
		trim := strings.TrimSpace(l)
		if trim != "" && !strings.HasPrefix(trim, "#") {
			return false
		}
	}
	return true
}
