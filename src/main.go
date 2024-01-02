package main

import (
	"context"
	"fmt"
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
	Language string `json:"language"`
}

type ExecutionResult struct {
	Output   string `json:"output"`
	ExitCode int    `json:"exitCode"`
}

func main() {
	router := gin.Default()
	router.POST("/execute", handleExecute)
	log.Fatal(router.Run(":8080"))
}
func handleExecute(c *gin.Context) {
	var req CodeRequest

	// JSONデコード
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()

	// コード書き込み
	err := writeStringToFile(c, req.Code, "./share/scripts/main"+getFileExtension(req.Language))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Dockerクライアントの作成
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// コンテナ名
	containerName := "go-playground-" + req.Language

	filename := "main" + getFileExtension(req.Language)
	var langCmd []string
	switch req.Language {
	case "perl":
		langCmd = []string{"perl", filename}
	case "ruby":
		langCmd = []string{"ruby", filename}
	case "go":
		langCmd = []string{"go", "run", filename}
	case "python":
		langCmd = []string{"python", filename}
	case "julia":
		langCmd = []string{"julia", filename}
	case "rust":
		langCmd = []string{"sh", "-c", "rustc " + filename + " && ./main"}
	case "swift":
		langCmd = []string{"sh", "-c", "swiftc " + filename + " && ./main"}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported language"})
		return
	}

	fmt.Println(langCmd)

	execResp, err := cli.ContainerExecCreate(ctx, containerName, types.ExecConfig{
		Cmd:          langCmd,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// コンテナ実行結果の読み取り
	execAttachResp, err := cli.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer execAttachResp.Close()

	// 実行結果の読み込み
	outputBytes, err := io.ReadAll(execAttachResp.Reader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 実行結果の整形
	output := string(outputBytes)
	// 制御文字や不可視文字を削除する
	output = removeNonPrintableChars(output)

	// コンテナ実行結果の詳細を取得
	execInspect, err := cli.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 実行結果をJSON形式で返す
	result := ExecutionResult{
		Output:   output,
		ExitCode: execInspect.ExitCode,
	}

	c.JSON(http.StatusOK, result)
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

// 制御文字や不可視文字を削除する関数
func removeNonPrintableChars(s string) string {
	reg := regexp.MustCompile("[[:cntrl:]]")
	return reg.ReplaceAllString(s, "")
}

func writeStringToFile(c *gin.Context, content, filename string) error {
	// ファイルに文字列を書き込む
	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		return err
	}

	return nil
}
