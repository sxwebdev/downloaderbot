# Environment Variables

## Application

| **Name**                                        | **Required** | **Secret** | **Default value** | **Usage**                                                                     | **Example**      |
| ----------------------------------------------- | ------------ | ---------- | ----------------- | ----------------------------------------------------------------------------- | ---------------- |
| `DOWNLOADERBOT_LOG_FORMAT`                      |              |            | `json`            | allows to set custom formatting                                               | `json`           |
| `DOWNLOADERBOT_LOG_LEVEL`                       |              |            | `info`            | allows to set custom logger level                                             | `info`           |
| `DOWNLOADERBOT_LOG_CONSOLE_COLORED`             |              |            | `false`           | allows to set colored console output                                          | `false`          |
| `DOWNLOADERBOT_LOG_TRACE`                       |              |            | `fatal`           | allows to set custom trace level                                              | `fatal`          |
| `DOWNLOADERBOT_LOG_WITH_CALLER`                 |              |            | `false`           | allows to show caller                                                         | `false`          |
| `DOWNLOADERBOT_LOG_WITH_STACK_TRACE`            |              |            | `false`           | allows to show stack trace                                                    | `false`          |
| `DOWNLOADERBOT_OPS_ENABLED`                     |              |            | `false`           | allows to enable ops server                                                   | `false`          |
| `DOWNLOADERBOT_OPS_NETWORK`                     | ✅           |            | `tcp`             | allows to set ops listen network: tcp/udp                                     | `tcp`            |
| `DOWNLOADERBOT_OPS_TRACING_ENABLED`             |              |            | `false`           | allows to enable tracing                                                      | `false`          |
| `DOWNLOADERBOT_OPS_METRICS_ENABLED`             |              |            | `false`           | allows to enable metrics                                                      | `true`           |
| `DOWNLOADERBOT_OPS_METRICS_PATH`                | ✅           |            | `/metrics`        | allows to set custom metrics path                                             | `/metrics`       |
| `DOWNLOADERBOT_OPS_METRICS_PORT`                | ✅           |            | `10000`           | allows to set custom metrics port                                             | `10000`          |
| `DOWNLOADERBOT_OPS_METRICS_BASIC_AUTH_ENABLED`  |              |            | `false`           | allows to enable basic auth                                                   |                  |
| `DOWNLOADERBOT_OPS_METRICS_BASIC_AUTH_USERNAME` |              |            |                   | auth username                                                                 |                  |
| `DOWNLOADERBOT_OPS_METRICS_BASIC_AUTH_PASSWORD` |              |            |                   | auth password                                                                 |                  |
| `DOWNLOADERBOT_OPS_HEALTHY_ENABLED`             |              |            | `false`           | allows to enable health checker                                               | `true`           |
| `DOWNLOADERBOT_OPS_HEALTHY_PATH`                | ✅           |            | `/healthy`        | allows to set custom healthy path                                             | `/healthy`       |
| `DOWNLOADERBOT_OPS_HEALTHY_PORT`                | ✅           |            | `10000`           | allows to set custom healthy port                                             | `10000`          |
| `DOWNLOADERBOT_OPS_HEALTHY_LIVENESS_PATH`       |              |            | `/livez`          | liveness probe path                                                           | `/livez`         |
| `DOWNLOADERBOT_OPS_HEALTHY_READINESS_PATH`      |              |            | `/readyz`         | readiness probe path                                                          | `/readyz`        |
| `DOWNLOADERBOT_OPS_PROFILER_ENABLED`            |              |            | `false`           | allows to enable profiler                                                     | `false`          |
| `DOWNLOADERBOT_OPS_PROFILER_PATH`               | ✅           |            | `/debug/pprof`    | allows to set custom profiler path                                            | `/debug/pprof`   |
| `DOWNLOADERBOT_OPS_PROFILER_PORT`               | ✅           |            | `10000`           | allows to set custom profiler port                                            | `10000`          |
| `DOWNLOADERBOT_OPS_PROFILER_WRITE_TIMEOUT`      |              |            | `60`              | HTTP server write timeout in seconds                                          | `60`             |
| `DOWNLOADERBOT_GRPC_ENABLED`                    |              |            | `true`            | allows to enable grpc server                                                  | `true`           |
| `DOWNLOADERBOT_GRPC_ADDR`                       | ✅           |            | `:9000`           | grpc server listen address                                                    | `localhost:9000` |
| `DOWNLOADERBOT_GRPC_NETWORK`                    | ✅           |            | `tcp`             | grpc server listen network: tpc/udp                                           | `tcp`            |
| `DOWNLOADERBOT_GRPC_REFLECT_ENABLED`            |              |            | `false`           | allows to enable grpc reflection service                                      | `false`          |
| `DOWNLOADERBOT_GRPC_HEALTH_CHECK_ENABLED`       |              |            | `false`           | allows to enable grpc health checker                                          | `false`          |
| `DOWNLOADERBOT_GRPC_LOGGER_ENABLED`             |              |            | `false`           | allows to enable logger. available only for default grpc sevrer               | `false`          |
| `DOWNLOADERBOT_GRPC_RECOVERY_ENABLED`           |              |            | `false`           | allows to enable recovery from panics. available only for default grpc sevrer | `false`          |
| `DOWNLOADERBOT_TELEGRAM_BOT_API_TOKEN`          | ✅           | ✅         |                   | use token for your telegram bot                                               |                  |
