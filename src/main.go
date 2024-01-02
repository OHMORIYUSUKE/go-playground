package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/gin-gonic/gin"
)

type CodeRequest struct {
	Code     string `json:"code"`
	Input    string `json:"input"`
	Language string `json:"language"`
}

type ExecutionResult struct {
	Output   string `json:"output"`
	ExitCode int    `json:"exitCode"`
}

func main() {
	router := gin.Default()
	router.Static("/assets", "./assets")
	router.LoadHTMLGlob("templates/*.html")
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	router.POST("/", func(c *gin.Context) {
		result := handleExecute(c)
		c.HTML(http.StatusOK, "index.html", gin.H{
			"Code":     c.PostForm("code"),
			"Input":    c.PostForm("input"),
			"Language": c.PostForm("language"),
			"Output":   result.Output,
			"ExitCode": result.ExitCode,
		})
	})
	log.Fatal(router.Run(":8080"))
}

func handleExecute(c *gin.Context) ExecutionResult {

	var req CodeRequest

	// HTML フォームからデータを取得する
	req.Code = c.PostForm("code")
	req.Language = c.PostForm("language")
	req.Input = c.PostForm("input")

	ctx := context.Background()

	// コード書き込み
	err := writeStringToFile(c, req.Code, "./share/scripts/main"+getFileExtension(req.Language))
	if err != nil {
		return ExecutionResult{}
	}
	// コード書き込み
	err = writeStringToFile(c, req.Input, "./share/scripts/input.txt")
	if err != nil {
		return ExecutionResult{}
	}

	// Dockerクライアントの作成
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return ExecutionResult{}
	}

	// コンテナ名
	containerName := "go-playground-" + req.Language

	filename := "main" + getFileExtension(req.Language)
	// コマンド
	var langCmd []string
	switch req.Language {
	case "perl":
		langCmd = []string{"sh", "-c", "perl " + filename + " < input.txt"}
	case "ruby":
		langCmd = []string{"sh", "-c", "ruby " + filename + " < input.txt"}
	case "go":
		langCmd = []string{"sh", "-c", "go run " + filename + " < input.txt"}
	case "python":
		langCmd = []string{"sh", "-c", "python " + filename + " < input.txt"}
	case "julia":
		langCmd = []string{"sh", "-c", "julia " + filename + " < input.txt"}
	case "rust":
		langCmd = []string{"sh", "-c", "rustc " + filename + " && ./main"}
	case "swift":
		langCmd = []string{"sh", "-c", "swiftc " + filename + " && ./main"}
	}

	execResp, err := cli.ContainerExecCreate(ctx, containerName, types.ExecConfig{
		Cmd:          langCmd,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return ExecutionResult{}
	}

	// コンテナ実行結果の読み取り
	execAttachResp, err := cli.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{})
	if err != nil {
		return ExecutionResult{}
	}
	defer execAttachResp.Close()

	// 実行結果の読み込み
	outputBytes, err := io.ReadAll(execAttachResp.Reader)
	if err != nil {
		return ExecutionResult{}
	}

	output := removeNonPrintableChars(string(outputBytes))

	// コンテナ実行結果の詳細を取得
	execInspect, err := cli.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return ExecutionResult{}
	}

	result := ExecutionResult{
		Output:   output,
		ExitCode: execInspect.ExitCode,
	}

	return result
}

func getFileExtension(language string) string {
	switch language {
	case "perl":
		return ".pl"
	case "ruby":
		return ".rb"
	case "go":
		return ".go"
	case "python":
		return ".py"
	case "julia":
		return ".jl"
	case "rust":
		return ".rs"
	case "swift":
		return ".swift"
	default:
		return ""
	}
}

func removeNonPrintableChars(s string) string {
	reg := regexp.MustCompile("[[:cntrl:]]")
	return reg.ReplaceAllString(s, "")
}

func writeStringToFile(c *gin.Context, content, filename string) error {
	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		return err
	}
	return nil
}
