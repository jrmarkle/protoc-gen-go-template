package generator

import (
	"bytes"
	"errors"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
	"io"
	"os"
	"reflect"
	"syscall"
	"testing"
)

const anyError = "any-error"

func expectOutput(t *testing.T, g *Generator, expectedOutput *plugin.CodeGeneratorResponse) {
	t.Helper()

	outputBuffer := new(bytes.Buffer)
	g.out = outputBuffer

	err := g.Run()
	if err != nil {
		t.Fatal("Run error:", err)
	}

	actualOutput := new(plugin.CodeGeneratorResponse)
	err = actualOutput.Unmarshal(outputBuffer.Bytes())
	if err != nil {
		t.Fatal("Error unmarshalling output:", err)
	}

	if expectedErr := expectedOutput.GetError(); len(expectedErr) == 0 {
		if actualErr := actualOutput.GetError(); len(actualErr) > 0 {
			t.Error("Unexpected error: ", actualErr)
		}
	} else if expectedErr == anyError {
		if len(actualOutput.GetError()) == 0 {
			t.Error("Expected an error")
		}
	} else {
		if actualErr := actualOutput.GetError(); actualErr != expectedErr {
			t.Errorf("Expected error %q, got error %q", expectedErr, actualErr)
		}
	}

	if !reflect.DeepEqual(actualOutput.File, expectedOutput.File) {
		t.Errorf("Actual file output does not match expected file output"+
			"\n----- Actual -----\n%+v\n----- Expected -----\n%+v\n", actualOutput, expectedOutput)
	}
}

func expectFail(t *testing.T, g *Generator) {
	t.Helper()

	outputBuffer := new(bytes.Buffer)
	g.out = outputBuffer

	err := g.Run()
	if err == nil {
		t.Error("Expected error")
	}
	if len(outputBuffer.Bytes()) != 0 {
		t.Error("Expected no output")
	}

}

type testFileSystem map[string][]byte // path -> content

func (fs testFileSystem) ReadFile(filename string) ([]byte, error) {
	data, found := fs[filename]
	if !found {
		return nil, &os.PathError{Op: "open", Path: filename, Err: syscall.ENOENT}
	}
	return data, nil
}

var fs = testFileSystem{
	"broken.tmpl":  []byte(`bad syntax, unfinished braces {{`),
	"package":      []byte(`package {{ .Package }}`),
	"unknown":      []byte(`hello {{ .foo }}`),
	"invalid.tmpl": []byte(`invalid go code`),
	"empty":        []byte(` `),
}

func makeGenerator() (*Generator, io.Writer) {
	inputBuffer := new(bytes.Buffer)
	return NewGenerator(inputBuffer, nil, fs), inputBuffer
}

type badReader struct{}

func (b badReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("bad")
}

func TestFailInput(t *testing.T) {
	g := NewGenerator(badReader{}, nil, fs)
	expectFail(t, g)
}

func TestBadInput(t *testing.T) {
	g, input := makeGenerator()
	_, err := input.Write([]byte("foo"))
	if err != nil {
		t.Fatal(err)
	}
	expectFail(t, g)
}

