package runner

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func pullDockerImage(ctx context.Context, image string, maxAttempts int, retryDelay time.Duration) error {
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("docker pull cancelled: %w", err)
		}
		out, err := exec.CommandContext(ctx, "docker", "pull", image).CombinedOutput()
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return fmt.Errorf("docker pull cancelled: %w", ctx.Err())
		}
		log.Warnf("runner: docker pull %s attempt %d/%d failed: %s", image, attempt, maxAttempts, strings.TrimSpace(string(out)))
		if attempt == maxAttempts {
			return fmt.Errorf("docker pull %s: %w - %s", image, err, strings.TrimSpace(string(out)))
		}
		timer := time.NewTimer(retryDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("docker pull cancelled: %w", ctx.Err())
		case <-timer.C:
		}
		retryDelay *= 2
	}
	return fmt.Errorf("docker pull %s: max attempts must be positive", image)
}
