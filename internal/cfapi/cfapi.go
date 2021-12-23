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

type CFAPI struct {
	Api *cloudflare.API
	// Ticker that takes 1 off Requests 1200 times over 5 minutes
	Ticker *time.Ticker
	// Request freed channel
	RequestFreed chan bool
}

func New() (*CFAPI, error) {
	api, err := cloudflare.New(os.Getenv("CLOUDFLARE_API_KEY"), os.Getenv("CLOUDFLARE_API_EMAIL"))
	if err != nil {
		return nil, err
	}
	api.AccountID = os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	cfapiw := &CFAPI{api, time.NewTicker((time.Minute * 5 / 1200)), make(chan bool, 1200)}
	// Queue initial requests, slightly less for sanity's sake
	for i := 0; i < 1100; i++ {
		cfapiw.RequestFreed <- true
	}

	go cfapiw.rateLimit()
	return cfapiw, nil
}

func (cfapiw *CFAPI) rateLimit() {
	// Wait for the ticker to tick
	for range cfapiw.Ticker.C {
		// Add free request to channel
		cfapiw.RequestFreed <- true
	}
}

func (cfapi *CFAPI) KVBulkWrite(records []cloudflare.WorkersKVPair, namespaceID string) error {
	// get freed request from channel
	<-cfapi.RequestFreed

	// Create a context
	ctx := context.Background()

	payload := cloudflare.WorkersKVBulkWriteRequest{}
	fmt.Printf("Building Payloads\n")
	// Process records in chunks of 10,000
	for i, record := range records {
		if len(payload) == 10000 || i == len(records)-1 {
			fmt.Printf("Writing %d records, completed: %d records\n", len(payload), i-len(payload))
			utils.PrintMemUsage()
			resp, err := cfapi.Api.WriteWorkersKVBulk(context.Background(), namespaceID, payload)
			if err != nil {
				fmt.Printf("Error writing to API: %v\n", err)
				ChunkUploadPayload(payload, cfapi.Api, &ctx, len(payload)/4, namespaceID)
			} else {
				fmt.Printf("%+v\n", resp)
			}
			payload = cloudflare.WorkersKVBulkWriteRequest{}
		}
		payload = append(payload, &record)
	}
	return nil
}

func (cf *CFAPI) WriteWorkersKV(ctx *context.Context, namespace, key string, value []byte) error {
	// get freed request from channel
	<-cf.RequestFreed

	cf.Api.WriteWorkersKV(*ctx, namespace, key, value)
	return nil
}
func (cfapi *CFAPI) KVSingleWrite(record cloudflare.WorkersKVPair, namespaceID string) error {
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
