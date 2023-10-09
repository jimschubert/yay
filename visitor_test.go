package yay

import (
	"context"
	"errors"
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
	t.documents = append(t.documents, key.Content[0].Content[0])
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
	empty                    = traverseAll{}
	_     VisitsSequenceNode = &empty
	_     VisitsScalarNode   = &empty
	_     VisitsMappingNode  = &empty
	_     VisitsAliasNode    = &empty
	_     VisitsDocumentNode = &empty
)

func TestVisitorTraversals(t *testing.T) {
	tests := map[string]visitorScenario[traverseAll]{
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

		"handles aborting on errors": {
			handler: &traverseAll{
				expectsMappings: true,
				expectsScalars:  true,
				abortOnError:    true,
			},
			wantErr: true,
			input: trimmed(`document:
				|  first: [1,2,3]
				|  second: "2nd"`),
			validator: func(t *testing.T, h traverseAll) error {
				assert.Equal(t, 1, len(h.documents))
				assert.Equal(t, 1, len(h.scalars))

				second := h.scalars[0]
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
