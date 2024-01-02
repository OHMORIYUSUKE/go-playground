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
	Code string `json:"code"`
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
	err := writeStringToFile(c, req.Code, "./share/scripts/main.pl")
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
	containerName := "go-play-langs-perl"

	execResp, err := cli.ContainerExecCreate(ctx, containerName, types.ExecConfig{
		Cmd:          []string{"perl", "main.pl"},
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
