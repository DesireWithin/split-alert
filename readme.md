# SRE Split Alert

SRE Split Alert 是一个用于接收Prometheus Alertmanager 告警并根据状态（`firing` 或 `resolved`）进行分组和转发的服务, 接受Alertmanager的信息，分组，并推送到[PrometheusAlert](https://github.com/feiyu563/PrometheusAlert)。原生Alertmanager group中既包括`firing` 也包括 `resolved`，参考此[issue](https://github.com/prometheus/alertmanager/issues/2334)

## 功能

- 接收 Prometheus Alertmanager告警请求。
- 根据告警的状态（`firing` 或 `resolved`）对告警进行分组。
- 将分组后的告警转发到指定的目标地址。

## 文件结构

- `Dockerfile`: 用于构建和运行服务的 Docker 镜像。
- `splitAlert.go`: 服务的核心逻辑，包括告警处理和配置加载。
- `config.yml`: 服务的配置文件（示例见下文）。

## 配置文件

服务需要一个 `config.yml` 文件，放置在 `/opt/splitAlert/config.yml` 路径下。以下是一个示例配置：

```yaml
prometheusAlertUrl: "http://example.com/alert"
config:
  exampleConfig:
    key1: "value1"
    key2: "value2"
```

- `prometheusAlertUrl`: 转发告警的目标地址。
- `config`: 配置映射，定义不同的配置名称及其对应的键值对，将其转换成额外的`url param`参数。

该配置文件完全兼容`PrometheusAlert`项目，例如

将Alertmanager的配置
```yaml
- name: 'prod'
   webhook_configs:
   - url: http://prometheusalert.monitoring:8080/prometheusalert?type=yourtype&tpl=yourtpl&fsurl=https://open.xxx.cn/open-apis/bot/v2/hook/xxxx
```
转换成
```yaml
- name: 'prod'
   webhook_configs:
   - url: 'http://split-alert.monitoring:8080/alert?config=prod'

```
同时，将`PrometheusAlert`的url以及额外的参数写进`config.yml`文件即可:
```yaml
    prometheusAlertUrl: http://prometheusalert.monitoring:8080/prometheusalert
    config:
      prod:
        fsurl: https://open.xxx.cn/open-apis/bot/v2/hook/xxxx
        tpl: yourtpl
        type: yourtype
```

## 构建和运行

### 使用 Docker 构建和运行

1. **构建 Docker 镜像**：

   进入到`code`目录中

   ```bash
   docker build -t sre-split-alert .
   ```

2. **运行容器**：

   ```bash
   docker run -d -p 8080:8080 -v /path/to/config.yml:/opt/splitAlert/config.yml sre-split-alert
   ```

   - 将本地的 `config.yml` 挂载到容器的 `/opt/splitAlert/config.yml`。

### 本地运行

1. **安装依赖**：

   确保已安装 Go 语言环境，然后运行以下命令：

   ```bash
   go mod tidy
   ```

2. **运行服务**：

   ```bash
   go run splitAlert.go
   ```

3. **访问服务**：

   服务默认运行在 `8080` 端口。

### K8S部署

进入到`deployment`中，手动apply yaml文件到`china-ops`集群中

## API 接口

### `/alert`

- **方法**: `POST`
- **描述**: 接收 Prometheus 告警并转发。
- **参数**: 
  - `config` (query 参数): 指定使用的配置名称。
- **请求体**: Prometheus 告警的 JSON 数据。

### `/reload`

- **方法**: `GET`
- **描述**: 手动重新加载配置文件。

## 日志

服务会输出以下类型的日志：

- `📥`: 表示接收到请求。
- `✅`: 表示操作成功。
- `❌`: 表示操作失败。
- `🔁`: 表示重新加载配置。

## 示例

以下是一个使用 `curl` 测试服务的示例：

```bash
curl -X POST "http://localhost:8080/alert?config=exampleConfig" \
-H "Content-Type: application/json" \
-d '{
  "status": "firing",
  "alerts": [
    {"status": "firing", "labels": {"alertname": "HighCPU"}},
    {"status": "resolved", "labels": {"alertname": "DiskFull"}}
  ]
}'
```

## 贡献

欢迎提交 Issue 或 Pull Request 来改进此项目。

## 许可证

此项目使用 MIT 许可证。