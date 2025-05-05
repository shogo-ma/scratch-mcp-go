# scratch-mcp-go

[Model Context Protocol (MCP)](https://modelcontextprotocol.io/) の理解を目的に書いたものです。toolsのみに絞ったMCPホストとMCPクライアントの実装をおこなっています。

## 設定ファイル例
`config.json` という名前で以下のような設定ファイルを用意します。

```json
{
  "mcpServers": {
    "example-server": {
      "command": "python3",
      "args": ["path/to/your/mcp_server.py"],
      "env": {
        "EXAMPLE_ENV": "value"
      }
    }
  }
}
```

## 環境変数
Anthropic APIキーを環境変数 `ANTHROPIC_API_KEY` で指定。

例: `.envrc` や `.env` ファイル
```
export ANTHROPIC_API_KEY=xxxxxxx
```

## 実行方法

```sh
go run main.go --config config.json
```
またはビルドして実行:
```sh
go build -o scratch-mcp-go
./scratch-mcp-go --config config.json
```

## 参考
- [Model Context Protocol 公式ドキュメント](https://modelcontextprotocol.io/)
- [Anthropic Claude API](https://docs.anthropic.com/claude)
- [最小限のMCP Host/Client/Serverをスクラッチで実装する](https://zenn.dev/razokulover/articles/9a0aee8ceb9f3f)
- [mcp-go](https://github.com/mark3labs/mcp-go)
- [mcphost](https://github.com/mark3labs/mcphost)
