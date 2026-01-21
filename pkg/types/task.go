package types

// TaskState represents the lifecycle state of a Task.
type TaskState string

const (
	// TaskStatePending indicates task is queued but not started.
	TaskStatePending TaskState = "pending"

	// TaskStateRunning indicates task is currently executing.
	TaskStateRunning TaskState = "running"

	// TaskStateCompleted indicates task finished successfully.
	TaskStateCompleted TaskState = "completed"

	// TaskStateFailed indicates task failed with an error.
	TaskStateFailed TaskState = "failed"

	// TaskStateCanceled indicates task was canceled.
	TaskStateCanceled TaskState = "canceled"
)

// Task represents an A2A task for longer-running operations.
type Task struct {
	// ID is the unique task identifier.
	ID string `json:"id"`

	// State is the current task state.
	State TaskState `json:"state"`

	// Progress is optional progress indicator (0-100).
	Progress *int `json:"progress,omitempty"`

	// Message contains task output or status messages.
	Message *Message `json:"message,omitempty"`

	// Metadata contains task-specific metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// TaskResult wraps the final result of a completed task.
type TaskResult struct {
	// ID is the task identifier.
	ID string `json:"id"`

	// State should be "completed" or "failed".
	State TaskState `json:"state"`

	// Result contains the task output on success.
	Result interface{} `json:"result,omitempty"`

	// Error contains error details on failure.
	Error *JSONRPCError `json:"error,omitempty"`
}

// IsTerminal returns true if the task state is final.
func (s TaskState) IsTerminal() bool {
	return s == TaskStateCompleted || s == TaskStateFailed || s == TaskStateCanceled
}

// IsSuccess returns true if the task completed successfully.
func (s TaskState) IsSuccess() bool {
	return s == TaskStateCompleted
}