func writeRequest(t testing.TB, writer io.Writer, request *plugin.CodeGeneratorRequest) {
	requestData, err := proto.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	_, err = writer.Write(requestData)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNoParameters(t *testing.T) {
	g, input := makeGenerator()
	writeRequest(t, input, &plugin.CodeGeneratorRequest{})
	expectFail(t, g)
}

func TestBrokenTemplate(t *testing.T) {
	g, input := makeGenerator()
	writeRequest(t, input, &plugin.CodeGeneratorRequest{
		Parameter: proto.String("broken"),
	})
	expectFail(t, g)
}

func TestMissingDescriptor(t *testing.T) {
	g, input := makeGenerator()
	writeRequest(t, input, &plugin.CodeGeneratorRequest{
		FileToGenerate: []string{"foo"},
		Parameter:      proto.String("package.tmpl"),
	})
	expectFail(t, g)
}

func TestPackage(t *testing.T) {
	g, input := makeGenerator()

	const protoFileName = "test.proto"
	const packageName = "util"
	writeRequest(t, input, &plugin.CodeGeneratorRequest{
		FileToGenerate: []string{protoFileName},
		Parameter:      proto.String("package.tmpl"),
		ProtoFile: []*descriptor.FileDescriptorProto{{
			Name:    proto.String(protoFileName),
			Package: proto.String(packageName),
		}},
	})

	expectOutput(t, g, &plugin.CodeGeneratorResponse{
		Error: nil,
		File: []*plugin.CodeGeneratorResponse_File{{
			Name:    proto.String("test.pb.package.go"),
			Content: proto.String("package " + packageName),
		}},
	})
}

func TestPackageFormatted(t *testing.T) {
	g, input := makeGenerator()

	const protoFileName = "test.proto"
	const packageName = "util"
	writeRequest(t, input, &plugin.CodeGeneratorRequest{
		FileToGenerate: []string{protoFileName},
		Parameter:      proto.String("package,format"),
		ProtoFile: []*descriptor.FileDescriptorProto{{
			Name:    proto.String(protoFileName),
			Package: proto.String(packageName),
		}},
	})

	expectOutput(t, g, &plugin.CodeGeneratorResponse{
		Error: nil,
		File: []*plugin.CodeGeneratorResponse_File{{
			Name:    proto.String("test.pb.package.go"),
			Content: proto.String("package " + packageName + "\n"), // gofmt style fix includes trailing newline
		}},
	})
}

func TestExtraParameters(t *testing.T) {
	g, input := makeGenerator()

	const protoFileName = "test.proto"
	const packageName = "util"
	writeRequest(t, input, &plugin.CodeGeneratorRequest{
		FileToGenerate: []string{protoFileName},
		Parameter:      proto.String("format,package,broken"),
		ProtoFile: []*descriptor.FileDescriptorProto{{
			Name:    proto.String(protoFileName),
			Package: proto.String(packageName),
		}},
	})

	expectOutput(t, g, &plugin.CodeGeneratorResponse{
		Error: nil,
		File: []*plugin.CodeGeneratorResponse_File{{
			Name:    proto.String("test.pb.package.go"),
			Content: proto.String("package " + packageName + "\n"), // gofmt style fix includes trailing newline
		}},
	})
}

func TestEmptyFileToGenerate(t *testing.T) {
	g, input := makeGenerator()

	const protoFileName = "test.proto"
	const packageName = "util"
	writeRequest(t, input, &plugin.CodeGeneratorRequest{
		FileToGenerate: []string{""},
		Parameter:      proto.String("package.tmpl"),
		ProtoFile: []*descriptor.FileDescriptorProto{{
			Name:    proto.String(protoFileName),
			Package: proto.String(packageName),
		}},
	})

	expectOutput(t, g, &plugin.CodeGeneratorResponse{
		Error: nil,
		File:  nil,
	})
}

func TestUnknownField(t *testing.T) {
	g, input := makeGenerator()

	const protoFileName = "test.proto"
	writeRequest(t, input, &plugin.CodeGeneratorRequest{
		FileToGenerate: []string{protoFileName},
		Parameter:      proto.String("unknown.tmpl"),
		ProtoFile: []*descriptor.FileDescriptorProto{{
			Name: proto.String(protoFileName),
		}},
	})

	expectOutput(t, g, &plugin.CodeGeneratorResponse{
		Error: proto.String(anyError),
		File:  nil,
	})
}

func TestFormatInvalidCode(t *testing.T) {
	g, input := makeGenerator()

	const protoFileName = "test.proto"
	writeRequest(t, input, &plugin.CodeGeneratorRequest{
		FileToGenerate: []string{protoFileName},
		Parameter:      proto.String("invalid,format"),
		ProtoFile: []*descriptor.FileDescriptorProto{{
			Name: proto.String(protoFileName),
		}},
	})

	expectOutput(t, g, &plugin.CodeGeneratorResponse{
		Error: proto.String(anyError),
		File:  nil,
	})
}
