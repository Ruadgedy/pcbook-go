package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"github.com/Ruadgedy/pcbook-go/pb"
	"github.com/Ruadgedy/pcbook-go/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"io/ioutil"
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

func loadTLSCredentials() (credentials.TransportCredentials, error) {
	// Load certificate of the CA who signed server\s certificate
	pemClientCA, err := ioutil.ReadFile("cert/ca-cert.pem")
	if err != nil {
		return nil,err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemClientCA) {
		return nil, fmt.Errorf("failed to add client CA's certificate")
	}

	// Load server's certificate and private key
	serverCert, err := tls.LoadX509KeyPair("cert/ca-cert.pem", "cert/ca-key.pem")
	if err != nil {
		return nil,err
	}

	// Create the credentials and return it
	config := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		//ClientAuth: tls.NoClientCert, // 这里我们定义client不需要认证，因为我们目前使用的是server端认证
		ClientAuth: tls.RequireAndVerifyClientCert,  // 定义client也需要认证，即server需要验证client的证书
		ClientCAs: certPool,    // 定义一组可信CA的证书，这些CA签名了client的证书
	}

	return credentials.NewTLS(config), nil
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
	tlsCredentials, err := loadTLSCredentials() // 加载TLS凭证
	if err != nil {
		log.Fatal("cannot load TLS credentials: ",err)
	}

	laptopServer := service.NewLaptopServer(laptopStore, imageStore, ratingStore)
	grpcServer := grpc.NewServer( // 创建新的gRPC服务器实例，但此时服务器实例未与我们定义的服务器注册绑定
		grpc.UnaryInterceptor(interceptor.Unary()),    // 添加unary interceptor
		grpc.StreamInterceptor(interceptor.Stream()),   // 添加stream interceptor
		grpc.Creds(tlsCredentials), // 添加服务器连接的凭证
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
