package yay

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

type traverseAll struct {
	documents        []*yaml.Node
	sequences        []*yaml.Node
	mappings         []pair
	scalars          []pair
	aliases          []pair
	expectsSequences bool
	expectsMappings  bool
	expectsScalars   bool
	expectsAliases   bool
	abortOnError     bool
	canceler         context.CancelFunc
}

func (t *traverseAll) VisitDocumentNode(ctx context.Context, key *yaml.Node) error {
	kind := key.Content[0].Kind
	if kind == yaml.MappingNode {
		// we'll just store the first node as a document indicator
		t.documents = append(t.documents, key.Content[0].Content[0])
	} else if kind == yaml.SequenceNode {
		// store the sequence itself
		t.documents = append(t.documents, key.Content[0])
	}
	return nil
}

func (t *traverseAll) VisitSequenceNode(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
	if !t.expectsSequences {
		if t.abortOnError {
			go func() { t.canceler() }()
		}
		return errors.New("Did not expect to process sequences")
	}
	t.sequences = append(t.sequences, key)
	return nil
}

func (t *traverseAll) VisitMappingNode(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
	if !t.expectsMappings {
		if t.abortOnError {
			go func() { t.canceler() }()
		}
		return errors.New("Did not expect to process mappings")
	}
	t.mappings = append(t.mappings, pair{key, value})
	return nil
}

func (t *traverseAll) VisitScalarNode(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
	if !t.expectsScalars {
		if t.abortOnError {
			go func() { t.canceler() }()
		}
		return errors.New("Did not expect to process scalars")
	}
	t.scalars = append(t.scalars, pair{key, value})
	return nil
}

func (t *traverseAll) VisitAliasNode(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
	if !t.expectsAliases {
		if t.abortOnError {
			go func() { t.canceler() }()
		}
		return errors.New("Did not expect to process aliases")
	}
	t.aliases = append(t.aliases, pair{key, value})
	return nil
}

var (
	empty                    = (*traverseAll)(nil)
	_     VisitsSequenceNode = empty
	_     VisitsScalarNode   = empty
	_     VisitsMappingNode  = empty
	_     VisitsAliasNode    = empty
	_     VisitsDocumentNode = empty
	_     VisitsYaml         = empty
)

