//go:build e2e
// +build e2e

package e2e

import (
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLICommands(t *testing.T) {
	// Skip if not running E2E tests
	if testing.Short() {
		t.Skip("Skipping E2E test")
	}

	// Build the binary for testing
	cmd := exec.Command("go", "build", "-o", "cloudrecon-test", "./cmd/cloudrecon")
	err := cmd.Run()
	require.NoError(t, err)

	tests := []struct {
		name     string
		args     []string
		expected int // expected exit code
	}{
		{
			name:     "Help command",
			args:     []string{"--help"},
			expected: 0,
		},
		{
			name:     "Version command",
			args:     []string{"--version"},
			expected: 0,
		},
		{
			name:     "Discover help",
			args:     []string{"discover", "--help"},
			expected: 0,
		},
		{
			name:     "Query help",
			args:     []string{"query", "--help"},
			expected: 0,
		},
		{
			name:     "Export help",
			args:     []string{"export", "--help"},
			expected: 0,
		},
		{
			name:     "Ask help",
			args:     []string{"ask", "--help"},
			expected: 0,
		},
		{
			name:     "Status help",
			args:     []string{"status", "--help"},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("./cloudrecon-test", tt.args...)
			output, err := cmd.CombinedOutput()

			if tt.expected == 0 {
				assert.NoError(t, err, "Command failed: %s", string(output))
			} else {
				assert.Error(t, err, "Expected command to fail but it succeeded")
			}

			// Verify output contains expected content
			if tt.name == "Help command" {
				assert.Contains(t, string(output), "CloudRecon discovers and inventories cloud infrastructure")
			}
		})
	}
}

func TestDiscoveryWorkflow(t *testing.T) {
	// Skip if not running E2E tests
	if testing.Short() {
		t.Skip("Skipping E2E test")
	}

	// Build the binary for testing
	cmd := exec.Command("go", "build", "-o", "cloudrecon-test", "./cmd/cloudrecon")
	err := cmd.Run()
	require.NoError(t, err)

	// Test discovery with no providers (should not error)
	cmd = exec.Command("./cloudrecon-test", "discover", "--providers", "none")
	output, err := cmd.CombinedOutput()

	// Should not error even with no providers
	assert.NoError(t, err, "Discovery failed: %s", string(output))
}

func TestQueryWorkflow(t *testing.T) {
	// Skip if not running E2E tests
	if testing.Short() {
		t.Skip("Skipping E2E test")
	}

	// Build the binary for testing
	cmd := exec.Command("go", "build", "-o", "cloudrecon-test", "./cmd/cloudrecon")
	err := cmd.Run()
	require.NoError(t, err)

	// Test query with no data (should not error)
	cmd = exec.Command("./cloudrecon-test", "query", "SELECT * FROM resources LIMIT 1")
	output, err := cmd.CombinedOutput()

	// Should not error even with no data
	assert.NoError(t, err, "Query failed: %s", string(output))
}

func TestExportWorkflow(t *testing.T) {
	// Skip if not running E2E tests
	if testing.Short() {
		t.Skip("Skipping E2E test")
	}

	// Build the binary for testing
	cmd := exec.Command("go", "build", "-o", "cloudrecon-test", "./cmd/cloudrecon")
	err := cmd.Run()
	require.NoError(t, err)

	// Test export with no data (should not error)
	cmd = exec.Command("./cloudrecon-test", "export", "--format", "json", "--output", "/tmp/test-export.json")
	output, err := cmd.CombinedOutput()

	// Should not error even with no data
	assert.NoError(t, err, "Export failed: %s", string(output))
}

func TestTimeoutHandling(t *testing.T) {
	// Skip if not running E2E tests
	if testing.Short() {
		t.Skip("Skipping E2E test")
	}

	// Build the binary for testing
	cmd := exec.Command("go", "build", "-o", "cloudrecon-test", "./cmd/cloudrecon")
	err := cmd.Run()
	require.NoError(t, err)

	// Test with very short timeout
	cmd = exec.Command("./cloudrecon-test", "discover", "--timeout", "1s")

	// Set a timeout for the test
	done := make(chan error, 1)
	go func() {
		_, err := cmd.CombinedOutput()
		done <- err
	}()

	select {
	case err := <-done:
		// Command completed within timeout
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		// Command timed out, kill it
		cmd.Process.Kill()
		t.Fatal("Command did not respect timeout")
	}
}
