package guapi

import (
	"fmt"
)

const (
	GU_API_ADDRESS         = "https://api.godsunchained.com/v0"
	GU_PROPERTIES_ENDPOINT = "/properties?page=%d&perPage=%d"
	GU_MATCH_ENDPOINT      = "/match?page=%d&perPage=%d&order=%s&end_time=%d-%d"
)

func PropertiesRequestURL(page int, perPage int) string {
	return fmt.Sprintf(GU_API_ADDRESS+GU_PROPERTIES_ENDPOINT, page, perPage)
}

func MatchRequestURL(page int, perPage int, startTime int64, endTime int64) string {
	return fmt.Sprintf(GU_API_ADDRESS+GU_MATCH_ENDPOINT, page, perPage, OrderDesc, startTime, endTime)
}
