package main

import (
	// "context"
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
	// "container/heap"
	"net"
	"strconv"
	"sync"
	"time"
)

func handle(err error) {
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
}

type server struct {
	db *sql.DB
	mu sync.Mutex
	pb.UnimplementedServicesServer
}

// dropping the time
var currDate time.Time

func parseDate(formattedDate string) (t time.Time) {
	dt, err := time.Parse("1/2/2006 15:04", formattedDate)
	handle(err)
	return dt
}

func getOldDate(unit string, amnt int) (t time.Time) {
	if unit == "day" || unit == "days" {
		return currDate.AddDate(0, 0, -amnt)
	} else if unit == "week" || unit == "weeks" {
		return currDate.AddDate(0, 0, -amnt*7)
	} else if unit == "month" || unit == "months" {
		return currDate.AddDate(0, -amnt, 0)
	}
	return currDate.AddDate(-amnt, 0, 0)
}

func (s *server) TopCategory(treq *pb.TopRequest, stream pb.Services_TopCategoryServer) error {
	currDate = time.Now()
	checkDate := getOldDate(treq.GetUnits(), int(treq.GetTime()))
	rows, err := s.db.Query("SELECT SUM(Amount) as total, Currency, Category, ReportTimestamp FROM requests GROUP BY Currency, Category, ReportTimestamp")
	handle(err)
	m := make(map[string]float64)
	for rows.Next() {
		var (
			total     float64
			Currency  string
			Category  string
			Timestamp string
		)
		rows.Scan(&total, &Currency, &Category, &Timestamp)
		formattedTimestamp := parseDate(Timestamp)
		if formattedTimestamp.Before(checkDate) == true {
			continue
		}
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
	for k, v := range m {
		err := stream.Send(&pb.TopResponse{
			ID:     k,
			Amount: v})
		handle(err)
	}
	return nil
}

func (s *server) TopEmployee(treq *pb.TopRequest, stream pb.Services_TopEmployeeServer) error {
	currDate = time.Now()
	checkDate := getOldDate(treq.GetUnits(), int(treq.GetTime()))
	rows, err := s.db.Query("SELECT SUM(Amount) as total, Currency, UserID, ReportTimestamp FROM requests GROUP BY Currency, UserID, ReportTimestamp")
	handle(err)
	m := make(map[string]float64)
	for rows.Next() {
		var (
			total     float64
			Currency  string
			UserID    int64
			Timestamp string
		)
		rows.Scan(&total, &Currency, &UserID, &Timestamp)
		formattedTimestamp := parseDate(Timestamp)
		if formattedTimestamp.Before(checkDate) == true {
			continue
		}
		swapper := swap.NewSwap()
		swapper.AddExchanger(ex.NewYahooApi(map[string]string{
			"userAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/84.0.4147.105 Safari/537.36"})).Build()
		if Currency != "USD" {
			convformat := Currency + "/USD"
			rate := swapper.Latest(convformat)
			m[strconv.FormatInt(UserID, 10)] += total * rate.GetRateValue()
		} else {
			m[strconv.FormatInt(UserID, 10)] += total
		}
	}
	for k, v := range m {
		err := stream.Send(&pb.TopResponse{
			ID:     k,
			Amount: v})
		handle(err)
	}
	return nil
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
