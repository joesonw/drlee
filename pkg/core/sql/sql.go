package sql

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	"github.com/joesonw/drlee/pkg/core"
	"github.com/joesonw/drlee/pkg/core/helpers"
	"github.com/joesonw/drlee/pkg/core/helpers/params"
	"github.com/joesonw/drlee/pkg/core/object"
	"github.com/joesonw/drlee/pkg/core/stream"
	"github.com/joesonw/drlee/pkg/utils"
	uuid "github.com/satori/go.uuid"
	lua "github.com/yuin/gopher-lua"
)

type Interface interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

type lSQL struct {
	open func(driverName, dataSourceName string) (*sql.DB, error)
	ec   *core.ExecutionContext
}

func Open(L *lua.LState, ec *core.ExecutionContext, open func(driverName, dataSourceName string) (*sql.DB, error)) {
	ud := L.NewUserData()
	ud.Value = &lSQL{
		open: open,
		ec:   ec,
	}

	utils.RegisterLuaModule(L, "sql", funcs, ud)
}

var funcs = map[string]lua.LGFunction{
	"open": lOpen,
}

func checkSQL(L *lua.LState) *lSQL {
	uv := L.Get(lua.UpvalueIndex(1)).(*lua.LUserData)
	if ctx, ok := uv.Value.(*lSQL); ok {
		return ctx
	}

	L.RaiseError("expected sql")
	return nil
}

func lOpen(L *lua.LState) int {
	uv := checkSQL(L)

	uri := params.String()
	cb := params.Check(L, 1, 1, "sql.open(uri, cb?)", uri)

	u, err := url.Parse(uri.String())
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}

	core.GoFunctionCallback(uv.ec, cb, func(ctx context.Context) (lua.LValue, error) {
		conn, err := uv.open(u.Scheme, fmt.Sprintf("%s@tcp(%s)%s?%s", u.User.String(), u.Host, u.Path, u.Query().Encode()))
		if err != nil {
			return lua.LNil, err
		}

		resource := core.NewResource("sql.Conn", func() {
			conn.Close()
		})
		uv.ec.Guard(resource)

		obj := object.NewProtected(L, connFuncs, map[string]lua.LValue{}, &lConn{
			ec:   uv.ec,
			id:   uuid.NewV4().String(),
			conn: conn,
		})
		obj.SetFunction("close", stream.NewCloser(L, uv.ec, resource, conn, true))

		return obj.Value(), nil
	})

	return 0
}

var connFuncs = map[string]lua.LGFunction{
	"query": lConnQuery,
	"exec":  lConnExec,
	"begin": lConnBegin,
	"close": lConnClose,
}

type lConn struct {
	ec   *core.ExecutionContext
	id   string
	conn *sql.DB
}

func checkConn(L *lua.LState) *lConn {
	conn, err := object.Value(L.CheckUserData(1))
	if err != nil {
		L.RaiseError(err.Error())
	}
	return conn.(*lConn)
}

func lConnQuery(L *lua.LState) int {
	conn := checkConn(L)
	return query(L, conn.ec, conn.conn)
}

func lConnExec(L *lua.LState) int {
	conn := checkConn(L)
	count := exec(L, conn.ec, conn.conn)
	return count
}

func lConnBegin(L *lua.LState) int {
	conn := checkConn(L)
	core.GoFunctionCallback(conn.ec, L.Get(2), func(ctx context.Context) (lua.LValue, error) {
		tx, err := conn.conn.BeginTx(ctx, nil)
		if err != nil {
			return lua.LNil, err
		}
		obj := object.NewProtected(L, txFuncs, map[string]lua.LValue{}, &lTx{
			conn: conn,
			tx:   tx,
		})
		return obj.Value(), nil
	})
	return 0
}

func lConnClose(L *lua.LState) int {
	conn := checkConn(L)
	core.GoFunctionCallback(conn.ec, L.Get(2), func(ctx context.Context) (lua.LValue, error) {
		return lua.LNil, conn.conn.Close()
	})
	return 0
}

var txFuncs = map[string]lua.LGFunction{
	"query":    lTxQuery,
	"exec":     lTxExec,
	"commit":   lTxCommit,
	"rollback": lTxRollback,
}

type lTx struct {
	conn *lConn
	tx   *sql.Tx
}

