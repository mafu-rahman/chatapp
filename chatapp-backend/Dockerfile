# Use a base image with Go installed
FROM golang:latest

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files
COPY go.mod go.sum ./

# Download and install dependencies
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go application
RUN go build -o main .

# Expose the port the application runs on
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
