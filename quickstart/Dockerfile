# docker build -t osdu-quickstart-go-srv .
# docker run -p 8080:8080 --env-file ./config-azure.properties osdu-quickstart-go-srv

### Building stage 
FROM golang:alpine as builder

WORKDIR /app

# resolving app dependencies
COPY go.mod go.sum ./
RUN go mod download

# building the app
COPY cmd/srv cmd/srv
RUN go build -o main ./cmd/srv

### New stage to pack the binary to Alpine image
FROM alpine:latest  

RUN apk --no-cache add ca-certificates

WORKDIR /root

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/main .

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["/root/main"] 
#CMD /bin/ash