package dao

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestTaskDaoUpdateStatusFrom(t *testing.T) {
	const updateQuery = `UPDATE "tasks" SET "status"=$1,"updated_at"=$2 WHERE id = $3 AND status = $4`

	tests := []struct {
		name         string
		rowsAffected int64
		queryErr     error
		wantUpdated  bool
		wantErr      bool
	}{
		{name: "transitions matching status", rowsAffected: 1, wantUpdated: true},
		{name: "leaves stale status unchanged"},
		{name: "returns update error", queryErr: errors.New("update failed"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMockUserDatabase(t)
			mock.ExpectBegin()
			expectation := mock.ExpectExec(regexp.QuoteMeta(updateQuery)).
				WithArgs(2, sqlmock.AnyArg(), 17, 5)
			if tt.queryErr != nil {
				expectation.WillReturnError(tt.queryErr)
				mock.ExpectRollback()
			} else {
				expectation.WillReturnResult(sqlmock.NewResult(0, tt.rowsAffected))
				mock.ExpectCommit()
			}

			updated, err := NewTaskDao(db).UpdateStatusFrom(context.Background(), 17, 5, 2)
			if (err != nil) != tt.wantErr {
				t.Fatalf("unexpected update error: %v", err)
			}
			if updated != tt.wantUpdated {
				t.Fatalf("updated = %t, want %t", updated, tt.wantUpdated)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet database expectations: %v", err)
			}
		})
	}
}
