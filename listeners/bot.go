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
	"encoding/json"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
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

// this method is just a checker to make sure that the input is valid

func top(botCtx slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) (amount int, time int64, units string, valid string) {
	// getting input from user through slack
	amount = request.IntegerParam("amount", 1000000)
	time = int64(request.IntegerParam("time", 1000000))
	units = request.StringParam("units", "nul") //21
	// checks to make sure that the input is valid
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
	// making sure input is valid and putting it into variables
	amount, timeAmount, units, valid := top(botCtx, request, response)
	// setting ctx
	ctx, cancel := context.WithTimeout(context.Background(), 1000000000*time.Second)
	defer cancel()
	if valid != "ok" {
		// confirm that it is indeed valid
		response.Reply(valid)
		return
	}
	// request a map from the server pertaining to the categories in which the most has been spent in
	stream, err := c.TopCategory(ctx, &pb.TopRequest{
		Time:  timeAmount,
		Units: units})
	handle(err)
	set := treeset.NewWith(byPair)
	for {
		// getting the next pair of values
		person, err := stream.Recv()
		if err == io.EOF {
			break
		}
		handle(err)
		// adding them to the set of categories
		set.Add(Pair{person.GetID(), person.GetAmount()})
		if set.Size() > amount {
			it := set.Iterator()
			it.Last()
			set.Remove(Pair{it.Value().(Pair).first, it.Value().(Pair).second})
		}
	}
	// pinting all the categories in the set
	var res string
	for _, val := range set.Values() {
		res += fmt.Sprintf("%q: %.2f\n", val.(Pair).first, val.(Pair).second)
	}
	stream.CloseSend()
	response.Reply(res)
}

func topEmployees(botCtx slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
	// verification and making a variable for top related requests
	amount, timeAmount, units, valid := top(botCtx, request, response)
	ctx, cancel := context.WithTimeout(context.Background(), 1000000000*time.Second)
	defer cancel()
	if valid != "ok" {
		response.Reply(valid)
		return
	}
	// starting the stream
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
		// adding the person and the cost into a set (to avoid duplicates)
		set.Add(Pair{person.GetID(), person.GetAmount()})
		if set.Size() > amount { // cutting off the the category with the least money spent currently to ensure that we are showing the right amount of entires
			it := set.Iterator()
			it.Last()
			set.Remove(Pair{it.Value().(Pair).first, it.Value().(Pair).second})
		}
	}
	// printing the values
	var res string
	for _, val := range set.Values() {
		res += fmt.Sprintf("%q: %.2f\n", val.(Pair).first, val.(Pair).second)
	}
	stream.CloseSend()
	response.Reply(res)
}

// post request object
type jsonImagePostRequest struct {
	BotToken   string `json:"token"`
	ImgChannel string `json:"channels"`
	FileName   string `json:"filename"`
	FileType   string `json:"filetype"`
}

func summary(botCtx slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
	// reading input from slack
	id := request.StringParam("ID", "100")
	timeAmount := request.IntegerParam("time", 1)
	units := request.StringParam("units", "week")
	ctx, cancel := context.WithTimeout(context.Background(), 1000000000*time.Second)
	defer cancel()
	// starting the stream for the summary image
	stream, err := c.Summary(ctx, &pb.SummaryRequest{
		ID:    id,
		Time:  int64(timeAmount),
		Units: units})
	handle(err)
	// creating a new image object
	imageData := bytes.Buffer{}
	for {
		log.Print("waiting to receive more data")
		req, err := stream.Recv()
		if err == io.EOF {
			log.Print("no more data")
			break
		}
		handle(err)
		// grab the next byte chunk in the image and write it to the image oject
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
	imageData.WriteTo(file) // write the image to a file and store it in the virtual os
	file, err = os.Open("output.png")
	handle(err)
	channel := botCtx.Event().Channel
	HttpClient := &http.Client{}
	// json payload for http request to slack
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
	// write the contents of params to the payload
	for key, val := range params {
		_ = writer.WriteField(key, val)
	}
	err = writer.Close()
	handle(err)
	req, err := http.NewRequest("POST", "https://slack.com/api/files.upload", body) // send the payload
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

func update(botCtx slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
	ctx, cancel := context.WithTimeout(context.Background(), 1000000000*time.Second)
	defer cancel()
	// check whether updates are needed
	stream, err := c.ProcessNewRequests(ctx, &pb.UpdateConfirmation{
		Confirmed: "update",
	})
	handle(err)
	for {
		// grab the entry in the table
		notif, err := stream.Recv()
		if err == io.EOF {
			break
		}
		handle(err)
		user, err := botCtx.Client().GetUserByEmail(notif.GetEmail())
		handle(err)
		userid := user.ID
		// type st1 struct {
		// 	id string
		// }
		// type jsonResponse struct {
		// 	ok      string
		// 	channel st1
		// }
		// put them as a map
		var result map[string]interface{}
		ireq := url.Values{
			"token": {Token},
			"users": {userid},
		}
		// this section is for posting an http request that opens a private message channel with the bot if there is none, and pm's the user that thier expense request has been denied
		resp, err := http.PostForm("https://slack.com/api/conversations.open", ireq)
		defer resp.Body.Close()
		jsonDataFromHttp, err := ioutil.ReadAll(resp.Body)
		fmt.Println(string(jsonDataFromHttp))
		handle(err)
		json.Unmarshal(jsonDataFromHttp, &result)
		msg := fmt.Sprintf("expense report %d has been denied due to a request of category %s. The amount to be expensed exceeded the maximum amount, %.2f$, as seen in our policy. Please edit or remove your request.", notif.GetReportID(), notif.GetCategory(), notif.GetLimit())
		req := url.Values{}
		req.Set("token", Token)
		tmp := result["channel"].(map[string]interface{})
		req.Set("channel", tmp["id"].(string))
		req.Set("text", msg)
		hreq, err := http.NewRequest("POST", "https://slack.com/api/chat.postMessage", bytes.NewBufferString(req.Encode()))
		handle(err)
		hreq.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
		// resp, err = http.PostForm("https://slack.com/api/chat.postMessage", req)
		client := &http.Client{}
		resp, err = client.Do(hreq)
		handle(err)
		defer resp.Body.Close()
		jsonDataFromHttp, err = ioutil.ReadAll(resp.Body)
		handle(err)
		fmt.Println(string(jsonDataFromHttp))
	}
}

//Run bot
func Run() {
	// connect with the grpc client
	conn, err = grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	// create a new slack client
	c = pb.NewServicesClient(conn)
	bot := slacker.NewClient(Token, slacker.WithDebug(true))
	definition1 := &slacker.CommandDefinition{
		Handler: ping,
	}
	definition2 := &slacker.CommandDefinition{
		Handler: test,
	}
	// definitions for different commands
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
	updateDefinition := &slacker.CommandDefinition{
		Description: "update the expense request tables",
		Handler:     update,
	}
	// these are commands + the structure needed to invoke them
	bot.Command("ping", definition1)
	bot.Command("test", definition2)
	bot.Command("top categories <amount> <time> <units>", topCategoriesDefinition)
	bot.Command("top employees <amount> <time> <units>", topEmployeesDefinition)
	bot.Command("summary <ID> <time> <units>", summaryDefinition)
	bot.Command("update", updateDefinition)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := bot.Listen(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
