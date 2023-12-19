package main

import (
	"encoding/json"
	"errors"
	"github.com/gofiber/fiber/v2"
	"time"
)

type User struct {
	Id                 int     `json:"id"`
	Name               string  `json:"name"`
	Balance            float64 `json:"balance"`
	VerificationStatus bool    `json:"verification_status"`
}

type Transaction struct {
	Id         int     `json:"id"`
	SenderId   int     `json:"sender_id"`
	ReceiverId int     `json:"receiver_id"`
	Amount     float64 `json:"amount"`
	Completed  bool    `json:"completed"`
}

var Users = make(map[int]*User)
var UserCount = 0
var verificationQueue = make(chan User)

var Transactions = make(map[int]*Transaction)
var TransactionCount = 0
var transactionQueue = make(chan Transaction)

func CreateTransaction(transaction Transaction) Transaction {
	transaction.Id = TransactionCount + 1
	TransactionCount++
	Transactions[transaction.Id] = &transaction

	go func() {
		transactionQueue <- transaction
	}()

	return transaction
}

func CreateUser(u User) User {

	u.Balance = 1000
	u.Id = UserCount + 1
	UserCount++
	Users[u.Id] = &u

	// send user to verification queue
	go func() {
		verificationQueue <- u
	}()
	// after verification, lookup user and update verification status

	return u
}

func handleError(err error, c *fiber.Ctx) error {
	return c.Status(400).JSON(fiber.Map{
		"error": false,
		"msg":   err.Error(),
	})
}

func setupVerificationWorkers(num int, ticker *time.Ticker) {

	go func() {
		for {
			select {
			case _ = <-ticker.C:

				for i := 1; i <= num; i++ {
					// spawn go routines for stuff
					go func(i int) {
						select {
						case user := <-verificationQueue:
							Users[user.Id].VerificationStatus = true
						}
					}(i)
				}
			}
		}
	}()
}

func setupTransactionWorkers(num int, ticker *time.Ticker) {
	go func() {
		for {
			select {
			case _ = <-ticker.C:
				for i := 1; i <= num; i++ {
					go func() {
						select {
						case transaction := <-transactionQueue:
							Users[transaction.SenderId].Balance -= transaction.Amount
							Users[transaction.ReceiverId].Balance += transaction.Amount
						}
					}()
				}
			}
		}
	}()
}

func main() {
	app := fiber.New()

	v1 := app.Group("/api/v1")
	user := v1.Group("/user")
	transaction := v1.Group("/transaction")

	user.Post("/", func(c *fiber.Ctx) error {

		// get the user data from the request body

		var user User

		if err := json.Unmarshal(c.Body(), &user); err != nil {
			return handleError(err, c)
		}

		user = CreateUser(user)

		return c.JSON(fiber.Map{
			"error": false,
			"data":  user,
		})
	})

	user.Get("/", func(c *fiber.Ctx) error {

		return c.JSON(fiber.Map{
			"error": false,
			"data":  Users,
		})
	})

	transaction.Post("/", func(c *fiber.Ctx) error {

		// get the user data from the request body

		var txn Transaction

		if err := json.Unmarshal(c.Body(), &txn); err != nil {
			return handleError(err, c)
		}

		// check all participating users and sender balance.

		if _, ok := Users[txn.SenderId]; !ok {
			if !Users[txn.SenderId].VerificationStatus {
				verificationQueue <- *Users[txn.SenderId]
				return handleError(errors.New("sender must be verified"), c)
			}
			return handleError(errors.New("invalid sender_id"), c)
		}

		if _, ok := Users[txn.ReceiverId]; !ok {
			if !Users[txn.ReceiverId].VerificationStatus {
				verificationQueue <- *Users[txn.ReceiverId]
				return handleError(errors.New("receiver must be verified"), c)
			}
			return handleError(errors.New("invalid receiver_id"), c)
		}

		if Users[txn.SenderId].Balance < txn.Amount {
			return handleError(errors.New("please top up your account"), c)
		}

		txn = CreateTransaction(txn)

		return c.JSON(fiber.Map{
			"error": false,
			"data":  txn,
		})
	})

	// run workers every time.sleep

	ticker := time.NewTicker(time.Second * 5)

	setupVerificationWorkers(4, ticker)
	setupTransactionWorkers(4, ticker)

	err := app.Listen(":5001")
	if err != nil {
		return
	}
}
