package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	pb "github.com/Zheleznov-Fedor/new-ya-long-calc/proto"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
)

type Server struct {
	pb.OperationServiceServer // сервис из сгенерированного пакета
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Calc(
	ctx context.Context,
	in *pb.OperationRequest,
) (*pb.OperationResult, error) {
	log.Println("request: ", in)
	var n int
	var res float32

	switch in.Oper {
	case "+":
		n, _ = strconv.Atoi(os.Getenv("TIME_ADD"))
		res = in.A + in.B
		break
	case "-":
		n, _ = strconv.Atoi(os.Getenv("TIME_SUBSTR"))
		res = in.A - in.B
	case "*":
		n, _ = strconv.Atoi(os.Getenv("TIME_MULT"))
		res = in.A * in.B
	case "/":
		n, _ = strconv.Atoi(os.Getenv("TIME_DIVISION"))
		res = in.A / in.B
	}

	time.Sleep(time.Duration(n) * time.Second)

	return &pb.OperationResult{
		Result: res,
	}, nil
}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Error loading .env file")
	}
	host := "localhost"
	port := os.Args[1]

	addr := fmt.Sprintf("%s:%s", host, port)
	lis, err := net.Listen("tcp", addr) // будем ждать запросы по этому адресу

	if err != nil {
		log.Println("error starting tcp listener: ", err)
		os.Exit(1)
	}

	log.Println("tcp listener started at port: ", port)
	// создадим сервер grpc
	grpcServer := grpc.NewServer()
	// объект структуры, которая содержит реализацию
	// серверной части GeometryService
	geomServiceServer := NewServer()
	// зарегистрируем нашу реализацию сервера
	pb.RegisterOperationServiceServer(grpcServer, geomServiceServer)
	// запустим grpc сервер
	if err := grpcServer.Serve(lis); err != nil {
		log.Println("error serving grpc: ", err)
		os.Exit(1)
	}
}
