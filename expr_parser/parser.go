package parser

import (
	"context"
	sql "database/sql"
	"fmt"
	pb "github.com/Zheleznov-Fedor/new-ya-long-calc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"regexp"
	"strconv"
	"strings"

	db "github.com/Zheleznov-Fedor/new-ya-long-calc/db"
	"github.com/Zheleznov-Fedor/new-ya-long-calc/utils"
)

func SplitHumanExpressionToTokens(expression string) []string {
	re := regexp.MustCompile(`(\d+\.\d+|\d+|\+|-|\*|/)`)
	return re.FindAllString(expression, -1)
}

func TokensToRPN(tokens []string) utils.Queue {
	precedence := map[string]int{
		"+": 1,
		"-": 1,
		"*": 2,
		"/": 2,
	}

	var output utils.Queue
	var operators utils.Stack

	for _, token := range tokens {
		if _, ok := precedence[token]; ok {
			for !operators.IsEmpty() && precedence[operators.Head()] >= precedence[token] {
				output.Put(operators.Pop())
			}
			operators.Push(token)
		} else {
			output.Put(token)
		}
	}

	for !operators.IsEmpty() {
		output.Put(operators.Pop())
	}

	return output
}

func StrToFloat64(s string) float64 {
	val, _ := strconv.ParseFloat(s, 64)
	return float64(val)
}

func SplitRPNToComputations(tokens utils.Queue, expr string, userId int64) (int64, []int64) {
	ctx := context.TODO()
	var nums utils.Stack
	var ready_opers_ids []int64
	var left_link, right_link string
	var left, right float64
	var leftSenderOperID, rightSenderOperID int
	var operID int64
	state := "created"
	leftSenderOperID, rightSenderOperID = 0, 0

	d, err := sql.Open("sqlite3", "./db/expressions.db")
	if err != nil {
		panic(err)
	}
	defer d.Close()

	err = d.PingContext(ctx)
	if err != nil {
		panic(err)
	}

	exprID, err := db.InsertExpression(ctx, d, &db.Expression{
		UserId: userId,
		Expr:   expr,
		State:  "calculating",
	})
	if err != nil {
		panic(err)
	}

	for !tokens.IsEmpty() {
		token := tokens.Get()
		state = "created"

		if strings.Contains("+-*/", token) {
			left, right = 0, 0
			right_link = nums.Pop()
			if right_link[0] != '@' {
				right = StrToFloat64(right_link)
				right_link = ""
			} else {
				leftSenderOperID, _ = strconv.Atoi(right_link[1:])
			}
			left_link = nums.Pop()
			if left_link[0] != '@' {
				left = StrToFloat64(left_link)
				left_link = ""
			} else {
				rightSenderOperID, _ = strconv.Atoi(left_link[1:])
			}

			if leftSenderOperID != 0 {
				state = "waiting_for_right"
			}
			if rightSenderOperID != 0 {
				if state == "waiting_for_right" {
					state = "waiting_for_left&right"
				} else {
					state = "waiting_for_left"
				}
			}
			oper := db.Operation{
				ExprId: exprID,
				A:      left,
				B:      right,
				Oper:   token,
				State:  state,
			}
			operID, err = db.InsertOperation(ctx, d, &oper)
			if state == "created" {
				ready_opers_ids = append(ready_opers_ids, operID)
			}
			if err != nil {
				panic(err)
			}
			if leftSenderOperID != 0 {
				db.SetOperationNotification(ctx, d, int64(leftSenderOperID), operID, "right")
			}
			if rightSenderOperID != 0 {
				db.SetOperationNotification(ctx, d, int64(rightSenderOperID), operID, "left")
			}
			leftSenderOperID, rightSenderOperID = 0, 0
			nums.Push("@" + strconv.Itoa(int(operID)))
		} else {
			nums.Push(token)
		}
	}

	err = db.MakeOperationFinal(ctx, d, operID)
	if err != nil {
		panic(err)
	}

	return exprID, ready_opers_ids
}

func BuildOperations(expression string, userId int64) (int64, []int64) {
	return SplitRPNToComputations(TokensToRPN(SplitHumanExpressionToTokens(expression)), expression, userId)
}

func SendTask(ctx context.Context, d *sql.DB, oper db.Operation) {
	host := "localhost"
	port := utils.Port.GetValue()

	conn, err := grpc.Dial(fmt.Sprintf("%s:%s", host, port),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	grpcClient := pb.NewOperationServiceClient(conn)

	res, err := grpcClient.Calc(context.TODO(), &pb.OperationRequest{
		Id:   int32(oper.Id),
		A:    float32(oper.A),
		B:    float32(oper.B),
		Oper: oper.Oper,
	})
	if err != nil {
		go func() {
			SendTask(ctx, d, oper)
		}()
		return
	}

	err = db.ExprOperationCalculated(ctx, d, oper.ExprId)
	if err != nil {
		panic(err)
	}
	err = db.SetOperationRes(ctx, d, oper.Id, float64(res.Result))
	if err != nil {
		panic(err)
	}
	err = db.SetOperationState(ctx, d, oper.Id, "calculated")
	if err != nil {
		panic(err)
	}

	if oper.Final == 1 {
		err := db.SetExpressionResult(ctx, d, oper.ExprId, float64(res.Result))
		if err != nil {
			panic(err)
		}
		return
	}

	if oper.NotifyOperationId != 0 {
		err := db.SetOperationNum(ctx, d, oper.NotifyOperationId, oper.NotifyOperationSide, float64(res.Result))
		if err != nil {
			panic(err)
		}
		receiverOper, err := db.SelectOperationById(ctx, d, oper.NotifyOperationId)
		if err != nil {
			panic(err)
		}
		if receiverOper.State == "waiting_for_left&right" {
			if oper.NotifyOperationSide == "right" {
				err = db.SetOperationState(ctx, d, oper.NotifyOperationId, "waiting_for_left")
			} else {
				err = db.SetOperationState(ctx, d, oper.NotifyOperationId, "waiting_for_right")
			}
			if err != nil {
				panic(err)
			}
		} else {
			err = db.SetOperationState(ctx, d, oper.NotifyOperationId, "ready_to_calc")
			if err != nil {
				panic(err)
			}
			go func() {
				SendTask(ctx, d, receiverOper)
			}()
		}
	}
}
