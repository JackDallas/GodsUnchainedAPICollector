package cfapi

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/JackDallas/Gods_Unchained_User_Lookup/internal/utils"
	"github.com/cloudflare/cloudflare-go"
)

const (
	ERROR_NO_KEY_FOUND = "HTTP status 404: get: 'key not found' (10009)"
	GU_USER_MATCHES    = "d1c4cebc85324ca3a4b896a265c9e363"
	GU_META            = "4eb866bad30c42418959bae31c93daa2"
	GU_MATCHES         = "ad0a9672569f4321813d497998b1be45"
	GU_ID_TO_USERNAME  = "2fa7af2e064f4c5499008a46dd65921c"
	GU_USERNAME_TO_ID  = "8b93ddd7666d48d5bc610c49827611df"
)

func New() (*cloudflare.API, error) {
	api, err := cloudflare.New(os.Getenv("CLOUDFLARE_API_KEY"), os.Getenv("CLOUDFLARE_API_EMAIL"))
	if err != nil {
		return api, err
	}
	api.AccountID = os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	return api, err
}

func ThreadedKVBulkWrite(api *cloudflare.API, records []cloudflare.WorkersKVPair, namespaceID string) error {
	ctx := context.Background()
	threads := 0
	threadsTotal := 0
	//Create channel for threads to send to
	done := make(chan bool)

	payload := cloudflare.WorkersKVBulkWriteRequest{}
	fmt.Printf("Building Payloads\n")
	// Process records in chunks of 10,000
	for i, record := range records {
		if len(payload) == 10000 || i == len(records)-1 {
			fmt.Printf("Writing %d records, completed: %d records\n", len(payload), i-len(payload))
			utils.PrintMemUsage()
			threads++
			threadsTotal++
			go func(payload cloudflare.WorkersKVBulkWriteRequest, done chan bool) {
				resp, err := api.WriteWorkersKVBulk(context.Background(), namespaceID, payload)
				if err != nil {
					fmt.Printf("Error writing to API: %v\n", err)
					ChunkUploadPayload(payload, api, &ctx, len(payload)/4, namespaceID)
				} else {
					fmt.Printf("%+v\n", resp)
				}
				done <- true
			}(payload, done)
			if i != len(records)-1 {
				// Sleep if not on last call
				fmt.Println("Fired thread, cooling for 2 seconds...")
				time.Sleep(2 * time.Second)
				fmt.Println("Continuing")
			}
			payload = cloudflare.WorkersKVBulkWriteRequest{}
		}
		payload = append(payload, &record)
	}

	fmt.Printf("Waiting for API calls to complete\n")
	for {
		if threads == 0 {
			break
		}
		<-done
		threads--
		fmt.Printf("%d of %d API calls complete\n", threadsTotal-threads, threadsTotal)
	}

	return nil
}

func ChunkUploadPayload(payload cloudflare.WorkersKVBulkWriteRequest, api *cloudflare.API, ctx *context.Context, chunkSize int, namespace string) {
	fmt.Printf("Chunking request in to %d's", chunkSize)
	//Divide the payload up in to chunkSize's
	for i := 0; i < len(payload); i += chunkSize {
		if i+chunkSize > len(payload) {
			fmt.Println("Uploading final chunk")
			resp, err := api.WriteWorkersKVBulk(*ctx, namespace, payload[i:])
			if err != nil {
				fmt.Printf("Error writing chunk to API: %v\n", err)
				if chunkSize > 1 {
					ChunkUploadPayload(payload[i:], api, ctx, chunkSize/10, namespace)
				} else {
					fmt.Printf("Found invalid record, skipping\n")
					fmt.Printf("%+v\n", payload[len(payload)-1])
				}
			} else {
				fmt.Printf("%+v\n", resp)
			}
		} else {
			fmt.Printf("Uploading chunk %d of %d\n", i/chunkSize, len(payload)/chunkSize)
			resp, err := api.WriteWorkersKVBulk(*ctx, namespace, payload[i:i+chunkSize])
			if err != nil {
				fmt.Printf("Error writing chunk to API: %v\n", err)
				if chunkSize > 1 {
					ChunkUploadPayload(payload[i:i+chunkSize], api, ctx, chunkSize/10, namespace)
				} else {
					fmt.Printf("Found invalid record, skipping\n")
					fmt.Printf("%+v\n", payload[len(payload)-1])
				}
			} else {
				fmt.Printf("%+v\n", resp)
			}
		}
		time.Sleep(2 * time.Second)
	}
}
