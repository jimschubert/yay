package yay

import (
	"context"

	"gopkg.in/yaml.v3"
)

var (
	_ VisitsMappingNode = (*multipleToSingleMergeHandler)(nil)
)

// multipleToSingleMergeHandler handles the transformation of multiple merge keys into a single merge key.
// See NewMultipleToSingleMergeHandler for more information.
type multipleToSingleMergeHandler struct {
	retainMergeKeyOrder bool
}

// VisitMappingNode processes a YAML mapping node to consolidate multiple merge keys into a single merge key.
func (m multipleToSingleMergeHandler) VisitMappingNode(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
	var mergeValue *yaml.Node
	contents := make([]*yaml.Node, 0)
	initialMergeKind := yaml.AliasNode

	for i := 0; i < len(value.Content); i += 2 {
		k, v := value.Content[i], value.Content[i+1]

		// if key.Tag == !!merge && value.Kind == yaml.AliasNode, value.value will be the target anchor name
		// if value.Value is empty, and value.Kind == yaml.SequenceNode, this is the correct/expected syntax, but we will
		// still need to traverse and make sure there are no stragglers
		if k.Tag == "!!merge" {
			if mergeValue == nil {
				mergeValue = &yaml.Node{
					Tag:     "!!seq",
					Style:   yaml.FlowStyle,
					Kind:    yaml.SequenceNode,
					Content: make([]*yaml.Node, 0),
				}

				// keep only the first '<<' key and it's value, regardless of AliasNode versus SequenceNode
				contents = append(contents, k, mergeValue)
				initialMergeKind = v.Kind
			}

			// we're in the context of an alias. Now, check if it's a scalar or sequence
			//goland:noinspection GoSwitchMissingCasesForIotaConsts
			switch v.Kind {
			case yaml.AliasNode:
				mergeValue.Content = append(mergeValue.Content, v)
			case yaml.SequenceNode:
				mergeValue.Content = append(mergeValue.Content, v.Content...)
			}
			continue
		}

		contents = append(contents, k, v)
	}

	if mergeValue != nil {
		if len(mergeValue.Content) == 1 && initialMergeKind == yaml.AliasNode {
			// drop to single-value syntax back to the original syntax
			for i, content := range contents {
				if content.Tag == "!!merge" {
					contents[i+1] = mergeValue.Content[0]
					break
				}
			}
		} else if initialMergeKind == yaml.AliasNode && !m.retainMergeKeyOrder {
			// reverse contents of mergeValue.Content, allowing overrides of original contents
			for i, j := 0, len(mergeValue.Content)-1; i < j; i, j = i+1, j-1 {
				mergeValue.Content[i], mergeValue.Content[j] = mergeValue.Content[j], mergeValue.Content[i]
			}
		}
	}

	value.Content = contents
	return nil
}

// MultipleMergeKeyOpt is an option for NewMultipleToSingleMergeHandler.
type MultipleMergeKeyOpt func(handler *multipleToSingleMergeHandler)

// WithRetainMergeKeyOrder is an option for NewMultipleToSingleMergeHandler which changes the behavior of merging keys
// to retain the original order.
//
// By default, NewMultipleToSingleMergeHandler will reverse the order of the merge keys, allowing for keys to be
// overridden according to the specification for merge keys as defined in the [YAML specification]. Since multiple merge
// keys are not defined by the specification, some parsers and libraries may not follow the same behaviors.
//
// This option will change the behavior to retain the original order of the multiple merge keys.
//
// [YAML specification]: https://yaml.org/type/merge.html
func WithRetainMergeKeyOrder() MultipleMergeKeyOpt {
	return func(handler *multipleToSingleMergeHandler) {
		handler.retainMergeKeyOrder = true
	}
}

// NewMultipleToSingleMergeHandler aims to address a common situation in which the [YAML specification] does not allow multiple merge keys
// under a given node, while many parsers and libraries allow it. This is defined in yaml.v3 [issue 624].
// When consuming such a document with yaml.v3 into a tag-based struct, parsing will fail.
// Passing a node through this handler will allow this scenario to succeed.
//
// [YAML specification]: https://yaml.org/type/merge.html
// [issue 624]: https://github.com/go-yaml/yaml/issues/624
//
//goland:noinspection GoExportedFuncWithUnexportedType
func NewMultipleToSingleMergeHandler(opts ...MultipleMergeKeyOpt) *multipleToSingleMergeHandler {
	handler := &multipleToSingleMergeHandler{}
	for _, opt := range opts {
		opt(handler)
	}
	return handler
}
