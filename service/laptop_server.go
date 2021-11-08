package service

import (
	"context"
	"errors"
	"github.com/Ruadgedy/pcbook-go/pb"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
)

// LaptopServer is the server that provides the laptop services.
type LaptopServer struct {
	Store LaptopStore
	pb.UnimplementedLaptopServiceServer
}

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
	if err := server.Store.Save(laptop); err != nil {
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

func (server *LaptopServer) SearchLaptop(req *pb.SearchLaptopRequest,stream pb.LaptopService_SearchLaptopServer) error{
	filter := req.GetFilter()
	log.Printf("receive a search-laptop request with filter: %v",filter)

	err := server.Store.Search(
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


func NewLaptopServer(store LaptopStore) *LaptopServer {
	return &LaptopServer{Store: store}
}
