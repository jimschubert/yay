package yay

import (
	"context"

	"gopkg.in/yaml.v3"
)

var (
	_ VisitsMappingNode = (*multipleToSingleMergeHandler)(nil)
)

type multipleToSingleMergeHandler struct {
}

func (m multipleToSingleMergeHandler) VisitMappingNode(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
	var mergeValue *yaml.Node
	contents := make([]*yaml.Node, 0)
	initialMergeKind := yaml.AliasNode

	for i := 0; i < len(value.Content); i += 2 {
		k := value.Content[i]
		v := value.Content[i+1]
		// anchor definition is v.Anchor

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

	if mergeValue != nil && len(mergeValue.Content) == 1 && initialMergeKind == yaml.AliasNode {
		// drop to single-value syntax back to the original syntax
		for i, content := range contents {
			if content.Tag == "!!merge" {
				contents[i+1] = mergeValue.Content[0]
				break
			}
		}
	}

	value.Content = contents
	return nil
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
func NewMultipleToSingleMergeHandler() *multipleToSingleMergeHandler {
	return &multipleToSingleMergeHandler{}
}
