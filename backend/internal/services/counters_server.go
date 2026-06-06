package services

import (
	"context"
	"errors"
	"fmt"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/domain"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/storage/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CounterService allocates gapless sequence numbers per (tenant, scope, year).
// Caller passes the request's tenant-scoped tx; allocation must run inside
// the same transaction as the consumer (e.g., creating a project) so a
// rollback rolls the counter back too.
type CounterService struct{}

func NewCounterService() *CounterService {
	return &CounterService{}
}

// Allocate returns the next value for (tenant, scope, year) and increments
// the counter. The two operations happen under a single FOR UPDATE lock,
// preventing two concurrent transactions from observing the same value.
//
// scope must be one of the values allowed by the DB CHECK constraint
// ('project', 'invoice', 'quote').
func (s *CounterService) Allocate(
	ctx context.Context,
	q *gen.Queries,
	tenantId uuid.UUID,
	scope string,
	year int32) (int64, error) {
	// Ensure the row exists. If two callers race here, ON CONFLICT DO NOTHING
	// means only one row gets created; both proceed to FOR UPDATE on it.
	if err := q.InsertCounterIfMissing(ctx, gen.InsertCounterIfMissingParams{
		TenantID: tenantId,
		Scope:    scope,
		Year:     year,
	}); err != nil {
		return 0, domain.ErrInternal(fmt.Errorf("ensure counter row failed : %w", err))
	}

	counter, err := q.GetCounterForUpdate(ctx, gen.GetCounterForUpdateParams{
		TenantID: tenantId,
		Scope:    scope,
		Year:     year,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Shouldn't happen — we just inserted. But defensive.
			return 0, domain.ErrInternal(fmt.Errorf("counter row missing after for this scope, year, tenant "))
		}
		return 0, domain.ErrInternal(err)
	}

	if err := q.IncrementCounter(ctx, gen.IncrementCounterParams{
		TenantID: tenantId,
		Scope:    scope,
		Year:     year,
	}); err != nil {
		return 0, domain.ErrInternal(err)
	}

	return counter.NextValue, nil

}
