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
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
)

func TestNewConditionalHandler(t *testing.T) {
	mustCreate := func(t *testing.T, opts ...conditionalHandlerOpt) *ConditionalHandler {
		t.Helper()
		h, err := NewConditionalHandler(opts...)
		if err != nil {
			t.Fatalf("unable to create ConditionalHandler: %s", err)
		}
		return h
	}

	tests := map[string]visitorScenario[ConditionalHandler]{
		"handles documents": {
			input: "document: 1",
			handler: mustCreate(t, OnVisitDocumentNode(func(ctx context.Context, value *yaml.Node) error {
				// pre-check: t here is the test parent, not this individual test
				assert.Equal(t, yaml.DocumentNode, value.Kind)
				value.HeadComment = "test: handles documents"
				return nil
			})),
			validatorWithNode: func(t *testing.T, h ConditionalHandler, node *yaml.Node) error {
				assert.Equal(t, yaml.DocumentNode, node.Kind)
				assert.Equal(t, "test: handles documents", node.HeadComment)
				return nil
			},
		},
		"handles sequences": {
			input: trimmed(`document:
							| first: [1,2,3]
							| second: [1,2,3]`),
			handler: mustCreate(t, OnVisitSequenceNode("$..first", func(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
				key.HeadComment = "test: handles sequences"
				return nil
			})),
			validatorWithNode: func(t *testing.T, h ConditionalHandler, node *yaml.Node) error {
				firstKey := node.Content[0].Content[1 /*document's value*/].Content[0]
				firstValue := node.Content[0].Content[1 /*document's value*/].Content[1]
				secondKey := node.Content[0].Content[1 /*document's value*/].Content[2]
				secondValue := node.Content[0].Content[1 /*document's value*/].Content[3]
				text := "test: handles sequences"
				assert.Equal(t, text, firstKey.HeadComment)
				assert.NotEqual(t, text, secondKey.HeadComment)
				assert.NotEqual(t, text, firstValue.HeadComment)
				assert.NotEqual(t, text, secondValue.HeadComment)
				return nil
			},
		},
		"handles mappings": {
			input: trimmed(`---
					|store:
					|  book:
					|  - author: Ernest Hemingway
					|    title: The Old Man and the Sea
					|  - author: Fyodor Mikhailovich Dostoevsky
					|    title: Crime and Punishment
					|  - author: Jane Austen
					|    title: Sense and Sensibility
					|  - author: Kurt Vonnegut Jr.
					|    title: Slaughterhouse-Five
					|  - author: J. R. R. Tolkien
					|    title: The Lord of the Rings`),
			handler: mustCreate(t, OnVisitMappingNode("$.store.book[?(@.title=~/^S.*$/)]", func(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
				assert.Nil(t, key, "Expected key for a mapping node within a sequence to be nil.")
				value.HeadComment = "testing: handles mappings"
				return nil
			})),
			validatorWithNode: func(t *testing.T, h ConditionalHandler, node *yaml.Node) error {
				assert.Equal(t, yaml.DocumentNode, node.Kind)
				yp, _ := yamlpath.NewPath(".store.book[*]")
				nodes, _ := yp.Find(node)
				found := 0
				for _, n := range nodes {
					if n.HeadComment == "testing: handles mappings" {
						found += 1
					}

					// our handler is expected only to apply the target comment if the title starts with a capital S
					for i := 0; i < len(n.Content); i += 2 {
						key := n.Content[i]
						val := n.Content[i+1]
						if key.Value == "title" && strings.HasPrefix(val.Value, "S") {
							assert.Equal(t, "testing: handles mappings", n.HeadComment)
						}
					}
				}
				assert.Equal(t, 2, found)
				return nil
			},
		},
		"handles scalars": {
			input: trimmed(`---
					|store:
					|  book:
					|  - author: Ernest Hemingway
					|    title: The Old Man and the Sea
					|  - author: Fyodor Mikhailovich Dostoevsky
					|    title: Crime and Punishment
					|  - author: Jane Austen
					|    title: Sense and Sensibility
					|  - author: Kurt Vonnegut Jr.
					|    title: Slaughterhouse-Five
					|  - author: J. R. R. Tolkien
					|    title: The Lord of the Rings`),
			handler: mustCreate(t, OnVisitScalarNode("$.store.book[?(@.title=~/^S.*$/)].title", func(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
				assert.Equal(t, "title", key.Value, "Shouldn't be processing any other scalars besides title")
				key.HeadComment = "testing: handles scalars"
				return nil
			})),
			validatorWithNode: func(t *testing.T, h ConditionalHandler, node *yaml.Node) error {
				assert.Equal(t, yaml.DocumentNode, node.Kind)
				yp, _ := yamlpath.NewPath("$.store.book")
				nodes, _ := yp.Find(node)
				found := 0
				for _, books := range nodes {
					for i := 0; i < len(books.Content); i += 1 {
						book := books.Content[i]
						for i := 0; i < len(book.Content); i += 2 {
							key := book.Content[i]
							val := book.Content[i+1]
							// our handler is expected only to apply the target comment if the title starts with a capital S
							if key.Value == "title" && strings.HasPrefix(val.Value, "S") {
								assert.Equal(t, "testing: handles scalars", key.HeadComment)
								found += 1
							}
						}
					}
				}
				assert.Equal(t, 2, found)
				return nil
			},
		},
		"handles aliases":   {},
		"handles multiples": {},
		"handles errors": {
			wantErr: true,
			input: trimmed(`---
					|store:
					|  book:
					|  - author: Ernest Hemingway
					|    title: The Old Man and the Sea
					|  - author: Fyodor Mikhailovich Dostoevsky
					|    title: Crime and Punishment
					|  - author: Jane Austen
					|    title: Sense and Sensibility
					|  - author: Kurt Vonnegut Jr.
					|    title: Slaughterhouse-Five
					|  - author: J. R. R. Tolkien
					|    title: The Lord of the Rings`),
			handler: mustCreate(t, OnVisitScalarNode("$.store.book[?(@.title=~/^S.*$/)].title", func(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
				// selector will select two nodes, we will apply a marker head comment to the first one, then error
				if value.Value == "Sense and Sensibility" {
					key.HeadComment = "testing: handles errors"
					return errors.New("it makes no sense")
				}
				return nil
			})),
			validatorWithNode: func(t *testing.T, h ConditionalHandler, node *yaml.Node) error {
				assert.Equal(t, yaml.DocumentNode, node.Kind)
				yp, _ := yamlpath.NewPath("$.store.book")
				nodes, _ := yp.Find(node)
				found := 0
				for _, books := range nodes {
					for i := 0; i < len(books.Content); i += 1 {
						book := books.Content[i]
						for i := 0; i < len(book.Content); i += 2 {
							key := book.Content[i]
							val := book.Content[i+1]
							if val.Value == "Sense and Sensibility" {
								assert.Equal(t, "testing: handles errors", key.HeadComment, "Handler should only apply to 'Sense and Sensibility' for this test.")
								found += 1
							}
						}
					}
				}
				assert.Equal(t, 1, found, "Handler should only apply to 'Sense and Sensibility' for this test. After raising the error, processing should halt.")
				return nil
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := NewVisitor(tt.handler)
			assert.NoError(t, err)
			ctx, cancel := context.WithDeadline(context.TODO(), time.Now().Add(5*time.Minute))
			defer cancel()

			d := yaml.NewDecoder(bytes.NewReader([]byte(tt.input)))
			var last *yaml.Node
			for {
				node := new(yaml.Node)
				err := d.Decode(node)
				if errors.Is(err, io.EOF) {
					break
				}
				last = node

				err = got.Visit(ctx, node)
				if (err != nil) != tt.wantErr {
					t.Errorf("NewConditionalHandler() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
			}

			if tt.validator != nil {
				assert.NoError(t, tt.validator(t, *tt.handler))
			}
			if tt.validatorWithNode != nil {
				assert.NoError(t, tt.validatorWithNode(t, *tt.handler, last))
			}
		})
	}
}
