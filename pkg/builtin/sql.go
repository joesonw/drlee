package builtin

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	"go.uber.org/zap"

	uuid "github.com/satori/go.uuid"
	lua "github.com/yuin/gopher-lua"
)

//go:generate syncmap -name stringLSQLConnMap -pkg libs -o stringlsqlconnmap_gen.go map[string]*lSQLConn

const (
	lSQLConnClass        = "SQLCONN*"
	lSQLTransactionClass = "SQLTRANSACTION*"
)

type sqlDBInterface interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

var sqlFuncs = map[string]lua.LGFunction{
	"sql_open": sqlOpen,
}

func OpenSQL(L *lua.LState, env *Env) {
	ctx := &sqlContext{
		open:   env.OpenSQL,
		logger: env.Logger,
	}
	ud := L.NewUserData()
	ud.Value = ctx

	RegisterGlobalFuncs(L, sqlFuncs, ud)
}

type sqlContext struct {
	logger *zap.Logger
	open   func(driverName, dataSourceName string) (*sql.DB, error)
}

func upSQLContext(L *lua.LState) *sqlContext {
	uv := L.Get(lua.UpvalueIndex(1)).(*lua.LUserData)
	if ctx, ok := uv.Value.(*sqlContext); ok {
		return ctx
	}

	L.RaiseError("expected sql context")
	return nil
}

func sqlOpen(L *lua.LState) int {
	ctx := upSQLContext(L)

	if L.GetTop() != 2 {
		L.RaiseError("sql_open(uri, cb): takes two inputs")
		return 0
	}

	uri := L.CheckString(1)
	u, err := url.Parse(uri)
	if err != nil {
		L.ArgError(1, "sql_open(uri, cb): not a valid uri")
		return 0
	}

	cb := NewCallback(L.Get(2))
	go func() {
		db, err := ctx.open(u.Scheme, fmt.Sprintf("%s@tcp(%s)%s?%s", u.User.String(), u.Host, u.Path, u.Query().Encode()))

		if err != nil {
			cb.Reject(L, lua.LString("sql_open(uri, cb): unable to connection to database: "+err.Error()))
			return
		}

		cb.Resolve(L, NewGoObject(L, sqlConnFuncs, nil, &lSQLConn{
			context: ctx,
			id:      uuid.NewV4().String(),
			uri:     uri,
			db:      db,
		}, false))
	}()

	return 0
}

var sqlConnFuncs = map[string]lua.LGFunction{
	"__tostring": lSQLConnToString,
	"close":      lSQLConnClose,
	"query":      lSQLConnQuery,
	"exec":       lSQLConnExec,
	"begin":      lSQLBegin,
}

type lSQLConn struct {
	context *sqlContext
	id      string
	uri     string
	db      *sql.DB
}

func upSQLConn(L *lua.LState) *lSQLConn {
	conn, err := GetValueFromGoObject(L.CheckUserData(1))
	if err != nil {
		L.RaiseError(err.Error())
	}
	return conn.(*lSQLConn)
}

func lSQLConnToString(L *lua.LState) int {
	conn := upSQLConn(L)
	L.Push(lua.LString(lSQLConnClass + "(" + conn.uri + ")"))
	return 1
}

func lSQLConnClose(L *lua.LState) int {
	conn := upSQLConn(L)
	cb := NewCallback(L.Get(2))
	go func() {
		if err := conn.db.Close(); err != nil {
			cb.Reject(L, lua.LString("("+lSQLConnClass+").close(cb): "+err.Error()))
			return
		}
		cb.Finish(L)
	}()

	return 0
}

func lSQLConnQuery(L *lua.LState) int {
	conn := upSQLConn(L)
	return sqlQueryAux(L, lSQLConnClass, conn.db)
}

func lSQLConnExec(L *lua.LState) int {
	conn := upSQLConn(L)
	count := sqlExecAux(L, lSQLConnClass, conn.db)
	return count
}

func lSQLBegin(L *lua.LState) int {
	conn := upSQLConn(L)
	cb := NewCallback(L.Get(2))
	go func() {
		tx, err := conn.db.BeginTx(L.Context(), nil)
		if err != nil {
			cb.Reject(L, lua.LString("("+lSQLTransactionClass+").begin(cb): "+err.Error()))
			return
		}
		cb.Resolve(L, NewGoObject(L, sqlTransactionFuncs, nil, &lSQLTransaction{
			conn: conn,
			tx:   tx,
		}, false))
	}()
	return 0
}

var sqlTransactionFuncs = map[string]lua.LGFunction{
	"__tostring": lSQLTransactionToString,
	"query":      lSQLTransactionQuery,
	"exec":       lSQLTransactionExec,
	"commit":     lSQLTransactionCommit,
	"rollback":   lSQLTransactionRollback,
}

type lSQLTransaction struct {
	conn *lSQLConn
	tx   *sql.Tx
}

