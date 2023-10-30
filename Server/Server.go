package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"

	pb "github.com/SkarpKat/ChatApp/Chat"
	"google.golang.org/grpc"
)

var (
	port            = flag.Int("port", 10000, "The server port")
	serverTimestamp = int64(0)
	clientsName     = make([]string, 0)
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
		maxLamportTimestamp(in.GetTimestamp())
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

		log.Printf("User: %s said: %s at time: %d", in.GetUsername(), in.GetMessage(), in.GetTimestamp())

		BroadcastString := in.GetUsername() + ": " + in.GetMessage()
		if in.Message != "" {
			s.BroadcastMsg(BroadcastString)
		}
	}
}

func (s *ChatServiceServer) Connect(in *pb.ConnectRequest, stream pb.ChatService_ConnectServer) error {
	maxLamportTimestamp(in.GetTimestamp())
	log.Printf("User: %s connected at time: %d", in.GetUsername(), serverTimestamp)
	clientsName = append(clientsName, in.GetUsername())

	connectMsg := in.GetUsername() + " has connected"
	s.BroadcastMsg(connectMsg)
	serverTimestamp++
	stream.Send(&pb.ConnectResponse{Message: "Welcome to the chat " + in.GetUsername() + "!", Timestamp: serverTimestamp})
	return nil
}

func (s *ChatServiceServer) Disconnect(in *pb.DisconnectRequest, stream pb.ChatService_DisconnectServer) error {
	maxLamportTimestamp(in.GetTimestamp())
	log.Printf("User: %s disconnected at time: %d", in.GetUsername(), in.GetTimestamp())
	//remove user from clientsName
	for i, clientName := range clientsName {
		if clientName == in.GetUsername() {
			copy(clientsName[i:], clientsName[i+1:])
			clientsName = clientsName[:len(clientsName)-1]
			break
		}
	}

	disconnectMsg := in.GetUsername() + " has disconnected"
	s.BroadcastMsg(disconnectMsg)
	serverTimestamp++
	stream.Send(&pb.DisconnectResponse{Message: "Goodbye " + in.GetUsername() + "!", Timestamp: serverTimestamp})
	return nil
}

func (s *ChatServiceServer) BroadcastMsg(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Send the message to all connected clients
	for _, clientStream := range s.clientStreams {
		serverTimestamp++
		if err := clientStream.Send(&pb.SendResponse{Message: message, Timestamp: serverTimestamp}); err != nil {
			log.Printf("Error sending message: %v", err)
		}
	}
}

func maxLamportTimestamp(timestamp int64) {
	if timestamp > serverTimestamp {
		serverTimestamp = timestamp
	}
}

func main() {
	flag.Parse()

	file, err := os.OpenFile("Server/Server.log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer file.Close()
	log.SetOutput(io.MultiWriter(file, os.Stdout))

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

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for {
			for scanner.Scan() {
				message := scanner.Text()
				switch message {
				case "/kill":
					log.Printf("Server shutting down ...")
					os.Exit(0)
				case "/timestamp":
					log.Printf("Current timestamp: %d", serverTimestamp)
				case "/clients":
					for _, clientName := range clientsName {
						log.Printf("Client: %s\n", clientName)
					}
				case "/help":
					log.Printf("Commands:\n/kill - kills the server\n/timestamp - prints the current timestamp\n/clients - prints the current clients connected to the server\n/help - prints the commands")
				default:
					log.Printf("Invalid command")
				}

			}

		}
	}()

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %s", err)
	}

	//Input for commands to server

}
