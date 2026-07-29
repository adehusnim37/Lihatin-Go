// Package jobs provides a centralized cron job scheduler infrastructure.
// It allows easy registration and management of periodic background tasks.
//
// Usage:
//
//  1. Create a new job by implementing the Job interface in a new file (e.g., jobs/your_job.go)
//  2. Register the job in RegisterJobs() function in registry.go
//  3. Initialize and start the scheduler in main.go
//
// Example:
//
//	scheduler := jobs.NewScheduler(db)
//	scheduler.Start()
//	defer scheduler.Stop()
package jobs

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/adehusnim37/lihatin-go/internal/pkg/logger"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// Job represents a scheduled job that can be registered with the scheduler.
type Job interface {
	// Name returns the unique name of the job for logging purposes
	Name() string

	// Schedule returns the cron expression (e.g., "@hourly", "0 */5 * * *")
	// See https://pkg.go.dev/github.com/robfig/cron/v3 for format details
	Schedule() string

	// Run executes the job logic
	Run(ctx context.Context) error
}

// Scheduler manages all cron jobs
type Scheduler struct {
	cron   *cron.Cron
	db     *gorm.DB
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	jobs   []Job
}

// NewScheduler creates a new scheduler instance
func NewScheduler(db *gorm.DB) (*Scheduler, error) {
	ctx, cancel := context.WithCancel(context.Background())

	if err := ctx.Err(); err != nil {
		cancel()
		return nil, err
	}

	return &Scheduler{
		cron:   cron.New(cron.WithSeconds()), // Support second-level precision
		db:     db,
		ctx:    ctx,
		cancel: cancel,
		jobs:   make([]Job, 0),
	}, nil
}

// Register adds a job to the scheduler
func (s *Scheduler) Register(job Job) error {
	rawSchedule := job.Schedule()
	schedule := normalizeScheduleSpec(rawSchedule)

	_, err := s.cron.AddFunc(schedule, func() {
		s.wg.Add(1)
		defer s.wg.Done()

		logger.Logger.Info("Starting cron job", "job", job.Name())

		if err := job.Run(s.ctx); err != nil {
			logger.Logger.Error("Cron job failed",
				"job", job.Name(),
				"error", err.Error(),
			)
		} else {
			logger.Logger.Info("Cron job completed successfully", "job", job.Name())
		}
	})

	if err != nil {
		return fmt.Errorf("invalid schedule %q (raw: %q): %w", schedule, rawSchedule, err)
	}

	s.jobs = append(s.jobs, job)
	if schedule == rawSchedule {
		logger.Logger.Info("Registered cron job",
			"job", job.Name(),
			"schedule", schedule,
		)
	} else {
		logger.Logger.Info("Registered cron job",
			"job", job.Name(),
			"schedule", schedule,
			"raw_schedule", rawSchedule,
		)
	}

	return nil
}

func normalizeScheduleSpec(spec string) string {
	normalized := strings.TrimSpace(spec)
	normalized = strings.ReplaceAll(normalized, `\"`, `"`)
	normalized = strings.ReplaceAll(normalized, `\'`, `'`)

	for len(normalized) >= 2 {
		first := normalized[0]
		last := normalized[len(normalized)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			normalized = strings.TrimSpace(normalized[1 : len(normalized)-1])
			continue
		}
		break
	}

	return normalized
}

// RegisterMany adds multiple jobs and returns an error with job context when one fails.
func (s *Scheduler) RegisterMany(jobs ...Job) error {
	for _, job := range jobs {
		if err := s.Register(job); err != nil {
			return fmt.Errorf("register job %q failed: %w", job.Name(), err)
		}
	}

	return nil
}

// Start begins executing all registered jobs
func (s *Scheduler) Start() {
	logger.Logger.Info("Starting cron scheduler", "total_jobs", len(s.jobs))
	s.cron.Start()
}

// Stop gracefully shuts down the scheduler
func (s *Scheduler) Stop() {
	logger.Logger.Info("Stopping cron scheduler...")

	// Signal all jobs to stop
	s.cancel()

	// Stop accepting new jobs
	ctx := s.cron.Stop()

	// Wait for running jobs to complete
	<-ctx.Done()
	s.wg.Wait()

	logger.Logger.Info("Cron scheduler stopped")
}

// GetDB returns the database connection for jobs to use
func (s *Scheduler) GetDB() *gorm.DB {
	return s.db
}
