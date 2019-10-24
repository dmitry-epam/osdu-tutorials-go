# osdu-tutorials-go
OSDU tutorials for various use cases in Go

# QuickStart web app
Simple web application demonstrating how to use Authentication, Search and Delivery APIs

## Before you start
1. Get client ID, client Secret, Authorization URL and API URL from your platform admin.
2. Clone this repository.
3. Go to 'quickstart' folder, rename default environment file to config-azure|aws|gcp.env and fill out the values:
```
# OAuth settings
OSDU_CLIENT_ID=<your-client-id>
OSDU_CLIENT_SECRET=<your-client-secret>
OSDU_AUTH_BASE_URL=<auth-server-url>

# API
OSDU_API_BASE_URL=<api-base-url>
```

## How to run locally in WSL/Linux

1. Download and install Go.
2. Export environment variables:

For Azure
```
$ export $(grep -v '^#' config-azure.env | xargs -d '\n')
```
For AWS
```
$ export $(grep -v '^#' config-aws.env | xargs -d '\n')
```
3. Build and run the server:
```
$ go run ./cmd/srv
```
4. Go to http://localhost:8080

## How to run inside Docker container

1. Edit docker-compose.yml to include configuration to your environment:
```
version : '3'
services:
  backend:
    env_file:
      - config-<your-env>.env
    build: .
    ports:
      - "8080:8080"
    command: /root/main
```
2. Build the image and run the container:
```
$ docker-compose up --build
```
3. Go to http://localhost:8080