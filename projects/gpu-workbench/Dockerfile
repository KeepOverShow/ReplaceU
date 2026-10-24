FROM golang:1.24.6-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -trimpath -o /workbench ./cmd/workbench
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/* && mkdir /data && chown 10001:10001 /data
COPY --from=build /workbench /usr/local/bin/workbench
USER 10001:10001
ENTRYPOINT ["/usr/local/bin/workbench"]
CMD ["-mode","k8s-sim","-listen","0.0.0.0:8787","-data","/data"]
