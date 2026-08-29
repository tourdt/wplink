// ci_policy_yaml 使用真正的 YAML 解析器校验 GitHub Actions 依赖安全策略。
// 它只读取 jobs.*.steps[] 的结构化字段，避免注释、块标量文本或 echo 命令伪装成有效配置。
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	expectedGoVersion       = "1.25.13"
	expectedSetupGoSteps    = 2
	dependencySecurityCheck = "make check-dependencies"
)

type workflow struct {
	Jobs map[string]job `yaml:"jobs"`
}

type job struct {
	Steps []step `yaml:"steps"`
}

type step struct {
	Uses            string         `yaml:"uses"`
	With            map[string]any `yaml:"with"`
	Run             string         `yaml:"run"`
	ContinueOnError bool           `yaml:"continue-on-error"`
}

type setupGoStep struct {
	JobName string
	Index   int
	Version string
}

func main() {
	workflowPath := flag.String("workflow", "../.github/workflows/ci.yml", "待校验的 GitHub Actions workflow 文件")
	flag.Parse()

	if err := validateWorkflow(*workflowPath); err != nil {
		fmt.Fprintf(os.Stderr, "CI 依赖安全策略校验失败：%v\n", err)
		os.Exit(1)
	}
	fmt.Println("CI 依赖安全 YAML 策略校验通过")
}

func validateWorkflow(workflowPath string) error {
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		return fmt.Errorf("读取 workflow 失败：%w", err)
	}

	var document workflow
	if err := yaml.Unmarshal(content, &document); err != nil {
		return fmt.Errorf("解析 workflow YAML 失败：%w", err)
	}

	setupGoSteps := make([]setupGoStep, 0, expectedSetupGoSteps)
	dependencySecurityRuns := 0
	for jobName, currentJob := range document.Jobs {
		for index, currentStep := range currentJob.Steps {
			if isSetupGoAction(currentStep.Uses) {
				version, err := setupGoVersion(currentStep.With)
				if err != nil {
					return fmt.Errorf("job %q 的第 %d 个 setup-go step 配置无效：%w", jobName, index+1, err)
				}
				setupGoSteps = append(setupGoSteps, setupGoStep{JobName: jobName, Index: index + 1, Version: version})
			}

			// 只接受 run 节点去除首尾空白后恰好等于统一门禁命令；echo 和多命令都不能通过。
			if strings.TrimSpace(currentStep.Run) == dependencySecurityCheck {
				if currentStep.ContinueOnError {
					return fmt.Errorf("job %q 的第 %d 个统一门禁 step 不得设置 continue-on-error", jobName, index+1)
				}
				dependencySecurityRuns++
			}
		}
	}

	if len(setupGoSteps) != expectedSetupGoSteps {
		return fmt.Errorf("actions/setup-go step 必须恰好 %d 个，实际为 %d 个", expectedSetupGoSteps, len(setupGoSteps))
	}
	for _, setupStep := range setupGoSteps {
		if setupStep.Version != expectedGoVersion {
			return fmt.Errorf("job %q 的第 %d 个 setup-go step 版本必须为 %s，实际为 %s", setupStep.JobName, setupStep.Index, expectedGoVersion, setupStep.Version)
		}
	}
	if dependencySecurityRuns != 1 {
		return fmt.Errorf("真实 run 节点中的 %q 必须恰好出现 1 次，实际为 %d 次", dependencySecurityCheck, dependencySecurityRuns)
	}

	return nil
}

func setupGoVersion(with map[string]any) (string, error) {
	value, ok := with["go-version"]
	if !ok {
		return "", fmt.Errorf("缺少 with.go-version")
	}
	version, ok := value.(string)
	if !ok || strings.TrimSpace(version) == "" {
		return "", fmt.Errorf("with.go-version 必须是非空字符串标量")
	}
	return strings.TrimSpace(version), nil
}

func isSetupGoAction(uses string) bool {
	actionName, _, ok := splitActionReference(uses)
	return ok && strings.EqualFold(actionName, "actions/setup-go")
}

func splitActionReference(uses string) (actionName string, ref string, ok bool) {
	trimmedUses := strings.TrimSpace(uses)
	separator := strings.Index(trimmedUses, "@")
	if separator <= 0 || separator == len(trimmedUses)-1 {
		return "", "", false
	}

	actionName = strings.TrimSpace(trimmedUses[:separator])
	ref = strings.TrimSpace(trimmedUses[separator+1:])
	// owner/repo 不能含空白；ref 必须非空且可保留 Git 合法的 @ 字符。
	if actionName == "" || ref == "" || strings.ContainsAny(actionName, " \t\r\n") ||
		strings.ContainsAny(ref, " \t\r\n") {
		return "", "", false
	}
	return actionName, ref, true
}
