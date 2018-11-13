[![Build Status](https://travis-ci.com/jrmarkle/protoc-gen-go-template.svg?branch=master)](https://travis-ci.com/jrmarkle/protoc-gen-go-template) [![Go Report Card](https://goreportcard.com/badge/github.com/jrmarkle/protoc-gen-go-template)](https://goreportcard.com/report/github.com/jrmarkle/protoc-gen-go-template) [![Coverage Status](https://coveralls.io/repos/github/jrmarkle/protoc-gen-go-template/badge.svg?branch=master)](https://coveralls.io/github/jrmarkle/protoc-gen-go-template?branch=master)

# protoc-gen-go-template
Protocol Buffer generator using go templates

# Install

Install `protoc` and `go get github.com/jrmarkle/protoc-gen-go-template`. `$(go env GOPATH)/bin` must be in your `PATH`.

# Example code generation

Suppose you have the file `example.proto` from which you want to generate custom code:
```protobuf
syntax = "proto3";
package example;
option go_package = "example";

enum MyEnum {
	foo = 0;
	bar = 1;
	baz = 2;
}
```

Using this template file `enums.tmpl`:
```
{{ if .EnumType }}
package {{ .Package }}

import "github.com/golang/protobuf/proto"

{{ range .EnumType }}

func (m {{ .Name }}) Enum() *{{ .Name }} {
	p := new({{ .Name }})
	*p = m
	return p
}

func (m *{{ .Name }}) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum({{ .Name }}_value, data, "{{ .Name }}")
	if err != nil {
		return err
	}
	*m = {{ .Name }}(value)
	return nil
}
{{ end }}
{{ end }}

```

Run `protoc --go-template_out=enums.tmpl,format:. example.proto` to generate `example.pb.enums.go`:

```golang
package example

import "github.com/golang/protobuf/proto"

func (m MyEnum) Enum() *MyEnum {
	p := new(MyEnum)
	*p = m
	return p
}

func (m *MyEnum) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(MyEnum_value, data, "MyEnum")
	if err != nil {
		return err
	}
	*m = MyEnum(value)
	return nil
}
```

You can use this example as a workaround for [protobuf issue #256](https://github.com/golang/protobuf/issues/256).
