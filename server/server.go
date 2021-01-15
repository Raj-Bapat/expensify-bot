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
	"bufio"
	"github.com/wcharczuk/go-chart"
	"io"
	"net"
	"os"
	"sort"
	"strconv"
	// "strings"
	"sync"
	"time"
)

// yesma

func handle(err error) {
	if err != nil {
		panic(err)
	}
}

type server struct {
	db *sql.DB
	mu sync.Mutex
	pb.UnimplementedServicesServer
}

// dropping the time
var currDate time.Time
var staticChecks map[string]float64

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

func (s *server) Summary(sreq *pb.SummaryRequest, stream pb.Services_SummaryServer) error {
	currDate = time.Now()
	checkDate := getOldDate(sreq.GetUnits(), int(sreq.GetTime()))
	m := make(map[string]float64)
	graph := chart.BarChart{
		Title: "Total in expenses per category",
		Background: chart.Style{
			Padding: chart.Box{
				Top: 40,
			},
		},
		Height:   512,
		BarWidth: 60,
		Bars:     []chart.Value{},
	}
	askSQL := fmt.Sprintf("select SUM(Amount) as Total, Currency, Category, TransactionTimestamp from requests where UserID=%q group by Currency, Category, TransactionTimestamp", sreq.GetID())
	rows, err := s.db.Query(askSQL)
	handle(err)
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
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Strings(keys)

	// To perform the opertion you want
	maxval := 0.0
	for _, k := range keys {
		graph.Bars = append(graph.Bars, chart.Value{Value: m[k], Label: k})
		if maxval < m[k] {
			maxval = m[k]
		}
	}
	graph.YAxis.Range = &chart.ContinuousRange{
		Min: 0,
		Max: maxval,
	}
	// for k, v := range m {
	// 	graph.Bars = append(graph.Bars, chart.Value{Value: v, Label: k})
	// }
	f, _ := os.Create("input.png")
	defer f.Close()
	err = graph.Render(chart.PNG, f)
	handle(err)
	f, _ = os.Open("input.png")
	reader := bufio.NewReader(f)
	buffer := make([]byte, 1024)
	for {
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal("cannot read chunk to buffer: ", err)
		}
		err = stream.Send(&pb.SummaryResponse{ChunkData: buffer[:n]})
		if err != nil {
			log.Fatal("cannot send chunk to server: ", err)
		}
	}
	return nil
}

func (s *server) make_table(name string) error {
	qry := fmt.Sprintf("SELECT count(*) FROM information_schema.TABLES WHERE (TABLE_SCHEMA = 'expensify_requests') AND (TABLE_NAME = '%s')", name)
	atqry := fmt.Sprintf("CREATE TABLE %s LIKE requests", name)
	rows, err := s.db.Query(qry)
	handle(err)
	var exists int64
	rows.Next()
	rows.Scan(&exists)
	if exists == 0 {
		stmt, err := s.db.Prepare(atqry)
		handle(err)
		_, err = stmt.Exec()
		handle(err)
	}
	return nil
}

func (s *server) initialise_tables() error {
	s.make_table("non_compliant")
	s.make_table("checked")
	rows, err := s.db.Query("select * from requests")
	handle(err)
	names, err := rows.Columns()
	handle(err)
	hasRow := false
	for _, val := range names {
		if "checked" == val {
			hasRow = true
		}
	}
	if !hasRow {
		s.db.Query("ALTER TABLE requests ADD COLUMN checked VARCHAR(255) Default 'no'")
	}
	return nil
}

