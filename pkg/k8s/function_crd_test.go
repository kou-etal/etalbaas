package k8s

import (
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestBuildFunctionCRDObject_GitSource(t *testing.T) {
	params := FunctionCRDParams{
		ProjectID:   "proj1",
		Name:        "my-func",
		DisplayName: "My Function",
		Kind:        "light-deployment",
		SourceType:  "git",
		SourceGit: &GitSourceCRD{
			Repo: "https://github.com/user/repo",
			Ref:  "main",
			Path: "/src",
		},
		RuntimePreset:       "python-3.11",
		RuntimeRequirements: []string{"requests==2.31.0"},
	}

	obj := buildFunctionCRDObject(params)

	if obj.GetName() != "my-func" {
		t.Errorf("name = %q, want %q", obj.GetName(), "my-func")
	}
	if obj.GetNamespace() != "project-proj1" {
		t.Errorf("namespace = %q, want %q", obj.GetNamespace(), "project-proj1")
	}

	spec, _, _ := unstructured.NestedMap(obj.Object, "spec")

	kind, _, _ := unstructured.NestedString(spec, "kind")
	if kind != "light-deployment" {
		t.Errorf("kind = %q, want %q", kind, "light-deployment")
	}

	sourceType, _, _ := unstructured.NestedString(spec, "source", "type")
	if sourceType != "git" {
		t.Errorf("source.type = %q, want %q", sourceType, "git")
	}

	repo, _, _ := unstructured.NestedString(spec, "source", "git", "repo")
	if repo != "https://github.com/user/repo" {
		t.Errorf("source.git.repo = %q", repo)
	}

	preset, _, _ := unstructured.NestedString(spec, "runtime", "preset")
	if preset != "python-3.11" {
		t.Errorf("runtime.preset = %q, want %q", preset, "python-3.11")
	}
}

func TestBuildFunctionCRDObject_InlineSource(t *testing.T) {
	params := FunctionCRDParams{
		ProjectID:   "proj2",
		Name:        "inline-func",
		DisplayName: "Inline Function",
		Kind:        "heavy-deployment",
		SourceType:  "inline",
		SourceInline: &InlineSourceCRD{
			Files:      map[string]string{"main.py": "print('hello')"},
			Entrypoint: "main.py",
		},
		RuntimePreset: "python-3.11",
	}

	obj := buildFunctionCRDObject(params)

	spec, _, _ := unstructured.NestedMap(obj.Object, "spec")
	sourceType, _, _ := unstructured.NestedString(spec, "source", "type")
	if sourceType != "inline" {
		t.Errorf("source.type = %q, want %q", sourceType, "inline")
	}

	entrypoint, _, _ := unstructured.NestedString(spec, "source", "inline", "entrypoint")
	if entrypoint != "main.py" {
		t.Errorf("source.inline.entrypoint = %q", entrypoint)
	}

	files, _, _ := unstructured.NestedStringMap(spec, "source", "inline", "files")
	if files["main.py"] != "print('hello')" {
		t.Errorf("source.inline.files[main.py] = %q", files["main.py"])
	}
}

func TestBuildFunctionCRDObject_WithTriggers(t *testing.T) {
	params := FunctionCRDParams{
		ProjectID:     "proj3",
		Name:          "trigger-func",
		DisplayName:   "Trigger Function",
		Kind:          "heavy-deployment",
		SourceType:    "git",
		SourceGit:     &GitSourceCRD{Repo: "https://github.com/user/repo"},
		RuntimePreset: "node-20",
		Triggers: []TriggerCRD{
			{
				Type: "DatabaseChange",
				DatabaseChange: &DatabaseChangeTriggerCRD{
					Table:      "users",
					Operations: []string{"INSERT", "UPDATE"},
				},
			},
			{
				Type: "Http",
				Http: &HttpTriggerCRD{
					Path:           "/invoke",
					Authentication: "apikey",
				},
			},
		},
		EnvVars: []EnvVarCRD{
			{Name: "API_KEY", SecretName: "my-secret"},
			{Name: "DEBUG", Value: "true"},
		},
	}

	obj := buildFunctionCRDObject(params)
	spec, _, _ := unstructured.NestedMap(obj.Object, "spec")

	triggers, _, _ := unstructured.NestedSlice(spec, "triggers")
	if len(triggers) != 2 {
		t.Fatalf("triggers len = %d, want 2", len(triggers))
	}

	envVars, _, _ := unstructured.NestedSlice(spec, "env")
	if len(envVars) != 2 {
		t.Fatalf("env len = %d, want 2", len(envVars))
	}
}

func TestBuildFunctionCRDObject_CustomDockerfile(t *testing.T) {
	params := FunctionCRDParams{
		ProjectID:         "proj4",
		Name:              "custom-func",
		DisplayName:       "Custom Dockerfile",
		Kind:              "heavy-job",
		SourceType:        "git",
		SourceGit:         &GitSourceCRD{Repo: "https://github.com/user/repo"},
		RuntimeDockerfile: "FROM python:3.11\nRUN pip install flask",
	}

	obj := buildFunctionCRDObject(params)
	spec, _, _ := unstructured.NestedMap(obj.Object, "spec")

	preset, _, _ := unstructured.NestedString(spec, "runtime", "preset")
	if preset != "custom" {
		t.Errorf("runtime.preset = %q, want %q", preset, "custom")
	}

	dockerfile, _, _ := unstructured.NestedString(spec, "runtime", "dockerfile")
	if dockerfile != "FROM python:3.11\nRUN pip install flask" {
		t.Errorf("runtime.dockerfile = %q, want %q", dockerfile, "FROM python:3.11\nRUN pip install flask")
	}
}

func TestParseStatus_Empty(t *testing.T) {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "etalbaas.io/v1alpha1",
			"kind":       "Function",
		},
	}

	status, err := parseStatus(obj)
	if err != nil {
		t.Fatal(err)
	}
	if status.Phase != "" {
		t.Errorf("phase = %q, want empty", status.Phase)
	}
}

