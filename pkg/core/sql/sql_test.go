package sql

import (
	"database/sql"
	"fmt"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/joesonw/drlee/pkg/core"
	"github.com/joesonw/drlee/pkg/core/test"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	lua "github.com/yuin/gopher-lua"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "core/sql")
}

var _ = Describe("SQL", func() {
	It("should open", func() {
		req := make(chan string, 2)
		db, _, err := sqlmock.New()
		Expect(err).To(BeNil())
		defer db.Close()

		test.Async(
			`
				local sql = require "sql"
				sql.open("mysql://user:password@localhost:1000/db1?utf=true", function (err)
					assert(err == nil, "err")
					resolve()
				end)
				`,
			func(L *lua.LState, ec *core.ExecutionContext) {
				Open(L, ec, func(driverName, dataSourceName string) (*sql.DB, error) {
					req <- driverName
					req <- dataSourceName
					return db, nil
				})
			},
			func(L *lua.LState) {
				Expect(<-req).To(Equal("mysql"))
				Expect(<-req).To(Equal("user:password@tcp(localhost:1000)/db1?utf=true"))
			})
	})

	It("should insert and query", func() {
		db, mock, err := sqlmock.New()
		Expect(err).To(BeNil())
		defer db.Close()

		rows := sqlmock.NewRows([]string{"id", "name"}).
			AddRow(1, "a").
			AddRow(2, "b")
		mock.ExpectQuery("SELECT \\* FROM test WHERE id = ?").
			WithArgs(float64(1)).
			WillReturnRows(rows)
		mock.ExpectExec("INSERT INTO test").
			WithArgs(float64(1), "test").
			WillReturnResult(sqlmock.NewResult(3, 1))
		mock.ExpectClose()

		test.Async(fmt.Sprintf(`
				local sql = require "sql"
				sql.open("mysql://user:password@localhost:1000/db1?utf=true", function (err, conn)
					assert(err == nil, "sql_open no error")
					conn:query("SELECT * FROM test WHERE id = ?", 1, function (err, result)
						assert(err == nil, "conn:query no error")
						assert(#result == 2, "sql result result")
						assert(result[1]["id"] == 1, "sql result 1 id")
						assert(result[1]["name"] == "a", "sql result 1 name")
						assert(result[2]["id"] == 2, "sql result 2 id")
						assert(result[2]["name"] == "b", "sql result 2 name")
						
						conn:exec("INSERT INTO test (id, name) VALUES (?, ?)", 1, "test", function (err, res)
							assert(err == nil, "conn:exec no error")
							assert(res.last_inserted_id == 3, "sql exec")
							assert(res.rows_affected == 1, "sql exec")
							conn:close(function (err)
								assert(err == nil, "conn:close no error")
								resolve()
							end)
						end)
					end)
				end)
				`),
			func(L *lua.LState, ec *core.ExecutionContext) {
				Open(L, ec, func(driverName, dataSourceName string) (*sql.DB, error) {
					return db, nil
				})
			})
	})
})
