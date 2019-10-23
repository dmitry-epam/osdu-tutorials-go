/*
This is an example web application to demonstrate how to work with OSDU APIs:

	* Authenticate using OpenID Connect (authorization code flow)
	- try me: http://localhost:8080

	* Find a well using Search API (/indexSearch)
	- try me: http://localhost:8080/find?wellname=A05-01

	* Fetch trajectory using Delivery API (/GetResources && azblob)
	- try me: http://localhost:8080/fetch?srn=srn:file/csv:6dd13750df8611e9b5df4fa704076d5c:1

*/
package main

import (
	"bytes"
	"encoding/json"
	"github.com/Azure/azure-storage-blob-go/azblob"
	oidc "github.com/coreos/go-oidc"
	"github.com/tidwall/gjson"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"os"
)

var (
	// get OSDU API base URL from your Cloud Administrator
	clientAPIBaseURL = os.Getenv("OSDU_API_BASE_URL")
	
	// get Client ID and Client Secret from mgmt portal during app registration
	clientAuthBaseURL = os.Getenv("OSDU_AUTH_BASE_URL")
	clientID = os.Getenv("OSDU_CLIENT_ID")
	clientSecret = os.Getenv("OSDU_CLIENT_SECRET")
)

/*
  Function extracts file names and srns for each resource type from
  JSON response body and strips out everything else
*/
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

/*
	This function constructs the pre-signed file URL based on
	JSON response received from Delivery API
	@ todo: Test with AWS
*/
func getFileURL(body []byte) string {

	bodyResponse := string(body)

	// getting parameters from JSON
	endPoint := gjson.Get(bodyResponse, "Result.0.FileLocation.EndPoint").String()
	bucket := gjson.Get(bodyResponse, "Result.0.FileLocation.Bucket").String()
	key := gjson.Get(bodyResponse, "Result.0.FileLocation.Key").String()
	SAS := gjson.Get(bodyResponse, "Result.0.FileLocation.TemporaryCredentials.SAS").String()

	log.Printf("Extracted file parameters: %s, %s, %s", endPoint, bucket, key)

	// creating file download URL
	fileURL := endPoint + bucket + "/" + key + "?" + SAS
	//log.Printf("Created file URI: %s", fileURL)

	return fileURL
}

/*
	This function downloads data from Azure Blob storage
	to an in-memory buffer, do not use in prod
	since buffer can fail to grow
*/
func getBufFromBlob(ctx context.Context, remoteFileURL string) []byte {

	var err error

	// When someone receives the URL, they access the SAS-protected resource with code like this:
	u, _ := url.Parse(remoteFileURL)

	// Create an BlobURL object that wraps the blob URL (and its SAS) and a pipeline.
	// When using a SAS URLs, anonymous credentials are required.
	blobURL := azblob.NewBlobURL(*u, azblob.NewPipeline(azblob.NewAnonymousCredential(), azblob.PipelineOptions{}))
	//log.Printf("Blob URL: %s", blobURL)

	// setting the properties for downloading a blob incl. the prgress function
	options := azblob.DownloadFromBlobOptions{
		BlockSize:   2048, // bytes, experiment depending on the file size
		Parallelism: 1,    // start with 1 and increase if you have bigger files
		Progress: func(bytesTransferred int64) {
			log.Printf("Downloaded %s bytes", strconv.FormatInt(bytesTransferred, 10))
		},
	}

	// first, we need to indentify the size of a blob to allocate buffer
	props, err := blobURL.GetProperties(ctx, options.AccessConditions)
	if err != nil {
		log.Printf("Cannot read blob size: %s", err)
	}
	size := props.ContentLength()
	log.Printf("Blob size is %s bytes", strconv.FormatInt(size, 10))

	buf := make([]byte, size)

	// next, we're reading the blob into in-memory buffer
	// @todo - there must be a better way to do this!
	err = azblob.DownloadBlobToBuffer(ctx, blobURL, 0, 0, buf, options)
	if err != nil {
		log.Printf("Error downloading Blob: %s", err)
	}

	log.Printf("Buffer len: %v, cap: %v", len(buf), cap(buf))

	return buf
}

