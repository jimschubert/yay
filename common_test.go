package yay

import (
	"bytes"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type visitorScenario[T any] struct {
	handler           *T
	input             string
	validator         func(t *testing.T, h T) error
	validatorWithNode func(t *testing.T, h T, node *yaml.Node) error
	wantErr           bool
}

type pair struct {
	left  *yaml.Node
	right *yaml.Node
}

func trimMargin(input string, cutset string) string {
	b := bytes.Buffer{}
	for _, s := range strings.FieldsFunc(input, func(r rune) bool {
		return r == '\n'
	}) {
		b.WriteString(strings.TrimLeft(s, cutset))
		b.WriteRune('\n')
	}
	return b.String()
}

func trimmed(input string) string {
	return trimMargin(input, "\t|")
}
