package worker

import (
	"encoding/json"
	"fmt"

	"ai-infra-platform/internal/model"
)

func buildJobManifest(job model.Job, namespace string) (string, error) {
	if len(job.Command) == 0 {
		return "", fmt.Errorf("job %s has empty command", job.ID)
	}
	if namespace == "" {
		namespace = "default"
	}
	manifest, err := json.MarshalIndent(map[string]any{
		"apiVersion": "batch/v1",
		"kind":       "Job",
		"metadata": map[string]any{
			"name":      job.Name,
			"namespace": namespace,
			"labels": map[string]string{
				"job_id":   job.ID,
				"job_type": string(job.Type),
			},
		},
		"spec": map[string]any{
			"template": map[string]any{
				"spec": map[string]any{
					"restartPolicy": "Never",
					"containers": []map[string]any{{
						"name":      "worker",
						"image":     job.ImageTag,
						"command":   job.Command,
						"env":       toK8sEnv(job.Environment),
						"resources": toK8sResources(job.ResourceSpec),
					}},
				},
			},
		},
	}, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal k8s manifest: %w", err)
	}
	return string(manifest), nil
}

func toK8sEnv(values map[string]string) []map[string]string {
	items := make([]map[string]string, 0, len(values))
	for key, value := range values {
		items = append(items, map[string]string{"name": key, "value": value})
	}
	return items
}

func toK8sResources(spec model.ResourceSpec) map[string]any {
	requests := map[string]any{}
	if spec.CPU != "" {
		requests["cpu"] = spec.CPU
	}
	if spec.Memory != "" {
		requests["memory"] = spec.Memory
	}
	if spec.GPU > 0 {
		requests["nvidia.com/gpu"] = spec.GPU
	}
	return map[string]any{"requests": requests}
}
