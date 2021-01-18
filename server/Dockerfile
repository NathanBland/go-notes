FROM golang:alpine 

WORKDIR /go/src/github.com/NathanBland/go-vite-docker-starter
ADD ./go.mod ./go.mod
ADD ./go.sum ./go.sum

ENV GOPATH /go

RUN apk add git gcc g++
# RUN go mod download
RUN go get -u github.com/cosmtrek/air
RUN go get go.mongodb.org/mongo-driver
RUN go get github.com/gofiber/fiber/v2

# RUN go mod init

# RUN go get -d -v ./...
# RUN go install -v ./...

# CMD ["air"]