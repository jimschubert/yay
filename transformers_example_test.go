package yay_test

import (
	"context"
	"os"

	"github.com/jimschubert/yay"
	"gopkg.in/yaml.v3"
)

func ExampleNewMultipleToSingleMergeHandler() {
	input := `---
input:
  first: &first-ref
    a: A
    msg: goodbye, world
  second: &second-ref
    b: B
    msg: hello, world

config:
  actual:
    <<: *first-ref
    <<: *second-ref
    c: C
`

	document := &yaml.Node{}
	_ = yaml.Unmarshal([]byte(input), document)

	handler := yay.NewMultipleToSingleMergeHandler()
	visitor, _ := yay.NewVisitor(handler)
	_ = visitor.Visit(context.TODO(), document)

	enc := yaml.NewEncoder(os.Stdout)
	enc.SetIndent(1)
	_ = enc.Encode(document)
	// Output:
	// input:
	//   first: &first-ref
	//     a: A
	//     msg: goodbye, world
	//   second: &second-ref
	//     b: B
	//     msg: hello, world
	// config:
	//   actual:
	//     !!merge <<: [*second-ref, *first-ref]
	//     c: C
}

func ExampleWithRetainMergeKeyOrder() {
	input := `---
input:
  first: &first-ref
    a: A
    msg: goodbye, world
  second: &second-ref
    b: B
    msg: hello, world

config:
  actual:
    <<: *first-ref
    <<: *second-ref
    c: C
`

	document := &yaml.Node{}
	_ = yaml.Unmarshal([]byte(input), document)

	handler := yay.NewMultipleToSingleMergeHandler(
		yay.WithRetainMergeKeyOrder(),
	)
	visitor, _ := yay.NewVisitor(handler)
	_ = visitor.Visit(context.TODO(), document)

	enc := yaml.NewEncoder(os.Stdout)
	enc.SetIndent(1)
	_ = enc.Encode(document)
	// Output:
	// input:
	//   first: &first-ref
	//     a: A
	//     msg: goodbye, world
	//   second: &second-ref
	//     b: B
	//     msg: hello, world
	// config:
	//   actual:
	//     !!merge <<: [*first-ref, *second-ref]
	//     c: C
}
