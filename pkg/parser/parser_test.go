package parser

import (
	"strings"
	"testing"
)

func TestParse_MultiDocument(t *testing.T) {
	input := `
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
  namespace: default
---
# comment only doc
# still comment
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
`
	objs, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(objs) != 2 { // comment-only ignored
		for i, o := range objs {
			t.Logf("obj[%d]=%+v", i, o)
		}
		t.Fatalf("expected 2 objects, got %d", len(objs))
	}
	if objs[0].Kind != "ConfigMap" || objs[0].Namespace != "default" || objs[0].Name != "my-config" {
		t.Errorf("unexpected first object: %+v", objs[0])
	}
	if objs[1].Kind != "Deployment" || objs[1].Namespace != "" || objs[1].Name != "web" {
		t.Errorf("unexpected second object: %+v", objs[1])
	}
	// Raw should contain trailing newline
	if !strings.HasSuffix(objs[0].Raw, "\n") {
		t.Errorf("expected Raw to end with newline")
	}
}

func TestIsCommentOnly(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"# just a comment", true},
		{"# c1\n# c2\n", true},
		{"# c1\nkind: Pod", false},
		{"\n# c1\n", true},
	}
	for _, c := range cases {
		if got := isCommentOnly(c.in); got != c.want {
			t.Errorf("isCommentOnly(%q)=%v want %v", c.in, got, c.want)
		}
	}
}
