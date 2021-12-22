package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"

	"github.com/JackDallas/Gods_Unchained_User_Lookup/internal/progressdownload"
)

func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func GetAndDecode(url string, target interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(target)
	if err != nil {
		return err
	}

	return nil
}

func GetAndDecodeWithProgress(url string, target interface{}) error {

	buf, err := progressdownload.DownloadFile(url)
	if err != nil {
		return err
	}

	//json decode buf to target
	err = json.Unmarshal(buf, target)
	if err != nil {
		return err
	}

	return nil
}
