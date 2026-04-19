# RUN INSTRUCTIONS ####################ß
# - docker build -t typing-test .
# - docker run -p 3333:3333 typing-test
# #####################################

# Stage 1: Build the React frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build

# Stage 2: Build the Go server
FROM golang:1.21-alpine AS backend-builder
WORKDIR /app
# Copy Go module files and download dependencies (if you have go.mod/go.sum initialized)
COPY go.mod go.sum* ./
RUN go mod download || true
COPY . .
# Build the Go app
RUN go build -o server main.go

# Stage 3: Create the final minimal production image
FROM alpine:latest
WORKDIR /app

# Copy the built Go binary from the backend-builder stage
COPY --from=backend-builder /app/server .

# Copy the built React static files from the frontend-builder stage
# main.go expects them to be in ./frontend/dist
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

EXPOSE 3333

CMD ["./server"]