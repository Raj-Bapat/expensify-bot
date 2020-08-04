package listeners

import (
	"context"
	"log"
	"time"
	// "database/sql"
	pb "expensify-bot/proto"
	"fmt"
	"github.com/shomali11/slacker"
	// "github.com/disiqueira/gocurrency"
	//idk
	_ "github.com/go-sql-driver/mysql"
	// "github.com/shopspring/decimal"
	"google.golang.org/grpc"
	// "google.golang.org/protobuf/proto"
	// "sort"
	// "net"
	// "strings"
	"sort"
)

func handle(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func ping(botCtx slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
	response.Reply("pong")
}

func test(botCtx slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
	response.Reply("It works!")
}

const (
	address = "localhost:2014"
)

var conn *grpc.ClientConn
var err error
var c pb.ServicesClient

type entry struct {
	val float64
	key string
}

type entries []entry

func (s entries) Len() int           { return len(s) }
func (s entries) Less(i, j int) bool { return s[i].val < s[j].val }
func (s entries) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func organizeAndSortByValue(m map[string]float64, amount int) string {
	var es entries
	for k, v := range m {
		es = append(es, entry{val: v, key: k})
	}
	sort.Sort(sort.Reverse(es))
	var s string
	for i, e := range es {
		if i < amount {
			s += fmt.Sprintf("%s $%.2f\n", e.key, e.val)
		}
	}
	return s
}

func top(botCtx slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) (amount int, time int64, units string, valid string) {
	amount = request.IntegerParam("amount", 1000000)
	time = int64(request.IntegerParam("time", 1000000))
	units = request.StringParam("units", "nul") //21
	bad := "Parameter(s) missing and/or invalid:"
	responsesize := len(bad)
	if amount == 1000000 {
		bad += " <amount>"
	}
	if time == 1000000 {
		bad += " <time>"
	}
	if units == "nul" {
		bad += " <units>"
	}
	if len(bad) > responsesize {
		valid = bad
	} else {
		valid = "ok"
	}
	return amount, time, units, valid
}

func topCategories(botCtx slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
	amount, timeAmount, units, valid := top(botCtx, request, response)
	ctx, cancel := context.WithTimeout(context.Background(), 1000000000*time.Second)
	defer cancel()
	if valid != "ok" {
		response.Reply(valid)
		return
	}
	m, err := c.TopCategory(ctx, &pb.TopRequest{
		Time:  timeAmount,
		Units: units})
	handle(err)
	response.Reply(organizeAndSortByValue(m.GetToplist(), amount))
}

func topEmployees(botCtx slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
	amount, timeAmount, units, valid := top(botCtx, request, response)
	ctx, cancel := context.WithTimeout(context.Background(), 1000000000*time.Second)
	defer cancel()
	if valid != "ok" {
		response.Reply(valid)
		return
	}
	m, err := c.TopEmployee(ctx, &pb.TopRequest{
		Time:  timeAmount,
		Units: units})
	handle(err)
	response.Reply(organizeAndSortByValue(m.GetToplist(), amount))
}

// have a gflag for the value, default is null - caller passes in the token
// main program (pipeline) with all flags and stuff
// different files not same package
// one single database with multiple tables
// schema
//	- check if the commands table exists - otherwise we can create it
// 	- have all the db calls as flags so others can change them easily
// gflags - at runtime, constant - at buildtime

//Run bot
func Run() {
	conn, err = grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c = pb.NewServicesClient(conn)
	bot := slacker.NewClient("xoxb-1262307775925-1292295054880-dt6mNK7v8ZVJGxG48rwwCdfp", slacker.WithDebug(true))
	definition1 := &slacker.CommandDefinition{
		Handler: ping,
	}
	definition2 := &slacker.CommandDefinition{
		Handler: test,
	}
	topCategoriesDefinition := &slacker.CommandDefinition{
		Description: "Gives an list of categories in sorted order decending by total spent based off of the parameters given.",
		Handler:     topCategories,
	}
	topEmployeesDefinition := &slacker.CommandDefinition{
		Description: "Gives an list of employees in sorted order decending by total spent based off of the parameters given.",
		Handler:     topEmployees,
	}
	bot.Command("ping", definition1)
	bot.Command("test", definition2)
	bot.Command("top categories <amount> <time> <units>", topCategoriesDefinition)
	bot.Command("top employees <amount> <time> <units>", topEmployeesDefinition)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := bot.Listen(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
