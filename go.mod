module github.com/jimschubert/yay

go 1.23.0

toolchain go1.24.3

require (
	github.com/stretchr/testify v1.10.0
	github.com/vmware-labs/yaml-jsonpath v0.3.2
	go.yaml.in/yaml/v3 v3.0.3
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/vmware-labs/yaml-jsonpath => github.com/jimschubert/yaml-jsonpath v0.4.0
