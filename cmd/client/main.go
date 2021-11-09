package main

import (
	"bufio"
	"context"
	"flag"
	"github.com/Ruadgedy/pcbook-go/pb"
	"github.com/Ruadgedy/pcbook-go/sample"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

func createLaptop(laptopClient pb.LaptopServiceClient, laptop *pb.Laptop) {
	// 测试重复的laptop
	//laptop.Id = "e9b01f03-38ee-4cef-b27d-156813e81fa0"
	req := &pb.CreateLaptopRequest{Laptop: laptop}

	// set timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 调用LaptopServer去执行请求
	res, err := laptopClient.CreateLaptop(ctx, req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.AlreadyExists {
			log.Print("laptop already exists")
		} else {
			log.Fatal("cannot create laptop", err)
		}
		return
	}

	log.Printf("created laptop with id : %s", res.Id)
}

func main() {
	serverAddress := flag.String("address", "", "the server address")
	flag.Parse()

	log.Printf("dial server: %s", *serverAddress)

	conn, err := grpc.Dial(*serverAddress, grpc.WithInsecure())
	if err != nil {
		log.Fatal("cannot dial server: ", err)
	}

	laptopClient := pb.NewLaptopServiceClient(conn)

	testUploadImage(laptopClient)
}

func searchLaptop(client pb.LaptopServiceClient, filter *pb.Filter) {
	log.Print("search filter: ",filter)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &pb.SearchLaptopRequest{Filter: filter}
	stream, err := client.SearchLaptop(ctx, req)
	if err != nil {
		log.Fatal("cannot search laptop: ",err)
	}

	for {
		res, err := stream.Recv()
		// 将流中数据读取完毕
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Fatal("cannot receive response:", err)
		}

		laptop := res.GetLaptop()
		log.Print("- found:", laptop.GetId())
		log.Print(" + brand:", laptop.GetBrand())
		log.Print(" + name:", laptop.GetName())
		log.Print(" + cpu cores:", laptop.GetCpu().GetNumberCores())
		log.Print(" + cpu min ghz:", laptop.GetCpu().GetMinGhz())
		log.Print(" + ram:", laptop.GetRam())
		log.Print(" + price:", laptop.GetPriceUsd(),"usd")
	}
}

func testCreateLaptop(laptopClient pb.LaptopServiceClient)  {
	createLaptop(laptopClient, sample.NewLaptop())
}

func testSearchLaptop(laptopClient pb.LaptopServiceClient) {
	for i := 0; i < 10; i++ {
		createLaptop(laptopClient, sample.NewLaptop())
	}

	filter := &pb.Filter{
		MaxPriceUsd: 3000,
		MinCpuCores: 4,
		MinCpuGhz:   2.5,
		MinRam: &pb.Memory{
			Value: 8,
			Unit:  pb.Memory_GIGABYTE,
		},
	}

	searchLaptop(laptopClient,filter)
}

func testUploadImage(laptopClient pb.LaptopServiceClient) {
	laptop := sample.NewLaptop()
	createLaptop(laptopClient, laptop)
	uploadImage(laptopClient, laptop.GetId(), "tmp/laptop.jpg")
}

func uploadImage(laptopClient pb.LaptopServiceClient, laptopID string, imagePath string) {
	file, err := os.Open(imagePath)
	if err != nil {
		log.Fatal("cannot open image file: ", err)
	}
	defer file.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := laptopClient.UploadImage(ctx)
	if err != nil {
		log.Fatal("cannot upload image:", err)
	}

	// 首先构造上传图片的起始请求，该请求中包含了图片信息
	req := &pb.UploadImageRequest{
		Data: &pb.UploadImageRequest_Info{
			Info: &pb.ImageInfo{
				LaptopId:  laptopID,
				ImageType: filepath.Ext(imagePath),
			},
		} ,
	}

	err = stream.Send(req)
	if err != nil {
		log.Fatal("cannot send image info: ",err)
	}

	// 开始准备具体的图片数据
	reader := bufio.NewReader(file)
	buffer := make([]byte, 1024)

	// 循环读取图片chunk数据，并发送达server
	for  {
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal("cannot read chunk to buffer: ", err)
		}

		req := &pb.UploadImageRequest{
			Data: &pb.UploadImageRequest_ChunkData{
				ChunkData: buffer[:n],
			},
		}

		err = stream.Send(req)
		if err != nil {
			// 当Send出错了后，服务器就不再发送信息，此时需要我们手动接受错误信息
			err2 := stream.RecvMsg(nil)
			log.Fatal("cannot send chunk to server: ", err, err2)
		}
	}

	// 发送结束，关闭通道并接受对端响应
	res, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatal("cannot receive response: ", err)
	}

	log.Printf("image uploaded with id: %s, size: %d", res.GetId(), res.GetSize())
}