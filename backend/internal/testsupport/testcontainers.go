//go:build integration

package testsupport

import (
	"strings"
	"testing"
)

func SkipIfDockerUnavailable(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		return
	}

	message := err.Error()
	if strings.Contains(message, "failed to create Docker provider") ||
		strings.Contains(message, "Cannot connect to the Docker daemon") ||
		strings.Contains(message, "rootless Docker is not supported on Windows") {
		t.Skipf("testcontainers Docker provider is unavailable: %v", err)
	}
}
