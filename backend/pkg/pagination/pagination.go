// Package pagination parses ?page=&page_size=&sort= consistently for every list endpoint.
package pagination

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

type Params struct {
	Page     int
	PageSize int
	Sort     string // safe "column ASC|DESC" string, or "" if not allowed
}

func (p Params) Offset() int { return (p.Page - 1) * p.PageSize }

type Meta struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

func (p Params) Meta(total int64) Meta { return Meta{Page: p.Page, PageSize: p.PageSize, Total: total} }

// Parse reads paging params. allowedSort is a whitelist of sortable columns;
// "?sort=-created_at" means created_at DESC. Anything not whitelisted falls back
// to defaultSort, which prevents SQL injection through the sort parameter.
func Parse(c *gin.Context, defaultSort string, allowedSort ...string) Params {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", strconv.Itoa(DefaultPageSize)))
	if size < 1 {
		size = DefaultPageSize
	}
	if size > MaxPageSize {
		size = MaxPageSize
	}

	sort := defaultSort
	if raw := c.Query("sort"); raw != "" {
		dir := "ASC"
		col := raw
		if strings.HasPrefix(raw, "-") {
			dir, col = "DESC", raw[1:]
		}
		for _, a := range allowedSort {
			if a == col {
				sort = col + " " + dir
				break
			}
		}
	}
	return Params{Page: page, PageSize: size, Sort: sort}
}
