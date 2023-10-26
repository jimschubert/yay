package yay

import (
	"context"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

var (
	emptyNode = &yaml.Node{}
)

// VisitsDocumentNode defines behaviors for visitors which want to handle document nodes
type VisitsDocumentNode interface {
	VisitDocumentNode(ctx context.Context, key *yaml.Node) error
}

// VisitsSequenceNode defines behaviors for visitors which want to handle sequence nodes
type VisitsSequenceNode interface {
	VisitSequenceNode(ctx context.Context, key *yaml.Node, value *yaml.Node) error
}

// VisitsMappingNode defines behaviors for visitors which want to handle mapping nodes
type VisitsMappingNode interface {
	VisitMappingNode(ctx context.Context, key *yaml.Node, value *yaml.Node) error
}

// VisitsScalarNode defines behaviors for visitors which want to handle scalar nodes
type VisitsScalarNode interface {
	VisitScalarNode(ctx context.Context, key *yaml.Node, value *yaml.Node) error
}

// VisitsAliasNode defines behaviors for visitors which want to handle alias nodes
type VisitsAliasNode interface {
	VisitAliasNode(ctx context.Context, key *yaml.Node, value *yaml.Node) error
}

// VisitsYaml defines all behaviors for YAML visitors
type VisitsYaml interface {
	VisitsDocumentNode
	VisitsSequenceNode
	VisitsMappingNode
	VisitsScalarNode
	VisitsAliasNode
}

// Visitor defines behaviors related to recursively visiting a yaml.Node
type Visitor interface {
	Visit(ctx context.Context, node *yaml.Node) error
}

type visitor struct {
	handler any
}

func (v *visitor) visit(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
	if ctx.Err() != nil || key == nil {
		return nil
	}

	var maybeErr error
	var err error

	var keyNode *yaml.Node
	// emptyNode is a sentinel value for sequences
	if key != emptyNode {
		keyNode = key
	}

	switch value.Kind {
	case yaml.SequenceNode:
		if handle, ok := v.handler.(VisitsSequenceNode); ok {
			if err = handle.VisitSequenceNode(ctx, keyNode, value); err != nil {
				maybeErr = errors.Join(maybeErr, err)
			}
		}

	case yaml.MappingNode:
		if handle, ok := v.handler.(VisitsMappingNode); ok {
			if err = handle.VisitMappingNode(ctx, keyNode, value); err != nil {
				maybeErr = errors.Join(maybeErr, err)
			}
		}

	case yaml.ScalarNode:
		if handle, ok := v.handler.(VisitsScalarNode); ok {
			if err = handle.VisitScalarNode(ctx, keyNode, value); err != nil {
				maybeErr = errors.Join(maybeErr, err)
			}
		}

	case yaml.AliasNode:
		if handle, ok := v.handler.(VisitsAliasNode); ok {
			if err = handle.VisitAliasNode(ctx, keyNode, value); err != nil {
				maybeErr = errors.Join(maybeErr, err)
			}
		}
	}

	// if there was an error, we won't recurse nodes any further
	if maybeErr == nil && value.Content != nil && len(value.Content) > 0 {
		maybeErr = v.iterate(ctx, value)
	}

	return maybeErr
}

func (v *visitor) iterate(ctx context.Context, value *yaml.Node) error {
	var err error
	var maybeErr error
	if ctx.Err() == nil {
		switch value.Kind {
		case yaml.SequenceNode:
			for i := 0; i < len(value.Content); i += 1 {
				var key *yaml.Node
				val := value.Content[i]
				key = emptyNode

				if err = v.visit(ctx, key, val); err != nil {
					maybeErr = errors.Join(maybeErr, err)
				}
				if ctx.Err() != nil {
					break
				}
			}
		case yaml.MappingNode:
			for i := 0; i < len(value.Content); i += 2 {
				key := value.Content[i]
				val := value.Content[i+1]
				if err = v.visit(ctx, key, val); err != nil {
					maybeErr = errors.Join(maybeErr, err)
				}
				if ctx.Err() != nil {
					break
				}
			}
		}
	}
	return maybeErr
}

func (v *visitor) Visit(parent context.Context, node *yaml.Node) error {
	if node == nil || parent.Err() != nil {
		return nil
	}

	var maybeErr error
	var err error

	if node.Kind != yaml.DocumentNode {
		return fmt.Errorf("visitor can only be invoked on a document or multi-document YAML")
	}

	ctx := withRootNode(parent, node)

	if handle, ok := v.handler.(VisitsDocumentNode); ok {
		if err = handle.VisitDocumentNode(ctx, node); err != nil {
			maybeErr = errors.Join(maybeErr, err)
		}

	}
	value := node.Content[0]
	err = v.iterate(ctx, value)
	maybeErr = errors.Join(maybeErr, err)

	return maybeErr
}

// NewVisitor constructs a new Visitor which handles yaml.Node processing defined by handler.
// The handler must satisfy one or more of the visitor interfaces.
// See:
//   - VisitsYaml
//   - VisitsDocumentNode
//   - VisitsSequenceNode
//   - VisitsMappingNode
//   - VisitsScalarNode
//   - VisitsAliasNode
func NewVisitor(handlers ...any) (Visitor, error) {
	for _, handler := range handlers {
		switch i := handler.(type) {
		case VisitsYaml, VisitsDocumentNode, VisitsSequenceNode, VisitsMappingNode, VisitsScalarNode, VisitsAliasNode:
		default:
			return nil, fmt.Errorf("type %T doesn't implement any visitor handlers", i)
		}
	}
	if len(handlers) == 1 {
		return &visitor{handler: handlers[0]}, nil
	}

	return &visitor{handler: &compositeHandler{handlers: handlers}}, nil
}
