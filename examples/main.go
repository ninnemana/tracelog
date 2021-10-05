package main

import (
	"log"

	"github.com/ninnemana/tracelog"
	"go.uber.org/zap"
)

func main() {
	l, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}

	lg := tracelog.NewLogger(tracelog.WithLogger(l))

	lg.With()

	lg.Info("hello world")
}
