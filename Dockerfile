# 使用官方 Golang 基础镜像
FROM golang:1.22-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 编译应用程序
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ddns .

# 使用最小化的 alpine 镜像
FROM alpine:latest

# 安装 ca-certificates 用于HTTPS请求
RUN apk --no-cache add ca-certificates tzdata

# 设置时区为上海
RUN cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && echo "Asia/Shanghai" > /etc/timezone

# 创建非root用户
RUN addgroup -g 1001 ddns && \
    adduser -D -s /bin/sh -u 1001 -G ddns ddns

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/ddns .

# 更改文件所有者
RUN chown ddns:ddns /app/ddns

# 切换到非root用户
USER ddns

# 暴露端口（虽然这个应用不需要端口，但保留以防将来需要）
# EXPOSE 8888

# 运行应用程序
CMD ["./ddns"]