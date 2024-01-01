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
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
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

	// Perlスクリプトの実行
	// cmd := exec.Command("perl", "-e", req.Code)
	// output, err := cmd.Output()

	ctx := context.Background()

	// Dockerクライアントの作成
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}

	// コンテナ名
	containerName := "go-play-langs-perl"

	execResp, err := cli.ContainerExecCreate(ctx, containerName, types.ExecConfig{
		Cmd:          []string{"perl", "-e", "print(111"},
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		log.Fatalf("Error creating exec: %s", err)
	}

	// コンテナ実行結果の読み取り
	execAttachResp, err := cli.ContainerExecAttach(ctx, execResp.ID, types.ExecStartCheck{})
	if err != nil {
		log.Fatalf("Error attaching to exec: %s", err)
	}

	defer execAttachResp.Close()

	// 実行結果の読み込み
	outputBytes, err := io.ReadAll(execAttachResp.Reader)
	if err != nil {
		panic(err)
	}

	// 実行結果の整形
	output := string(outputBytes)
	// 制御文字や不可視文字を削除する
	output = removeNonPrintableChars(output)

	// 実行結果をJSON形式で返す
	result := ExecutionResult{}
	if err != nil {
		result.Error = err.Error()
	} else {
		result.Output = output
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
