FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/webhook-relay .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/webhook-relay /webhook-relay
ENV PORT=8080
EXPOSE 8080
USER nonroot
ENTRYPOINT ["/webhook-relay"]
