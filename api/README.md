# Golang API service

A cake-giving service - you do a request, and it prints you a "cake"

# Install

`go mod init api/cake`<br />
`go mod tidy`<br />
`go install api/cake`<br />

# Build

`go build`

# Run

`export CAKE_ADMIN_EMAIL=superuser@cake.com`<br />
`export CAKE_ADMIN_PASSWORD=superpassword`<br />
`sudo ./cake`<br />

# Tests

<b>Simple test:</b><br />
`go test -v`

<b>Formatting check:</b><br />
`gofmt -d -s .`

<b>Coverage percentage check:</b><br />
`go test -race -count 1 -cover -v ./...`