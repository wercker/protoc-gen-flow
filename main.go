//   Copyright 2017 Wercker Holding BV
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
)

func main() {
	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	var req plugin.CodeGeneratorRequest
	if err := proto.Unmarshal(data, &req); err != nil {
		log.Fatalf("unable to parse protobuf: %v", err)
	}

	var files []*plugin.CodeGeneratorResponse_File
	for _, f := range req.ProtoFile[3:] {
		code := bytes.NewBuffer(nil)
		fileTemplate.Execute(code, f)

		for _, msg := range f.MessageType {
			m := &Message{
				Name:   *msg.Name,
				Fields: []*Field{},
			}

			for _, field := range msg.Field {
				m.Fields = append(m.Fields, &Field{
					Name: *field.Name,
					Type: getFieldType(field, *f.Package),
				})
			}

			messageTemplate.Execute(code, m)
		}

		name := f.GetName()
		ext := filepath.Ext(name)
		base := strings.TrimSuffix(name, ext)
		output := fmt.Sprintf("%s.flow.js", base)

		files = append(files, &plugin.CodeGeneratorResponse_File{
			Name:    proto.String(output),
			Content: proto.String(strings.TrimLeft(code.String(), "\n")),
		})
	}

	emitFiles(files)
}

func emitFiles(out []*plugin.CodeGeneratorResponse_File) {
	emitResp(&plugin.CodeGeneratorResponse{File: out})
}

func emitError(err error) {
	emitResp(&plugin.CodeGeneratorResponse{Error: proto.String(err.Error())})
}

func emitResp(resp *plugin.CodeGeneratorResponse) {
	buf, err := proto.Marshal(resp)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := os.Stdout.Write(buf); err != nil {
		log.Fatal(err)
	}
}

func getFieldType(field *descriptor.FieldDescriptorProto, pkg string) string {
	ret := "any" // unknonwn

	switch *field.Type {
	case descriptor.FieldDescriptorProto_TYPE_DOUBLE,
		descriptor.FieldDescriptorProto_TYPE_FLOAT,
		descriptor.FieldDescriptorProto_TYPE_INT32,
		descriptor.FieldDescriptorProto_TYPE_FIXED32,
		descriptor.FieldDescriptorProto_TYPE_UINT32,
		descriptor.FieldDescriptorProto_TYPE_SFIXED32,
		descriptor.FieldDescriptorProto_TYPE_SINT32:
		ret = "number"
	case descriptor.FieldDescriptorProto_TYPE_INT64,
		descriptor.FieldDescriptorProto_TYPE_UINT64,
		descriptor.FieldDescriptorProto_TYPE_FIXED64,
		descriptor.FieldDescriptorProto_TYPE_SFIXED64,
		descriptor.FieldDescriptorProto_TYPE_SINT64:
		// javascript doesn't support 64bit ints
		ret = "string"
	case descriptor.FieldDescriptorProto_TYPE_BOOL:
		ret = "boolean"
	case descriptor.FieldDescriptorProto_TYPE_STRING:
		ret = "string"
	case descriptor.FieldDescriptorProto_TYPE_GROUP:
	case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
		if *field.TypeName == ".google.protobuf.Timestamp" {
			// Special case for handling timestamps
			ret = "string"
		} else {
			ret = strings.TrimPrefix(*field.TypeName, fmt.Sprintf(".%s.", pkg))
		}
	case descriptor.FieldDescriptorProto_TYPE_BYTES:
	case descriptor.FieldDescriptorProto_TYPE_ENUM:
		ret = "UNKNONWN TYPE"
	}

	if *field.Label == descriptor.FieldDescriptorProto_LABEL_REPEATED {
		ret += "[]"
	}

	return ret
}
