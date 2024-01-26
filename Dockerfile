FROM  golang:1.20-alpine as builder
WORKDIR src
COPY . .
RUN go mod download
RUN go build -o dist/feishu_gpt_search main.go

FROM alipine:3.36
WORKDIR /app/
COPY --from=builder  /src/dist/feishu_gpt_search /app/feishu_gpt_search
VOLUME /app/.feishu.env
VOLUME /app/.chatgpt.env
ENTRYPOINT ["/app/feishu_gpt_search"]
