/*
This is an example application to demonstrate querying the Search API.
*/
package main

import (
	"log"
	"fmt"
	"net/http"
	"encoding/json"
	"bytes"
	"io/ioutil"
)

func main() {
	
	//metadata struct
	type Metadata struct {
		ResourceType []string `json:"resource_type"`
	}
	// search request struct
	type SearchRequest struct {
		FullText string `json:"fulltext"`
		Metadata Metadata `json:"metadata"`
		Facets []string `json:"facets"`
	}

	// construct an initial well search request
	wellReq := SearchRequest{
		FullText:	"*",
		Metadata:	Metadata{ResourceType:	[]string{"master-data/well", "work-product-component/wellbore"},},
		Facets:	[]string{"resource_type"},
	}

	log.Printf("Initialized well request: \n%s", wellReq)

	// search handler takes "wellname" as input parameters and makes Search API call to find the well
	http.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
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
		resp, err := http.Post("https://osdu-demo-portal-dev.azure-api.net/indexSearch", "application/json", bytes.NewBuffer(buf))
		
		if err != nil {
			log.Printf("HTTP request failed with %s", err)
		}
		body, err := ioutil.ReadAll(resp.Body)
		
		// return response JSON back to browser
		fmt.Fprintf(w, string(body))
	})

	log.Printf("listening on http://%s/", "127.0.0.1:8080")
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", nil))
}
