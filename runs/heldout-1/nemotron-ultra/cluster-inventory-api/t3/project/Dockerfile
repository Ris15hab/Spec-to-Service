FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/cluster-inventory-api .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/cluster-inventory-api /cluster-inventory-api
ENV PORT=8080
EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/cluster-inventory-api"]
