FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/job-queue-api .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/job-queue-api /job-queue-api
ENV PORT=8080
EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/job-queue-api"]
