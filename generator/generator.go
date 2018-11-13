package generator

import (
	"bytes"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
	"go/format"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
	"text/template"
)

// FileReader is an interface for a Generator to read files
type FileReader interface {
	ReadFile(filename string) ([]byte, error)
}

// Generator reads in a CodeGeneratorRequest and writes out a CodeGeneratorResponse
type Generator struct {
	in           io.Reader
	out          io.Writer
	fileReader   FileReader
	formatOutput bool
}

// NewGenerator creates a new Generator
func NewGenerator(in io.Reader, out io.Writer, fileReader FileReader) *Generator {
	return &Generator{
		in:           in,
		out:          out,
		fileReader:   fileReader,
		formatOutput: false,
	}
}

// Run the Generator to read the input and generate the output
func (g *Generator) Run() error {
	input, err := ioutil.ReadAll(g.in)
	if err != nil {
		return fmt.Errorf("error reading input: %s", err)
	}

	request := new(plugin.CodeGeneratorRequest)
	if err := proto.Unmarshal(input, request); err != nil {
		return fmt.Errorf("error parsing input: %s", err)
	}

	templateName, templateData := g.parseParameters(request.GetParameter())
	if len(templateName) == 0 {
		return fmt.Errorf("no template files in the parameters %q", request.GetParameter())
	}

	fileTemplate, err := template.New(filepath.Base(templateName)).Parse(string(templateData))
	if err != nil {
		return fmt.Errorf("error parsing template %s: %s", templateName, err)
	}

	protoFiles := make(map[string]*descriptor.FileDescriptorProto)
	for _, protoFile := range request.GetProtoFile() {
		protoFiles[protoFile.GetName()] = protoFile
	}

	response := new(plugin.CodeGeneratorResponse)
	for _, fileName := range request.GetFileToGenerate() {
		if len(fileName) == 0 {
			continue
		}

		protoFile, protoFileFound := protoFiles[fileName]
		if !protoFileFound {
			return fmt.Errorf("%s descriptor not found", fileName)
		}

		fileResponse, err := g.applyTemplate(fileTemplate, protoFile)
		if err != nil {
			response.Error = proto.String(err.Error())
			break
		}

		response.File = append(response.File, fileResponse)
	}

	{
		responseData, err := proto.Marshal(response)
		if err != nil {
			return fmt.Errorf("error marshalling output: %s", err)
		}

		if _, err := g.out.Write(responseData); err != nil {
			return fmt.Errorf("error writing output: %s", err)
		}
	}
	return nil
}

// parseParameters handles the command-line parameters passed in via protoc.
// The parameters must be comma-separated and must contain a valid file name
// for the go template. Additional valid parameters:
//  - format: apply gofmt style fixes to the output
func (g *Generator) parseParameters(parameters string) (templateName string, templateData []byte) {
	for _, parameter := range strings.Split(parameters, ",") {
		switch parameter {
		case "format":
			g.formatOutput = true
		default:
			if len(templateData) > 0 {
				break // only use the first valid filename
			}

			// try to read the file name both with and without the common ".tmpl" file extension
			const templateSuffix = ".tmpl"
			templateFileName := strings.TrimSuffix(parameter, templateSuffix)
			fileContent, err := g.fileReader.ReadFile(templateFileName)
			if err == nil {
				templateData = fileContent
				templateName = filepath.Base(templateFileName)
			} else {
				var err error
				fileContent, err = g.fileReader.ReadFile(templateFileName + templateSuffix)
				if err == nil {
					templateData = fileContent
					templateName = filepath.Base(templateFileName)
				}
			}
		}
	}
	return
}

func (g *Generator) applyTemplate(t *template.Template, input *descriptor.FileDescriptorProto) (*plugin.CodeGeneratorResponse_File, error) {
	buf := new(bytes.Buffer)
	err := t.Execute(buf, input)
	if err != nil {
		return nil, err
	}

	content := buf.Bytes()
	if g.formatOutput {
		var err error
		content, err = formatGoSource(content)
		if err != nil {
			return nil, err
		}
	}

	return &plugin.CodeGeneratorResponse_File{
		Name:    proto.String(strings.TrimSuffix(input.GetName(), ".proto") + ".pb." + t.Name() + ".go"),
		Content: proto.String(string(content)),
	}, nil
}

func formatGoSource(in []byte) ([]byte, error) {
	out, err := format.Source(bytes.TrimSpace(in))
	if err != nil {
		return nil, err
	}
	return out, nil
}
