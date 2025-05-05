package main

import (
	"os"
	"shogo-ma/scratch-mcp-go/mcp"

	"log/slog"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "scratch-mcp-go",
		Usage: "MCPホストを起動します",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "config file path",
				EnvVars: []string{"MCP_CONFIG_PATH"},
			},
		},
		Action: func(c *cli.Context) error {
			ctx := c.Context

			configPath := c.String("config")
			if configPath == "" {
				slog.ErrorContext(ctx, "設定ファイルパスが指定されていません")
				return cli.Exit("設定ファイルパスが指定されていません", 1)
			}

			anthropicApiKey := os.Getenv("ANTHROPIC_API_KEY")
			if anthropicApiKey == "" {
				slog.ErrorContext(ctx, "Anthropic APIキーが指定されていません")
				return cli.Exit("Anthropic APIキーが指定されていません", 1)
			}

			host, err := mcp.NewHost(ctx, anthropicApiKey, configPath)
			if err != nil {
				slog.ErrorContext(ctx, "ホストの作成に失敗しました", slog.Any("error", err))
				return cli.Exit("ホストの作成に失敗しました", 1)
			}

			if err := host.Start(ctx); err != nil {
				slog.ErrorContext(ctx, "ホストの起動に失敗しました", slog.Any("error", err))
				return cli.Exit("ホストの起動に失敗しました", 1)
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		slog.Error("アプリケーションエラー", slog.Any("error", err))
		os.Exit(1)
	}
}
