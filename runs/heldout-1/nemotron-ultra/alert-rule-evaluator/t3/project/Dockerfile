FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/alert-rule-evaluator .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/alert-rule-evaluator /alert-rule-evaluator
ENV PORT=8080
EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/alert-rule-evaluator"]
