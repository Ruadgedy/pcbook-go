package serializer

import (
	"fmt"
	"github.com/Ruadgedy/pcbook-go/pb"
	"github.com/Ruadgedy/pcbook-go/sample"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestFileSerializer(t *testing.T) {
	t.Parallel()

	binaryFile := "../tmp/laptop.bin"
	jsonFile := "../tmp/laptop.json"

	laptop1 := sample.NewLaptop()
	err := WriteProtobufToBinary(laptop1, binaryFile)
	require.NoError(t, err)
	err = WriteProtobufToJSONFile(laptop1, jsonFile)
	require.NoError(t, err)
}

func TestFileUnserialize(t *testing.T) {
	t.Parallel()

	binaryFile := "../tmp/laptop.bin"
	laptop2 := &pb.Laptop{}

	err := ReadProtobufFromBinaryFile(binaryFile, laptop2)
	require.NoError(t, err)

	fmt.Printf("%v", laptop2)
}
