# AI Gateway

Go gateway for image generation and video generation. The HTTP server uses Gin, upstream requests use Resty, logging uses Logrus, configuration is loaded from YAML through Viper, and database access plus schema migration use GORM.

## Endpoints

- `GET /healthz`
- `POST /v1/images/generations`
- `GET /v1/images/generations/{task_id}`
- `POST /v1/video/generations`
- `GET /v1/video/generations/{task_id}`
- `POST /v1/video/tasks`
- `GET /v1/video/tasks/{task_id}`

All endpoints require:

```http
Authorization: Bearer <GATEWAY_API_KEY>
Content-Type: application/json
X-Request-Id: optional-request-id
```

Every response includes `X-Request-Id`. Errors use:

```json
{
  "error": {
    "code": "string",
    "message": "string",
    "request_id": "string"
  }
}
```

## Configuration

Edit `config.yaml` or override values with environment variables.

```yaml
addr: ":8080"
database_url: "postgres://user:password@localhost:5432/ai_gateway"
gateway_api_key: "local-dev-key"
gateway_api_keys: []
kling_base_url: "https://api-singapore.klingai.com"
kling_access_key: ""
kling_secret_key: ""
dashscope_base_url: "https://dashscope.aliyuncs.com/api/v1"
dashscope_api_key: ""

image_model_providers:
  - model: "ep-20260313204854-n5jb5"
    provider: "ark"
  - model: "wan2.6-image"
    provider: "dashscope"
  - model: "kling-v2-1"
    provider: "kling"
  - model: "kling-v2-6"
    provider: "kling"

video_model_providers:
  - model: "doubao-seedance-1-5-pro-251215"
    provider: "ark"
  - model: "wan2.7-t2v-2026-04-25"
    provider: "dashscope"
  - model: "wan2.7-i2v-2026-04-25"
    provider: "dashscope"
  - model: "kling-v2-1"
    provider: "kling"
  - model: "kling-v2-6"
    provider: "kling"
```

Image models route through `image_model_providers`; video models route through `video_model_providers`. Each entry maps one `model` to one provider; supported providers are `ark`, `dashscope`, and `kling`. Requests whose model is not listed in the matching section are rejected.

Video task IDs returned by the gateway include a provider prefix such as `kling-t2v_...`, `kling-i2v_...`, `dashscope-t2v_...`, `dashscope-i2v_...`, or `ark_...`; pass the full value back to the GET endpoint.

Postgres API keys are stored as SHA-256 hashes in `gateway_api_keys.key_hash`.

Database tables are migrated on startup with GORM `AutoMigrate`.
