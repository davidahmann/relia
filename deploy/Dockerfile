FROM golang:1.24-alpine AS build

WORKDIR /src

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /out/relia-gateway ./cmd/relia-gateway
RUN CGO_ENABLED=0 go build -o /out/relia ./cmd/relia-cli

FROM alpine:3.20

RUN apk add --no-cache ca-certificates \
  && adduser -D -g "" relia

WORKDIR /app

COPY --from=build /out/relia-gateway /usr/local/bin/relia-gateway
COPY --from=build /out/relia /usr/local/bin/relia

COPY policies /app/policies
COPY relia.yaml /app/relia.yaml

USER relia

EXPOSE 8080

ENV RELIA_LISTEN_ADDR=":8080"
ENV RELIA_POLICY_PATH="/app/policies/relia.yaml"
ENV RELIA_CONFIG_PATH="/app/relia.yaml"

ENTRYPOINT ["relia-gateway"]
