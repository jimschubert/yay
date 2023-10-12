package yay

import (
	"context"
	"errors"
	"strings"
	"testing"

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

	commonDoc := trimmed(`---
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
					|    title: The Lord of the Rings`)

	tests := map[string]visitorScenario[ConditionalHandler]{
		"handles documents": {
			input:              "document: 1",
			requireVerifyCount: 1,
			handler: mustCreate(t, OnVisitDocumentNode(func(ctx context.Context, value *yaml.Node) error {
				verify(ctx, func(t *testing.T) {
					assert.Equal(t, yaml.DocumentNode, value.Kind)
				})

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
			input:              commonDoc,
			requireVerifyCount: 2,
			handler: mustCreate(t, OnVisitMappingNode("$.store.book[?(@.title=~/^S.*$/)]", func(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
				verify(ctx, func(t *testing.T) {
					assert.Nil(t, key, "Expected key for a mapping node within a sequence to be nil.")
				})
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
			input:              commonDoc,
			requireVerifyCount: 2,
			handler: mustCreate(t, OnVisitScalarNode("$.store.book[?(@.title=~/^S.*$/)].title", func(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
				verify(ctx, func(t *testing.T) {
					assert.Equal(t, "title", key.Value, "Shouldn't be processing any other scalars besides title")
				})
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
		"handles aliases": {
			input: trimmed(`---
					|store:
					|  book: &books
					|    - author: Ernest Hemingway
					|      title: The Old Man and the Sea
					|    - author: Fyodor Mikhailovich Dostoevsky
					|      title: Crime and Punishment
					|    - author: Jane Austen
					|      title: Sense and Sensibility
					|    - author: Kurt Vonnegut Jr.
					|      title: Slaughterhouse-Five
					|    - author: J. R. R. Tolkien
					|      title: The Lord of the Rings
					|  audiobooks:
					|    - *books
					|    - author: Stephen "Steve-O" Glover
					|      title: 'Professional Idiot: A Memoir'`),
			handler: mustCreate(t, OnVisitAliasNode("$.store.audiobooks.*", func(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
				verify(ctx, func(t *testing.T) {
					assert.Equal(t, yaml.AliasNode, value.Kind)
					assert.Equal(t, "books", value.Value)
					assert.Equal(t, 5, len(value.Alias.Content))
				})
				return nil
			})),
			requireVerifyCount: 1,
		},
		"handles multiple docs of same structure": func() visitorScenario[ConditionalHandler] {
			processed := make([]string, 0)
			return visitorScenario[ConditionalHandler]{
				input: trimmed(`---
					|store:
					|  book:
					|    - author: Ernest Hemingway
					|      title: The Old Man and the Sea
					|    - author: Fyodor Mikhailovich Dostoevsky
					|      title: Crime and Punishment
					|    - author: Jane Austen
					|      title: Sense and Sensibility
					|---
					|store:
					|  book:
					|    - author: Kurt Vonnegut Jr.
					|      title: Slaughterhouse-Five
					|    - author: J. R. R. Tolkien
					|      title: The Lord of the Rings`),
				handler: mustCreate(t, OnVisitScalarNode("$.store.book[*].author", func(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
					processed = append(processed, value.Value)
					return nil
				})),
				validator: func(t *testing.T, h ConditionalHandler) error {
					assert.EqualValues(t, []string{
						"Ernest Hemingway",
						"Fyodor Mikhailovich Dostoevsky",
						"Jane Austen",
						"Kurt Vonnegut Jr.",
						"J. R. R. Tolkien",
					}, processed)
					return nil
				},
			}
		}(),

		"handles multiple docs of different structures": func() visitorScenario[ConditionalHandler] {
			processed := make([]string, 0)
			return visitorScenario[ConditionalHandler]{
				input: trimmed(`---
					|store:
					|  book:
					|    - author: Ernest Hemingway
					|      title: The Old Man and the Sea
					|    - author: Fyodor Mikhailovich Dostoevsky
					|      title: Crime and Punishment
					|    - author: Jane Austen
					|      title: Sense and Sensibility
					|---
					|library:
					|  audiobook:
					|    - author: Kurt Vonnegut Jr.
					|      title: Slaughterhouse-Five
					|    - author: J. R. R. Tolkien
					|      title: The Lord of the Rings`),
				handler: mustCreate(t, OnVisitScalarNode("$..author", func(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
					processed = append(processed, value.Value)
					return nil
				})),
				validator: func(t *testing.T, h ConditionalHandler) error {
					assert.EqualValues(t, []string{
						"Ernest Hemingway",
						"Fyodor Mikhailovich Dostoevsky",
						"Jane Austen",
						"Kurt Vonnegut Jr.",
						"J. R. R. Tolkien",
					}, processed)
					return nil
				},
			}
		}(),
		"handles multiple handlers of same node type": {
			input:              commonDoc,
			requireVerifyCount: 3, /* 2 of S*, 1 of C* */
			handler: mustCreate(t,
				OnVisitScalarNode("$.store.book[?(@.title=~/^S.*$/)].title", func(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
					verify(ctx, func(t *testing.T) {
						assert.Equal(t, "title", key.Value, "Shouldn't be processing any other scalars besides title")
					})
					key.HeadComment = "testing: handles scalars"
					return nil
				}),
				OnVisitScalarNode("$.store.book[?(@.title=~/^C.*$/)].title", func(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
					verify(ctx, func(t *testing.T) {
						assert.Equal(t, "title", key.Value, "Shouldn't be processing any other scalars besides title")
					})
					key.HeadComment = "testing: handles scalars"
					return nil
				}),
			),
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
							// our handler is expected invoke multiple handlers, one matching on S* and one matching on C*
							if key.Value == "title" && (strings.HasPrefix(val.Value, "S") || strings.HasPrefix(val.Value, "C")) {
								assert.Equal(t, "testing: handles scalars", key.HeadComment)
								found += 1
							}
						}
					}
				}
				assert.Equal(t, 3, found, "Expected to match S* and C* titles")
				return nil
			},
		},
		"handles errors": {
			wantErr: true,
			input:   commonDoc,
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
			validateScenario(t, context.TODO(), tt)
		})
	}
}
