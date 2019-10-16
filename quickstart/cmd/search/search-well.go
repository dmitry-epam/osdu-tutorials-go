/*
This is an example application to demonstrate querying the Search API.
*/
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"log"
	"net/http"
)

const API_BASE_URL = "<your API base url here>" // https://osdu-demo-dev.azure-api.net

// extracts files and srns for each resource type from response
// body and strips out everything else
func getFilesFromResults(responseBody []byte) map[string][]interface{} {

	type FileStruct struct {
		Filename string `json:"filename"`
		Srn      string `json:"srn"`
	}

	// create a map to hold parsed files and srns
	SRNs := map[string][]interface{}{}

	// iterate over an array of search results
	result := gjson.Get(string(responseBody), "results")
	result.ForEach(func(key, value gjson.Result) bool {
		// each resource type will contain one or more file structs
		mapKey := gjson.Get(value.String(), "resource_type").String()
		mapValues := gjson.Get(value.String(), "files").Array()
		// iterate over an array of files to extract filename and srn
		for _, v := range mapValues {
			var fileStruct FileStruct
			fileStruct.Filename = v.Get("filename").String()
			fileStruct.Srn = v.Get("srn").String()
			// add new file with its srn to resource type
			SRNs[mapKey] = append(SRNs[mapKey], fileStruct)
			log.Printf("Adding value: %s\n", fileStruct)
		}
		return true // keep iterating
	})

	return SRNs
}

func main() {

	//metadata struct
	type Metadata struct {
		ResourceType []string `json:"resource_type"`
	}
	// search request struct
	type SearchRequest struct {
		FullText string   `json:"fulltext"`
		Metadata Metadata `json:"metadata"`
		Facets   []string `json:"facets"`
	}

	// construct an initial well search request
	wellReq := SearchRequest{
		FullText: "*",
		Metadata: Metadata{ResourceType: []string{"master-data/Well", "work-product-component/WellLog", "work-product-component/WellborePath"}},
		Facets:   []string{"resource_type"},
	}

	log.Printf("Initialized well request: \n%s", wellReq)

	// find handler takes "wellname" as input parameter and makes Search API call to find the well
	http.HandleFunc("/find", func(w http.ResponseWriter, r *http.Request) {

		wellName := r.URL.Query().Get("wellname")

		// assign the search term to be a well passed to a handler
		wellReq.FullText = wellName

		// prepare request JSON from well request struct
		buf, err := json.Marshal(wellReq)
		if err != nil {
			log.Printf("Error creating JSON: %s", err)
		}
		log.Printf("Request JSON: %s", buf)

		// call Search API with the well search request JSON
		resp, err := http.Post(API_BASE_URL+"/indexSearch", "application/json", bytes.NewBuffer(buf))

		if err != nil {
			log.Printf("HTTP request failed with %s", err)
		}
		body, err := ioutil.ReadAll(resp.Body)

		// parse the results and extract files/srns for each resource type
		SRNs := getFilesFromResults(body)
		resJSON, err := json.Marshal(SRNs)
		if err != nil {
			log.Printf("Marshalling result JSON failed with %s", err)
		}

		// return response JSON back to browser
		fmt.Fprintf(w, string(resJSON))
	})

	log.Printf("listening on http://%s/", "127.0.0.1:8080")
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", nil))
}
