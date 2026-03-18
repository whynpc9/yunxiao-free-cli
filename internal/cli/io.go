package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

func PromptLine(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			return strings.TrimSpace(line), nil
		}
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func PromptSelect(prompt string, options []string) (int, error) {
	if len(options) == 0 {
		return -1, fmt.Errorf("no options")
	}
	for i, option := range options {
		fmt.Fprintf(os.Stderr, "[%d] %s\n", i+1, option)
	}
	for {
		line, err := PromptLine(prompt)
		if err != nil {
			return -1, err
		}
		n, err := strconv.Atoi(line)
		if err != nil || n < 1 || n > len(options) {
			fmt.Fprintln(os.Stderr, "请输入有效序号")
			continue
		}
		return n - 1, nil
	}
}

func PrintJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
