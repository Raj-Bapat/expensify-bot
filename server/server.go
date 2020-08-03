package main

import (
	"context"
	"database/sql"
	pb "expensify-bot/proto"
	"fmt"
	ex "github.com/me-io/go-swap/pkg/exchanger"
	"github.com/me-io/go-swap/pkg/swap"
	//idk
	_ "github.com/go-sql-driver/mysql"
	//hi
	"google.golang.org/grpc"
	// "google.golang.org/protobuf/proto"
	"log"
	// "sort"
	"net"
)

func handle(err error) {
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
}

type server struct {
	db *sql.DB
	pb.UnimplementedServicesServer
}

func (s *server) TopCategory(ctx context.Context, treq *pb.TopRequest) (*pb.TopResponse, error) {
	m := make(map[string]float64)
	rows, err := s.db.Query("SELECT SUM(Amount) as total, Currency, Category FROM requests GROUP BY Currency, Category")
	handle(err)
	for rows.Next() {
		var (
			total    float64
			Currency string
			Category string
		)
		rows.Scan(&total, &Currency, &Category)
		swapper := swap.NewSwap()
		swapper.AddExchanger(ex.NewYahooApi(map[string]string{
			"userAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/84.0.4147.105 Safari/537.36"})).Build()
		if Currency != "USD" {
			convformat := Currency + "/USD"
			rate := swapper.Latest(convformat)
			m[Category] += total * rate.GetRateValue()
		} else {
			m[Category] += total
		}
	}
	return &pb.TopResponse{Toplist: m}, nil
}

func newServer() *server {
	db, err := sql.Open("mysql", "root:bapat2017@tcp(0.0.0.0:6603)/expensify_requests")
	handle(err)
	s := &server{db: db}
	return s
}

// Run the server
func main() {
	fmt.Println("hi")
	lis, err := net.Listen("tcp", ":2014")
	handle(err)
	s := grpc.NewServer()
	pb.RegisterServicesServer(s, newServer())
	s.Serve(lis)
}
