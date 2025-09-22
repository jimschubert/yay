package yay

import (
	"context"

	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"go.yaml.in/yaml/v3"
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
		if node, ok := rootNode(ctx); ok {
			matcher.root = node
			matcher.matches = nil
			// forces re-evaluation of path with new root for alias/anchor lookups
			matcher.path, _ = yamlpath.NewPathWithRoot(matcher.rawPath, node)
		}
	}

	return matcher, err
}

// WithPathMatcher derives from the parent context a new context containing the provided PathMatcher
func WithPathMatcher(ctx context.Context, matcher *PathMatcher) context.Context {
	if matcher != nil && matcher.root == nil {
		if node, ok := rootNode(ctx); ok {
			matcher.root = node
			matcher.matches = nil
			// forces re-evaluation of path with new root for alias/anchor lookups
			matcher.path, _ = yamlpath.NewPathWithRoot(matcher.rawPath, node)
		}
	}
	return context.WithValue(ctx, pathMatchKey{}, matcher)
}

func withRootNode(ctx context.Context, node *yaml.Node) context.Context {
	if result, ok := ctx.Value(pathMatchKey{}).(*PathMatcher); ok {
		result.root = node
		// forces re-evaluation of path with new root for alias/anchor lookups
		result.path, _ = yamlpath.NewPathWithRoot(result.rawPath, node)
	}

	return context.WithValue(ctx, rootNodeKey{}, node)
}

func rootNode(ctx context.Context) (*yaml.Node, bool) {
	n, ok := ctx.Value(rootNodeKey{}).(*yaml.Node)
	return n, ok
}
