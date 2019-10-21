/*
This is an example application to demonstrate querying the Search API.

Example to call it:
http://localhost:8080/fetch?srn=srn:file/csv:6dd13750df8611e9b5df4fa704076d5c:1

*/
package main

import (
	"bytes"
	"encoding/json"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/tidwall/gjson"
	"golang.org/x/net/context"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

const API_BASE_URL = "<your API base url here>"

/*
	This function creates the file URI based on JSON response
	received from Delivery API
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
func getBufFromBlob(remoteFileURL string) []byte {

	ctx := context.Background()

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
		buf, err := json.Marshal(fileReq)
		if err != nil {
			log.Printf("Error creating JSON: %s", err)
		}
		log.Printf("Request JSON: %s", buf)

		// call Delivery API with the file search request JSON
		resp, err := http.Post(API_BASE_URL+"/GetResources", "application/json", bytes.NewBuffer(buf))
		if err != nil {
			log.Printf("HTTP request failed with %s", err)
		}

		body, err := ioutil.ReadAll(resp.Body)

		remoteURL := getFileURL(body)
		buff := getBufFromBlob(remoteURL)

		// return response JSON back to browser

		w.Write(buff)
	})

	log.Printf("listening on http://%s/", "127.0.0.1:8080")
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", nil))
}
