package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"github.com/ding-live/firebase/pkg/ding"
	"github.com/google/uuid"
	"google.golang.org/api/option"
)

const port = ":8080"

type service struct {
	dingClient *ding.Client
	authClient *auth.Client
}

func SendCode(service *service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		type request struct {
			PhoneNumber string `json:"phone_number"`
		}

		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		authUUID, err := service.dingClient.Authenticate(r.Context(), req.PhoneNumber)
		if err != nil {
			http.Error(w, fmt.Sprintf("%s", err), dingErrToHTTP(err))
			return
		}

		w.Write([]byte(authUUID))
	}
}

func Verify(service *service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		type request struct {
			PhoneNumber        string `json:"phone_number"`
			Code               string `json:"code"`
			AuthenticationUUID string `json:"authentication_uuid"`
		}

		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		authUUID, err := uuid.Parse(req.AuthenticationUUID)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid auth UUID: %s", err), http.StatusBadRequest)
			return
		}

		success, err := service.dingClient.Check(r.Context(), authUUID, req.PhoneNumber, req.Code)
		if err != nil {
			http.Error(w, fmt.Sprintf("%s", err), dingErrToHTTP(err))
			return
		}

		if !success {
			http.Error(w, "invalid code", http.StatusUnauthorized)
			return
		}

		user, err := service.authClient.GetUserByPhoneNumber(r.Context(), req.PhoneNumber)
		if err != nil {
			newUser := (&auth.UserToCreate{}).PhoneNumber(req.PhoneNumber)
			createdUser, err := service.authClient.CreateUser(r.Context(), newUser)
			if err != nil {
				http.Error(w, fmt.Sprintf("create user: %s", err), http.StatusInternalServerError)
				return
			}
			user = createdUser
		}

		customToken, err := service.authClient.CustomToken(r.Context(), user.UID)
		if err != nil {
			http.Error(w, fmt.Sprintf("create custom token: %s", err), http.StatusInternalServerError)
			return
		}

		w.Write([]byte(customToken))
	}
}

func main() {
	ctx := context.Background()

	opt := option.WithCredentialsFile(os.Getenv("SA_FILE_PATH"))

	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		log.Fatalf("error initializing app: %s", err)
	}

	authClient, err := app.Auth(ctx)
	if err != nil {
		log.Fatalf("error initializing auth client: %s", err)
	}

	service := &service{
		authClient: authClient,
		dingClient: ding.New(&ding.Params{
			APIKey:       os.Getenv("DING_API_KEY"),
			CustomerUUID: os.Getenv("DING_CUSTOMER_UUID"),
		}),
	}

	http.HandleFunc("/send_code", SendCode(service))
	http.HandleFunc("/verify", Verify(service))

	log.Printf("listening on port %s", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("start server: %v", err)
	}
}

func dingErrToHTTP(err error) int {
	switch err {
	case ding.ErrUnauthorized:
		return http.StatusUnauthorized
	case ding.ErrInvalidInput:
		return http.StatusBadRequest
	case ding.ErrRateLimited:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}
