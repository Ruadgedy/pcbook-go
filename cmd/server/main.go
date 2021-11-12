package main

import (
	"flag"
	"fmt"
	"github.com/Ruadgedy/pcbook-go/pb"
	"github.com/Ruadgedy/pcbook-go/service"
	"google.golang.org/grpc"
	"log"
	"net"
)

func main() {
	port := flag.Int("port", 0, "the server port") // 返回值是指针类型
	flag.Parse()
	log.Printf("start server on port %d", *port)

	laptopStore := service.NewInMemoryLaptopStore()
	imageStore := service.NewDiskImageStore("img")
	ratingStore := service.NewInMemoryRatingStore()

	laptopServer := service.NewLaptopServer(laptopStore, imageStore, ratingStore)
	grpcServer := grpc.NewServer() // 创建新的gRPC服务器实例，但此时服务器实例未与我们定义的服务器注册绑定
	pb.RegisterLaptopServiceServer(grpcServer, laptopServer)    // 将我们自定义跌服务器与gRPC服务器绑定

	address := fmt.Sprintf("0.0.0.0:%d", *port)
	log.Printf("address: %s", address)
	listener, err := net.Listen("tcp", address) // 注册监听器
	if err != nil {
		log.Fatalf("cannnot start server:%v", err)
	}

	err = grpcServer.Serve(listener)    // 开启服务
	if err != nil {
		log.Fatalf("cannnot start server:%v", err)
	}
}
