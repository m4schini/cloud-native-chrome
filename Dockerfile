## Build
FROM golang:1.19-buster AS build

WORKDIR /build

COPY . .

RUN go mod tidy

RUN go build -o /app

## Deploy
FROM chromedp/headless-shell

RUN apt-get update
RUN apt-get install -y curl

WORKDIR /

COPY --from=build /app /app

ENV PORT "30051"
ENV METRICS_PORT "4242"
ENV DEV "false"

HEALTHCHECK --interval=5s --timeout=3s CMD curl --fail http://localhost:${METRICS_PORT}/metrics || exit 1

EXPOSE ${PORT}
EXPOSE ${METRICS_PORT}

ENTRYPOINT ["/app"]