func checkTx(L *lua.LState) *lTx {
	ud := L.CheckUserData(1)
	if tx, ok := ud.Value.(*lTx); ok {
		return tx
	}
	L.RaiseError("expected tx")
	return nil
}

func lTxQuery(L *lua.LState) int {
	tx := checkTx(L)
	return query(L, tx.conn.ec, tx.tx)
}

func lTxExec(L *lua.LState) int {
	tx := checkTx(L)
	return exec(L, tx.conn.ec, tx.tx)
}

func lTxCommit(L *lua.LState) int {
	tx := checkTx(L)
	core.GoFunctionCallback(tx.conn.ec, L.Get(2), func(ctx context.Context) (lua.LValue, error) {
		err := tx.tx.Commit()
		return lua.LNil, err
	})
	return 0
}

func lTxRollback(L *lua.LState) int {
	tx := checkTx(L)
	core.GoFunctionCallback(tx.conn.ec, L.Get(2), func(ctx context.Context) (lua.LValue, error) {
		err := tx.tx.Rollback()
		return lua.LNil, err
	})
	return 0
}

func query(L *lua.LState, ec *core.ExecutionContext, db Interface) int {
	top := L.GetTop()
	query := L.CheckString(2)
	args := make([]interface{}, top-3)

	for i := 3; i < top; i++ {
		args[i-3] = helpers.MustUnmarshalToMap(L, L.Get(i))
	}
	cb := L.Get(L.GetTop())

	ec.Call(core.Go(func(ctx context.Context) (err error) {
		// nolint:rowserrcheck
		rows, err := db.QueryContext(ctx, query, args...)
		if err != nil {
			ec.Call(core.Lua(cb, utils.LError(err)))
			return nil
		}
		if err = rows.Err(); err != nil {
			ec.Call(core.Lua(cb, utils.LError(err)))
			return rows.Close()
		}
		defer func() {
			err = rows.Close()
		}()

		cols, err := rows.Columns()
		if err != nil {
			ec.Call(core.Lua(cb, utils.LError(err)))
			return nil
		}

		var result []map[string]interface{}
		for rows.Next() {
			columns := make([]interface{}, len(cols))
			columnPointers := make([]interface{}, len(cols))
			for i := range columns {
				columnPointers[i] = &columns[i]
			}

			if err := rows.Scan(columnPointers...); err != nil {
				ec.Call(core.Lua(cb, utils.LError(err)))
				return nil
			}

			m := make(map[string]interface{})
			for i, colName := range cols {
				val := columnPointers[i].(*interface{})
				m[colName] = *val
			}
			result = append(result, m)
		}

		ec.Call(core.Scoped(func(L *lua.LState) error {
			tb := L.NewTable()
			for _, item := range result {
				value, err := helpers.MarshalMap(L, item)
				if err != nil {
					return utils.CallLuaFunction(L, cb, utils.LError(err))
				}
				tb.Append(value)
			}
			return utils.CallLuaFunction(L, cb, lua.LNil, tb)
		}))

		return nil
	}))

	return 0
}

func exec(L *lua.LState, ec *core.ExecutionContext, db Interface) int {
	top := L.GetTop()
	query := L.CheckString(2)
	args := make([]interface{}, top-3)

	for i := 3; i < top; i++ {
		args[i-3] = helpers.MustUnmarshalToMap(L, L.Get(i))
	}
	cb := L.Get(L.GetTop())

	ec.Call(core.Go(func(ctx context.Context) (err error) {
		result, err := db.ExecContext(ctx, query, args...)
		if err != nil {
			ec.Call(core.Lua(cb, utils.LError(err)))
			return nil
		}

		id, err := result.LastInsertId()
		if err != nil {
			ec.Call(core.Lua(cb, utils.LError(err)))
			return nil
		}

		rows, err := result.RowsAffected()
		if err != nil {
			ec.Call(core.Lua(cb, utils.LError(err)))
			return nil
		}

		ec.Call(core.Scoped(func(L *lua.LState) error {
			tb := L.NewTable()
			tb.RawSetString("last_inserted_id", lua.LNumber(id))
			tb.RawSetString("rows_affected", lua.LNumber(rows))
			return utils.CallLuaFunction(L, cb, lua.LNil, tb)
		}))
		return nil
	}))

	return 0
}
