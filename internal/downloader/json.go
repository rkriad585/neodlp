package downloader

import "encoding/json"

func parseJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