func main() {

	ctx := context.Background()

	// clientAuthBaseURL is used to discover /authorize, /token and
	// /userinfo endpoints automatically
	provider, err := oidc.NewProvider(ctx, clientAuthBaseURL)

	if err != nil {
		log.Println("Failed to discover provider details. Program will terminate.")
		log.Fatal(err)
	}

	log.Printf("Provider details (discovered):\n%s", provider)

	config := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  "http://localhost:8080/auth/callback",
		// "openid" is a required scope for OpenID Connect flows
		// keep in mind: not all providers support "profile" scope
		// "offline_access" is required to get refresh_token
		Scopes: []string{oidc.ScopeOpenID, "email", "offline_access"},
	}

	// this is typically the page a user was on
	// before the sign-in process, otherwise a random string
	state := "foobar"

	// this handler initiates the sign-in process by redirecting
	// to the provider authorization endpoint
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, config.AuthCodeURL(state), http.StatusFound)
	})

	// this handler validates the state, so it hasn't changed during the communication
	// process, then exchanges the authorization code for the access_token using
	// clientId/clientSecret; and finally, extracts user info from the id_token
	// and returns everything back to browser
	http.HandleFunc("/auth/callback", func(w http.ResponseWriter, r *http.Request) {

		if r.URL.Query().Get("state") != state {
			http.Error(w, "state did not match", http.StatusBadRequest)
			return
		}

		oauth2Token, err := config.Exchange(ctx, r.URL.Query().Get("code"))
		if err != nil {
			http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
			return
		}

		IDToken, ok := oauth2Token.Extra("id_token").(string)
		if !ok {
			http.Error(w, "No id_token field in oauth2 token.", http.StatusInternalServerError)
			return
		}

/* 		refreshToken, ok := oauth2Token.Extra("refresh_token").(string)
		if !ok {
			http.Error(w, "No refresh_token field in oauth2 token.", http.StatusInternalServerError)
			return
		} */

		userInfo, err := provider.UserInfo(ctx, oauth2.StaticTokenSource(oauth2Token))
		if err != nil {
			http.Error(w, "Failed to get userinfo: "+err.Error(), http.StatusInternalServerError)
			return
		}

		resp := struct {
			OAuth2Token *oauth2.Token
			UserInfo    *oidc.UserInfo
			IDToken string `json:"id_token"`
		}{oauth2Token, userInfo, IDToken}
		data, err := json.MarshalIndent(resp, "", "    ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(data)
	})

	///////////////////////////////////////////////////////////////////////////

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
		resp, err := http.Post(clientAPIBaseURL+"/indexSearch", "application/json", bytes.NewBuffer(buf))

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
		w.Write(resJSON)
	})

	///////////////////////////////////////////////////////////////////////////

	type FileRequest struct {
		SRNS           []string
		TargetRegionID string
	}

	// find handler takes "srn" as input parameter and makes Delivery API call
	// to get the pre-signed File URL to download
	http.HandleFunc("/fetch", func(w http.ResponseWriter, r *http.Request) {

		SRN := r.URL.Query().Get("srn")

		// assign the search parameter to SRN that is passed;
		// we can pass multiple SRNs if needed, just append them all
		var fileReq FileRequest
		fileReq.SRNS = append(fileReq.SRNS, SRN)

		// prepare request JSON from file request struct
		searchRequest, err := json.Marshal(fileReq)
		if err != nil {
			log.Printf("Error creating JSON: %s", err)
		}
		log.Printf("Request JSON: %s", searchRequest)

		// call Delivery API with the file search request JSON
		resp, err := http.Post(clientAPIBaseURL+"/GetResources", "application/json", bytes.NewBuffer(searchRequest))
		if err != nil {
			log.Printf("HTTP request failed with %s", err)
		}

		body, err := ioutil.ReadAll(resp.Body)

		// construct pre-signed blob URL from response JSON
		remoteURL := getFileURL(body)

		// download Blob to a buffer using pre-signed URL
		buf := getBufFromBlob(ctx, remoteURL)

		// return response CSV back to browser
		w.Write(buf)
	})

	log.Printf("listening on http://%s/", "0.0.0.0:8080")
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", nil))
}
