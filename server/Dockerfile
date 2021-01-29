FROM golang:alpine 

WORKDIR /go/src/github.com/NathanBland/go-notes
ADD ./go.mod ./go.mod
ADD ./go.sum ./go.sum

ENV GOPATH /go

RUN apk add git gcc g++
# RUN go mod download
RUN go get -u github.com/cosmtrek/air && \
  go get go.mongodb.org/mongo-driver && \
  go get github.com/gofiber/fiber/v2 && \
  go get github.com/ramya-rao-a/go-outline && \
  go get github.com/go-delve/delve/cmd/dlv && \
  go get golang.org/x/lint/golint && \
  go get golang.org/x/tools/gopls && \
  go get github.com/uudashr/gopkgs/v2/cmd/gopkgs 

# RUN go mod init

# RUN go get -d -v ./...
# RUN go install -v ./...

# CMD ["air"]