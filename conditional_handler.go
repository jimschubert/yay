package yay

import (
	"context"
	"errors"

	"gopkg.in/yaml.v3"
)

type FnVisitValueNode func(ctx context.Context, value *yaml.Node) error
type FnVisitKeyValueNode func(ctx context.Context, key *yaml.Node, value *yaml.Node) error
type FnConditional func(path string, fn FnVisitKeyValueNode) FnVisitKeyValueNode

func precondition(path string, fn FnVisitKeyValueNode) FnVisitKeyValueNode {
	var pm *PathMatcher
	return func(parent context.Context, key *yaml.Node, value *yaml.Node) error {
		if pm == nil {
			var err error
			pm, err = PathMatcherFor(parent, path)
			if err != nil {
				return err
			}
		} else if root, ok := rootNode(parent); ok && pm.root != root {
			pm.root = root
			pm.matches = nil
			if err := pm.ensureMatchLookup(); err != nil {
				return err
			}
		}

		if pm.MustMatch(value) {
			// We will only invoke this function if it's applicable to the current node.
			// Passing the path matcher along on context allows the user to obtain the path matcher and match
			// against any nested children if needed
			return fn(WithPathMatcher(parent, pm), key, value)
		}

		return nil
	}
}

type conditionalHandlerOpt func(handler *ConditionalHandler)

//goland:noinspection GoExportedFuncWithUnexportedType
func OnVisitDocumentNode(fn FnVisitValueNode) conditionalHandlerOpt {
	return func(handler *ConditionalHandler) {
		handler.fnVisitDocumentNode = append(handler.fnVisitDocumentNode, fn)
	}
}

//goland:noinspection GoExportedFuncWithUnexportedType
func OnVisitSequenceNode(path string, fn FnVisitKeyValueNode) conditionalHandlerOpt {
	return func(handler *ConditionalHandler) {
		handler.fnVisitSequenceNode = append(handler.fnVisitSequenceNode, precondition(path, fn))
	}
}

//goland:noinspection GoExportedFuncWithUnexportedType
func OnVisitMappingNode(path string, fn FnVisitKeyValueNode) conditionalHandlerOpt {
	return func(handler *ConditionalHandler) {
		handler.fnVisitMappingNode = append(handler.fnVisitMappingNode, precondition(path, fn))
	}
}

//goland:noinspection GoExportedFuncWithUnexportedType
func OnVisitScalarNode(path string, fn FnVisitKeyValueNode) conditionalHandlerOpt {
	return func(handler *ConditionalHandler) {
		handler.fnVisitScalarNode = append(handler.fnVisitScalarNode, precondition(path, fn))
	}
}

//goland:noinspection GoExportedFuncWithUnexportedType
func OnVisitAliasNode(path string, fn FnVisitKeyValueNode) conditionalHandlerOpt {
	return func(handler *ConditionalHandler) {
		handler.fnVisitAliasNode = append(handler.fnVisitAliasNode, precondition(path, fn))
	}
}

// ConditionalHandler allows the user to create handler functions which are conditional on a [yamlpath] selector syntax.
//
// [yamlpath]: https://github.com/vmware-labs/yaml-jsonpath#syntax
type ConditionalHandler struct {
	fnVisitDocumentNode []FnVisitValueNode
	fnVisitSequenceNode []FnVisitKeyValueNode
	fnVisitMappingNode  []FnVisitKeyValueNode
	fnVisitScalarNode   []FnVisitKeyValueNode
	fnVisitAliasNode    []FnVisitKeyValueNode
}

// VisitDocumentNode satisfies VisitsDocumentNode such that a visitor always invokes this method, which defers to the handler passed by the user
func (c *ConditionalHandler) VisitDocumentNode(ctx context.Context, key *yaml.Node) error {
	var err error
	for _, fn := range c.fnVisitDocumentNode {
		if err != nil {
			break
		}
		err = fn(ctx, key)
	}
	return err
}

// VisitSequenceNode satisfies VisitsSequenceNode such that a visitor always invokes this method, which defers to the handler passed by the user
func (c *ConditionalHandler) VisitSequenceNode(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
	var err error
	for _, fn := range c.fnVisitSequenceNode {
		if err != nil {
			break
		}
		err = fn(ctx, key, value)
	}
	return err
}

// VisitMappingNode satisfies VisitsMappingNode such that a visitor always invokes this method, which defers to the handler passed by the user
func (c *ConditionalHandler) VisitMappingNode(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
	var err error
	for _, fn := range c.fnVisitMappingNode {
		if err != nil {
			break
		}
		err = fn(ctx, key, value)
	}
	return err
}

// VisitScalarNode satisfies VisitsScalarNode such that a visitor always invokes this method, which defers to the handler passed by the user
func (c *ConditionalHandler) VisitScalarNode(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
	var err error
	for _, fn := range c.fnVisitScalarNode {
		if err != nil {
			break
		}
		err = fn(ctx, key, value)
	}
	return err
}

// VisitAliasNode satisfies VisitsAliasNode such that a visitor always invokes this method, which defers to the handler passed by the user
func (c *ConditionalHandler) VisitAliasNode(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
	var err error
	for _, fn := range c.fnVisitAliasNode {
		if err != nil {
			break
		}
		err = fn(ctx, key, value)
	}
	return err
}

// NewConditionalHandler creates a new ConditionalHandler, allowing the user to provide 1..n handler functions with [yamlpath] preconditions.
//
// [yamlpath]: https://github.com/vmware-labs/yaml-jsonpath#syntax
func NewConditionalHandler(opts ...conditionalHandlerOpt) (*ConditionalHandler, error) {
	if len(opts) == 0 {
		return nil, errors.New("no handlers provided, at least one is expected")
	}
	handler := &ConditionalHandler{
		fnVisitDocumentNode: make([]FnVisitValueNode, 0),
		fnVisitSequenceNode: make([]FnVisitKeyValueNode, 0),
		fnVisitMappingNode:  make([]FnVisitKeyValueNode, 0),
		fnVisitScalarNode:   make([]FnVisitKeyValueNode, 0),
		fnVisitAliasNode:    make([]FnVisitKeyValueNode, 0),
	}

	for _, opt := range opts {
		opt(handler)
	}
	return handler, nil
}
