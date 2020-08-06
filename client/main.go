package main

import (
	// "context"
	// "database/sql"
	"expensify-bot/listeners"
	// pb "expensify-bot/proto"
	"fmt"
	// "github.com/disiqueira/gocurrency"
	//idk
	// _ "github.com/go-sql-driver/mysql"
	// "github.com/shopspring/decimal"
	// "google.golang.org/grpc"
	// "google.golang.org/protobuf/proto"
	// "log"
	// "sort"
	// "net"
)

// Run the server
// how recent the non compliance was, how frequent it occurs, size of the non compliance, how they compare to other people - count/sum of non compliance expense
// control things as gflags - if we want to increase somethings
// cron job
// scheduled / constantly running
// expected run time

func main() {
	fmt.Println("here")
	listeners.Run()
}
