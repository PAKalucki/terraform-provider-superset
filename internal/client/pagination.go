package client

import (
	"context"
	"fmt"
)

const defaultMaxPaginationPages = 10000

func validatePagination(ctx context.Context, page int, maxPages int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if maxPages <= 0 {
		maxPages = defaultMaxPaginationPages
	}

	if page >= maxPages {
		return fmt.Errorf("superset API pagination exceeded %d pages", maxPages)
	}

	return nil
}

func (c *Client) paginationLimit() int {
	if c == nil {
		return defaultMaxPaginationPages
	}

	if c.maxPaginationPages > 0 {
		return c.maxPaginationPages
	}

	return defaultMaxPaginationPages
}
