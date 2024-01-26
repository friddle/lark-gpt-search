FROM  golang:1.20-alpine as builder
WORKDIR src
COPY . .
RUN go mod download
RUN mkdir -p /src/dist/
RUN go build -o /src/dist/feishu_gpt_search main.go

FROM alpine:latest
WORKDIR /app/
COPY --from=builder /src/dist/feishu_gpt_search /app/feishu_gpt_search
VOLUME /app/.feishu.env
VOLUME /app/.chatgpt.env
ENTRYPOINT ["/app/feishu_gpt_search"]
