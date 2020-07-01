package libs

import (
	"database/sql"
	"fmt"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	lua "github.com/yuin/gopher-lua"
)

var _ = Describe("SQL", func() {
	Describe("Open", func() {
		It("should open", func() {
			req := make(chan string, 2)
			db, _, err := sqlmock.New()
			Expect(err).To(BeNil())
			defer db.Close()

			runAsyncLuaTest(
				`
				sql_open("mysql://user:password@localhost:1000/db1?utf=true", function (err)
					assert(err == nil, "err")
					resolve()
				end)
				`,
				func(L *lua.LState) {
					OpenSQL(L, &Env{
						OpenSQL: func(driverName, dataSourceName string) (*sql.DB, error) {
							req <- driverName
							req <- dataSourceName
							return db, nil
						},
					})
				},
				func(L *lua.LState) {
					Expect(<-req).To(Equal("mysql"))
					Expect(<-req).To(Equal("user:password@tcp(localhost:1000)/db1?utf=true"))
				})
		})
	})

	Describe("SQLConn", func() {
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

			runAsyncLuaTest(fmt.Sprintf(`
				sql_open("mysql://user:password@localhost:1000/db1?utf=true", function (err, conn)
					assert(err == nil, "sql_open no error")
					conn:query("SELECT * FROM test WHERE id = ?", 1, function (err, result)
						assert(err == nil, "conn:query no error")
						assert(#result == 2, "sql result result")
						assert(result[1]["id"] == 1, "sql result 1 id")
						assert(result[1]["name"] == "a", "sql result 1 name")
						assert(result[2]["id"] == 2, "sql result 2 id")
						assert(result[2]["name"] == "b", "sql result 2 name")
						
						conn:exec("INSERT INTO test (id, name) VALUES (?, ?)", 1, "test", function (err, lastInsertID)
							assert(err == nil, "conn:exec no error")
							assert(lastInsertID == 3, "sql exec")
							conn:close(function (err)
								assert(err == nil, "conn:close no error")
								resolve()
							end)
						end)
					end)
				end)
				`),
				func(L *lua.LState) {
					OpenSQL(L, &Env{
						OpenSQL: func(driverName, dataSourceName string) (*sql.DB, error) {
							return db, nil
						},
					})
				},
				func(L *lua.LState) {
					OpenSQL(L, &Env{
						OpenSQL: func(driverName, dataSourceName string) (*sql.DB, error) {
							return db, nil
						},
					})
				})
		})
	})
})
