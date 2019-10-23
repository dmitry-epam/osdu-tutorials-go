# docker build -t osdu-quickstart-go-srv .
# docker run -p 8080:8080 --env-file ./config-azure.properties osdu-quickstart-go-srv

FROM golang:1.13

WORKDIR /app

# resolving app dependencies
COPY go.mod go.sum ./
RUN go mod download

# building the app
COPY cmd/srv cmd/srv
RUN go build -o main ./cmd/srv

# localhost:8080
EXPOSE 8080

CMD ["./main"]