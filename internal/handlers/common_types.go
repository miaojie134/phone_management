package handlers

// PaginationInfo 定义了通用的分页信息结构
type PaginationInfo struct {
	TotalItems  int64 `json:"totalItems"`
	TotalPages  int64 `json:"totalPages"`
	CurrentPage int   `json:"currentPage"`
	PageSize    int   `json:"pageSize"`
}
