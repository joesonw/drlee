package server

import (
	_ "github.com/denisenkom/go-mssqldb"            // sql driver
	_ "github.com/go-sql-driver/mysql"              // sql driver
	_ "github.com/lib/pq"                           // sql driver
	_ "github.com/mattn/go-sqlite3"                 // sql driver
	_ "github.com/prestodb/presto-go-client/presto" // sql driver
)
