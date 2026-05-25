# llm-translate

LibreTranslate API互換の翻訳サーバーです。[Hy-MT2](https://huggingface.co/collections/tencent/hy-mt2) などのLLMを翻訳エンジンとして使用し、MastodonなどLibreTranslate対応アプリから利用できます。

ライセンス：AGPL-3.0

## 特徴

- **LibreTranslate API互換** — Mastodonの翻訳機能をそのまま使用可能
- **Hy-MT2最適化** — 専用プロンプト形式と推論パラメーター対応
- **OpenAI互換バックエンド対応** — vLLM / SGLang / Ollama / LM Studio / OpenAI
- **38言語対応** — Hy-MT2の全対応言語を含む
- **Docker対応**

## 対応エンドポイント

| メソッド | パス | 説明 |
|---|---|---|
| `POST` | `/translate` | テキスト翻訳 |
| `GET` | `/languages` | 対応言語一覧 |
| `POST` | `/detect` | 言語検出 |
| `GET` | `/frontend/settings` | フロントエンド設定 |

## クイックスタート

### 1. モデルサーバーを起動

**vLLM (推奨)**
```bash
pip install vllm
vllm serve tencent/Hy-MT2-7B --tensor-parallel-size 1
```

**SGLang**
```bash
python3 -m sglang.launch_server --model tencent/Hy-MT2-7B --tp 1
```

**Ollama (汎用LLM)**
```bash
ollama serve
ollama pull llama3.2
```

### 2. 翻訳サーバーを起動

```bash
cp .env.example .env
# 必要に応じて .env を編集

go build -o translate .
./translate
```

または Docker Compose:

```bash
docker compose up -d
```

### 3. Mastodonで設定

管理画面 → **翻訳** → **LibreTranslate** を選択し、サーバーURLに `http://<ホスト>:5000` を入力。

## 設定

`.env.example` を `.env` にコピーして編集します。

```bash
cp .env.example .env
```

### 主要な設定項目

| 環境変数 | デフォルト | 説明 |
|---|---|---|
| `OPENAI_BASE_URL` | `http://localhost:8000/v1` | LLMバックエンドのURL |
| `OPENAI_API_KEY` | _(空)_ | APIキー（ローカルサーバーは不要） |
| `OPENAI_MODEL` | `tencent/Hy-MT2-7B` | モデル名 |
| `PORT` | `5000` | サーバーポート |
| `API_KEY` | _(空)_ | クライアント認証キー（空で無認証） |
| `LANGUAGES` | _(空=全言語)_ | 公開する言語コードをカンマ区切りで指定 |
| `REQUEST_TIMEOUT` | `60` | LLMリクエストのタイムアウト秒数 |

### 推論パラメーター

Hy-MT2の推奨値をモデルサイズに合わせて設定します。

| 環境変数 | 1.8B / 7B | 30B-A3B |
|---|---|---|
| `LLM_TEMPERATURE` | `0.7` | `0.7` |
| `LLM_TOP_P` | `0.6` | `1.0` |
| `LLM_TOP_K` | `20` | `-1` |
| `LLM_REPETITION_PENALTY` | `1.05` | `1.0` |
| `LLM_MAX_TOKENS` | `4096` | `4096` |

## バックエンド別の設定例

### Hy-MT2 (vLLM)

```env
OPENAI_BASE_URL=http://localhost:8000/v1
OPENAI_MODEL=tencent/Hy-MT2-7B
LLM_TEMPERATURE=0.7
LLM_TOP_P=0.6
LLM_TOP_K=20
LLM_REPETITION_PENALTY=1.05
```

### Ollama

```env
OPENAI_BASE_URL=http://localhost:11434/v1
OPENAI_API_KEY=ollama
OPENAI_MODEL=llama3.2
```

### OpenAI

```env
OPENAI_BASE_URL=https://api.openai.com/v1
OPENAI_API_KEY=sk-...
OPENAI_MODEL=gpt-4o-mini
```

## 対応言語

Hy-MT2の全対応言語 + 一般的な言語を含む38言語。

`ar` `bn` `bo` `cs` `de` `en` `es` `fa` `fr` `gu` `he` `hi` `id` `it` `ja` `kk` `km` `ko` `mn` `mr` `ms` `my` `nl` `pl` `pt` `ru` `ta` `te` `th` `tl` `tr` `ug` `uk` `ur` `vi` `yue` `zh` `zh-Hant`

`LANGUAGES=en,ja,zh,ko` のように環境変数で公開する言語を絞り込めます。

## API使用例

```bash
# 翻訳
curl -X POST http://localhost:5000/translate \
  -H "Content-Type: application/json" \
  -d '{"q": "Hello, world!", "source": "en", "target": "ja"}'
# → {"translatedText": "こんにちは、世界！"}

# 言語自動検出
curl -X POST http://localhost:5000/detect \
  -H "Content-Type: application/json" \
  -d '{"q": "Bonjour le monde"}'
# → [{"confidence": 0.9, "language": "fr"}]

# 対応言語一覧
curl http://localhost:5000/languages
```

ソースを `"auto"` にすると言語自動検出で翻訳します。

```bash
curl -X POST http://localhost:5000/translate \
  -H "Content-Type: application/json" \
  -d '{"q": "今日は良い天気ですね", "source": "auto", "target": "en"}'
```
