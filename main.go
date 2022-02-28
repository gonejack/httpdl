package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/gonejack/httpdl/httpdl"
)

func init() {
	log.SetOutput(os.Stdout)
}

func main() {
	ctx, _ := signal.NotifyContext(context.TODO(), os.Interrupt)
	cmd := httpdl.HTTPDl{
		Options: httpdl.MustParseOptions(),
	}
	err := cmd.Run(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
