package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/JackDallas/Gods_Unchained_User_Lookup/internal/cfapi"
	"github.com/JackDallas/Gods_Unchained_User_Lookup/internal/utils"
	"github.com/JackDallas/Gods_Unchained_User_Lookup/pkg/guapi"
	"github.com/JackDallas/Gods_Unchained_User_Lookup/pkg/threadeddownload"
	"github.com/cloudflare/cloudflare-go"
)

func main() {
	//Login to cloudflare
	api, err := cfapi.New()
	if err != nil {
		panic(err)
	}
	kvRes, err := api.ReadWorkersKV(context.Background(), cfapi.GU_META, "matcheslastupdated")
	if err != nil {
		panic(err)
	}
	lastUpdateEndTimeStartInt, err := strconv.Atoi(string(kvRes))
	if err != nil {
		panic(err)
	}

	lastUpdateTimeStart := int64(lastUpdateEndTimeStartInt)
	lastUpdateTimeEnd := time.Now().Unix()

	var matchRecords []guapi.MatchRecord

	fmt.Println("Downloading data from api...")

	perPage := 10000
	url := guapi.MatchRequestURL(1, perPage, lastUpdateTimeStart, lastUpdateTimeEnd)
	var res guapi.MatchResponse
	err = utils.GetAndDecode(url, &res)
	if err != nil {
		panic(err)
	}

	matchRecords = res.Records

	if res.Total/perPage > 1 {
		//Paginate
		fmt.Println("Paginating")
		fmt.Println("Building url list")
		urlList := make([]string, 0)
		for i := 2; i <= (res.Total/perPage)+1; i++ {
			urlList = append(urlList, guapi.MatchRequestURL(i, perPage, lastUpdateTimeStart, lastUpdateTimeEnd))
		}

		fmt.Println("Starting Threaded Download...")
		utils.PrintMemUsage()

		results, errList := threadeddownload.DownloadMultipleDataURLS(urlList, 500*time.Millisecond)

		//If all downloads return an error
		if len(errList) >= len(urlList) {
			for _, err := range errList {
				fmt.Printf("%s\n", err)
			}
			panic("Errors occurred")
		} else if len(errList) > 0 {
			fmt.Printf("%d errors occurred\n", len(errList))
			for _, err := range errList {
				fmt.Printf("%s\n", err)
			}
		}

		fmt.Printf("Download complete, parsing %d pages\n", len(results))

		for _, result := range results {
			//decode result json to res
			var res guapi.MatchResponse
			json.Unmarshal(result, &res)

			matchRecords = append(matchRecords, res.Records...)
			utils.PrintMemUsage()
		}

		fmt.Printf("Parsing complete, got %d records\n", len(matchRecords))
	}

	gu_matches := make(map[string]guapi.MatchRecord)
	gu_matches_by_user_id := map[int][]string{}

	fmt.Println("Building maps")
	for _, match := range matchRecords {
		gu_matches[match.GameID] = match
		for _, player := range match.PlayerInfo {
			gu_matches_by_user_id[player.UserID] = append(gu_matches_by_user_id[player.UserID], match.GameID)
		}
	}
	utils.PrintMemUsage()

	fmt.Println("Building kv pairs")
	gu_matches_kv := []cloudflare.WorkersKVPair{}
	for GameID, match := range gu_matches {
		matchJson, err := json.Marshal(match)
		if err != nil {
			panic(err)
		}

		matchKV := cloudflare.WorkersKVPair{
			Key:   GameID,
			Value: string(matchJson),
		}

		gu_matches_kv = append(gu_matches_kv, matchKV)
	}

	//Write GU_MATCHES to cf kv
	fmt.Println("Bulk writing GU_MATCHES to kv")
	cfapi.ThreadedKVBulkWrite(api, gu_matches_kv, cfapi.GU_MATCHES)

	fmt.Println("Completed GU_MATCHES bulk write")
	utils.PrintMemUsage()
	fmt.Printf("Iterating over GU_MATCHES_BY_USER_ID \n")
	totalUserRecords := len(gu_matches_by_user_id)
	currentUserRecord := 0

	ctx := context.Background()
	wg := new(sync.WaitGroup)

	for k, v := range gu_matches_by_user_id {
		wg.Add(1)
		go func(k int, v []string, ctx *context.Context, wg *sync.WaitGroup) {
			//Check for existing record on cloudflare kv
			res, err := api.ReadWorkersKV(*ctx, cfapi.GU_USER_MATCHES, strconv.FormatInt(int64(k), 10))
			if err == nil {
				//Record exists
				var currentRecord []string
				err = json.Unmarshal(res, &currentRecord)
				if err != nil {
					panic(err)
				}
				currentRecord = append(currentRecord, v...)
				//Write to cloudflare
				//json encode currentRecord
				currentRecordJson, err := json.Marshal(currentRecord)
				if err != nil {
					panic(err)
				}
				api.WriteWorkersKV(*ctx, cfapi.GU_USER_MATCHES, fmt.Sprintf("%d", k), currentRecordJson)
			} else {
				if err.Error() == cfapi.ERROR_NO_KEY_FOUND {
					//Record does not exist, create it
					valueJSON, err := json.Marshal(v)
					if err != nil {
						panic(err)
					}
					api.WriteWorkersKV(*ctx, cfapi.GU_USER_MATCHES, fmt.Sprintf("%d", k), valueJSON)
				} else {
					//Something fucked happened
					panic(err)
				}
			}
			wg.Done()
		}(k, v, &ctx, wg)
		currentUserRecord++
	}

	utils.PrintMemUsage()
	fmt.Println("Setting matcheslastupdated to current time")

	//Set last updated time
	api.WriteWorkersKV(context.Background(), cfapi.GU_META, "matcheslastupdated", []byte(fmt.Sprintf("%d", lastUpdateTimeEnd)))
	fmt.Printf("Completed GU_MATCHES_BY_USER_ID bulk write\n")
}
