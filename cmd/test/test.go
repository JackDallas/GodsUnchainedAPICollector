package main

import (
	"context"
	"fmt"

	"github.com/JackDallas/Gods_Unchained_User_Lookup/internal/cfapi"
)

func main() {
	api, err := cfapi.New()
	if err != nil {
		panic(err)
	}

	fmt.Println("Getting missing key")
	_, err = api.ReadWorkersKV(context.Background(), "4eb866bad30c42418959bae31c93daa2", "asdasd")
	if err != nil {
		if err.Error() == "HTTP status 404: get: 'key not found' (10009)" {
			fmt.Println("Key not found")
		} else {
			panic(err)
		}
	}
}
