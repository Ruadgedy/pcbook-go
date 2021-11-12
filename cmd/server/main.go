package main

import (
	"flag"
	"fmt"
	"github.com/Ruadgedy/pcbook-go/pb"
	"github.com/Ruadgedy/pcbook-go/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"time"
)

func seedUsers(userStore service.UserStore) error {
	err := createUser(userStore, "admin1", "secret", "admin")
	if err != nil {
		return nil
	}
	return createUser(userStore,"user1","secret", "user")
}

func createUser(userStore service.UserStore, username,password, role string) error {
	user, err := service.NewUser(username, password, role)
	if err != nil {
		return err
	}
	return userStore.Save(user)
}

const(
	secretKey = "secret"
	tokenDuration = 15 *time.Minute
)

func accessibleRoles() map[string][]string {
	const laptopServicePath = "/techschool.pcbook.LaptopService/"
	return map[string][]string{
		laptopServicePath + "CreateLaptop": {"admin"},
		laptopServicePath + "UploadImage" : {"admin"},
		laptopServicePath + "RateLaptop" : {"admin", "user"},
	}
}

func main() {
	port := flag.Int("port", 0, "the server port") // 返回值是指针类型
	flag.Parse()
	log.Printf("start server on port %d", *port)

	laptopStore := service.NewInMemoryLaptopStore()
	imageStore := service.NewDiskImageStore("img")
	ratingStore := service.NewInMemoryRatingStore()
	userStore := service.NewInMemoryUserStore()
	err := seedUsers(userStore) // 注册模拟用户
	if err != nil {
		log.Fatal("cannot seed users")
	}
	jwtManager := service.NewJWTManager(secretKey, tokenDuration)
	authServer := service.NewAuthServer(userStore, jwtManager)
	interceptor := service.NewAuthInterceptor(jwtManager, accessibleRoles())

	laptopServer := service.NewLaptopServer(laptopStore, imageStore, ratingStore)
	grpcServer := grpc.NewServer( // 创建新的gRPC服务器实例，但此时服务器实例未与我们定义的服务器注册绑定
		grpc.UnaryInterceptor(interceptor.Unary()),    // 添加unary interceptor
		grpc.StreamInterceptor(interceptor.Stream()),   // 添加stream interceptor
	)
	pb.RegisterLaptopServiceServer(grpcServer, laptopServer)    // 将我们自定义跌服务器与gRPC服务器绑定
	pb.RegisterAuthServiceServer(grpcServer, authServer)    // 将我们自定义的认证服务器与gRPC服务器绑定
	reflection.Register(grpcServer) // 注册gRPC reflection

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
