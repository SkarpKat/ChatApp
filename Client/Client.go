package main

import (
	"bufio"
	"context"
	"flag"
	"io"
	"log"
	"os"

	pb "github.com/SkarpKat/ChatApp/Chat"
	"google.golang.org/grpc"
)

var (
	serverAddress   = flag.String("server_address", "localhost:10000", "The server address in the format of host:port")
	userName        = flag.String("username", "SkarpKat", "The username")
	clientTimestamp = int64(0)
)

func updateLamportTimestamp(timestamp int64) {
	if timestamp > clientTimestamp {
		clientTimestamp = timestamp + 1
	} else {
		clientTimestamp++
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	flag.Parse()

	logPath := "Client/" + *userName + "Client.log"

	file, err := os.OpenFile(logPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer file.Close()
	log.SetOutput(io.MultiWriter(file, os.Stdout))

	username := *userName
	conn, err := grpc.DialContext(ctx, *serverAddress, grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()
	client := pb.NewChatServiceClient(conn)

	clientTimestamp++
	log.Printf("Connecting to server at time: %d", clientTimestamp)
	connectRequest := &pb.ConnectRequest{Username: username, Timestamp: clientTimestamp}
	clicon, err := client.Connect(ctx, connectRequest)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	resp, err := clicon.Recv()
	updateLamportTimestamp(resp.GetTimestamp())

	if err != nil {
		log.Fatalf("Failed to receive message: %v", err)
	}
	log.Printf("%s at time: %d", resp.GetMessage(), clientTimestamp)
	clicon.CloseSend()

	stream, err := client.ChatRoute(ctx)
	if err != nil {
		log.Fatalf("Failed to create stream: %v", err)
	}

	go func() {
		for {
			in, err := stream.Recv()
			updateLamportTimestamp(in.GetTimestamp())
			if err != nil {
				log.Fatalf("Failed to receive a note : %v", err)
			}
			log.Printf("%s at time: %d", in.GetMessage(), clientTimestamp)
			// fmt.Println(in.GetMessage())
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		var message string
		for scanner.Scan() {
			message = scanner.Text()
			//check if message is below or equal to 128 characters
			if len(message) <= 128 {
				break
			} else {
				log.Printf("Message is too long, please keep it under 128 characters")
			}
		}
		if message == "/quit" {
			clientTimestamp++
			log.Printf("Disconnecting from server at time: %d", clientTimestamp)
			disCon, err := client.Disconnect(ctx, &pb.DisconnectRequest{Username: username, Timestamp: clientTimestamp})
			if err != nil {
				log.Fatalf("Failed to disconnect: %v", err)
			}
			resp, err := disCon.Recv()
			if err != nil {
				log.Fatalf("Failed to receive message: %v", err)
			}
			updateLamportTimestamp(resp.GetTimestamp())
			log.Printf("%s at time %d", resp.GetMessage(), clientTimestamp)
			// fmt.Println(resp.GetMessage())
			disCon.CloseSend()
			stream.CloseSend()
			break
		}
		clientTimestamp++
		log.Printf("Sending message: %s at time: %d", message, clientTimestamp)
		stream.Send(&pb.SendRequest{Username: username, Message: message, Timestamp: clientTimestamp})
	}
}
