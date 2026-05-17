package usecase

import "mmo/pkg/util"

func paginationOf(page, perPage int) util.Pagination {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	return util.Pagination{Page: page, PerPage: perPage}
}