func checkSQLTransaction(L *lua.LState) *lSQLTransaction {
	ud := L.CheckUserData(1)
	if tx, ok := ud.Value.(*lSQLTransaction); ok {
		return tx
	}
	L.RaiseError("expected " + lSQLTransactionClass)
	return nil
}

func lSQLTransactionToString(L *lua.LState) int {
	L.Push(lua.LString(lSQLTransactionClass))
	return 1
}

func lSQLTransactionQuery(L *lua.LState) int {
	tx := checkSQLTransaction(L)
	return sqlQueryAux(L, lSQLTransactionClass, tx.tx)
}

func lSQLTransactionExec(L *lua.LState) int {
	tx := checkSQLTransaction(L)
	return sqlExecAux(L, lTimestampClass, tx.tx)
}

func lSQLTransactionCommit(L *lua.LState) int {
	tx := checkSQLTransaction(L)
	cb := NewCallback(L.Get(2))
	go func() {
		if err := tx.tx.Commit(); err != nil {
			cb.Reject(L, lua.LString("("+lSQLTransactionClass+").commit(cb): "+err.Error()))
			return
		}
		cb.Finish(L)

	}()
	return 0
}

func lSQLTransactionRollback(L *lua.LState) int {
	tx := checkSQLTransaction(L)
	cb := NewCallback(L.Get(2))
	go func() {
		if err := tx.tx.Commit(); err != nil {
			cb.Reject(L, lua.LString("("+lSQLTransactionClass+").rollback(cb): "+err.Error()))
			return
		}
		cb.Finish(L)
	}()
	return 0
}

func sqlQueryAux(L *lua.LState, className string, db sqlDBInterface) int {
	argn := L.GetTop()
	query := L.CheckString(2)
	args := make([]interface{}, argn-3)
	cb := NewCallback(L.Get(argn))

	for i := 3; i < argn; i++ {
		arg := L.Get(i)
		switch arg.Type() {
		case lua.LTBool:
			args[i-3] = lua.LVAsBool(arg)
		case lua.LTString:
			args[i-3] = lua.LVAsString(arg)
		case lua.LTNumber:
			args[i-3] = lua.LVAsNumber(arg)
		default:
			L.ArgError(i, fmt.Sprintf("(%s).query(query, ...args): arg %d with type %s is not supported", className, i-2, arg.Type().String()))
			return 0
		}
	}

	go func() {
		rows, err := db.QueryContext(L.Context(), query, args...)
		if err != nil {
			cb.Reject(L, lua.LString(fmt.Sprintf("(%s).query(query, ...args): unable to query: %s", className, err.Error())))
			return
		}
		defer func() {
			err := rows.Close()
			if err != nil {
				L.RaiseError(err.Error())
			}
		}()

		cols, err := rows.Columns()
		if err != nil {
			cb.Reject(L, lua.LString(fmt.Sprintf("(%s).query(query, ...args): unable to get columns: %s", className, err.Error())))
			return
		}

		resultSet := L.NewTable()
		for rows.Next() {
			columns := make([]interface{}, len(cols))
			columnPointers := make([]interface{}, len(cols))
			for i := range columns {
				columnPointers[i] = &columns[i]
			}

			if err := rows.Scan(columnPointers...); err != nil {
				cb.Reject(L, lua.LString(fmt.Sprintf("(%s).query(query, ...args): unable to scan: %s", className, err.Error())))
				return
			}

			m := make(map[string]interface{})
			for i, colName := range cols {
				val := columnPointers[i].(*interface{})
				m[colName] = *val
			}
			resultSet.Append(JSONDecodeValue(L, m))
		}

		cb.Resolve(L, resultSet)
	}()
	return 0
}

func sqlExecAux(L *lua.LState, className string, db sqlDBInterface) int {
	argn := L.GetTop()
	query := L.CheckString(2)
	args := make([]interface{}, argn-3)
	cb := NewCallback(L.Get(argn))

	for i := 3; i < argn; i++ {
		arg := L.Get(i)
		switch arg.Type() {
		case lua.LTBool:
			args[i-3] = lua.LVAsBool(arg)
		case lua.LTString:
			args[i-3] = lua.LVAsString(arg)
		case lua.LTNumber:
			args[i-3] = lua.LVAsNumber(arg)
		default:
			L.ArgError(i, fmt.Sprintf("(%s).exec(query, ...args): arg %d with type %s is not supported", className, i-2, arg.Type().String()))
			return 0
		}
	}

	go func() {
		result, err := db.ExecContext(L.Context(), query, args...)
		if err != nil {
			cb.Reject(L, lua.LString(fmt.Sprintf("(%s).exec(query, ...args): unable to exec: %s", className, err.Error())))
			return
		}

		id, err := result.LastInsertId()
		if err != nil {
			cb.Reject(L, lua.LString(fmt.Sprintf("(%s).exec(query, ...args): unable to exec: %s", className, err.Error())))
			return
		}

		cb.Resolve(L, lua.LNumber(id))
	}()
	return 0
}
