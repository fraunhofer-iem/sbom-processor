package mvn

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func queryApi(cName string) (*MvnSearchResponse, error) {
	encodedName := url.QueryEscape(cName)

	url := fmt.Sprintf("https://search.maven.org/solrsearch/select?q=a:%s&rows=20&wt=json", encodedName)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Request failed with %s\n", err.Error())
		return nil, err
	}

	defer resp.Body.Close()

	var mvnRes MvnSearchResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&mvnRes); err != nil {
		fmt.Printf("Decoding of response failed with %s\n", err.Error())
		return nil, err
	}

	return &mvnRes, nil
}
