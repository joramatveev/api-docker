package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

type parsedResponse struct {
	status int
	body   []byte
}

func createRequester(t *testing.T) func(req *http.Request, err error) parsedResponse {
	return func(req *http.Request, err error) parsedResponse {
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return parsedResponse{}
		}

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		resp, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		return parsedResponse{res.StatusCode, resp}
	}
}

func prepareParams(t *testing.T, params map[string]interface{}) io.Reader {
	body, err := json.Marshal(params)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	return bytes.NewBuffer(body)
}

func newTestUserService() *UserService {
	return &UserService{
		repository: NewInMemoryUserStorage(),
		toasts:     make(chan []byte, 10),
		reg:        make(chan bool, 5),
		cake:       make(chan bool, 5),
	}
}

func assertStatus(t *testing.T, expected int, r parsedResponse) {
	if r.status != expected {
		t.Errorf("Unexpected response status. Expected: %d, actual: %d", expected, r.status)
	}
}

func assertBody(t *testing.T, expected string, r parsedResponse) {
	actual := string(r.body)
	if actual != expected {
		t.Errorf("Unexpected response body. Expected: %s, actual: %s", expected, actual)
	}
}

func TestUsers_Register(t *testing.T) {
	doRequest := createRequester(t)

	t.Run("email checking", func(t *testing.T) {
		u := newTestUserService()

		ts := httptest.NewServer(http.HandlerFunc(u.Register))
		params := map[string]interface{}{
			"email":         "test[dog]com",
			"password":      "one-pass",
			"favorite_cake": "MyFavCake",
		}

		resp := doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		assertStatus(t, 422, resp)
		assertBody(t, "email is not valid", resp)
		ts.Close()
	})

	t.Run("password checking", func(t *testing.T) {
		u := newTestUserService()

		ts := httptest.NewServer(http.HandlerFunc(u.Register))
		params := map[string]interface{}{
			"email":         "spam@cake.com",
			"password":      "few",
			"favorite_cake": "MyFavCake",
		}

		resp := doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		assertStatus(t, 422, resp)
		assertBody(t, "password length must be at least 8 symbols ", resp)
		ts.Close()
	})

	t.Run("favorite cake checking", func(t *testing.T) {
		u := newTestUserService()

		ts := httptest.NewServer(http.HandlerFunc(u.Register))
		params := map[string]interface{}{
			"email":         "spam@cake.com",
			"password":      "one-pass",
			"favorite_cake": "",
		}

		resp := doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		assertStatus(t, 422, resp)
		assertBody(t, "favorite cake should not be empty", resp)

		params["favorite_cake"] = "!!Cake have $ymbols"
		resp = doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		assertStatus(t, 422, resp)
		assertBody(t, "favorite cake should have only alphabetic characters", resp)

		ts.Close()
	})

	t.Run("succesful registration", func(t *testing.T) {
		u := newTestUserService()

		ts := httptest.NewServer(http.HandlerFunc(u.Register))
		params := map[string]interface{}{
			"email":         "spam@cake.com",
			"password":      "one-pass",
			"favorite_cake": "MyFavCake",
		}

		resp := doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		assertStatus(t, 201, resp)
		assertBody(t, "registered", resp)
		ts.Close()
	})

	t.Run("unsuccesful registration", func(t *testing.T) {
		u := newTestUserService()

		ts := httptest.NewServer(http.HandlerFunc(u.Register))
		params := map[string]interface{}{
			"email":         "spam@cake.com",
			"password":      "one-pass",
			"favorite_cake": "MyFavCake",
		}

		resp := doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		resp = doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		assertStatus(t, 422, resp)
		assertBody(t, "this email address is already exists in the database", resp)
		ts.Close()
	})

}

