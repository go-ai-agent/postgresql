package pgxsql

import (
	"errors"
	"fmt"
	"github.com/go-ai-agent/core/runtime"
	"github.com/go-ai-agent/postgresql/pgxdml"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"net/http"
	"time"
)

type TestConditions struct {
	Time        time.Time
	Location    string
	Temperature float64
}

type rowsT struct {
}

func (r *rowsT) CommandTag() pgconn.CommandTag {
	//TODO implement me
	return pgconn.CommandTag{}
}

func (r *rowsT) FieldDescriptions() []pgconn.FieldDescription {
	//TODO implement me
	return nil
}

func (r *rowsT) Conn() *pgx.Conn {
	//TODO implement me
	return nil
}

func (r *rowsT) Close()     {}
func (r *rowsT) Err() error { return nil }

//	func (r *rowsT) CommandTag() CommandTag {
//		return pgconn.CommandTag{}//RowsAffected: 1, Insert: false, Update: false, Delete: false, Select: true}
//	}
//
// func (r *rowsT) FieldDescriptions() []FieldDescription { return nil }
func (r *rowsT) Next() bool             { return false }
func (r *rowsT) Scan(dest ...any) error { return nil }
func (r *rowsT) Values() ([]any, error) { return nil, nil }
func (r *rowsT) RawValues() [][]byte    { return nil }

const (
	queryErrorSql = "select * from test"
	queryRowsSql  = "select * from table"

	queryConditions      = "select * from conditions"
	queryConditionsWhere = "select * from conditions where $1 order by temperature desc"
	queryConditionsError = "select test,test2 from conditions"
	queryErrorRsc        = "error"
	queryRowsRsc         = "rows"
)

var queryTestExchange = runtime.NewProxyContext(nil, queryTestProxy)

func queryTestProxy(req Request) (pgx.Rows, error) {
	switch req.Uri() {
	case BuildQueryUri(queryErrorRsc):
		return nil, errors.New("pgxsql query error")
	case BuildQueryUri(queryRowsRsc):
		var i pgx.Rows = &rowsT{}
		return i, nil
	}
	return nil, nil
}

func ExampleQuery_TestError() {
	ctx := queryTestExchange
	result, status := Query(ctx, NewQueryRequest(queryErrorRsc, queryErrorSql, nil))
	fmt.Printf("test: Query(ctx,%v) -> [rows:%v] [status:%v]\n", queryErrorSql, result, status)

	//Output:
	//[[] github.com/go-ai-agent/postgresql/pgxsql/exec [pgxsql query error]]
	//test: Query(ctx,select * from test) -> [rows:<nil>] [status:Internal]

}

func ExampleQuery_TestRows() {
	ctx := queryTestExchange
	result, status := Query(ctx, NewQueryRequest(queryRowsRsc, queryRowsSql, nil))
	fmt.Printf("test: Query(ctx,%v) -> [rows:%v] [status:%v] [cmd:%v]\n", queryRowsSql, result, status, result.CommandTag())

	//Output:
	//test: Query(ctx,select * from table) -> [rows:&{}] [status:OK] [cmd:{select * 1 false false false true}]

}

func ExampleQuery_Conditions_Error() {
	err := testStartup()
	if err != nil {
		fmt.Printf("test: testStartup() -> [error:%v]\n", err)
	} else {
		defer ClientShutdown()
		req := NewQueryRequest(queryRowsRsc, queryConditionsError, nil)
		results, status := Query(nil, req)
		if !status.OK() {
			fmt.Printf("test: Query(nil,%v) -> [status:%v]\n", queryConditionsError, status)
		} else {
			fmt.Printf("test: Query(nil,%v) -> [status:%v] [cmd:%v]\n", queryConditions, status, results.CommandTag())
			conditions, status1 := processResults(results, "")
			fmt.Printf("test: processResults(results) -> [status:%v] [rows:%v]\n", status1, conditions)
		}
	}

	//Output:
	//[[] github.com/gotemplates/postgresql/pgxsql/query [serverity:ERROR, code:42703, message:column "test" does not exist, position:8, SQLState:42703]]
	//test: Query(nil,select test,test2 from conditions) -> [status:Internal]

}

func ExampleQuery_Conditions() {
	err := testStartup()
	if err != nil {
		fmt.Printf("test: testStartup() -> [error:%v]\n", err)
	} else {
		defer ClientShutdown()
		req := NewQueryRequest(queryRowsRsc, queryConditions, nil)
		results, status := Query(nil, req)
		if !status.OK() {
			fmt.Printf("test: Query(nil,%v) -> [status:%v]\n", queryConditions, status)
		} else {
			fmt.Printf("test: Query(nil,%v) -> [status:%v] [cmd:%v]\n", queryConditions, status, results.CommandTag())
			conditions, status1 := processResults(results, "")
			fmt.Printf("test: processResults(results) -> [status:%v] [rows:%v]\n", status1, conditions)
		}
	}

	//Output:
	//test: Query(nil,select * from conditions) -> [status:OK] [cmd:{ 0 false false false false}]
	//test: processResults(results) -> [status:OK] [rows:[{2023-01-26 12:09:12.426535 -0600 CST office 70} {2023-01-26 12:09:12.426535 -0600 CST basement 66.5} {2023-01-26 12:09:12.426535 -0600 CST garage 45.1234}]]

}

func ExampleQuery_Conditions_Where() {
	err := testStartup()
	if err != nil {
		fmt.Printf("test: testStartup() -> [error:%v]\n", err)
	} else {
		defer ClientShutdown()

		where := []pgxdml.Attr{{"location", "garage"}}
		req := NewQueryRequest(queryRowsRsc, queryConditionsWhere, where)
		results, status := Query(nil, req)
		if !status.OK() {
			fmt.Printf("test: Query(nil,%v) -> [status:%v]\n", queryConditionsWhere, status)
		} else {
			fmt.Printf("test: Query(nil,%v) -> [status:%v] [cmd:%v]\n", queryConditions, status, results.CommandTag())
			conditions, status1 := processResults(results, "")
			fmt.Printf("test: processResults(results) -> [status:%v] [rows:%v]\n", status1, conditions)
		}
	}

	//Output:
	//test: Query(nil,select * from conditions) -> [status:OK] [cmd:{ 0 false false false false}]
	//test: processResults(results) -> [status:OK] [rows:[{2023-01-26 12:09:12.426535 -0600 CST garage 45.1234}]]

}

func processResults(results pgx.Rows, msg string) (conditions []TestConditions, status *runtime.Status) {
	conditions, status = scanRows(results)
	if status.OK() && len(conditions) == 0 {
		return nil, runtime.NewStatus(http.StatusNotFound)
	}
	return conditions, status
}

func scanRows(rows pgx.Rows) ([]TestConditions, *runtime.Status) {
	if rows == nil {
		return nil, runtime.NewStatusError(runtime.StatusInvalidArgument, "", errors.New("invalid request: Rows interface is empty"))
	}
	var err error
	var values []any
	var conditions []TestConditions
	for rows.Next() {
		err = rows.Err()
		if err != nil {
			return nil, runtime.NewStatusError(0, "", err)
		}
		values, err = rows.Values()
		if err != nil {
			return nil, runtime.NewStatusError(0, "", err)
		}
		conditions = append(conditions, scanColumns(values))
	}
	return conditions, runtime.NewStatusOK()
}

func scanColumns(values []any) TestConditions {
	var ts = new(pgtype.Timestamp)
	ts.Scan(values[0])

	cond := TestConditions{
		Time:        ts.Time,
		Location:    values[1].(string),
		Temperature: values[2].(float64),
	}
	return cond
}