func TestParseStatus_Building(t *testing.T) {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "etalbaas.io/v1alpha1",
			"kind":       "Function",
			"status": map[string]interface{}{
				"phase": "Building",
				"build": map[string]interface{}{
					"status": "Building",
				},
			},
		},
	}

	status, err := parseStatus(obj)
	if err != nil {
		t.Fatal(err)
	}
	if status.Phase != "Building" {
		t.Errorf("phase = %q, want %q", status.Phase, "Building")
	}
	if status.ImageRef != "" {
		t.Errorf("imageRef = %q, want empty", status.ImageRef)
	}
}

func TestParseStatus_Ready(t *testing.T) {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "etalbaas.io/v1alpha1",
			"kind":       "Function",
			"status": map[string]interface{}{
				"phase": "Ready",
				"build": map[string]interface{}{
					"status":               "Succeeded",
					"imageRef":             "zot.platform-system.svc/project-proj1/myfunc:42-abc",
					"imageDigest":          "sha256:deadbeef",
					"buildDurationSeconds": int64(45),
					"lastBuiltAt":          "2026-01-15T10:30:00Z",
				},
			},
		},
	}

	status, err := parseStatus(obj)
	if err != nil {
		t.Fatal(err)
	}
	if status.Phase != "Ready" {
		t.Errorf("phase = %q, want %q", status.Phase, "Ready")
	}
	if status.ImageRef != "zot.platform-system.svc/project-proj1/myfunc:42-abc" {
		t.Errorf("imageRef = %q", status.ImageRef)
	}
	if status.ImageDigest != "sha256:deadbeef" {
		t.Errorf("imageDigest = %q", status.ImageDigest)
	}
	if status.BuildDurationSec == nil || *status.BuildDurationSec != 45 {
		t.Errorf("buildDurationSec = %v", status.BuildDurationSec)
	}
	if status.LastBuiltAt == nil {
		t.Fatal("lastBuiltAt is nil")
	}
	expected := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	if !status.LastBuiltAt.Equal(expected) {
		t.Errorf("lastBuiltAt = %v, want %v", status.LastBuiltAt, expected)
	}
}
