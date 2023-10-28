package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	pb "github.com/SkarpKat/ChatApp/Chat"
	"google.golang.org/grpc"
)

var (
	port = flag.Int("port", 10000, "The server port")
)

type ChatServiceServer struct {
	pb.UnimplementedChatServiceServer
}

func (s *ChatServiceServer) ChatRoute(stream pb.ChatService_ChatRouteServer) error {
	for {
		in, err := stream.Recv()
		if err != nil {
			return err
		}

		log.Printf("User: %s said: %s", in.GetUsername(), in.GetMessage())

		BrodcastMsg := in.GetUsername() + ": " + in.GetMessage()
		if in.Message != "" {
			// Send to all clients
			stream.Send(&pb.SendResponse{Message: BrodcastMsg})
		}
	}
}

func (s *ChatServiceServer) Connect(in *pb.ConnectRequest, stream pb.ChatService_ConnectServer) error {
	log.Printf("User: %s connected", in.GetUsername())
	stream.Send(&pb.ConnectResponse{Message: "Welcome to the chat " + in.GetUsername()})
	return nil
}

func (s *ChatServiceServer) Disconnect(ctx context.Context, in *pb.DisconnectRequest) (*pb.DisconnectResponse, error) {
	log.Printf("User: %s disconnected", in.GetUsername())
	return &pb.DisconnectResponse{}, nil
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))

	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	pb.RegisterChatServiceServer(grpcServer, &ChatServiceServer{})
	log.Printf("Server listening at %v", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %s", err)
	}
}
