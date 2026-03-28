# LLM Provider 多模型支持

NLQ 支持多种 LLM Provider，通过统一的接口设计实现灵活切换。

## 支持的 Provider

| Provider | 模型示例 | API 格式 | 配置值 |
|----------|---------|---------|--------|
| **ZhipuAI** (智谱) | glm-4, glm-4-plus, glm-4-flash | OpenAI兼容 | `zhipuai` |
| **MiniMax** | M2-her, M2.5, M2.7 | OpenAI兼容 | `minimax` |
| **OpenAI** | gpt-4, gpt-3.5-turbo | OpenAI原生 | `openai` |
| **Azure OpenAI** | gpt-4, gpt-35-turbo | Azure格式 | `azure` |
| **Ollama** | llama2, mistral | 本地兼容 | `ollama` |

## 配置方法

### 1. 配置文件 (config/config.yaml)

```yaml
llm:
  provider: zhipuai    # 或 minimax, openai, azure, ollama
  model: glm-4-plus   # 模型名称
  api_key: your-api-key
  base_url: https://api.minimaxi.com  # Provider API地址
  timeout: 60s
  temperature: 0.0    # 温度参数，越低越确定
  max_tokens: 2048    # 最大生成token数
```

### 2. 环境变量

```bash
export LLM_API_KEY="your-api-key"
export LLM_MODEL="M2-her"
```

## Provider 配置示例

### MiniMax

```yaml
llm:
  provider: minimax
  model: M2-her
  api_key: your-minimax-api-key
  base_url: https://api.minimaxi.com
  timeout: 60s
```

**获取 API Key**: [MiniMax 开放平台](https://platform.minimaxi.com/user-center/basic-information/interface-key)

### ZhipuAI (默认)

```yaml
llm:
  provider: zhipuai
  model: glm-4-plus
  api_key: your-zhipuai-api-key
  base_url: https://open.bigmodel.cn/api/coding/paas/v4
  timeout: 90s
```

### OpenAI

```yaml
llm:
  provider: openai
  model: gpt-4
  api_key: your-openai-api-key
  base_url: https://api.openai.com/v1
  timeout: 60s
```

### Azure OpenAI

```yaml
llm:
  provider: azure
  model: gpt-4
  api_key: your-azure-api-key
  base_url: https://your-resource.openai.azure.com
  timeout: 60s
```

### Ollama (本地)

```yaml
llm:
  provider: ollama
  model: llama2
  api_key: ""  # 本地不需要 API Key
  base_url: http://localhost:11434/v1
  timeout: 120s
```

## 架构设计

```
┌─────────────────────────────────────────────┐
│              LLMClient Interface            │
│  - GenerateSQL()                           │
│  - GenerateContent()                        │
│  - SetKnowledge()                           │
│  - IsAvailable()                            │
└─────────────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────┐
│           OpenAIClient                      │
│  使用 langchaingo openai 包                │
│  支持所有 OpenAI 兼容格式的 API            │
└─────────────────────────────────────────────┘
                      │
          ┌───────────┼───────────┐
          ▼           ▼           ▼
      ZhipuAI     MiniMax     OpenAI
      (glm-*)    (M2-*)      (gpt-*)
```

## 代码使用

```go
// 创建 LLM 客户端
client, err := llm.NewLLMClient(
    "minimax",           // provider
    "your-api-key",      // apiKey
    "https://api.minimaxi.com",  // baseURL
    "M2-her",            // model
)
if err != nil {
    log.Fatal(err)
}

// 生成 SQL
sql, err := client.GenerateSQL(ctx, schema, question)
```

## 注意事项

1. **API Key 安全**: 不要将 API Key 提交到代码仓库，使用环境变量
2. **Base URL**: 不同 Provider 的 API 端点不同，请参考官方文档
3. **模型支持**: 某些功能可能因模型而异，请查看 Provider 文档
