package yay

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

// captor is a context key allowing for a verify of handler details for post-run verifications
type captor struct{}

// captorFunction allows registration of any test verifications which run within the context of an individual test case
type captorFunction func(t *testing.T)

// visitorScenario defines common behavior binding an input YAML document to a handler and setting up common yay.Visitor behaviors.
// A test should expect an error or define either a validator, validatorWithNode or requireVerifyCount
type visitorScenario[T any] struct {
	handler            *T
	input              string
	validator          func(t *testing.T, h T) error
	validatorWithNode  func(t *testing.T, h T, node *yaml.Node) error
	wantErr            assert.ErrorAssertionFunc
	requireVerifyCount int
}

type pair struct {
	left  *yaml.Node
	right *yaml.Node
}

func verify(ctx context.Context, fn captorFunction) {
	if fns, ok := ctx.Value(captor{}).(*[]captorFunction); ok {
		*fns = append(*fns, fn)
	}
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

func validateScenario[T any](t *testing.T, parent context.Context, scenario visitorScenario[T]) {
	got, err := NewVisitor(scenario.handler)
	assert.NoError(t, err)
	ctx, cancel := context.WithDeadline(parent, time.Now().Add(5*time.Minute))
	defer cancel()

	var errFn assert.ErrorAssertionFunc
	if scenario.wantErr == nil {
		errFn = assert.NoError
	} else {
		errFn = scenario.wantErr
	}

	captures := make([]captorFunction, 0)
	ctx = context.WithValue(ctx, captor{}, &captures)

	d := yaml.NewDecoder(bytes.NewReader([]byte(scenario.input)))
	var last *yaml.Node

	for {
		node := new(yaml.Node)
		err := d.Decode(node)
		if errors.Is(err, io.EOF) {
			break
		}
		assert.NoError(t, err)
		last = node

		err = got.Visit(ctx, node)

		if !errFn(t, err, "%s | error = %v", t.Name(), err) {
			break
		}
	}

	if scenario.validator != nil {
		assert.NoError(t, scenario.validator(t, *scenario.handler))
	}
	if scenario.validatorWithNode != nil {
		assert.NoError(t, scenario.validatorWithNode(t, *scenario.handler, last))
	}

	count := len(captures)
	assert.Equal(t, scenario.requireVerifyCount, count)
	for _, fn := range captures {
		fn(t)
	}

	if scenario.wantErr == nil && (scenario.validator == nil && scenario.validatorWithNode == nil && scenario.requireVerifyCount == 0) {
		t.Fatal("non-error scenarios must define a validator or invoke verify and define requireVerifyCount for post-test verifications.")
	}
}
