package db

import (
	"context"
	"database/sql"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
)

type (
	Expression struct {
		Id         int64           `json:"id"`
		UserId     int64           `json:"userId"`
		Expr       string          `json:"expr"`
		Res        sql.NullFloat64 `json:"res"`
		State      string          `json:"state"`
		ReadyOpers int64           `json:"ready_opers"`
	}
)

func CreateExpressionsTable(ctx context.Context, db *sql.DB) error {
	const (
		expressionsTable = `
		CREATE TABLE "expressions" (
			"id"	INTEGER NOT NULL,
			"expr"	TEXT NOT NULL,
			"res"	REAL,
			"state"	TEXT NOT NULL,
			"ready_opers"	INTEGER NOT NULL DEFAULT 0,
			"user_id"	INTEGER NOT NULL,
			PRIMARY KEY("id" AUTOINCREMENT),
			FOREIGN KEY("user_id") REFERENCES "users"("id")
		);`
	)

	if _, err := db.ExecContext(ctx, expressionsTable); err != nil {
		return err
	}

	return nil
}

func (e Expression) Print() string {
	id := strconv.FormatInt(e.Id, 10)
	Res := strconv.FormatFloat(e.Res.Float64, 'f', -1, 64)
	return "Id: " + id + " Expression: " + e.Expr + " State:" + e.State + " Res:" + Res
}

func InsertExpression(ctx context.Context, db *sql.DB, expression *Expression) (int64, error) {
	var q = `
	INSERT INTO expressions (expr, state, user_id) values ($1, $2, $3)
	`
	result, err := db.ExecContext(ctx, q, expression.Expr, expression.State, expression.UserId)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}

func SelectExpressions(ctx context.Context, db *sql.DB) ([]Expression, error) {
	var expressions []Expression
	var q = "SELECT * FROM expressions"

	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		e := Expression{}
		var res sql.NullFloat64
		err := rows.Scan(&e.Id, &e.Expr, &res, &e.State, &e.ReadyOpers, &e.UserId)
		if err != nil {
			return nil, err
		}
		if res.Valid {
			e.Res = res
		}
		expressions = append(expressions, e)
	}

	return expressions, nil
}

func SelectExpressionById(ctx context.Context, db *sql.DB, id int64) (Expression, error) {
	e := Expression{}
	var q = "SELECT * FROM expressions WHERE id = $1"
	err := db.QueryRowContext(ctx, q, id).Scan(&e.Id, &e.Expr, &e.Res, &e.State, &e.ReadyOpers, &e.UserId)
	if err != nil {
		return e, err
	}
	return e, nil
}

func SetExpressionResult(ctx context.Context, db *sql.DB, id int64, res float64) error {
	var q = "UPDATE expressions SET state = 'ready', res = $1 WHERE id = $2"
	_, err := db.ExecContext(ctx, q, res, id)
	if err != nil {
		return err
	}

	return nil
}

func ExprOperationCalculated(ctx context.Context, db *sql.DB, id int64) error {
	var q = "UPDATE expressions SET ready_opers = ready_opers + 1 WHERE id = $1"
	_, err := db.ExecContext(ctx, q, id)
	if err != nil {
		return err
	}

	return nil
}
