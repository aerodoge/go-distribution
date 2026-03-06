package model

import (
	"time"
)

// 数据库结构
type UserRefer struct {
	ID        int64     `db:"id"`
	UserID    int64     `db:"user_id"`
	ParentID  int64     `db:"parent_id"`
	CreatedAt time.Time `db:"created_at"`
}

// 树形结构
type UserTreeNode struct {
	UserID   int64           `json:"user_id"`
	ParentID int64           `json:"parent_id"`
	Children []*UserTreeNode `json:"children"`
}
