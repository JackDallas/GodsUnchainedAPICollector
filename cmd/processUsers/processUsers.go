package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/cloudflare/cloudflare-go"

	"github.com/JackDallas/Gods_Unchained_User_Lookup/internal/cfapi"
	"github.com/JackDallas/Gods_Unchained_User_Lookup/internal/utils"
	"github.com/JackDallas/Gods_Unchained_User_Lookup/pkg/guapi"
)

func main() {
	//Time exectution
	start := time.Now()
	fmt.Printf("Starting...\n")

	PropertiesEndpointProcessing()

	//Time execution
	elapsed := time.Since(start)
	fmt.Printf("Completed in %s\n", elapsed)
}

func PropertiesEndpointProcessing() {
	var props guapi.PropertiesResponse
	propsURL := guapi.PropertiesRequestURL(1, 1)
	err := utils.GetAndDecode(propsURL, &props)
	if err != nil {
		panic(err)
	}

	propsURL = guapi.PropertiesRequestURL(1, props.Total)
	err = utils.GetAndDecodeWithProgress(propsURL, &props)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Uploading %d records to CF USERNAMES to IDS\n", len(props.Records))
	utils.PrintMemUsage()
	UploadToUsernamesToIDS(props, &props)

	fmt.Printf("Uploading %d records to IDS TO USERNAMES\n", len(props.Records))
	utils.PrintMemUsage()
	UploadToIDSToUsernames(props)
}

func UploadToIDSToUsernames(result guapi.PropertiesResponse) {
	blankNames := make([]guapi.UserRecord, 0)
	fmt.Println("Uploading to IDS TO USERNAMES")
	usernameToID := make(map[int64]string)
	for _, user := range result.Records {
		if user.Username == "" {
			// fmt.Printf("Blank Username (%s) skipping...\n", user.Username)
			blankNames = append(blankNames, user)
			continue
		}
		if user.UserID < 0 {
			// fmt.Printf("Negative UserID %d skipping...\n", user.UserID)
			blankNames = append(blankNames, user)
			continue
		}
		if len(user.Username) == 0 {
			// fmt.Printf("Username has length 0 (%s) skipping...\n", user.Username)
			blankNames = append(blankNames, user)
			continue
		}
		if name, ok := usernameToID[user.UserID]; ok {
			fmt.Printf("UserID %d already exists as %s, new name would have been %s\n", user.UserID, name, user.Username)
		} else {
			usernameToID[user.UserID] = user.Username
		}
	}
	//Dump usernameToID to file
	fmt.Println("Dumping to file...")
	bytes, err := json.Marshal(usernameToID)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile("usernameToID.json", bytes, 0644)
	if err != nil {
		panic(err)
	}

	fmt.Printf("BlankNames: %d\n", len(blankNames))
	for _, user := range blankNames {
		fmt.Printf("%d\n", user.UserID)
	}
	api, err := cfapi.New()
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	ctx := context.Background()
	limiter := make(chan struct{}, 4)

	for k, v := range usernameToID {
		wg.Add(1)
		go handleKVUpload(&wg, limiter, strconv.FormatInt(k, 10), []byte(url.QueryEscape(v)), cfapi.GU_ID_TO_USERNAME, api, &ctx)
	}

	fmt.Println("All added waiting...")
	wg.Wait()
	fmt.Println("Process complete.")
}

func handleKVUpload(wg *sync.WaitGroup, sema chan struct{}, k string, v []byte, namespace string, api *cloudflare.API, ctx *context.Context) {
	sema <- struct{}{}
	defer func() {
		<-sema
		wg.Done()
	}()
	//
	time.Sleep(time.Duration(300 * time.Millisecond))
	fmt.Printf("Uploading %s\n", k)
	_, err := api.WriteWorkersKV(*ctx, namespace, k, v)
	if err != nil {
		fmt.Printf("Error writing %s: %s\n", k, err)
		panic(err)
	}
}

func UploadToUsernamesToIDS(result guapi.PropertiesResponse, props *guapi.PropertiesResponse) {
	records := make(map[string][]guapi.UserRecord)

	utils.PrintMemUsage()
	fmt.Println("Extracting user details from api results...")
	fmt.Printf("Processing %d of %d records\n", len(records), props.Total)
	for _, prop := range props.Records {
		if prop.UserID < 0 {
			continue
		}
		if prop.Username == "" {
			continue
		}
		if len(prop.Username) > 255 {
			continue
		}
		// if record already exists, append to the list
		records[prop.Username] = append(records[prop.Username], prop)
	}

	pairs := make([]cloudflare.WorkersKVPair, 0)
	fmt.Println("Building KV Pairs")
	for _, record := range records {
		value, err := json.Marshal(record)
		if err != nil {
			panic(err)
		}
		valueStr := string(value)

		//URL encode the username as the key
		key := url.QueryEscape(record[0].Username)

		// Add the record to the payload
		pairs = append(pairs, cloudflare.WorkersKVPair{
			Key:   key,
			Value: valueStr,
		})
	}

	fmt.Printf("Uploading %d records to CF USERNAMES to IDS\n", len(pairs))
	utils.PrintMemUsage()

	api, err := cfapi.New()
	if err != nil {
		panic(err)
	}

	cfapi.ThreadedKVBulkWrite(api, pairs, cfapi.GU_USERNAME_TO_ID)

	fmt.Println("Process complete.")
}
