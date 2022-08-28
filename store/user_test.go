package store

import (
	"context"
	"errors"
	"testing"

	"github.com/HeRoMo/go_todo_app/clock"
	"github.com/HeRoMo/go_todo_app/entity"
	"github.com/HeRoMo/go_todo_app/testutil"
)

func TestRepository_RegisterUser(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	c := clock.FixedClocker{}
	tx, err := testutil.OpenDBForTest(t).BeginTxx(ctx, nil)
	t.Cleanup(func() { _ = tx.Rollback() })
	if err != nil {
		t.Fatal(err)
	}
	preparedUser := prepareUserTest(ctx, t, tx)

	tests := map[string]struct {
		user entity.User
	}{
		"ok": {
			user: entity.User{
				Name:     "john2",
				Password: "test",
				Role:     "role",
				Created:  c.Now(),
				Modified: c.Now(),
			},
		},
		"duplicate name": {
			user: entity.User{
				Name:     preparedUser.Name,
				Password: "test",
				Role:     "role",
				Created:  c.Now(),
				Modified: c.Now(),
			},
		},
	}

	for n, tt := range tests {
		tt := tt
		t.Run(n, func(t *testing.T) {
			t.Parallel()

			sut := &Repository{Clocker: c}
			err := sut.RegisterUser(ctx, tx, &tt.user)
			if err != nil {
				if !errors.Is(err, ErrAlreadyEntry) {
					t.Fatalf("unexecuted error: %v", err)
				}
			} else {
				if tt.user.ID <= 0 {
					t.Fatalf("ID not assgined: %v", tt.user)
				}
			}
		})
	}
}

func prepareUserTest(ctx context.Context, t *testing.T, con Execer) *entity.User {
	t.Helper()
	if _, err := con.ExecContext(ctx, "DELETE FROM user"); err != nil {
		t.Logf("failed to initialize user: %v", err)
	}
	c := clock.FixedClocker{}
	want := entity.User{
		Name:     "UserName",
		Password: "UserPassword",
		Role:     "user",
		Created:  c.Now(),
		Modified: c.Now(),
	}
	result, err := con.ExecContext(ctx,
		`INSERT INTO user (name, password, role, created, modified) VALUES (?,?,?,?,?);`,
		want.Name, want.Password, want.Role, want.Created, want.Modified)
	if err != nil {
		t.Fatal(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	want.ID = entity.UserID(id)
	return &want
}
