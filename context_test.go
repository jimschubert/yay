package yay

import (
	"context"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestWithPathMatcher_NilMatcherReturnsNewMatcher(t *testing.T) {
	ctx := context.Background()
	ctx2 := WithPathMatcher(ctx, nil)
	if ctx2.Value(pathMatchKey{}) == nil {
		t.Error("expected non-nil matcher in context")
	}
}

func TestWithPathMatcher_MatcherWithRoot(t *testing.T) {
	ctx := context.Background()
	matcher := &PathMatcher{root: &yaml.Node{}}
	ctx2 := WithPathMatcher(ctx, matcher)
	stored := ctx2.Value(pathMatchKey{}).(*PathMatcher)
	if stored.root == nil {
		t.Error("expected root to be set on matcher")
	}
}

func TestWithPathMatcher_MatcherWithoutRoot(t *testing.T) {
	ctx := context.Background()
	node := &yaml.Node{Kind: yaml.DocumentNode}
	ctx = withRootNode(ctx, node)
	matcher := &PathMatcher{}
	ctx2 := WithPathMatcher(ctx, matcher)
	stored := ctx2.Value(pathMatchKey{}).(*PathMatcher)
	if stored.root != node {
		t.Error("expected root node to be set from context")
	}
}

func TestPathMatcherFor_NoRootNode(t *testing.T) {
	ctx := context.Background()
	matcher, err := PathMatcherFor(ctx, "foo")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if matcher == nil {
		t.Error("expected matcher to be created")
	}
}

func TestPathMatcherFor_WithRootNode(t *testing.T) {
	ctx := context.Background()
	node := &yaml.Node{Kind: yaml.DocumentNode}
	ctx = withRootNode(ctx, node)
	matcher, err := PathMatcherFor(ctx, "foo")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if matcher.root != node {
		t.Error("expected matcher.root to be set from context root node")
	}
}

func TestRootNode(t *testing.T) {
	ctx := context.Background()
	if n, ok := rootNode(ctx); ok || n != nil {
		t.Error("expected no root node in context")
	}
	node := &yaml.Node{Kind: yaml.DocumentNode}
	ctx = withRootNode(ctx, node)
	n, ok := rootNode(ctx)
	if !ok || n != node {
		t.Error("expected root node to be found in context")
	}
}
