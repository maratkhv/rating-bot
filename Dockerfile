FROM golang:1.22.2
WORKDIR /ratinger
COPY . .
RUN go mod download
RUN go build -o /bin/bot ./cmd/main
ENTRYPOINT [ "/bin/bot" ]