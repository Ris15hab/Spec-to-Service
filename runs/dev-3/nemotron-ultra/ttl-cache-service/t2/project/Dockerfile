FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/ttl-cache-service .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/ttl-cache-service /ttl-cache-service
ENV PORT=8080
EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/ttl-cache-service"]
