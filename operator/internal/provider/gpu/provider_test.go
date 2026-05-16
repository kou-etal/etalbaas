package gpu

import "testing"

func TestJobState_IsTerminal(t *testing.T) {
	tests := []struct {
		state    JobState
		terminal bool
	}{
		{JobStateQueued, false},
		{JobStateRunning, false},
		{JobStateCompleted, true},
		{JobStateFailed, true},
		{JobStateCancelled, true},
	}
	for _, tt := range tests {
		if got := tt.state.IsTerminal(); got != tt.terminal {
			t.Errorf("JobState(%q).IsTerminal() = %v, want %v", tt.state, got, tt.terminal)
		}
	}
}

func TestGPUJob_Validate(t *testing.T) {
	tests := []struct {
		name    string
		job     GPUJob
		wantErr bool
	}{
		{
			name:    "valid",
			job:     GPUJob{Image: "myimage:latest", GPUType: "H100"},
			wantErr: false,
		},
		{
			name:    "empty image",
			job:     GPUJob{GPUType: "H100"},
			wantErr: true,
		},
		{
			name:    "empty gpu type",
			job:     GPUJob{Image: "myimage:latest"},
			wantErr: true,
		},
		{
			name:    "both empty",
			job:     GPUJob{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.job.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
