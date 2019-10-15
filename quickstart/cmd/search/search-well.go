/*
This is an example application to demonstrate querying the Search API.
*/
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

const API_BASE_URL = "https://osdu-demo-portal-dev.azure-api.net"


// TO DO
// this is a stub to parse SRNs out of search response
func parseBody(responseBody []byte, resourceType []string) []string {
	
	log.Printf("Parsing resource types: %s\n", resourceType)

	SRNs := []string{"srn:master-data/Wellbore:8438:", "srn:work-product-component/WellborePath:8438_csv:", "srn:file/csv:6dd13750df8611e9b5df4fa704076d5c:1"}
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
		Metadata: Metadata{ResourceType: []string{"master-data/well", "work-product-component/welllog"}},
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

		SRNs := parseBody(body, wellReq.Metadata.ResourceType)

		// return response JSON back to browser
		for _, srn := range SRNs {
			fmt.Fprintln(w, srn)	
		}
		//fmt.Fprintf(w, SRNs[0])
	})

	log.Printf("listening on http://%s/", "127.0.0.1:8080")
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", nil))
}
