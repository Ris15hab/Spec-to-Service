FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/artifact-registry-metadata .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/artifact-registry-metadata /artifact-registry-metadata
ENV PORT=8080
EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/artifact-registry-metadata"]
