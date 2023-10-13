package yay

import (
	"context"
	"errors"

	"gopkg.in/yaml.v3"
)

type compositeHandler struct {
	handlers []any
}

// VisitDocumentNode satisfies VisitsDocumentNode such that a visitor always invokes this method, which defers to the handler passed by the user
func (c *compositeHandler) VisitDocumentNode(ctx context.Context, key *yaml.Node) error {
	var err error
	for _, handler := range c.handlers {
		if h, ok := handler.(VisitsDocumentNode); ok {
			err = errors.Join(err, h.VisitDocumentNode(ctx, key))
		}
	}
	return err
}

// VisitSequenceNode satisfies VisitsSequenceNode such that a visitor always invokes this method, which defers to the handler passed by the user
func (c *compositeHandler) VisitSequenceNode(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
	var err error
	for _, handler := range c.handlers {
		if h, ok := handler.(VisitsSequenceNode); ok {
			err = errors.Join(err, h.VisitSequenceNode(ctx, key, value))
		}
	}
	return err
}

// VisitMappingNode satisfies VisitsMappingNode such that a visitor always invokes this method, which defers to the handler passed by the user
func (c *compositeHandler) VisitMappingNode(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
	var err error
	for _, handler := range c.handlers {
		if h, ok := handler.(VisitsMappingNode); ok {
			err = errors.Join(err, h.VisitMappingNode(ctx, key, value))
		}
	}
	return err
}

// VisitScalarNode satisfies VisitsScalarNode such that a visitor always invokes this method, which defers to the handler passed by the user
func (c *compositeHandler) VisitScalarNode(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
	var err error
	for _, handler := range c.handlers {
		if h, ok := handler.(VisitsScalarNode); ok {
			err = errors.Join(err, h.VisitScalarNode(ctx, key, value))
		}
	}
	return err
}

// VisitAliasNode satisfies VisitsAliasNode such that a visitor always invokes this method, which defers to the handler passed by the user
func (c *compositeHandler) VisitAliasNode(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
	var err error
	for _, handler := range c.handlers {
		if h, ok := handler.(VisitsAliasNode); ok {
			err = errors.Join(err, h.VisitAliasNode(ctx, key, value))
		}
	}
	return err
}