func TestUsers_JWT(t *testing.T) {
	doRequest := createRequester(t)

	t.Run("user does not exist", func(t *testing.T) {
		u := newTestUserService()
		j, err := NewMyJWTService()
		if err != nil {
			t.FailNow()
		}

		ts := httptest.NewServer(wrapJWT(j, u.JWT))
		defer ts.Close()

		params := map[string]interface{}{
			"email":    "spam@cake.com",
			"password": "one-pass",
		}

		resp := doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		assertStatus(t, 422, resp)
		assertBody(t, "user not found", resp)
	})

	t.Run("wrong password", func(t *testing.T) {
		u := newTestUserService()
		j, err := NewMyJWTService()
		if err != nil {
			t.FailNow()
		}

		ts := httptest.NewServer(http.HandlerFunc(u.Register))
		params := map[string]interface{}{
			"email":         "spam@cake.com",
			"password":      "good_pass",
			"favorite_cake": "MyFavCake",
		}

		resp := doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		ts.Close()

		ts = httptest.NewServer(wrapJWT(j, u.JWT))
		params = map[string]interface{}{
			"email":    "spam@cake.com",
			"password": "bad_pass",
		}

		resp = doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		assertStatus(t, 422, resp)
		assertBody(t, "email or password is incorrect", resp)
		ts.Close()
	})

	t.Run("unauthorized cake", func(t *testing.T) {
		u := newTestUserService()
		j, err := NewMyJWTService()
		if err != nil {
			t.FailNow()
		}

		ts := httptest.NewServer(j.jwtAuth(u.repository, getCakeHandler))
		defer ts.Close()

		resp := doRequest(http.NewRequest(http.MethodGet, ts.URL, nil))
		assertStatus(t, 401, resp)
		assertBody(t, "unauthorized", resp)
	})

	t.Run("wrong credentials", func(t *testing.T) {
		u := newTestUserService()
		j, err := NewMyJWTService()
		if err != nil {
			t.FailNow()
		}

		ts := httptest.NewServer(j.jwtAuth(u.repository, getCakeHandler))
		defer ts.Close()

		req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
		req.Header.Set("Authorization", "Bearer one-thing strange instead of jwt")
		resp := doRequest(req, err)
		assertStatus(t, 401, resp)
		assertBody(t, "unauthorized", resp)
	})

	t.Run("authorized cake", func(t *testing.T) {
		u := newTestUserService()
		j, err := NewMyJWTService()
		if err != nil {
			t.FailNow()
		}

		ts := httptest.NewServer(http.HandlerFunc(u.Register))
		params := map[string]interface{}{
			"email":         "spam@cake.com",
			"password":      "one-pass",
			"favorite_cake": "MyFavCake",
		}

		resp := doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		ts.Close()

		ts = httptest.NewServer(wrapJWT(j, u.JWT))
		params = map[string]interface{}{
			"email":    "spam@cake.com",
			"password": "one-pass",
		}

		resp = doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		ts.Close()

		ts = httptest.NewServer(j.jwtAuth(u.repository, getCakeHandler))
		req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
		req.Header.Set("Authorization", "Bearer "+string(resp.body))
		resp = doRequest(req, err)
		assertStatus(t, 200, resp)
		assertBody(t, "MyFavCake", resp)
		ts.Close()
	})
}

