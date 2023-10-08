package yay

import (
	"context"

	"gopkg.in/yaml.v3"
)

type pathMatchKey struct{}
type rootNodeKey struct{}

// PathMatcherFor retrieves the PathMatcher for the given path on this context, or creates a new one if it differs.
func PathMatcherFor(ctx context.Context, path string) (*PathMatcher, error) {
	var err error
	var matcher *PathMatcher
	if result, ok := ctx.Value(pathMatchKey{}).(*PathMatcher); ok {
		matcher = result
	}

	if matcher == nil || matcher.rawPath != path {
		matcher, err = newPathMatcher(path)
		if err != nil {
			return nil, err
		}
	}

	if matcher.root == nil {
		if rootNode, ok := ctx.Value(rootNodeKey{}).(*yaml.Node); ok {
			matcher.root = rootNode
			matcher.matches = nil
		}
	}

	return matcher, err
}

// WithPathMatcher derives from the parent context a new context containing the provided PathMatcher
func WithPathMatcher(ctx context.Context, matcher *PathMatcher) context.Context {
	if matcher.root == nil {
		if rootNode, ok := ctx.Value(rootNodeKey{}).(*yaml.Node); ok {
			matcher.root = rootNode
			matcher.matches = nil
		}
	}
	return context.WithValue(ctx, pathMatchKey{}, matcher)
}

func withRootNode(ctx context.Context, node *yaml.Node) context.Context {
	return context.WithValue(ctx, rootNodeKey{}, node)
}
