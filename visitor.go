package yay

import (
	"context"
	"errors"
	"fmt"

	"go.yaml.in/yaml/v3"
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
	options opts
}

func (v *visitor) Visit(parent context.Context, node *yaml.Node) error {
	if node == nil || parent.Err() != nil {
		return nil
	}

	var maybeErr error

	if !v.options.skipDocumentCheck && node.Kind != yaml.DocumentNode {
		return fmt.Errorf("visitor can only be invoked on a document or multi-document YAML")
	}

	ctx, canceler := context.WithCancel(parent)
	defer canceler()

	if node.Kind == yaml.DocumentNode {
		// TODO: We should be able to move the document visit into iterate and simplify this function
		ctx := withRootNode(ctx, node)
		if handle, ok := v.handler.(VisitsDocumentNode); ok {
			if err := handle.VisitDocumentNode(ctx, node); err != nil {
				maybeErr = errors.Join(maybeErr, err)
			}
		}

		if node.Content == nil || len(node.Content) == 0 {
			// ex: if user invokes as v.Visit(ctx, &yaml.Node{ Kind: yaml.DocumentNode })
			return maybeErr
		}

		value := node.Content[0]
		err := v.iterate(ctx, value)
		maybeErr = errors.Join(maybeErr, err)
	} else if node.Content != nil && len(node.Content) == 2 {
		if node.Content[1] != nil {
			// HACK: yaml-jsonpath requires a "root node" having children to match against. 2 children must be within a mapping node
			wrapper := &yaml.Node{
				Kind: yaml.DocumentNode,
			}
			if node.Content[0].Kind != yaml.MappingNode {
				wrapper.Content = []*yaml.Node{{Kind: yaml.MappingNode, Content: node.Content}}
			} else {
				wrapper.Content = append(wrapper.Content, node.Content[0], node.Content[1])
			}

			nestedCtx := withRootNode(ctx, wrapper)
			err := v.visit(nestedCtx, node.Content[0], node.Content[1])
			maybeErr = errors.Join(maybeErr, err)
		} else {
			nestedCtx := withRootNode(ctx, &yaml.Node{Kind: yaml.DocumentNode, Content: node.Content})
			err := v.iterate(nestedCtx, node.Content[0])
			maybeErr = errors.Join(maybeErr, err)
		}
	} else {
		virtualRoot := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{node}}
		nestedCtx := withRootNode(ctx, virtualRoot)
		err := v.iterate(nestedCtx, node)
		maybeErr = errors.Join(maybeErr, err)
	}

	return maybeErr
}

func (v *visitor) visit(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
	if ctx.Err() != nil || key == nil {
		return nil
	}

	var maybeErr error

	var keyNode *yaml.Node
	// emptyNode is a sentinel value for sequences
	if key != emptyNode {
		keyNode = key
	}

	switch value.Kind {
	case yaml.SequenceNode:
		if handle, ok := v.handler.(VisitsSequenceNode); ok {
			if err := handle.VisitSequenceNode(ctx, keyNode, value); err != nil {
				maybeErr = errors.Join(maybeErr, err)
			}
		}

	case yaml.MappingNode:
		if handle, ok := v.handler.(VisitsMappingNode); ok {
			if err := handle.VisitMappingNode(ctx, keyNode, value); err != nil {
				maybeErr = errors.Join(maybeErr, err)
			}
		}

	case yaml.ScalarNode:
		if handle, ok := v.handler.(VisitsScalarNode); ok {
			if err := handle.VisitScalarNode(ctx, keyNode, value); err != nil {
				maybeErr = errors.Join(maybeErr, err)
			}
		}

	case yaml.AliasNode:
		if handle, ok := v.handler.(VisitsAliasNode); ok {
			if err := handle.VisitAliasNode(ctx, keyNode, value); err != nil {
				maybeErr = errors.Join(maybeErr, err)
			}
		}
	default:
		panic("unhandled default case")
	}

	// if there was an error, we won't recurse nodes any further
	if maybeErr == nil && value.Content != nil && len(value.Content) > 0 {
		maybeErr = v.iterate(ctx, value)
	}

	return maybeErr
}

func (v *visitor) iterate(ctx context.Context, value *yaml.Node) error {
	var maybeErr error
	if ctx.Err() == nil {
		switch value.Kind {
		case yaml.SequenceNode:
			for i := 0; i < len(value.Content); i++ {
				val := value.Content[i]
				if err := v.visit(ctx, emptyNode, val); err != nil {
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
				if err := v.visit(ctx, key, val); err != nil {
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
	return NewVisitorWithOptions(NewOptions(), handlers...)
}

// NewVisitorWithOptions allows constructing a visitor with options addressing edge case scenarios.
// It constructs a new Visitor which handles yaml.Node processing defined by handler.
// The handler must satisfy one or more of the visitor interfaces.
// See:
//   - VisitsYaml
//   - VisitsDocumentNode
//   - VisitsSequenceNode
//   - VisitsMappingNode
//   - VisitsScalarNode
//   - VisitsAliasNode
func NewVisitorWithOptions(options FnOptions, handlers ...any) (Visitor, error) {
	for _, handler := range handlers {
		switch i := handler.(type) {
		case VisitsYaml, VisitsDocumentNode, VisitsSequenceNode, VisitsMappingNode, VisitsScalarNode, VisitsAliasNode:
		default:
			return nil, fmt.Errorf("type %T doesn't implement any visitor handlers", i)
		}
	}

	o := opts{}
	if options != nil {
		options(&o)
	}

	if len(handlers) == 1 {
		return &visitor{handler: handlers[0], options: o}, nil
	}

	return &visitor{handler: &compositeHandler{handlers: handlers}, options: o}, nil
}
