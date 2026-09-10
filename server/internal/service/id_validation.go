package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/tangwy-t/webmanager-server/internal/pkg/apperror"
)

// missingIDs returns the elements of requested that are absent from existing.
// Both slices must already be de-duplicated; the result is sorted for stable
// error messages.
func missingIDs(requested, existing []uint64) []uint64 {
	if len(requested) == 0 {
		return nil
	}
	have := make(map[uint64]struct{}, len(existing))
	for _, id := range existing {
		have[id] = struct{}{}
	}
	var missing []uint64
	for _, id := range requested {
		if _, ok := have[id]; !ok {
			missing = append(missing, id)
		}
	}
	sort.Slice(missing, func(i, j int) bool { return missing[i] < missing[j] })
	return missing
}

// dedupIDs removes duplicates while preserving first-seen order.
func dedupIDs(ids []uint64) []uint64 {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[uint64]struct{}, len(ids))
	out := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// validateIDsExist checks that every requested ID exists in the repository,
// returning a 400 BadRequest naming the missing IDs otherwise.
//
// fetch must return the subset of ids that actually exist (may be a subset or
// superset of requested — only requested-missing matters). The existence check
// is intentionally best-effort and runs outside the write transaction: if an
// ID is deleted concurrently after validation, the FK constraint on the join
// table still surfaces a loud error rather than a phantom row.
func validateIDsExist(ctx context.Context, kind string, requested []uint64, fetch func(context.Context, []uint64) ([]uint64, error)) error {
	requested = dedupIDs(requested)
	if len(requested) == 0 {
		return nil
	}
	existing, err := fetch(ctx, requested)
	if err != nil {
		return err
	}
	missing := missingIDs(requested, existing)
	if len(missing) == 0 {
		return nil
	}
	return apperror.BadRequest(fmt.Sprintf("%s不存在: %s", kind, formatIDs(missing)))
}

func formatIDs(ids []uint64) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = fmt.Sprintf("%d", id)
	}
	return strings.Join(parts, ", ")
}
