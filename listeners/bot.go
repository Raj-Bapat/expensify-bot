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
	//idk
	_ "github.com/go-sql-driver/mysql"
	// "github.com/shopspring/decimal"
	"google.golang.org/grpc"
	// "google.golang.org/protobuf/proto"
	// "sort"
	// "net"
	// "strings"
	"bytes"
	// "github.com/slack-go/slack"
	// "encoding/json"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
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

// Token for the slack client
var Token = "xoxb-1262307775925-1292295054880-dt6mNK7v8ZVJGxG48rwwCdfp"

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
	stream, err := c.TopCategory(ctx, &pb.TopRequest{
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
	stream.CloseSend()
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
	stream.CloseSend()
	response.Reply(res)
}

type jsonImagePostRequest struct {
	BotToken   string `json:"token"`
	ImgChannel string `json:"channels"`
	FileName   string `json:"filename"`
	FileType   string `json:"filetype"`
}

func summary(botCtx slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
	id := request.StringParam("ID", "100")
	timeAmount := request.IntegerParam("time", 1)
	units := request.StringParam("units", "week")
	ctx, cancel := context.WithTimeout(context.Background(), 1000000000*time.Second)
	defer cancel()
	stream, err := c.Summary(ctx, &pb.SummaryRequest{
		ID:    id,
		Time:  int64(timeAmount),
		Units: units})
	handle(err)
	imageData := bytes.Buffer{}
	for {
		log.Print("waiting to receive more data")
		req, err := stream.Recv()
		if err == io.EOF {
			log.Print("no more data")
			break
		}
		handle(err)
		chunk := req.GetChunkData()
		_, err = imageData.Write(chunk)
		if err != nil {
			log.Printf("cannot write chunk data: %v", err)
			log.Fatal(err)
		}
	}
	stream.CloseSend()
	file, err := os.Create("output.png")
	handle(err)
	imageData.WriteTo(file)
	file, err = os.Open("output.png")
	handle(err)
	channel := botCtx.Event().Channel
	HttpClient := &http.Client{}
	// req, err := http.NewRequest("POST", "https://slack.com/api/files.upload", &imageData)
	// req.Header.Set("Content-Type", "multipart/form-data")
	// req.ParseMultipartForm(5000000)
	// resp, err := HttpClient.Do(req)
	// handle(err)
	// defer resp.Body.Close()
	// jsonDataFromHttp, err := ioutil.ReadAll(resp.Body)
	// payload := &jsonImagePostRequest{
	// 	BotToken:   Token,
	// 	ImgChannel: channel,
	// 	FileName:   "output.png",
	// 	FileType:   "png"}
	// jsonpayload, err := json.Marshal(payload)
	// handle(err)
	// u := bytes.NewReader(jsonpayload)
	// req, err := http.NewRequest("POST", "https://slack.com/api/files.upload", u)
	// handle(err)
	// HttpClient := &http.Client{}
	// req.Header.Set("Content-Type", "multipart/form-data")
	// resp, err := HttpClient.Do(req)
	// handle(err)
	// defer resp.Body.Close()
	// jsonDataFromHttp, err := ioutil.ReadAll(resp.Body)
	// err = json.Unmarshal([]byte(jsonDataFromHttp), &jsonData) // here!
	// handle(err)
	// fmt.Println(string(jsonDataFromHttp))
	// handle(err)
	// var url string = "https://slack.com/api/files.upload?token=" + Token + "&file=" + "output.png" + "&channels=[" + Channel + "]"
	// attachments := []slack.Block{}
	// attachments = append(attachments, slack.NewContextBlock("1", slack.NewImageBlockElement(url, "Not working :(")))

	// response.Reply("hi", slacker.WithBlocks(attachments))
	params := map[string]string{
		"token":    Token,
		"channels": channel,
	}
	filecontents, err := ioutil.ReadAll(file)
	handle(err)
	fi, err := file.Stat()
	handle(err)
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", fi.Name())
	part.Write(filecontents)
	handle(err)
	for key, val := range params {
		_ = writer.WriteField(key, val)
	}
	err = writer.Close()
	handle(err)
	req, err := http.NewRequest("POST", "https://slack.com/api/files.upload", body)
	handle(err)
	req.Header.Add("Content-Type", writer.FormDataContentType())
	resp, err := HttpClient.Do(req)
	handle(err)
	defer resp.Body.Close()
	jsonDataFromHttp, err := ioutil.ReadAll(resp.Body)
	handle(err)
	fmt.Println(string(jsonDataFromHttp))
}

// have a gflag for the value, default is null - caller passes in the token
// main program (pipeline) with all flags and stuff
// different files not same package
// one single database with multiple tables
// schema
//	- check if the commands table exists - otherwise we can create it
// 	- have all the db calls as flags so others can change them easily
// gflags - at runtime, constant - at buildtime

// 2 tables, one for submitted and approved

//Run bot
func Run() {
	conn, err = grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c = pb.NewServicesClient(conn)
	bot := slacker.NewClient(Token, slacker.WithDebug(true))
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
	summaryDefinition := &slacker.CommandDefinition{
		Description: "Gives a bar graph showing how much money a particular employee spent on each category",
		Handler:     summary,
	}

	bot.Command("ping", definition1)
	bot.Command("test", definition2)
	bot.Command("top categories <amount> <time> <units>", topCategoriesDefinition)
	bot.Command("top employees <amount> <time> <units>", topEmployeesDefinition)
	bot.Command("summary <ID> <time> <units>", summaryDefinition)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := bot.Listen(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
