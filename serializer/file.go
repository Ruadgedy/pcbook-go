package serializer

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"io/ioutil"
)

// WriteProtobufToBinary writes protocol buffer message tp binary file
func WriteProtobufToBinary(message proto.Message, filename string) error {
	data, err := proto.Marshal(message)
	if err != nil {
		return fmt.Errorf("cannot marshal proto message to binary: %w", err)
	}

	err = ioutil.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("cannot write binary data to file : %w", err)
	}

	return nil
}

func ReadProtobufFromBinaryFile(filePath string, message proto.Message) error {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("cannot read binary data from file: %w", err)
	}

	if err = proto.Unmarshal(data, message); err != nil {
		return fmt.Errorf("cannot unmarsh binary to proto message: %w", err)
	}
	return nil
}

func WriteProtobufToJSONFile(message proto.Message, filename string) error {
	data, err := ProtobufToJSON(message)
	if err != nil {
		return fmt.Errorf("cannot marshal protobuf message to json: %w", err)
	}
	if err = ioutil.WriteFile(filename, []byte(data), 0644); err != nil {
		return fmt.Errorf("cannot write data to file : %w", err)
	}
	return nil
}

func ReadProtobufFromJSONFile(filename string, message proto.Message) error {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("read file error: %v", err)
	}

	if err = JSONToProtobufMessage(string(file), message); err != nil {
		return fmt.Errorf("cannot unmarshal message: %v", err)
	}
	return nil
}
