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
	"github.com/emirpasic/gods/sets/treeset"
	_ "github.com/go-sql-driver/mysql"
	// "github.com/shopspring/decimal"
	"google.golang.org/grpc"
	// "google.golang.org/protobuf/proto"
	// "sort"
	// "net"
	// "strings"
	"io"
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

//Pair implements a string float pair
type Pair struct {
	first  string
	second float64
}

func byPair(a, b interface{}) int {

	// Type assertion, program will panic if this is not respected
	c1 := a.(Pair)
	c2 := b.(Pair)

	switch {
	case c1.second < c2.second:
		return 1
	case c1.second > c2.second:
		return -1
	default:
		return 0
	}
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
	stream, err := c.TopEmployee(ctx, &pb.TopRequest{
		Time:  timeAmount,
		Units: units})
	handle(err)
	set := treeset.NewWith(byPair)
	for {
		person, err := stream.Recv()
		if err == io.EOF {
			break
		}
		handle(err)
		set.Add(Pair{person.GetID(), person.GetAmount()})
		if set.Size() > amount {
			it := set.Iterator()
			it.Last()
			set.Remove(Pair{it.Value().(Pair).first, it.Value().(Pair).second})
		}
	}
	var res string
	for _, val := range set.Values() {
		res += fmt.Sprintf("%q: %.2f\n", val.(Pair).first, val.(Pair).second)
	}
	response.Reply(res)
}

func topEmployees(botCtx slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
	amount, timeAmount, units, valid := top(botCtx, request, response)
	ctx, cancel := context.WithTimeout(context.Background(), 1000000000*time.Second)
	defer cancel()
	if valid != "ok" {
		response.Reply(valid)
		return
	}
	stream, err := c.TopEmployee(ctx, &pb.TopRequest{
		Time:  timeAmount,
		Units: units})
	handle(err)
	set := treeset.NewWith(byPair)
	for {
		person, err := stream.Recv()
		if err == io.EOF {
			break
		}
		handle(err)
		set.Add(Pair{person.GetID(), person.GetAmount()})
		if set.Size() > amount {
			it := set.Iterator()
			it.Last()
			set.Remove(Pair{it.Value().(Pair).first, it.Value().(Pair).second})
		}
	}
	var res string
	for _, val := range set.Values() {
		res += fmt.Sprintf("%q: %.2f\n", val.(Pair).first, val.(Pair).second)
	}
	response.Reply(res)
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

// 2 tables, one for submitted and approved

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
		Description: "Gives a list of categories in sorted order decending by total spent based off of the parameters given.",
		Handler:     topCategories,
	}
	topEmployeesDefinition := &slacker.CommandDefinition{
		Description: "Gives a list of employees in sorted order decending by total spent based off of the parameters given.",
		Handler:     topEmployees,
	}
	// summaryDefinition := &slacker.CommandDefinition{
	// 	Description: "Gives a bar graph showing how much money a particular employee spent on each category"
	// }

	bot.Command("ping", definition1)
	bot.Command("test", definition2)
	bot.Command("top categories <amount> <time> <units>", topCategoriesDefinition)
	bot.Command("top employees <amount> <time> <units>", topEmployeesDefinition)
	// bot.Command("histdata <amount> <time> <units>", )
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := bot.Listen(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
