package main

import (
	"math/rand"
	"time"

	"github.com/hashicorp/enos/internal/command/enos/cmd"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	cmd.Execute()
}