func TestUsers_Update(t *testing.T) {
	doRequest := createRequester(t)

	t.Run("favorite cake updating", func(t *testing.T) {
		us := newTestUserService()
		js, err := NewMyJWTService()
		if err != nil {
			t.FailNow()
		}

		ts := httptest.NewServer(http.HandlerFunc(us.Register))
		params := map[string]interface{}{
			"email":         "spam@cake.com",
			"password":      "one-pass",
			"favorite_cake": "MyFavCake",
		}

		resp := doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		ts.Close()

		ts = httptest.NewServer(wrapJWT(js, us.JWT))
		params = map[string]interface{}{
			"email":    "spam@cake.com",
			"password": "one-pass",
		}

		resp = doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		jwtToken := string(resp.body)
		ts.Close()

		ts = httptest.NewServer(js.jwtAuth(us.repository, us.UpdateCake))

		params = map[string]interface{}{
			"favorite_cake": "",
		}
		req, err := http.NewRequest(http.MethodPut, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 422, resp)
		assertBody(t, "favorite cake should not be empty", resp)

		params["favorite_cake"] = "some cake"
		req, err = http.NewRequest(http.MethodPut, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 422, resp)
		assertBody(t, "favorite cake should have only alphabetic characters", resp)

		params["favorite_cake"] = "MyFavCake"
		req, err = http.NewRequest(http.MethodPut, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 201, resp)
		assertBody(t, "favorite cake updated", resp)
	})

	t.Run("password updating", func(t *testing.T) {
		us := newTestUserService()
		js, err := NewMyJWTService()
		if err != nil {
			t.FailNow()
		}

		ts := httptest.NewServer(http.HandlerFunc(us.Register))
		params := map[string]interface{}{
			"email":         "spam@cake.com",
			"password":      "one-pass",
			"favorite_cake": "MyFavCake",
		}

		resp := doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		ts.Close()

		ts = httptest.NewServer(wrapJWT(js, us.JWT))
		params = map[string]interface{}{
			"email":    "spam@cake.com",
			"password": "one-pass",
		}

		resp = doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		jwtToken := string(resp.body)
		ts.Close()

		ts = httptest.NewServer(
			js.jwtAuth(us.repository, us.UpdatePassword),
		)

		params = map[string]interface{}{
			"password": "pwd",
		}
		req, err := http.NewRequest(http.MethodPut, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 422, resp)
		assertBody(t, "password length must be at least 8 symbols ", resp)

		params["password"] = "one-pass"
		req, err = http.NewRequest(http.MethodPut, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 201, resp)
		assertBody(t, "password updated", resp)

		req, err = http.NewRequest(http.MethodPut, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 401, resp)
		assertBody(t, "token is banned", resp)
	})

	t.Run("email updating", func(t *testing.T) {
		us := newTestUserService()
		js, err := NewMyJWTService()
		if err != nil {
			t.FailNow()
		}

		ts := httptest.NewServer(http.HandlerFunc(us.Register))
		params := map[string]interface{}{
			"email":         "spam@cake.com",
			"password":      "one-pass",
			"favorite_cake": "MyFavCake",
		}

		resp := doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		ts.Close()

		ts = httptest.NewServer(wrapJWT(js, us.JWT))
		params = map[string]interface{}{
			"email":    "spam@cake.com",
			"password": "one-pass",
		}

		resp = doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		jwtToken := string(resp.body)
		ts.Close()

		ts = httptest.NewServer(
			js.jwtAuth(us.repository, us.UpdateEmail),
		)

		params = map[string]interface{}{
			"email": "spam[dog]cake.com",
		}
		req, err := http.NewRequest(http.MethodPut, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 422, resp)
		assertBody(t, "email is not valid", resp)

		params["email"] = "admin@penware.com"
		req, err = http.NewRequest(http.MethodPut, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 201, resp)
		assertBody(t, "email updated", resp)

		req, err = http.NewRequest(http.MethodPut, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 401, resp)
		assertBody(t, "unauthorized", resp)
	})
}

