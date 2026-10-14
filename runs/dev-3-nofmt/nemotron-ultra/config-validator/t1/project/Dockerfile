FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/config-validator .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/config-validator /config-validator
ENV PORT=8080
EXPOSE 8080
USER nonroot
ENTRYPOINT ["/config-validator"]
