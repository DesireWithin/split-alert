# 使用官方 Go 镜像作为构建阶段
FROM golang:1.24.2-alpine3.21 AS builder

# 设置工作目录
WORKDIR /app

# 将项目的所有文件复制到容器中
COPY . .

# 下载依赖并构建可执行文件
RUN go mod tidy && go build -o splitAlert .

# 使用一个更小的基础镜像运行程序
FROM alpine:3.21

# 安装必要的依赖（如 ca-certificates 和 curl）
RUN apk --no-cache add ca-certificates curl

# 设置工作目录为 /opt/splitAlert
WORKDIR /opt/splitAlert

# 从构建阶段复制可执行文件到 /opt/splitAlert
COPY --from=builder /app/splitAlert .

# 确保可执行文件具有执行权限
RUN chmod +x /opt/splitAlert/splitAlert

# 暴露服务端口
EXPOSE 8080

# 设置容器启动时的默认命令
CMD ["/opt/splitAlert/splitAlert"]