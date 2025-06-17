package yay

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v3"
)

func TestMultipleToSingleMerge_VisitMappingNode(t *testing.T) {
	type testable struct {
		Config struct {
			Actual struct {
				A string `yaml:"a,omitempty"`
				B string `yaml:"b,omitempty"`
				C string `yaml:"c,omitempty"`
			} `yaml:"actual"`
		} `yaml:"config"`
	}

	tests := map[string]visitorScenario[multipleToSingleMergeHandler]{
		"merges multiple merge keys into one reversing order to allow for overrides": {
			input: trimmed(`---
					|input:
					|  first: &first-ref
					|    a: A
					|  second: &second-ref
					|    b: B
					|
					|config:
					|  actual:
					|    <<: *first-ref
					|    <<: *second-ref
					|    c: C`),
			handler: NewMultipleToSingleMergeHandler(),
			validatorWithNode: func(t *testing.T, h multipleToSingleMergeHandler, node *yaml.Node) error {
				b, err := yaml.Marshal(node)
				assert.NoError(t, err)

				actual := testable{}
				assert.NoError(t, yaml.Unmarshal(b, &actual))
				assert.Equal(t, "A", actual.Config.Actual.A)
				assert.Equal(t, "B", actual.Config.Actual.B)
				assert.Equal(t, "C", actual.Config.Actual.C)

				mergeNode := node.Content[0].Content[3].Content[1]
				assert.Equal(t, yaml.MappingNode, mergeNode.Kind)
				assert.Equal(t, "!!merge", mergeNode.Content[0].Tag)
				assert.Equal(t, "<<", mergeNode.Content[0].Value)
				assert.Equal(t, yaml.SequenceNode, mergeNode.Content[1].Kind)
				assert.Equal(t, yaml.FlowStyle, mergeNode.Content[1].Style)
				assert.Equal(t, 2, len(mergeNode.Content[1].Content))
				assert.Equal(t, yaml.AliasNode, mergeNode.Content[1].Content[0].Kind)
				assert.Equal(t, "second-ref", mergeNode.Content[1].Content[0].Value)
				assert.Equal(t, yaml.AliasNode, mergeNode.Content[1].Content[1].Kind)
				assert.Equal(t, "first-ref", mergeNode.Content[1].Content[1].Value)

				assert.NotEqual(t, "!!merge", mergeNode.Content[2].Tag)
				return nil
			},
		},
		"handles existing keys without reordering": {
			input: trimmed(`---
					|input:
					|  first: &first-ref
					|    a: A
					|  second: &second-ref
					|    b: B
					|
					|config:
					|  actual:
					|    <<: [*first-ref, *second-ref]
					|    c: C`),
			handler: NewMultipleToSingleMergeHandler(),
			validatorWithNode: func(t *testing.T, h multipleToSingleMergeHandler, node *yaml.Node) error {
				b, err := yaml.Marshal(node)
				assert.NoError(t, err)

				actual := testable{}
				assert.NoError(t, yaml.Unmarshal(b, &actual))
				assert.Equal(t, "A", actual.Config.Actual.A)
				assert.Equal(t, "B", actual.Config.Actual.B)
				assert.Equal(t, "C", actual.Config.Actual.C)

				mergeNode := node.Content[0].Content[3].Content[1]
				assert.Equal(t, yaml.MappingNode, mergeNode.Kind)
				assert.Equal(t, "!!merge", mergeNode.Content[0].Tag)
				assert.Equal(t, "<<", mergeNode.Content[0].Value)
				assert.Equal(t, yaml.SequenceNode, mergeNode.Content[1].Kind)
				assert.Equal(t, yaml.FlowStyle, mergeNode.Content[1].Style)
				assert.Equal(t, 2, len(mergeNode.Content[1].Content))
				assert.Equal(t, yaml.AliasNode, mergeNode.Content[1].Content[0].Kind)
				assert.Equal(t, "first-ref", mergeNode.Content[1].Content[0].Value)
				assert.Equal(t, yaml.AliasNode, mergeNode.Content[1].Content[1].Kind)
				assert.Equal(t, "second-ref", mergeNode.Content[1].Content[1].Value)

				assert.NotEqual(t, "!!merge", mergeNode.Content[2].Tag)
				return nil
			},
		},

		"handles single merge aliase node": {
			input: trimmed(`---
					|input:
					|  first: &first-ref
					|    a: A
					|  second: &second-ref
					|    b: B
					|
					|config:
					|  actual:
					|    <<: *first-ref
					|    c: C`),
			handler: NewMultipleToSingleMergeHandler(),
			validatorWithNode: func(t *testing.T, h multipleToSingleMergeHandler, node *yaml.Node) error {
				b, err := yaml.Marshal(node)
				assert.NoError(t, err)

				actual := testable{}
				assert.NoError(t, yaml.Unmarshal(b, &actual))
				assert.Equal(t, "A", actual.Config.Actual.A)
				assert.Empty(t, actual.Config.Actual.B, "B should not have been merged in this scenario")
				assert.Equal(t, "C", actual.Config.Actual.C)

				mergeNode := node.Content[0].Content[3].Content[1]
				assert.Equal(t, yaml.MappingNode, mergeNode.Kind)
				assert.Equal(t, "!!merge", mergeNode.Content[0].Tag)
				assert.Equal(t, "<<", mergeNode.Content[0].Value)
				assert.Equal(t, yaml.AliasNode, mergeNode.Content[1].Kind)
				assert.Nil(t, mergeNode.Content[1].Content)
				assert.Equal(t, "first-ref", mergeNode.Content[1].Value)

				assert.NotEqual(t, "!!merge", mergeNode.Content[2].Tag)
				return nil
			},
		},

		"handles single merge sequence node": {
			input: trimmed(`---
					|input:
					|  first: &first-ref
					|    a: A
					|  second: &second-ref
					|    b: B
					|
					|config:
					|  actual:
					|    <<: [*first-ref]
					|    c: C`),
			handler: NewMultipleToSingleMergeHandler(),
			validatorWithNode: func(t *testing.T, h multipleToSingleMergeHandler, node *yaml.Node) error {
				b, err := yaml.Marshal(node)
				assert.NoError(t, err)

				actual := testable{}
				assert.NoError(t, yaml.Unmarshal(b, &actual))
				assert.Equal(t, "A", actual.Config.Actual.A)
				assert.Empty(t, actual.Config.Actual.B, "B should not have been merged in this scenario")
				assert.Equal(t, "C", actual.Config.Actual.C)

				mergeNode := node.Content[0].Content[3].Content[1]
				assert.Equal(t, yaml.MappingNode, mergeNode.Kind)
				assert.Equal(t, "!!merge", mergeNode.Content[0].Tag)
				assert.Equal(t, "<<", mergeNode.Content[0].Value)
				assert.Equal(t, yaml.SequenceNode, mergeNode.Content[1].Kind)
				assert.Equal(t, yaml.FlowStyle, mergeNode.Content[1].Style)
				assert.Equal(t, 1, len(mergeNode.Content[1].Content))
				assert.Equal(t, yaml.AliasNode, mergeNode.Content[1].Content[0].Kind)
				assert.Equal(t, "first-ref", mergeNode.Content[1].Content[0].Value)

				assert.NotEqual(t, "!!merge", mergeNode.Content[2].Tag)
				return nil
			},
		},

		"handles multiple merge keys with overrides": {
			input: trimmed(`---
					|input:
					|  first: &first-ref
					|    a: A
					|    msg: goodbye, world
					|  second: &second-ref
					|    b: B
					|    msg: hello, world
					|
					|config:
					|  actual:
					|    <<: *first-ref
					|    <<: *second-ref
					|    c: C`),
			handler: NewMultipleToSingleMergeHandler(),
			validatorWithNode: func(t *testing.T, h multipleToSingleMergeHandler, node *yaml.Node) error {
				// use yaml v3's built-in merging behavior to validate that 'msg' is as expected after value merges
				type evaluate struct {
					Config struct {
						Actual struct {
							A   string `json:"a"`
							B   string `json:"b"`
							C   string `json:"c"`
							Msg string `json:"msg"`
						} `json:"actual"`
					} `json:"config"`
				}

				e := evaluate{}
				assert.NoError(t, node.Decode(&e))
				assert.Equal(t, "A", e.Config.Actual.A)
				assert.Equal(t, "B", e.Config.Actual.B)
				assert.Equal(t, "C", e.Config.Actual.C)
				assert.Equal(t, "hello, world", e.Config.Actual.Msg)

				mergeNode := node.Content[0].Content[3].Content[1]
				assert.Equal(t, yaml.MappingNode, mergeNode.Kind)
				assert.NotEqual(t, "!!merge", mergeNode.Content[1].Tag)
				assert.Equal(t, "!!merge", mergeNode.Content[0].Tag)
				assert.Equal(t, "<<", mergeNode.Content[0].Value)
				assert.Equal(t, yaml.SequenceNode, mergeNode.Content[1].Kind)
				assert.Equal(t, "second-ref", mergeNode.Content[1].Content[0].Value)
				assert.Equal(t, yaml.AliasNode, mergeNode.Content[1].Content[0].Kind)
				assert.Equal(t, "first-ref", mergeNode.Content[1].Content[1].Value)
				assert.Equal(t, yaml.AliasNode, mergeNode.Content[1].Content[1].Kind)

				return nil
			},
		},

		"handles multiple merge keys with retaining key order option": {
			input: trimmed(`---
					|input:
					|  first: &first-ref
					|    a: A
					|    msg: goodbye, world
					|  second: &second-ref
					|    b: B
					|    msg: hello, world
					|
					|config:
					|  actual:
					|    <<: *first-ref
					|    <<: *second-ref
					|    c: C`),
			handler: NewMultipleToSingleMergeHandler(WithRetainMergeKeyOrder()),
			validatorWithNode: func(t *testing.T, h multipleToSingleMergeHandler, node *yaml.Node) error {
				// use yaml v3's built-in merging behavior to validate that 'msg' is as expected after value merges
				type evaluate struct {
					Config struct {
						Actual struct {
							A   string `json:"a"`
							B   string `json:"b"`
							C   string `json:"c"`
							Msg string `json:"msg"`
						} `json:"actual"`
					} `json:"config"`
				}

				e := evaluate{}
				assert.NoError(t, node.Decode(&e))
				assert.Equal(t, "A", e.Config.Actual.A)
				assert.Equal(t, "B", e.Config.Actual.B)
				assert.Equal(t, "C", e.Config.Actual.C)
				assert.Equal(t, "goodbye, world", e.Config.Actual.Msg)

				mergeNode := node.Content[0].Content[3].Content[1]
				assert.Equal(t, yaml.MappingNode, mergeNode.Kind)
				assert.NotEqual(t, "!!merge", mergeNode.Content[1].Tag)
				assert.Equal(t, "!!merge", mergeNode.Content[0].Tag)
				assert.Equal(t, "<<", mergeNode.Content[0].Value)
				assert.Equal(t, yaml.SequenceNode, mergeNode.Content[1].Kind)
				assert.Equal(t, "first-ref", mergeNode.Content[1].Content[0].Value)
				assert.Equal(t, yaml.AliasNode, mergeNode.Content[1].Content[0].Kind)
				assert.Equal(t, "second-ref", mergeNode.Content[1].Content[1].Value)
				assert.Equal(t, yaml.AliasNode, mergeNode.Content[1].Content[1].Kind)

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
