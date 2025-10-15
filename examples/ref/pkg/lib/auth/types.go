package auth

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	EmailCT      []byte
	EmailIV      []byte
	EmailTag     []byte
	EmailLookup  []byte
	PasswordHash []byte
	PasswordSalt []byte
	MFASecretCT  []byte
	Status       UserStatus
	CreatedAt    time.Time
}

type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusSuspended UserStatus = "suspended"
	UserStatusDeleted   UserStatus = "deleted"
)

type Role struct {
	ID          uuid.UUID
	Name        string
	Permissions []string
}

type Grant struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	GrantType GrantType
	Value     string
	Scope     Scope
	ExpiresAt *time.Time
}

type GrantType string

const (
	GrantTypeRole       GrantType = "role"
	GrantTypePermission GrantType = "permission"
)

type Scope struct {
	Type string
	ID   string
}

type ResourcePolicy struct {
	ID      string
	Type    string
	Version int
	Actions map[string]PolicyRule
}

type PolicyRule struct {
	AnyOf []string
	AllOf []string
}

type TokenClaims struct {
	Subject      string            `json:"sub"`
	SessionID    string            `json:"sid"`
	Audience     string            `json:"aud"`
	Context      map[string]string `json:"ctx"`
	ExpiresAt    int64             `json:"exp"`
	AuthzVersion int               `json:"authz_ver"`
}

type EmailSubscription struct {
	UserID      *uuid.UUID
	EmailCT     []byte
	EmailLookup []byte
	Consent     ConsentRecord
	ConfirmedAt *time.Time
}

type ConsentRecord struct {
	Type      string
	Scope     string
	Timestamp time.Time
	SourceIP  string
}
