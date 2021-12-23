package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/cloudflare/cloudflare-go"

	"github.com/JackDallas/Gods_Unchained_User_Lookup/internal/cfapi"
	"github.com/JackDallas/Gods_Unchained_User_Lookup/internal/utils"
	"github.com/JackDallas/Gods_Unchained_User_Lookup/pkg/guapi"
)

func ProcessUsers(cfapiw *cfapi.CFAPI) {
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
	UploadToUsernamesToIDS(props, &props, cfapiw)

	fmt.Printf("Uploading %d records to IDS TO USERNAMES\n", len(props.Records))
	utils.PrintMemUsage()
	UploadToIDToUsernames(props, cfapiw)
}

func UploadToIDToUsernames(result guapi.PropertiesResponse, cfapiw *cfapi.CFAPI) {
	fmt.Println("Processing to ID TO USERNAMES")

	blankNames := make([]guapi.UserRecord, 0)
	IDToUsernamePairs := []cloudflare.WorkersKVPair{}

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
		IDToUsernamePairs = append(IDToUsernamePairs, cloudflare.WorkersKVPair{
			Key:   strconv.FormatInt(user.UserID, 10),
			Value: user.Username,
		})
	}

	fmt.Printf("BlankNames: %d\n", len(blankNames))
	for _, user := range blankNames {
		fmt.Printf("%d\n", user.UserID)
	}

	fmt.Println("Uploading to ID TO USERNAMES")

	cfapiw.KVBulkWrite(IDToUsernamePairs, cfapi.GU_ID_TO_USERNAME)
}

func UploadToUsernamesToIDS(result guapi.PropertiesResponse, props *guapi.PropertiesResponse, cfapiw *cfapi.CFAPI) {
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

	cfapiw.KVBulkWrite(pairs, cfapi.GU_USERNAME_TO_ID)

	fmt.Println("Process complete.")
}