func (s *server) ProcessNewRequests(inp *pb.UpdateConfirmation, stream pb.Services_ProcessNewRequestsServer) error {
	rows, err := s.db.Query("SELECT id, ReportID, ReportName, ReportTimestamp, UserID, AccountEmail, Merchant, Amount, TransactionTimestamp, TransactionID, Category, TransactionType, PolicyName, Comment, Currency, CategoryGlCode, Tag, RecieptUrl FROM requests where checked = 'no'")
	handle(err)
	for rows.Next() {
		var (
			id                   int64
			ReportID             int64
			ReportName           string
			ReportTimestamp      string
			UserID               int64
			AccountEmail         string
			Merchant             string
			Amount               float64
			TransactionTimestamp string
			TransactionID        string
			Category             string
			TransactionType      string
			PolicyName           string
			Comment              string
			Currency             string
			CategoryGlCode       int64
			Tag                  string
			RecieptUrl           string
		)
		rows.Scan(&id, &ReportID, &ReportName, &ReportTimestamp, &UserID, &AccountEmail, &Merchant, &Amount, &TransactionTimestamp, &TransactionID, &Category, &TransactionType, &PolicyName, &Comment, &Currency, &CategoryGlCode, &Tag, &RecieptUrl)

		swapper := swap.NewSwap()
		swapper.AddExchanger(ex.NewYahooApi(map[string]string{
			"userAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/84.0.4147.105 Safari/537.36"})).Build()
		var amnt float64
		if Currency != "USD" {
			convformat := Currency + "/USD"
			rate := swapper.Latest(convformat)
			amnt = Amount * rate.GetRateValue()
		} else {
			amnt = Amount
		}
		_, err := s.db.Query("UPDATE requests SET checked = 'yes' WHERE id = id")
		handle(err)
		if val, ok := staticChecks[Category]; ok {
			if amnt > val {
				err := stream.Send(&pb.NonCompliantRequest{
					Email:    AccountEmail,
					Category: Category,
					Amount:   Amount,
					Limit:    val,
					ReportID: ReportID,
				})
				handle(err)
				insertQuery := fmt.Sprintf("INSERT INTO non_compliant(id, ReportID, ReportName, ReportTimestamp, UserID, AccountEmail, Merchant, Amount, TransactionTimestamp, TransactionID, Category, TransactionType, PolicyName, Comment, Currency, CategoryGlCode, Tag, RecieptUrl) VALUES(%v, %v, %q, %q, %v, %q, %q, %f, %q, %q, %q, %q, %q, %q, %q, %v, %q, %q)", id, ReportID, ReportName, ReportTimestamp, UserID, AccountEmail, Merchant, Amount, TransactionTimestamp, TransactionID, Category, TransactionType, PolicyName, Comment, Currency, CategoryGlCode, Tag, RecieptUrl)
				_, err = s.db.Query(insertQuery)
				handle(err)
				continue
			}
		}
		insertQuery := fmt.Sprintf("INSERT INTO checked(id, ReportID, ReportName, ReportTimestamp, UserID, AccountEmail, Merchant, Amount, TransactionTimestamp, TransactionID, Category, TransactionType, PolicyName, Comment, Currency, CategoryGlCode, Tag, RecieptUrl) VALUES(%v, %v, %q, %q, %v, %q, %q, %f, %q, %q, %q, %q, %q, %q, %q, %v, %q, %q)", id, ReportID, ReportName, ReportTimestamp, UserID, AccountEmail, Merchant, Amount, TransactionTimestamp, TransactionID, Category, TransactionType, PolicyName, Comment, Currency, CategoryGlCode, Tag, RecieptUrl)
		fmt.Println(insertQuery)
		_, err = s.db.Query(insertQuery)
		handle(err)
	}
	return nil
}

func newServer() *server {
	db, err := sql.Open("mysql", "root:bapat2017@tcp(0.0.0.0:6603)/expensify_requests")
	handle(err)
	s := &server{db: db}
	s.initialise_tables()
	// go s.processNewRequests()
	return s
}

func addChecks(k string, v float64) {
	staticChecks[k] = v
}

// Run the server

func main() {
	fmt.Println("hi")
	staticChecks = make(map[string]float64)
	addChecks("Transportation", 2000.0)
	addChecks("Lodging", 300.0)
	addChecks("Employee Meals", 100.0)
	lis, err := net.Listen("tcp", ":2014")
	handle(err)
	s := grpc.NewServer()
	pb.RegisterServicesServer(s, newServer())
	s.Serve(lis)
}
