package util

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

type Pagination struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

func (p Pagination) Offset() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.PerPage
}

func (p Pagination) Limit() int {
	return p.PerPage
}

func ParsePagination(c *gin.Context) Pagination {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	return Pagination{Page: page, PerPage: perPage}
}
