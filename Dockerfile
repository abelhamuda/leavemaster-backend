# Gunakan image resmi Go
FROM golang:1.22-alpine

# Set working directory
WORKDIR /app

# Copy semua file ke dalam container
COPY . .

# Download dependencies
RUN go mod tidy

# Build binary
RUN go build -o main .

# Expose port
EXPOSE 8080

# Jalankan aplikasi
CMD ["./main"]
