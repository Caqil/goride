package utils

import (
	"math"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type PaginationParams struct {
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"page_size" form:"page_size"`
	Sort     string `json:"sort" form:"sort"`
	Order    string `json:"order" form:"order"`
	Search   string `json:"search" form:"search"`
}

type PaginationMeta struct {
	Page         int   `json:"page"`
	PageSize     int   `json:"page_size"`
	Total        int64 `json:"total"`
	TotalPages   int   `json:"total_pages"`
	HasNext      bool  `json:"has_next"`
	HasPrevious  bool  `json:"has_previous"`
	NextPage     *int  `json:"next_page,omitempty"`
	PreviousPage *int  `json:"previous_page,omitempty"`
}

func GetPaginationParams(c *gin.Context) *PaginationParams {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", strconv.Itoa(DefaultPageSize)))
	sort := c.DefaultQuery("sort", "created_at")
	order := c.DefaultQuery("order", "desc")
	search := c.Query("search")

	// Validate page
	if page < 1 {
		page = 1
	}

	// Validate page size
	if pageSize < MinPageSize {
		pageSize = MinPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	// Validate order
	if order != "asc" && order != "desc" {
		order = "desc"
	}

	return &PaginationParams{
		Page:     page,
		PageSize: pageSize,
		Sort:     sort,
		Order:    order,
		Search:   search,
	}
}

func (p *PaginationParams) GetSkip() int {
	return (p.Page - 1) * p.PageSize
}

func (p *PaginationParams) GetLimit() int {
	return p.PageSize
}

func (p *PaginationParams) GetSortOptions() *options.FindOptions {
	opts := options.Find()
	opts.SetSkip(int64(p.GetSkip()))
	opts.SetLimit(int64(p.GetLimit()))

	// Set sort order
	sortOrder := 1
	if p.Order == "desc" {
		sortOrder = -1
	}
	opts.SetSort(bson.D{{Key: p.Sort, Value: sortOrder}})

	return opts
}

func (p *PaginationParams) GetSearchFilter(fields []string) bson.M {
	if p.Search == "" || len(fields) == 0 {
		return bson.M{}
	}

	var orConditions []bson.M
	for _, field := range fields {
		orConditions = append(orConditions, bson.M{
			field: bson.M{"$regex": p.Search, "$options": "i"},
		})
	}

	return bson.M{"$or": orConditions}
}

func CreatePaginationMeta(params *PaginationParams, total int64) *PaginationMeta {
	totalPages := int(math.Ceil(float64(total) / float64(params.PageSize)))

	meta := &PaginationMeta{
		Page:        params.Page,
		PageSize:    params.PageSize,
		Total:       total,
		TotalPages:  totalPages,
		HasNext:     params.Page < totalPages,
		HasPrevious: params.Page > 1,
	}

	if meta.HasNext {
		nextPage := params.Page + 1
		meta.NextPage = &nextPage
	}

	if meta.HasPrevious {
		previousPage := params.Page - 1
		meta.PreviousPage = &previousPage
	}

	return meta
}
