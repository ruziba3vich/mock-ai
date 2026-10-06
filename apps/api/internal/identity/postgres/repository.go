package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/own/mock-ai/apps/api/internal/identity/domain"
	db "github.com/own/mock-ai/apps/api/internal/platform/postgres/sqlc"
)

var (
	ErrNotFound = errors.New("identity record not found")
	ErrConflict = errors.New("identity record already exists")
)

type Repository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool, queries: db.New(pool)}
}

func (r *Repository) RegisterEmail(ctx context.Context, name, email, passwordHash string) (domain.User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.User{}, fmt.Errorf("begin email registration: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := r.queries.WithTx(tx)

	user, err := q.CreateUser(ctx, db.CreateUserParams{
		DisplayName: name,
		Email:       text(email),
	})
	if err != nil {
		return domain.User{}, classify(err)
	}
	_, err = q.CreateIdentity(ctx, db.CreateIdentityParams{
		UserID:          user.ID,
		Provider:        db.IdentityProviderEmail,
		ProviderSubject: email,
		ProviderEmail:   text(email),
		Profile:         []byte(`{}`),
	})
	if err != nil {
		return domain.User{}, classify(err)
	}
	if err := q.CreatePasswordCredential(ctx, db.CreatePasswordCredentialParams{
		UserID:       user.ID,
		PasswordHash: passwordHash,
	}); err != nil {
		return domain.User{}, classify(err)
	}
	if err := q.AssignUserRole(ctx, db.AssignUserRoleParams{UserID: user.ID, Role: db.UserRoleStudent}); err != nil {
		return domain.User{}, fmt.Errorf("assign student role: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.User{}, fmt.Errorf("commit email registration: %w", err)
	}
	return toDomain(user, []db.UserRole{db.UserRoleStudent}), nil
}

func (r *Repository) PasswordLogin(ctx context.Context, email string) (domain.User, string, error) {
	record, err := r.queries.GetPasswordLogin(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, "", ErrNotFound
	}
	if err != nil {
		return domain.User{}, "", fmt.Errorf("get password login: %w", err)
	}
	roles, err := r.queries.ListUserRoles(ctx, record.ID)
	if err != nil {
		return domain.User{}, "", fmt.Errorf("list user roles: %w", err)
	}
	user := domain.User{
		ID:            uuidString(record.ID),
		DisplayName:   record.DisplayName,
		AvatarURL:     textString(record.AvatarUrl),
		Email:         textString(record.Email),
		EmailVerified: record.EmailVerifiedAt.Valid,
		CreatedAt:     record.CreatedAt.Time,
		Roles:         roleStrings(roles),
	}
	return user, record.PasswordHash, nil
}

func (r *Repository) FindOrCreateExternal(ctx context.Context, profile domain.ExternalProfile) (domain.User, bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.User{}, false, fmt.Errorf("begin external login: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := r.queries.WithTx(tx)
	provider := db.IdentityProvider(profile.Provider)
	created := false

	user, err := q.GetUserByIdentity(ctx, db.GetUserByIdentityParams{
		Provider:        provider,
		ProviderSubject: profile.Subject,
	})
	if err == nil {
		user, err = q.UpdateExternalProfile(ctx, db.UpdateExternalProfileParams{
			ID:          user.ID,
			DisplayName: profile.DisplayName,
			AvatarUrl:   text(profile.AvatarURL),
		})
		if err != nil {
			return domain.User{}, false, fmt.Errorf("update external profile: %w", err)
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, false, fmt.Errorf("find external identity: %w", err)
	} else {
		user, err = r.resolveExternalUser(ctx, q, profile)
		if err != nil {
			return domain.User{}, false, err
		}
		_, err = q.CreateIdentity(ctx, db.CreateIdentityParams{
			UserID:          user.ID,
			Provider:        provider,
			ProviderSubject: profile.Subject,
			ProviderEmail:   text(profile.Email),
			Profile:         profile.RawProfile,
		})
		if err != nil {
			return domain.User{}, false, classify(err)
		}
		created = true
	}

	if err := q.AssignUserRole(ctx, db.AssignUserRoleParams{UserID: user.ID, Role: db.UserRoleStudent}); err != nil {
		return domain.User{}, false, fmt.Errorf("assign student role: %w", err)
	}
	roles, err := q.ListUserRoles(ctx, user.ID)
	if err != nil {
		return domain.User{}, false, fmt.Errorf("list user roles: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.User{}, false, fmt.Errorf("commit external login: %w", err)
	}
	return toDomain(user, roles), created, nil
}

func (r *Repository) resolveExternalUser(ctx context.Context, q *db.Queries, profile domain.ExternalProfile) (db.User, error) {
	if profile.Email != "" && profile.EmailVerified {
		user, err := q.GetUserByEmail(ctx, profile.Email)
		if err == nil {
			return q.MarkEmailVerified(ctx, user.ID)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return db.User{}, fmt.Errorf("find user by verified email: %w", err)
		}
	}
	params := db.CreateVerifiedUserParams{
		DisplayName: profile.DisplayName,
		AvatarUrl:   text(profile.AvatarURL),
		Email:       text(profile.Email),
	}
	if profile.EmailVerified {
		user, err := q.CreateVerifiedUser(ctx, params)
		if err != nil {
			return db.User{}, classify(err)
		}
		return user, nil
	}
	user, err := q.CreateUser(ctx, db.CreateUserParams{
		DisplayName: profile.DisplayName,
		AvatarUrl:   text(profile.AvatarURL),
		Email:       text(profile.Email),
	})
	if err != nil {
		return db.User{}, classify(err)
	}
	return user, nil
}

func (r *Repository) CreateSession(ctx context.Context, userID string, tokenHash []byte, expiresAt time.Time, userAgent, ipAddress string) error {
	id, err := parseUUID(userID)
	if err != nil {
		return err
	}
	_, err = r.queries.CreateSession(ctx, db.CreateSessionParams{
		UserID:    id,
		TokenHash: tokenHash,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
		UserAgent: userAgent,
		IpAddress: ipAddress,
	})
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (r *Repository) UserBySession(ctx context.Context, tokenHash []byte) (domain.User, error) {
	user, err := r.queries.GetUserBySession(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, ErrNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("get session user: %w", err)
	}
	roles, err := r.queries.ListUserRoles(ctx, user.ID)
	if err != nil {
		return domain.User{}, fmt.Errorf("list user roles: %w", err)
	}
	_ = r.queries.TouchSession(ctx, tokenHash)
	return toDomain(user, roles), nil
}

func (r *Repository) RevokeSession(ctx context.Context, tokenHash []byte) error {
	if err := r.queries.RevokeSession(ctx, tokenHash); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

func parseUUID(value string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("parse user id: %w", err)
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
}

func uuidString(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	return uuid.UUID(value.Bytes).String()
}

func text(value string) pgtype.Text {
	value = strings.TrimSpace(value)
	return pgtype.Text{String: value, Valid: value != ""}
}

func textString(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func toDomain(user db.User, roles []db.UserRole) domain.User {
	return domain.User{
		ID:            uuidString(user.ID),
		DisplayName:   user.DisplayName,
		AvatarURL:     textString(user.AvatarUrl),
		Email:         textString(user.Email),
		EmailVerified: user.EmailVerifiedAt.Valid,
		Roles:         roleStrings(roles),
		CreatedAt:     user.CreatedAt.Time,
	}
}

func roleStrings(roles []db.UserRole) []string {
	values := make([]string, len(roles))
	for i, role := range roles {
		values[i] = string(role)
	}
	return values
}

func classify(err error) error {
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && postgresError.Code == "23505" {
		return ErrConflict
	}
	return err
}
