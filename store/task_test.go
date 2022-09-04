package store

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/HeRoMo/go_todo_app/clock"
	"github.com/HeRoMo/go_todo_app/entity"
	"github.com/HeRoMo/go_todo_app/testutil"
	"github.com/HeRoMo/go_todo_app/testutil/fixture"
	"github.com/google/go-cmp/cmp"
	"github.com/jmoiron/sqlx"
)

func prepareUser(ctx context.Context, t *testing.T, db Execer) entity.UserID {
	t.Helper()
	u := fixture.User(nil)
	result, err := db.ExecContext(ctx,
		`INSERT INTO user (name, password, role, created, modified) VALUES (?, ?, ?, ?, ?)`,
		u.Name, u.Password, u.Role, u.Created, u.Modified,
	)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("got user_id: %v", err)
	}
	return entity.UserID(id)
}

func prepareTasks(ctx context.Context, t *testing.T, con Execer) (entity.UserID, entity.Tasks) {
	t.Helper()
	userID := prepareUser(ctx, t, con)
	otherUserID := prepareUser(ctx, t, con)
	c := clock.FixedClocker{}
	wants := entity.Tasks{
		{
			UserID:   userID,
			Title:    "want task 1",
			Status:   "todo",
			Created:  c.Now(),
			Modified: c.Now(),
		},
		{
			UserID:   userID,
			Title:    "want task2",
			Status:   "done",
			Created:  c.Now(),
			Modified: c.Now(),
		},
	}
	tasks := entity.Tasks{
		wants[0],
		{
			UserID:   otherUserID,
			Title:    "not want task",
			Status:   "todo",
			Created:  c.Now(),
			Modified: c.Now(),
		},
		wants[1],
	}
	result, err := con.ExecContext(ctx,
		`INSERT INTO task (user_id, title, status, created, modified)
		VALUES (?, ?, ?, ?, ?), (?, ?, ?, ?, ?), (?, ?, ?, ?, ?)`,
		tasks[0].UserID, tasks[0].Title, tasks[0].Status, tasks[0].Created, tasks[0].Modified,
		tasks[1].UserID, tasks[1].Title, tasks[1].Status, tasks[1].Created, tasks[1].Modified,
		tasks[2].UserID, tasks[2].Title, tasks[2].Status, tasks[2].Created, tasks[2].Modified,
	)
	if err != nil {
		t.Fatal(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	tasks[0].ID = entity.TaskID(id)
	tasks[1].ID = entity.TaskID(id + 1)
	tasks[2].ID = entity.TaskID(id + 2)
	return userID, wants
}

func TestRepository_ListTasks(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	// entity.Task を作成する他のテストケースと混ざるとテストがフェイルする
	// そのため、トランザクションをはることでこのテストケースの中だけのテーブル状態にする。
	tx, err := testutil.OpenDBForTest(t).BeginTxx(ctx, nil)
	t.Cleanup(func() { _ = tx.Rollback() })
	if err != nil {
		t.Fatal(err)
	}
	wantUserID, wants := prepareTasks(ctx, t, tx)

	sut := &Repository{}
	gots, err := sut.ListTasks(ctx, tx, wantUserID)
	if err != nil {
		t.Fatalf("unexecuted error: %v", err)
	}
	if d := cmp.Diff(gots, wants); len(d) != 0 {
		t.Errorf("differs: (-got, +want)\n%s", d)
	}
}

func prepareTest(ctx context.Context, t *testing.T, con Execer) entity.Tasks {
	t.Helper()
	// 一度きれいにしておく
	if _, err := con.ExecContext(ctx, "DELETE FROM task;"); err != nil {
		t.Logf("failed to initialize task : %v", err)
	}
	c := clock.FixedClocker{}
	wants := entity.Tasks{
		{
			Title:    "want task1",
			Status:   "todo",
			Created:  c.Now(),
			Modified: c.Now(),
		},
		{
			Title:    "want task2",
			Status:   "todo",
			Created:  c.Now(),
			Modified: c.Now(),
		},
		{
			Title:    "want task3",
			Status:   "todo",
			Created:  c.Now(),
			Modified: c.Now(),
		},
	}
	result, err := con.ExecContext(ctx,
		`INSERT INTO task (title, status, created, modified) values (?,?,?,?),(?,?,?,?),(?,?,?,?);`,
		wants[0].Title, wants[0].Status, wants[0].Created, wants[0].Modified,
		wants[1].Title, wants[1].Status, wants[1].Created, wants[1].Modified,
		wants[2].Title, wants[2].Status, wants[2].Created, wants[2].Modified,
	)
	if err != nil {
		t.Fatal(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	wants[0].ID = entity.TaskID(id)
	wants[1].ID = entity.TaskID(id + 1)
	wants[2].ID = entity.TaskID(id + 2)
	return wants
}

func TestRepository_AddTask(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	c := clock.FixedClocker{}
	var wantID int64 = 20
	okTask := &entity.Task{
		UserID:   1,
		Title:    "ok task",
		Status:   "todo",
		Created:  c.Now(),
		Modified: c.Now(),
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
	})
	mock.ExpectExec(
		`INSERT INTO task \(user_id, title, status, created, modified\) VALUES \(\?, \?, \?, \?, \?\)`).
		WithArgs(okTask.UserID, okTask.Title, okTask.Status, c.Now(), c.Now()).
		WillReturnResult(sqlmock.NewResult(wantID, 1))

	xdb := sqlx.NewDb(db, "mysql")
	r := &Repository{Clocker: c}
	if err := r.AddTask(ctx, xdb, okTask); err != nil {
		t.Errorf("want no error, but got %v", err)
	}
}
