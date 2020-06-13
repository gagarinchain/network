# Dockerfile References: https://docs.docker.com/engine/reference/builder/

# Start from the latest golang base image
FROM golang:latest as builder

# Add Maintainer Info
LABEL maintainer="dabasov"

RUN apt-get update && apt-get install -y ca-certificates git-core ssh
COPY git-docker /root/.ssh/id_rsa
RUN chmod 700 /root/.ssh/id_rsa
RUN echo "Host github.com\n\tStrictHostKeyChecking no\n" >> /root/.ssh/config
RUN git config --global url.ssh://git@github.com/.insteadOf https://github.com/

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

#ignore golang sum checking for private repos
RUN go env -w GOPRIVATE=github.com/gagarinchain/*

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .
# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o gnetwork .
#RUN GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o /go/bin/app

######## Start a new stage from scratch #######
FROM alpine:latest

ENV COMMAND "--help"

RUN apk --no-cache add ca-certificates

WORKDIR /root/
# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/gnetwork .
COPY --from=builder /app/static ./static

# Expose port 8080 to the outside world
EXPOSE 9080
RUN chmod +x ./gnetwork
# Command to run the executable
CMD ["sh", "-c", "/root/gnetwork $COMMAND"]
