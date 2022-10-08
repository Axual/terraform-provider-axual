package webclient

import (
	"fmt"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"net/http"
)

func SignIn(auth AuthStruct) (*http.Client, error) {
	userName := auth.Username
	password := auth.Password
	clientId := auth.ClientId
	conf := oauth2.Config{
		ClientID: clientId,
		Endpoint: oauth2.Endpoint{
			TokenURL: auth.Url,
		},
	}
	if auth.Scopes != nil {
		conf.Scopes = auth.Scopes
	}
	token, err := conf.PasswordCredentialsToken(context.Background(), userName, password)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	client := conf.Client(context.Background(), token)
	return client, nil
}
