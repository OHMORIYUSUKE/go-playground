package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"regexp"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type CodeRequest struct {
	Code string `json:"code"`
}

type ExecutionResult struct {
	Output   string `json:"output"`
	ExitCode int    `json:"exitCode"`
}

func main() {
	http.HandleFunc("/execute", handleExecute)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleExecute(w http.ResponseWriter, r *http.Request) {
	// リクエストボディの読み取り
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// JSONデコード
	var req CodeRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	// Dockerクライアントの作成
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// コンテナ名
	containerName := "go-play-langs-perl"

	execResp, err := cli.ContainerExecCreate(ctx, containerName, types.ExecConfig{
		Cmd:          []string{"perl", "-e", req.Code},
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// コンテナ実行結果の読み取り
	execAttachResp, err := cli.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer execAttachResp.Close()

	// 実行結果の読み込み
	outputBytes, err := io.ReadAll(execAttachResp.Reader)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 実行結果の整形
	output := string(outputBytes)
	// 制御文字や不可視文字を削除する
	output = removeNonPrintableChars(output)

	// コンテナ実行結果の詳細を取得
	execInspect, err := cli.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 実行結果をJSON形式で返す
	result := ExecutionResult{
		Output:   output,
		ExitCode: execInspect.ExitCode,
	}

	response, err := json.Marshal(result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

// 制御文字や不可視文字を削除する関数
func removeNonPrintableChars(s string) string {
	reg := regexp.MustCompile("[[:cntrl:]]")
	return reg.ReplaceAllString(s, "")
}
