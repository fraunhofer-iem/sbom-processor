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
		fmt.Printf("Request to %s failed with %s\n", url, err.Error())
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests {
			retry := resp.Header.Get("Retry-After")
			fmt.Printf("Failed due to too many requests. Retry-After %s\n", retry)
		}
		err := fmt.Errorf("request failed with status code %d", resp.StatusCode)
		fmt.Printf("Request to %s failed with %s\n", url, err.Error())
		return nil, err
	}

	var mvnRes MvnSearchResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&mvnRes); err != nil {
		fmt.Printf("Decoding of response to %s failed with %s\n", url, err.Error())
		return nil, err
	}

	return &mvnRes, nil
}
