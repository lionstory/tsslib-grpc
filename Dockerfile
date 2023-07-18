FROM golang:1.16.7-buster  as builder
WORKDIR /tss-server/
ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn
COPY . .

RUN make build

FROM debian:buster-slim
WORKDIR /tss-server
COPY --from=builder /tss-server/bin/tss-server /tss-server/bin/tss-server
COPY --from=builder /tss-server/conf/config.yaml /tss-server/conf/config.yaml

ENTRYPOINT ["sh", "-c", "bin/tss-server"]