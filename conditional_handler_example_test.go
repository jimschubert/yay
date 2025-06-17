package yay_test

import (
	"context"
	"fmt"

	"github.com/jimschubert/yay"
	"go.yaml.in/yaml/v3"
)

func ExampleNewConditionalHandler_scalars_selectors() {
	input := `---
store:
  book:
  - author: Ernest Hemingway
    title: The Old Man and the Sea
  - author: Fyodor Mikhailovich Dostoevsky
    title: Crime and Punishment
  - author: Jane Austen
    title: Sense and Sensibility
  - author: Kurt Vonnegut Jr.
    title: Slaughterhouse-Five
  - author: J. R. R. Tolkien
    title: The Lord of the Rings`

	document := &yaml.Node{}
	_ = yaml.Unmarshal([]byte(input), document)

	handler, _ := yay.NewConditionalHandler(
		yay.OnVisitScalarNode("$.store.book[?(@.title=~/^S.*$/)].title",
			func(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
				fmt.Printf("processed item: key=%s, value=%q\n", key.Value, value.Value)
				return nil
			}))

	visitor, _ := yay.NewVisitor(handler)
	_ = visitor.Visit(context.TODO(), document)
	// Output:
	// processed item: key=title, value="Sense and Sensibility"
	// processed item: key=title, value="Slaughterhouse-Five"
}

func ExampleNewConditionalHandler_mapping_selectors() {

	input := `---
store:
  book:
  - author: Ernest Hemingway
    title: The Old Man and the Sea
  - author: Fyodor Mikhailovich Dostoevsky
    title: Crime and Punishment
  - author: Jane Austen
    title: Sense and Sensibility
  - author: Kurt Vonnegut Jr.
    title: Slaughterhouse-Five
  - author: J. R. R. Tolkien
    title: The Lord of the Rings`

	document := &yaml.Node{}
	_ = yaml.Unmarshal([]byte(input), document)

	count := 0
	handler, _ := yay.NewConditionalHandler(
		yay.OnVisitMappingNode("$.store.book[?(@.title=~/^S.*$/)]",
			func(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
				// note: key is nil here because each map value is an item in a sequence
				fmt.Printf("processed item at index %d\n", count)
				count += 1
				return nil
			}))

	visitor, _ := yay.NewVisitor(handler)
	_ = visitor.Visit(context.TODO(), document)
	// Output:
	// processed item at index 0
	// processed item at index 1
}
