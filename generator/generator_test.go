package generator

import (
	"bytes"
	"testing"
)

func testExpectedOutput(t *testing.T, g Generator, expectedOutput []byte) {
	t.Helper()

	outputBuffer := new(bytes.Buffer)
	g.out = outputBuffer

	err := g.Run()
	if err != nil {
		t.Fatal(err)
	}

	formattedExpectedOutput, err := formatGoSource(expectedOutput)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(formattedExpectedOutput, expectedOutput) {
		t.Error("output does not match expected output")
	}
}

func testExpectFail(t *testing.T, g Generator) {
	t.Helper()

	err := g.Run()
	if err == nil {
		t.Error("expected error")
	}
}

func TestGeneratorRun(t *testing.T) {
	templateData := `
	{{ if .EnumType }}
	package {{ .Package }}

	import "github.com/golang/protobuf/proto"
	{{ range .EnumType }}
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
	`
}
