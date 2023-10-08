# yay

Working with YAML in Go can be fun. :yay:

This project provides some utilities to make it slightly more fun by wrapping  [yaml.v3](https://github.com/go-yaml/yaml/tree/v3) and [YAML JSONPath](https://github.com/vmware-labs/yaml-jsonpath/).


![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/jimschubert/yay?color=blue&sort=semver)
![Go Version](https://img.shields.io/github/go-mod/go-version/jimschubert/yay)
[![Apache 2.0 License](https://img.shields.io/badge/License-Apache%202.0-blue)](./LICENSE)  
[![build](https://github.com/jimschubert/yay/actions/workflows/build.yml/badge.svg)](https://github.com/jimschubert/yay/actions/workflows/build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/jimschubert/yay)](https://goreportcard.com/report/github.com/jimschubert/yay)

## Features

* A visitor allowing user defined handlers for standard [yaml.v3](https://github.com/go-yaml/yaml/tree/v3)
* A [ConditionalHandler](./conditional_handler.go) allowing to define YAML JSONPath preconditions to visitor methods

## Examples

### Standard Handler

First, create a handler which satisfies one or more of the interfaces:

* VisitsYaml
* VisitsDocumentNode
* VisitsSequenceNode
* VisitsMappingNode
* VisitsScalarNode
* VisitsAliasNode

```go
type myHandler struct{}
func (m *myHandler) VisitScalarNode(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
	fmt.Printf("found key=%s value=%s\n", key.Value, value.Value)
    return nil
}
```

Then, pass create a new visitor with this handler and run against the target document node. Error handling omitted from the below example:

```go
document := &yaml.Node{}
// parse your document
visitor, _ := yay.NewVisitor(myHandler{})
_ = visitor.Visit(context.TODO(), document)
```

### Conditional Handler

Suppose you have a complex YAML document, and you only want to parse nodes based on some condition. You could use the standard handler and apply those checks on the key/value nodes. 
This is fine, but may lead to unexpected bugs/behaviors. You can also apply a selector as a precondition using a conditional handler.

Consider this YAML document used in [conditional_handler_test.go](./conditional_handler_test.go):

```yaml
---
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
    title: The Lord of the Rings
```

IF you only want to process titles beginning with `S`, you could use the [selector syntax](https://github.com/vmware-labs/yaml-jsonpath#syntax):

```
$.store.book[?(@.title=~/^S.*$/)]
```

This will select each mapping node matching that criteria. Rather than your visitor processing 5 items, you will process 2 items.
For example:

```go
document := &yaml.Node{}
_ = yaml.Unmarshal([]byte(input), document)

count := 0
handler, _ := yay.NewConditionalHandler(
    yay.OnVisitMappingNode("$.store.book[?(@.title=~/^S.*$/)]",
        func(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
            fmt.Printf("processed item at index %d\n", count)
            count += 1
            return nil
        }))

visitor, _ := yay.NewVisitor(handler)
_ = visitor.Visit(context.TODO(), document)
```

This will output:

```go
processed item at index 0
processed item at index 1
```

If you want to process the item for which your conditional applies, you can continue chaining dot-notation. For example, to process just the scalar `title`:

```go
handler, _ := yay.NewConditionalHandler(
    yay.OnVisitScalarNode("$.store.book[?(@.title=~/^S.*$/)].title",
        func(ctx context.Context, key *yaml.Node, value *yaml.Node) error {
            fmt.Printf("processed item at index %d\n", count)
            count += 1
            return nil
        }))
```

Notice the use of the functional `OnVisitScalarNode` and the matcher is now `$.store.book[?(@.title=~/^S.*$/)].title`.


## Caveats

Note that `key` may be nil if the node type you're processing exists within a sequence in the original document. That is, items within sequences don't have keys.

## Install

```
go get -u github.com/jimschubert/yay
```

## Build/Test

```shell
go test -v -race -cover ./...
```

## License

This project is [licensed](./LICENSE) under Apache 2.0.
