package db

import (
	"context"
	"database/sql"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
)

type (
	Operation struct {
		Id                  int64
		ExprId              int64
		A                   float64
		B                   float64
		Oper                string
		Res                 sql.NullFloat64
		State               string
		NotifyOperationId   int64
		NotifyOperationSide string
		Final               int64
	}
)

func CreateOpersTable(ctx context.Context, db *sql.DB) error {
	const (
		opersTable = `
		CREATE TABLE "operations" (
			"id"	INTEGER,
			"a"	REAL,
			"b"	REAL,
			"oper"	TEXT NOT NULL,
			"res"	REAL,
			"state"	TEXT NOT NULL,
			"expression_id"	INTEGER NOT NULL,
			"final"	INTEGER,
			"notify_operation_id"	INTEGER,
			"notify_operation_side"	TEXT,
			FOREIGN KEY("expression_id") REFERENCES "expressions"("id"),
			PRIMARY KEY("id" AUTOINCREMENT)
		);`
	)

	if _, err := db.ExecContext(ctx, opersTable); err != nil {
		return err
	}

	return nil
}

func (o Operation) Print() string {
	id := strconv.FormatInt(o.Id, 10)
	exprId := strconv.FormatInt(o.ExprId, 10)
	a := strconv.FormatFloat(o.A, 'f', -1, 64)
	b := strconv.FormatFloat(o.B, 'f', -1, 64)
	res := strconv.FormatFloat(o.Res.Float64, 'f', -1, 64)
	return "Id: " + id + " ExprId: " + exprId + " A: " + a + " B: " + b + " Oper: " + o.Oper + " State: " + o.State + " Res: " + res
}

func InsertOperation(ctx context.Context, db *sql.DB, o *Operation) (int64, error) {
	var q = `
	INSERT INTO operations (expression_id, a, b, oper, state,
		 notify_operation_id, notify_operation_side, final) 
		 values ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	result, err := db.ExecContext(ctx, q, o.ExprId, o.A, o.B, o.Oper, o.State,
		o.NotifyOperationId, o.NotifyOperationSide, o.Final)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

func SelectOperations(ctx context.Context, db *sql.DB) ([]Operation, error) {
	var operations []Operation
	var q = "SELECT * FROM operations"
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		o := Operation{}
		err := rows.Scan(&o.Id, &o.ExprId, &o.A, &o.B, &o.Oper, &o.Res, &o.State,
			&o.NotifyOperationId, &o.NotifyOperationSide, &o.Final)
		if err != nil {
			return nil, err
		}
		operations = append(operations, o)
	}
	return operations, nil
}

func SelectOperationById(ctx context.Context, db *sql.DB, id int64) (Operation, error) {
	o := Operation{}
	var q = `SELECT * FROM operations WHERE id = $1`
	err := db.QueryRowContext(ctx, q, id).Scan(&o.Id, &o.A, &o.B, &o.Oper, &o.Res, &o.State, &o.ExprId, &o.Final,
		&o.NotifyOperationId, &o.NotifyOperationSide)
	if err != nil {
		panic(err)
	}

	return o, nil
}

func SelectOperationsToCalc(ctx context.Context, db *sql.DB) ([]Operation, error) {
	var operations []Operation
	var q = `SELECT * FROM operations WHERE state IN ('created', 'ready_to_calc')`
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		o := Operation{}
		err := rows.Scan(&o.Id, &o.A, &o.B, &o.Oper, &o.Res, &o.State, &o.ExprId, &o.Final,
			&o.NotifyOperationId, &o.NotifyOperationSide)
		if err != nil {
			return nil, err
		}
		operations = append(operations, o)
	}
	return operations, nil
}

func SetOperationNotification(ctx context.Context, db *sql.DB, id int64, receiver_id int64, num_side string) error {
	var q = "UPDATE operations SET notify_operation_id = $1, notify_operation_side = $2 WHERE id = $3"
	_, err := db.ExecContext(ctx, q, receiver_id, num_side, id)
	if err != nil {
		panic(err)
	}

	return nil
}

func MakeOperationFinal(ctx context.Context, db *sql.DB, id int64) error {
	var q = "UPDATE operations SET final = 1 WHERE id = $1"
	_, err := db.ExecContext(ctx, q, id)
	if err != nil {
		return err
	}

	return nil
}

func SetOperationNum(ctx context.Context, db *sql.DB, id int64, side string, number float64) error {
	var q string
	if side == "left" {
		q = "UPDATE operations SET A = $1 WHERE id = $2"
	} else {
		q = "UPDATE operations SET B = $1 WHERE id = $2"
	}
	_, err := db.ExecContext(ctx, q, number, id)
	if err != nil {
		return err
	}

	return nil
}

func SetOperationRes(ctx context.Context, db *sql.DB, id int64, res float64) error {
	var q = "UPDATE operations SET res = $1 WHERE id = $2"
	_, err := db.ExecContext(ctx, q, res, id)
	if err != nil {
		return err
	}

	return nil
}

func SetOperationState(ctx context.Context, db *sql.DB, id int64, state string) error {
	var q = "UPDATE operations SET state = $1 WHERE id = $2"
	_, err := db.ExecContext(ctx, q, state, id)
	if err != nil {
		return err
	}

	return nil
}
