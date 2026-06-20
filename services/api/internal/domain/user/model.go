// Package user 是用户与 Profile 领域模型
package user

import (
	"github.com/mediahub/api/internal/domain/common"

	"github.com/google/uuid"
)

// User 用户（家庭账号）
type User struct {
	common.BaseModel
	Username     string `gorm:"type:varchar(100);uniqueIndex;not null" json:"username"`
	PasswordHash string `gorm:"type:varchar(200);not null" json:"-"`
	DisplayName  string `gorm:"type:varchar(200)" json:"display_name,omitempty"`
	Email        string `gorm:"type:varchar(200)" json:"email,omitempty"`
	Role         string `gorm:"type:varchar(20);default:'member'" json:"role"` // admin | member
	AvatarURL    string `gorm:"type:text" json:"avatar_url,omitempty"`
	IsActive     bool   `gorm:"default:true" json:"is_active"`

	Profiles []Profile `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"profiles,omitempty"`
}

func (User) TableName() string { return "users" }

// Profile 家庭成员（观影身份）
type Profile struct {
	common.BaseModel
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Name      string    `gorm:"type:varchar(100);not null" json:"name"`
	AvatarURL string    `gorm:"type:text" json:"avatar_url,omitempty"`
	IsKid     bool      `gorm:"default:false" json:"is_kid"` // 儿童模式
	PinHash   string    `gorm:"type:varchar(200)" json:"-"`   // 家长锁 PIN
}

func (Profile) TableName() string { return "profiles" }

// IsAdmin 是否管理员
func (u *User) IsAdmin() bool {
	return u.Role == "admin"
}
