package service_test

import (
	"context"
	"github.com/Ruadgedy/pcbook-go/pb"
	"github.com/Ruadgedy/pcbook-go/sample"
	"github.com/Ruadgedy/pcbook-go/serializer"
	"github.com/Ruadgedy/pcbook-go/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"net"
	"testing"
)

func TestClientCreateLaptop(t *testing.T) {
	t.Parallel()

	laptopServer, serverAddress := startTestLaptopServer(t)
	laptopClient := newTestLaptopClient(t, serverAddress)

	laptop := sample.NewLaptop()
	expected := laptop.Id
	req := &pb.CreateLaptopRequest{Laptop: laptop}

	res, err := laptopClient.CreateLaptop(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, expected, res.Id)

	// check that the laptops
	other, err := laptopServer.Store.Find(res.Id)
	require.NoError(t, err)
	require.NotNil(t, other)

	// check that the saved laptop is the same as the one we send
	requireSameLaptop(t, laptop, other)

}

func requireSameLaptop(t *testing.T, laptop *pb.Laptop, other *pb.Laptop) {
	// 不能直接比较，因为laptop中有一些proto产生的grpc字段
	//require.Equal(t,laptop, other)

	json1, err := serializer.ProtobufToJSON(laptop)
	require.NoError(t, err)

	json2, err := serializer.ProtobufToJSON(other)
	require.NoError(t, err)

	require.Equal(t, json1, json2)
}

func newTestLaptopClient(t *testing.T, serverAddress string) pb.LaptopServiceClient {
	conn, err := grpc.Dial(serverAddress, grpc.WithInsecure())
	require.NoError(t, err)
	return pb.NewLaptopServiceClient(conn)
}

func startTestLaptopServer(t *testing.T) (*service.LaptopServer, string) {
	LaptopServer := service.NewLaptopServer(service.NewInMemoryLaptopStore())

	grpcServer := grpc.NewServer()
	pb.RegisterLaptopServiceServer(grpcServer, LaptopServer)

	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	go grpcServer.Serve(listener) // block call

	return LaptopServer, listener.Addr().String()
}
