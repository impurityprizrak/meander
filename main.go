package main

import (
	"flag"
	"fmt"
	pb "grpc"
	"log"
	"net"
	"node/node"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
)

func registerExitHandler(f func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		f()
		os.Exit(0)
	}()
}

func waitForSignal() {
	select {}
}

func main() {
	var basePath string

	flag.StringVar(&basePath, "path", "", "The path to store the server resources")
	flag.Parse()

	os.Setenv("BASE_PATH", basePath)

	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		err := os.Mkdir(node.BasePath, 0755)
		if err != nil {
			log.Fatalf("Failed to create path \"%s\": %v", basePath, err)
		} else {
			fmt.Printf("Path \"%s\" successfully created\n", basePath)
		}

	} else {
		fmt.Printf("Path \"%s\" already created\n", basePath)
	}

	node := node.NewLocalNode("0.0.0.0")
	node.Initialize()
	node.Attach()
	registerExitHandler(node.Dettach)

	const port string = "1313"
	listener, err := net.Listen("tcp", ":"+port)

	if err != nil {
		log.Fatalf("net.Listen: %v", err)
	}

	server := grpc.NewServer()
	service := &pb.MeanderServer{}

	pb.RegisterMeanderClientIOServer(server, service)

	if err = server.Serve(listener); err != nil {
		log.Fatal(err)
	} else {
		fmt.Println("Server started listening the port ", port)
	}

	waitForSignal()
}