func TestVisitorTraversals(t *testing.T) {
	tests := map[string]visitorScenario[traverseAll]{

		"handles empty documents": {
			handler: &traverseAll{},
			input:   "",
			validator: func(t *testing.T, h traverseAll) error {
				assert.Equal(t, 0, len(h.documents))
				return nil
			},
		},

		"handles single documents": {
			handler: &traverseAll{
				expectsMappings: true,
				expectsScalars:  true,
			},
			input: trimmed(`document:
				|  first: "1st"
				|  second: "2nd"`),
			validator: func(t *testing.T, h traverseAll) error {
				assert.Equal(t, 1, len(h.documents))
				assert.Equal(t, 2, len(h.scalars))

				assert.Equal(t, "document", h.documents[0].Value)

				first := h.scalars[0]
				assert.Equal(t, "first", first.left.Value)
				assert.Equal(t, "1st", first.right.Value)

				second := h.scalars[1]
				assert.Equal(t, "second", second.left.Value)
				assert.Equal(t, "2nd", second.right.Value)
				return nil
			},
		},

		"handles multiple documents": {
			handler: &traverseAll{
				expectsMappings: true,
				expectsScalars:  true,
			},
			input: trimmed(`# first document
				|document:
				|  first: "1st"
				|  second: "2nd"
				|---
				|# second document
				|document2:
				|  A: "a"
				|  B: "b"`),
			validator: func(t *testing.T, h traverseAll) error {
				assert.Equal(t, 2, len(h.documents))
				assert.Equal(t, 4, len(h.scalars))

				first := h.scalars[0]
				assert.Equal(t, "first", first.left.Value)
				assert.Equal(t, "1st", first.right.Value)

				second := h.scalars[1]
				assert.Equal(t, "second", second.left.Value)
				assert.Equal(t, "2nd", second.right.Value)

				a := h.scalars[2]
				assert.Equal(t, "A", a.left.Value)
				assert.Equal(t, "a", a.right.Value)

				b := h.scalars[3]
				assert.Equal(t, "B", b.left.Value)
				assert.Equal(t, "b", b.right.Value)
				return nil
			},
		},

		"handles multiple documents with empties": {
			handler: &traverseAll{
				expectsMappings: true,
				expectsScalars:  true,
			},
			input: trimmed(`---
				|# second document
				|message: hello`),
			validator: func(t *testing.T, h traverseAll) error {
				assert.Equal(t, 1, len(h.documents))
				assert.Equal(t, 1, len(h.scalars))

				first := h.scalars[0]
				assert.Equal(t, "message", first.left.Value)
				assert.Equal(t, "hello", first.right.Value)

				return nil
			},
		},

		"handles aborting on errors": {
			handler: &traverseAll{
				expectsMappings:  true,
				expectsScalars:   true,
				expectsSequences: false,
				abortOnError:     true,
			},
			input: trimmed(`document:
				|  first: [1,2,3]
				|  second: "2nd"`),
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorContains(t, err, "Did not expect to process sequences")
			},
			validator: func(t *testing.T, h traverseAll) error {
				assert.Equal(t, 1, len(h.documents))
				assert.Equal(t, 1, len(h.scalars))

				second := h.scalars[0]
				assert.Equal(t, "second", second.left.Value)
				assert.Equal(t, "2nd", second.right.Value)

				return nil
			},
		},

		"handles array-based documents": {
			handler: &traverseAll{
				expectsSequences: true,
				expectsScalars:   true,
				expectsMappings:  true,
			},
			input: trimmed(`---
				|- first: "1st"
				|- second: "2nd"`),
			validator: func(t *testing.T, h traverseAll) error {
				assert.Equal(t, 1, len(h.documents))
				assert.Equal(t, 0, len(h.sequences))
				assert.Equal(t, 2, len(h.mappings))
				assert.Equal(t, 2, len(h.scalars))

				assert.Equal(t, yaml.SequenceNode, h.documents[0].Kind)
				assert.Equal(t, 2, len(h.documents[0].Content))

				item1 := h.mappings[0]
				assert.Nil(t, item1.left, "No key should be passed on sequence map")
				assert.Equal(t, 2, len(item1.right.Content))
				assert.Equal(t, "first", item1.right.Content[0].Value)
				assert.Equal(t, "1st", item1.right.Content[1].Value)

				item2 := h.mappings[1]
				assert.Nil(t, item2.left, "No key should be passed on sequence map")
				assert.Equal(t, 2, len(item2.right.Content))
				assert.Equal(t, "second", item2.right.Content[0].Value)
				assert.Equal(t, "2nd", item2.right.Content[1].Value)

				first := h.scalars[0]
				assert.Equal(t, "first", first.left.Value)
				assert.Equal(t, "1st", first.right.Value)

				second := h.scalars[1]
				assert.Equal(t, "second", second.left.Value)
				assert.Equal(t, "2nd", second.right.Value)
				return nil
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.TODO())
			tt.handler.canceler = cancel
			validateScenario(t, ctx, tt)
		})
	}
}

