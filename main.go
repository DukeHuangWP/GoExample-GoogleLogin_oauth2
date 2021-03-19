package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	googleOauthConfig *oauth2.Config
	hostPort          = "8080"
	hostURL           = "http://localhost:8080"
)

type GoogleAcc struct {
	ID            int
	Email         string
	VerifiedEmail bool
	PictureUrl    string
}

func main() {
	//https://console.cloud.google.com/apis/credentials
	googleOauthConfig = &oauth2.Config{
		RedirectURL:  hostURL + "/callback",
		ClientID:     "XXXXX.apps.googleusercontent.com", //set from google credentials
		ClientSecret: "XXXXX", //set from google credentials
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email"},
		Endpoint:     google.Endpoint,
	}

	http.HandleFunc("/", handleGoogleLogin)            //front-web
	http.HandleFunc("/action", handleAction)           //login and route
	http.HandleFunc("/callback", handleGoogleCallback) //callback from google oauth2
	fmt.Println(http.ListenAndServe(":"+hostPort, nil))
}

func handleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	var htmlIndex = `
<html>
<body>
	<a href="/action?state=test">Server Test</a>
	<br>
	<a href="/action?state=account">Show Google Email</a>
</body>
</html>
`
	fmt.Fprintf(w, htmlIndex)
}

func handleAction(w http.ResponseWriter, r *http.Request) {
	url := googleOauthConfig.AuthCodeURL(r.FormValue("state")) //URI Query : state
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)     //that client get Token and oauth2 from google
}

func handleGoogleCallback(w http.ResponseWriter, r *http.Request) {

	//http://localhost:8080/start?state=test&code=4%2F0AY0e-g42CDKUTrW7IG_0k4nI6tCxjILz776Z9zizcAcU4x0BTf1RfmbZWfTrbV0D3_UILQ&scope=email+openid+https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fuserinfo.email&authuser=0&prompt=none
	token, err := googleOauthConfig.Exchange(oauth2.NoContext, r.FormValue("code"))
	if err != nil {
		//fmt.Fprintf(w, "code exchange failed: %s", err.Error())
		state := r.FormValue("state")
		http.Redirect(w, r, "/action?state="+state, http.StatusTemporaryRedirect) //force login again
		return
	}

	response, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		fmt.Fprintf(w, "ailed getting user info: %s", err.Error())
		return
	}

	defer response.Body.Close()

	var account GoogleAcc
	json.NewDecoder(response.Body).Decode(&account) //save the google account to struct
	if err != nil {
		fmt.Fprintf(w, "failed reading response body: %s", err.Error())
		return
	}

	state := r.FormValue("state")
	handler, err := NewAction(state)
	if handler == nil || err != nil {
		fmt.Fprintf(w, "nothing to do (state=%v)", state)
		return
	}

	handler.Direct(&account, w, r)
	return
}

//Design Pattern : Factory
type Handlers interface {
	Direct(account *GoogleAcc, w http.ResponseWriter, r *http.Request)
}

func NewAction(actionType string) (Handlers, error) {
	switch actionType {

	case "test":
		return &HandlerTest{}, nil
	case "account":
		return &HandlerPrintAcc{}, nil

	default:
		return nil, fmt.Errorf("Wrong gun type passed")
	}

}

type HandlerTest struct{}

func (_ *HandlerTest) Direct(account *GoogleAcc, w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Client was login : %v\n", account.Email)
	fmt.Fprintf(w, "Is working.\n")
	return
}

type HandlerPrintAcc struct{}

func (_ *HandlerPrintAcc) Direct(account *GoogleAcc, w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Your Email: %v\n", account.Email)
	return
}
