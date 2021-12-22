package threadeddownload

//https://medium.com/@dhanushgopinath/concurrent-http-downloads-using-go-32fecfa1ed27

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

func DownloadMultipleDataURLS(urls []string, sleepTime time.Duration) ([][]byte, []error) {
	done := make(chan []byte, len(urls))
	errch := make(chan error, len(urls))
	for _, URL := range urls {
		go func(URL string) {
			fmt.Printf("Downloading %s\n", URL)
			b, err := downloadDataAsByteArray(URL)
			if err != nil {
				// Retry once
				fmt.Printf("Retrying %s\n", URL)
				b, err = downloadDataAsByteArray(URL)
				if err != nil {
					errch <- err
					done <- nil
					return
				}
			}
			fmt.Printf("Downloaded %s\n", URL)
			done <- b
			errch <- nil
		}(URL)
		//Rate limiting prevention
		time.Sleep(sleepTime)
	}
	byteArrayArray := make([][]byte, 0)
	var errList []error
	for i := 0; i < len(urls); i++ {
		byteArrayArray = append(byteArrayArray, <-done)
		if err := <-errch; err != nil {
			errList = append(errList, err)
		}
	}
	return byteArrayArray, errList
}

func downloadDataAsByteArray(URL string) ([]byte, error) {
	response, err := http.Get(URL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, errors.New(response.Status)
	}
	var data []byte
	data, err = ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}
