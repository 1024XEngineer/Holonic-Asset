package dao_test

import (
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/internal/repository/dao"
)

func TestInitTablesRejectsNilDatabase(t *testing.T) {
	if err := dao.InitTables(nil); err == nil {
		t.Fatal("expected nil database to be rejected")
	}
}
