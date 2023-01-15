FROM golang:1.19-alpine

COPY . /usr/local/go/src/ns-gobridge/

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY *.go ./

RUN go build -o /ns-gobridge

CMD [ "/ns-gobridge" ]