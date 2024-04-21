package main

import (
	"context"
	sql "database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Zheleznov-Fedor/new-ya-long-calc/db"
	parser "github.com/Zheleznov-Fedor/new-ya-long-calc/expr_parser"
	"github.com/Zheleznov-Fedor/new-ya-long-calc/utils"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

const hmacSampleSecret = "super_secret_signature"

func expressionHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.TODO()
	c, err := r.Cookie("token")
	var tokenString string
	if err != nil {
		tokenString = r.Header.Get("auth-token")
	} else {
		tokenString = c.Value
	}
	tokenFromString, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			panic(fmt.Errorf("unexpected signing method: %v", token.Header["alg"]))
		}

		return []byte(hmacSampleSecret), nil
	})
	claims, _ := tokenFromString.Claims.(jwt.MapClaims)

	if r.Method == http.MethodGet {
		exprIdStr := strings.TrimPrefix(r.URL.Path, "/expr/")
		exprId, err := strconv.ParseInt(exprIdStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid expr_id", http.StatusBadRequest)
			return
		}

		expr, err := db.SelectExpressionById(ctx, database, exprId)
		if err != nil {
			fmt.Println(err)
			http.Error(w, "some DataBase error", http.StatusInternalServerError)
			return
		}
		if expr.UserId != int64(claims["userId"].(float64)) {
			http.Error(w, "no access", http.StatusForbidden)
			return
		}
		jsonData, err := json.Marshal(expr)
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(jsonData)
		if err != nil {
			fmt.Println(err)
		}
		return
	}

	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Error parsing form", http.StatusBadRequest)
			return
		}

		var expr string
		err = json.NewDecoder(r.Body).Decode(&expr)
		if err != nil {
			http.Error(w, "Error parsing JSON", http.StatusBadRequest)
			return
		}
		exprID, readyOperIDs := parser.BuildOperations(expr, int64(claims["userId"].(float64)))
		for _, operId := range readyOperIDs {
			go func() {
				o, _ := db.SelectOperationById(ctx, database, operId)
				parser.SendTask(ctx, database, o)
			}()
		}
		fmt.Fprintf(w, strconv.Itoa(int(exprID)))
		return
	}

}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	ctx := context.TODO()
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	var userInfo db.UserWeb
	err = json.NewDecoder(r.Body).Decode(&userInfo)
	if err != nil {
		http.Error(w, "Error parsing JSON", http.StatusBadRequest)
		return
	}
	_, err = db.InsertUser(ctx, database, userInfo.Login, userInfo.Password)
	if err != nil {
		http.Error(w, "Error creating user", http.StatusBadRequest)
		return
	}
}
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	ctx := context.TODO()
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	var userInfo db.UserWeb
	err = json.NewDecoder(r.Body).Decode(&userInfo)
	if err != nil {
		http.Error(w, "Error parsing JSON", http.StatusBadRequest)
		return
	}
	user, err := db.SelectUser(ctx, database, userInfo.Login)
	if err != nil {
		http.Error(w, "No such user", http.StatusBadRequest)
		return
	}
	err = user.ComparePassword(userInfo.Password)
	if err != nil {
		http.Error(w, "Bad password", http.StatusUnauthorized)
		return
	}

	now := time.Now()
	expirationTime := now.Add(5 * time.Minute).Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId": user.Id,
		"nbf":    now.Unix(),
		"exp":    now.Add(5 * time.Minute).Unix(),
		"iat":    now.Unix(),
	})

	tokenString, err := token.SignedString([]byte(hmacSampleSecret))
	if err != nil {
		panic(err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:    "token",
		Value:   tokenString,
		Expires: time.Unix(expirationTime, 0),
	})
	fmt.Fprintf(w, tokenString)
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("token")
		var tokenString string
		if err != nil {
			if err == http.ErrNoCookie {
				tokenString = r.Header.Get("auth-token")
			} else {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}
		} else {
			tokenString = c.Value
		}
		tokenFromString, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				panic(fmt.Errorf("unexpected signing method: %v", token.Header["alg"]))
			}

			return []byte(hmacSampleSecret), nil
		})

		if err != nil {
			log.Fatal(err)
		}

		if _, ok := tokenFromString.Claims.(jwt.MapClaims); !ok {
			panic(err)
		}

		next.ServeHTTP(w, r)
	})
}

var database *sql.DB

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}
	x, _ := strconv.Atoi(os.Getenv("AGENTS_CNT"))
	utils.Port.SetCnt(x)

	database, err = sql.Open("sqlite3", "./db/expressions.db")
	if err != nil {
		panic(err)
	}
	defer database.Close()

	ctx := context.TODO()
	err = database.PingContext(ctx)
	if err != nil {
		panic(err)
	}
	_ = db.CreateUsersTable(ctx, database)
	_ = db.CreateExpressionsTable(ctx, database)
	_ = db.CreateOpersTable(ctx, database)

	ready, _ := db.SelectOperationsToCalc(ctx, database)
	for _, oper := range ready {
		go func() {
			parser.SendTask(ctx, database, oper)
		}()
	}
	fmt.Println("Ready! Listening on 8080")
	//
	http.HandleFunc("/expr/", authMiddleware(expressionHandler))
	http.HandleFunc("/expr", authMiddleware(expressionHandler))
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/register", registerHandler)

	http.ListenAndServe(":8080", nil)
}
