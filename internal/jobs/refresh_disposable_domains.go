package jobs

import (
	"context"

	"github.com/adehusnim37/lihatin-go/internal/pkg/disposable"
)

// RefreshDisposableDomainsJob refreshes disposable email domains from source.
type RefreshDisposableDomainsJob struct{}

// NewRefreshDisposableDomainsJob creates job instance.
func NewRefreshDisposableDomainsJob() *RefreshDisposableDomainsJob {
	return &RefreshDisposableDomainsJob{}
}

// Name returns job name.
func (j *RefreshDisposableDomainsJob) Name() string {
	return "refresh-disposable-email-domains"
}

// Schedule runs every 12 hours.
func (j *RefreshDisposableDomainsJob) Schedule() string {
	return "0 0 */12 * * *"
}

// Run executes job.
func (j *RefreshDisposableDomainsJob) Run(ctx context.Context) error {
	policy := disposable.Global()
	if policy == nil {
		return nil
	}
	return policy.RefreshFromSource(ctx)
}
