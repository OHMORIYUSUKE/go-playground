package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"

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
		language := c.Query("language")
		if language == "" {
			language = "perl"
		}
		code := getSampleCode(language)
		c.HTML(http.StatusOK, "index.html", gin.H{
			"Code":     code,
			"Input":    "World !!",
			"Language": language,
		})
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
		langCmd = []string{"sh", "-c", "rustc " + filename + " && ./main < input.txt"}
	case "swift":
		langCmd = []string{"sh", "-c", "swiftc " + filename + " && ./main < input.txt"}
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

	output := string(outputBytes)

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

func writeStringToFile(c *gin.Context, content, filename string) error {
	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		return err
	}
	return nil
}

func getSampleCode(language string) string {
	switch language {
	case "perl":
		return `#!/usr/bin/perl

use strict;
use warnings;

my $input = <STDIN>;
chomp($input);

print "Hello $input";
`
	case "ruby":
		return `user_input = gets.chomp

greeting = "Hello #{user_input}"

puts greeting`
	case "go":
		return `package main

import "fmt"

func main() {
var input string
fmt.Scanln(&input)

fmt.Printf("Hello %s", input)
}`
	case "kotlin":
		return `fun main() {
val input = readLine()

println("Hello $input")
}`
	case "julia":
		return `input = readline()

println("Hello $input")`
	case "rust":
		return `use std::io;

fn main() {
let mut input = String::new();

io::stdin().read_line(&mut input).expect("Failed to read line");
let input = input.trim();

println!("Hello {}", input);
}`
	case "python":
		return `
user_input = input()

greeting = f"Hello {user_input}"

print(greeting)
`
	case "swift":
		return `import Foundation

if let input = readLine() {
print("Hello \(input)")
}
`
	}
	return ""
}
