package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"sync"

	pb "github.com/SkarpKat/ChatApp/Chat"
	"google.golang.org/grpc"
)

var (
	port = flag.Int("port", 10000, "The server port")
)

type ChatServiceServer struct {
	pb.UnimplementedChatServiceServer

	// Mutex to synchronize access to the clientStreams slice
	mu            sync.Mutex
	clientStreams []pb.ChatService_ChatRouteServer
}

func (s *ChatServiceServer) ChatRoute(stream pb.ChatService_ChatRouteServer) error {
	s.mu.Lock()
	s.clientStreams = append(s.clientStreams, stream)
	s.mu.Unlock()

	for {
		in, err := stream.Recv()
		if err != nil {
			s.mu.Lock()
			// Remove the stream from the clientStreams slice upon client disconnect
			for i, clientStream := range s.clientStreams {
				if clientStream == stream {
					copy(s.clientStreams[i:], s.clientStreams[i+1:])
					s.clientStreams = s.clientStreams[:len(s.clientStreams)-1]
					break
				}
			}
			s.mu.Unlock()
			return err
		}

		log.Printf("User: %s said: %s", in.GetUsername(), in.GetMessage())

		BroadcastMsg := in.GetUsername() + ": " + in.GetMessage()
		if in.Message != "" {
			s.mu.Lock()
			// Send the message to all connected clients
			for _, clientStream := range s.clientStreams {
				if err := clientStream.Send(&pb.SendResponse{Message: BroadcastMsg}); err != nil {
					log.Printf("Error sending message: %v", err)
				}
			}
			s.mu.Unlock()
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

	chatServer := &ChatServiceServer{
		clientStreams: make([]pb.ChatService_ChatRouteServer, 0),
	}

	pb.RegisterChatServiceServer(grpcServer, chatServer)
	log.Printf("Server listening at %v", lis.Addr())

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %s", err)
	}
}
