package service

import (
	"bytes"
	"context"
	"errors"
	"github.com/Ruadgedy/pcbook-go/pb"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"log"
)

const maxImageSize = 1 << 20

// LaptopServer is the server that provides the laptop services.
type LaptopServer struct {
	laptopStore LaptopStore
	imageStore ImageStore
	ratingStore RatingStore
	pb.UnimplementedLaptopServiceServer // UnimplementedLaptopServiceServer must be embedded to have forward compatible implementations.
}

// 输入unary，输出unary
func (server *LaptopServer) CreateLaptop(cxt context.Context, req *pb.CreateLaptopRequest) (*pb.CreateLaptopResponse, error) {
	laptop := req.GetLaptop()
	log.Printf("receive a create laptop request with id :%s", laptop.Id)

	if len(laptop.Id) > 0 {
		// check if it's a valid UUID
		_, err := uuid.Parse(laptop.Id)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "code ID is not a valid UUID: %v", err)
		}
	} else {
		id, err := uuid.NewRandom()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "cannot generate a new laptop ID:%v", err)
		}
		laptop.Id = id.String()
	}

	// some heavy processing to satisfy timeout
	//time.Sleep(6*time.Second)
	// 判断请求是否被取消
	if cxt.Err() == context.Canceled {
		log.Print("request is canceled")
		return nil, status.Errorf(codes.Canceled, "request is canceled")
	}
	// 判断请求是否超时
	if cxt.Err() == context.DeadlineExceeded {
		log.Print("deadline exceeded")
		return nil, status.Error(codes.DeadlineExceeded,"deadline exceeded")
	}

	// save the laptop to in-memory storage
	if err := server.laptopStore.Save(laptop); err != nil {
		code := codes.Internal
		if errors.Is(err, ErrAlreadyExists) {
			code = codes.AlreadyExists
		}

		return nil, status.Errorf(code, "cannot save laptop to the store: %v", err)
	}

	log.Printf("saved laptop with id : %s", laptop.Id)

	res := &pb.CreateLaptopResponse{Id: laptop.Id}
	return res, nil
}

// 输入unary， 输出stream
func (server *LaptopServer) SearchLaptop(req *pb.SearchLaptopRequest,stream pb.LaptopService_SearchLaptopServer) error{
	filter := req.GetFilter()
	log.Printf("receive a search-laptop request with filter: %v",filter)

	err := server.laptopStore.Search(
		stream.Context(), // 传递流上下文
		filter,
		func(laptop *pb.Laptop) error {
			res := &pb.SearchLaptopResponse{Laptop: laptop}

			err := stream.Send(res)
			if err != nil {
				return err
			}

			log.Printf("sent laptop with id: %s", laptop.GetId())
			return nil
		},
	)
	if err != nil {
		return status.Errorf(codes.Internal, "unexpected error: %v",err)
	}
	return nil
}

// 输入stream，输出unary
// UploadImage is a client-streaming RPC to upload a laptop image
func (server *LaptopServer)UploadImage(stream pb.LaptopService_UploadImageServer) error{
	req, err := stream.Recv()
	if err != nil {
		return logError(status.Errorf(codes.Unknown, "cannot receive image info"))
	}

	// 获取到请求中的laptopID和imageType
	laptopId := req.GetInfo().GetLaptopId()
	imageType := req.GetInfo().GetImageType()
	log.Printf("receivae an upload-image request for laptop %s with image type %s", laptopId, imageType)

	// 查找给定的laptop是否存在
	laptop, err := server.laptopStore.Find(laptopId)
	if err != nil {
		return logError(status.Errorf(codes.Internal,"cannot find laptop: %v", err))
	}
	if laptop == nil {
		return logError(status.Errorf(codes.InvalidArgument, "laptop id %s doesn't exist",laptopId))
	}

	imageData := bytes.Buffer{}
	imageSize := 0

	for{
		err := contextError(stream.Context())
		if err != nil {
			return err
		}

		log.Println("waiting to receive more data")

		req, err := stream.Recv()
		if err == io.EOF{
			log.Println(" no more data")
			break
		}
		if err != nil {
			return logError(status.Errorf(codes.Unknown, "cannot receive chunk data: %v",err))
		}

		chunk := req.GetChunkData()
		size := len(chunk)

		log.Printf("receive a chunk with size: %d", size)

		imageSize += size
		if imageSize > maxImageSize {
			return logError(status.Errorf(codes.InvalidArgument," image is too large:%d > %d", imageSize, maxImageSize))
		}

		// write slowly
		//time.Sleep(time.Second)

		// write data to file
		_, err = imageData.Write(chunk)
		if err != nil {
			return logError(status.Errorf(codes.Internal, "cannot write chunk data: %v", err))
		}
	}

	imageID, err := server.imageStore.Save(laptopId, imageType, imageData)
	if err != nil {
		return logError(status.Errorf(codes.Internal, "cannot save image to the store: %v", err))
	}

	res:= &pb.UploadImageResponse{
		Id:   imageID,
		Size: uint32(imageSize),
	}

	// send response to client
	err = stream.SendAndClose(res)
	if err != nil {
		return logError(status.Errorf(codes.Unknown, "cannot send response: %v",err))
	}

	log.Printf("saved image with id: %s, size: %d", imageID, imageSize)
	return nil
}

// 输入stream，输出stream
// RateLaptop is a bidirectional-stream RPC that allows client to rate a stream of laptops
// with a score, and returns a stream of average score for each of them
func (server *LaptopServer)RateLaptop(stream pb.LaptopService_RateLaptopServer) error{
	for {
		err := contextError(stream.Context())
		if err != nil {
			return err
		}

		req, err := stream.Recv()
		if err == io.EOF {
			log.Println("no more data")
			break
		}
		if err != nil {
			return logError(status.Errorf(codes.Unknown, " cannot receive request: %v",err))
		}

		laptopId := req.GetLaptopId()
		score := req.GetScore()

		log.Printf("receive a rate-laptop request: id=%s, score=%.2f",laptopId, score)

		found, err := server.laptopStore.Find(laptopId)
		if err != nil {
			return logError(status.Errorf(codes.Internal,"cannot find laptop: %v",err))
		}
		if found == nil {
			return logError(status.Errorf(codes.NotFound,"laptop %s is not found",laptopId))
		}

		rating, err := server.ratingStore.Add(laptopId, score)
		if err != nil {
			return logError(status.Errorf(codes.Internal,"cannot add rating to the store: %v",err))
		}

		res := &pb.RateLaptopResponse{
			LaptopId:     laptopId,
			RatedCount:   rating.Count,
			AverageScore: rating.Sum / float64(rating.Count),
		}

		err = stream.Send(res)
		if err != nil {
			return logError(status.Errorf(codes.Unknown,"cannot send stream response: %v", err))
		}
	}
	return nil
}

func contextError(ctx context.Context) error {
	switch ctx.Err() {
	case context.Canceled:
		return logError(status.Error(codes.Canceled,"request is canceled"))
	case context.DeadlineExceeded:
		return logError(status.Error(codes.DeadlineExceeded, "deadline is exceeded"))
	default:
		return nil
	}
}

func logError(err error) error {
	if err != nil {
		log.Print(err)
	}
	return err
}

func NewLaptopServer(laptopStore LaptopStore, imageStore ImageStore, ratingStore RatingStore) *LaptopServer {
	return &LaptopServer{
		laptopStore:                      laptopStore,
		imageStore:                       imageStore,
		ratingStore:                      ratingStore,
	}
}
