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
database_url: "postgres://user:password@localhost:5433/ai_gateway"
gateway_api_key: "local-dev-key"
gateway_api_keys: []
dashscope_base_url: "https://dashscope.aliyuncs.com/api/v1"
dashscope_api_key: ""
yunwu_base_url: "https://yunwu.ai"
yunwu_api_key: ""
ark_image_endpoint: "https://ark.cn-beijing.volces.com/api/v3/images/generations"
ark_image_api_key: ""
ark_video_endpoint: "https://ark.cn-beijing.volces.com/api/v3/contents/generations/tasks"
ark_video_api_key: ""

image_model_providers:
  - model: "ep-20260313204854-n5jb5"
    provider: "ark"
  - model: "wan2.7-image-pro"
    provider: "dashscope"
  - model: "gemini-3.1-flash-image-preview"
    provider: "yunwu"
  - model: "gemini-3-pro-image-preview"
    provider: "yunwu"
  - model: "gpt-image-2"
    provider: "yunwu"

video_model_providers:
  - model: "doubao-seedance-1-5-pro-251215"
    provider: "ark"
  - model: "wan2.7-t2v-2026-04-25"
    provider: "dashscope"
  - model: "wan2.7-i2v-2026-04-25"
    provider: "dashscope"
  - model: "wan2.7-r2v"
    provider: "dashscope"
```

Image models route through `image_model_providers`; video models route through `video_model_providers`. Each entry maps one `model` to one provider; supported providers are `ark`, `dashscope`, and `yunwu` for images, and `ark` and `dashscope` for videos. Requests whose model is not listed in the matching section are rejected.

Video task IDs returned by the gateway include a provider prefix such as `dashscope-t2v_...`, `dashscope-i2v_...`, `dashscope-r2v_...`, or `ark_...`; pass the full value back to the GET endpoint.

For DashScope image generation, the gateway sends BaiLian Wan 2.7 requests as `input.messages[].content[]` with `{"text": ...}` followed by zero or more `{"image": ...}` reference images. Set `"async": true` on `POST /v1/images/generations` to create an async image task through `POST /services/aigc/image-generation/generation`; otherwise the gateway uses synchronous `POST /services/aigc/multimodal-generation/generation`. DashScope image defaults include `n: 1` and `watermark: false` unless overridden.

For Yunwu image generation, Gemini image models such as `gemini-3.1-flash-image-preview` and `gemini-3-pro-image-preview` are sent to `POST /v1beta/models/{model}:generateContent` with `response_modalities: ["IMAGE", "TEXT"]`. `gpt-image-2` is sent to `POST /v1/images/generations` with `model`, `prompt`, `n`, `size`, `quality`, and `format`. Inline base64 images from Yunwu are returned as `data:image/...;base64,...` values in the existing `url` and `urls` fields.

For DashScope `wan2.7-r2v`, pass reference assets in `media`, for example `{"type":"reference_image","url":"https://...","reference_voice":"https://..."}` or `{"type":"reference_video","url":"https://..."}`. The gateway forwards them as `input.prompt` plus `input.media` to `POST /services/aigc/video-generation/video-synthesis`.

For Ark video generation, the gateway submits tasks to `POST /api/v3/contents/generations/tasks` and queries `GET /api/v3/contents/generations/tasks/{id}`. Ark statuses `queued` and `running` are intermediate polling states; `succeeded`, `failed`, `cancelled`, and `expired` are terminal states. On `succeeded`, the gateway returns Ark's `content.video_url` as `video_url`.

Postgres API keys are stored as SHA-256 hashes in `gateway_api_keys.key_hash`.

Database tables are migrated on startup with GORM `AutoMigrate`.