func TestUsers_Admin(t *testing.T) {
	doRequest := createRequester(t)
	suLogin := os.Getenv("CAKE_ADMIN_EMAIL")
	suPassword := os.Getenv("CAKE_ADMIN_PASSWORD")

	t.Run("banning user", func(t *testing.T) {
		us := newTestUserService()
		js, err := NewMyJWTService()
		if err != nil {
			t.FailNow()
		}

		ts := httptest.NewServer(http.HandlerFunc(us.Register))
		params := map[string]interface{}{
			"email":         "spam@cake.com",
			"password":      "one-pass",
			"favorite_cake": "MyFavCake",
		}

		resp := doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		ts.Close()

		ts = httptest.NewServer(wrapJWT(js, us.JWT))
		params = map[string]interface{}{
			"email":    "spam@cake.com",
			"password": "one-pass",
		}

		resp = doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		jwtToken := string(resp.body)

		params = map[string]interface{}{
			"email":    suLogin,
			"password": suPassword,
		}

		resp = doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		suJwtToken := string(resp.body)

		ts = httptest.NewServer(js.jwtAuth(us.repository, us.History))

		url := ts.URL + "?email=spam@cake.com"
		req, err := http.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Authorization", "Bearer "+suJwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 422, resp)
		assertBody(t, "user history is clear", resp)
		ts.Close()

		ts = httptest.NewServer(js.jwtAuth(us.repository, us.BanUser))
		defer ts.Close()

		params = map[string]interface{}{
			"email":  "spam@cake.com",
			"reason": "some reason",
		}
		req, err = http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+suJwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 201, resp)
		assertBody(t, "user \"spam@cake.com\" is banned with reason\"some reason\" by \""+suLogin+"\"", resp)

		req, err = http.NewRequest(http.MethodPut, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 401, resp)
		assertBody(t, "user is banned with reason \"some reason\" by \""+suLogin+"\"", resp)
	})

	t.Run("unbanning user", func(t *testing.T) {
		us := newTestUserService()
		js, err := NewMyJWTService()
		if err != nil {
			t.FailNow()
		}

		ts := httptest.NewServer(http.HandlerFunc(us.Register))
		params := map[string]interface{}{
			"email":         "spam@cake.com",
			"password":      "one-pass",
			"favorite_cake": "MyFavCake",
		}

		resp := doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		ts.Close()

		ts = httptest.NewServer(wrapJWT(js, us.JWT))
		params = map[string]interface{}{
			"email":    "spam@cake.com",
			"password": "one-pass",
		}

		resp = doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		jwtToken := string(resp.body)

		params = map[string]interface{}{
			"email":    suLogin,
			"password": suPassword,
		}

		resp = doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		suJwtToken := string(resp.body)

		ts.Close()

		ts = httptest.NewServer(js.jwtAuth(us.repository, us.BanUser))

		params = map[string]interface{}{
			"email":  "spam@cake.com",
			"reason": "some reason",
		}
		req, err := http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+suJwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 201, resp)
		assertBody(t, "user \"spam@cake.com\" is banned with reason\"some reason\" by \""+suLogin+"\"", resp)
		ts.Close()

		ts = httptest.NewServer(js.jwtAuth(us.repository, us.UnBanUser))
		defer ts.Close()

		params = map[string]interface{}{
			"email": "spam@cake.com",
		}
		req, err = http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+suJwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 201, resp)
		assertBody(t, "user \"spam@cake.com\" is UnBanned by \""+suLogin+"\"", resp)

		req, err = http.NewRequest(http.MethodPut, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 422, resp)
		assertBody(t, "no privileges, access denied", resp)
	})

	t.Run("view history", func(t *testing.T) {
		us := newTestUserService()
		js, err := NewMyJWTService()
		if err != nil {
			t.FailNow()
		}

		ts := httptest.NewServer(http.HandlerFunc(us.Register))
		params := map[string]interface{}{
			"email":         "spam@cake.com",
			"password":      "one-pass",
			"favorite_cake": "MyFavCake",
		}

		resp := doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		ts.Close()

		ts = httptest.NewServer(wrapJWT(js, us.JWT))
		params = map[string]interface{}{
			"email":    "spam@cake.com",
			"password": "one-pass",
		}

		resp = doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		jwtToken := string(resp.body)

		ts = httptest.NewServer(js.jwtAuth(us.repository, us.History))

		url := ts.URL + "?email="
		req, err := http.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 422, resp)
		assertBody(t, "email is empty", resp)

		url = ts.URL + "?email=spam@cake.com"
		req, err = http.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 422, resp)
		assertBody(t, "no privileges, access denied", resp)

		ts.Close()
	})

	t.Run("admin promote", func(t *testing.T) {

		us := newTestUserService()
		js, err := NewMyJWTService()
		if err != nil {
			t.FailNow()
		}

		ts := httptest.NewServer(http.HandlerFunc(us.Register))
		params := map[string]interface{}{
			"email":         "spam@cake.com",
			"password":      "one-pass",
			"favorite_cake": "MyFavCake",
		}

		resp := doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		ts.Close()

		ts = httptest.NewServer(wrapJWT(js, us.JWT))
		params = map[string]interface{}{
			"email":    "spam@cake.com",
			"password": "one-pass",
		}

		resp = doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		jwtToken := string(resp.body)

		params = map[string]interface{}{
			"email":    suLogin,
			"password": suPassword,
		}

		resp = doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		suJwtToken := string(resp.body)

		ts.Close()

		ts = httptest.NewServer(js.jwtAuth(us.repository, us.Promote))

		params = map[string]interface{}{
			"email": "spam[dog]cake.com",
		}
		req, err := http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+suJwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 422, resp)
		assertBody(t, "user not found", resp)

		params = map[string]interface{}{
			"email": suLogin,
		}
		req, err = http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+suJwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 422, resp)
		assertBody(t, "you cannot change the role to yourself", resp)

		req, err = http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 422, resp)
		assertBody(t, "no privileges, access denied", resp)

		params = map[string]interface{}{
			"email": "spam@cake.com",
		}
		req, err = http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+suJwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 201, resp)
		assertBody(t, "user \""+suLogin+"\" changed the privileges of user \"spam@cake.com\" to \"admin\"", resp)

		ts.Close()
	})

	t.Run("admin fire", func(t *testing.T) {

		us := newTestUserService()
		js, err := NewMyJWTService()
		if err != nil {
			t.FailNow()
		}

		ts := httptest.NewServer(http.HandlerFunc(us.Register))
		params := map[string]interface{}{
			"email":         "spam@cake.com",
			"password":      "one-pass",
			"favorite_cake": "MyFavCake",
		}

		resp := doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		ts.Close()

		ts = httptest.NewServer(wrapJWT(js, us.JWT))
		params = map[string]interface{}{
			"email":    "spam@cake.com",
			"password": "one-pass",
		}

		resp = doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		jwtToken := string(resp.body)

		params = map[string]interface{}{
			"email":    suLogin,
			"password": suPassword,
		}

		resp = doRequest(http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params)))
		suJwtToken := string(resp.body)

		ts.Close()

		ts = httptest.NewServer(js.jwtAuth(us.repository, us.Promote))
		params = map[string]interface{}{
			"email": "spam@cake.com",
		}
		req, err := http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+suJwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 201, resp)
		assertBody(t, "user \""+suLogin+"\" changed the privileges of user \"spam@cake.com\" to \"admin\"", resp)
		ts.Close()

		ts = httptest.NewServer(js.jwtAuth(us.repository, us.Fire))

		params = map[string]interface{}{
			"email": "spam[dog]cake.com",
		}
		req, err = http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+suJwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 422, resp)
		assertBody(t, "user not found", resp)

		params = map[string]interface{}{
			"email": suLogin,
		}
		req, err = http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+suJwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 422, resp)
		assertBody(t, "you cannot change the role to yourself", resp)

		req, err = http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 422, resp)
		assertBody(t, "no privileges, access denied", resp)

		params = map[string]interface{}{
			"email": "spam@cake.com",
		}
		req, err = http.NewRequest(http.MethodPost, ts.URL, prepareParams(t, params))
		req.Header.Set("Authorization", "Bearer "+suJwtToken)
		resp = doRequest(req, err)
		assertStatus(t, 201, resp)
		assertBody(t, "user \""+suLogin+"\" has revoked the privileges of user \"spam@cake.com\"", resp)
		ts.Close()
	})

}
