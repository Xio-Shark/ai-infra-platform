package worker

import (
	"context"
	"testing"

	"ai-infra-platform/internal/model"
)

func TestBenchmarkExecutor_Name(t *testing.T) {
	executor := BenchmarkExecutor{}
	if executor.Name() != "benchmark" {
		t.Errorf("name: want %q, got %q", "benchmark", executor.Name())
	}
}

func TestBenchmarkExecutor_MissingTarget(t *testing.T) {
	executor := BenchmarkExecutor{}
	job := model.Job{
		Metadata: map[string]string{},
	}
	_, err := executor.Execute(context.Background(), job)
	if err == nil {
		t.Error("expected error for missing bench_target")
	}
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry(BenchmarkExecutor{}, ShellExecutor{})

	exec, err := registry.Get("benchmark")
	if err != nil {
		t.Fatalf("expected executor, got %v", err)
	}
	if exec.Name() != "benchmark" {
		t.Errorf("name: want benchmark, got %s", exec.Name())
	}

	_, err = registry.Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent executor")
	}
}

func TestShellExecutor_Name(t *testing.T) {
	executor := ShellExecutor{}
	if executor.Name() != "shell" {
		t.Errorf("name: want %q, got %q", "shell", executor.Name())
	}
}

func TestHTTPExecutor_Name(t *testing.T) {
	executor := HTTPExecutor{}
	if executor.Name() != "http" {
		t.Errorf("name: want %q, got %q", "http", executor.Name())
	}
}