func TestVisitorTraversals_composite_handlers(t *testing.T) {
	tests := map[string]visitorScenario[compositeHandler]{
		"supports multiple handlers": {
			handler: &compositeHandler{
				handlers: []any{
					&traverseAll{
						expectsMappings: true,
						expectsScalars:  true,
					},
					&traverseAll{
						expectsMappings: true,
						expectsScalars:  true,
					},
				},
			},
			input: trimmed(`document:
				|  first: "1st"
				|  second: "2nd"`),
			validator: func(t *testing.T, composite compositeHandler) error {
				iterations := 0
				for _, handler := range composite.handlers {
					h := handler.(*traverseAll)

					assert.Equal(t, 1, len(h.documents))
					assert.Equal(t, 2, len(h.scalars))

					assert.Equal(t, "document", h.documents[0].Value)

					first := h.scalars[0]
					assert.Equal(t, "first", first.left.Value)
					assert.Equal(t, "1st", first.right.Value)

					second := h.scalars[1]
					assert.Equal(t, "second", second.left.Value)
					assert.Equal(t, "2nd", second.right.Value)
					iterations++
				}
				assert.Equal(t, 2, iterations)
				return nil
			},
		},
		"raises first error": {
			handler: &compositeHandler{
				// first error should come from handler[1] lacking expectsMappings
				handlers: []any{
					&traverseAll{
						expectsMappings: true,
					},
					&traverseAll{
						expectsScalars: true,
					},
				},
			},
			input: trimmed(`document:
				|  first: "1st"
				|  second: "2nd"`),
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorContains(t, err, "Did not expect to process mappings")
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.TODO())
			defer cancel()
			for _, handler := range tt.handler.handlers {
				h := handler.(*traverseAll)
				h.canceler = cancel
			}
			validateScenario(t, ctx, tt)
		})
	}
}

func TestNewVisitorWithOptions(t *testing.T) {
	testHandler := func(handler conditionalHandlerOpt) []any {
		handle, err := NewConditionalHandler(
			handler,
			OnVisitDocumentNode(func(ctx context.Context, value *yaml.Node) error {
				t.Error("Should not treat as Document Node")
				return nil
			}),
		)
		if err != nil {
			t.Fatal(err)
		}
		return []any{
			handle,
		}
	}

	type args struct {
		options  FnOptions
		handlers []any
	}
	tests := []struct {
		name       string
		args       args
		targetNode *yaml.Node
		wantErr    assert.ErrorAssertionFunc
		validate   func(t *testing.T, node *yaml.Node)
	}{
		{
			name: "allows skipping document check via option, node document kind in parent",
			targetNode: &yaml.Node{
				Content: []*yaml.Node{
					{
						Kind:  yaml.ScalarNode,
						Value: "key",
					},
					{
						Kind:  yaml.ScalarNode,
						Value: "value",
					},
				},
			},
			wantErr: assert.NoError,
			args: args{
				options: NewOptions().WithSkipDocumentCheck(true),
				handlers: testHandler(
					OnVisitScalarNode("$.key", func(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
						key.Value = "updated"
						return nil
					}),
				),
			},
			validate: func(t *testing.T, node *yaml.Node) {
				assert.Equal(t, "updated", node.Content[0].Value)
			},
		},
		{
			name: "allows skipping document check via option, parent node is mapping",
			targetNode: &yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{
						Kind:  yaml.ScalarNode,
						Value: "key",
					},
					{
						Kind:  yaml.ScalarNode,
						Value: "value",
					},
				},
			},
			wantErr: assert.NoError,
			args: args{
				options: NewOptions().WithSkipDocumentCheck(true),
				handlers: testHandler(
					OnVisitScalarNode("$.key", func(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
						key.Value = "updated"
						return nil
					}),
				),
			},
			validate: func(t *testing.T, node *yaml.Node) {
				assert.Equal(t, "updated", node.Content[0].Value)
			},
		},
		{
			name: "allows skipping document check via option, sequences",
			targetNode: func() *yaml.Node {
				node := &yaml.Node{}
				assert.NoError(t, yaml.Unmarshal([]byte(`[1,2,3]`), node))

				return node.Content[0]
			}(),
			wantErr: assert.NoError,
			args: args{
				options: NewOptions().WithSkipDocumentCheck(true),
				handlers: testHandler(
					OnVisitScalarNode("$[0]", func(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
						value.Tag = "!!str"
						value.Value = "updated"
						return nil
					}),
				),
			},
			validate: func(t *testing.T, node *yaml.Node) {
				assert.Equal(t, "updated", node.Content[0].Value)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := NewVisitorWithOptions(tt.args.options, tt.args.handlers...)
			assert.NoError(t, err)
			if !tt.wantErr(t, v.Visit(context.Background(), tt.targetNode), fmt.Sprintf("NewVisitorWithOptions(%v, handlers)", tt.args.options)) {
				return
			}

			if tt.validate != nil {
				tt.validate(t, tt.targetNode)
			}
		})
	}
}
