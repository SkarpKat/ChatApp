package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	pb "github.com/SkarpKat/ChatApp/Chat"
	"google.golang.org/grpc"
)

var (
	serverAddress = flag.String("server_address", "localhost:10000", "The server address in the format of host:port")
	userName      = flag.String("username", "SkarpKat", "The username")
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	flag.Parse()
	username := *userName
	conn, err := grpc.DialContext(ctx, *serverAddress, grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()
	client := pb.NewChatServiceClient(conn)

	stream, err := client.ChatRoute(ctx)
	if err != nil {
		log.Fatalf("Failed to create stream: %v", err)
	}
	connectRequest := &pb.ConnectRequest{Username: username}
	clicon, err := client.Connect(ctx, connectRequest)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	in, err := clicon.Recv()
	if err != nil {
		log.Fatalf("Failed to receive a ConnectRequest msg : %v", err)
	}
	fmt.Println(in.GetMessage())

	go func() {
		for {
			in, err := stream.Recv()
			if err != nil {
				log.Fatalf("Failed to receive a note : %v", err)
			}
			fmt.Println(in.GetMessage())
		}
	}()
	scanner := bufio.NewScanner(os.Stdin)
	for {
		var message string
		for scanner.Scan() {
			message = scanner.Text()
			break
		}
		if message == "/quit" {
			disCon, err := client.Disconnect(ctx, &pb.DisconnectRequest{Username: username})
			if err != nil {
				log.Fatalf("Failed to disconnect: %v", err)
			}
			fmt.Println(disCon.GetMessage())
			break
		}
		if message != "" {
			stream.Send(&pb.SendRequest{Username: username, Message: message})
		}
	}
	stream.CloseSend()
}
