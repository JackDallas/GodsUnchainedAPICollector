package main

import (
	"fmt"
	"time"

	"github.com/JackDallas/Gods_Unchained_User_Lookup/internal/cfapi"
)

func main() {
	//Time exectution
	start := time.Now()
	fmt.Printf("Starting...\n")

	//Login to cloudflare
	api, err := cfapi.New()
	if err != nil {
		panic(err)
	}

	ProcessUsers(api)
	ProcessMatches(api)
	//Time execution
	elapsed := time.Since(start)
	fmt.Printf("Completed in %s\n", elapsed)
}
