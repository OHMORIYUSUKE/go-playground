FROM golang:1.20.4

CMD ["/bin/bash", "-b"]

ENV GOPATH=/go

RUN apt-get update && apt-get upgrade -y\
    && apt-get autoremove -y\
    && apt-get clean\
    && rm -rf /var/lib/apt/lists/*

WORKDIR /go/src

COPY ./src/go.mod /go/src/go.mod
COPY ./src/go.sum /go/src/go.sum
RUN go get -u github.com/cosmtrek/air && \
    go build -o /go/bin/air github.com/cosmtrek/air
RUN go mod download

