# osdu-tutorials-go
OSDU tutorials for various use cases in Go

# QuickStart web app
Simple web application demonstrating how to use Authentication, Search and Delivery APIs

## How to run locally in WSL/Linux

1. Get client ID, client Secret, Authorization URL and API URL from your platform admin.
2. Download and install Go.
3. Clone this repository.
4. Go to 'quickstart' folder, rename default config file to config-azure|aws.profile and fill out the values:
```
# OAuth settings
export OSDU_CLIENT_ID="<your-client-id>"
export OSDU_CLIENT_SECRET="<your-client-secret>"
export OSDU_AUTH_BASE_URL="<auth-server-url>"

# API
export OSDU_API_BASE_URL="<api-base-url>"
```
5. Export configuration settings:
```
$ source config.profile
```
6. Build and run the server:
```
$ go run ./cmd/srv
```